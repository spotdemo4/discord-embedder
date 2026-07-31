package video

import (
	"errors"
	"net/url"
	"strings"
)

func sanitizeOriginalURL(rawURL string) (string, error) {
	link, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	link.Scheme = strings.ToLower(link.Scheme)
	if (link.Scheme != "http" && link.Scheme != "https") || link.Host == "" || link.Opaque != "" {
		return "", errors.New("source URL must be an absolute HTTP or HTTPS URL")
	}

	link.User = nil
	link.Fragment = ""
	link.RawFragment = ""

	host := strings.TrimSuffix(strings.ToLower(link.Hostname()), ".")
	if matchesHost(host, "youtube.com") && strings.TrimSuffix(link.Path, "/") == "/watch" {
		videoID := firstNonempty(link.Query()["v"])
		if videoID == "" {
			return "", errors.New("YouTube watch URL is missing a video ID")
		}
		link.RawQuery = url.Values{"v": []string{videoID}}.Encode()
		link.ForceQuery = false
	} else {
		clearURLQuery(link)
	}

	return link.String(), nil
}

func matchesHost(host string, domain string) bool {
	return host == domain || strings.HasSuffix(host, "."+domain)
}

func clearURLQuery(link *url.URL) {
	link.RawQuery = ""
	link.ForceQuery = false
}

func firstNonempty(values []string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
