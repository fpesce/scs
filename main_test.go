package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joke/scs/cli"
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

			// Verify binary format: first 12 bytes are the header.
			if len(got) < 12 {
				t.Fatalf("output too short: %d bytes", len(got))
			}

			// Check magic string "SCS".
			if string(got[0:3]) != "SCS" {
				t.Errorf("magic = %q, want 'SCS'", string(got[0:3]))
			}

			// Check version 0x02.
			if got[3] != 0x02 {
				t.Errorf("version = 0x%02X, want 0x02", got[3])
			}

			// Extract superstring via footer offset.
			// Bytes 5-11 contain the 56-bit LE footer offset field.
			var leBytes [8]byte
			copy(leBytes[0:7], got[5:12])
			val := uint64(leBytes[0]) | uint64(leBytes[1])<<8 | uint64(leBytes[2])<<16 |
				uint64(leBytes[3])<<24 | uint64(leBytes[4])<<32 | uint64(leBytes[5])<<40 |
				uint64(leBytes[6])<<48
			footerOffset := val & ((1 << 55) - 1)

			if footerOffset > uint64(len(got)) || footerOffset < 12 {
				t.Fatalf("footer offset %d out of bounds (file size %d)", footerOffset, len(got))
			}

			superstring := string(got[12:footerOffset])
			for _, s := range tt.wantContains {
				if !strings.Contains(superstring, s) {
					t.Errorf("superstring %q does not contain %q", superstring, s)
				}
			}
		})
	}
}

// TestBuildUnorderedMode tests that --unordered produces a valid binary format.
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

	// Verify binary header.
	if len(got) < 12 {
		t.Fatalf("output too short: %d bytes", len(got))
	}
	if string(got[0:3]) != "SCS" {
		t.Errorf("magic = %q, want 'SCS'", string(got[0:3]))
	}
	if got[3] != 0x02 {
		t.Errorf("version = 0x%02X, want 0x02", got[3])
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
	superWords := pipeline.AssembleConcurrently(islands, &cli.BuildConfig{
		DPLimit:    15,
		MinOverlap: 3,
	})

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

	// Build a temp binary.
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

	source := "password\nswordfish\nfishbone\n"
	if err := os.WriteFile(inputPath, []byte(source), 0o644); err != nil {
		t.Fatalf("writing input: %v", err)
	}

	if err := run([]string{"build", "-i", inputPath, "-o", scsPath}); err != nil {
		t.Fatalf("build failed: %v", err)
	}

	binPath := filepath.Join(dir, "scs")
	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = "."
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("building scs binary: %v\n%s", err, out)
	}

	// Test 1: True positive.
	cmd := exec.Command(binPath, "search", "password", scsPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("search 'password' should exit 0, got error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(string(out), "Match found") {
		t.Errorf("expected 'Match found', got %q", string(out))
	}

	// Test 2: Non-existent word.
	cmd2 := exec.Command(binPath, "search", "zzzzz", scsPath)
	out2, err2 := cmd2.CombinedOutput()
	if err2 == nil {
		t.Errorf("search 'zzzzz' should exit 1, but exited 0\noutput: %s", out2)
	}
	if !strings.Contains(string(out2), "Not found") {
		t.Errorf("expected 'Not found', got %q", string(out2))
	}

	// Test 3: False positive.
	cmd3 := exec.Command(binPath, "search", "sword", scsPath)
	out3, err3 := cmd3.CombinedOutput()
	if err3 == nil {
		t.Errorf("search 'sword' should exit 1 (false positive filtered), but exited 0\noutput: %s", out3)
	}
	if !strings.Contains(string(out3), "Not found") {
		t.Errorf("expected 'Not found', got %q", string(out3))
	}
}
