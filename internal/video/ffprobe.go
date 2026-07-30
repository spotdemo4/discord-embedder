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

// HasAudio returns whether the video has an audio stream.
func (v *Video) HasAudio(ctx context.Context) (bool, error) {
	return hasAudio(ctx, v.Path)
}

func hasAudio(ctx context.Context, path string) (bool, error) {
	cmd := exec.CommandContext(ctx,
		"ffprobe",
		"-v", "error",
		"-select_streams", "a:0",
		"-show_entries", "stream=codec_name",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	)

	out, err := cmd.Output()
	if err != nil {
		return false, err
	}

	return parseAudioProbe(string(out)), nil
}

func parseAudioProbe(output string) bool {
	return strings.TrimSpace(output) != ""
}

func requireAudio(ctx context.Context, path string) error {
	hasAudio, err := hasAudio(ctx, path)
	if err != nil {
		return err
	}
	if !hasAudio {
		return errors.New("media contains no audio stream")
	}

	return nil
}

// Resolution returns the width and height of the video.
func (v *Video) Resolution(ctx context.Context) (width string, height string, err error) {
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
