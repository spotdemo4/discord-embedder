package cookies

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"

	"trev.zip/llc/discord-embedder/internal/config"
	"trev.zip/llc/discord-embedder/internal/logger"

	"golang.org/x/net/publicsuffix"
)

const (
	// MaxSize is the largest cookies file that will be accepted.
	MaxSize = 1 << 20

	header       = "# Netscape HTTP Cookie File"
	httpOnly     = "#HttpOnly_"
	cookieFields = 7
)

var headerPattern = regexp.MustCompile(`^#( Netscape)? HTTP Cookie File`)

// sites maps each domain that cookies are kept for to the site it belongs to.
var sites = map[string]string{
	"instagram.com": "instagram.com",
	"reddit.com":    "reddit.com",
	"tiktok.com":    "tiktok.com",
	"twitter.com":   "x.com",
	"x.com":         "x.com",
}

// ErrDisabled is returned when cookies cannot be used because no encryption key is configured.
var ErrDisabled = errors.New("cookies are disabled, COOKIES_KEY is not set")

// saveMu serializes saves so concurrent uploads by a user do not drop each other's cookies.
var saveMu sync.Mutex

// Enabled reports whether an encryption key is configured so cookies can be saved and used.
func Enabled(ctx context.Context) bool {
	return len(config.FromContext(ctx).CookiesKey) > 0
}

// Sites returns the sites that cookies are kept for.
func Sites() []string {
	var out []string
	for _, site := range sites {
		if !slices.Contains(out, site) {
			out = append(out, site)
		}
	}
	slices.Sort(out)

	return out
}

// Site returns the site that cookies are kept under for rawURL, or an error if cookies are not kept for it.
func Site(rawURL string) (string, error) {
	link, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	return domainSite(link.Hostname())
}

func domainSite(domain string) (string, error) {
	domain = strings.TrimPrefix(strings.ToLower(domain), ".")
	if domain == "" {
		return "", errors.New("no domain")
	}

	registrable, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if err != nil {
		return "", err
	}

	site, ok := sites[registrable]
	if !ok {
		return "", fmt.Errorf("cookies are not kept for %s", registrable)
	}

	return site, nil
}

type cookie struct {
	line string
	site string
}

// parse validates a Netscape cookies file and returns the cookies for supported sites.
func parse(data []byte) ([]cookie, error) {
	trimmed := bytes.TrimSpace(data)
	if bytes.HasPrefix(trimmed, []byte("[")) || bytes.HasPrefix(trimmed, []byte("{")) {
		return nil, errors.New("cookies file must be Netscape formatted, not JSON")
	}

	var cookies []cookie
	seenHeader := false
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), MaxSize)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimRight(scanner.Text(), "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !seenHeader {
			if !headerPattern.MatchString(line) {
				return nil, errors.New("cookies file must be a Netscape formatted cookies.txt")
			}
			seenHeader = true
			continue
		}
		if strings.HasPrefix(line, "#") && !strings.HasPrefix(line, httpOnly) {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) != cookieFields {
			return nil, fmt.Errorf("invalid cookie on line %d", lineNumber)
		}
		site, err := domainSite(strings.TrimPrefix(fields[0], httpOnly))
		if err != nil {
			continue
		}

		cookies = append(cookies, cookie{line: line, site: site})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if !seenHeader {
		return nil, errors.New("cookies file is empty")
	}

	return cookies, nil
}

// Save stores the supported cookies from a Netscape cookies file for userID.
// Saved cookies for each site in the file are replaced, and saved cookies for other sites are kept.
// It returns the sites that were saved.
func Save(ctx context.Context, userID string, data []byte) ([]string, error) {
	cfg := config.FromContext(ctx)
	if !Enabled(ctx) {
		return nil, ErrDisabled
	}

	uploaded, err := parse(data)
	if err != nil {
		return nil, err
	}
	saved := map[string]bool{}
	for _, c := range uploaded {
		saved[c.site] = true
	}
	if len(saved) == 0 {
		return nil, fmt.Errorf("cookies file has no cookies for %s", strings.Join(Sites(), ", "))
	}

	path, err := path(cfg.CookiesDir, userID)
	if err != nil {
		return nil, err
	}

	saveMu.Lock()
	defer saveMu.Unlock()

	// Keep previously saved cookies for sites that were not uploaded
	existing, err := load(path, cfg.CookiesKey, userID)
	if errors.Is(err, errDecrypt) {
		// The key changed, so the saved cookies can never be read again
		logger.FromContext(ctx).WarnContext(ctx, "replacing saved cookies that could not be decrypted", "error", err)
		existing = nil
	} else if err != nil {
		return nil, errors.Join(errors.New("could not read saved cookies"), err)
	}

	out := bytes.NewBufferString(header + "\n")
	for _, c := range existing {
		if !saved[c.site] {
			out.WriteString(c.line + "\n")
		}
	}
	for _, c := range uploaded {
		out.WriteString(c.line + "\n")
	}

	encrypted, err := encrypt(cfg.CookiesKey, userID, out.Bytes())
	if err != nil {
		return nil, err
	}
	if err = write(path, encrypted); err != nil {
		return nil, err
	}

	return slices.Sorted(maps.Keys(saved)), nil
}

// Load returns the decrypted cookies file saved for userID if it has cookies for site, or nil if it does not.
func Load(ctx context.Context, userID string, site string) ([]byte, error) {
	cfg := config.FromContext(ctx)
	if !Enabled(ctx) {
		return nil, ErrDisabled
	}

	path, err := path(cfg.CookiesDir, userID)
	if err != nil {
		return nil, err
	}

	saved, err := load(path, cfg.CookiesKey, userID)
	if err != nil {
		return nil, err
	}
	if !slices.ContainsFunc(saved, func(c cookie) bool { return c.site == site }) {
		return nil, nil
	}

	out := bytes.NewBufferString(header + "\n")
	for _, c := range saved {
		out.WriteString(c.line + "\n")
	}

	return out.Bytes(), nil
}

func load(path string, key []byte, userID string) ([]cookie, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("saved cookies are not a regular file")
	}

	encrypted, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	data, err := decrypt(key, userID, encrypted)
	if err != nil {
		return nil, err
	}

	return parse(data)
}

func write(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	// Write to a temporary file first so a concurrent download never reads a partial file
	file, err := os.CreateTemp(dir, ".cookies-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())

	if _, err = file.Write(data); err != nil {
		return errors.Join(err, file.Close())
	}
	if err = file.Close(); err != nil {
		return err
	}

	return os.Rename(file.Name(), path)
}

func path(cookiesDir string, userID string) (string, error) {
	if cookiesDir == "" {
		return "", errors.New("cookies directory is not configured")
	}
	if userID == "" ||
		userID == "." ||
		userID == ".." ||
		strings.ContainsAny(userID, `/\`) ||
		filepath.Base(userID) != userID {
		return "", errors.New("invalid user ID")
	}

	return filepath.Join(cookiesDir, userID+".txt"), nil
}
