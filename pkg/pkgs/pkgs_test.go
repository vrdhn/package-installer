package pkgs

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input     string
		wantName  string
		wantVer   string
		wantError bool
	}{
		{"python", "python", "latest", false},
		{"python=3.12", "python", "3.12", false},
		{"pip:numpy", "pip:numpy", "latest", false},
		{"pip:numpy=1.26.4", "pip:numpy", "1.26.4", false},
		{"npm:@gemini-cli/latest", "npm:@gemini-cli/latest", "latest", false},
		{"", "", "", true},
		{":name", ":name", "latest", false},
	}

	for _, tt := range tests {
		got, err := Parse(tt.input)
		if (err != nil) != tt.wantError {
			t.Errorf("Parse(%q) error = %v, wantError %v", tt.input, err, tt.wantError)
			continue
		}
		if tt.wantError {
			continue
		}
		if got.Name != tt.wantName {
			t.Errorf("Parse(%q) Name = %q, want %q", tt.input, got.Name, tt.wantName)
		}
		if got.Version != tt.wantVer {
			t.Errorf("Parse(%q) Version = %q, want %q", tt.input, got.Version, tt.wantVer)
		}
	}
}
