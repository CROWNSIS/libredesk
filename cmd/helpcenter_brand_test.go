package main

import (
	"os"
	"strings"
	"testing"
)

func TestHelpCenterBrandShowsLogoAndName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "docs",
			path:     "../static/public/web-templates/help/docs/layout.html",
			expected: `{{ if .Data.HelpCenter.LogoURL }}<img src="{{ .Data.HelpCenter.LogoURL }}" alt="" aria-hidden="true" />{{ end }}<span class="hcd-brand-name">{{ .Data.HelpCenter.Name }}</span>`,
		},
		{
			name:     "classic",
			path:     "../static/public/web-templates/help/classic/layout.html",
			expected: `{{ if .Data.HelpCenter.LogoURL }}<img src="{{ .Data.HelpCenter.LogoURL }}" alt="" aria-hidden="true" />{{ end }}<span class="hc-name">{{ .Data.HelpCenter.Name }}</span>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			contents, err := os.ReadFile(tt.path)
			if err != nil {
				t.Fatalf("read help center template: %v", err)
			}
			if !strings.Contains(string(contents), tt.expected) {
				t.Fatalf("help center brand must render its configured logo and visible name together")
			}
		})
	}
}

func TestHelpCenterBrandNameCollapsesOnPhones(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{name: "docs", path: "../static/public/static/help-center-docs.css", expected: ".hcd-brand-name { display: none; }"},
		{name: "classic", path: "../static/public/static/help-center-classic.css", expected: ".hc-brand .hc-name { display: none; }"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			contents, err := os.ReadFile(tt.path)
			if err != nil {
				t.Fatalf("read help center stylesheet: %v", err)
			}
			if !strings.Contains(string(contents), tt.expected) {
				t.Fatalf("help center name must collapse at the phone breakpoint")
			}
		})
	}
}
