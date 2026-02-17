package encoding

import (
	"testing"
)

func TestEncode(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		{"zero", 0, "a"},
		{"one", 1, "b"},
		{"twenty-five", 25, "z"},
		{"twenty-six", 26, "A"},
		{"fifty-one", 51, "Z"},
		{"fifty-two", 52, "0"},
		{"sixty-one", 61, "9"},
		{"sixty-two (first two-char)", 62, "ba"},
		{"sixty-three", 63, "bb"},
		{"bbb equals 3907", 3907, "bbb"},
		{"known value", 3844, "baa"},
		{"large number", 123456789, "iwaUH"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Encode(tt.input)
			if result != tt.expected {
				t.Errorf("Encode(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEncode_Negative(t *testing.T) {
	result := Encode(-1)
	if result != "" {
		t.Errorf("Encode(-1) = %q, want empty string", result)
	}
}

func TestDecode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
		wantErr  bool
	}{
		{"a to zero", "a", 0, false},
		{"b to one", "b", 1, false},
		{"z to twenty-five", "z", 25, false},
		{"A to twenty-six", "A", 26, false},
		{"Z to fifty-one", "Z", 51, false},
		{"0 to fifty-two", "0", 52, false},
		{"9 to sixty-one", "9", 61, false},
		{"ba to sixty-two", "ba", 62, false},
		{"bb to sixty-three", "bb", 63, false},
		{"bbb to 3907", "bbb", 3907, false},
		{"baa to known", "baa", 3844, false},
		{"large known", "iwaUH", 123456789, false},
		{"empty string", "", 0, true},
		{"invalid char", "a!b", 0, true},
		{"space", "a b", 0, true},
		{"underscore", "a_b", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Decode(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Decode(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("Decode(%q) unexpected error: %v", tt.input, err)
				return
			}
			if result != tt.expected {
				t.Errorf("Decode(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	// Test that encoding and decoding produces the original value
	values := []int64{0, 1, 10, 61, 62, 100, 1000, 10000, 100000, 1000000, 123456789}

	for _, v := range values {
		encoded := Encode(v)
		decoded, err := Decode(encoded)
		if err != nil {
			t.Errorf("RoundTrip(%d): decode error: %v", v, err)
			continue
		}
		if decoded != v {
			t.Errorf("RoundTrip(%d): encoded to %q, decoded to %d", v, encoded, decoded)
		}
	}
}

func BenchmarkEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Encode(int64(i))
	}
}

func BenchmarkDecode(b *testing.B) {
	codes := []string{"a", "ba", "baa", "dGvVBF"}
	for i := 0; i < b.N; i++ {
		_, _ = Decode(codes[i%len(codes)])
	}
}
