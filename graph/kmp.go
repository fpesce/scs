package graph

// CompileLPS constructs the KMP failure function (Longest Proper Prefix-Suffix array).
func CompileLPS(pattern string) []int {
	m := len(pattern)
	f := make([]int, m)

	if m == 0 {
		return f
	}

	f[0] = 0
	j := 0

	for i := 1; i < m; i++ {
		for j > 0 && pattern[i] != pattern[j] {
			j = f[j-1]
		}
		if pattern[i] == pattern[j] {
			j++
		}
		f[i] = j
	}

	return f
}

// CalculateMaxOverlap finds the maximum length where a proper suffix of left
// exactly matches a proper prefix of right using zero-allocation slice comparison.
func CalculateMaxOverlap(left, right string) int {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}

	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}

	// Try from longest possible overlap down to 1.
	// Fast boundary check: first and last chars must match before full slice comparison.
	for k := limit; k > 0; k-- {
		if left[len(left)-k] == right[0] && left[len(left)-1] == right[k-1] {
			if left[len(left)-k:] == right[:k] {
				return k
			}
		}
	}

	return 0
}
