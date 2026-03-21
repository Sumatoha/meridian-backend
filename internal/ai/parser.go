package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseJSON extracts and parses JSON from an AI response.
// Handles cases where the AI wraps JSON in markdown code blocks.
func ParseJSON[T any](raw string) (T, error) {
	var result T

	cleaned := cleanJSONResponse(raw)

	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return result, fmt.Errorf("ai: parse JSON response: %w\nraw: %.500s", err, raw)
	}

	return result, nil
}

func cleanJSONResponse(raw string) string {
	s := strings.TrimSpace(raw)

	// Strip markdown code fences
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
	}

	return strings.TrimSpace(s)
}
