package video

import (
	"context"
	"discord-embedder/internal/config"
	"discord-embedder/internal/logger"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func Download(ctx context.Context, downloadURL string) (*Video, error) {
	log := logger.FromContext(ctx)
	cfg := config.FromContext(ctx)

	// Parse URL
	link, err := url.Parse(downloadURL)
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
	path, err := ytdlp(ctx, link, id)
	if err != nil {
		return nil, err
	}

	log.DebugContext(ctx, "yt-dlp", "path", path)

	v := &Video{
		ID: id,
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

func ytdlp(ctx context.Context, link *url.URL, id string) (string, error) {
	log := logger.FromContext(ctx)
	cfg := config.FromContext(ctx)

	domain := strings.TrimPrefix(link.Hostname(), "www.")
	username, password := creds(ctx, domain)
	if username != "" && password != "" {
		log.InfoContext(ctx, "downloading", "credentials", true)

		path, err := runYTDLP(ctx, ytdlpArgs(cfg.FilesDir, id, link.String(), username, password), cfg.FilesDir, id)
		if err == nil {
			return path, nil
		}

		log.WarnContext(ctx, "could not download with credentials, trying without", "error", err)
		if cleanupErr := cleanupDownloadArtifacts(cfg.FilesDir, id); cleanupErr != nil {
			return "", errors.Join(errors.New("could not clean credentialed download artifacts"), cleanupErr)
		}
	}

	log.InfoContext(ctx, "downloading", "credentials", false)
	path, err := runYTDLP(ctx, ytdlpArgs(cfg.FilesDir, id, link.String(), "", ""), cfg.FilesDir, id)
	if err != nil {
		if cleanupErr := cleanupDownloadArtifacts(cfg.FilesDir, id); cleanupErr != nil {
			return "", errors.Join(err, cleanupErr)
		}
		return "", err
	}

	return path, nil
}

func ytdlpArgs(filesDir string, id string, downloadURL string, username string, password string) []string {
	args := []string{
		"--ignore-config",
		"--no-playlist",
		"--no-simulate",
		"--format", "bv+ba/b",
		"--output", fmt.Sprintf("%s.%%(ext)s", filepath.Join(filesDir, id)),
		"--print", "after_move:filepath",
	}
	if username != "" && password != "" {
		args = append(args, "--username", username, "--password", password)
	}

	return append(args, downloadURL)
}

func runYTDLP(ctx context.Context, args []string, filesDir string, id string) (string, error) {
	out, err := exec.CommandContext(ctx, "yt-dlp", args...).Output()
	if err != nil {
		return "", fmt.Errorf("yt-dlp failed: %w", err)
	}

	path, err := parseFinalPath(string(out))
	if err != nil {
		return "", err
	}

	return validateFinalPath(filesDir, id, path)
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
