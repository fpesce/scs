package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/joke/scs/cli"
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
	cfg, err := cli.ParseFlags(args)
	if err != nil {
		return fmt.Errorf("configuration: %w", err)
	}

	lines, err := scsio.ReadLines(cfg.InputPath)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}
	if cfg.Verbose {
		fmt.Printf("Read %d lines from %s\n", len(lines), cfg.InputPath)
	}

	// Phase 1A: Hash deduplication.
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
	// Real-time progress is reported inside AssembleConcurrently when verbose.
	superWords := pipeline.AssembleConcurrently(islands, cfg.DPLimit, cfg.MinOverlap, cfg.Verbose)
	if cfg.Verbose {
		fmt.Printf("Phase 4: assembled %d super-words\n", len(superWords))
	}

	// Phase 5: Sort by length descending, single-alloc concatenation.
	sort.Slice(superWords, func(i, j int) bool {
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
	if err := scsio.WriteResult(cfg.OutputPath, masterString); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	if cfg.Verbose {
		fmt.Printf("Final superstring: %d characters written to %s\n",
			len(masterString), cfg.OutputPath)
	}

	return nil
}
