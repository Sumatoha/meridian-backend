package ai

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// ParseJSON extracts and parses JSON from an AI response.
// Handles cases where the AI wraps JSON in markdown code blocks or
// includes extra text before/after the JSON array.
func ParseJSON[T any](raw string) (T, error) {
	var result T

	cleaned := cleanJSONResponse(raw)

	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		// Try to extract the JSON array/object from surrounding text
		extracted := extractJSON(cleaned)
		if extracted != "" && extracted != cleaned {
			if err2 := json.Unmarshal([]byte(extracted), &result); err2 == nil {
				return result, nil
			}
		}

		// Try sanitizing invalid UTF-8 sequences
		sanitized := sanitizeUTF8(cleaned)
		if sanitized != cleaned {
			if err2 := json.Unmarshal([]byte(sanitized), &result); err2 == nil {
				return result, nil
			}
		}

		return result, fmt.Errorf("ai: parse JSON response: %w\nraw_tail: %.500s", err, tail(raw, 500))
	}

	return result, nil
}

func cleanJSONResponse(raw string) string {
	s := strings.TrimSpace(raw)

	// Strip markdown code fences (```json ... ``` or ``` ... ```)
	if strings.HasPrefix(s, "```") {
		// Remove the opening fence line
		if idx := strings.Index(s, "\n"); idx >= 0 {
			s = s[idx+1:]
		} else {
			s = strings.TrimPrefix(s, "```json")
			s = strings.TrimPrefix(s, "```")
		}
		// Remove the closing fence
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
	}

	return strings.TrimSpace(s)
}

// extractJSON finds the outermost JSON array or object in a string.
// Handles cases where the AI adds commentary before/after the JSON.
func extractJSON(s string) string {
	// Find the first [ or { that starts a JSON structure
	arrayStart := strings.Index(s, "[")
	objectStart := strings.Index(s, "{")

	start := -1
	var openChar, closeChar byte

	switch {
	case arrayStart >= 0 && (objectStart < 0 || arrayStart <= objectStart):
		start = arrayStart
		openChar = '['
		closeChar = ']'
	case objectStart >= 0:
		start = objectStart
		openChar = '{'
		closeChar = '}'
	default:
		return ""
	}

	// Find the matching closing bracket by counting depth
	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(s); i++ {
		c := s[i]

		if escaped {
			escaped = false
			continue
		}

		if c == '\\' && inString {
			escaped = true
			continue
		}

		if c == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		if c == openChar {
			depth++
		} else if c == closeChar {
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}

	return ""
}

// sanitizeUTF8 replaces invalid UTF-8 byte sequences that can break JSON parsing.
func sanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}

	var b strings.Builder
	b.Grow(len(s))

	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			// Skip invalid byte
			i++
			continue
		}
		b.WriteRune(r)
		i += size
	}

	return b.String()
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "..." + s[len(s)-n:]
}
