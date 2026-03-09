package pipeline

import (
	"testing"
)

func TestExactDeduplication(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "no duplicates",
			input: []string{"alpha", "beta", "gamma"},
			want:  []string{"alpha", "beta", "gamma"},
		},
		{
			name:  "all duplicates",
			input: []string{"dup", "dup", "dup"},
			want:  []string{"dup"},
		},
		{
			name:  "mixed duplicates preserving order",
			input: []string{"a", "b", "a", "c", "b", "d"},
			want:  []string{"a", "b", "c", "d"},
		},
		{
			name:  "empty input",
			input: []string{},
			want:  []string{},
		},
		{
			name:  "single element",
			input: []string{"only"},
			want:  []string{"only"},
		},
		{
			name:  "case sensitive",
			input: []string{"Hello", "hello", "HELLO"},
			want:  []string{"Hello", "hello", "HELLO"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExactDeduplication(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d items, want %d: %v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("index %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
