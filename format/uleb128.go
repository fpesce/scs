// Package format implements the binary encoding and decoding
// primitives for the .scs file format.
package format

import (
	"errors"
	"fmt"
	"io"
)

// maxULEB128Bytes is the maximum number of bytes a valid ULEB128-encoded
// uint64 can span (ceil(64/7) = 10). Reading beyond this limit indicates
// a maliciously crafted or corrupt stream (DoS protection).
const maxULEB128Bytes = 10

// EncodeULEB128 encodes a uint64 value into ULEB128 (Little-Endian Base 128) format.
func EncodeULEB128(value uint64) []byte {
	if value == 0 {
		return []byte{0}
	}
	var buf []byte
	for value > 0 {
		b := byte(value & 0x7F)
		value >>= 7
		if value > 0 {
			b |= 0x80 // Set continuation bit.
		}
		buf = append(buf, b)
	}
	return buf
}

// DecodeULEB128 reads a single ULEB128-encoded uint64 from a ByteReader.
// Returns the decoded value, the number of bytes read, and any error.
// It fatally errors if more than 10 bytes are consumed (DoS protection).
func DecodeULEB128(r io.ByteReader) (uint64, int, error) {
	var result uint64
	var shift uint
	for i := 0; i < maxULEB128Bytes; i++ {
		b, err := r.ReadByte()
		if err != nil {
			return 0, i, fmt.Errorf("reading ULEB128 byte %d: %w", i, err)
		}
		result |= uint64(b&0x7F) << shift
		if b&0x80 == 0 {
			return result, i + 1, nil
		}
		shift += 7
	}
	return 0, maxULEB128Bytes, errors.New("ULEB128 overflow: exceeded 10-byte limit (DoS protection)")
}
