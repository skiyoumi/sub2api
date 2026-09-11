//go:build unit

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestSidebarMenuOrder_PublicInjection(t *testing.T) {
	raw := `{"user":["/purchase","/custom/cards","/orders"],"admin":["/custom/private","/admin/settings"]}`
	svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{SettingKeySidebarMenuOrder: raw}}, &config.Config{})
	public, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, raw, public.SidebarMenuOrder)
	injected, err := svc.GetPublicSettingsForInjection(context.Background())
	require.NoError(t, err)
	encoded, err := json.Marshal(injected)
	require.NoError(t, err)
	var payload struct {
		Order SidebarMenuOrder `json:"sidebar_menu_order"`
	}
	require.NoError(t, json.Unmarshal(encoded, &payload))
	require.Equal(t, []string{"/purchase", "/custom/cards", "/orders"}, payload.Order.User)
	require.Empty(t, payload.Order.Admin)
	require.NotContains(t, string(encoded), "/custom/private")
}

func TestSidebarMenuOrder_DefaultAndValidation(t *testing.T) {
	for _, raw := range []string{"", "null", "{}", "[]", "not-json", `{"user":null}`, `{"user":123}`} {
		require.Equal(t, SidebarMenuOrder{User: []string{}}, ParseSidebarMenuOrder(raw), raw)
	}
	require.NoError(t, (SidebarMenuOrder{User: []string{"/keys", "/custom/a-b_C9"}, Admin: []string{"/keys"}}).Validate())
	for _, path := range []string{"", "//host/path", "/keys?x=1", "/keys#hash", "/../admin", "/" + strings.Repeat("x", 128)} {
		require.Error(t, (SidebarMenuOrder{User: []string{path}}).Validate(), path)
	}
	paths := make([]string, 101)
	for i := range paths {
		paths[i] = fmt.Sprintf("/custom/item-%d", i)
	}
	require.Error(t, (SidebarMenuOrder{Admin: paths}).Validate())
	require.NoError(t, (SidebarMenuOrder{Admin: paths[:100]}).Validate())
}
