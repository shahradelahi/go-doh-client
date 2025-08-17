package doh

import (
	"testing"
)

func TestToPunycode(t *testing.T) {
	tests := []struct {
		name   string
		domain Domain
		want   string
	}{
		{"Chinese", "中文.com", "xn--fiq228c.com"},
		{"Persian", "فارسی.com", "xn--mgbug7c06b.com"},
		{"Underscore", "_esni.example.com", "_esni.example.com"},
		{"Standard", "example.com", "example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.domain.Punycode()
			if err != nil {
				t.Fatalf("Punycode() returned an unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Punycode() got = %v, want %v", got, tt.want)
			}
		})
	}
}
