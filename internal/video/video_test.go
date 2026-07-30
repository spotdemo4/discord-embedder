package video

import (
	"context"
	"discord-embedder/internal/config"
	"os"
	"path/filepath"
	"testing"
)

func TestGet(t *testing.T) {
	id := "video-id"
	tests := []struct {
		name          string
		files         []string
		wantName      string
		wantThumbnail bool
		wantErr       bool
	}{
		{
			name:          "canonical mp4 preferred",
			files:         []string{id + ".mp4", id + ".webm", id + ".webm.part", id + ".jpeg"},
			wantName:      id + ".mp4",
			wantThumbnail: true,
		},
		{
			name:          "single legacy video",
			files:         []string{id + ".mkv", id + ".json", id + ".jpeg"},
			wantName:      id + ".mkv",
			wantThumbnail: true,
		},
		{
			name:    "partial and sidecars ignored",
			files:   []string{id + ".webm.part", id + ".f137.mp4", id + ".json", id + ".vtt"},
			wantErr: true,
		},
		{
			name:    "prefix match rejected",
			files:   []string{id + "-other.mp4", id + "-other.jpeg"},
			wantErr: true,
		},
		{
			name:    "ambiguous legacy videos",
			files:   []string{id + ".webm", id + ".mkv"},
			wantErr: true,
		},
		{
			name:          "thumbnail must match exactly",
			files:         []string{id + ".mp4", id + "-other.jpeg"},
			wantName:      id + ".mp4",
			wantThumbnail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, name := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte("test"), 0600); err != nil {
					t.Fatal(err)
				}
			}

			ctx := config.WithConfig(context.Background(), &config.Config{FilesDir: dir})
			got, err := Get(ctx, id)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Get() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got.Name != tt.wantName {
				t.Errorf("Get().Name = %q, want %q", got.Name, tt.wantName)
			}
			if (got.Thumbnail != nil) != tt.wantThumbnail {
				t.Errorf("Get().Thumbnail present = %v, want %v", got.Thumbnail != nil, tt.wantThumbnail)
			}
		})
	}
}

func TestIsLegacyVideo(t *testing.T) {
	tests := []struct {
		name string
		file string
		want bool
	}{
		{name: "webm", file: "video-id.webm", want: true},
		{name: "uppercase mp4", file: "video-id.MP4", want: true},
		{name: "canonical mp4", file: "video-id.mp4", want: true},
		{name: "less common video container", file: "video-id.m2ts", want: true},
		{name: "format fragment", file: "video-id.f137.mp4", want: false},
		{name: "partial", file: "video-id.webm.part", want: false},
		{name: "subtitle", file: "video-id.vtt", want: false},
		{name: "internet shortcut", file: "video-id.url", want: false},
		{name: "avif thumbnail", file: "video-id.avif", want: false},
		{name: "different ID", file: "other-id.webm", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLegacyVideo(tt.file, "video-id"); got != tt.want {
				t.Errorf("isLegacyVideo() = %v, want %v", got, tt.want)
			}
		})
	}
}
