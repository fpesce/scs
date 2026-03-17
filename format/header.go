package format

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// HeaderSize is the fixed byte length of the .scs file header.
const HeaderSize = 12

// VersionCurrent is the only supported .scs format version.
const VersionCurrent = 0x02

// ModeFlagBit is the bit position within the 56-bit footer field
// that encodes ORDERED (1) vs UNORDERED (0) mode.
const ModeFlagBit = 55

// FilePermissions is the default permission mode for output files (owner-only rw).
const FilePermissions = 0o600

// Header represents the 12-byte magic pointer header of an .scs file.
type Header struct {
	Version      byte
	Separator    byte
	IsOrdered    bool
	FooterOffset uint64 // Lower 55 bits of the 56-bit field.
}

// Magic is the 3-byte ASCII signature for SCS files.
var Magic = [3]byte{'S', 'C', 'S'} //nolint:gochecknoglobals // file format constant

// EncodeHeader packs the Header into exactly 12 raw bytes.
func EncodeHeader(h *Header) []byte {
	var buf [HeaderSize]byte

	// Bytes 0-2: Magic string "SCS".
	copy(buf[0:3], Magic[:])

	// Byte 3: Version.
	buf[3] = h.Version

	// Byte 4: Separator character.
	buf[4] = h.Separator

	// Bytes 5-11: 56-bit LE uint.
	// Bit 55 = mode flag (1=ORDERED, 0=UNORDERED).
	// Lower 55 bits = footer offset.
	val := h.FooterOffset & ((1 << ModeFlagBit) - 1)
	if h.IsOrdered {
		val |= 1 << ModeFlagBit
	}

	// Write as 7 bytes, little-endian.
	var leBytes [8]byte
	binary.LittleEndian.PutUint64(leBytes[:], val)
	copy(buf[5:12], leBytes[0:7])

	return buf[:]
}

// DecodeHeader decodes 12 raw bytes into a Header struct.
func DecodeHeader(raw []byte) (*Header, error) {
	if len(raw) < HeaderSize {
		return nil, fmt.Errorf("header must be at least %d bytes, got %d", HeaderSize, len(raw))
	}

	// Validate magic string.
	if raw[0] != 'S' || raw[1] != 'C' || raw[2] != 'S' {
		return nil, errors.New("invalid magic string: expected 'SCS'")
	}

	// Validate version.
	if raw[3] != VersionCurrent {
		return nil, fmt.Errorf("unsupported version: 0x%02X (expected 0x02)", raw[3])
	}

	h := &Header{
		Version:   raw[3],
		Separator: raw[4],
	}

	// Read 56-bit LE uint from bytes 5-11.
	var leBytes [8]byte
	copy(leBytes[0:7], raw[5:12])
	val := binary.LittleEndian.Uint64(leBytes[:])

	// Extract mode flag (bit 55) and footer offset (lower 55 bits).
	h.IsOrdered = (val>>ModeFlagBit)&1 == 1
	h.FooterOffset = val & ((1 << ModeFlagBit) - 1)

	return h, nil
}
