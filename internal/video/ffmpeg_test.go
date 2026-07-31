package video

import (
	"context"
	"discord-embedder/internal/config"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCompressionArgs(t *testing.T) {
	tests := []struct {
		name       string
		videoCodec string
		audioCodec string
		quicksync  bool
		want       []string
	}{
		{
			name:       "copy h264 and aac",
			videoCodec: "h264",
			audioCodec: "aac",
			want: []string{
				"-y",
				"-i", "/videos/input.webm",
				"-map", "0:v:0",
				"-map", "0:a:0",
				"-sn",
				"-dn",
				"-c:v:0", "copy",
				"-c:a:0", "copy",
				"-movflags", "+faststart",
				"-hide_banner",
				"-loglevel", "error",
				"/tmp/output.mp4",
			},
		},
		{
			name:       "copy h264 and transcode opus with quicksync enabled",
			videoCodec: "h264",
			audioCodec: "opus",
			quicksync:  true,
			want: []string{
				"-y",
				"-i", "/videos/input.webm",
				"-map", "0:v:0",
				"-map", "0:a:0",
				"-sn",
				"-dn",
				"-c:v:0", "copy",
				"-c:a:0", "aac",
				"-movflags", "+faststart",
				"-hide_banner",
				"-loglevel", "error",
				"/tmp/output.mp4",
			},
		},
		{
			name:       "transcode vp9 and copy aac",
			videoCodec: "vp9",
			audioCodec: "aac",
			want: []string{
				"-y",
				"-i", "/videos/input.webm",
				"-map", "0:v:0",
				"-map", "0:a:0",
				"-sn",
				"-dn",
				"-c:v:0", "libx264",
				"-crf:v:0", "23",
				"-c:a:0", "copy",
				"-movflags", "+faststart",
				"-hide_banner",
				"-loglevel", "error",
				"/tmp/output.mp4",
			},
		},
		{
			name:       "transcode vp9 and opus",
			videoCodec: "vp9",
			audioCodec: "opus",
			want: []string{
				"-y",
				"-i", "/videos/input.webm",
				"-map", "0:v:0",
				"-map", "0:a:0",
				"-sn",
				"-dn",
				"-c:v:0", "libx264",
				"-crf:v:0", "23",
				"-c:a:0", "aac",
				"-movflags", "+faststart",
				"-hide_banner",
				"-loglevel", "error",
				"/tmp/output.mp4",
			},
		},
		{
			name:       "transcode vp9 with quicksync and copy aac",
			videoCodec: "vp9",
			audioCodec: "aac",
			quicksync:  true,
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
				"-c:a:0", "copy",
				"-movflags", "+faststart",
				"-hide_banner",
				"-loglevel", "error",
				"/tmp/output.mp4",
			},
		},
		{
			name:       "transcode vp9 and opus with quicksync",
			videoCodec: "vp9",
			audioCodec: "opus",
			quicksync:  true,
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
			got := compressionArgs(
				"/videos/input.webm",
				"/tmp/output.mp4",
				tt.videoCodec,
				tt.audioCodec,
				tt.quicksync,
			)
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

func TestCompressCopiesCompatibleStreams(t *testing.T) {
	filesDir := t.TempDir()
	tempDir := t.TempDir()
	id := "video-id"
	sourcePath := filepath.Join(filesDir, id+".mkv")
	if err := os.WriteFile(sourcePath, []byte("source"), 0600); err != nil {
		t.Fatal(err)
	}
	argsPath := filepath.Join(t.TempDir(), "ffmpeg-args")

	writeFakeCommand(t, "ffprobe", `
case " $* " in
  *" -select_streams v:0 "*) printf 'h264\n' ;;
  *" -select_streams a:0 "*) printf 'aac\n' ;;
  *) exit 2 ;;
esac
`)
	writeFakeCommand(t, "ffmpeg", `
last=""
for arg in "$@"; do
  printf '%s\n' "$arg" >> "$FAKE_FFMPEG_ARGS"
  last="$arg"
done
printf encoded > "$last"
`)
	t.Setenv("FAKE_FFMPEG_ARGS", argsPath)

	ctx := config.WithConfig(context.Background(), &config.Config{
		FilesDir: filesDir,
		TempDir:  tempDir,
	})
	video := &Video{ID: id, File: File{Name: filepath.Base(sourcePath), Path: sourcePath}}
	if err := video.Compress(ctx); err != nil {
		t.Fatalf("Compress() error = %v", err)
	}

	finalPath := filepath.Join(filesDir, id+".mp4")
	if video.Name != id+".mp4" || video.Path != finalPath {
		t.Errorf("Video = name %q path %q, want name %q path %q", video.Name, video.Path, id+".mp4", finalPath)
	}
	if _, err := os.Stat(finalPath); err != nil {
		t.Errorf("final video missing: %v", err)
	}
	if _, err := os.Stat(sourcePath); !os.IsNotExist(err) {
		t.Errorf("source still exists, error = %v", err)
	}

	args, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	command := string(args)
	if !strings.Contains(command, "-c:v:0\ncopy\n") {
		t.Errorf("ffmpeg args do not copy video: %q", command)
	}
	if !strings.Contains(command, "-c:a:0\ncopy\n") {
		t.Errorf("ffmpeg args do not copy audio: %q", command)
	}
}

func TestCompressProbeFailurePreservesSource(t *testing.T) {
	filesDir := t.TempDir()
	tempDir := t.TempDir()
	id := "video-id"
	sourcePath := filepath.Join(filesDir, id+".webm")
	if err := os.WriteFile(sourcePath, []byte("source"), 0600); err != nil {
		t.Fatal(err)
	}
	ffmpegCalled := filepath.Join(t.TempDir(), "ffmpeg-called")

	writeFakeCommand(t, "ffprobe", "exit 1")
	writeFakeCommand(t, "ffmpeg", `printf called > "$FAKE_FFMPEG_CALLED"`)
	t.Setenv("FAKE_FFMPEG_CALLED", ffmpegCalled)

	ctx := config.WithConfig(context.Background(), &config.Config{
		FilesDir: filesDir,
		TempDir:  tempDir,
	})
	video := &Video{ID: id, File: File{Name: filepath.Base(sourcePath), Path: sourcePath}}
	if err := video.Compress(ctx); err == nil {
		t.Fatal("Compress() expected error")
	}
	if _, err := os.Stat(ffmpegCalled); !os.IsNotExist(err) {
		t.Errorf("ffmpeg was called, error = %v", err)
	}
	if _, err := os.Stat(sourcePath); err != nil {
		t.Errorf("source was removed: %v", err)
	}
	if video.Path != sourcePath || video.Name != filepath.Base(sourcePath) {
		t.Errorf("Video changed after failure: name %q path %q", video.Name, video.Path)
	}
}

func TestCompressMissingAudioPreservesSource(t *testing.T) {
	filesDir := t.TempDir()
	tempDir := t.TempDir()
	id := "video-id"
	sourcePath := filepath.Join(filesDir, id+".mp4")
	if err := os.WriteFile(sourcePath, []byte("source"), 0600); err != nil {
		t.Fatal(err)
	}
	ffmpegCalled := filepath.Join(t.TempDir(), "ffmpeg-called")

	writeFakeCommand(t, "ffprobe", `
case " $* " in
  *" -select_streams v:0 "*) printf 'h264\n' ;;
  *" -select_streams a:0 "*) exit 0 ;;
  *) exit 2 ;;
esac
`)
	writeFakeCommand(t, "ffmpeg", `printf called > "$FAKE_FFMPEG_CALLED"`)
	t.Setenv("FAKE_FFMPEG_CALLED", ffmpegCalled)

	ctx := config.WithConfig(context.Background(), &config.Config{
		FilesDir: filesDir,
		TempDir:  tempDir,
	})
	video := &Video{ID: id, File: File{Name: filepath.Base(sourcePath), Path: sourcePath}}
	err := video.Compress(ctx)
	if err == nil || !strings.Contains(err.Error(), "media contains no audio stream") {
		t.Fatalf("Compress() error = %v, want missing audio error", err)
	}
	if _, err = os.Stat(ffmpegCalled); !os.IsNotExist(err) {
		t.Errorf("ffmpeg was called, error = %v", err)
	}
	if _, err = os.Stat(sourcePath); err != nil {
		t.Errorf("source was removed: %v", err)
	}
}

func TestCompressFFmpegFailurePreservesSource(t *testing.T) {
	filesDir := t.TempDir()
	tempDir := t.TempDir()
	id := "video-id"
	sourcePath := filepath.Join(filesDir, id+".webm")
	tempPath := filepath.Join(tempDir, id+".encode.mp4")
	if err := os.WriteFile(sourcePath, []byte("source"), 0600); err != nil {
		t.Fatal(err)
	}

	writeFakeCommand(t, "ffprobe", `
case " $* " in
  *" -select_streams v:0 "*) printf 'vp9\n' ;;
  *" -select_streams a:0 "*) printf 'opus\n' ;;
  *) exit 2 ;;
esac
`)
	writeFakeCommand(t, "ffmpeg", `
last=""
for arg in "$@"; do
  last="$arg"
done
printf partial > "$last"
exit 1
`)

	ctx := config.WithConfig(context.Background(), &config.Config{
		FilesDir: filesDir,
		TempDir:  tempDir,
	})
	video := &Video{ID: id, File: File{Name: filepath.Base(sourcePath), Path: sourcePath}}
	if err := video.Compress(ctx); err == nil {
		t.Fatal("Compress() expected error")
	}
	if _, err := os.Stat(sourcePath); err != nil {
		t.Errorf("source was removed: %v", err)
	}
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Errorf("temporary output still exists, error = %v", err)
	}
	if video.Path != sourcePath || video.Name != filepath.Base(sourcePath) {
		t.Errorf("Video changed after failure: name %q path %q", video.Name, video.Path)
	}
}

func TestCompressRejectsOutputWithoutAudio(t *testing.T) {
	filesDir := t.TempDir()
	tempDir := t.TempDir()
	id := "video-id"
	sourcePath := filepath.Join(filesDir, id+".mkv")
	tempPath := filepath.Join(tempDir, id+".encode.mp4")
	if err := os.WriteFile(sourcePath, []byte("source"), 0600); err != nil {
		t.Fatal(err)
	}

	writeFakeCommand(t, "ffprobe", `
last=""
for arg in "$@"; do
  last="$arg"
done
case " $* " in
  *" -select_streams v:0 "*) printf 'h264\n' ;;
  *" -select_streams a:0 "*)
    if [ "$last" != "$FAKE_ENCODE_PATH" ]; then
      printf 'aac\n'
    fi
    ;;
  *) exit 2 ;;
esac
`)
	writeFakeCommand(t, "ffmpeg", `
last=""
for arg in "$@"; do
  last="$arg"
done
printf encoded > "$last"
`)
	t.Setenv("FAKE_ENCODE_PATH", tempPath)

	ctx := config.WithConfig(context.Background(), &config.Config{
		FilesDir: filesDir,
		TempDir:  tempDir,
	})
	video := &Video{ID: id, File: File{Name: filepath.Base(sourcePath), Path: sourcePath}}
	err := video.Compress(ctx)
	if err == nil || !strings.Contains(err.Error(), "compressed video must contain audio") {
		t.Fatalf("Compress() error = %v, want output audio error", err)
	}
	if _, err = os.Stat(sourcePath); err != nil {
		t.Errorf("source was removed: %v", err)
	}
	if _, err = os.Stat(tempPath); !os.IsNotExist(err) {
		t.Errorf("temporary output still exists, error = %v", err)
	}
	if _, err = os.Stat(filepath.Join(filesDir, id+".mp4")); !os.IsNotExist(err) {
		t.Errorf("canonical output exists, error = %v", err)
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
