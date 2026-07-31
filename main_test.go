package main

import (
	"bytes"
	"discord-embedder/internal/web"
	"html/template"
	"strings"
	"testing"
)

func TestHomeTemplateSourceURL(t *testing.T) {
	tmpl, err := template.ParseFS(home, "templates/home.html")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		sourceURL string
		wantLink  bool
	}{
		{
			name:      "source link",
			sourceURL: "https://example.com/video?id=1&token=2",
			wantLink:  true,
		},
		{
			name:     "legacy video without source",
			wantLink: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			page := web.Page{
				ImageURL:  "https://embed.example/files/video.jpeg",
				VideoURL:  "https://embed.example/files/video.mp4",
				SourceURL: tt.sourceURL,
				Width:     "1920",
				Height:    "1080",
			}
			if err = tmpl.Execute(&output, page); err != nil {
				t.Fatal(err)
			}

			html := output.String()
			hasLink := strings.Contains(html, ">View original</a>")
			if hasLink != tt.wantLink {
				t.Errorf("source link present = %v, want %v", hasLink, tt.wantLink)
			}
			if tt.wantLink && !strings.Contains(html, `href="https://example.com/video?id=1&amp;token=2"`) {
				t.Errorf("source URL was not safely rendered: %q", html)
			}
			if !strings.Contains(html, `content="https://embed.example/files/video.mp4"`) {
				t.Errorf("video Open Graph URL missing: %q", html)
			}
			if !strings.Contains(html, `<source src="https://embed.example/files/video.mp4" />`) {
				t.Errorf("video source missing: %q", html)
			}
		})
	}
}
