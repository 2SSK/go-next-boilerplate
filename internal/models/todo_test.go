package models

import (
	"strings"
	"testing"
)

func TestNormalizeTitle(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr string
	}{
		{"trims surrounding space", "  buy milk  ", "buy milk", ""},
		{"blank", "   ", "", "required"},
		{"too long", strings.Repeat("a", MaxTitleLen+1), "", "at most 200"},
		{"multibyte at the limit counts characters, not bytes", strings.Repeat("é", MaxTitleLen), strings.Repeat("é", MaxTitleLen), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeTitle(tt.in)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want it to mention %q", err, tt.wantErr)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("NormalizeTitle = %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}
