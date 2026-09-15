//go:build unit

package repository

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestImageStorageS3DeletesUnjournaledAndRetriedVersions(t *testing.T) {
	for _, knownVersion := range []string{"", "version-2"} {
		t.Run("known="+knownVersion, func(t *testing.T) {
			versions := []string{"version-1", "version-2"}
			deleted := []string{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/images/studio/asset.png", r.URL.Path)
				switch r.Method {
				case http.MethodHead:
					if len(versions) == 0 {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					w.Header().Set("x-amz-version-id", versions[len(versions)-1])
				case http.MethodDelete:
					version := r.URL.Query().Get("versionId")
					require.NotEmpty(t, version, "a delete marker would not release stored bytes")
					deleted = append(deleted, version)
					for index, item := range versions {
						if item == version {
							versions = append(versions[:index], versions[index+1:]...)
							break
						}
					}
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Errorf("unexpected request: %s", r.Method)
					w.WriteHeader(http.StatusBadRequest)
				}
			}))
			defer server.Close()
			storage, err := NewS3ImageStorage(context.Background(), &config.ImageStorageConfig{
				Endpoint: server.URL, Bucket: "images", Region: "test", AccessKeyID: "test", SecretAccessKey: "test", ForcePathStyle: true,
			})
			require.NoError(t, err)
			require.NoError(t, storage.Delete(context.Background(), "studio/asset.png", knownVersion))
			require.Empty(t, versions)
			require.Equal(t, []string{"version-2", "version-1"}, deleted)
		})
	}
}
