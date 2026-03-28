package utils

import (
	"fmt"
	"time"

	"github.com/atotto/clipboard"
)

// CopyToClipboard copies text to clipboard and blocks until the TTL expires,
// then clears the clipboard. The caller must wait for this function to return
// to guarantee clipboard clearing.
func CopyToClipboard(text string, ttl time.Duration) error {
	// Copy to clipboard
	if err := clipboard.WriteAll(text); err != nil {
		return fmt.Errorf("failed to copy to clipboard: %w", err)
	}

	// Block until TTL expires, then clear clipboard
	time.Sleep(ttl)
	_ = clipboard.WriteAll("")

	return nil
}
