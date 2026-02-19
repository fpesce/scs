// Package cli provides command-line flag parsing for the SCS tool.
package cli

import (
	"errors"
	"flag"
	"fmt"
)

// Config holds all application configuration derived from CLI flags.
type Config struct {
	InputPath  string
	OutputPath string
	MinOverlap int
	DPLimit    int
	Verbose    bool
}

// ParseFlags parses and validates CLI flags from the given arguments.
func ParseFlags(args []string) (*Config, error) {
	fs := flag.NewFlagSet("scs", flag.ContinueOnError)

	cfg := &Config{}

	fs.StringVar(&cfg.InputPath, "i", "", "Path to the input file")
	fs.StringVar(&cfg.InputPath, "input", "", "Path to the input file")
	fs.StringVar(&cfg.OutputPath, "o", "", "Path to the output file")
	fs.StringVar(&cfg.OutputPath, "output", "", "Path to the output file")
	fs.IntVar(&cfg.MinOverlap, "k", 3, "Minimum meaningful overlap threshold")
	fs.IntVar(&cfg.MinOverlap, "min-overlap", 3, "Minimum meaningful overlap threshold")
	fs.IntVar(&cfg.DPLimit, "dp-limit", 15, "Exact DP threshold for assembly")
	fs.BoolVar(&cfg.Verbose, "v", false, "Enable verbose progress updates")
	fs.BoolVar(&cfg.Verbose, "verbose", false, "Enable verbose progress updates")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("parsing flags: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate checks that all required fields are set.
func (c *Config) validate() error {
	if c.InputPath == "" {
		return errors.New("input path (-i/--input) is required")
	}
	if c.OutputPath == "" {
		return errors.New("output path (-o/--output) is required")
	}
	return nil
}
