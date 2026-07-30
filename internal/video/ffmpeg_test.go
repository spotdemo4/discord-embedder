package video

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCompressionArgs(t *testing.T) {
	tests := []struct {
		name      string
		quicksync bool
		want      []string
	}{
		{
			name: "software",
			want: []string{
				"-y",
				"-i", "/videos/input.webm",
				"-map", "0:v:0",
				"-map", "0:a:0",
				"-sn",
				"-dn",
				"-c:v:0", "libx264",
				"-global_quality:v:0", "23",
				"-c:a:0", "aac",
				"-movflags", "+faststart",
				"-hide_banner",
				"-loglevel", "error",
				"/tmp/output.mp4",
			},
		},
		{
			name:      "quicksync",
			quicksync: true,
			want: []string{
				"-y",
				"-hwaccel", "qsv",
				"-hwaccel_output_format", "qsv",
				"-i", "/videos/input.webm",
				"-map", "0:v:0",
				"-map", "0:a:0",
				"-sn",
				"-dn",
				"-c:v:0", "h264_qsv",
				"-global_quality:v:0", "23",
				"-c:a:0", "aac",
				"-movflags", "+faststart",
				"-hide_banner",
				"-loglevel", "error",
				"/tmp/output.mp4",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compressionArgs("/videos/input.webm", "/tmp/output.mp4", tt.quicksync)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("compressionArgs() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCompressionOutputPaths(t *testing.T) {
	tempPath, finalName, finalPath := compressionOutputPaths("/tmp", "/videos", "video-id")
	if tempPath != filepath.Join("/tmp", "video-id.encode.mp4") {
		t.Errorf("tempPath = %q", tempPath)
	}
	if finalName != "video-id.mp4" {
		t.Errorf("finalName = %q", finalName)
	}
	if finalPath != filepath.Join("/videos", "video-id.mp4") {
		t.Errorf("finalPath = %q", finalPath)
	}
}

func TestTrimArgs(t *testing.T) {
	want := []string{
		"-y",
		"-ss", "00:00:01",
		"-to", "00:00:05",
		"-i", "/videos/input.mkv",
		"-map", "0:v:0",
		"-map", "0:a:0",
		"-sn",
		"-dn",
		"-hide_banner",
		"-loglevel", "error",
		"/tmp/output.mkv",
	}
	got := trimArgs("/videos/input.mkv", "/tmp/output.mkv", "00:00:01", "00:00:05")
	if !reflect.DeepEqual(got, want) {
		t.Errorf("trimArgs() = %q, want %q", got, want)
	}
}

func TestTrimOutputPath(t *testing.T) {
	tests := []struct {
		name       string
		sourceName string
		want       string
	}{
		{name: "webm", sourceName: "video-id.webm", want: filepath.Join("/tmp", "video-id.trim.webm")},
		{name: "mkv", sourceName: "video-id.mkv", want: filepath.Join("/tmp", "video-id.trim.mkv")},
		{name: "mp4", sourceName: "video-id.mp4", want: filepath.Join("/tmp", "video-id.trim.mp4")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trimOutputPath("/tmp", "video-id", tt.sourceName); got != tt.want {
				t.Errorf("trimOutputPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMove(t *testing.T) {
	t.Run("replace destination", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "source.mp4")
		dst := filepath.Join(dir, "destination.mp4")
		if err := os.WriteFile(src, []byte("new video"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, []byte("old video"), 0600); err != nil {
			t.Fatal(err)
		}

		if err := move(src, dst); err != nil {
			t.Fatalf("move() error = %v", err)
		}
		if _, err := os.Stat(src); !os.IsNotExist(err) {
			t.Errorf("source still exists, error = %v", err)
		}
		got, err := os.ReadFile(dst)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "new video" {
			t.Errorf("destination = %q, want %q", got, "new video")
		}
	})

	t.Run("relative and absolute aliases", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "video.mp4")
		if err := os.WriteFile(path, []byte("video"), 0600); err != nil {
			t.Fatal(err)
		}

		workingDir, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if err = os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chdir(workingDir) })

		if err = move("video.mp4", path); err != nil {
			t.Fatalf("move() error = %v", err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "video" {
			t.Errorf("file = %q, want %q", got, "video")
		}
	})

	t.Run("hard link aliases", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "video.mp4")
		alias := filepath.Join(dir, "video-alias.mp4")
		if err := os.WriteFile(path, []byte("video"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Link(path, alias); err != nil {
			t.Fatal(err)
		}

		if !sameFile(path, alias) {
			t.Error("sameFile() = false, want true")
		}
		if err := move(path, alias); err != nil {
			t.Fatalf("move() error = %v", err)
		}
		if _, err := os.Stat(path); err != nil {
			t.Errorf("source alias was removed: %v", err)
		}
	})
}
