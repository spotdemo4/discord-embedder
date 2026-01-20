package video

import (
	"context"
	"discord-embedder/internal/config"
	"discord-embedder/internal/logger"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func Download(ctx context.Context, downloadURL string) (*Video, error) {
	log := logger.FromContext(ctx)

	// Parse URL
	link, err := url.Parse(downloadURL)
	if err != nil {
		return nil, err
	}

	// Generate unique ID for video
	id := uuid.New().String()

	// Download video using yt-dlp
	out, err := ytdlp(ctx, link, id)
	if err != nil {
		return nil, err
	}

	log.DebugContext(ctx, "yt-dlp", "output", string(out))

	// Get downloaded video
	var v *Video
	v, err = Get(ctx, id)
	if err != nil {
		return nil, errors.Join(errors.New("could not get video after download"), err)
	}

	// Generate thumbnail
	err = v.thumbnail(ctx)
	if err != nil {
		return nil, errors.Join(errors.New("could not generate thumbnail"), err)
	}

	return v, nil
}

func ytdlp(ctx context.Context, link *url.URL, id string) ([]byte, error) {
	log := logger.FromContext(ctx)
	cfg := config.FromContext(ctx)

	domain := strings.TrimPrefix(link.Hostname(), "www.")
	username, password := creds(ctx, domain)
	if username != "" && password != "" {
		log.InfoContext(ctx, "downloading", "credentials", true)

		// Download video using credentials
		cmd := exec.CommandContext(ctx,
			"yt-dlp",
			"-o", fmt.Sprintf("%s.%%(ext)s", filepath.Join(cfg.FilesDir, id)),
			"--dump-json",
			"--no-simulate",
			"--username", username,
			"--password", password,
			link.String(),
		)

		out, err := cmd.Output()
		if err == nil {
			return out, nil
		}

		log.WarnContext(ctx, "could not download with credentials, trying without", "error", err)
	}

	log.InfoContext(ctx, "downloading", "credentials", false)

	// Download video without credentials
	cmd := exec.CommandContext(ctx,
		"yt-dlp",
		"-o", fmt.Sprintf("%s.%%(ext)s", filepath.Join(cfg.FilesDir, id)),
		"--dump-json",
		"--no-simulate",
		link.String(),
	)

	return cmd.Output()
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
