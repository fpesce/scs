package io

import (
	"fmt"
	"os"
)

// WriteResult writes a single string to the specified file.
func WriteResult(filepath string, content string) error {
	if err := os.WriteFile(filepath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing to %q: %w", filepath, err)
	}
	return nil
}
