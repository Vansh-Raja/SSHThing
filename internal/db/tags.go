package db

import (
	"encoding/json"
	"strings"
	"unicode"
)

// NormalizeTagToken converts a raw tag token to canonical lowercase alphanumeric form.
func NormalizeTagToken(token string) string {
	token = strings.TrimSpace(token)
	token = strings.TrimPrefix(token, "#")
	if token == "" {
		return ""
	}

	var b strings.Builder
	for _, r := range token {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

// NormalizeTags normalizes and de-duplicates tag tokens.
func NormalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}

	seen := make(map[string]bool, len(tags))
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		n := NormalizeTagToken(t)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ParseTagInput parses comma-separated user input into canonical tags.
func ParseTagInput(input string) []string {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	raw := make([]string, 0, 8)
	parts := strings.Split(input, ",")
	for _, part := range parts {
		for _, field := range strings.Fields(part) {
			raw = append(raw, field)
		}
	}
	return NormalizeTags(raw)
}

// EncodeTags encodes canonical tags for DB storage.
func EncodeTags(tags []string) (string, error) {
	norm := NormalizeTags(tags)
	if len(norm) == 0 {
		return "", nil
	}
	b, err := json.Marshal(norm)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// DecodeTags decodes tags from DB storage (JSON preferred, comma fallback).
func DecodeTags(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	var tags []string
	if strings.HasPrefix(raw, "[") {
		if err := json.Unmarshal([]byte(raw), &tags); err == nil {
			return NormalizeTags(tags)
		}
	}
	return ParseTagInput(raw)
}
