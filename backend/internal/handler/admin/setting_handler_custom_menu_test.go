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

func TestSettingHandler_SidebarMenuOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	old := `{"user":["/profile","/custom/cards"],"admin":["/admin/settings"]}`
	for _, tc := range []struct {
		name, body, want string
		invalid          bool
	}{
		{name: "save mixed order", body: `{"sidebar_menu_order":{"user":["/purchase","/custom/cards","/orders"],"admin":["/admin/settings","/admin/dashboard"]}}`, want: `{"user":["/purchase","/custom/cards","/orders"],"admin":["/admin/settings","/admin/dashboard"]}`},
		{name: "omitted field preserves order", body: `{}`, want: old},
		{name: "null field preserves order", body: `{"sidebar_menu_order":null}`, want: old},
		{name: "reset order", body: `{"sidebar_menu_order":{}}`, want: `{"user":[]}`},
		{name: "duplicate user path", body: `{"sidebar_menu_order":{"user":["/keys","/keys"]}}`, invalid: true},
		{name: "duplicate admin path", body: `{"sidebar_menu_order":{"admin":["/admin/settings","/admin/settings"]}}`, invalid: true},
		{name: "external URL", body: `{"sidebar_menu_order":{"user":["https://example.com"]}}`, invalid: true},
		{name: "invalid shape", body: `{"sidebar_menu_order":{"user":"/keys"}}`, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &settingHandlerRepoStub{values: map[string]string{service.SettingKeyPromoCodeEnabled: "true", service.SettingKeySidebarMenuOrder: old}}
			svc := service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}})
			handler := NewSettingHandler(svc, nil, nil, nil, nil, nil, nil)
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewBufferString(tc.body))
			c.Request.Header.Set("Content-Type", "application/json")
			handler.UpdateSettings(c)
			if tc.invalid {
				require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
				require.Nil(t, repo.lastUpdates)
				require.Equal(t, old, repo.values[service.SettingKeySidebarMenuOrder])
				return
			}
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			require.JSONEq(t, tc.want, repo.values[service.SettingKeySidebarMenuOrder])
			var result struct {
				Data struct {
					Order service.SidebarMenuOrder `json:"sidebar_menu_order"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
			require.Equal(t, service.ParseSidebarMenuOrder(tc.want), result.Data.Order)

			getRec := httptest.NewRecorder()
			getCtx, _ := gin.CreateTestContext(getRec)
			getCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)
			handler.GetSettings(getCtx)
			require.Equal(t, http.StatusOK, getRec.Code, getRec.Body.String())
			require.NoError(t, json.Unmarshal(getRec.Body.Bytes(), &result))
			require.Equal(t, service.ParseSidebarMenuOrder(tc.want), result.Data.Order)
		})
	}
}
