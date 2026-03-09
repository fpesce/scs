package format

import (
	"bytes"
	"math"
	"testing"
)

func TestEncodeDecodeULEB128_RoundTrip(t *testing.T) {
	values := []uint64{
		0, 1, 127, 128, 255, 256, 16383, 16384,
		1<<21 - 1, 1 << 21, 1<<28 - 1, 1 << 28,
		1<<35 - 1, 1 << 35, 1<<63 - 1, math.MaxUint64,
	}

	for _, v := range values {
		encoded := EncodeULEB128(v)
		r := bytes.NewReader(encoded)
		decoded, n, err := DecodeULEB128(r)
		if err != nil {
			t.Fatalf("DecodeULEB128(%d) error: %v", v, err)
		}
		if decoded != v {
			t.Errorf("DecodeULEB128(EncodeULEB128(%d)) = %d", v, decoded)
		}
		if n != len(encoded) {
			t.Errorf("bytes read = %d, encoded length = %d", n, len(encoded))
		}
	}
}

func TestEncodeULEB128_KnownValues(t *testing.T) {
	tests := []struct {
		value uint64
		want  []byte
	}{
		{0, []byte{0x00}},
		{1, []byte{0x01}},
		{127, []byte{0x7F}},
		{128, []byte{0x80, 0x01}},
		{624485, []byte{0xE5, 0x8E, 0x26}},
	}

	for _, tt := range tests {
		got := EncodeULEB128(tt.value)
		if !bytes.Equal(got, tt.want) {
			t.Errorf("EncodeULEB128(%d) = %x, want %x", tt.value, got, tt.want)
		}
	}
}

func TestDecodeULEB128_MaxUint64(t *testing.T) {
	encoded := EncodeULEB128(math.MaxUint64)
	r := bytes.NewReader(encoded)
	v, _, err := DecodeULEB128(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != math.MaxUint64 {
		t.Errorf("got %d, want %d", v, uint64(math.MaxUint64))
	}
}

func TestDecodeULEB128_DoSOverflow(t *testing.T) {
	// Create a byte sequence that keeps the continuation bit set for 11 bytes.
	malicious := make([]byte, 11)
	for i := 0; i < 11; i++ {
		malicious[i] = 0x80 // All continuation bits set, no termination.
	}

	r := bytes.NewReader(malicious)
	_, _, err := DecodeULEB128(r)
	if err == nil {
		t.Fatal("expected DoS overflow error, got nil")
	}
}

func TestDecodeULEB128_EmptyReader(t *testing.T) {
	r := bytes.NewReader([]byte{})
	_, _, err := DecodeULEB128(r)
	if err == nil {
		t.Fatal("expected error for empty reader, got nil")
	}
}
