package format

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Word holds extracted word data from the .scs format.
type Word struct {
	String string
	Offset int
	Length int
}

// DecodeSCS reads and parses an .scs file, returning the header,
// the superstring payload, and the reconstructed word list.
func DecodeSCS(filepath string) (*Header, string, []Word, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, "", nil, fmt.Errorf("reading %q: %w", filepath, err)
	}

	// Split into exactly 3 lines.
	content := string(data)

	// The file must end with \n, and have exactly 3 lines.
	// We split on \n and expect 4 parts (3 lines + empty trailing).
	parts := strings.SplitN(content, "\n", 4)
	if len(parts) < 3 {
		return nil, "", nil, fmt.Errorf("invalid .scs file: expected 3 lines, got %d", len(parts))
	}

	line1 := parts[0] // Base64 header
	line2 := parts[1] // Superstring payload
	line3 := parts[2] // Base64 footer

	// Validate Line 1 header.
	if len(line1) != 16 {
		return nil, "", nil, fmt.Errorf("invalid header: expected 16 Base64 characters, got %d", len(line1))
	}

	header, err := DecodeHeader(line1)
	if err != nil {
		return nil, "", nil, fmt.Errorf("decoding header: %w", err)
	}

	// Decode Line 3 footer.
	footerBytes, err := base64.StdEncoding.DecodeString(line3)
	if err != nil {
		return nil, "", nil, fmt.Errorf("decoding footer Base64: %w", err)
	}

	var words []Word

	if header.IsOrdered {
		words, err = decodeOrderedFooter(footerBytes, line2)
	} else {
		words, err = decodeUnorderedFooter(footerBytes, line2)
	}

	if err != nil {
		return nil, "", nil, err
	}

	return header, line2, words, nil
}

func decodeOrderedFooter(footerBytes []byte, superstring string) ([]Word, error) {
	r := bytes.NewReader(footerBytes)

	maxWordLen, _, err := DecodeULEB128(r)
	if err != nil {
		return nil, fmt.Errorf("reading max word length: %w", err)
	}

	totalLineCount, _, err := DecodeULEB128(r)
	if err != nil {
		return nil, fmt.Errorf("reading total line count: %w", err)
	}

	lengthBits := bitsNeeded(maxWordLen)
	offsetBits := bitsNeeded(uint64(len(superstring)))
	if offsetBits == 0 {
		offsetBits = 1
	}

	remaining := make([]byte, r.Len())
	_, _ = r.Read(remaining)
	br := NewBitReader(remaining)

	words := make([]Word, 0, totalLineCount)
	for i := uint64(0); i < totalLineCount; i++ {
		length := int(br.ReadBits(lengthBits))
		offset := int(br.ReadBits(offsetBits))

		if length == 0 {
			words = append(words, Word{String: "", Offset: 0, Length: 0})
			continue
		}

		// CRITICAL SECURITY CHECK: bounds validation.
		if offset+length > len(superstring) {
			return nil, fmt.Errorf("corrupt metadata: offset %d + length %d > superstring length %d",
				offset, length, len(superstring))
		}

		words = append(words, Word{
			String: superstring[offset : offset+length],
			Offset: offset,
			Length: length,
		})
	}

	return words, nil
}

func decodeUnorderedFooter(footerBytes []byte, superstring string) ([]Word, error) {
	r := bytes.NewReader(footerBytes)

	var words []Word

	for r.Len() > 0 {
		wordLen, _, err := DecodeULEB128(r)
		if err != nil {
			if errors.Is(err, errors.New("")) {
				break
			}
			return nil, fmt.Errorf("reading word length: %w", err)
		}

		count, _, err := DecodeULEB128(r)
		if err != nil {
			return nil, fmt.Errorf("reading word count: %w", err)
		}

		absOffset := 0
		for j := uint64(0); j < count; j++ {
			delta, _, err := DecodeULEB128(r)
			if err != nil {
				return nil, fmt.Errorf("reading delta offset: %w", err)
			}
			absOffset += int(delta)

			length := int(wordLen)

			// CRITICAL SECURITY CHECK: bounds validation.
			if absOffset+length > len(superstring) {
				return nil, fmt.Errorf("corrupt metadata: offset %d + length %d > superstring length %d",
					absOffset, length, len(superstring))
			}

			words = append(words, Word{
				String: superstring[absOffset : absOffset+length],
				Offset: absOffset,
				Length: length,
			})
		}
	}

	// Sort by offset for consistent output in unordered mode.
	sort.Slice(words, func(i, j int) bool {
		return words[i].Offset < words[j].Offset
	})

	return words, nil
}
