package main

import (
	"fmt"
	"os"
	"runtime/pprof"
	"sort"
	"strings"

	"github.com/joke/scs/cli"
	"github.com/joke/scs/format"
	scsio "github.com/joke/scs/io"
	"github.com/joke/scs/pipeline"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cmd, cfg, err := cli.ParseSubcommand(args)
	if err != nil {
		return fmt.Errorf("configuration: %w", err)
	}

	switch cmd {
	case "build":
		return runBuild(cfg.(*cli.BuildConfig))
	case "merge":
		return runMerge(cfg.(*cli.MergeConfig))
	case "cat":
		return runCat(cfg.(*cli.CatConfig))
	case "search":
		return runSearch(cfg.(*cli.SearchConfig))
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

func runBuild(cfg *cli.BuildConfig) error {
	// Optional CPU profiling.
	if cfg.Profile {
		f, err := os.Create("cpu.prof")
		if err != nil {
			return fmt.Errorf("creating profile: %w", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			return fmt.Errorf("starting profile: %w", err)
		}
		defer pprof.StopCPUProfile()
	}

	lines, err := scsio.ReadLines(cfg.InputPath)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}
	if cfg.Verbose {
		fmt.Printf("Read %d lines from %s\n", len(lines), cfg.InputPath)
	}

	// Phase 1A: Hash deduplication — get unique set for pipeline.
	survivors := pipeline.ExactDeduplication(lines)
	if cfg.Verbose {
		fmt.Printf("Phase 1A: %d → %d strings (removed %d exact duplicates)\n",
			len(lines), len(survivors), len(lines)-len(survivors))
	}

	// Phase 1B: Substring elimination.
	beforeSubstring := len(survivors)
	survivors = pipeline.EliminateSubstrings(survivors)
	if cfg.Verbose {
		fmt.Printf("Phase 1B: %d → %d strings (removed %d substrings)\n",
			beforeSubstring, len(survivors), beforeSubstring-len(survivors))
	}

	// Phase 2 & 3: Partition into weakly connected components.
	islands := pipeline.ShatterGraph(survivors, cfg.MinOverlap)
	if cfg.Verbose {
		fmt.Printf("Phase 2-3: shattered into %d isolated islands\n", len(islands))
	}

	// Phase 4: Solve islands concurrently via worker pool.
	superWords := pipeline.AssembleConcurrently(islands, cfg.DPLimit, cfg.MinOverlap, cfg.Verbose)
	if cfg.Verbose {
		fmt.Printf("Phase 4: assembled %d super-words\n", len(superWords))
	}

	// Phase 5: Sort by length descending with lexicographical tie-breaker.
	sort.Slice(superWords, func(i, j int) bool {
		if len(superWords[i]) == len(superWords[j]) {
			return superWords[i] < superWords[j]
		}
		return len(superWords[i]) > len(superWords[j])
	})

	var builder strings.Builder
	var totalLen int
	for _, sw := range superWords {
		totalLen += len(sw)
	}
	builder.Grow(totalLen)

	for i, sw := range superWords {
		builder.WriteString(sw)
		if cfg.Verbose {
			fmt.Printf("\r  Concatenating... %d/%d (%d%%)",
				i+1, len(superWords), (i+1)*100/len(superWords))
		}
	}
	if cfg.Verbose && len(superWords) > 0 {
		fmt.Println()
	}

	masterString := builder.String()

	// --- Format Generation: Build the 3-line .scs file ---

	// Get the deduplicated unique strings (for mapping).
	uniqueStrings := pipeline.ExactDeduplication(lines)
	// Filter out empty strings for mapping purposes.
	var nonEmptyUnique []string
	for _, s := range uniqueStrings {
		if s != "" {
			nonEmptyUnique = append(nonEmptyUnique, s)
		}
	}

	// Map offsets via Aho-Corasick single-pass.
	offsetMap := format.MapOffsets(masterString, nonEmptyUnique)

	// Generate the metadata footer (Line 3).
	var footer string
	if cfg.Unordered {
		footer = format.EncodeUnordered(nonEmptyUnique, offsetMap)
	} else {
		footer = format.EncodeOrdered(lines, offsetMap, len(masterString))
	}

	// Calculate the byte offset for Line 3.
	// Line 1: 16 chars (Base64 header) + 1 (\n)
	// Line 2: len(masterString) chars + 1 (\n)
	footerOffset := uint64(16 + 1 + len(masterString) + 1)

	// Determine separator byte.
	sepByte := byte('\n')
	if cfg.Separator != "\n" && len(cfg.Separator) > 0 {
		sepByte = cfg.Separator[0]
	}

	// Generate the header (Line 1).
	header := format.EncodeHeader(&format.Header{
		Version:      0x01,
		Separator:    sepByte,
		IsOrdered:    !cfg.Unordered,
		FooterOffset: footerOffset,
	})

	// Write exactly 3 lines, each terminated by \n.
	f, err := os.Create(cfg.OutputPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, "%s\n%s\n%s\n", header, masterString, footer); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	if cfg.Verbose {
		fmt.Printf("Final superstring: %d characters written to %s\n",
			len(masterString), cfg.OutputPath)
	}

	return nil
}

func runMerge(cfg *cli.MergeConfig) error {
	// Decode both files.
	header1, superstring1, words1, err := format.DecodeSCS(cfg.PrimaryPath)
	if err != nil {
		return fmt.Errorf("decoding primary %q: %w", cfg.PrimaryPath, err)
	}

	header2, _, words2, err := format.DecodeSCS(cfg.UpdatePath)
	if err != nil {
		return fmt.Errorf("decoding update %q: %w", cfg.UpdatePath, err)
	}

	// Extract the word strings from File 2.
	var updateWordStrings []string
	for _, w := range words2 {
		if w.String != "" {
			updateWordStrings = append(updateWordStrings, w.String)
		}
	}

	// Step 1: Eliminate substrings already present in File 1's superstring.
	survivors, eliminatedMap := pipeline.MergeEliminateSubstrings(superstring1, updateWordStrings)

	// Step 2: Generate mini-superstring from survivors and calculate overlap.
	fragment, overlapLength := pipeline.TruncateAndAppend(superstring1, survivors, 3, 15)

	// Step 3: Build the combined superstring.
	combinedPayload := superstring1 + fragment

	// Step 4: Build the combined word list with correct offsets.
	// Start with all words from File 1 (offsets unchanged).
	var allWords []format.Word
	allWords = append(allWords, words1...)

	// Add eliminated words (remapped to primary's address space).
	for word, offset := range eliminatedMap {
		allWords = append(allWords, format.Word{
			String: word,
			Offset: offset,
			Length: len(word),
		})
	}

	// Map surviving words' offsets in the new combined payload.
	if len(survivors) > 0 && fragment != "" {
		// Get the unique survivors for mapping.
		uniqueSurvivors := pipeline.ExactDeduplication(survivors)
		var nonEmpty []string
		for _, s := range uniqueSurvivors {
			if s != "" {
				nonEmpty = append(nonEmpty, s)
			}
		}

		// Map offsets within the combined payload for surviving words.
		// We need to find them in the appended portion.
		survivorOffsets := format.MapOffsets(combinedPayload, nonEmpty)

		for _, surv := range survivors {
			if offset, ok := survivorOffsets[surv]; ok {
				allWords = append(allWords, format.Word{
					String: surv,
					Offset: offset,
					Length: len(surv),
				})
			}
		}
	}

	// Step 5: Graceful Downgrade — if either file is UNORDERED, force UNORDERED.
	isOrdered := header1.IsOrdered && header2.IsOrdered

	// Step 6: Build the merged footer.
	// Build an offset map from allWords.
	offsetMap := make(map[string]int)
	for _, w := range allWords {
		if w.String != "" {
			if _, exists := offsetMap[w.String]; !exists {
				offsetMap[w.String] = w.Offset
			}
		}
	}

	var footer string
	if !isOrdered {
		// UNORDERED mode: collect unique words.
		uniqueWords := make([]string, 0)
		seen := make(map[string]bool)
		for _, w := range allWords {
			if w.String != "" && !seen[w.String] {
				seen[w.String] = true
				uniqueWords = append(uniqueWords, w.String)
			}
		}
		footer = format.EncodeUnordered(uniqueWords, offsetMap)
	} else {
		// ORDERED mode: preserve chronological order from both files.
		var chronoLines []string
		for _, w := range allWords {
			chronoLines = append(chronoLines, w.String)
		}
		footer = format.EncodeOrdered(chronoLines, offsetMap, len(combinedPayload))
	}

	// Step 7: Calculate footer offset and build header.
	footerOffset := uint64(16 + 1 + len(combinedPayload) + 1)
	mergedHeader := format.EncodeHeader(&format.Header{
		Version:      0x01,
		Separator:    header1.Separator,
		IsOrdered:    isOrdered,
		FooterOffset: footerOffset,
	})

	// Step 8: Write the merged 3-line file.
	f, err := os.Create(cfg.OutputPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, "%s\n%s\n%s\n", mergedHeader, combinedPayload, footer); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	if cfg.Verbose {
		fmt.Printf("Merged: %d + %d words → %d characters written to %s\n",
			len(words1), len(words2), len(combinedPayload), cfg.OutputPath)
	}

	_ = overlapLength // Used for the truncation; the combined payload already reflects it.

	return nil
}

func runCat(cfg *cli.CatConfig) error {
	header, _, words, err := format.DecodeSCS(cfg.FilePath)
	if err != nil {
		return fmt.Errorf("decoding %q: %w", cfg.FilePath, err)
	}

	sep := string(header.Separator)

	for i, w := range words {
		if i > 0 {
			fmt.Print(sep)
		}
		fmt.Print(w.String)
	}
	// Trailing separator (to match the original file that ends with \n).
	if len(words) > 0 {
		fmt.Print(sep)
	}

	return nil
}

func runSearch(cfg *cli.SearchConfig) error {
	// Read the file to get the superstring.
	data, err := os.ReadFile(cfg.FilePath)
	if err != nil {
		return fmt.Errorf("reading %q: %w", cfg.FilePath, err)
	}

	content := string(data)
	parts := strings.SplitN(content, "\n", 4)
	if len(parts) < 3 {
		return fmt.Errorf("invalid .scs file: expected 3 lines")
	}

	superstring := parts[1] // Line 2

	// Bloom Filter Fast-Path: native substring check.
	if !strings.Contains(superstring, cfg.Word) {
		fmt.Println("Not found")
		os.Exit(1)
	}

	// Verification Slow-Path: decode metadata and check boundaries.
	_, _, words, err := format.DecodeSCS(cfg.FilePath)
	if err != nil {
		return fmt.Errorf("decoding %q: %w", cfg.FilePath, err)
	}

	for _, w := range words {
		if w.String == cfg.Word {
			fmt.Println("Match found")
			return nil // Exit 0.
		}
	}

	// False positive: exists as substring but not as a dataset member.
	fmt.Println("Not found")
	os.Exit(1)
	return nil
}
