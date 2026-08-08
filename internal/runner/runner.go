package runner

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// BuildCommand resolves the command to run for a gist's files.
// Resolution order: python override, shebang, extension mapping.
// userArgs are appended verbatim after the resolved command.
func BuildCommand(dir string, files []string, userArgs []string, python string) ([]string, string, error) {
	if len(files) == 0 {
		return nil, "", fmt.Errorf("no files in gist to run")
	}

	chosen := selectFile(files)
	chosenPath := filepath.Join(dir, chosen)

	if python != "" {
		return append([]string{python, chosenPath}, userArgs...), "python override", nil
	}

	if cmd, reason, ok := commandFromShebang(chosenPath); ok {
		return append(cmd, userArgs...), reason, nil
	}

	cmd, reason, err := commandFromExtension(chosenPath)
	if err != nil {
		return nil, "", err
	}
	return append(cmd, userArgs...), reason, nil
}

// Execute runs the resolved command in dir, wiring stdin/stdout/stderr and
// propagating exit codes and signals.
func Execute(ctx context.Context, dir string, cmd []string) error {
	c := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	c.Dir = dir
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Cancel = func() error {
		if c.Process != nil {
			return c.Process.Signal(os.Interrupt)
		}
		return nil
	}
	c.WaitDelay = 5 * time.Second

	if err := c.Run(); err != nil {
		return fmt.Errorf("exec %s: %w", cmd[0], err)
	}
	return nil
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
