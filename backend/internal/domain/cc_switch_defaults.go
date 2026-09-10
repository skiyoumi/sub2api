package domain

// GroupCCSwitchDefaults controls the initial selections shown by the CCS import dialog.
type GroupCCSwitchDefaults struct {
	Claude   GroupCCSwitchClaudeDefaults `json:"claude,omitempty"`
	Codex    string                      `json:"codex,omitempty"`
	Gemini   string                      `json:"gemini,omitempty"`
	OpenCode string                      `json:"opencode,omitempty"`
}

type GroupCCSwitchClaudeDefaults struct {
	Model  string `json:"model,omitempty"`
	Haiku  string `json:"haiku,omitempty"`
	Sonnet string `json:"sonnet,omitempty"`
	Opus   string `json:"opus,omitempty"`
}
