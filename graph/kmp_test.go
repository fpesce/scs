package graph

import (
	"testing"
)

func TestCompileLPS(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    []int
	}{
		{
			name:    "ABAB pattern",
			pattern: "ABAB",
			want:    []int{0, 0, 1, 2},
		},
		{
			name:    "AAAA pattern",
			pattern: "AAAA",
			want:    []int{0, 1, 2, 3},
		},
		{
			name:    "ABCABD pattern",
			pattern: "ABCABD",
			want:    []int{0, 0, 0, 1, 2, 0},
		},
		{
			name:    "empty pattern",
			pattern: "",
			want:    []int{},
		},
		{
			name:    "single char",
			pattern: "A",
			want:    []int{0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompileLPS(tt.pattern)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d: %v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("LPS[%d] = %d, want %d", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestCalculateMaxOverlap(t *testing.T) {
	tests := []struct {
		name  string
		left  string
		right string
		want  int
	}{
		{
			name:  "classic overlap",
			left:  "aba",
			right: "bab",
			want:  2,
		},
		{
			name:  "full suffix-prefix match",
			left:  "abcabc",
			right: "abcxyz",
			want:  3,
		},
		{
			name:  "no overlap",
			left:  "abc",
			right: "xyz",
			want:  0,
		},
		{
			name:  "single char overlap",
			left:  "abc",
			right: "cde",
			want:  1,
		},
		{
			name:  "identical strings",
			left:  "abc",
			right: "abc",
			want:  3,
		},
		{
			name:  "empty left",
			left:  "",
			right: "abc",
			want:  0,
		},
		{
			name:  "empty right",
			left:  "abc",
			right: "",
			want:  0,
		},
		{
			name:  "left longer than right",
			left:  "xyzab",
			right: "ab",
			want:  2,
		},
		{
			name:  "right longer than left",
			left:  "ab",
			right: "abcdef",
			want:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateMaxOverlap(tt.left, tt.right)
			if got != tt.want {
				t.Errorf("CalculateMaxOverlap(%q, %q) = %d, want %d", tt.left, tt.right, got, tt.want)
			}
		})
	}
}
