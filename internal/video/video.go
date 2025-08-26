package video

import (
	"context"
	"discord-embedder/internal/app"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type Video struct {
	*app.App

	ID           string
	Name         string
	AbsolutePath string
}

func Download(ctx context.Context, downloadURL string, filesDir string) (*Video, error) {
	link, err := url.Parse(downloadURL)
	if err != nil {
		return nil, err
	}

	domain := strings.TrimPrefix(link.Hostname(), "www.")
	id := uuid.New().String()

	// Creds
	username := ""
	password := ""
	switch domain {
	case "reddit.com":
		username = os.Getenv("REDDIT_USERNAME")
		password = os.Getenv("REDDIT_PASSWORD")
	case "tiktok.com":
		username = os.Getenv("TIKTOK_USERNAME")
		password = os.Getenv("TIKTOK_PASSWORD")
	case "instagram.com":
		username = os.Getenv("INSTAGRAM_USERNAME")
		password = os.Getenv("INSTAGRAM_PASSWORD")
	case "x.com":
		username = os.Getenv("X_USERNAME")
		password = os.Getenv("X_PASSWORD")
	}

	// Download video with creds
	if username != "" && password != "" {
		cmd := exec.CommandContext(ctx,
			"yt-dlp",
			"-o", fmt.Sprintf("%s.%%(ext)s", filepath.Join(filesDir, id)),
			"--username", username,
			"--password", password,
			link.String(),
		)
		if err = cmd.Run(); err == nil {
			return Get(id, filesDir)
		}
	}

	// Download video without creds
	cmd := exec.CommandContext(ctx,
		"yt-dlp",
		"-o", fmt.Sprintf("%s.%%(ext)s", filepath.Join(filesDir, id)),
		link.String(),
	)

	if err = cmd.Run(); err != nil {
		return nil, err
	}

	return Get(id, filesDir)
}

func Get(id string, filesDir string) (*Video, error) {
	// Find video file
	var name string
	var absolutePath string
	err := filepath.WalkDir(filesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasPrefix(strings.TrimPrefix(d.Name(), "SPOILER_"), id) {
			return nil
		}
		if strings.HasSuffix(d.Name(), "jpeg") {
			return nil
		}

		name = d.Name()
		absolutePath = filepath.Join(path, d.Name())

		return fs.SkipAll
	})
	if err != nil {
		return nil, err
	}
	if name == "" || absolutePath == "" {
		return nil, errors.New("could not find video file")
	}

	// Return video
	return &Video{
		ID:           id,
		Name:         name,
		AbsolutePath: absolutePath,
	}, nil
}

// Compress converts and compresses the video.
func (v *Video) Compress(ctx context.Context, quicksync bool) error {
	id := uuid.New().String()
	newPath := filepath.Join(v.FilesDir, fmt.Sprintf("%s.mp4", id))

	var cmd *exec.Cmd
	if quicksync {
		cmd = exec.CommandContext(ctx, "ffmpeg",
			"-hwaccel", "qsv",
			"-hwaccel_output_format", "qsv",
			"-i", v.AbsolutePath,
			"-c:v:0", "h264_qsv",
			"-global_quality:v:0", "23",
			"-c:a", "aac",
			newPath,
		)
	} else {
		cmd = exec.CommandContext(ctx, "ffmpeg",
			"-i", v.AbsolutePath,
			"-c:v:0", "libx264",
			"-global_quality:v:0", "23",
			"-c:a", "aac",
			newPath,
		)
	}
	if err := cmd.Run(); err != nil {
		return err
	}

	// Delete original video
	if err := v.delete(); err != nil {
		return err
	}

	// Get new video
	v.ID = id
	v.Name = fmt.Sprintf("%s.mp4", id)
	v.AbsolutePath = newPath

	return nil
}

// Trim video to start and end time.
func (v *Video) Trim(ctx context.Context, start string, end string) error {
	id := uuid.New().String()
	newPath := filepath.Join(v.FilesDir, fmt.Sprintf("%s.mp4", id))

	cmd := exec.CommandContext(ctx, "ffmpeg", "-ss", start, "-to", end, "-i", v.AbsolutePath, newPath)

	if err := cmd.Run(); err != nil {
		return err
	}

	// Delete original video
	if err := v.delete(); err != nil {
		return err
	}

	// Set new video
	v.ID = id
	v.Name = fmt.Sprintf("%s.mp4", id)
	v.AbsolutePath = newPath

	return nil
}

// Thumbnail generates a thumbnail for video.
func (v *Video) Thumbnail(ctx context.Context) error {
	imagePath := filepath.Join(v.FilesDir, fmt.Sprintf("%s.jpeg", v.ID))

	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", v.AbsolutePath, "-vframes", "1", imagePath)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

// Codec returns the codec of the video.
func (v *Video) Codec(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx,
		"ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=codec_name",
		"-of", "default=noprint_wrappers=1:nokey=1",
		v.AbsolutePath,
	)

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(out), nil
}

// Resolution returns the width and height of the video.
func (v *Video) Resolution(ctx context.Context) (string, string, error) {
	cmd := exec.CommandContext(ctx,
		"ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=s=x:p=0",
		v.AbsolutePath,
	)

	out, err := cmd.Output()
	if err != nil {
		return "", "", err
	}

	resolution := strings.Split(string(out), "x")
	if len(resolution) != 2 {
		return "", "", errors.New("could not get resolution")
	}

	return resolution[0], resolution[1], nil
}

// delete deletes the video file.
func (v *Video) delete() error {
	if err := os.Remove(v.AbsolutePath); err != nil {
		return err
	}

	return nil
}
