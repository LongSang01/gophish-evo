package api

import "testing"

func TestSanitizeCSVValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"normal text", "hello", "hello"},
		{"empty string", "", ""},
		{"formula equals", "=cmd", "'=cmd"},
		{"formula plus", "+SUM(A1)", "'+SUM(A1)"},
		{"formula minus", "-2+3", "'-2+3"},
		{"formula at", "@SUM", "'@SUM"},
		{"formula tab", "\tdata", "'\tdata"},
		{"formula cr", "\rdata", "'\rdata"},
		{"safe dash in middle", "a-b", "a-b"},
		{"safe number", "123", "123"},
		{"space prefix", " hello", " hello"},
		{"single char equals", "=", "'="},
		{"single char plus", "+", "'+"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeCSVValue(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeCSVValue(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
