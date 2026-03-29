package cmd

import (
	"testing"
)

func TestSecretAudit_EmptyValue(t *testing.T) {
	secrets := map[string]string{
		"myapp/empty_key": "",
	}

	findings := secretAudit(secrets)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Severity != severityWarn {
		t.Errorf("expected severity WARN, got %s", findings[0].Severity)
	}
	if findings[0].Message != "empty value" {
		t.Errorf("expected 'empty value' message, got %s", findings[0].Message)
	}
}

func TestSecretAudit_WeakValue(t *testing.T) {
	secrets := map[string]string{
		"myapp/db_pass": "short",
	}

	findings := secretAudit(secrets)

	foundWeak := false
	for _, f := range findings {
		if f.Severity == severityWarn && f.Message == "value length < 16 chars (weak secret)" {
			foundWeak = true
		}
	}
	if !foundWeak {
		t.Error("expected weak value warning, not found")
	}
}

func TestSecretAudit_StrongValue(t *testing.T) {
	secrets := map[string]string{
		"myapp/api_key": "this-is-a-very-long-secret-value-1234",
	}

	findings := secretAudit(secrets)

	for _, f := range findings {
		if f.Severity == severityWarn {
			t.Errorf("unexpected warning: %s - %s", f.Path, f.Message)
		}
	}
}

func TestSecretAudit_DuplicateValues(t *testing.T) {
	secrets := map[string]string{
		"aws/access_key":         "duplicate-value-that-is-long-enough",
		"aws-staging/access_key": "duplicate-value-that-is-long-enough",
	}

	findings := secretAudit(secrets)

	foundDuplicate := false
	for _, f := range findings {
		if f.Severity == severityInfo && f.Message == "duplicate value found in aws/access_key" {
			foundDuplicate = true
		}
	}
	if !foundDuplicate {
		t.Error("expected duplicate value info finding, not found")
	}
}

func TestSecretAudit_ShortKeyName(t *testing.T) {
	secrets := map[string]string{
		"myapp/x": "this-is-a-long-enough-secret-value",
	}

	findings := secretAudit(secrets)

	foundShortKey := false
	for _, f := range findings {
		if f.Severity == severityInfo && f.Message == "key name very short (< 3 chars)" {
			foundShortKey = true
		}
	}
	if !foundShortKey {
		t.Error("expected short key name info finding, not found")
	}
}

func TestSecretAudit_NoIssues(t *testing.T) {
	secrets := map[string]string{
		"myapp/database_url": "postgres://user:pass@host:5432/db",
		"myapp/api_token":    "sk-1234567890abcdefghij",
	}

	findings := secretAudit(secrets)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(findings))
		for _, f := range findings {
			t.Logf("  [%s] %s - %s", f.Severity, f.Path, f.Message)
		}
	}
}

func TestSecretAudit_SortOrder(t *testing.T) {
	secrets := map[string]string{
		"proj/short_val":  "abc",                                // WARN: weak
		"proj/info_key":   "this-is-a-long-enough-secret-value", // no issue
		"proj/ab":         "this-is-a-long-enough-secret-value", // INFO: short key + INFO: duplicate
		"proj/empty_val":  "",                                   // WARN: empty
	}

	findings := secretAudit(secrets)

	// Verify WARN findings come before INFO findings
	seenInfo := false
	for _, f := range findings {
		if f.Severity == severityInfo {
			seenInfo = true
		}
		if f.Severity == severityWarn && seenInfo {
			t.Error("WARN finding found after INFO finding; expected all WARNs first")
		}
	}
}

func TestSecretAudit_EmptyValueSkipsWeakCheck(t *testing.T) {
	secrets := map[string]string{
		"myapp/empty": "",
	}

	findings := secretAudit(secrets)

	warnCount := 0
	for _, f := range findings {
		if f.Severity == severityWarn {
			warnCount++
		}
	}
	// Should only get one WARN (empty), not two (empty + weak)
	if warnCount != 1 {
		t.Errorf("expected 1 warning for empty value, got %d", warnCount)
	}
}

func TestSecretAudit_ExactBoundary16Chars(t *testing.T) {
	secrets := map[string]string{
		"proj/key_fifteen": "123456789012345",  // 15 chars - weak
		"proj/key_sixteen": "1234567890123456", // 16 chars - not weak
	}

	findings := secretAudit(secrets)

	weakPaths := make(map[string]bool)
	for _, f := range findings {
		if f.Message == "value length < 16 chars (weak secret)" {
			weakPaths[f.Path] = true
		}
	}

	if !weakPaths["proj/key_fifteen"] {
		t.Error("expected 15-char value to be flagged as weak")
	}
	if weakPaths["proj/key_sixteen"] {
		t.Error("expected 16-char value to NOT be flagged as weak")
	}
}
