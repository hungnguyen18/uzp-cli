package rotation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const rotationFileName = "rotation.json"

// Policy defines the rotation period for a secret.
type Policy struct {
	PeriodDays int `json:"period_days"`
}

// RotationData holds all rotation policies and last-updated timestamps.
type RotationData struct {
	Policies    map[string]Policy `json:"policies"`
	LastUpdated map[string]string `json:"last_updated"`
}

// Store manages rotation policy persistence.
type Store struct {
	path string
	data *RotationData
}

// NewStore creates a new rotation store using the default ~/.uzp directory.
func NewStore() (*Store, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to determine home directory: %w", err)
	}

	storePath := filepath.Join(homeDir, ".uzp", rotationFileName)

	return &Store{
		path: storePath,
	}, nil
}

// Load reads the rotation data from disk. Returns empty data if file does not exist.
func (s *Store) Load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.data = &RotationData{
				Policies:    make(map[string]Policy),
				LastUpdated: make(map[string]string),
			}
			return nil
		}
		return fmt.Errorf("failed to read rotation file: %w", err)
	}

	var rd RotationData
	if err := json.Unmarshal(data, &rd); err != nil {
		return fmt.Errorf("failed to parse rotation file: %w", err)
	}

	if rd.Policies == nil {
		rd.Policies = make(map[string]Policy)
	}
	if rd.LastUpdated == nil {
		rd.LastUpdated = make(map[string]string)
	}

	s.data = &rd
	return nil
}

// Save writes the rotation data to disk with atomic write and 0600 permissions.
func (s *Store) Save() error {
	if s.data == nil {
		return fmt.Errorf("no rotation data to save")
	}

	jsonData, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal rotation data: %w", err)
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create rotation directory: %w", err)
	}

	// Atomic write: temp file then rename
	tmpFile, err := os.CreateTemp(dir, "rotation.*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(jsonData); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write rotation data: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to sync rotation data: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Chmod(tmpPath, 0600); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	if err := os.Rename(tmpPath, s.path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to save rotation file: %w", err)
	}

	return nil
}

// SetPolicy sets a rotation policy for a secret key (project/key format).
// If no last_updated exists for the key, records the current time.
func (s *Store) SetPolicy(secretPath string, periodDays int) {
	s.data.Policies[secretPath] = Policy{PeriodDays: periodDays}

	if _, exists := s.data.LastUpdated[secretPath]; !exists {
		s.data.LastUpdated[secretPath] = time.Now().UTC().Format(time.RFC3339)
	}
}

// UnsetPolicy removes the rotation policy for a secret key.
// Keeps the last_updated timestamp for historical reference.
func (s *Store) UnsetPolicy(secretPath string) {
	delete(s.data.Policies, secretPath)
}

// GetPolicy returns the policy for a secret, or nil if none is set.
func (s *Store) GetPolicy(secretPath string) *Policy {
	p, ok := s.data.Policies[secretPath]
	if !ok {
		return nil
	}
	return &p
}

// GetLastUpdated returns the last-updated time for a secret, or zero time if unknown.
func (s *Store) GetLastUpdated(secretPath string) (time.Time, bool) {
	ts, ok := s.data.LastUpdated[secretPath]
	if !ok {
		return time.Time{}, false
	}

	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return time.Time{}, false
	}

	return t, true
}

// RecordUpdate sets the last_updated timestamp for a secret to now.
func (s *Store) RecordUpdate(secretPath string) {
	s.data.LastUpdated[secretPath] = time.Now().UTC().Format(time.RFC3339)
}

// AllPolicies returns a copy of all policies.
func (s *Store) AllPolicies() map[string]Policy {
	result := make(map[string]Policy, len(s.data.Policies))
	for k, v := range s.data.Policies {
		result[k] = v
	}
	return result
}

// AllLastUpdated returns a copy of all last-updated timestamps.
func (s *Store) AllLastUpdated() map[string]string {
	result := make(map[string]string, len(s.data.LastUpdated))
	for k, v := range s.data.LastUpdated {
		result[k] = v
	}
	return result
}

// DaysAgo returns how many days ago a given time was, relative to now.
func DaysAgo(t time.Time) int {
	return int(time.Since(t).Hours() / 24)
}

// ParseDuration parses a duration string like "90d" and returns the number of days.
func ParseDuration(s string) (int, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration format: %q (use e.g. 90d)", s)
	}

	suffix := s[len(s)-1]
	if suffix != 'd' {
		return 0, fmt.Errorf("unsupported duration unit %q (only 'd' for days is supported)", string(suffix))
	}

	numStr := s[:len(s)-1]
	var days int
	for i := 0; i < len(numStr); i++ {
		c := numStr[i]
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid duration format: %q (use e.g. 90d)", s)
		}
		days = days*10 + int(c-'0')
	}

	if days <= 0 {
		return 0, fmt.Errorf("duration must be a positive number of days")
	}

	return days, nil
}
