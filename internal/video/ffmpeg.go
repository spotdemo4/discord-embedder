package video

import (
	"context"
	"discord-embedder/internal/config"
	"discord-embedder/internal/logger"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// Compress compresses the video to reduce file size and converts to h264 mp4.
func (v *Video) Compress(ctx context.Context) error {
	log := logger.FromContext(ctx)
	cfg := config.FromContext(ctx)

	videoCodec, err := probeCodec(ctx, v.Path, "v:0")
	if err != nil {
		return fmt.Errorf("could not inspect video codec: %w", err)
	}
	if videoCodec == "" {
		return errors.New("input media contains no video stream")
	}

	audioCodec, err := probeCodec(ctx, v.Path, "a:0")
	if err != nil {
		return fmt.Errorf("could not inspect audio codec: %w", err)
	}
	if audioCodec == "" {
		return errors.New("media contains no audio stream")
	}

	tempPath, finalName, finalPath := compressionOutputPaths(cfg.TempDir, cfg.FilesDir, v.ID)
	defer os.Remove(tempPath)

	cmd := exec.CommandContext(ctx, "ffmpeg", compressionArgs(v.Path, tempPath, videoCodec, audioCodec, cfg.Quicksync)...)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("could not compress video: %w", err)
	}

	log.DebugContext(ctx, "ffmpeg", "output", string(out))

	if err = requireAudio(ctx, tempPath); err != nil {
		return fmt.Errorf("compressed video must contain audio: %w", err)
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

func compressionArgs(input string, output string, videoCodec string, audioCodec string, quicksync bool) []string {
	transcodeVideo := videoCodec != "h264"
	args := []string{"-y"}
	if transcodeVideo && quicksync {
		args = append(args,
			"-hwaccel", "qsv",
			"-hwaccel_output_format", "qsv",
		)
	}

	args = append(args,
		"-i", input,
		"-map", "0:v:0",
		"-map", "0:a:0",
		"-sn",
		"-dn",
	)

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

	if audioCodec == "aac" {
		args = append(args, "-c:a:0", "copy")
	} else {
		args = append(args, "-c:a:0", "aac")
	}

	return append(args,
		"-movflags", "+faststart",
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
	defer os.Remove(tempPath)

	cmd := exec.CommandContext(ctx, "ffmpeg", trimArgs(v.Path, tempPath, start, end)...)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("could not trim video: %w", err)
	}

	log.DebugContext(ctx, "ffmpeg", "output", string(out))

	if err = requireAudio(ctx, tempPath); err != nil {
		return fmt.Errorf("trimmed video must contain audio: %w", err)
	}

	if err = move(tempPath, v.Path); err != nil {
		return err
	}

	return nil
}

func trimArgs(input string, output string, start string, end string) []string {
	return []string{
		"-y",
		"-ss", start,
		"-to", end,
		"-i", input,
		"-map", "0:v:0",
		"-map", "0:a:0",
		"-sn",
		"-dn",
		"-hide_banner",
		"-loglevel", "error",
		output,
	}
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
	defer srcFile.Close()

	tempFile, err := os.CreateTemp(filepath.Dir(dst), "."+filepath.Base(dst)+".*")
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err = io.Copy(tempFile, srcFile); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to copy file content: %w", err)
	}
	if err = tempFile.Sync(); err != nil {
		tempFile.Close()
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
