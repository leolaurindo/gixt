package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/leolaurindo/gixt/internal/config"
)

// decideExecMode selects the execution directory mode given stored settings and per-run overrides.
func decideExecMode(stored config.ExecMode, forceIsolate, forceCWD bool) (config.ExecMode, error) {
	if forceIsolate && forceCWD {
		return "", errors.New("cannot use --isolate and --cwd together")
	}
	if forceIsolate {
		return config.ExecModeIsolate, nil
	}
	if forceCWD {
		return config.ExecModeCWD, nil
	}
	if stored == "" {
		return config.ExecModeIsolate, nil
	}
	return stored, nil
}

// promptExecMode asks the user which execution directory mode to use when no preference is stored.
func promptExecMode() (config.ExecMode, error) {
	fmt.Printf("%sChoose execution directory%s\n", clrTitle, clrReset)
	fmt.Printf("  [i] isolate (run from temp/cache dir; safer default)\n")
	fmt.Printf("  [c] cwd     (run in current directory; can modify your files)\n")
	fmt.Printf("%sChoice [i/c]: %s", clrPrompt, clrReset)
	var resp string
	if _, err := fmt.Scanln(&resp); err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	resp = strings.ToLower(strings.TrimSpace(resp))
	switch resp {
	case "c", "cwd":
		return config.ExecModeCWD, nil
	default:
		return config.ExecModeIsolate, nil
	}
}
