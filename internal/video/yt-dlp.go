package video

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"trev.zip/llc/discord-embedder/internal/config"
	"trev.zip/llc/discord-embedder/internal/logger"

	"github.com/google/uuid"
)

// ErrLoginRequired is returned when yt-dlp needs cookies or credentials to download a video.
var ErrLoginRequired = errors.New("login required")

// Download downloads a video, authenticating with the Netscape formatted cookies if they are not empty.
func Download(ctx context.Context, downloadURL string, cookies []byte) (*Video, error) {
	log := logger.FromContext(ctx)
	cfg := config.FromContext(ctx)

	// Parse URL
	link, err := url.Parse(downloadURL)
	if err != nil {
		return nil, err
	}
	originalURL, err := sanitizeOriginalURL(downloadURL)
	if err != nil {
		return nil, err
	}

	// Generate unique ID for video
	id := uuid.New().String()
	complete := false
	defer func() {
		if complete {
			return
		}
		if cleanupErr := cleanupDownloadArtifacts(cfg.FilesDir, id); cleanupErr != nil {
			log.WarnContext(ctx, "could not clean failed download", "error", cleanupErr)
		}
	}()

	// Download video using yt-dlp
	path, err := ytdlp(ctx, link, downloadURL, id, cookies)
	if err != nil {
		return nil, err
	}

	log.DebugContext(ctx, "yt-dlp", "path", path)

	v := &Video{
		ID:          id,
		originalURL: originalURL,
		File: File{
			Name: filepath.Base(path),
			Path: path,
		},
	}

	if err = requireAudio(ctx, v.Path); err != nil {
		return nil, errors.Join(errors.New("downloaded media must contain audio"), err)
	}

	// Generate thumbnail
	if err = v.thumbnail(ctx); err != nil {
		return nil, errors.Join(errors.New("could not generate thumbnail"), err)
	}

	complete = true
	return v, nil
}

type auth struct {
	cookies  string
	username string
	password string
}

func ytdlp(ctx context.Context, link *url.URL, downloadURL string, id string, cookies []byte) (string, error) {
	log := logger.FromContext(ctx)
	cfg := config.FromContext(ctx)

	// Try saved cookies, then configured credentials, then without authentication
	var auths []auth
	if len(cookies) > 0 {
		// yt-dlp only reads cookies from a file, so keep the decrypted copy for as short as possible
		cookiesPath, err := writeCookies(cfg.TempDir, id, cookies)
		if err != nil {
			return "", errors.Join(errors.New("could not write cookies"), err)
		}
		defer func() {
			if removeErr := os.Remove(cookiesPath); removeErr != nil {
				log.WarnContext(ctx, "could not remove cookies file", "error", removeErr)
			}
		}()

		auths = append(auths, auth{cookies: cookiesPath})
	}
	domain := strings.TrimPrefix(link.Hostname(), "www.")
	if username, password := creds(ctx, domain); username != "" && password != "" {
		auths = append(auths, auth{username: username, password: password})
	}
	auths = append(auths, auth{})

	var err error
	for index, a := range auths {
		log.InfoContext(ctx, "downloading", "cookies", a.cookies != "", "credentials", a.username != "")

		var path string
		path, err = runYTDLP(ctx, ytdlpArgs(cfg.FilesDir, id, downloadURL, a), cfg.FilesDir, id)
		if err == nil {
			return path, nil
		}

		if cleanupErr := cleanupDownloadArtifacts(cfg.FilesDir, id); cleanupErr != nil {
			return "", errors.Join(err, cleanupErr)
		}
		if index < len(auths)-1 {
			log.WarnContext(ctx, "could not download, trying next authentication method", "error", err)
		}
	}

	return "", err
}

func ytdlpArgs(filesDir string, id string, downloadURL string, a auth) []string {
	args := []string{
		"--ignore-config",
		"--no-playlist",
		"--no-simulate",
		"--format", "bv[vcodec~='^(h264|avc)']+ba/b[vcodec~='^(h264|avc)']/bv+ba/b",
		"--output", fmt.Sprintf("%s.%%(ext)s", filepath.Join(filesDir, id)),
		"--print", "after_move:filepath",
	}
	if a.cookies != "" {
		args = append(args, "--cookies", a.cookies)
	}
	if a.username != "" && a.password != "" {
		args = append(args, "--username", a.username, "--password", a.password)
	}

	return append(args, downloadURL)
}

func runYTDLP(ctx context.Context, args []string, filesDir string, id string) (string, error) {
	out, err := exec.CommandContext(ctx, "yt-dlp", args...).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && loginRequired(string(exitErr.Stderr)) {
			return "", fmt.Errorf("yt-dlp failed: %w: %w", ErrLoginRequired, err)
		}
		return "", fmt.Errorf("yt-dlp failed: %w", err)
	}

	path, err := parseFinalPath(string(out))
	if err != nil {
		return "", err
	}

	return validateFinalPath(filesDir, id, path)
}

// loginRequired reports whether yt-dlp failed because it needs cookies or credentials.
func loginRequired(stderr string) bool {
	for line := range strings.Lines(stderr) {
		if strings.HasPrefix(line, "ERROR:") && strings.Contains(line, "--cookies") {
			return true
		}
	}

	return false
}

func writeCookies(tempDir string, id string, cookies []byte) (string, error) {
	path := filepath.Join(tempDir, id+".cookies.txt")
	if err := os.WriteFile(path, cookies, 0600); err != nil {
		return "", err
	}

	return path, nil
}

func parseFinalPath(output string) (string, error) {
	var paths []string
	for line := range strings.Lines(output) {
		line = strings.TrimSpace(line)
		if line != "" {
			paths = append(paths, line)
		}
	}

	if len(paths) != 1 {
		return "", fmt.Errorf("yt-dlp returned %d final paths", len(paths))
	}

	return paths[0], nil
}

func validateFinalPath(filesDir string, id string, path string) (string, error) {
	filesDir, err := filepath.Abs(filesDir)
	if err != nil {
		return "", err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(filesDir, path)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("yt-dlp final path is outside files directory")
	}

	name := filepath.Base(path)
	if strings.TrimSuffix(name, filepath.Ext(name)) != id {
		return "", errors.New("yt-dlp final path does not match download ID")
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("yt-dlp final path is not a regular file")
	}

	return path, nil
}

func cleanupDownloadArtifacts(filesDir string, id string) error {
	entries, err := os.ReadDir(filesDir)
	if err != nil {
		return err
	}

	var cleanupErr error
	for _, entry := range entries {
		if entry.IsDir() || !isDownloadArtifact(entry.Name(), id) {
			continue
		}
		if err = os.Remove(filepath.Join(filesDir, entry.Name())); err != nil {
			cleanupErr = errors.Join(cleanupErr, err)
		}
	}

	return cleanupErr
}

func isDownloadArtifact(name string, id string) bool {
	return strings.HasPrefix(name, id+".")
}

func creds(ctx context.Context, domain string) (username string, password string) {
	cfg := config.FromContext(ctx)

	switch domain {
	case "reddit.com":
		return cfg.RedditUsername, cfg.RedditPassword
	case "tiktok.com":
		return cfg.TikTokUsername, cfg.TikTokPassword
	case "instagram.com":
		return cfg.InstagramUsername, cfg.InstagramPassword
	case "x.com":
		return cfg.XUsername, cfg.XPassword
	default:
		return "", ""
	}
}
