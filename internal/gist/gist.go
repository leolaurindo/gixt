package gist

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

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
	Raw         map[string]any  `json:"-"`
}

type ListItem struct {
	ID          string          `json:"id"`
	Description string          `json:"description"`
	Files       map[string]File `json:"files"`
	Owner       Owner           `json:"owner"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

var gistURLRe = regexp.MustCompile(`gist\.github\.com/[^/]+/([a-fA-F0-9]+)`) // gist url path extraction

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

func callGH(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh %v failed: %v: %s", args, err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

func Fetch(ctx context.Context, id string, ref string) (Gist, error) {
	path := fmt.Sprintf("/gists/%s", id)
	if ref != "" {
		path = fmt.Sprintf("/gists/%s/%s", id, ref)
	}
	out, err := callGH(ctx, "api", path)
	if err != nil {
		return Gist{}, err
	}
	var g Gist
	if err := json.Unmarshal(out, &g); err != nil {
		return Gist{}, fmt.Errorf("parse gist response: %w", err)
	}
	g.Raw = map[string]any{}
	if err := json.Unmarshal(out, &g.Raw); err != nil {
		// ignore secondary parse failure
	}
	return g, nil
}

func UpdateFiles(ctx context.Context, id string, files map[string]string) (Gist, error) {
	type filePayload struct {
		Content string `json:"content"`
	}
	payload := struct {
		Files map[string]filePayload `json:"files"`
	}{
		Files: map[string]filePayload{},
	}
	for name, content := range files {
		payload.Files[name] = filePayload{Content: content}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return Gist{}, fmt.Errorf("encode gist payload: %w", err)
	}

	path := fmt.Sprintf("/gists/%s", id)
	cmd := exec.CommandContext(ctx, "gh", "api", "-X", "PATCH", path, "--input", "-")
	cmd.Stdin = bytes.NewReader(body)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return Gist{}, fmt.Errorf("gh api patch %s failed: %v: %s", id, err, strings.TrimSpace(stderr.String()))
	}
	var g Gist
	if err := json.Unmarshal(out, &g); err != nil {
		return Gist{}, fmt.Errorf("parse gist response: %w", err)
	}
	g.Raw = map[string]any{}
	if err := json.Unmarshal(out, &g.Raw); err != nil {
		// ignore secondary parse failure
	}
	return g, nil
}

func List(ctx context.Context, perPage, maxPages int) ([]ListItem, error) {
	if perPage <= 0 {
		perPage = 50
	}
	if maxPages <= 0 {
		maxPages = 1
	}
	var all []ListItem
	for page := 1; page <= maxPages; page++ {
		path := fmt.Sprintf("/gists?per_page=%d&page=%d", perPage, page)
		out, err := callGH(ctx, "api", path)
		if err != nil {
			return nil, err
		}
		var batch []ListItem
		if err := json.Unmarshal(out, &batch); err != nil {
			return nil, fmt.Errorf("parse gist list: %w", err)
		}
		all = append(all, batch...)
		if len(batch) < perPage {
			break
		}
	}
	return all, nil
}

func ListForOwner(ctx context.Context, owner string, perPage, maxPages int) ([]ListItem, error) {
	if perPage <= 0 {
		perPage = 50
	}
	if maxPages <= 0 {
		maxPages = 1
	}
	var all []ListItem
	for page := 1; page <= maxPages; page++ {
		path := fmt.Sprintf("/users/%s/gists?per_page=%d&page=%d", owner, perPage, page)
		out, err := callGH(ctx, "api", path)
		if err != nil {
			return nil, err
		}
		var batch []ListItem
		if err := json.Unmarshal(out, &batch); err != nil {
			return nil, fmt.Errorf("parse gist list for owner %s: %w", owner, err)
		}
		all = append(all, batch...)
		if len(batch) < perPage {
			break
		}
	}
	return all, nil
}

func CurrentUser(ctx context.Context) (string, error) {
	out, err := callGH(ctx, "api", "/user")
	if err != nil {
		return "", err
	}
	var resp struct {
		Login string `json:"login"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return "", fmt.Errorf("parse current user: %w", err)
	}
	return resp.Login, nil
}

func GuessOwner(g Gist) string {
	if g.Owner.Login != "" {
		return g.Owner.Login
	}
	if login, ok := g.Raw["owner"].(map[string]any); ok {
		if l, ok := login["login"].(string); ok {
			return l
		}
	}
	return ""
}

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

// IsNotFound reports whether the error from gh indicates a 404.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "http 404") || strings.Contains(msg, " 404 ") || strings.Contains(msg, "not found")
}

var ErrAmbiguous = errors.New("multiple matching gists")
