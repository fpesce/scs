package io

import (
	"fmt"
	"os"

	"github.com/joke/scs/format"
)

// WriteResult writes a single string to the specified file.
func WriteResult(filepath string, content string) error {
	if err := os.WriteFile(filepath, []byte(content), format.FilePermissions); err != nil {
		return fmt.Errorf("writing to %q: %w", filepath, err)
	}
	return nil
}
