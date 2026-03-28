package dotenv

import (
	"fmt"
	"strings"
)

// Entry represents a single key-value pair from a .env file.
type Entry struct {
	Key   string
	Value string
}

// Parse parses .env file content into key-value entries.
// Returns parsed entries and any warnings for malformed lines.
// Handles double-quoted (with escape sequences), single-quoted (literal), and unquoted values.
func Parse(content string) ([]Entry, []string) {
	var entries []Entry
	var warnings []string

	lines := strings.Split(content, "\n")
	for lineNum, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Strip optional "export " prefix
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimPrefix(line, "export ")
			line = strings.TrimSpace(line)
		}

		// Find the first = sign
		eqIdx := strings.Index(line, "=")
		if eqIdx < 0 {
			warnings = append(warnings, fmt.Sprintf("line %d: skipping malformed line (no '='): %s", lineNum+1, line))
			continue
		}

		key := strings.TrimSpace(line[:eqIdx])
		rawValue := line[eqIdx+1:]

		value := parseValue(rawValue)
		entries = append(entries, Entry{Key: key, Value: value})
	}

	return entries, warnings
}

// parseValue handles the three .env value formats.
func parseValue(raw string) string {
	raw = strings.TrimSpace(raw)

	if len(raw) == 0 {
		return ""
	}

	// Double-quoted: unescape \n, \\, \"
	if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		inner := raw[1 : len(raw)-1]
		inner = strings.ReplaceAll(inner, `\n`, "\n")
		inner = strings.ReplaceAll(inner, `\\`, "\\")
		inner = strings.ReplaceAll(inner, `\"`, `"`)
		return inner
	}

	// Single-quoted: literal, no escaping
	if len(raw) >= 2 && raw[0] == '\'' && raw[len(raw)-1] == '\'' {
		return raw[1 : len(raw)-1]
	}

	// Unquoted: strip inline comments and trailing whitespace
	if idx := strings.Index(raw, " #"); idx >= 0 {
		raw = raw[:idx]
	}
	return strings.TrimRight(raw, " \t")
}
