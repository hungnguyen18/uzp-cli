package dotenv

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Entry
	}{
		{
			name:  "simple unquoted",
			input: "KEY=value",
			expected: []Entry{{Key: "KEY", Value: "value"}},
		},
		{
			name:  "double quoted with spaces",
			input: `DB_URL="postgres://user:pass@host/db"`,
			expected: []Entry{{Key: "DB_URL", Value: "postgres://user:pass@host/db"}},
		},
		{
			name:  "single quoted literal",
			input: `SECRET='raw $value "here"'`,
			expected: []Entry{{Key: "SECRET", Value: `raw $value "here"`}},
		},
		{
			name:  "double quoted escape sequences",
			input: `MSG="line1\nline2\\end"`,
			expected: []Entry{{Key: "MSG", Value: "line1\nline2\\end"}},
		},
		{
			name:  "empty value allowed",
			input: "EMPTY=",
			expected: []Entry{{Key: "EMPTY", Value: ""}},
		},
		{
			name:     "comment line skipped",
			input:    "# this is a comment",
			expected: nil,
		},
		{
			name:     "blank line skipped",
			input:    "   ",
			expected: nil,
		},
		{
			name:     "malformed line no equals",
			input:    "NOVALUE",
			expected: nil,
		},
		{
			name:  "inline comment after unquoted value",
			input: "KEY=value # comment",
			expected: []Entry{{Key: "KEY", Value: "value"}},
		},
		{
			name:  "trailing whitespace trimmed on unquoted",
			input: "KEY=value   ",
			expected: []Entry{{Key: "KEY", Value: "value"}},
		},
		{
			name:  "export prefix stripped",
			input: "export API_KEY=secret",
			expected: []Entry{{Key: "API_KEY", Value: "secret"}},
		},
		{
			name:  "multiline input",
			input: "A=1\n# comment\nB=2\n\nC=3",
			expected: []Entry{
				{Key: "A", Value: "1"},
				{Key: "B", Value: "2"},
				{Key: "C", Value: "3"},
			},
		},
		{
			name:  "duplicate keys last wins",
			input: "KEY=first\nKEY=second",
			expected: []Entry{
				{Key: "KEY", Value: "first"},
				{Key: "KEY", Value: "second"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, warnings := Parse(tt.input)
			if len(entries) != len(tt.expected) {
				t.Fatalf("expected %d entries, got %d (warnings: %v)", len(tt.expected), len(entries), warnings)
			}
			for i, e := range entries {
				if e.Key != tt.expected[i].Key || e.Value != tt.expected[i].Value {
					t.Errorf("entry %d: got {%q, %q}, want {%q, %q}", i, e.Key, e.Value, tt.expected[i].Key, tt.expected[i].Value)
				}
			}
		})
	}
}

func TestParseMalformedWarnings(t *testing.T) {
	input := "GOOD=value\nBADLINE\nALSO_GOOD=ok"
	entries, warnings := Parse(input)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
}
