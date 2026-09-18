package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func noisyImagePNG(t *testing.T, width, height int) []byte {
	t.Helper()
	source := image.NewNRGBA(image.Rect(0, 0, width, height))
	state := uint32(12345)
	for i := 0; i < len(source.Pix); i += 4 {
		state ^= state << 13
		state ^= state >> 17
		state ^= state << 5
		source.Pix[i], source.Pix[i+1], source.Pix[i+2], source.Pix[i+3] = byte(state), byte(state>>8), byte(state>>16), 255
	}
	var encoded bytes.Buffer
	require.NoError(t, png.Encode(&encoded, source))
	return encoded.Bytes()
}

func TestImagePreview4KPreservesOriginalAndReducesTransfer(t *testing.T) {
	original := noisyImagePNG(t, 3840, 2160)
	storage := &fakeImageStorage{}
	uploader := NewImageResultUploader(storage, "images/", 0, nil)
	input, err := json.Marshal(map[string]any{"data": []map[string]string{{
		"b64_json": base64.StdEncoding.EncodeToString(original), "preview_url": "https://untrusted.test/preview.jpg",
	}}})
	require.NoError(t, err)
	result, err := uploader.Rewrite(context.Background(), "imgtask_preview", input)
	require.NoError(t, err)
	require.Len(t, storage.saved, 2)
	require.Equal(t, original, storage.saved[0].data)
	require.Equal(t, "images/imgtask_preview-preview0.jpg", storage.saved[1].key)
	require.Equal(t, "image/jpeg", storage.saved[1].contentType)
	preview, err := jpeg.DecodeConfig(bytes.NewReader(storage.saved[1].data))
	require.NoError(t, err)
	require.Equal(t, 1024, preview.Width)
	require.Equal(t, 576, preview.Height)
	require.Less(t, len(storage.saved[1].data), len(original)/10)
	require.Equal(t, "3840x2160", gjson.GetBytes(result, "data.0.size").String())
	require.Equal(t, "https://cdn.test/images/imgtask_preview-0.png", gjson.GetBytes(result, "data.0.url").String())
	require.Equal(t, "https://cdn.test/images/imgtask_preview-preview0.jpg", gjson.GetBytes(result, "data.0.preview_url").String())
	t.Logf("4K fixture: original %d bytes, preview %d bytes", len(original), len(storage.saved[1].data))
}

func TestImagePreviewTransparencyAndNoUpscaling(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 32, 16))
	var encoded bytes.Buffer
	require.NoError(t, png.Encode(&encoded, source))
	preview, err := buildImagePreview(context.Background(), encoded.Bytes())
	require.NoError(t, err)
	decoded, err := jpeg.Decode(bytes.NewReader(preview))
	require.NoError(t, err)
	require.Equal(t, source.Bounds(), decoded.Bounds())
	pixel := color.RGBAModel.Convert(decoded.At(0, 0)).(color.RGBA)
	require.Greater(t, pixel.R, uint8(250))
	require.Greater(t, pixel.G, uint8(250))
	require.Greater(t, pixel.B, uint8(250))
}

func TestImagePreviewRejectsOversizedPixelsAndCancellation(t *testing.T) {
	// A valid IHDR is enough to enforce the cap before decoding pixel data.
	oversized := noisyImagePNG(t, 16, 16)
	binary.BigEndian.PutUint32(oversized[16:20], 65536)
	binary.BigEndian.PutUint32(oversized[20:24], 65536)
	binary.BigEndian.PutUint32(oversized[29:33], crc32.ChecksumIEEE(oversized[12:29]))
	_, err := buildImagePreview(context.Background(), oversized)
	require.ErrorContains(t, err, "preview pixel limit")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = buildImagePreview(ctx, noisyImagePNG(t, 16, 16))
	require.ErrorIs(t, err, context.Canceled)
}

type previewFailureStorage struct{ fakeImageStorage }

func (s *previewFailureStorage) Save(ctx context.Context, key, ct string, data []byte) (string, error) {
	if strings.Contains(key, "-preview") {
		return "", errors.New("preview storage unavailable")
	}
	return s.fakeImageStorage.Save(ctx, key, ct, data)
}

func TestImagePreviewFailureKeepsOriginal(t *testing.T) {
	storage := &previewFailureStorage{}
	uploader := NewImageResultUploader(storage, "images/", 0, nil)
	input, err := json.Marshal(map[string]any{"data": []map[string]string{{
		"b64_json":    base64.StdEncoding.EncodeToString(noisyImagePNG(t, 256, 256)),
		"preview_url": "https://untrusted.test/preview.jpg",
	}}})
	require.NoError(t, err)
	result, err := uploader.Rewrite(context.Background(), "imgtask_fallback", input)
	require.NoError(t, err)
	require.Len(t, storage.saved, 1)
	require.NotEmpty(t, gjson.GetBytes(result, "data.0.url").String())
	require.False(t, gjson.GetBytes(result, "data.0.preview_url").Exists())
}
