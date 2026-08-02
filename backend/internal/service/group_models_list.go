package service

import "strings"

func normalizeGroupModelsListConfig(cfg GroupModelsListConfig) GroupModelsListConfig {
	out := GroupModelsListConfig{
		Enabled: cfg.Enabled,
		CCSwitchDefaults: GroupCCSwitchDefaults{
			Claude: GroupCCSwitchClaudeDefaults{
				Model:  strings.TrimSpace(cfg.CCSwitchDefaults.Claude.Model),
				Haiku:  strings.TrimSpace(cfg.CCSwitchDefaults.Claude.Haiku),
				Sonnet: strings.TrimSpace(cfg.CCSwitchDefaults.Claude.Sonnet),
				Opus:   strings.TrimSpace(cfg.CCSwitchDefaults.Claude.Opus),
			},
			Codex:    strings.TrimSpace(cfg.CCSwitchDefaults.Codex),
			Gemini:   strings.TrimSpace(cfg.CCSwitchDefaults.Gemini),
			OpenCode: strings.TrimSpace(cfg.CCSwitchDefaults.OpenCode),
		},
	}
	if len(cfg.Models) == 0 {
		return out
	}

	seen := make(map[string]struct{}, len(cfg.Models))
	out.Models = make([]string, 0, len(cfg.Models))
	for _, model := range cfg.Models {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		if _, ok := seen[model]; ok {
			continue
		}
		seen[model] = struct{}{}
		out.Models = append(out.Models, model)
	}
	if len(out.Models) == 0 {
		out.Models = nil
	}
	return out
}

func (g *Group) CustomModelsListEnabled() bool {
	return g != nil && g.ModelsListConfig.Enabled && len(g.ModelsListConfig.Models) > 0
}
