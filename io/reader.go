// Package io provides file reading and writing utilities for the SCS tool.
package io

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ReadLines reads a file and returns all non-empty lines.
// Lines consisting solely of whitespace are dropped.
func ReadLines(filepath string) ([]string, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("opening %q: %w", filepath, err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning %q: %w", filepath, err)
	}

	return lines, nil
}
