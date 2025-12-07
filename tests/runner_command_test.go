package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/leolaurindo/gix/internal/runner"
)

func TestBuildCommandPrefersExtension(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.py")
	if err := os.WriteFile(mainPath, []byte("print('ok')"), 0o644); err != nil {
		t.Fatalf("write main file: %v", err)
	}

	cmd, _, reason, err := runner.BuildCommand(dir, "", []string{"main.py"}, []string{"--foo"}, dir)
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

func TestBuildCommandUsesManifestWhenPresent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.sh"), []byte("echo hi"), 0o644); err != nil {
		t.Fatalf("write main file: %v", err)
	}
	manifest := `{"run":"echo hi","env":{"FOO":"BAR"}}`
	if err := os.WriteFile(filepath.Join(dir, "gix.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	cmd, env, reason, err := runner.BuildCommand(dir, "gix.json", []string{"main.sh"}, []string{"ARG"}, dir)
	if err != nil {
		t.Fatalf("BuildCommand error: %v", err)
	}
	if reason != "manifest" {
		t.Fatalf("expected manifest reason, got %q", reason)
	}
	foundRun := false
	for _, c := range cmd {
		if c == "echo hi" {
			foundRun = true
			break
		}
	}
	if !foundRun {
		t.Fatalf("expected manifest run command in %v", cmd)
	}
	if env["FOO"] != "BAR" {
		t.Fatalf("expected env from manifest, got %v", env)
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

	cmd, _, reason, err := runner.BuildCommand(dir, "", []string{"script.txt"}, nil, dir)
	if err != nil {
		t.Fatalf("BuildCommand shebang error: %v", err)
	}
	if reason != "shebang" {
		t.Fatalf("expected shebang reason, got %q", reason)
	}
	if cmd[len(cmd)-1] != scriptPath {
		t.Fatalf("expected script path in command, got %v", cmd)
	}

	unknownPath := filepath.Join(dir, "weird.xyz")
	if err := os.WriteFile(unknownPath, []byte("data"), 0o644); err != nil {
		t.Fatalf("write unknown file: %v", err)
	}
	if _, _, _, err := runner.BuildCommand(dir, "", []string{"weird.xyz"}, nil, dir); err == nil {
		t.Fatalf("expected error for unknown extension")
	}
}
