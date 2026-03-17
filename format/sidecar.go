package format

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

// WriteRankSidecar writes the .rank binary sidecar file.
// It iterates over orderedWords (in footer delta-encoded order),
// looks up each word's rank from rankMap, and packs them as
// sequential Little-Endian uint32 values.
func WriteRankSidecar(filepath string, orderedWords []string, rankMap map[string]uint32) error {
	var buf bytes.Buffer
	buf.Grow(len(orderedWords) * 4)

	for i, word := range orderedWords {
		rank, ok := rankMap[word]
		if !ok {
			return fmt.Errorf("rank missing for word at index %d (%q)", i, word)
		}
		if err := binary.Write(&buf, binary.LittleEndian, rank); err != nil {
			return fmt.Errorf("writing rank %d: %w", i, err)
		}
	}

	if err := os.WriteFile(filepath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing rank sidecar %q: %w", filepath, err)
	}
	return nil
}

// WriteMetadataSidecar copies a JSON metadata file from inPath to outPath.
func WriteMetadataSidecar(outPath string, inPath string) error {
	data, err := os.ReadFile(inPath)
	if err != nil {
		return fmt.Errorf("reading metadata %q: %w", inPath, err)
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("writing metadata %q: %w", outPath, err)
	}
	return nil
}
