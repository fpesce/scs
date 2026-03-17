package format

import (
	"bytes"
	"sort"
)

type wordOffset struct {
	word   string
	offset int
}

// EncodeUnordered generates the UNORDERED mode metadata footer as raw bytes.
// Words are grouped by ascending length, offsets within each group are
// delta-encoded after sorting them ascending.
// Returns the footer bytes and the exact sequence of words in offset order.
func EncodeUnordered(uniqueWords []string, offsetMap map[string]int) ([]byte, []string) {
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
	var orderedWords []string

	for _, wordLen := range lengths {
		words := lengthGroups[wordLen]

		// Build wordOffset pairs for stable sorting by offset.
		pairs := make([]wordOffset, len(words))
		for i, w := range words {
			pairs[i] = wordOffset{word: w, offset: offsetMap[w]}
		}
		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].offset < pairs[j].offset
		})

		// Write ULEB128(Word_Length) and ULEB128(Total_Words_In_Group).
		buf.Write(EncodeULEB128(uint64(wordLen)))
		buf.Write(EncodeULEB128(uint64(len(pairs))))

		// Write delta-encoded offsets and track word sequence.
		prev := 0
		for _, p := range pairs {
			delta := p.offset - prev
			buf.Write(EncodeULEB128(uint64(delta)))
			prev = p.offset
			orderedWords = append(orderedWords, p.word)
		}
	}

	return buf.Bytes(), orderedWords
}
