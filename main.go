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

	// --- Format Generation: Binary .scs file ---

	// Get the deduplicated unique strings (for mapping).
	uniqueStrings := pipeline.ExactDeduplication(lines)
	// Filter out empty strings for mapping purposes.
	var nonEmptyUnique []string
	for _, s := range uniqueStrings {
		if s != "" {
			nonEmptyUnique = append(nonEmptyUnique, s)
		}
	}

	// Map offsets via suffix array.
	offsetMap := format.MapOffsets(masterString, nonEmptyUnique)

	// Generate the raw binary metadata footer.
	var footer []byte
	if cfg.Unordered {
		footer = format.EncodeUnordered(nonEmptyUnique, offsetMap)
	} else {
		footer = format.EncodeOrdered(lines, offsetMap, len(masterString))
	}

	// Calculate precise byte offset: 12 header bytes + len(superstring).
	footerOffset := uint64(12 + len(masterString))

	// Determine separator byte.
	sepByte := byte('\n')
	if cfg.Separator != "\n" && len(cfg.Separator) > 0 {
		sepByte = cfg.Separator[0]
	}

	// Generate the raw 12-byte header.
	headerBytes := format.EncodeHeader(&format.Header{
		Version:      0x02,
		Separator:    sepByte,
		IsOrdered:    !cfg.Unordered,
		FooterOffset: footerOffset,
	})

	// Write contiguous binary: [12B header][superstring][footer].
	f, err := os.Create(cfg.OutputPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(headerBytes); err != nil {
		return fmt.Errorf("writing header: %w", err)
	}
	if _, err := f.WriteString(masterString); err != nil {
		return fmt.Errorf("writing superstring: %w", err)
	}
	if _, err := f.Write(footer); err != nil {
		return fmt.Errorf("writing footer: %w", err)
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
	miniSuper, fragment, overlapLength := pipeline.TruncateAndAppend(superstring1, survivors, 3, 15)

	// Step 3: Build the combined superstring.
	combinedPayload := superstring1 + fragment

	// Step 4: Build the combined word list with correct offsets.
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

	// OPTIMIZATION: Map against the tiny miniSuper instead of the full combined payload.
	// This avoids building a massive suffix array for 1GB+ payloads just to find
	// offsets of a small handful of surviving words.
	if len(survivors) > 0 && miniSuper != "" {
		uniqueSurvivors := pipeline.ExactDeduplication(survivors)
		var nonEmpty []string
		for _, s := range uniqueSurvivors {
			if s != "" {
				nonEmpty = append(nonEmpty, s)
			}
		}

		// Map offsets within the small miniSuper, then shift forward.
		survivorOffsets := format.MapOffsets(miniSuper, nonEmpty)

		// Mathematically shift offsets to absolute position in combined payload.
		shift := len(superstring1) - overlapLength
		for _, surv := range survivors {
			if offset, ok := survivorOffsets[surv]; ok {
				allWords = append(allWords, format.Word{
					String: surv,
					Offset: offset + shift,
					Length: len(surv),
				})
			}
		}
	}

	// Step 5: Graceful Downgrade.
	isOrdered := header1.IsOrdered && header2.IsOrdered

	// Step 6: Build the merged footer.
	offsetMap := make(map[string]int)
	for _, w := range allWords {
		if w.String != "" {
			if _, exists := offsetMap[w.String]; !exists {
				offsetMap[w.String] = w.Offset
			}
		}
	}

	var footer []byte
	if !isOrdered {
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
		var chronoLines []string
		for _, w := range allWords {
			chronoLines = append(chronoLines, w.String)
		}
		footer = format.EncodeOrdered(chronoLines, offsetMap, len(combinedPayload))
	}

	// Step 7: Calculate footer offset and build header.
	footerOffset := uint64(12 + len(combinedPayload))
	mergedHeader := format.EncodeHeader(&format.Header{
		Version:      0x02,
		Separator:    header1.Separator,
		IsOrdered:    isOrdered,
		FooterOffset: footerOffset,
	})

	// Step 8: Write contiguous binary.
	f, err := os.Create(cfg.OutputPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(mergedHeader); err != nil {
		return fmt.Errorf("writing header: %w", err)
	}
	if _, err := f.WriteString(combinedPayload); err != nil {
		return fmt.Errorf("writing superstring: %w", err)
	}
	if _, err := f.Write(footer); err != nil {
		return fmt.Errorf("writing footer: %w", err)
	}

	if cfg.Verbose {
		fmt.Printf("Merged: %d + %d words → %d characters written to %s\n",
			len(words1), len(words2), len(combinedPayload), cfg.OutputPath)
	}

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
	// Read the raw file bytes.
	data, err := os.ReadFile(cfg.FilePath)
	if err != nil {
		return fmt.Errorf("reading %q: %w", cfg.FilePath, err)
	}

	if len(data) < 12 {
		return fmt.Errorf("invalid .scs file: missing header")
	}

	header, err := format.DecodeHeader(data[:12])
	if err != nil {
		return fmt.Errorf("corrupt file: %w", err)
	}

	if header.FooterOffset > uint64(len(data)) || header.FooterOffset < 12 {
		return fmt.Errorf("corrupt file: footer offset out of bounds")
	}

	// Instant O(1) bounds extraction.
	superstring := string(data[12:header.FooterOffset])

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
			return nil
		}
	}

	// False positive: exists as substring but not as a dataset member.
	fmt.Println("Not found")
	os.Exit(1)
	return nil
}
