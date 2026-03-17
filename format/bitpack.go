package format

// BitWriter packs arbitrary-width integers into a byte buffer using
// LSB-first (Little-Endian Bit Order). Bits are written starting from
// the least significant bit of each byte, seamlessly crossing byte boundaries.
type BitWriter struct {
	buf    []byte
	bitPos uint // Current bit position within the buffer.
}

// WriteBits writes the lowest `width` bits of `value` to the buffer.
func (bw *BitWriter) WriteBits(value uint64, width uint) {
	for i := range width {
		byteIdx := bw.bitPos / 8
		bitIdx := bw.bitPos % 8

		// Grow buffer if needed.
		for byteIdx >= uint(len(bw.buf)) {
			bw.buf = append(bw.buf, 0)
		}

		// Extract bit i from value and place it at bitIdx in the byte.
		bit := (value >> i) & 1
		bw.buf[byteIdx] |= byte(bit << bitIdx)

		bw.bitPos++
	}
}

// Flush returns the final byte buffer, zero-padded to align with an 8-bit boundary.
func (bw *BitWriter) Flush() []byte {
	return bw.buf
}

// BitReader reads arbitrary-width integers from a byte buffer using
// LSB-first (Little-Endian Bit Order).
type BitReader struct {
	buf    []byte
	bitPos uint
}

// NewBitReader creates a BitReader from a byte slice.
func NewBitReader(data []byte) *BitReader {
	return &BitReader{buf: data}
}

// ReadBits reads `width` bits from the buffer and returns them as a uint64.
func (br *BitReader) ReadBits(width uint) uint64 {
	var value uint64
	for i := range width {
		byteIdx := br.bitPos / 8
		bitIdx := br.bitPos % 8

		if byteIdx < uint(len(br.buf)) {
			bit := (uint64(br.buf[byteIdx]) >> bitIdx) & 1
			value |= bit << i
		}

		br.bitPos++
	}
	return value
}
