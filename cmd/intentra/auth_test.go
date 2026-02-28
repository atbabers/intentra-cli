package main

import (
	"testing"
)

func TestOpenBrowserRejectsHTTP(t *testing.T) {
	err := openBrowser("http://evil.com")
	if err == nil {
		t.Error("openBrowser should reject non-HTTPS URLs")
	}
}

func TestOpenBrowserRejectsJavascript(t *testing.T) {
	err := openBrowser("javascript:alert(1)")
	if err == nil {
		t.Error("openBrowser should reject javascript: URLs")
	}
}

func TestOpenBrowserAcceptsHTTPS(t *testing.T) {
	// This will fail to actually open a browser in CI, but should not reject the URL
	err := openBrowser("https://example.com")
	// We allow err != nil since the browser command may not exist in test env,
	// but the error should NOT be about the scheme
	if err != nil && err.Error() == "refusing to open non-HTTPS URL: http" {
		t.Error("openBrowser should accept HTTPS URLs")
	}
}

func TestCapitalizeFirst(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"free", "Free"},
		{"pro", "Pro"},
		{"enterprise", "Enterprise"},
		{"", ""},
		{"A", "A"},
		{"already Capitalized", "Already Capitalized"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := capitalizeFirst(tt.input)
			if got != tt.want {
				t.Errorf("capitalizeFirst(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
