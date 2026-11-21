package video

import (
	"context"
	"errors"
	"os/exec"
	"strings"
)

// Codec returns the codec of the video.
func (v *Video) Codec(ctx context.Context) (string, error) {
	return probeCodec(ctx, v.Path, "v:0")
}

func probeCodec(ctx context.Context, path string, stream string) (string, error) {
	cmd := exec.CommandContext(ctx,
		"ffprobe",
		"-v", "error",
		"-select_streams", stream,
		"-show_entries", "stream=codec_name",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	)

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return parseCodecProbe(string(out)), nil
}

func parseCodecProbe(output string) string {
	return strings.TrimSpace(output)
}

// startsWithKeyframe reports whether the first displayed frame of the first video stream is a keyframe.
//
// Some sources, such as X, place the keyframe just before the start of the MP4 edit list, so the first
// displayed frame depends on a hidden keyframe. Browsers decode it anyway, but stricter players like
// Discord's skip it and show a frozen picture until the next keyframe while the audio keeps playing.
func startsWithKeyframe(ctx context.Context, path string) (bool, error) {
	cmd := exec.CommandContext(ctx,
		"ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		// Enough packets to get the first frame out of the decoder despite frame reordering
		"-read_intervals", "%+#16",
		"-show_entries", "frame=key_frame",
		"-of", "csv=p=0",
		path,
	)

	out, err := cmd.Output()
	if err != nil {
		return false, err
	}

	return parseKeyframeProbe(string(out)), nil
}

// parseKeyframeProbe reports whether the first frame in ffprobe's key_frame CSV output is a keyframe.
func parseKeyframeProbe(output string) bool {
	line, _, _ := strings.Cut(strings.TrimSpace(output), "\n")
	keyFrame, _, _ := strings.Cut(line, ",")

	return strings.TrimSpace(keyFrame) == "1"
}

// OriginalURL returns the sanitized URL used to download the video.
func (v *Video) OriginalURL(ctx context.Context) (string, error) {
	if v.originalURL != "" {
		return v.originalURL, nil
	}

	originalURL, err := probeFormatTag(ctx, v.Path, originalURLMetadataKey)
	if err != nil || originalURL == "" {
		return originalURL, err
	}

	originalURL, err = sanitizeOriginalURL(originalURL)
	if err != nil {
		return "", err
	}
	v.originalURL = originalURL
	return originalURL, nil
}

func probeFormatTag(ctx context.Context, path string, tag string) (string, error) {
	cmd := exec.CommandContext(ctx,
		"ffprobe",
		"-v", "error",
		"-show_entries", "format_tags="+tag,
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	)

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

// HasAudio returns whether the video has an audio stream.
func (v *Video) HasAudio(ctx context.Context) (bool, error) {
	return hasAudio(ctx, v.Path)
}

func hasAudio(ctx context.Context, path string) (bool, error) {
	codec, err := probeCodec(ctx, path, "a:0")
	if err != nil {
		return false, err
	}

	return codec != "", nil
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
