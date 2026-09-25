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

// BuildCommand resolves how to run an already-selected gist entry.
// Resolution order: python override, shebang, extension mapping.
// userArgs are appended verbatim after the resolved command.
func BuildCommand(dir string, entry string, userArgs []string, python string) ([]string, string, error) {
	if entry == "" {
		return nil, "", fmt.Errorf("no entry selected to run")
	}

	chosenPath := filepath.Join(dir, entry)

	if python != "" {
		return append([]string{python, chosenPath}, userArgs...), "python override", nil
	}

	if runtime.GOOS != "windows" {
		if cmd, reason, ok := commandFromShebang(chosenPath); ok {
			return append(cmd, userArgs...), reason, nil
		}
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
