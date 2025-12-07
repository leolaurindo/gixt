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
	if err := json.Unmarshal(data, &m); err != nil {
		return RunManifest{}, fmt.Errorf("parse run manifest: %w", err)
	}
	if m.Env == nil {
		m.Env = map[string]string{}
	}
	if strings.TrimSpace(m.Details) == "" {
		m.Details = DefaultDetails
	}
	return m, nil
}

func BuildCommand(dir string, manifestPath string, files []string, userArgs []string) ([]string, map[string]string, string, error) {
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
			shellCmd := shellCommand(m.Run)
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
	for _, f := range files {
		name := strings.ToLower(filepath.Base(f))
		if strings.HasPrefix(name, "main.") {
			return f
		}
	}
	for _, f := range files {
		name := strings.ToLower(filepath.Base(f))
		if strings.HasPrefix(name, "index.") {
			return f
		}
	}
	return files[0]
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
