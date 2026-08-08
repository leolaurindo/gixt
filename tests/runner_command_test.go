package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/leolaurindo/gixt/internal/runner"
)

func TestBuildCommandUsesExtension(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.py")
	if err := os.WriteFile(mainPath, []byte("print('ok')"), 0o644); err != nil {
		t.Fatalf("write main file: %v", err)
	}

	cmd, reason, err := runner.BuildCommand(dir, []string{"main.py"}, []string{"--foo"}, "")
	if err != nil {
		t.Fatalf("BuildCommand error: %v", err)
	}
	if reason == "" || cmd[0] != "python" {
		t.Fatalf("expected python command, got %v (reason %q)", cmd, reason)
	}
	if got := cmd[len(cmd)-1]; got != "--foo" {
		t.Fatalf("expected forwarded arg, got %s", got)
	}
}

func TestBuildCommandPythonOverride(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.py")
	if err := os.WriteFile(mainPath, []byte("print('ok')"), 0o644); err != nil {
		t.Fatalf("write main file: %v", err)
	}

	venv := filepath.Join(dir, ".venv", "bin", "python")
	cmd, reason, err := runner.BuildCommand(dir, []string{"main.py"}, []string{"ARG"}, venv)
	if err != nil {
		t.Fatalf("BuildCommand error: %v", err)
	}
	if reason != "python override" || cmd[0] != venv {
		t.Fatalf("expected python override, got %v (reason %q)", cmd, reason)
	}
	if cmd[len(cmd)-1] != "ARG" {
		t.Fatalf("expected forwarded arg, got %v", cmd)
	}
}

func TestBuildCommandRespectsShebangAndUnknownExtension(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "script.txt")
	content := "#!/usr/bin/env bash\necho hi\n"
	if err := os.WriteFile(scriptPath, []byte(content), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	cmd, reason, err := runner.BuildCommand(dir, []string{"script.txt"}, nil, "")
	if runtime.GOOS == "windows" {
		if err == nil {
			t.Fatalf("expected unknown-extension error on windows (shebang skipped), got %v", cmd)
		}
	} else {
		if err != nil {
			t.Fatalf("BuildCommand shebang error: %v", err)
		}
		if reason != "shebang" {
			t.Fatalf("expected shebang reason, got %q", reason)
		}
		if cmd[len(cmd)-1] != scriptPath {
			t.Fatalf("expected script path in command, got %v", cmd)
		}
	}

	unknownPath := filepath.Join(dir, "weird.xyz")
	if err := os.WriteFile(unknownPath, []byte("data"), 0o644); err != nil {
		t.Fatalf("write unknown file: %v", err)
	}
	if _, _, err := runner.BuildCommand(dir, []string{"weird.xyz"}, nil, ""); err == nil {
		t.Fatalf("expected error for unknown extension")
	}
}
