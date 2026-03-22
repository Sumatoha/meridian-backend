package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseJSON extracts and parses JSON from an AI response.
// Handles cases where the AI wraps JSON in markdown code blocks.
// Also attempts to repair truncated JSON arrays.
func ParseJSON[T any](raw string) (T, error) {
	var result T

	cleaned := cleanJSONResponse(raw)

	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		// Try to repair truncated JSON array
		repaired := repairTruncatedArray(cleaned)
		if repaired != cleaned {
			if err2 := json.Unmarshal([]byte(repaired), &result); err2 == nil {
				return result, nil
			}
		}
		return result, fmt.Errorf("ai: parse JSON response: %w\nraw_tail: %.500s", err, tail(raw, 500))
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

// repairTruncatedArray attempts to fix a JSON array that was cut off mid-element.
// It finds the last complete object and closes the array.
func repairTruncatedArray(s string) string {
	if !strings.HasPrefix(s, "[") {
		return s
	}
	// Already valid
	if strings.HasSuffix(s, "]") {
		return s
	}

	// Find the last complete object by looking for the last "},"
	// or "}" that completes an array element
	lastComplete := strings.LastIndex(s, "},")
	if lastComplete > 0 {
		return s[:lastComplete+1] + "]"
	}

	// Try just closing after last "}"
	lastBrace := strings.LastIndex(s, "}")
	if lastBrace > 0 {
		return s[:lastBrace+1] + "]"
	}

	return s
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "..." + s[len(s)-n:]
}
