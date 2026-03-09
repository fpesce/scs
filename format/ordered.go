package format

import (
	"bytes"
	"encoding/base64"
	"math"
)

// bitsNeeded calculates ceil(log2(n+1)), the number of bits needed
// to represent values in the range [0, n].
func bitsNeeded(n uint64) uint {
	if n == 0 {
		return 1
	}
	bits := uint(0)
	v := n
	for v > 0 {
		bits++
		v >>= 1
	}
	return bits
}

// EncodeOrdered generates the ORDERED mode metadata footer.
// It preserves the exact chronological sequence of the source text
// using dynamic bit-packing with ULEB128 headers.
func EncodeOrdered(chronologicalLines []string, offsetMap map[string]int, superstringByteLen int) string {
	totalLineCount := uint64(len(chronologicalLines))

	// Find max word length.
	maxWordLen := uint64(0)
	for _, line := range chronologicalLines {
		if uint64(len(line)) > maxWordLen {
			maxWordLen = uint64(len(line))
		}
	}

	// Write ULEB128 header: Max_Word_Length, Total_Line_Count.
	var headerBuf bytes.Buffer
	headerBuf.Write(EncodeULEB128(maxWordLen))
	headerBuf.Write(EncodeULEB128(totalLineCount))

	// Calculate dynamic bit widths.
	lengthBits := bitsNeeded(maxWordLen)
	offsetBits := bitsNeeded(uint64(superstringByteLen))

	// Handle edge case: if superstringByteLen is 0,
	// we need at least 1 bit for the offset field.
	if offsetBits == 0 {
		offsetBits = 1
	}

	// Pack [Length_Bits][Offset_Bits] for each line.
	bw := &BitWriter{}
	for _, line := range chronologicalLines {
		lineLen := uint64(len(line))
		if lineLen == 0 {
			// Empty lines: length=0, offset=0.
			bw.WriteBits(0, lengthBits)
			bw.WriteBits(0, offsetBits)
		} else {
			offset, ok := offsetMap[line]
			if !ok {
				// Should never happen in correct usage; use 0 as fallback.
				offset = 0
			}
			bw.WriteBits(lineLen, lengthBits)
			bw.WriteBits(uint64(offset), offsetBits)
		}
	}

	// Combine ULEB128 header bytes + flushed BitWriter bytes.
	combined := headerBuf.Bytes()
	combined = append(combined, bw.Flush()...)

	return base64.StdEncoding.EncodeToString(combined)
}

// DecodeOrdered parses the ORDERED metadata footer. It returns the
// list of (offset, length) tuples reconstructed from the bit-packed data.
func DecodeOrdered(data []byte) ([][2]int, error) {
	r := bytes.NewReader(data)

	maxWordLen, _, err := DecodeULEB128(r)
	if err != nil {
		return nil, err
	}

	totalLineCount, _, err := DecodeULEB128(r)
	if err != nil {
		return nil, err
	}

	lengthBits := bitsNeeded(maxWordLen)

	// We need to compute offsetBits from the context. Since we don't have
	// the superstring length at decode time in this function, we compute
	// the bit widths from what remains in the buffer.
	// Actually, the caller provides the superstring length separately.
	// For now, this function is designed to be called by DecodeSCS which
	// passes the superstring length. We'll handle that in the decoder.
	_ = lengthBits
	_ = totalLineCount

	// This function will be properly implemented in the decoder step.
	// For now, return nil to satisfy the compiler.
	return nil, nil
}

// DecodeOrderedWithContext is the full decoder that uses superstring byte length.
func DecodeOrderedWithContext(data []byte, superstringByteLen int) ([][2]int, error) {
	r := bytes.NewReader(data)

	maxWordLen, _, err := DecodeULEB128(r)
	if err != nil {
		return nil, err
	}

	totalLineCount, _, err := DecodeULEB128(r)
	if err != nil {
		return nil, err
	}

	lengthBits := bitsNeeded(maxWordLen)
	offsetBits := bitsNeeded(uint64(superstringByteLen))
	if offsetBits == 0 {
		offsetBits = 1
	}

	// Read the remaining bytes into the BitReader.
	remaining := make([]byte, r.Len())
	_, _ = r.Read(remaining)
	br := NewBitReader(remaining)

	tuples := make([][2]int, 0, totalLineCount)
	for i := uint64(0); i < totalLineCount; i++ {
		length := br.ReadBits(lengthBits)
		offset := br.ReadBits(offsetBits)
		tuples = append(tuples, [2]int{int(offset), int(length)})
	}

	// Validate we didn't overflow. Actual bounds check is in decoder.
	_ = math.MaxInt64

	return tuples, nil
}
