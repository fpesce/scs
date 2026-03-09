package pipeline

import (
	"testing"
)

func TestEliminateSubstrings(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "no substrings",
			input: []string{"abc", "xyz", "def"},
			want:  []string{"abc", "xyz", "def"},
		},
		{
			name:  "simple substring removal",
			input: []string{"cart", "art", "car"},
			want:  []string{"cart"},
		},
		{
			name:  "nested substring chain",
			input: []string{"abcdef", "bcd", "cd"},
			want:  []string{"abcdef"},
		},
		{
			name:  "equal strings deduplicated",
			input: []string{"abc", "abc"},
			want:  []string{"abc"},
		},
		{
			name:  "single element",
			input: []string{"alone"},
			want:  []string{"alone"},
		},
		{
			name:  "empty input",
			input: []string{},
			want:  []string{},
		},
		{
			name:  "no containment but overlapping",
			input: []string{"abc", "bcd", "cde"},
			want:  []string{"abc", "bcd", "cde"},
		},
		{
			name:  "prefix is substring",
			input: []string{"hello", "helloworld", "world"},
			want:  []string{"helloworld"},
		},
		{
			name:  "suffix is substring",
			input: []string{"world", "helloworld", "hello"},
			want:  []string{"helloworld"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EliminateSubstrings(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d items %v, want %d items %v", len(got), got, len(tt.want), tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("index %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
