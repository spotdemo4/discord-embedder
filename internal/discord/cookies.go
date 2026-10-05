package discord

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"trev.zip/llc/discord-embedder/internal/cookies"
	"trev.zip/llc/discord-embedder/internal/logger"
	"trev.zip/llc/discord-embedder/internal/video"

	"github.com/bwmarrin/discordgo"
)

const cookiesFAQ = "https://github.com/yt-dlp/yt-dlp/wiki/FAQ#how-do-i-pass-cookies-to-yt-dlp"

// loginRequiredError asks the user to provide cookies for a site that requires a login.
type loginRequiredError struct {
	err         error
	site        string
	usedCookies bool
}

func (e *loginRequiredError) Error() string {
	return e.err.Error()
}

func (e *loginRequiredError) Unwrap() error {
	return e.err
}

func (e *loginRequiredError) message() string {
	var reason string
	if e.usedCookies {
		reason = fmt.Sprintf("Your saved cookies for %s did not work, they may have expired.", e.site)
	} else {
		reason = fmt.Sprintf("%s requires a login to download this.", e.site)
	}

	return fmt.Sprintf(
		"%s\nSave a Netscape `cookies.txt` exported from a browser logged in to %s (<%s>) with `/cookies`, then try again. "+
			"Only you can see `/cookies`, and only the cookies for %s are kept.",
		reason,
		e.site,
		cookiesFAQ,
		strings.Join(cookies.Sites(), ", "),
	)
}

// download downloads a video using the cookies saved by userID for the URL's site.
func download(ctx context.Context, userID string, downloadURL string) (*video.Video, error) {
	log := logger.FromContext(ctx)

	var site string
	var saved []byte
	if cookies.Enabled(ctx) {
		var err error
		site, err = cookies.Site(downloadURL)
		if err != nil {
			log.DebugContext(ctx, "not using cookies", "reason", err)
		}
		if site != "" {
			saved, err = cookies.Load(ctx, userID, site)
			if err != nil {
				log.WarnContext(ctx, "could not load saved cookies", "site", site, "error", err)
			}
		}
	}

	v, err := video.Download(ctx, downloadURL, saved)
	if err != nil && site != "" && errors.Is(err, video.ErrLoginRequired) {
		return nil, &loginRequiredError{err: err, site: site, usedCookies: len(saved) > 0}
	}

	return v, err
}

// saveCookies saves the supported cookies in a cookies.txt attachment for userID and returns the saved sites.
func saveCookies(ctx context.Context, userID string, attachment *discordgo.MessageAttachment) ([]string, error) {
	if !cookies.Enabled(ctx) {
		return nil, cookies.ErrDisabled
	}
	if attachment.Size > cookies.MaxSize {
		return nil, errors.New("cookies file is too large")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, attachment.URL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Join(errors.New("could not download cookies file"), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("could not download cookies file: %s", resp.Status)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, cookies.MaxSize+1))
	if err != nil {
		return nil, errors.Join(errors.New("could not download cookies file"), err)
	}
	if len(data) > cookies.MaxSize {
		return nil, errors.New("cookies file is too large")
	}

	sites, err := cookies.Save(ctx, userID, data)
	if err != nil {
		return nil, errors.Join(errors.New("could not save cookies"), err)
	}

	return sites, nil
}

func interactionUserID(i *discordgo.InteractionCreate) string {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}

	return ""
}
