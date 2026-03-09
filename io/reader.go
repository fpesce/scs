// Package io provides file reading and writing utilities for the SCS tool.
package io

import (
	"bufio"
	"fmt"
	"os"
)

// ReadLines reads a file and returns every line with exact byte fidelity.
// Whitespace, tabs, and empty lines are all preserved.
// Only the trailing newline delimiter is stripped.
func ReadLines(filepath string) ([]string, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("opening %q: %w", filepath, err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning %q: %w", filepath, err)
	}

	return lines, nil
}
