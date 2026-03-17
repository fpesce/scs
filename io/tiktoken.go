package io

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ReadTiktoken reads a .tiktoken file (base64 + rank format) and returns:
//   - words: decoded raw byte strings (as Go strings)
//   - rankMap: mapping from each decoded string to its uint32 rank
func ReadTiktoken(filepath string) ([]string, map[string]uint32, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, nil, fmt.Errorf("reading %q: %w", filepath, err)
	}

	lines := strings.Split(string(data), "\n")

	words := make([]string, 0, len(lines))
	rankMap := make(map[string]uint32, len(lines))

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			return nil, nil, fmt.Errorf("line %d: expected 'base64 rank', got %q", i+1, line)
		}

		decoded, err := base64.StdEncoding.DecodeString(parts[0])
		if err != nil {
			return nil, nil, fmt.Errorf("line %d: base64 decode: %w", i+1, err)
		}

		rank, err := strconv.ParseUint(parts[1], 10, 32)
		if err != nil {
			return nil, nil, fmt.Errorf("line %d: parsing rank %q: %w", i+1, parts[1], err)
		}

		word := string(decoded)
		words = append(words, word)
		rankMap[word] = uint32(rank)
	}

	return words, rankMap, nil
}
