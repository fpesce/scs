package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/joke/scs/format"
	scsio "github.com/joke/scs/io"
)

// TestE2ETiktoken is the end-to-end pipeline test for tiktoken mode.
// It runs runBuild on the synthetic.tiktoken + synthetic.json testdata,
// decodes the resulting .scs, reads the .rank sidecar, and asserts
// that word-rank alignment is perfect against the original mapping.
func TestE2ETiktoken(t *testing.T) {
	dir := t.TempDir()
	outSCS := filepath.Join(dir, "synthetic.scs")

	// Run the pipeline in tiktoken mode.
	args := []string{
		"build",
		"--tiktoken",
		"-i", filepath.Join("testdata", "synthetic.tiktoken"),
		"-o", outSCS,
		"--metadata", filepath.Join("testdata", "synthetic.json"),
	}
	if err := run(args); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	// --- Verify .scs ---
	_, _, decodedWords, err := format.DecodeSCS(outSCS)
	if err != nil {
		t.Fatalf("DecodeSCS: %v", err)
	}

	// --- Verify .rank ---
	rankPath := filepath.Join(dir, "synthetic.rank")
	rankData, err := os.ReadFile(rankPath)
	if err != nil {
		t.Fatalf("reading rank sidecar: %v", err)
	}

	// The .rank file should have exactly len(decodedWords) * 4 bytes.
	if len(rankData) != len(decodedWords)*4 {
		t.Fatalf("rank file length = %d, want %d (words=%d)",
			len(rankData), len(decodedWords)*4, len(decodedWords))
	}

	// Read original tiktoken mapping.
	origWords, origRanks, err := scsio.ReadTiktoken(filepath.Join("testdata", "synthetic.tiktoken"))
	if err != nil {
		t.Fatalf("ReadTiktoken: %v", err)
	}

	// Build word→rank from the .rank sidecar.
	// The .rank entries are in footer encoding order (length-grouped,
	// offset-sorted within each group), which is the same order as
	// orderedWords from EncodeUnordered. DecodeSCS also returns words
	// in the same footer-decoded order (before the final sort by offset),
	// but decodeUnorderedFooter re-sorts by offset. So instead of zipping
	// by index, we map by word string.
	scsRankMap := make(map[string]uint32, len(decodedWords))
	for i, w := range decodedWords {
		rank := binary.LittleEndian.Uint32(rankData[i*4 : (i+1)*4])
		scsRankMap[w.String] = rank
	}

	// Wait — the above is wrong because decodedWords is sorted by offset,
	// but rankData[i] corresponds to the i-th footer entry (length-grouped order).
	// We need to decode the footer ourselves to get the correct word order.
	//
	// Alternatively, we can build a (word→rank) map from the EncodeUnordered
	// output directly. But the simplest correct approach is to rebuild the
	// orderedWords by re-running EncodeUnordered and matching.
	//
	// The REAL CONTRACT: each entry in .rank corresponds to orderedWords[i],
	// which is the i-th word as written to the footer by EncodeUnordered.
	// These words are ordered: all length-1 words (by offset), then all
	// length-2 words (by offset), etc. The decoder returns them in the same
	// order initially, then sorts by offset. Since we can't easily get the
	// pre-sort order from DecodeSCS, let's instead verify the mapping
	// by word string: for every original word, verify it appears in the .scs
	// and that its rank can be found associated with it.

	// Build the test differently: reconstruct the orderedWords→rank pairing
	// from the raw .scs file footer directly, matching length groups.
	data, err := os.ReadFile(outSCS)
	if err != nil {
		t.Fatalf("reading .scs: %v", err)
	}
	header, err := format.DecodeHeader(data[:12])
	if err != nil {
		t.Fatalf("decoding header: %v", err)
	}
	superstring := string(data[12:header.FooterOffset])

	// Manually decode the unordered footer to get words in footer order.
	footerWords := decodeFooterOrder(t, data[header.FooterOffset:], superstring)

	if len(footerWords) != len(rankData)/4 {
		t.Fatalf("footer words count %d != rank count %d",
			len(footerWords), len(rankData)/4)
	}

	// Now zip footerWords with rankData: footerWords[i]↔rankData[i*4..(i+1)*4]
	for i, word := range footerWords {
		gotRank := binary.LittleEndian.Uint32(rankData[i*4 : (i+1)*4])
		wantRank, ok := origRanks[word]
		if !ok {
			t.Errorf("footer word %d (%q): not in original tiktoken mapping", i, word)
			continue
		}
		if gotRank != wantRank {
			t.Errorf("footer word %d (%q): rank=%d, want=%d", i, word, gotRank, wantRank)
		}
	}

	// Verify all original words are present in the decoded set.
	decodedSet := make(map[string]bool, len(decodedWords))
	for _, w := range decodedWords {
		decodedSet[w.String] = true
	}
	for _, w := range origWords {
		if !decodedSet[w] {
			t.Errorf("original word %q missing from decoded .scs", w)
		}
	}

	// --- Verify .json ---
	jsonPath := filepath.Join(dir, "synthetic.json")
	gotJSON, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("reading json sidecar: %v", err)
	}
	wantJSON, err := os.ReadFile(filepath.Join("testdata", "synthetic.json"))
	if err != nil {
		t.Fatalf("reading original json: %v", err)
	}
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("json sidecar mismatch")
	}

	t.Logf("E2E tiktoken: %d words, %d rank entries, all aligned",
		len(decodedWords), len(rankData)/4)
}

// decodeFooterOrder parses an unordered footer and returns words
// in the exact order they were delta-encoded (length-grouped, offset-sorted).
func decodeFooterOrder(t *testing.T, footerBytes []byte, superstring string) []string {
	t.Helper()
	var words []string
	r := &footerReader{data: footerBytes, pos: 0}

	for r.pos < len(r.data) {
		wordLen := r.readULEB128(t)
		count := r.readULEB128(t)

		absOffset := uint64(0)
		for j := uint64(0); j < count; j++ {
			delta := r.readULEB128(t)
			absOffset += delta

			end := absOffset + wordLen
			if end > uint64(len(superstring)) {
				t.Fatalf("offset %d + len %d > superstring len %d",
					absOffset, wordLen, len(superstring))
			}
			words = append(words, superstring[absOffset:end])
		}
	}
	return words
}

type footerReader struct {
	data []byte
	pos  int
}

func (r *footerReader) readULEB128(t *testing.T) uint64 {
	t.Helper()
	var result uint64
	var shift uint
	for {
		if r.pos >= len(r.data) {
			t.Fatal("ULEB128 read past end of footer")
		}
		b := r.data[r.pos]
		r.pos++
		result |= uint64(b&0x7F) << shift
		if b&0x80 == 0 {
			break
		}
		shift += 7
	}
	return result
}
