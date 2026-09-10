package service

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestNormalizeGroupModelAllowlistPreservesCCSwitchDefaults(t *testing.T) {
	cfg, err := normalizeGroupModelAllowlist(GroupModelAllowlist{
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

	require.NoError(t, err)
	require.Equal(t, []string{"model-a", "model-b"}, cfg.Models)
	require.Equal(t, "claude-main", cfg.CCSwitchDefaults.Claude.Model)
	require.Equal(t, "haiku", cfg.CCSwitchDefaults.Claude.Haiku)
	require.Equal(t, "sonnet", cfg.CCSwitchDefaults.Claude.Sonnet)
	require.Equal(t, "opus", cfg.CCSwitchDefaults.Claude.Opus)
	require.Equal(t, "codex", cfg.CCSwitchDefaults.Codex)
	require.Equal(t, "gemini", cfg.CCSwitchDefaults.Gemini)
	require.Equal(t, "opencode", cfg.CCSwitchDefaults.OpenCode)
}

func TestGroupModelAllowlistPreservesMigratedCCSwitchDefaults(t *testing.T) {
	// The column rename keeps the legacy JSON unchanged, including CCS defaults.
	legacyJSON := `{"enabled":false,"cc_switch_defaults":{"claude":{"model":" claude-main "},"codex":" gpt-5.5 ","gemini":" gemini-2.5-pro ","opencode":" gpt-5.5 "}}`
	var stored domain.GroupModelAllowlist
	require.NoError(t, json.Unmarshal([]byte(legacyJSON), &stored))

	cfg, err := normalizeGroupModelAllowlist(GroupModelAllowlistFromDomain(stored))
	require.NoError(t, err)
	require.False(t, cfg.Enabled)
	require.Empty(t, cfg.Models)
	require.Equal(t, "claude-main", cfg.CCSwitchDefaults.Claude.Model)
	require.Equal(t, "gpt-5.5", cfg.CCSwitchDefaults.Codex)
	require.Equal(t, "gemini-2.5-pro", cfg.CCSwitchDefaults.Gemini)
	require.Equal(t, "gpt-5.5", cfg.CCSwitchDefaults.OpenCode)

	copy := cloneGroupForDuplicate(&Group{ModelAllowlist: cfg}, "copy")
	roundTrip, err := json.Marshal(DomainGroupModelAllowlist(copy.ModelAllowlist))
	require.NoError(t, err)
	var restored domain.GroupModelAllowlist
	require.NoError(t, json.Unmarshal(roundTrip, &restored))
	require.Equal(t, cfg, GroupModelAllowlistFromDomain(restored))
}
