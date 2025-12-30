package blockchain

import (
	"bytes"
	"strings"
	"testing"
)

func TestStringToBytes32(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  [32]byte
	}{
		{
			name:  "empty string",
			input: "",
			want:  [32]byte{},
		},
		{
			name:  "short string",
			input: "test",
			want: func() [32]byte {
				var b [32]byte
				copy(b[:], "test")
				return b
			}(),
		},
		{
			name:  "exactly 32 bytes",
			input: "12345678901234567890123456789012",
			want: func() [32]byte {
				var b [32]byte
				copy(b[:], "12345678901234567890123456789012")
				return b
			}(),
		},
		{
			name:  "longer than 32 bytes (truncated)",
			input: "this_is_a_very_long_string_that_exceeds_32_bytes_and_will_be_truncated",
			want: func() [32]byte {
				var b [32]byte
				copy(b[:], "this_is_a_very_long_string_that_")
				return b
			}(),
		},
		{
			name:  "organization id",
			input: "snet",
			want: func() [32]byte {
				var b [32]byte
				copy(b[:], "snet")
				return b
			}(),
		},
		{
			name:  "service id",
			input: "example-service",
			want: func() [32]byte {
				var b [32]byte
				copy(b[:], "example-service")
				return b
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StringToBytes32(tt.input)
			if got != tt.want {
				t.Errorf("StringToBytes32() = %v, want %v", got, tt.want)
			}

			// Verify the string portion matches
			inputLen := len(tt.input)
			if inputLen > 32 {
				inputLen = 32
			}
			if !bytes.Equal(got[:inputLen], []byte(tt.input[:inputLen])) {
				t.Errorf("StringToBytes32() content mismatch")
			}
		})
	}
}

func TestBytes32ArrayToStrings(t *testing.T) {
	tests := []struct {
		name  string
		input [][32]byte
		want  []string
	}{
		{
			name:  "empty array",
			input: [][32]byte{},
			want:  []string{},
		},
		{
			name: "single element",
			input: [][32]byte{
				StringToBytes32("test"),
			},
			want: []string{"test"},
		},
		{
			name: "multiple elements",
			input: [][32]byte{
				StringToBytes32("org1"),
				StringToBytes32("org2"),
				StringToBytes32("org3"),
			},
			want: []string{"org1", "org2", "org3"},
		},
		{
			name: "elements with padding",
			input: [][32]byte{
				StringToBytes32("snet"),
				StringToBytes32("example-service"),
				StringToBytes32("a"),
			},
			want: []string{"snet", "example-service", "a"},
		},
		{
			name: "real organization ids",
			input: [][32]byte{
				StringToBytes32("singularitynet"),
				StringToBytes32("nunet"),
				StringToBytes32("rejuve"),
			},
			want: []string{"singularitynet", "nunet", "rejuve"},
		},
		{
			name: "mixed length strings",
			input: [][32]byte{
				StringToBytes32(""),
				StringToBytes32("x"),
				StringToBytes32("medium_length_id"),
				StringToBytes32("12345678901234567890123456789012"),
			},
			want: []string{"", "x", "medium_length_id", "12345678901234567890123456789012"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Bytes32ArrayToStrings(tt.input)

			if len(got) != len(tt.want) {
				t.Fatalf("Bytes32ArrayToStrings() length = %d, want %d", len(got), len(tt.want))
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("Bytes32ArrayToStrings()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestStringToBytes32_RoundTrip(t *testing.T) {
	testStrings := []string{
		"",
		"a",
		"test",
		"organization-id",
		"service-identifier",
		"exactly-32-chars-string-here",
		"this-exceeds-32-characters-limit",
	}

	for _, original := range testStrings {
		t.Run(original, func(t *testing.T) {
			bytes32 := StringToBytes32(original)
			arr := [][32]byte{bytes32}
			result := Bytes32ArrayToStrings(arr)

			expected := original
			if len(original) > 32 {
				expected = original[:32]
			}

			if len(result) != 1 {
				t.Fatalf("expected 1 element, got %d", len(result))
			}

			if result[0] != expected {
				t.Errorf("round trip failed: got %q, want %q", result[0], expected)
			}
		})
	}
}

func TestBytes32ArrayToStrings_PreservesOrder(t *testing.T) {
	orgIDs := []string{"alpha", "beta", "gamma", "delta", "epsilon"}

	var bytes32Array [][32]byte
	for _, id := range orgIDs {
		bytes32Array = append(bytes32Array, StringToBytes32(id))
	}

	result := Bytes32ArrayToStrings(bytes32Array)

	if len(result) != len(orgIDs) {
		t.Fatalf("length mismatch: got %d, want %d", len(result), len(orgIDs))
	}

	for i, expected := range orgIDs {
		if result[i] != expected {
			t.Errorf("order not preserved at index %d: got %q, want %q", i, result[i], expected)
		}
	}
}

func TestBytes32ArrayToStrings_TrimsNullBytes(t *testing.T) {
	var b [32]byte
	copy(b[:], "test")
	// Remaining bytes are already zero/null

	arr := [][32]byte{b}
	result := Bytes32ArrayToStrings(arr)

	if len(result) != 1 {
		t.Fatalf("expected 1 element, got %d", len(result))
	}

	if result[0] != "test" {
		t.Errorf("expected 'test', got %q", result[0])
	}

	// Verify no null bytes in result
	if strings.Contains(result[0], "\x00") {
		t.Error("result contains null bytes")
	}
}

func TestStringToBytes32_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"hyphen", "my-org-id"},
		{"underscore", "my_service_id"},
		{"numbers", "org123"},
		{"mixed", "Org-123_Test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StringToBytes32(tt.input)

			// Convert back and verify
			arr := [][32]byte{result}
			recovered := Bytes32ArrayToStrings(arr)[0]

			if recovered != tt.input {
				t.Errorf("special characters not preserved: got %q, want %q", recovered, tt.input)
			}
		})
	}
}
