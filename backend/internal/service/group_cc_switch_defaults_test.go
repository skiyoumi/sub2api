package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeGroupModelsListConfigPreservesCCSwitchDefaults(t *testing.T) {
	cfg := normalizeGroupModelsListConfig(GroupModelsListConfig{
		Enabled: true,
		Models:  []string{" model-a ", "model-a", "model-b"},
		CCSwitchDefaults: GroupCCSwitchDefaults{
			Claude: GroupCCSwitchClaudeDefaults{
				Model:  " claude-main ",
				Haiku:  " haiku ",
				Sonnet: " sonnet ",
				Opus:   " opus ",
			},
			Codex:    " codex ",
			Gemini:   " gemini ",
			OpenCode: " opencode ",
		},
	})

	require.Equal(t, []string{"model-a", "model-b"}, cfg.Models)
	require.Equal(t, "claude-main", cfg.CCSwitchDefaults.Claude.Model)
	require.Equal(t, "haiku", cfg.CCSwitchDefaults.Claude.Haiku)
	require.Equal(t, "sonnet", cfg.CCSwitchDefaults.Claude.Sonnet)
	require.Equal(t, "opus", cfg.CCSwitchDefaults.Claude.Opus)
	require.Equal(t, "codex", cfg.CCSwitchDefaults.Codex)
	require.Equal(t, "gemini", cfg.CCSwitchDefaults.Gemini)
	require.Equal(t, "opencode", cfg.CCSwitchDefaults.OpenCode)
}
