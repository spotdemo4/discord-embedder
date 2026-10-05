package video

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"trev.zip/llc/discord-embedder/internal/config"
)

func TestYTDLPArgs(t *testing.T) {
	filesDir := "/videos"
	id := "video-id"
	downloadURL := "https://example.com/watch?v=1"

	tests := []struct {
		name string
		auth auth
		want []string
	}{
		{
			name: "without credentials",
			want: []string{
				"--ignore-config",
				"--no-playlist",
				"--no-simulate",
				"--format", "bv[vcodec~='^(h264|avc)']+ba/b[vcodec~='^(h264|avc)']/bv+ba/b",
				"--output", filepath.Join(filesDir, id) + ".%(ext)s",
				"--print", "after_move:filepath",
				downloadURL,
			},
		},
		{
			name: "with credentials",
			auth: auth{username: "user", password: "pass"},
			want: []string{
				"--ignore-config",
				"--no-playlist",
				"--no-simulate",
				"--format", "bv[vcodec~='^(h264|avc)']+ba/b[vcodec~='^(h264|avc)']/bv+ba/b",
				"--output", filepath.Join(filesDir, id) + ".%(ext)s",
				"--print", "after_move:filepath",
				"--username", "user",
				"--password", "pass",
				downloadURL,
			},
		},
		{
			name: "with cookies",
			auth: auth{cookies: "/tmp/cookies.txt"},
			want: []string{
				"--ignore-config",
				"--no-playlist",
				"--no-simulate",
				"--format", "bv[vcodec~='^(h264|avc)']+ba/b[vcodec~='^(h264|avc)']/bv+ba/b",
				"--output", filepath.Join(filesDir, id) + ".%(ext)s",
				"--print", "after_move:filepath",
				"--cookies", "/tmp/cookies.txt",
				downloadURL,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ytdlpArgs(filesDir, id, downloadURL, tt.auth)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ytdlpArgs() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestYTDLPPreservesDownloadURL(t *testing.T) {
	filesDir := t.TempDir()
	argsPath := filepath.Join(t.TempDir(), "download-url")
	id := "video-id"
	downloadURL := "https://example.com/watch?token=a%2Fb&token=c#fragment"
	writeFakeYTDLP(t, `
last=""
for arg in "$@"; do
  last="$arg"
done
printf '%s' "$last" > "$FAKE_ARGS_PATH"
printf video > "$FAKE_FILES_DIR/$FAKE_ID.webm"
printf '%s\n' "$FAKE_FILES_DIR/$FAKE_ID.webm"
`)
	t.Setenv("FAKE_ARGS_PATH", argsPath)
	t.Setenv("FAKE_FILES_DIR", filesDir)
	t.Setenv("FAKE_ID", id)

	link, err := url.Parse(downloadURL)
	if err != nil {
		t.Fatal(err)
	}
	ctx := config.WithConfig(context.Background(), &config.Config{FilesDir: filesDir})
	if _, err = ytdlp(ctx, link, downloadURL, id, nil); err != nil {
		t.Fatalf("ytdlp() error = %v", err)
	}

	got, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != downloadURL {
		t.Errorf("yt-dlp URL = %q, want %q", got, downloadURL)
	}
}

func TestParseFinalPath(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    string
		wantErr bool
	}{
		{
			name:   "line feed",
			output: "/videos/video-id.webm\n",
			want:   "/videos/video-id.webm",
		},
		{
			name:   "carriage return and line feed",
			output: "/videos/video-id.mp4\r\n",
			want:   "/videos/video-id.mp4",
		},
		{
			name:   "spaces in path",
			output: "/videos with spaces/video-id.mkv\n",
			want:   "/videos with spaces/video-id.mkv",
		},
		{
			name:    "empty output",
			output:  "\r\n",
			wantErr: true,
		},
		{
			name:    "multiple paths",
			output:  "/videos/one.mp4\n/videos/two.mp4\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFinalPath(tt.output)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseFinalPath() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("parseFinalPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateFinalPath(t *testing.T) {
	filesDir := t.TempDir()
	id := "video-id"
	path := filepath.Join(filesDir, id+".webm")
	if err := os.WriteFile(path, []byte("video"), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := validateFinalPath(filesDir, id, path)
	if err != nil {
		t.Fatalf("validateFinalPath() error = %v", err)
	}
	want, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("validateFinalPath() = %q, want %q", got, want)
	}

	if _, err = validateFinalPath(filesDir, "other-id", path); err == nil {
		t.Error("validateFinalPath() expected ID mismatch error")
	}

	outside := filepath.Join(t.TempDir(), id+".webm")
	if err = os.WriteFile(outside, []byte("video"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = validateFinalPath(filesDir, id, outside); err == nil {
		t.Error("validateFinalPath() expected outside directory error")
	}
}

func TestIsDownloadArtifact(t *testing.T) {
	id := "video-id"
	tests := []struct {
		name string
		file string
		want bool
	}{
		{name: "final file", file: "video-id.webm", want: true},
		{name: "format file", file: "video-id.f137.mp4", want: true},
		{name: "partial file", file: "video-id.webm.part", want: true},
		{name: "thumbnail", file: "video-id.jpeg", want: true},
		{name: "prefix only", file: "video-id-other.mp4", want: false},
		{name: "other ID", file: "other-video-id.mp4", want: false},
		{name: "bare ID", file: "video-id", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDownloadArtifact(tt.file, id); got != tt.want {
				t.Errorf("isDownloadArtifact() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestYTDLPCleansCredentialedAttemptBeforeRetry(t *testing.T) {
	filesDir := t.TempDir()
	id := "video-id"
	writeFakeYTDLP(t, `
case " $* " in
  *" --username "*)
    printf partial > "$FAKE_FILES_DIR/$FAKE_ID.webm.part"
    exit 1
    ;;
esac
if [ -e "$FAKE_FILES_DIR/$FAKE_ID.webm.part" ]; then
  exit 2
fi
printf video > "$FAKE_FILES_DIR/$FAKE_ID.webm"
printf '%s\n' "$FAKE_FILES_DIR/$FAKE_ID.webm"
`)
	t.Setenv("FAKE_FILES_DIR", filesDir)
	t.Setenv("FAKE_ID", id)

	ctx := config.WithConfig(context.Background(), &config.Config{
		FilesDir:       filesDir,
		RedditUsername: "user",
		RedditPassword: "pass",
	})
	link, err := url.Parse("https://reddit.com/video")
	if err != nil {
		t.Fatal(err)
	}

	got, err := ytdlp(ctx, link, link.String(), id, nil)
	if err != nil {
		t.Fatalf("ytdlp() error = %v", err)
	}
	want := filepath.Join(filesDir, id+".webm")
	if got != want {
		t.Errorf("ytdlp() = %q, want %q", got, want)
	}
	if _, err = os.Stat(filepath.Join(filesDir, id+".webm.part")); !os.IsNotExist(err) {
		t.Errorf("credentialed partial file still exists, error = %v", err)
	}
}

func TestYTDLPCleansArtifactsAfterFailure(t *testing.T) {
	filesDir := t.TempDir()
	id := "video-id"
	writeFakeYTDLP(t, `
printf partial > "$FAKE_FILES_DIR/$FAKE_ID.webm.part"
exit 1
`)
	t.Setenv("FAKE_FILES_DIR", filesDir)
	t.Setenv("FAKE_ID", id)

	ctx := config.WithConfig(context.Background(), &config.Config{FilesDir: filesDir})
	link, err := url.Parse("https://example.com/video")
	if err != nil {
		t.Fatal(err)
	}

	if _, err = ytdlp(ctx, link, link.String(), id, nil); err == nil {
		t.Fatal("ytdlp() expected error")
	}
	if _, err = os.Stat(filepath.Join(filesDir, id+".webm.part")); !os.IsNotExist(err) {
		t.Errorf("failed download artifact still exists, error = %v", err)
	}
}

func TestYTDLPTriesCookiesFirst(t *testing.T) {
	filesDir := t.TempDir()
	tempDir := t.TempDir()
	argsPath := filepath.Join(t.TempDir(), "args")
	cookiesCopy := filepath.Join(t.TempDir(), "cookies.txt")
	id := "video-id"
	writeFakeYTDLP(t, `
printf '%s\n' "$*" >> "$FAKE_ARGS_PATH"
case " $* " in
  *" --cookies "*)
    cookies="$(printf '%s\n' "$@" | grep -A1 -x -- --cookies | tail -n1)"
    cp "$cookies" "$FAKE_COOKIES_COPY"
    printf video > "$FAKE_FILES_DIR/$FAKE_ID.webm"
    printf '%s\n' "$FAKE_FILES_DIR/$FAKE_ID.webm"
    exit 0
    ;;
esac
exit 1
`)
	t.Setenv("FAKE_ARGS_PATH", argsPath)
	t.Setenv("FAKE_COOKIES_COPY", cookiesCopy)
	t.Setenv("FAKE_FILES_DIR", filesDir)
	t.Setenv("FAKE_ID", id)

	ctx := config.WithConfig(context.Background(), &config.Config{
		FilesDir:       filesDir,
		TempDir:        tempDir,
		TikTokUsername: "user",
		TikTokPassword: "pass",
	})
	link, err := url.Parse("https://www.tiktok.com/@user/video/1")
	if err != nil {
		t.Fatal(err)
	}

	cookies := []byte("# Netscape HTTP Cookie File\n.tiktok.com\tTRUE\t/\tTRUE\t0\tsessionid\tsecret\n")
	if _, err = ytdlp(ctx, link, link.String(), id, cookies); err != nil {
		t.Fatalf("ytdlp() error = %v", err)
	}

	args, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Count(string(args), "\n"); lines != 1 {
		t.Errorf("yt-dlp ran %d times, want 1", lines)
	}

	got, err := os.ReadFile(cookiesCopy)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(cookies) {
		t.Errorf("yt-dlp cookies = %q, want %q", got, cookies)
	}
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("temp dir has %d entries, want cookies file removed", len(entries))
	}
}

func TestYTDLPLoginRequired(t *testing.T) {
	filesDir := t.TempDir()
	id := "video-id"
	writeFakeYTDLP(t, `
printf '%s\n' "ERROR: [TikTok] 1: This post may not be comfortable for some audiences. Log in for access. Use --cookies-from-browser or --cookies for the authentication." >&2
exit 1
`)

	ctx := config.WithConfig(context.Background(), &config.Config{FilesDir: filesDir})
	link, err := url.Parse("https://www.tiktok.com/@user/video/1")
	if err != nil {
		t.Fatal(err)
	}

	_, err = ytdlp(ctx, link, link.String(), id, nil)
	if !errors.Is(err, ErrLoginRequired) {
		t.Errorf("ytdlp() error = %v, want ErrLoginRequired", err)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Errorf("ytdlp() error = %v, want exec.ExitError", err)
	}
}

func TestLoginRequired(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
		want   bool
	}{
		{
			name:   "tiktok",
			stderr: "ERROR: [TikTok] 1: This post may not be comfortable for some audiences. Log in for access. Use --cookies-from-browser or --cookies for the authentication.\n",
			want:   true,
		},
		{
			name:   "youtube bot check",
			stderr: "WARNING: something\nERROR: [youtube] 1: Sign in to confirm you're not a bot. Use --cookies-from-browser or --cookies for the authentication.\n",
			want:   true,
		},
		{
			name:   "warning only",
			stderr: "WARNING: Use --cookies for more formats\nERROR: Unsupported URL: https://example.com\n",
			want:   false,
		},
		{
			name:   "other error",
			stderr: "ERROR: Unsupported URL: https://example.com\n",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := loginRequired(tt.stderr); got != tt.want {
				t.Errorf("loginRequired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func writeFakeYTDLP(t *testing.T, body string) {
	t.Helper()

	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not available")
	}
	binDir := t.TempDir()
	script := fmt.Sprintf("#!%s\nset -eu\n%s", bash, body)
	if err = os.WriteFile(filepath.Join(binDir, "yt-dlp"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
