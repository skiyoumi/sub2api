package service

// CNProviderDefaultModelIDs returns static public model candidates for the
// Chinese OpenAI-compatible platforms. These are used when an account has not
// provided a live/custom model list yet.
func CNProviderDefaultModelIDs(platform string) []string {
	switch platform {
	case PlatformKimi:
		return []string{
			"kimi-k3",
			"k3",
			"k3-256k",
			"kimi-k2.6",
			"kimi-k2.5",
			"kimi-k2-thinking",
			"kimi-k2",
			"moonshot-v1-8k",
			"moonshot-v1-32k",
			"moonshot-v1-128k",
		}
	case PlatformZhipu:
		return []string{
			"glm-5.2",
			"glm-5.1",
			"glm-5",
			"glm-5-turbo",
			"glm-4.7",
			"glm-4.7-flash",
			"glm-4.7-flashx",
			"glm-4.6",
			"glm-4.5",
			"glm-4.5-air",
			"glm-4.5-airx",
			"glm-4.5-flash",
			"glm-4.5-x",
			"glm-4-32b-0414-128k",
		}
	case PlatformDeepseek:
		return []string{
			"deepseek-v4-pro",
			"deepseek-v4-flash",
			"deepseek-chat",
			"deepseek-reasoner",
		}
	default:
		return nil
	}
}
