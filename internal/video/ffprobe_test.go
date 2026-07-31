package video

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestParseCodecProbe(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{name: "h264", output: "h264\n", want: "h264"},
		{name: "aac with carriage return", output: "aac\r\n", want: "aac"},
		{name: "surrounding whitespace", output: " \tflac \n", want: "flac"},
		{name: "empty", output: "", want: ""},
		{name: "whitespace only", output: " \r\n\t", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseCodecProbe(tt.output); got != tt.want {
				t.Errorf("parseCodecProbe() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCodecProbesStreams(t *testing.T) {
	path := filepath.Join(t.TempDir(), "video with spaces.mp4")
	writeFakeCommand(t, "ffprobe", `
last=""
for arg in "$@"; do
  last="$arg"
done
if [ "$last" != "$FAKE_VIDEO_PATH" ]; then
  exit 2
fi
case " $* " in
  *" -select_streams v:0 "*) printf ' h264\n' ;;
  *" -select_streams a:0 "*) printf 'aac\n' ;;
  *) exit 3 ;;
esac
`)
	t.Setenv("FAKE_VIDEO_PATH", path)

	video := &Video{File: File{Path: path}}
	codec, err := video.Codec(context.Background())
	if err != nil {
		t.Fatalf("Codec() error = %v", err)
	}
	if codec != "h264" {
		t.Errorf("Codec() = %q, want %q", codec, "h264")
	}

	hasAudio, err := video.HasAudio(context.Background())
	if err != nil {
		t.Fatalf("HasAudio() error = %v", err)
	}
	if !hasAudio {
		t.Error("HasAudio() = false, want true")
	}
}

func TestCodecProbeFailure(t *testing.T) {
	writeFakeCommand(t, "ffprobe", "exit 1")

	video := &Video{File: File{Path: "/videos/input.mp4"}}
	if _, err := video.Codec(context.Background()); err == nil {
		t.Fatal("Codec() expected error")
	}
	if _, err := video.HasAudio(context.Background()); err == nil {
		t.Fatal("HasAudio() expected error")
	}
}

func TestOriginalURL(t *testing.T) {
	t.Run("cached", func(t *testing.T) {
		writeFakeCommand(t, "ffprobe", "exit 1")
		video := &Video{originalURL: "https://example.com/video"}

		got, err := video.OriginalURL(context.Background())
		if err != nil {
			t.Fatalf("OriginalURL() error = %v", err)
		}
		if got != "https://example.com/video" {
			t.Errorf("OriginalURL() = %q, want %q", got, "https://example.com/video")
		}
	})

	t.Run("probes and sanitizes metadata", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "video with spaces.mp4")
		writeFakeCommand(t, "ffprobe", `
last=""
for arg in "$@"; do
  last="$arg"
done
if [ "$last" != "$FAKE_VIDEO_PATH" ]; then
  exit 2
fi
case " $* " in
  *" -show_entries format_tags=original_url "*) ;;
  *) exit 3 ;;
esac
printf '%s\n' 'https://x.com/user/status/123?s=20&t=tracking#fragment'
`)
		t.Setenv("FAKE_VIDEO_PATH", path)
		video := &Video{File: File{Path: path}}

		got, err := video.OriginalURL(context.Background())
		if err != nil {
			t.Fatalf("OriginalURL() error = %v", err)
		}
		if got != "https://x.com/user/status/123" {
			t.Errorf("OriginalURL() = %q, want %q", got, "https://x.com/user/status/123")
		}
	})

	t.Run("missing metadata", func(t *testing.T) {
		writeFakeCommand(t, "ffprobe", "exit 0")
		video := &Video{File: File{Path: "/videos/input.mp4"}}

		got, err := video.OriginalURL(context.Background())
		if err != nil {
			t.Fatalf("OriginalURL() error = %v", err)
		}
		if got != "" {
			t.Errorf("OriginalURL() = %q, want empty", got)
		}
	})

	t.Run("rejects unsafe metadata", func(t *testing.T) {
		writeFakeCommand(t, "ffprobe", "printf '%s\\n' 'javascript:alert(1)'")
		video := &Video{File: File{Path: "/videos/input.mp4"}}

		if _, err := video.OriginalURL(context.Background()); err == nil {
			t.Fatal("OriginalURL() expected error")
		}
	})

	t.Run("probe failure", func(t *testing.T) {
		writeFakeCommand(t, "ffprobe", "exit 1")
		video := &Video{File: File{Path: "/videos/input.mp4"}}

		if _, err := video.OriginalURL(context.Background()); err == nil {
			t.Fatal("OriginalURL() expected error")
		}
	})
}

func writeFakeCommand(t *testing.T, name string, body string) {
	t.Helper()

	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not available")
	}
	binDir := t.TempDir()
	script := fmt.Sprintf("#!%s\nset -eu\n%s", bash, body)
	if err = os.WriteFile(filepath.Join(binDir, name), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
