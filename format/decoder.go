package format

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"sort"
)

// Word holds extracted word data from the .scs format.
type Word struct {
	String string
	Offset int
	Length  int
}

// DecodeSCS reads and parses an .scs file, returning the header,
// the superstring payload, and the reconstructed word list.
// Uses O(1) byte slicing via the header's FooterOffset.
func DecodeSCS(filepath string) (*Header, string, []Word, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, "", nil, fmt.Errorf("reading %q: %w", filepath, err)
	}

	if len(data) < 12 {
		return nil, "", nil, errors.New("invalid .scs file: missing header")
	}

	header, err := DecodeHeader(data[:12])
	if err != nil {
		return nil, "", nil, fmt.Errorf("decoding header: %w", err)
	}

	if header.FooterOffset > uint64(len(data)) || header.FooterOffset < 12 {
		return nil, "", nil, fmt.Errorf("corrupt metadata: footer offset %d out of bounds (file size %d)",
			header.FooterOffset, len(data))
	}

	// Instant O(1) slicing. No \n scanning required.
	superstring := string(data[12:header.FooterOffset])
	footerBytes := data[header.FooterOffset:]

	var words []Word

	if header.IsOrdered {
		words, err = decodeOrderedFooter(footerBytes, superstring)
	} else {
		words, err = decodeUnorderedFooter(footerBytes, superstring)
	}

	if err != nil {
		return nil, "", nil, err
	}

	return header, superstring, words, nil
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
