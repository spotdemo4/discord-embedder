package video

import (
	"context"
	"errors"
	"os/exec"
	"strings"
)

// Codec returns the codec of the video.
func (v *Video) Codec(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx,
		"ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=codec_name",
		"-of", "default=noprint_wrappers=1:nokey=1",
		v.Path,
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
		v.Path,
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
