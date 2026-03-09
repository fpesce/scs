package format

import "testing"

func TestHeaderRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		h    Header
	}{
		{
			name: "ordered with newline separator",
			h: Header{
				Version:      0x01,
				Separator:    '\n',
				IsOrdered:    true,
				FooterOffset: 1234,
			},
		},
		{
			name: "unordered with comma separator",
			h: Header{
				Version:      0x01,
				Separator:    ',',
				IsOrdered:    false,
				FooterOffset: 99999,
			},
		},
		{
			name: "zero offset",
			h: Header{
				Version:      0x01,
				Separator:    '\n',
				IsOrdered:    true,
				FooterOffset: 0,
			},
		},
		{
			name: "max 55-bit offset",
			h: Header{
				Version:      0x01,
				Separator:    0,
				IsOrdered:    false,
				FooterOffset: (1 << 55) - 1,
			},
		},
		{
			name: "null separator",
			h: Header{
				Version:      0x01,
				Separator:    0x00,
				IsOrdered:    true,
				FooterOffset: 42,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := EncodeHeader(&tt.h)

			// Verify exactly 16 Base64 characters.
			if len(encoded) != 16 {
				t.Fatalf("encoded length = %d, want 16", len(encoded))
			}

			decoded, err := DecodeHeader(encoded)
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}

			if decoded.Version != tt.h.Version {
				t.Errorf("Version = 0x%02X, want 0x%02X", decoded.Version, tt.h.Version)
			}
			if decoded.Separator != tt.h.Separator {
				t.Errorf("Separator = 0x%02X, want 0x%02X", decoded.Separator, tt.h.Separator)
			}
			if decoded.IsOrdered != tt.h.IsOrdered {
				t.Errorf("IsOrdered = %v, want %v", decoded.IsOrdered, tt.h.IsOrdered)
			}
			if decoded.FooterOffset != tt.h.FooterOffset {
				t.Errorf("FooterOffset = %d, want %d", decoded.FooterOffset, tt.h.FooterOffset)
			}
		})
	}
}

func TestDecodeHeader_InvalidMagic(t *testing.T) {
	// Manually create an invalid header.
	encoded := EncodeHeader(&Header{Version: 0x01, Separator: '\n', IsOrdered: true, FooterOffset: 0})
	// Corrupt the first character by modifying the encoded string.
	// We'll test with an entirely wrong Base64 string.
	_, err := DecodeHeader("AAAAAAAAAAAAAAAA") // 16 chars but wrong magic.
	if err == nil {
		t.Fatal("expected error for invalid magic, got nil")
	}
	_ = encoded
}

func TestDecodeHeader_InvalidLength(t *testing.T) {
	_, err := DecodeHeader("short")
	if err == nil {
		t.Fatal("expected error for short input, got nil")
	}
}
