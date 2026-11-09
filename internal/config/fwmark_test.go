package config

import "testing"

func TestParseFWMark(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  uint32
	}{
		{"0", 0}, {"256", 256}, {"0x100", 256}, {"0Xff", 255},
		{"010", 10}, {"4294967295", 0xffffffff}, {"0xffffffff", 0xffffffff},
	} {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseFWMark(tt.input)
			if err != nil || got != tt.want {
				t.Fatalf("ParseFWMark(%q) = %d, %v; want %d", tt.input, got, err, tt.want)
			}
		})
	}
	for _, input := range []string{"", "-1", "+1", "4294967296", "0x100000000", "0x", "0xgg", "1_000", "1.5", " 1", "1\n", "0b10", "1;id"} {
		if _, err := ParseFWMark(input); err == nil {
			t.Errorf("ParseFWMark(%q) should fail", input)
		}
	}
}
