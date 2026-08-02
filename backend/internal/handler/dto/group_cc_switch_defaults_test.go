package dto

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestGroupFromServiceShallowExposesCCSwitchDefaults(t *testing.T) {
	group := &service.Group{
		ModelsListConfig: service.GroupModelsListConfig{
			CCSwitchDefaults: service.GroupCCSwitchDefaults{
				Codex:    "gpt-5.5",
				OpenCode: "gpt-5.5",
			},
		},
	}

	out := GroupFromServiceShallow(group)
	require.NotNil(t, out)
	require.Equal(t, "gpt-5.5", out.CCSwitchDefaults.Codex)
	require.Equal(t, "gpt-5.5", out.CCSwitchDefaults.OpenCode)
}
