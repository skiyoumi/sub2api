package service

import "strings"

func normalizeGroupCCSwitchDefaults(cfg GroupCCSwitchDefaults) GroupCCSwitchDefaults {
	return GroupCCSwitchDefaults{
		Claude: GroupCCSwitchClaudeDefaults{
			Model:  strings.TrimSpace(cfg.Claude.Model),
			Haiku:  strings.TrimSpace(cfg.Claude.Haiku),
			Sonnet: strings.TrimSpace(cfg.Claude.Sonnet),
			Opus:   strings.TrimSpace(cfg.Claude.Opus),
		},
		Codex:    strings.TrimSpace(cfg.Codex),
		Gemini:   strings.TrimSpace(cfg.Gemini),
		OpenCode: strings.TrimSpace(cfg.OpenCode),
	}
}
