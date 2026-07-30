package version

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

var Application = "dev"

func YTDLP(ctx context.Context) (string, error) {
	output, err := exec.CommandContext(ctx, "yt-dlp", "--version").Output()
	if err != nil {
		return "", fmt.Errorf("get yt-dlp version: %w", err)
	}

	version, err := firstLine(string(output))
	if err != nil {
		return "", fmt.Errorf("get yt-dlp version: %w", err)
	}

	return version, nil
}

func FFmpeg(ctx context.Context) (string, error) {
	output, err := exec.CommandContext(ctx, "ffmpeg", "-version").Output()
	if err != nil {
		return "", fmt.Errorf("get ffmpeg version: %w", err)
	}

	return parseFFmpeg(string(output))
}

func parseFFmpeg(output string) (string, error) {
	line, err := firstLine(output)
	if err != nil {
		return "", fmt.Errorf("get ffmpeg version: %w", err)
	}

	const prefix = "ffmpeg version"
	if line == prefix {
		return "", errors.New("get ffmpeg version: empty version")
	}
	if !strings.HasPrefix(line, prefix+" ") {
		return line, nil
	}

	fields := strings.Fields(strings.TrimPrefix(line, prefix))
	if len(fields) == 0 {
		return "", errors.New("get ffmpeg version: empty version")
	}

	return fields[0], nil
}

func firstLine(output string) (string, error) {
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line, nil
		}
	}

	return "", errors.New("empty output")
}
