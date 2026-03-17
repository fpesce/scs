package cli

import (
	"testing"
	"time"
)

//nolint:gocognit // table-driven test with field-by-field assertions
func TestParseSubcommand_Build(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantErr    bool
		wantConfig *BuildConfig
	}{
		{
			name: "all flags short form",
			args: []string{"build", "-i", "in.txt", "-o", "out.txt", "-k", "5", "-v"},
			wantConfig: &BuildConfig{
				InputPath:  "in.txt",
				OutputPath: "out.txt",
				MinOverlap: 5,
				DPLimit:    15,
				Separator:  "\n",
				Verbose:    true,
			},
		},
		{
			name: "unordered mode",
			args: []string{"build", "-i", "in.txt", "-o", "out.scs", "--unordered"},
			wantConfig: &BuildConfig{
				InputPath:  "in.txt",
				OutputPath: "out.scs",
				MinOverlap: 3,
				DPLimit:    15,
				Separator:  "\n",
				Unordered:  true,
			},
		},
		{
			name: "custom separator",
			args: []string{"build", "-i", "in.txt", "-o", "out.scs", "--sep", ","},
			wantConfig: &BuildConfig{
				InputPath:  "in.txt",
				OutputPath: "out.scs",
				MinOverlap: 3,
				DPLimit:    15,
				Separator:  ",",
			},
		},
		{
			name: "defaults applied",
			args: []string{"build", "-i", "in.txt", "-o", "out.txt"},
			wantConfig: &BuildConfig{
				InputPath:  "in.txt",
				OutputPath: "out.txt",
				MinOverlap: 3,
				DPLimit:    15,
				Separator:  "\n",
			},
		},
		{
			name:    "missing input",
			args:    []string{"build", "-o", "out.txt"},
			wantErr: true,
		},
		{
			name:    "missing output",
			args:    []string{"build", "-i", "in.txt"},
			wantErr: true,
		},
		{
			name: "ga flags parsed",
			args: []string{"build", "-i", "in.txt", "-o", "out.scs", "--ga-time", "30s", "--ga-pop", "200", "--ga-tourney", "5", "--ga-stag", "100"},
			wantConfig: &BuildConfig{
				InputPath:  "in.txt",
				OutputPath: "out.scs",
				MinOverlap: 3,
				DPLimit:    15,
				Separator:  "\n",
				GATime:     30 * time.Second,
				GAPop:      200,
				GATourney:  5,
				GAStag:     100,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, cfg, err := ParseSubcommand(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cmd != "build" {
				t.Fatalf("cmd = %q, want %q", cmd, "build")
			}
			bc, ok := cfg.(*BuildConfig)
			if !ok {
				t.Fatal("cfg is not *BuildConfig")
			}
			if bc.InputPath != tt.wantConfig.InputPath {
				t.Errorf("InputPath = %q, want %q", bc.InputPath, tt.wantConfig.InputPath)
			}
			if bc.OutputPath != tt.wantConfig.OutputPath {
				t.Errorf("OutputPath = %q, want %q", bc.OutputPath, tt.wantConfig.OutputPath)
			}
			if bc.MinOverlap != tt.wantConfig.MinOverlap {
				t.Errorf("MinOverlap = %d, want %d", bc.MinOverlap, tt.wantConfig.MinOverlap)
			}
			if bc.DPLimit != tt.wantConfig.DPLimit {
				t.Errorf("DPLimit = %d, want %d", bc.DPLimit, tt.wantConfig.DPLimit)
			}
			if bc.Unordered != tt.wantConfig.Unordered {
				t.Errorf("Unordered = %v, want %v", bc.Unordered, tt.wantConfig.Unordered)
			}
			if bc.Separator != tt.wantConfig.Separator {
				t.Errorf("Separator = %q, want %q", bc.Separator, tt.wantConfig.Separator)
			}
			if bc.Verbose != tt.wantConfig.Verbose {
				t.Errorf("Verbose = %v, want %v", bc.Verbose, tt.wantConfig.Verbose)
			}
			if bc.GATime != tt.wantConfig.GATime {
				t.Errorf("GATime = %v, want %v", bc.GATime, tt.wantConfig.GATime)
			}
			if bc.GAPop != tt.wantConfig.GAPop {
				t.Errorf("GAPop = %d, want %d", bc.GAPop, tt.wantConfig.GAPop)
			}
			if bc.GATourney != tt.wantConfig.GATourney {
				t.Errorf("GATourney = %d, want %d", bc.GATourney, tt.wantConfig.GATourney)
			}
			if bc.GAStag != tt.wantConfig.GAStag {
				t.Errorf("GAStag = %d, want %d", bc.GAStag, tt.wantConfig.GAStag)
			}
		})
	}
}

func TestParseSubcommand_Cat(t *testing.T) {
	cmd, cfg, err := ParseSubcommand([]string{"cat", "test.scs"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd != "cat" {
		t.Fatalf("cmd = %q, want %q", cmd, "cat")
	}
	cc, ok := cfg.(*CatConfig)
	if !ok {
		t.Fatal("cfg is not *CatConfig")
	}
	if cc.FilePath != "test.scs" {
		t.Errorf("FilePath = %q, want %q", cc.FilePath, "test.scs")
	}
}

func TestParseSubcommand_Search(t *testing.T) {
	cmd, cfg, err := ParseSubcommand([]string{"search", "hello", "test.scs"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd != "search" {
		t.Fatalf("cmd = %q, want %q", cmd, "search")
	}
	sc, ok := cfg.(*SearchConfig)
	if !ok {
		t.Fatal("cfg is not *SearchConfig")
	}
	if sc.Word != "hello" {
		t.Errorf("Word = %q, want %q", sc.Word, "hello")
	}
	if sc.FilePath != "test.scs" {
		t.Errorf("FilePath = %q, want %q", sc.FilePath, "test.scs")
	}
}

func TestParseSubcommand_Merge(t *testing.T) {
	cmd, cfg, err := ParseSubcommand([]string{"merge", "--primary", "a.scs", "--update", "b.scs", "-o", "c.scs"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd != "merge" {
		t.Fatalf("cmd = %q, want %q", cmd, "merge")
	}
	mc, ok := cfg.(*MergeConfig)
	if !ok {
		t.Fatal("cfg is not *MergeConfig")
	}
	if mc.PrimaryPath != "a.scs" {
		t.Errorf("PrimaryPath = %q, want %q", mc.PrimaryPath, "a.scs")
	}
	if mc.UpdatePath != "b.scs" {
		t.Errorf("UpdatePath = %q, want %q", mc.UpdatePath, "b.scs")
	}
	if mc.OutputPath != "c.scs" {
		t.Errorf("OutputPath = %q, want %q", mc.OutputPath, "c.scs")
	}
}

func TestParseSubcommand_Errors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no args", []string{}},
		{"unknown command", []string{"foo"}},
		{"cat missing file", []string{"cat"}},
		{"search missing args", []string{"search", "word"}},
		{"merge missing primary", []string{"merge", "--update", "b.scs", "-o", "c.scs"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseSubcommand(tt.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}
