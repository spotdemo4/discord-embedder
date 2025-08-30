package video

import (
	"context"
	"discord-embedder/internal/config"
	"discord-embedder/internal/logger"
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
	tempPath := filepath.Join(cfg.TempDir, v.Name)

	var cmd *exec.Cmd
	if cfg.Quicksync {
		cmd = exec.CommandContext(ctx, "ffmpeg",
			"-hwaccel", "qsv",
			"-hwaccel_output_format", "qsv",
			"-i", v.Path,
			"-c:v:0", "h264_qsv",
			"-global_quality:v:0", "23",
			"-c:a", "aac",
			tempPath,
		)
	} else {
		cmd = exec.CommandContext(ctx, "ffmpeg",
			"-i", v.Path,
			"-c:v:0", "libx264",
			"-global_quality:v:0", "23",
			"-c:a", "aac",
			tempPath,
		)
	}
	out, err := cmd.Output()
	if err != nil {
		return err
	}

	log.DebugContext(ctx, "ffmpeg", "output", string(out))

	// Delete original
	err = os.Remove(v.Path)
	if err != nil {
		return err
	}

	// Move temp to original path
	err = move(tempPath, v.Path)
	if err != nil {
		return err
	}

	return nil
}

// Trim video to start and end time.
func (v *Video) Trim(ctx context.Context, start string, end string) error {
	log := logger.FromContext(ctx)
	cfg := config.FromContext(ctx)
	tempPath := filepath.Join(cfg.TempDir, v.Name)

	cmd := exec.CommandContext(ctx, "ffmpeg", "-ss", start, "-to", end, "-i", v.Path, tempPath)
	out, err := cmd.Output()
	if err != nil {
		return err
	}

	log.DebugContext(ctx, "ffmpeg", "output", string(out))

	// Delete original
	err = os.Remove(v.Path)
	if err != nil {
		return err
	}

	// Move temp to original path
	err = move(tempPath, v.Path)
	if err != nil {
		return err
	}

	return nil
}

// thumbnail generates a thumbnail for video.
func (v *Video) thumbnail(ctx context.Context) error {
	log := logger.FromContext(ctx)
	cfg := config.FromContext(ctx)
	name := fmt.Sprintf("%s.jpeg", v.ID)
	path := filepath.Join(cfg.FilesDir, name)

	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", v.Path, "-update", "true", "-vframes:v", "1", path)
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
	// Open the source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close() // Ensure source file is closed

	// Create the destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close() // Ensure destination file is closed

	// Copy the content from source to destination
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Remove the original file
	err = os.Remove(src)
	if err != nil {
		return fmt.Errorf("failed to remove original file: %w", err)
	}

	return nil
}
