package video

import (
	"context"
	"discord-embedder/internal/app"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func Download(ctx context.Context, a *app.App, downloadURL string) (*Video, error) {
	link, err := url.Parse(downloadURL)
	if err != nil {
		return nil, err
	}

	id := uuid.New().String()

	// Creds
	var cmd *exec.Cmd
	domain := strings.TrimPrefix(link.Hostname(), "www.")
	username, password := creds(a, domain)
	if username != "" && password != "" {
		a.Logger.InfoContext(ctx, "downloading", "credentials", true)

		// Download video using credentials
		cmd = exec.CommandContext(ctx,
			"yt-dlp",
			"-o", fmt.Sprintf("%s.%%(ext)s", filepath.Join(a.FilesDir, id)),
			"--username", username,
			"--password", password,
			link.String(),
		)
	} else {
		a.Logger.InfoContext(ctx, "downloading", "credentials", false)

		// Download video without credentials
		cmd = exec.CommandContext(ctx,
			"yt-dlp",
			"-o", fmt.Sprintf("%s.%%(ext)s", filepath.Join(a.FilesDir, id)),
			link.String(),
		)
	}

	// Start yt-dlp
	if err = cmd.Run(); err != nil {
		return nil, err
	}

	// Get downloaded video
	var v *Video
	v, err = Get(a, id)
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

func creds(a *app.App, domain string) (string, string) {
	switch domain {
	case "reddit.com":
		return a.RedditUsername, a.RedditPassword
	case "tiktok.com":
		return a.TikTokUsername, a.TikTokPassword
	case "instagram.com":
		return a.InstagramUsername, a.InstagramPassword
	case "x.com":
		return a.XUsername, a.XPassword
	default:
		return "", ""
	}
}
