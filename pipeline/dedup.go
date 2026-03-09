// Package pipeline implements the multi-phase SCS processing pipeline.
package pipeline

// ExactDeduplication removes exact duplicate strings in O(N) time,
// preserving the order of first occurrence.
func ExactDeduplication(lines []string) []string {
	seen := make(map[string]struct{}, len(lines))
	result := make([]string, 0, len(lines))

	for _, line := range lines {
		if _, exists := seen[line]; exists {
			continue
		}
		seen[line] = struct{}{}
		result = append(result, line)
	}

	return result
}
