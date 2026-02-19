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
// exactly matches a proper prefix of right. Uses KMP logic on the concatenated
// pattern right+"$"+left to avoid O(L^2) naive checks.
func CalculateMaxOverlap(left, right string) int {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}

	// Build pattern = right + sentinel + left.
	// The LPS of this combined string tells us the longest prefix of right
	// that matches a suffix of left.
	combined := right + "$" + left
	lps := CompileLPS(combined)

	// The last entry of the LPS array is the overlap length,
	// bounded by min(len(left), len(right)).
	return lps[len(combined)-1]
}
