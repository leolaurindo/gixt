package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/leolaurindo/gix/internal/version"
)

const updateRepo = "leolaurindo/gix"

type releaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type releaseInfo struct {
	TagName string         `json:"tag_name"`
	HTMLURL string         `json:"html_url"`
	Assets  []releaseAsset `json:"assets"`
}

type updateResult struct {
	CurrentVersion    string   `json:"current"`
	LatestVersion     string   `json:"latest"`
	UpdateAvailable   bool     `json:"update_available"`
	ReleaseURL        string   `json:"release_url"`
	DownloadURL       string   `json:"download_url,omitempty"`
	AssetName         string   `json:"asset_name,omitempty"`
	InstallPath       string   `json:"install_path,omitempty"`
	InstallWritable   bool     `json:"install_path_writable,omitempty"`
	SuggestedCommands []string `json:"suggested_commands,omitempty"`
	Error             string   `json:"error,omitempty"`
}

func handleCheckUpdates(ctx context.Context, outputJSON bool) error {
	current := strings.TrimSpace(version.Version)
	if current == "" {
		current = "dev"
	}

	res := updateResult{CurrentVersion: current}

	rel, err := fetchLatestRelease(ctx)
	if err != nil {
		res.Error = err.Error()
		if outputJSON {
			return printUpdateJSON(res)
		}
		return err
	}
	latest := trimVersion(rel.TagName)
	res.LatestVersion = latest
	res.ReleaseURL = rel.HTMLURL
	res.UpdateAvailable = current == "dev" || compareVersions(latest, current) > 0

	installPath, _ := os.Executable()
	installPath = filepath.Clean(installPath)
	res.InstallPath = installPath
	res.InstallWritable = isPathWritable(filepath.Dir(installPath))

	if res.UpdateAvailable {
		asset := selectAsset(rel.Assets)
		if asset != nil {
			res.AssetName = asset.Name
			res.DownloadURL = asset.BrowserDownloadURL
			res.SuggestedCommands = buildUpdateCommands(asset, installPath, res.InstallWritable)
		} else {
			res.SuggestedCommands = buildFallbackCommands(rel.HTMLURL, installPath, res.InstallWritable)
		}
	}

	if outputJSON {
		return printUpdateJSON(res)
	}

	fmt.Printf("current version: %s\n", res.CurrentVersion)
	fmt.Printf("latest version:  %s\n", res.LatestVersion)
	if !res.UpdateAvailable {
		fmt.Println("gix is up to date.")
		return nil
	}

	fmt.Printf("%supdate available!%s\n", clrInfo, clrReset)
	fmt.Printf("release page: %s\n", res.ReleaseURL)
	if res.DownloadURL != "" {
		fmt.Printf("direct download: %s (%s)\n", res.DownloadURL, res.AssetName)
	}
	fmt.Println("Suggested commands (copy/paste):")
	for _, cmd := range res.SuggestedCommands {
		fmt.Printf("  %s\n", cmd)
	}
	return nil
}

func fetchLatestRelease(ctx context.Context) (releaseInfo, error) {
	args := []string{"api", fmt.Sprintf("repos/%s/releases/latest", updateRepo)}
	out, err := callGH(ctx, args...)
	if err != nil {
		return releaseInfo{}, err
	}
	var rel releaseInfo
	if err := json.Unmarshal(out, &rel); err != nil {
		return releaseInfo{}, fmt.Errorf("parse release info: %w", err)
	}
	if rel.TagName == "" {
		return releaseInfo{}, errors.New("latest release missing tag_name")
	}
	return rel, nil
}

func selectAsset(assets []releaseAsset) *releaseAsset {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	for _, a := range assets {
		name := strings.ToLower(a.Name)
		if strings.Contains(name, goos) && (strings.Contains(name, goarch) || archAliasMatch(name, goarch)) {
			return &a
		}
	}
	return nil
}

func archAliasMatch(name, arch string) bool {
	switch arch {
	case "amd64":
		return strings.Contains(name, "x86_64") || strings.Contains(name, "x64")
	case "arm64":
		return strings.Contains(name, "aarch64")
	default:
		return false
	}
}

func buildUpdateCommands(asset *releaseAsset, installPath string, writable bool) []string {
	if runtime.GOOS == "windows" {
		return buildWindowsCommands(asset, installPath)
	}
	return buildPosixCommands(asset, installPath, writable)
}

func buildPosixCommands(asset *releaseAsset, installPath string, writable bool) []string {
	tmpDir := filepath.Join(os.TempDir(), "gix-update")
	downloadPath := filepath.Join(tmpDir, asset.Name)
	extractDir := filepath.Join(tmpDir, "extract")
	binaryName := "gix"
	if strings.HasSuffix(strings.ToLower(asset.Name), ".zip") {
		binaryName = "gix"
	}
	binaryPath := filepath.Join(extractDir, binaryName)

	var cmds []string
	cmds = append(cmds,
		fmt.Sprintf("mkdir -p %s %s", shellQuote(tmpDir), shellQuote(extractDir)),
		fmt.Sprintf("curl -L -o %s %s", shellQuote(downloadPath), shellQuote(asset.BrowserDownloadURL)),
	)

	lower := strings.ToLower(asset.Name)
	switch {
	case strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz"):
		cmds = append(cmds, fmt.Sprintf("tar -xzf %s -C %s", shellQuote(downloadPath), shellQuote(extractDir)))
	case strings.HasSuffix(lower, ".zip"):
		cmds = append(cmds, fmt.Sprintf("unzip -o %s -d %s", shellQuote(downloadPath), shellQuote(extractDir)))
	case strings.HasSuffix(lower, ".gz"):
		binaryPath = filepath.Join(tmpDir, "gix")
		cmds = append(cmds,
			fmt.Sprintf("gunzip -c %s > %s", shellQuote(downloadPath), shellQuote(binaryPath)),
		)
	default:
		binaryPath = downloadPath
	}

	cmds = append(cmds, fmt.Sprintf("chmod +x %s", shellQuote(binaryPath)))

	moveCmd := fmt.Sprintf("mv %s %s", shellQuote(binaryPath), shellQuote(installPath))
	if !writable {
		moveCmd = fmt.Sprintf("sudo %s", moveCmd)
	}
	cmds = append(cmds, moveCmd)
	return cmds
}

func buildWindowsCommands(asset *releaseAsset, installPath string) []string {
	tmpBase := `$env:TEMP\\gix-update`
	downloadPath := tmpBase + `\\` + asset.Name
	extractDir := tmpBase + `\\extract`
	binaryPath := extractDir + `\\gix.exe`

	var cmds []string
	cmds = append(cmds,
		fmt.Sprintf(`powershell -Command "New-Item -Force -ItemType Directory %s >$null"`, tmpBase),
		fmt.Sprintf(`powershell -Command "Invoke-WebRequest -OutFile %s %s"`, downloadPath, asset.BrowserDownloadURL),
	)

	lower := strings.ToLower(asset.Name)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		cmds = append(cmds, fmt.Sprintf(`powershell -Command "Expand-Archive -Force %s %s"`, downloadPath, extractDir))
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		// No native tar everywhere; rely on tar from Git or modern Windows.
		cmds = append(cmds, fmt.Sprintf(`tar -xzf %s -C %s`, downloadPath, extractDir))
	default:
		binaryPath = downloadPath
	}

	cmds = append(cmds,
		fmt.Sprintf(`powershell -Command "Move-Item -Force %s %s"`, binaryPath, installPath),
	)
	return cmds
}

func buildFallbackCommands(releaseURL, installPath string, writable bool) []string {
	var cmds []string
	cmds = append(cmds, fmt.Sprintf("Download the latest release from: %s", releaseURL))
	moveCmd := fmt.Sprintf("mv <downloaded_binary> %s", shellQuote(installPath))
	if runtime.GOOS == "windows" {
		moveCmd = fmt.Sprintf(`Move-Item -Force <downloaded_binary> "%s"`, installPath)
	} else if !writable {
		moveCmd = fmt.Sprintf("sudo %s", moveCmd)
	}
	cmds = append(cmds, moveCmd)
	return cmds
}

func printUpdateJSON(res updateResult) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(res)
}

func compareVersions(a, b string) int {
	pa := strings.Split(trimVersion(a), ".")
	pb := strings.Split(trimVersion(b), ".")
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

func isPathWritable(dir string) bool {
	tmp := filepath.Join(dir, ".gix-write-test")
	if err := os.WriteFile(tmp, []byte("ok"), 0o644); err != nil {
		return false
	}
	_ = os.Remove(tmp)
	return true
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

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if strings.ContainsAny(s, " \t\n\"'`$") {
		return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
	}
	return s
}
