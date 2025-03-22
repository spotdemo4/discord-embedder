package video

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type video struct {
	ID   string
	Name string
	Url  *url.URL
	File *os.File
}

func New(downloadURL string) (*video, error) {
	URL, err := url.Parse(downloadURL)
	if err != nil {
		return nil, err
	}
	ID := uuid.New().String()

	video := &video{
		ID:   ID,
		Name: ID,
		Url:  URL,
	}

	return video, nil
}

// download downloads the video
func (v *video) Download() error {
	// Find domain of URL
	domain := strings.TrimPrefix(v.Url.Hostname(), "www.")

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
	}

	// Check if cookie file exists for URL
	cookieFileName := ""
	err := filepath.Walk("cookies", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if cookie file exists for domain
		if strings.Contains(info.Name(), domain) {
			cookieFileName = info.Name()
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Download video
	if username != "" && password != "" {
		log.Println("Trying to download with credentials...")
		cmd := exec.Command(
			"yt-dlp",
			"-o", fmt.Sprintf("%s.%%(ext)s", v.Name),
			"--username", username,
			"--password", password,
			v.Url.String(),
		)
		if err := cmd.Run(); err == nil {
			if err := v.find(); err == nil {
				return nil
			}
		}
	}

	if cookieFileName != "" {
		log.Printf("Trying to download with cookie file: %s", cookieFileName)
		cmd := exec.Command(
			"yt-dlp",
			"-o", fmt.Sprintf("%s.%%(ext)s", v.Name),
			"--cookies", filepath.Join("cookies", cookieFileName),
			v.Url.String(),
		)
		if err := cmd.Run(); err == nil {
			if err := v.find(); err == nil {
				return nil
			}
		}
	}

	log.Println("Falling back to default downloader...")
	cmd := exec.Command(
		"yt-dlp",
		"-o", fmt.Sprintf("%s.%%(ext)s", v.Name),
		v.Url.String(),
	)

	if err := cmd.Run(); err != nil {
		return err
	}
	if err := v.find(); err != nil {
		return err
	}

	return nil
}

// find finds the video file
func (v *video) find() error {
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasPrefix(info.Name(), v.Name) {
			v.File, err = os.Open(info.Name())
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}
	if v.File == nil {
		return errors.New("could not find video file")
	}

	return nil
}

// delete deletes the video file
func (v *video) Delete() error {
	if err := v.File.Close(); err != nil {
		log.Printf("could not close file: %s", err)
	}

	if err := os.Remove(v.File.Name()); err != nil {
		return err
	}

	return nil
}

// codec returns the codec of the video
func (v *video) Codec() (string, error) {
	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=codec_name",
		"-of", "default=noprint_wrappers=1:nokey=1",
		v.File.Name(),
	)

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(out), nil
}

// convert and compresses the video to <10MB
func (v *video) Compress(quicksync bool) error {
	var cmd *exec.Cmd
	if quicksync {
		cmd = exec.Command("ffmpeg",
			"-hwaccel", "qsv",
			"-hwaccel_output_format", "qsv",
			"-i", v.File.Name(),
			"-c:v:0", "av1_qsv",
			"-global_quality:v:0", "23",
			"-c:a", "aac",
			fmt.Sprintf("%s-compress.mp4", v.Name),
		)
	} else {
		cmd = exec.Command("ffmpeg",
			"-i", v.File.Name(),
			"-c:v:0", "libsvtav1",
			"-global_quality:v:0", "23",
			"-c:a", "aac",
			fmt.Sprintf("%s-compress.mp4", v.Name),
		)
	}
	if err := cmd.Run(); err != nil {
		return err
	}

	// Delete original video
	if err := v.Delete(); err != nil {
		return err
	}

	// Set new video name
	v.Name = fmt.Sprintf("%s-compress", v.Name)

	// Find new video file
	if err := v.find(); err != nil {
		return err
	}

	return nil
}

// Trim video to start and end time
func (v *video) Trim(start string, end string) error {
	cmd := exec.Command("ffmpeg", "-ss", start, "-to", end, "-i", v.File.Name(), fmt.Sprintf("%s-trim.mp4", v.Name))

	if err := cmd.Run(); err != nil {
		return err
	}

	// Delete original video
	if err := v.Delete(); err != nil {
		return err
	}

	// Set new video name
	v.Name = fmt.Sprintf("%s-trim", v.Name)

	// Find new video file
	if err := v.find(); err != nil {
		return err
	}

	return nil
}

// Add spoiler to video
func (v *video) Spoiler() error {
	if err := v.File.Close(); err != nil {
		log.Printf("could not close file: %s", err)
	}

	// Rename starting with SPOILER
	err := os.Rename(v.File.Name(), "SPOILER_"+v.File.Name())
	if err != nil {
		return err
	}

	// Set new video name
	v.Name = fmt.Sprintf("SPOILER_%s", v.Name)

	// Open new file
	v.File, err = os.Open(v.File.Name())
	if err != nil {
		return err
	}

	return nil
}

func (v *video) Export() error {
	fn := fmt.Sprintf("%s%s", v.ID, filepath.Ext(v.File.Name()))

	// Create new file
	destFile, err := os.Create(fmt.Sprintf("files/%s", fn))
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy file over to /files
	_, err = io.Copy(destFile, v.File)
	if err != nil {
		return err
	}

	// Make sure copy completes
	err = destFile.Sync()
	if err != nil {
		return err
	}

	return nil
}

// Generate thumbnail for video
func (v *video) Thumbnail() error {
	cmd := exec.Command("ffmpeg", "-i", v.File.Name(), "-vframes", "1", fmt.Sprintf("files/%s.jpeg", v.ID))
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
