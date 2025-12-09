package runner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type RunManifest struct {
	Run     string            `json:"run"`
	Env     map[string]string `json:"env"`
	Details string            `json:"details,omitempty"`
	Version string            `json:"version,omitempty"`
}

const DefaultDetails = "No description provided"

func LoadRunManifest(path string) (RunManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RunManifest{}, err
	}
	return LoadRunManifestBytes(data)
}

func LoadRunManifestBytes(data []byte) (RunManifest, error) {
	var m RunManifest
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return RunManifest{}, fmt.Errorf("parse run manifest: %w", err)
	}
	if err := validateRunManifest(m); err != nil {
		return RunManifest{}, err
	}
	return normalizeRunManifest(m), nil
}

func BuildCommand(dir string, manifestPath string, files []string, userArgs []string, execDir string) ([]string, map[string]string, string, error) {
	if manifestPath != "" {
		full := filepath.Join(dir, manifestPath)
		if _, err := os.Stat(full); err == nil {
			m, err := LoadRunManifest(full)
			if err != nil {
				return nil, nil, "", err
			}
			if strings.TrimSpace(m.Run) == "" {
				return nil, nil, "", fmt.Errorf("run manifest %s has empty run field", manifestPath)
			}
			runCmd := m.Run
			if execDir != "" && execDir != dir {
				runCmd = rebaseRunToDir(runCmd, dir)
			}
			shellCmd := shellCommand(runCmd)
			return append(shellCmd, userArgs...), m.Env, "manifest", nil
		}
	}

	if len(files) == 0 {
		return nil, nil, "", fmt.Errorf("no files in gist to run")
	}

	chosen := selectFile(files)
	chosenPath := filepath.Join(dir, chosen)

	if cmd, reason, ok := commandFromShebang(chosenPath); ok {
		return append(cmd, userArgs...), nil, reason, nil
	}

	cmd, reason, err := commandFromExtension(chosenPath)
	if err != nil {
		return nil, nil, "", err
	}
	return append(cmd, userArgs...), nil, reason, nil
}

func selectFile(files []string) string {
	if len(files) == 0 {
		return ""
	}
	mainCandidates := filterByPrefix(files, "main.")
	if chosen := choosePlatformSpecific(mainCandidates); chosen != "" {
		return chosen
	}
	if len(mainCandidates) > 0 {
		return mainCandidates[0]
	}
	indexCandidates := filterByPrefix(files, "index.")
	if chosen := choosePlatformSpecific(indexCandidates); chosen != "" {
		return chosen
	}
	if len(indexCandidates) > 0 {
		return indexCandidates[0]
	}
	if chosen := choosePlatformSpecific(files); chosen != "" {
		return chosen
	}
	return files[0]
}

func filterByPrefix(files []string, prefix string) []string {
	var out []string
	for _, f := range files {
		if strings.HasPrefix(strings.ToLower(filepath.Base(f)), prefix) {
			out = append(out, f)
		}
	}
	return out
}

func choosePlatformSpecific(files []string) string {
	if len(files) == 0 {
		return ""
	}
	allowed := platformAllowedExts()
	preferred := platformPreferredExts()

	type info struct {
		files        []string
		exts         []string
		allAllowed   bool
		preferredHit []string
	}
	byBase := map[string]*info{}
	var order []string
	for _, f := range files {
		base := strings.ToLower(strings.TrimSuffix(filepath.Base(f), filepath.Ext(f)))
		ext := strings.ToLower(filepath.Ext(f))
		if _, ok := byBase[base]; !ok {
			byBase[base] = &info{allAllowed: true}
			order = append(order, base)
		}
		entry := byBase[base]
		entry.files = append(entry.files, f)
		entry.exts = append(entry.exts, ext)
		if !allowed[ext] {
			entry.allAllowed = false
		}
		if preferred[ext] {
			entry.preferredHit = append(entry.preferredHit, f)
		}
	}
	for _, base := range order {
		info := byBase[base]
		if !info.allAllowed {
			continue
		}
		if len(info.preferredHit) == 1 {
			return info.preferredHit[0]
		}
	}
	return ""
}

func platformAllowedExts() map[string]bool {
	return map[string]bool{
		".bat":  true,
		".cmd":  true,
		".ps1":  true,
		".sh":   true,
		".bash": true,
		".zsh":  true,
	}
}

func platformPreferredExts() map[string]bool {
	if runtime.GOOS == "windows" {
		return map[string]bool{
			".bat": true,
			".cmd": true,
			".ps1": true,
		}
	}
	return map[string]bool{
		".sh":   true,
		".bash": true,
		".zsh":  true,
	}
}

func commandFromShebang(path string) ([]string, string, bool) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, "", false
	}
	defer fh.Close()
	scanner := bufio.NewScanner(fh)
	if !scanner.Scan() {
		return nil, "", false
	}
	line := scanner.Text()
	if !strings.HasPrefix(line, "#!") {
		return nil, "", false
	}
	trimmed := strings.TrimSpace(strings.TrimPrefix(line, "#!"))
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return nil, "", false
	}
	return append(parts, path), "shebang", true
}

func commandFromExtension(path string) ([]string, string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".sh":
		return []string{"sh", path}, "extension .sh", nil
	case ".ps1":
		return []string{"powershell", "-ExecutionPolicy", "Bypass", "-File", path}, "extension .ps1", nil
	case ".bat", ".cmd":
		if runtime.GOOS == "windows" {
			return []string{"cmd", "/C", path}, "extension .bat", nil
		}
		return []string{path}, "extension .bat", nil
	case ".py":
		return []string{"python", path}, "extension .py", nil
	case ".js":
		return []string{"node", path}, "extension .js", nil
	case ".ts":
		return []string{"npx", "ts-node", path}, "extension .ts", nil
	case ".go":
		return []string{"go", "run", path}, "extension .go", nil
	case ".rb":
		return []string{"ruby", path}, "extension .rb", nil
	case ".pl":
		return []string{"perl", path}, "extension .pl", nil
	case ".php":
		return []string{"php", path}, "extension .php", nil
	}
	return nil, "", fmt.Errorf("cannot determine how to run %s (unknown extension)", filepath.Base(path))
}

func shellCommand(cmd string) []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/C", cmd}
	}
	return []string{"sh", "-c", cmd}
}

func rebaseRunToDir(run string, dir string) string {
	if strings.TrimSpace(run) == "" {
		return run
	}
	parts := strings.Fields(run)
	if len(parts) == 0 {
		return run
	}
	for i, p := range parts {
		if strings.HasPrefix(p, "-") {
			continue
		}
		if filepath.IsAbs(p) {
			continue
		}
		candidate := filepath.Join(dir, p)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			parts[i] = candidate
			break
		}
	}
	return strings.Join(parts, " ")
}

func validateRunManifest(m RunManifest) error {
	run := strings.TrimSpace(m.Run)
	if run == "" {
		return fmt.Errorf("run manifest has empty run field")
	}
	if strings.ContainsAny(run, "\r\n") {
		return fmt.Errorf("run manifest run field must not contain newlines")
	}
	if len(run) > 4096 {
		return fmt.Errorf("run manifest run field too long")
	}
	for k := range m.Env {
		if strings.TrimSpace(k) == "" {
			return fmt.Errorf("run manifest env contains empty key")
		}
		if len(k) > 256 {
			return fmt.Errorf("run manifest env key too long: %s", k)
		}
		if strings.ContainsAny(k, "\r\n") {
			return fmt.Errorf("run manifest env key contains newline: %s", k)
		}
	}
	if len(strings.TrimSpace(m.Details)) > 4096 {
		return fmt.Errorf("run manifest details too long")
	}
	if len(strings.TrimSpace(m.Version)) > 256 {
		return fmt.Errorf("run manifest version too long")
	}
	return nil
}

func normalizeRunManifest(m RunManifest) RunManifest {
	if m.Env == nil {
		m.Env = map[string]string{}
	}
	if strings.TrimSpace(m.Details) == "" {
		m.Details = DefaultDetails
	}
	return m
}
