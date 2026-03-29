package rotation

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func tempStorePath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "rotation.json")
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s := &Store{path: tempStorePath(t)}
	if err := s.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	return s
}

func TestLoadNonExistentFile(t *testing.T) {
	s := &Store{path: filepath.Join(t.TempDir(), "missing.json")}
	if err := s.Load(); err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}

	if len(s.data.Policies) != 0 {
		t.Errorf("expected empty policies, got %d", len(s.data.Policies))
	}
	if len(s.data.LastUpdated) != 0 {
		t.Errorf("expected empty last_updated, got %d", len(s.data.LastUpdated))
	}
}

func TestSetAndGetPolicy(t *testing.T) {
	s := newTestStore(t)

	s.SetPolicy("myapp/api_key", 90)

	p := s.GetPolicy("myapp/api_key")
	if p == nil {
		t.Fatal("expected policy, got nil")
	}
	if p.PeriodDays != 90 {
		t.Errorf("expected 90 days, got %d", p.PeriodDays)
	}

	// SetPolicy should also set last_updated if not present
	_, ok := s.GetLastUpdated("myapp/api_key")
	if !ok {
		t.Error("expected last_updated to be set after SetPolicy")
	}
}

func TestSetPolicyPreservesExistingLastUpdated(t *testing.T) {
	s := newTestStore(t)

	past := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	s.data.LastUpdated["myapp/api_key"] = past.Format(time.RFC3339)

	s.SetPolicy("myapp/api_key", 30)

	ts, ok := s.GetLastUpdated("myapp/api_key")
	if !ok {
		t.Fatal("expected last_updated to exist")
	}
	if !ts.Equal(past) {
		t.Errorf("expected last_updated to remain %v, got %v", past, ts)
	}
}

func TestUnsetPolicy(t *testing.T) {
	s := newTestStore(t)

	s.SetPolicy("myapp/api_key", 90)
	s.UnsetPolicy("myapp/api_key")

	if p := s.GetPolicy("myapp/api_key"); p != nil {
		t.Error("expected policy to be removed")
	}

	// last_updated should be preserved
	_, ok := s.GetLastUpdated("myapp/api_key")
	if !ok {
		t.Error("expected last_updated to be preserved after UnsetPolicy")
	}
}

func TestSaveAndLoad(t *testing.T) {
	path := tempStorePath(t)

	// Save
	s1 := &Store{path: path}
	if err := s1.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	s1.SetPolicy("proj/key1", 30)
	s1.SetPolicy("proj/key2", 60)
	if err := s1.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("expected permissions 0600, got %o", perm)
	}

	// Load into new store
	s2 := &Store{path: path}
	if err := s2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	p1 := s2.GetPolicy("proj/key1")
	if p1 == nil || p1.PeriodDays != 30 {
		t.Errorf("expected policy 30d for proj/key1, got %v", p1)
	}

	p2 := s2.GetPolicy("proj/key2")
	if p2 == nil || p2.PeriodDays != 60 {
		t.Errorf("expected policy 60d for proj/key2, got %v", p2)
	}
}

func TestRecordUpdate(t *testing.T) {
	s := newTestStore(t)

	before := time.Now().UTC().Truncate(time.Second)
	s.RecordUpdate("myapp/token")
	after := time.Now().UTC().Truncate(time.Second).Add(time.Second)

	ts, ok := s.GetLastUpdated("myapp/token")
	if !ok {
		t.Fatal("expected last_updated to exist")
	}

	if ts.Before(before) || ts.After(after) {
		t.Errorf("expected timestamp between %v and %v, got %v", before, after, ts)
	}
}

func TestDaysAgo(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		input    time.Time
		expected int
	}{
		{"today", now, 0},
		{"10 days ago", now.Add(-10 * 24 * time.Hour), 10},
		{"100 days ago", now.Add(-100 * 24 * time.Hour), 100},
	}

	for i := 0; i < len(tests); i++ {
		tc := tests[i]
		t.Run(tc.name, func(t *testing.T) {
			got := DaysAgo(tc.input)
			if got != tc.expected {
				t.Errorf("DaysAgo(%v) = %d, want %d", tc.input, got, tc.expected)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		wantErr  bool
	}{
		{"90d", 90, false},
		{"30d", 30, false},
		{"1d", 1, false},
		{"365d", 365, false},
		{"", 0, true},
		{"d", 0, true},
		{"0d", 0, true},
		{"90h", 0, true},
		{"abc", 0, true},
		{"90", 0, true},
		{"-5d", 0, true},
	}

	for i := 0; i < len(tests); i++ {
		tc := tests[i]
		t.Run(tc.input, func(t *testing.T) {
			got, err := ParseDuration(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseDuration(%q) expected error, got %d", tc.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseDuration(%q) unexpected error: %v", tc.input, err)
				return
			}
			if got != tc.expected {
				t.Errorf("ParseDuration(%q) = %d, want %d", tc.input, got, tc.expected)
			}
		})
	}
}

func TestAllPoliciesReturnsCopy(t *testing.T) {
	s := newTestStore(t)
	s.SetPolicy("a/b", 10)

	all := s.AllPolicies()
	all["a/b"] = Policy{PeriodDays: 999}

	// Original should be unchanged
	p := s.GetPolicy("a/b")
	if p.PeriodDays != 10 {
		t.Errorf("AllPolicies returned reference, not copy")
	}
}

func TestGetLastUpdatedInvalidFormat(t *testing.T) {
	s := newTestStore(t)
	s.data.LastUpdated["bad/key"] = "not-a-date"

	_, ok := s.GetLastUpdated("bad/key")
	if ok {
		t.Error("expected ok=false for invalid timestamp")
	}
}

func TestGetPolicyNonExistent(t *testing.T) {
	s := newTestStore(t)

	p := s.GetPolicy("nonexistent/key")
	if p != nil {
		t.Errorf("expected nil for nonexistent policy, got %v", p)
	}
}
