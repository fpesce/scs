package format

import (
	"bytes"
	"encoding/base64"
	"sort"
)

// EncodeUnordered generates the UNORDERED mode metadata footer.
// Words are grouped by ascending length, offsets within each group are
// delta-encoded after sorting them ascending. Returns Base64 encoded result.
func EncodeUnordered(uniqueWords []string, offsetMap map[string]int) string {
	// Group words by length.
	lengthGroups := make(map[int][]string)
	for _, w := range uniqueWords {
		lengthGroups[len(w)] = append(lengthGroups[len(w)], w)
	}

	// Sort group keys ascending.
	lengths := make([]int, 0, len(lengthGroups))
	for l := range lengthGroups {
		lengths = append(lengths, l)
	}
	sort.Ints(lengths)

	var buf bytes.Buffer

	for _, wordLen := range lengths {
		words := lengthGroups[wordLen]

		// Extract and sort absolute offsets for this group.
		offsets := make([]int, 0, len(words))
		for _, w := range words {
			offsets = append(offsets, offsetMap[w])
		}
		sort.Ints(offsets)

		// Write ULEB128(Word_Length) and ULEB128(Total_Words_In_Group).
		buf.Write(EncodeULEB128(uint64(wordLen)))
		buf.Write(EncodeULEB128(uint64(len(words))))

		// Write delta-encoded offsets.
		prev := 0
		for _, offset := range offsets {
			delta := offset - prev
			buf.Write(EncodeULEB128(uint64(delta)))
			prev = offset
		}
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes())
}
