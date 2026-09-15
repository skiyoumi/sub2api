package service

import (
	"encoding/json"
	"fmt"
	"strings"
)

// OAuth uses a Responses model to invoke image_generation. Tool configuration
// alone can be normalized to auto by that endpoint, so also tell the driver
// which explicit controls to use. This is guidance, not proof of output size;
// the returned image bytes remain authoritative.
func openAIImagesExecutionInstructions(parsed *OpenAIImagesRequest) string {
	instructions := openAIImagesVerbatimPromptInstructions
	controls := map[string]any{}
	canvas := ""
	if width, height, ok := parseImageBillingDimensions(parsed.Size); ok {
		controls["size"] = fmt.Sprintf("%dx%d", width, height)
		orientation := "square"
		if width > height {
			orientation = "landscape"
		} else if height > width {
			orientation = "portrait"
		}
		canvas = fmt.Sprintf(" The required canvas is %d pixels wide and %d pixels high (%s). Preserve this orientation regardless of the subject in the prompt.", width, height, orientation)
	}
	switch quality := strings.TrimSpace(parsed.Quality); quality {
	case "low", "medium", "high", "xhigh", "max", "standard", "hd":
		controls["quality"] = quality
	}
	if len(controls) == 0 {
		return instructions
	}
	encoded, _ := json.Marshal(controls)
	return instructions + "\n\nExplicit generation controls, separate from the image prompt: " + string(encoded) +
		". Use these values in the actual image_generation tool invocation, even if the tool's defaults are auto. Do not substitute auto or infer different dimensions or quality from the image subject." + canvas +
		" Keep these controls outside the image prompt; the original prompt must remain verbatim. If the tool cannot honor these controls, report the unsupported setting instead of silently substituting another value."
}
