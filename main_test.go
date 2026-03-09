package main

import (
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joke/scs/pipeline"
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
			outputPath := filepath.Join(dir, "output.scs")

			// Write input file.
			content := strings.Join(tt.input, "\n")
			if err := os.WriteFile(inputPath, []byte(content), 0o644); err != nil {
				t.Fatalf("writing input: %v", err)
			}

			// Run the pipeline via the run function with build subcommand.
			args := []string{
				"build",
				"-i", inputPath,
				"-o", outputPath,
				"-k", itoa(tt.minOverlap),
				"--dp-limit", itoa(tt.dpLimit),
			}
			if err := run(args); err != nil {
				t.Fatalf("run failed: %v", err)
			}

			// Read the output file.
			got, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("reading output: %v", err)
			}

			// Verify 3-line format.
			fileLines := strings.SplitAfter(string(got), "\n")
			// SplitAfter keeps the delimiter; we expect exactly 3 lines + possibly empty trailing.
			nonEmpty := 0
			for _, l := range fileLines {
				if l != "" {
					nonEmpty++
				}
			}
			if nonEmpty != 3 {
				t.Fatalf("expected 3 lines, got %d (raw: %q)", nonEmpty, string(got))
			}

			// Line 1: should be exactly 16 Base64 chars + \n.
			line1 := strings.TrimSuffix(fileLines[0], "\n")
			if len(line1) != 16 {
				t.Errorf("Line 1 length = %d, want 16", len(line1))
			}

			// Line 2: the superstring.
			line2 := strings.TrimSuffix(fileLines[1], "\n")
			for _, s := range tt.wantContains {
				if !strings.Contains(line2, s) {
					t.Errorf("Line 2 (superstring) %q does not contain %q", line2, s)
				}
			}

			// Line 3: should be valid Base64.
			line3 := strings.TrimSuffix(fileLines[2], "\n")
			if _, err := base64.StdEncoding.DecodeString(line3); err != nil {
				t.Errorf("Line 3 is not valid Base64: %v", err)
			}
		})
	}
}

// TestBuildUnorderedMode tests that --unordered produces a valid 3-line format.
func TestBuildUnorderedMode(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.txt")
	outputPath := filepath.Join(dir, "output.scs")

	content := "hello\nworld\nfoo\nbar"
	if err := os.WriteFile(inputPath, []byte(content), 0o644); err != nil {
		t.Fatalf("writing input: %v", err)
	}

	args := []string{"build", "-i", inputPath, "-o", outputPath, "--unordered"}
	if err := run(args); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	parts := strings.Split(strings.TrimSuffix(string(got), "\n"), "\n")
	if len(parts) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(parts))
	}

	// Verify Line 1 is valid Base64 of length 16.
	if len(parts[0]) != 16 {
		t.Errorf("Line 1 length = %d, want 16", len(parts[0]))
	}

	// Verify Line 3 is valid Base64.
	if _, err := base64.StdEncoding.DecodeString(parts[2]); err != nil {
		t.Errorf("Line 3 is not valid Base64: %v", err)
	}
}

// TestAllWordsAreIncluded is a pipeline-level E2E test with chained overlaps,
// duplicates, and a compression assertion.
func TestAllWordsAreIncluded(t *testing.T) {
	inputWords := []string{
		"password", "swordfish", "fishbone", "123456", "admin", "duplicate", "duplicate",
	}

	// Run pipeline pieces exactly as main.go does.
	survivors := pipeline.ExactDeduplication(inputWords)
	survivors = pipeline.EliminateSubstrings(survivors)
	islands := pipeline.ShatterGraph(survivors, 3)
	superWords := pipeline.AssembleConcurrently(islands, 15, 3, false)

	finalSuperstring := strings.Join(superWords, "")

	// Verify integrity: every original word must exist in the result.
	for _, word := range inputWords {
		if !strings.Contains(finalSuperstring, word) {
			t.Errorf("final superstring is missing word: %q", word)
		}
	}

	// Verify compression: naive concat of unique words is 53 chars.
	naiveLen := 0
	seen := make(map[string]bool)
	for _, w := range inputWords {
		if !seen[w] {
			naiveLen += len(w)
			seen[w] = true
		}
	}
	if len(finalSuperstring) >= naiveLen {
		t.Errorf("pipeline did not compress! got len=%d, naive=%d, result=%q",
			len(finalSuperstring), naiveLen, finalSuperstring)
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

// TestStrictFidelityRoundTrip verifies that build → cat produces byte-for-byte
// identical output to the original source file.
func TestStrictFidelityRoundTrip(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "source.txt")
	scsPath := filepath.Join(dir, "test.scs")

	// Create a source file with various edge cases: empty lines, whitespace.
	source := "hello\nworld\n\nfoo\nbar\nbaz\n"
	if err := os.WriteFile(inputPath, []byte(source), 0o644); err != nil {
		t.Fatalf("writing input: %v", err)
	}

	// Build the .scs file.
	if err := run([]string{"build", "-i", inputPath, "-o", scsPath}); err != nil {
		t.Fatalf("build failed: %v", err)
	}

	// Build the scs binary if needed for exec, or use run directly.
	// We'll build a temp binary.
	binPath := filepath.Join(dir, "scs")
	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = "."
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("building scs binary: %v\n%s", err, out)
	}

	// Run cat and capture output.
	catCmd := exec.Command(binPath, "cat", scsPath)
	catOutput, err := catCmd.Output()
	if err != nil {
		t.Fatalf("cat failed: %v", err)
	}

	// Compare byte-for-byte.
	if string(catOutput) != source {
		t.Errorf("round-trip mismatch!\ngot:  %q\nwant: %q", string(catOutput), source)
	}
}

// TestSearchExitCodes verifies search returns correct exit codes.
func TestSearchExitCodes(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "source.txt")
	scsPath := filepath.Join(dir, "test.scs")

	// Source contains "password", "swordfish", "fishbone".
	// "sword" is an incidental substring (not a dataset member).
	source := "password\nswordfish\nfishbone\n"
	if err := os.WriteFile(inputPath, []byte(source), 0o644); err != nil {
		t.Fatalf("writing input: %v", err)
	}

	// Build the .scs file.
	if err := run([]string{"build", "-i", inputPath, "-o", scsPath}); err != nil {
		t.Fatalf("build failed: %v", err)
	}

	// Build a temp binary.
	binPath := filepath.Join(dir, "scs")
	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = "."
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("building scs binary: %v\n%s", err, out)
	}

	// Test 1: True positive - "password" should return exit 0.
	cmd := exec.Command(binPath, "search", "password", scsPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("search 'password' should exit 0, got error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(string(out), "Match found") {
		t.Errorf("expected 'Match found', got %q", string(out))
	}

	// Test 2: Non-existent word - "zzzzz" should return exit 1.
	cmd2 := exec.Command(binPath, "search", "zzzzz", scsPath)
	out2, err2 := cmd2.CombinedOutput()
	if err2 == nil {
		t.Errorf("search 'zzzzz' should exit 1, but exited 0\noutput: %s", out2)
	}
	if !strings.Contains(string(out2), "Not found") {
		t.Errorf("expected 'Not found', got %q", string(out2))
	}

	// Test 3: False positive - "sword" is a substring of "swordfish"
	// but is NOT a dataset member. Should return exit 1.
	cmd3 := exec.Command(binPath, "search", "sword", scsPath)
	out3, err3 := cmd3.CombinedOutput()
	if err3 == nil {
		t.Errorf("search 'sword' should exit 1 (false positive filtered), but exited 0\noutput: %s", out3)
	}
	if !strings.Contains(string(out3), "Not found") {
		t.Errorf("expected 'Not found', got %q", string(out3))
	}
}
