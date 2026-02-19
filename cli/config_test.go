package cli

import (
	"testing"
)

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantErr    bool
		wantConfig *Config
	}{
		{
			name: "all flags short form",
			args: []string{"-i", "in.txt", "-o", "out.txt", "-k", "5", "-v"},
			wantConfig: &Config{
				InputPath:  "in.txt",
				OutputPath: "out.txt",
				MinOverlap: 5,
				DPLimit:    15,
				Verbose:    true,
			},
		},
		{
			name: "all flags long form",
			args: []string{"--input", "in.txt", "--output", "out.txt", "--min-overlap", "4", "--dp-limit", "20", "--verbose"},
			wantConfig: &Config{
				InputPath:  "in.txt",
				OutputPath: "out.txt",
				MinOverlap: 4,
				DPLimit:    20,
				Verbose:    true,
			},
		},
		{
			name: "defaults applied",
			args: []string{"-i", "in.txt", "-o", "out.txt"},
			wantConfig: &Config{
				InputPath:  "in.txt",
				OutputPath: "out.txt",
				MinOverlap: 3,
				DPLimit:    15,
				Verbose:    false,
			},
		},
		{
			name:    "missing input",
			args:    []string{"-o", "out.txt"},
			wantErr: true,
		},
		{
			name:    "missing output",
			args:    []string{"-i", "in.txt"},
			wantErr: true,
		},
		{
			name:    "no flags at all",
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseFlags(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.InputPath != tt.wantConfig.InputPath {
				t.Errorf("InputPath = %q, want %q", cfg.InputPath, tt.wantConfig.InputPath)
			}
			if cfg.OutputPath != tt.wantConfig.OutputPath {
				t.Errorf("OutputPath = %q, want %q", cfg.OutputPath, tt.wantConfig.OutputPath)
			}
			if cfg.MinOverlap != tt.wantConfig.MinOverlap {
				t.Errorf("MinOverlap = %d, want %d", cfg.MinOverlap, tt.wantConfig.MinOverlap)
			}
			if cfg.DPLimit != tt.wantConfig.DPLimit {
				t.Errorf("DPLimit = %d, want %d", cfg.DPLimit, tt.wantConfig.DPLimit)
			}
			if cfg.Verbose != tt.wantConfig.Verbose {
				t.Errorf("Verbose = %v, want %v", cfg.Verbose, tt.wantConfig.Verbose)
			}
		})
	}
}
