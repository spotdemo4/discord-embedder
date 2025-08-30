package video

import (
	"context"
	"discord-embedder/internal/config"
	"errors"
	"io/fs"
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

type Video struct {
	File

	ID        string
	Thumbnail *Thumbnail
}

func Get(ctx context.Context, id string) (*Video, error) {
	cfg := config.FromContext(ctx)

	// Find video file
	var videoName string
	var videoPath string
	var thumbnailName string
	var thumbnailPath string
	err := filepath.WalkDir(cfg.FilesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasPrefix(d.Name(), id) {
			return nil
		}

		// Check if thumbnail
		if strings.HasSuffix(d.Name(), "jpeg") {
			thumbnailName = d.Name()
			thumbnailPath = path
		} else {
			videoName = d.Name()
			videoPath = path
		}

		// Stop walking if we found both
		if thumbnailPath != "" && videoPath != "" {
			return fs.SkipAll
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	if videoName == "" || videoPath == "" {
		return nil, errors.New("could not find video file")
	}

	v := &Video{
		ID: id,
		File: File{
			Name: videoName,
			Path: videoPath,
		},
	}

	// Set thumbnail if found
	if thumbnailName != "" && thumbnailPath != "" {
		v.Thumbnail = &Thumbnail{
			File: File{
				Name: thumbnailName,
				Path: thumbnailPath,
			},
		}
	}

	return v, nil
}
