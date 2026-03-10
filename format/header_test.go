package format

import (
	"testing"
)

func TestHeaderRoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		header    *Header
	}{
		{
			name: "ordered with newline separator",
			header: &Header{
				Version:      0x02,
				Separator:    '\n',
				IsOrdered:    true,
				FooterOffset: 12345,
			},
		},
		{
			name: "unordered with tab separator",
			header: &Header{
				Version:      0x02,
				Separator:    '\t',
				IsOrdered:    false,
				FooterOffset: 67890,
			},
		},
		{
			name: "zero offset ordered",
			header: &Header{
				Version:      0x02,
				Separator:    '\n',
				IsOrdered:    true,
				FooterOffset: 12, // Minimum: header size.
			},
		},
		{
			name: "large offset",
			header: &Header{
				Version:      0x02,
				Separator:    0x00,
				IsOrdered:    true,
				FooterOffset: (1 << 55) - 1, // Max 55-bit value.
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := EncodeHeader(tt.header)

			// Must be exactly 12 bytes.
			if len(encoded) != 12 {
				t.Fatalf("encoded length = %d, want 12", len(encoded))
			}

			decoded, err := DecodeHeader(encoded)
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}

			if decoded.Version != tt.header.Version {
				t.Errorf("version = 0x%02X, want 0x%02X", decoded.Version, tt.header.Version)
			}
			if decoded.Separator != tt.header.Separator {
				t.Errorf("separator = %d, want %d", decoded.Separator, tt.header.Separator)
			}
			if decoded.IsOrdered != tt.header.IsOrdered {
				t.Errorf("isOrdered = %v, want %v", decoded.IsOrdered, tt.header.IsOrdered)
			}
			if decoded.FooterOffset != tt.header.FooterOffset {
				t.Errorf("footerOffset = %d, want %d", decoded.FooterOffset, tt.header.FooterOffset)
			}
		})
	}
}

func TestDecodeHeader_InvalidMagic(t *testing.T) {
	bad := []byte("XXX\x02\n\x00\x00\x00\x00\x00\x00\x00")
	_, err := DecodeHeader(bad)
	if err == nil {
		t.Error("expected error for invalid magic string")
	}
}

func TestDecodeHeader_InvalidVersion(t *testing.T) {
	bad := []byte("SCS\x01\n\x00\x00\x00\x00\x00\x00\x00")
	_, err := DecodeHeader(bad)
	if err == nil {
		t.Error("expected error for version 0x01 (we require 0x02)")
	}
}

func TestDecodeHeader_TooShort(t *testing.T) {
	_, err := DecodeHeader([]byte("SCS"))
	if err == nil {
		t.Error("expected error for short input")
	}
}
