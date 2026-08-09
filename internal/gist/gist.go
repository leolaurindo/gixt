package gist

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const apiBase = "https://api.github.com"

type File struct {
	Filename  string `json:"filename"`
	Type      string `json:"type"`
	Language  string `json:"language"`
	RawURL    string `json:"raw_url"`
	Size      int    `json:"size"`
	Truncated bool   `json:"truncated"`
	Content   string `json:"content"`
}

type Owner struct {
	Login string `json:"login"`
}

type HistoryEntry struct {
	Version     string    `json:"version"`
	CommittedAt time.Time `json:"committed_at"`
}

type Gist struct {
	ID          string          `json:"id"`
	Description string          `json:"description"`
	Files       map[string]File `json:"files"`
	Owner       Owner           `json:"owner"`
	History     []HistoryEntry  `json:"history"`
	UpdatedAt   time.Time       `json:"updated_at"`
	HTMLURL     string          `json:"html_url"`
}

type ListItem struct {
	ID          string          `json:"id"`
	Description string          `json:"description"`
	Files       map[string]File `json:"files"`
	Owner       Owner           `json:"owner"`
	History     []HistoryEntry  `json:"history"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

var gistURLRe = regexp.MustCompile(`gist\.github\.com/[^/]+/([a-fA-F0-9]+)`)

func ExtractID(input string) string {
	if matches := gistURLRe.FindStringSubmatch(input); len(matches) == 2 {
		return matches[1]
	}
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	trimmed = strings.TrimSuffix(trimmed, "/")
	if u, err := url.Parse(trimmed); err == nil && u.Host != "" {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) > 0 {
			last := parts[len(parts)-1]
			last = strings.SplitN(last, "#", 2)[0]
			last = strings.SplitN(last, "?", 2)[0]
			if last != "" {
				return last
			}
		}
	}
	return trimmed
}

func (g Gist) LatestVersion() string {
	if len(g.History) > 0 && g.History[0].Version != "" {
		return g.History[0].Version
	}
	return ""
}

// Client is a thin GitHub REST client. An empty token means unauthenticated.
type Client struct {
	hc    *http.Client
	base  string
	token string
}

func New(token string) *Client {
	return newWithBase(apiBase, token)
}

func newWithBase(base, token string) *Client {
	return &Client{hc: &http.Client{Timeout: 30 * time.Second}, base: base, token: token}
}

func (c *Client) HasToken() bool { return c.token != "" }

// get performs a GET and returns the body, the response headers, and whether
// the server replied 304 (content unchanged given the If-None-Match header).
func (c *Client) get(ctx context.Context, path string, etag string) ([]byte, http.Header, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return nil, nil, false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, nil, false, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotModified:
		return nil, resp.Header, true, nil
	case http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, nil, false, fmt.Errorf("read response: %w", err)
		}
		return body, resp.Header, false, nil
	case http.StatusNotFound:
		return nil, nil, false, &NotFoundError{Path: path}
	default:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, nil, false, fmt.Errorf("github api %s: http %d: %s", path, resp.StatusCode, strings.TrimSpace(string(body)))
	}
}

type NotFoundError struct{ Path string }

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("github api %s: http 404 (not found)", e.Path)
}

func IsNotFound(err error) bool {
	var nf *NotFoundError
	return errors.As(err, &nf)
}

// Fetch returns gist metadata, optionally short-circuited with an ETag.
// On a 304, gist is zero-valued, notModified is true, and etag is unchanged.
func (c *Client) Fetch(ctx context.Context, id string, ref string) (Gist, error) {
	g, _, _, err := c.FetchCached(ctx, id, ref, "")
	return g, err
}

func (c *Client) FetchCached(ctx context.Context, id string, ref string, etag string) (Gist, string, bool, error) {
	path := fmt.Sprintf("/gists/%s", id)
	if ref != "" {
		path = fmt.Sprintf("/gists/%s/%s", id, ref)
	}
	body, hdr, notModified, err := c.get(ctx, path, etag)
	if err != nil {
		return Gist{}, "", false, err
	}
	if notModified {
		return Gist{}, "", true, nil
	}
	var g Gist
	if err := json.Unmarshal(body, &g); err != nil {
		return Gist{}, "", false, fmt.Errorf("parse gist response: %w", err)
	}
	return g, hdr.Get("Etag"), false, nil
}

// Download fetches a file's raw content (e.g. from raw_url).
func (c *Client) Download(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download %s: http %d", rawURL, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (c *Client) CurrentUser(ctx context.Context) (string, error) {
	if c.token == "" {
		return "", errors.New("not authenticated")
	}
	body, _, _, err := c.get(ctx, "/user", "")
	if err != nil {
		return "", err
	}
	var resp struct {
		Login string `json:"login"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parse current user: %w", err)
	}
	return resp.Login, nil
}

func (c *Client) ListForOwner(ctx context.Context, owner string, perPage, maxPages int) ([]ListItem, error) {
	return c.list(ctx, fmt.Sprintf("/users/%s/gists", owner), perPage, maxPages)
}

func (c *Client) list(ctx context.Context, base string, perPage, maxPages int) ([]ListItem, error) {
	if perPage <= 0 {
		perPage = 50
	}
	if maxPages <= 0 {
		maxPages = 1
	}
	var all []ListItem
	for page := 1; page <= maxPages; page++ {
		body, _, _, err := c.get(ctx, fmt.Sprintf("%s?per_page=%d&page=%d", base, perPage, page), "")
		if err != nil {
			return nil, err
		}
		var batch []ListItem
		if err := json.Unmarshal(body, &batch); err != nil {
			return nil, fmt.Errorf("parse gist list: %w", err)
		}
		all = append(all, batch...)
		if len(batch) < perPage {
			break
		}
	}
	return all, nil
}

func (c *Client) UpdateDescription(ctx context.Context, id string, description string) (Gist, error) {
	return c.patch(ctx, fmt.Sprintf("/gists/%s", id), map[string]string{"description": description})
}

func (c *Client) Create(ctx context.Context, files map[string]string, description string, public bool) (Gist, error) {
	payload := map[string]any{
		"files":  map[string]any{},
		"public": public,
	}
	if description != "" {
		payload["description"] = description
	}
	for name, content := range files {
		payload["files"].(map[string]any)[name] = map[string]string{"content": content}
	}
	return c.post(ctx, "/gists", payload)
}

func (c *Client) patch(ctx context.Context, path string, payload any) (Gist, error) {
	return c.send(ctx, http.MethodPatch, path, payload)
}

func (c *Client) post(ctx context.Context, path string, payload any) (Gist, error) {
	return c.send(ctx, http.MethodPost, path, payload)
}

func (c *Client) send(ctx context.Context, method, path string, payload any) (Gist, error) {
	if c.token == "" {
		return Gist{}, errors.New("this action requires authentication; run `gixt auth login`")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return Gist{}, fmt.Errorf("encode payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, bytes.NewReader(body))
	if err != nil {
		return Gist{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return Gist{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return Gist{}, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return Gist{}, fmt.Errorf("github api %s %s: http %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var g Gist
	if err := json.Unmarshal(data, &g); err != nil {
		return Gist{}, fmt.Errorf("parse gist response: %w", err)
	}
	return g, nil
}

func GuessOwner(g Gist) string { return g.Owner.Login }

func IsLikelyGistID(id string) bool {
	trimmed := strings.TrimSpace(id)
	if len(trimmed) < 8 {
		return false
	}
	for _, r := range trimmed {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			continue
		}
		return false
	}
	return true
}
