package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEndToEnd(t *testing.T) {
	tests := []struct {
		name         string
		input        []string
		minOverlap   int
		dpLimit      int
		wantContains []string
	}{
		{
			name: "basic overlapping strings",
			input: []string{
				"abcdef",
				"defghi",
				"ghijkl",
				"abcdef", // duplicate
				"def",    // substring
			},
			minOverlap:   3,
			dpLimit:      15,
			wantContains: []string{"abcdef", "defghi", "ghijkl"},
		},
		{
			name: "disjoint strings",
			input: []string{
				"alpha",
				"bravo",
				"charlie",
			},
			minOverlap:   3,
			dpLimit:      15,
			wantContains: []string{"alpha", "bravo", "charlie"},
		},
		{
			name: "single string",
			input: []string{
				"hello",
			},
			minOverlap:   3,
			dpLimit:      15,
			wantContains: []string{"hello"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			inputPath := filepath.Join(dir, "input.txt")
			outputPath := filepath.Join(dir, "output.txt")

			// Write input file.
			content := strings.Join(tt.input, "\n")
			if err := os.WriteFile(inputPath, []byte(content), 0o644); err != nil {
				t.Fatalf("writing input: %v", err)
			}

			// Run the pipeline via the run function.
			args := []string{
				"-i", inputPath,
				"-o", outputPath,
				"-k", itoa(tt.minOverlap),
				"--dp-limit", itoa(tt.dpLimit),
				"-v",
			}
			if err := run(args); err != nil {
				t.Fatalf("run failed: %v", err)
			}

			// Read the output file.
			got, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("reading output: %v", err)
			}
			result := string(got)

			// Verify all expected strings are contained in the output.
			for _, s := range tt.wantContains {
				if !strings.Contains(result, s) {
					t.Errorf("output %q does not contain %q", result, s)
				}
			}

			// Verify the output is a single line (no newlines in the superstring).
			if strings.Contains(result, "\n") {
				t.Errorf("output should be a single superstring, but contains newlines: %q", result)
			}
		})
	}
}

// itoa is a minimal int-to-string converter to avoid importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
