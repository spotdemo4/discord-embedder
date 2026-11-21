package video

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"trev.zip/llc/discord-embedder/internal/config"
	"trev.zip/llc/discord-embedder/internal/logger"
)

// Compress compresses the video to reduce file size, adjusts its playback speed, and converts it to h264 mp4.
func (v *Video) Compress(ctx context.Context, factor float64) error {
	if factor != 1 && factor != 1.5 && factor != 2 {
		return fmt.Errorf("unsupported playback speed: %s", strconv.FormatFloat(factor, 'f', -1, 64))
	}

	log := logger.FromContext(ctx)
	cfg := config.FromContext(ctx)

	videoCodec, err := probeCodec(ctx, v.Path, "v:0")
	if err != nil {
		return fmt.Errorf("could not inspect video codec: %w", err)
	}
	if videoCodec == "" {
		return errors.New("input media contains no video stream")
	}

	// An empty audio codec means the video has no audio stream
	audioCodec, err := probeCodec(ctx, v.Path, "a:0")
	if err != nil {
		return fmt.Errorf("could not inspect audio codec: %w", err)
	}

	// Copying video keeps the source's frame layout, so only copy when it starts with a keyframe
	keyframeStart := true
	if videoCodec == "h264" && factor == 1 {
		keyframeStart, err = startsWithKeyframe(ctx, v.Path)
		if err != nil {
			return fmt.Errorf("could not inspect first video frame: %w", err)
		}
		if !keyframeStart {
			log.InfoContext(ctx, "video does not start with a keyframe, transcoding")
		}
	}

	tempPath, finalName, finalPath := compressionOutputPaths(cfg.TempDir, cfg.FilesDir, v.ID)
	defer func() { _ = os.Remove(tempPath) }()

	cmd := exec.CommandContext(ctx, "ffmpeg", compressionArgs(v.Path, tempPath, videoCodec, keyframeStart, audioCodec, factor, cfg.Quicksync, v.originalURL)...)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("could not compress video: %w", err)
	}

	log.DebugContext(ctx, "ffmpeg", "output", string(out))

	if audioCodec != "" {
		if err = requireAudio(ctx, tempPath); err != nil {
			return fmt.Errorf("compressed video must contain audio: %w", err)
		}
	}

	sourcePath := v.Path
	if err = move(tempPath, finalPath); err != nil {
		return err
	}

	v.Name = finalName
	v.Path = finalPath

	if !sameFile(sourcePath, finalPath) {
		if err = os.Remove(sourcePath); err != nil && !os.IsNotExist(err) {
			log.WarnContext(ctx, "could not remove original video", "path", sourcePath, "error", err)
		}
	}

	return nil
}

func compressionArgs(input string, output string, videoCodec string, keyframeStart bool, audioCodec string, factor float64, quicksync bool, originalURL string) []string {
	adjustSpeed := factor != 1
	audio := audioCodec != ""
	transcodeVideo := adjustSpeed || videoCodec != "h264" || !keyframeStart
	transcodeAudio := adjustSpeed || audioCodec != "aac"
	args := []string{"-y"}
	if !adjustSpeed && transcodeVideo && quicksync {
		args = append(args,
			"-hwaccel", "qsv",
			"-hwaccel_output_format", "qsv",
		)
	}

	args = append(args, "-i", input)
	args = append(args, streamArgs(audio)...)

	if adjustSpeed {
		formattedFactor := strconv.FormatFloat(factor, 'f', -1, 64)
		args = append(args, "-filter:v:0", "setpts=PTS/"+formattedFactor)
		if audio {
			args = append(args, "-filter:a:0", "atempo="+formattedFactor+",asetpts=PTS-STARTPTS+STARTPTS/"+formattedFactor)
		}
	}

	switch {
	case !transcodeVideo:
		args = append(args, "-c:v:0", "copy")
	case quicksync:
		args = append(args,
			"-c:v:0", "h264_qsv",
			"-global_quality:v:0", "23",
		)
	default:
		args = append(args,
			"-c:v:0", "libx264",
			"-crf:v:0", "23",
		)
	}

	switch {
	case !audio:
	case transcodeAudio:
		args = append(args, "-c:a:0", "aac")
	default:
		args = append(args, "-c:a:0", "copy")
	}

	if originalURL != "" {
		args = append(args, "-metadata", originalURLMetadataKey+"="+originalURL)
	}

	return append(args,
		"-movflags", "+faststart+use_metadata_tags",
		"-hide_banner",
		"-loglevel", "error",
		output,
	)
}

func compressionOutputPaths(tempDir string, filesDir string, id string) (tempPath string, finalName string, finalPath string) {
	finalName = fmt.Sprintf("%s.mp4", id)
	return filepath.Join(tempDir, fmt.Sprintf("%s.encode.mp4", id)), finalName, filepath.Join(filesDir, finalName)
}

// Trim video to start and end time.
func (v *Video) Trim(ctx context.Context, start string, end string) error {
	log := logger.FromContext(ctx)
	cfg := config.FromContext(ctx)
	tempPath := trimOutputPath(cfg.TempDir, v.ID, v.Name)
	defer func() { _ = os.Remove(tempPath) }()

	audio, err := hasAudio(ctx, v.Path)
	if err != nil {
		return fmt.Errorf("could not inspect audio stream: %w", err)
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", trimArgs(v.Path, tempPath, start, end, audio)...)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("could not trim video: %w", err)
	}

	log.DebugContext(ctx, "ffmpeg", "output", string(out))

	if audio {
		if err = requireAudio(ctx, tempPath); err != nil {
			return fmt.Errorf("trimmed video must contain audio: %w", err)
		}
	}

	if err = move(tempPath, v.Path); err != nil {
		return err
	}

	return nil
}

func trimArgs(input string, output string, start string, end string, audio bool) []string {
	args := []string{
		"-y",
		"-ss", start,
		"-to", end,
		"-i", input,
	}
	args = append(args, streamArgs(audio)...)

	return append(args,
		"-hide_banner",
		"-loglevel", "error",
		output,
	)
}

// streamArgs maps the first video stream, and the first audio stream if audio is true, without inherited metadata.
func streamArgs(audio bool) []string {
	args := []string{"-map", "0:v:0"}
	if audio {
		args = append(args, "-map", "0:a:0")
	}

	return append(args,
		"-sn",
		"-dn",
		"-map_metadata", "-1",
		"-map_chapters", "-1",
	)
}

func trimOutputPath(tempDir string, id string, sourceName string) string {
	return filepath.Join(tempDir, fmt.Sprintf("%s.trim%s", id, filepath.Ext(sourceName)))
}

func sameFile(a string, b string) bool {
	absoluteA, err := filepath.Abs(a)
	if err == nil {
		absoluteB, absErr := filepath.Abs(b)
		if absErr == nil && filepath.Clean(absoluteA) == filepath.Clean(absoluteB) {
			return true
		}
	}

	infoA, err := os.Stat(a)
	if err != nil {
		return false
	}
	infoB, err := os.Stat(b)
	if err != nil {
		return false
	}

	return os.SameFile(infoA, infoB)
}

// thumbnail generates a thumbnail for video.
func (v *Video) thumbnail(ctx context.Context) error {
	log := logger.FromContext(ctx)
	cfg := config.FromContext(ctx)
	name := fmt.Sprintf("%s.jpeg", v.ID)
	path := filepath.Join(cfg.FilesDir, name)

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", v.Path,
		"-update", "true",
		"-vframes:v", "1",
		"-hide_banner",
		"-loglevel", "error",
		path,
	)
	out, err := cmd.Output()
	if err != nil {
		return err
	}

	log.DebugContext(ctx, "ffmpeg", "output", string(out))

	t := Thumbnail{
		File: File{
			Name: name,
			Path: path,
		},
	}
	v.Thumbnail = &t

	return nil
}

// move moves a file from src to dst.
func move(src string, dst string) error {
	if sameFile(src, dst) {
		return nil
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() { _ = srcFile.Close() }()

	tempFile, err := os.CreateTemp(filepath.Dir(dst), "."+filepath.Base(dst)+".*")
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() { _ = os.Remove(tempPath) }()

	if _, err = io.Copy(tempFile, srcFile); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("failed to copy file content: %w", err)
	}
	if err = tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("failed to sync destination file: %w", err)
	}
	if err = tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close destination file: %w", err)
	}
	if err = os.Rename(tempPath, dst); err != nil {
		return fmt.Errorf("failed to install destination file: %w", err)
	}
	if err = os.Remove(src); err != nil {
		return fmt.Errorf("failed to remove source file: %w", err)
	}

	return nil
}
