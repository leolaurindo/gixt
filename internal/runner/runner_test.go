package runner

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestBuildCommandUsesExtension(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.py")
	if err := os.WriteFile(mainPath, []byte("print('ok')"), 0o644); err != nil {
		t.Fatalf("write main file: %v", err)
	}

	cmd, reason, err := BuildCommand(dir, "main.py", []string{"--foo"}, "")
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
	cmd, reason, err := BuildCommand(dir, "main.py", []string{"--foo"}, venv)
	if err != nil {
		t.Fatalf("BuildCommand error: %v", err)
	}
	if reason != "python override" || cmd[0] != venv {
		t.Fatalf("expected python override, got %v (reason %q)", cmd, reason)
	}
	if got := cmd[len(cmd)-1]; got != "--foo" {
		t.Fatalf("expected forwarded arg, got %s", got)
	}
}

func TestBuildCommandRespectsShebangAndUnknownExtension(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "script.txt")
	content := "#!/usr/bin/env bash\necho hi\n"
	if err := os.WriteFile(scriptPath, []byte(content), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	cmd, reason, err := BuildCommand(dir, "script.txt", nil, "")
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
	if _, _, err := BuildCommand(dir, "weird.xyz", nil, ""); err == nil {
		t.Fatalf("expected error for unknown extension")
	}
}

func TestBuildCommandPlatformSelection(t *testing.T) {
	cases := []struct {
		name     string
		goos     string
		entry    string
		shebang  string
		launcher bool
		python   string
		prefix   []string
		reason   string
	}{
		{name: "windows env python launcher", goos: "windows", entry: "script", shebang: "#!/usr/bin/env python", launcher: true, prefix: []string{"launcher"}, reason: "shebang"},
		{name: "windows direct versioned python launcher reads flags", goos: "windows", entry: "script.txt", shebang: "#!/usr/bin/python3.12 -u", launcher: true, prefix: []string{"launcher"}, reason: "shebang"},
		{name: "windows python3 fallback", goos: "windows", entry: "script", shebang: "#!/usr/bin/env python3", prefix: []string{"python"}, reason: "shebang"},
		{name: "windows env split flags fallback", goos: "windows", entry: "script", shebang: "#!/usr/bin/env -S python3 -u\r", prefix: []string{"python", "-u"}, reason: "shebang"},
		{name: "windows py unchanged", goos: "windows", entry: "script.py", shebang: "#!/usr/bin/env python3", launcher: true, prefix: []string{"python"}, reason: "extension .py"},
		{name: "windows override", goos: "windows", entry: "script", shebang: "#!/usr/bin/env python3", launcher: true, python: "custom-python", prefix: []string{"custom-python"}, reason: "python override"},
		{name: "windows shell ignored", goos: "windows", entry: "script", shebang: "#!/usr/bin/env bash"},
		{name: "windows non-python extension", goos: "windows", entry: "script.js", shebang: "#!/usr/bin/env node", prefix: []string{"node"}, reason: "extension .js"},
		{name: "windows batch extension", goos: "windows", entry: "script.bat", shebang: "#!/bin/sh", prefix: []string{"cmd", "/C"}, reason: "extension .bat"},
		{name: "windows not python interpreter", goos: "windows", entry: "script", shebang: "#!/usr/bin/env python-tool"},
		{name: "linux python unchanged", goos: "linux", entry: "script", shebang: "#!/usr/bin/env python3", launcher: true, prefix: []string{"/usr/bin/env", "python3"}, reason: "shebang"},
		{name: "macos py shebang unchanged", goos: "darwin", entry: "script.py", shebang: "#!/usr/bin/python3", launcher: true, prefix: []string{"/usr/bin/python3"}, reason: "shebang"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "gist with spaces")
			if err := os.Mkdir(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			scriptPath := filepath.Join(dir, tc.entry)
			if err := os.WriteFile(scriptPath, []byte(tc.shebang+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			lookPath := func(name string) (string, error) {
				if tc.goos != "windows" || tc.reason != "shebang" || name != "py" {
					t.Fatalf("unexpected launcher lookup: %q", name)
				}
				if tc.launcher {
					return "launcher", nil
				}
				return "", os.ErrNotExist
			}
			cmd, reason, err := buildCommand(dir, tc.entry, []string{"--foo", "arg with spaces"}, tc.python, tc.goos, lookPath)
			if tc.prefix == nil {
				if err == nil {
					t.Fatalf("expected unknown-extension error, got %v", cmd)
				}
				return
			}
			want := append(append([]string{}, tc.prefix...), scriptPath, "--foo", "arg with spaces")
			if err != nil || reason != tc.reason || !reflect.DeepEqual(cmd, want) {
				t.Fatalf("got (%v, %q, %v), want (%v, %q, nil)", cmd, reason, err, want, tc.reason)
			}
		})
	}
}
