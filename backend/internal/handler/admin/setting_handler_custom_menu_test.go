package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSettingHandler_CustomMenuOpenMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, mode := range []string{"", "iframe", "new_tab", "invalid"} {
		t.Run("mode="+mode, func(t *testing.T) {
			repo := &settingHandlerRepoStub{values: map[string]string{service.SettingKeyPromoCodeEnabled: "true"}}
			svc := service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}})
			handler := NewSettingHandler(svc, nil, nil, nil, nil, nil, nil)
			items := []dto.CustomMenuItem{
				{ID: "cards", Label: "Card recharge", URL: "https://cards.example.com/", Visibility: "user", OpenMode: mode},
				{ID: "admin", Label: "Admin link", URL: "https://admin.example.com/", Visibility: "admin", OpenMode: mode},
			}
			body, err := json.Marshal(map[string]any{"custom_menu_items": items})
			require.NoError(t, err)
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")
			handler.UpdateSettings(c)

			if mode == "invalid" {
				require.Equal(t, http.StatusBadRequest, rec.Code)
				require.Contains(t, rec.Body.String(), "open_mode")
				require.Nil(t, repo.lastUpdates)
				return
			}
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			stored := repo.values[service.SettingKeyCustomMenuItems]
			require.Equal(t, items, dto.ParseCustomMenuItems(stored))
			require.Equal(t, items[:1], dto.ParseUserVisibleMenuItems(stored))
			var result struct {
				Data struct {
					CustomMenuItems []dto.CustomMenuItem `json:"custom_menu_items"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
			require.Equal(t, items, result.Data.CustomMenuItems)
		})
	}
}
