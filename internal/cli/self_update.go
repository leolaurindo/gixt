package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/leolaurindo/gixt/internal/update"
	"github.com/leolaurindo/gixt/internal/version"
)

func selfUpdate(cmd *cobra.Command, args []string) error {
	target, err := os.Executable()
	if err != nil {
		return err
	}
	target, err = filepath.EvalSymlinks(target)
	if err != nil {
		return err
	}
	if strings.Contains(filepath.ToSlash(target), "/Cellar/gixt/") {
		return homebrewUpdateAdvice(cmd.Context(), target)
	}
	if (runtime.GOOS != "linux" && runtime.GOOS != "darwin" && runtime.GOOS != "windows") ||
		(runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64") ||
		(runtime.GOOS == "windows" && runtime.GOARCH != "amd64") {
		return fmt.Errorf("no release binary for %s/%s; update using your original installation method", runtime.GOOS, runtime.GOARCH)
	}

	rel, err := fetchLatestRelease(cmd.Context())
	if err != nil {
		return err
	}
	latest := strings.TrimSpace(rel.TagName)
	if !regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(latest) {
		return fmt.Errorf("invalid stable release version %q", latest)
	}
	if version.Version != "dev" && compareVersions(trimVersion(latest), trimVersion(version.Version)) <= 0 {
		logf("gixt is up to date (%s).", version.Version)
		return nil
	}

	extension := "tar.gz"
	if runtime.GOOS == "windows" {
		extension = "zip"
	}
	asset := fmt.Sprintf("gixt_%s_%s_%s.%s", latest, runtime.GOOS, runtime.GOARCH, extension)
	releaseURL := "https://github.com/" + updateRepo + "/releases/download/" + latest
	logf("Updating %s to %s...", target, latest)
	client := &http.Client{Timeout: 2 * time.Minute}
	if err := update.Apply(cmd.Context(), client, releaseURL, asset, target); err != nil {
		return fmt.Errorf("updating %s: %w", target, err)
	}
	logf("Updated gixt to %s.", latest)
	return nil
}

func homebrewUpdateAdvice(ctx context.Context, target string) error {
	brew, err := exec.LookPath("brew")
	if err != nil {
		return fmt.Errorf("Homebrew-managed executable; restore brew on PATH and run `brew upgrade gixt`: %w", err)
	}
	checkCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	prefix, err := exec.CommandContext(checkCtx, brew, "--prefix", "gixt").Output()
	if err != nil {
		return fmt.Errorf("cannot verify Homebrew installation; use `brew upgrade gixt`: %w", err)
	}
	installed, err := os.Stat(filepath.Join(strings.TrimSpace(string(prefix)), "bin", "gixt"))
	if err != nil {
		return fmt.Errorf("cannot verify Homebrew executable: %w", err)
	}
	current, err := os.Stat(target)
	if err != nil {
		return err
	}
	if !os.SameFile(installed, current) {
		return fmt.Errorf("running executable does not match Homebrew's installed gixt; use `brew upgrade gixt` directly")
	}
	logf("gixt is managed by Homebrew.\nUpdate with: brew upgrade gixt\nThe available version is controlled by your tap.")
	return nil
}
