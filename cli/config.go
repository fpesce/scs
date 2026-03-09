// Package cli provides command-line flag parsing for the SCS tool.
package cli

import (
	"errors"
	"flag"
	"fmt"
)

// BuildConfig holds configuration for the build subcommand.
type BuildConfig struct {
	InputPath  string
	OutputPath string
	MinOverlap int
	DPLimit    int
	Unordered  bool
	Separator  string
	Verbose    bool
	Profile    bool
}

// MergeConfig holds configuration for the merge subcommand.
type MergeConfig struct {
	PrimaryPath string
	UpdatePath  string
	OutputPath  string
	Verbose     bool
}

// CatConfig holds configuration for the cat subcommand.
type CatConfig struct {
	FilePath string
}

// SearchConfig holds configuration for the search subcommand.
type SearchConfig struct {
	Word     string
	FilePath string
}

// ParseSubcommand parses CLI args and returns the subcommand name and its typed config.
func ParseSubcommand(args []string) (string, interface{}, error) {
	if len(args) < 1 {
		return "", nil, errors.New("usage: scs <command> [options]\ncommands: build, cat, search, merge")
	}

	cmd := args[0]
	subArgs := args[1:]

	switch cmd {
	case "build":
		cfg, err := parseBuild(subArgs)
		return cmd, cfg, err
	case "merge":
		cfg, err := parseMerge(subArgs)
		return cmd, cfg, err
	case "cat":
		cfg, err := parseCat(subArgs)
		return cmd, cfg, err
	case "search":
		cfg, err := parseSearch(subArgs)
		return cmd, cfg, err
	default:
		return "", nil, fmt.Errorf("unknown command %q; available: build, cat, search, merge", cmd)
	}
}

func parseBuild(args []string) (*BuildConfig, error) {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	cfg := &BuildConfig{}

	fs.StringVar(&cfg.InputPath, "i", "", "Path to the input file")
	fs.StringVar(&cfg.InputPath, "input", "", "Path to the input file")
	fs.StringVar(&cfg.OutputPath, "o", "", "Path to the output file")
	fs.StringVar(&cfg.OutputPath, "output", "", "Path to the output file")
	fs.IntVar(&cfg.MinOverlap, "k", 3, "Minimum meaningful overlap threshold")
	fs.IntVar(&cfg.MinOverlap, "min-overlap", 3, "Minimum meaningful overlap threshold")
	fs.IntVar(&cfg.DPLimit, "dp-limit", 15, "Exact DP threshold for assembly")
	fs.BoolVar(&cfg.Unordered, "unordered", false, "Build in UNORDERED (dictionary) mode")
	fs.StringVar(&cfg.Separator, "sep", "\n", "Separator character for cat output")
	fs.BoolVar(&cfg.Verbose, "v", false, "Enable verbose progress updates")
	fs.BoolVar(&cfg.Verbose, "verbose", false, "Enable verbose progress updates")
	fs.BoolVar(&cfg.Profile, "profile", false, "Write CPU profile to cpu.prof")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("parsing build flags: %w", err)
	}

	if cfg.InputPath == "" {
		return nil, errors.New("build: input path (-i) is required")
	}
	if cfg.OutputPath == "" {
		return nil, errors.New("build: output path (-o) is required")
	}

	return cfg, nil
}

func parseMerge(args []string) (*MergeConfig, error) {
	fs := flag.NewFlagSet("merge", flag.ContinueOnError)
	cfg := &MergeConfig{}

	fs.StringVar(&cfg.PrimaryPath, "primary", "", "Path to the primary .scs file")
	fs.StringVar(&cfg.UpdatePath, "update", "", "Path to the update .scs file")
	fs.StringVar(&cfg.OutputPath, "o", "", "Path to the merged output .scs file")
	fs.StringVar(&cfg.OutputPath, "output", "", "Path to the merged output .scs file")
	fs.BoolVar(&cfg.Verbose, "v", false, "Enable verbose progress updates")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("parsing merge flags: %w", err)
	}

	if cfg.PrimaryPath == "" {
		return nil, errors.New("merge: --primary is required")
	}
	if cfg.UpdatePath == "" {
		return nil, errors.New("merge: --update is required")
	}
	if cfg.OutputPath == "" {
		return nil, errors.New("merge: -o is required")
	}

	return cfg, nil
}

func parseCat(args []string) (*CatConfig, error) {
	fs := flag.NewFlagSet("cat", flag.ContinueOnError)

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("parsing cat flags: %w", err)
	}

	remaining := fs.Args()
	if len(remaining) < 1 {
		return nil, errors.New("cat: <file.scs> argument is required")
	}

	return &CatConfig{FilePath: remaining[0]}, nil
}

func parseSearch(args []string) (*SearchConfig, error) {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("parsing search flags: %w", err)
	}

	remaining := fs.Args()
	if len(remaining) < 2 {
		return nil, errors.New("search: usage: scs search \"<word>\" <file.scs>")
	}

	return &SearchConfig{Word: remaining[0], FilePath: remaining[1]}, nil
}
