package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/leolaurindo/gixt/internal/version"
)

const updateRepo = "leolaurindo/gixt"

func newSelfCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "self",
		Short: "gixt self-management",
		Args:  cobra.NoArgs,
	}
	c.AddCommand(
		&cobra.Command{Use: "version", Short: "print the gixt version", Args: cobra.NoArgs, RunE: selfVersion},
		&cobra.Command{Use: "update-check", Short: "check whether a newer gixt release exists", Args: cobra.NoArgs, RunE: selfUpdateCheck},
	)
	return c
}

func selfVersion(cmd *cobra.Command, args []string) error {
	fmt.Println(version.Version)
	return nil
}

type releaseInfo struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

func selfUpdateCheck(cmd *cobra.Command, args []string) error {
	current := strings.TrimSpace(version.Version)
	rel, err := fetchLatestRelease(cmd.Context())
	if err != nil {
		return err
	}
	latest := strings.TrimSpace(rel.TagName)
	logf("current version: %s", current)
	logf("latest version:  %s", latest)
	if current != "dev" && compareVersions(trimVersion(latest), trimVersion(current)) <= 0 {
		logf("gixt is up to date.")
		return nil
	}
	logf("update available: %s", rel.HTMLURL)
	return nil
}

func fetchLatestRelease(ctx context.Context) (releaseInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+updateRepo+"/releases/latest", nil)
	if err != nil {
		return releaseInfo{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	hc := &http.Client{Timeout: 15 * time.Second}
	resp, err := hc.Do(req)
	if err != nil {
		return releaseInfo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return releaseInfo{}, fmt.Errorf("checking releases: http %d", resp.StatusCode)
	}
	var rel releaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return releaseInfo{}, fmt.Errorf("parse release info: %w", err)
	}
	if rel.TagName == "" {
		return releaseInfo{}, fmt.Errorf("latest release missing tag_name")
	}
	return rel, nil
}

func compareVersions(a, b string) int {
	pa := strings.Split(a, ".")
	pb := strings.Split(b, ".")
	max := len(pa)
	if len(pb) > max {
		max = len(pb)
	}
	for len(pa) < max {
		pa = append(pa, "0")
	}
	for len(pb) < max {
		pb = append(pb, "0")
	}
	for i := 0; i < max; i++ {
		ai := toInt(pa[i])
		bi := toInt(pb[i])
		if ai > bi {
			return 1
		}
		if ai < bi {
			return -1
		}
	}
	return 0
}

func trimVersion(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

func toInt(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n
}
