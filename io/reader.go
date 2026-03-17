// Package io provides file reading and writing utilities for the SCS tool.
package io

import (
	"bytes"
	"fmt"
	"os"
)

// ReadSeparated reads a file and splits it by the given separator.
// It performs a single os.ReadFile followed by bytes.Split, making it
// binary-safe for arbitrary byte payloads (including embedded \n and \0).
// A trailing empty element (standard EOF separator) is dropped.
func ReadSeparated(filepath string, sep []byte) ([]string, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("reading %q: %w", filepath, err)
	}

	parts := bytes.Split(data, sep)

	// Drop trailing empty element from a terminating separator.
	if len(parts) > 0 && len(parts[len(parts)-1]) == 0 {
		parts = parts[:len(parts)-1]
	}

	lines := make([]string, len(parts))
	for i, p := range parts {
		lines[i] = string(p)
	}

	return lines, nil
}

// ReadLines reads a file line-by-line (splitting on \n).
// Thin wrapper over ReadSeparated for backward compatibility.
func ReadLines(filepath string) ([]string, error) {
	return ReadSeparated(filepath, []byte("\n"))
}
