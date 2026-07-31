package video

import (
	"context"
	"discord-embedder/internal/config"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type File struct {
	Name string
	Path string
}

type Thumbnail struct {
	File
}

const originalURLMetadataKey = "original_url"

type Video struct {
	File

	ID          string
	Thumbnail   *Thumbnail
	originalURL string
}

func Get(ctx context.Context, id string) (*Video, error) {
	cfg := config.FromContext(ctx)
	entries, err := os.ReadDir(cfg.FilesDir)
	if err != nil {
		return nil, err
	}

	canonicalName := fmt.Sprintf("%s.mp4", id)
	thumbnailName := fmt.Sprintf("%s.jpeg", id)
	var canonicalPath string
	var thumbnailPath string
	var legacy []File

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		path := filepath.Join(cfg.FilesDir, name)
		switch {
		case name == canonicalName:
			canonicalPath = path
		case name == thumbnailName:
			thumbnailPath = path
		case isLegacyVideo(name, id):
			legacy = append(legacy, File{Name: name, Path: path})
		}
	}

	var file File
	switch {
	case canonicalPath != "":
		file = File{Name: canonicalName, Path: canonicalPath}
	case len(legacy) == 1:
		file = legacy[0]
	case len(legacy) > 1:
		return nil, errors.New("found multiple video files")
	default:
		return nil, errors.New("could not find video file")
	}

	v := &Video{ID: id, File: file}
	if thumbnailPath != "" {
		v.Thumbnail = &Thumbnail{
			File: File{
				Name: thumbnailName,
				Path: thumbnailPath,
			},
		}
	}

	return v, nil
}

func isLegacyVideo(name string, id string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" || strings.TrimSuffix(name, filepath.Ext(name)) != id {
		return false
	}

	switch ext {
	case ".3g2", ".3gp", ".asf", ".avi", ".divx", ".f4v", ".flv", ".m2ts", ".m4v", ".mkv", ".mov", ".mp4", ".mpeg", ".mpg", ".mts", ".mxf", ".ogv", ".rm", ".rmvb", ".ts", ".vob", ".webm", ".wmv":
		return true
	default:
		return false
	}
}
