package format

import "testing"

func TestBitWriterReader_RoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		values []struct {
			val   uint64
			width uint
		}
	}{
		{
			name: "6-bit and 17-bit crossing byte boundary",
			values: []struct {
				val   uint64
				width uint
			}{
				{42, 6},      // 6 bits
				{100000, 17}, // 17 bits, total 23 bits = crosses 3 bytes
			},
		},
		{
			name: "mixed widths",
			values: []struct {
				val   uint64
				width uint
			}{
				{7, 3},
				{255, 8},
				{1023, 10},
				{0, 5},
				{1, 1},
			},
		},
		{
			name: "single bit values",
			values: []struct {
				val   uint64
				width uint
			}{
				{0, 1},
				{1, 1},
				{1, 1},
				{0, 1},
			},
		},
		{
			name: "large values",
			values: []struct {
				val   uint64
				width uint
			}{
				{(1 << 20) - 1, 20},
				{(1 << 15) - 1, 15},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bw := &BitWriter{}
			for _, v := range tt.values {
				bw.WriteBits(v.val, v.width)
			}
			data := bw.Flush()

			br := NewBitReader(data)
			for _, v := range tt.values {
				got := br.ReadBits(v.width)
				if got != v.val {
					t.Errorf("ReadBits(%d) = %d, want %d", v.width, got, v.val)
				}
			}
		})
	}
}

func TestBitWriter_ZeroPadding(t *testing.T) {
	bw := &BitWriter{}
	bw.WriteBits(0x1F, 5) // 5 bits: 11111 -> in LSB-first: byte = 0001_1111 = 0x1F
	data := bw.Flush()

	if len(data) != 1 {
		t.Fatalf("expected 1 byte, got %d", len(data))
	}
	// The remaining 3 bits should be zero-padded.
	if data[0] != 0x1F {
		t.Errorf("byte = 0x%02X, want 0x1F", data[0])
	}
}

func TestBitWriter_EmptyFlush(t *testing.T) {
	bw := &BitWriter{}
	data := bw.Flush()
	if len(data) != 0 {
		t.Errorf("expected empty buffer, got %d bytes", len(data))
	}
}
