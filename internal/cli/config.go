package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/leolaurindo/gixt/internal/config"
)

func handleConfigTrust(_ context.Context, mode string, owners []string, removeOwners []string, removeGists []string, clearOwners, clearGists, reset, show bool) error {
	paths, settings, err := ensurePathsAndSettings("")
	if err != nil {
		return err
	}

	if reset {
		settings.Mode = config.TrustNever
		settings.TrustedOwners = map[string]bool{}
		settings.TrustedGists = map[string]bool{}
		fmt.Printf("%scleared stored trust decisions (mode=never).%s\n", clrWarn, clrReset)
	}
	if settings.TrustedOwners == nil {
		settings.TrustedOwners = map[string]bool{}
	}
	if settings.TrustedGists == nil {
		settings.TrustedGists = map[string]bool{}
	}

	if clearOwners {
		settings.TrustedOwners = map[string]bool{}
	}
	if clearGists {
		settings.TrustedGists = map[string]bool{}
	}
	for _, o := range owners {
		settings.TrustedOwners[strings.ToLower(o)] = true
	}
	for _, o := range removeOwners {
		delete(settings.TrustedOwners, strings.ToLower(o))
	}
	for _, g := range removeGists {
		delete(settings.TrustedGists, strings.ToLower(g))
	}
	if mode != "" {
		switch strings.ToLower(mode) {
		case string(config.TrustNever):
			settings.Mode = config.TrustNever
		case string(config.TrustMine):
			settings.Mode = config.TrustMine
		case string(config.TrustAll):
			settings.Mode = config.TrustAll
		default:
			return fmt.Errorf("unknown mode %s (expected never|mine|all)", mode)
		}
	}
	if err := config.SaveSettings(paths.Settings, settings); err != nil {
		return err
	}

	if show || mode != "" || len(owners) > 0 || len(removeOwners) > 0 || len(removeGists) > 0 || clearOwners || clearGists || reset {
		fmt.Println(colorize("Trust configuration:", clrTitle))
		fmt.Printf("  mode: %s\n", settings.Mode)
		if len(settings.TrustedOwners) > 0 {
			var list []string
			for o := range settings.TrustedOwners {
				list = append(list, o)
			}
			sort.Strings(list)
			fmt.Printf("  trusted owners: %s\n", strings.Join(list, ", "))
		} else {
			fmt.Println("  trusted owners: (none)")
		}
		if len(settings.TrustedGists) > 0 {
			fmt.Printf("  trusted gists: %d stored\n", len(settings.TrustedGists))
		} else {
			fmt.Println("  trusted gists: (none)")
		}
	}
	return nil
}

func handleConfigCache(mode string, show bool) error {
	paths, settings, err := ensurePathsAndSettings("")
	if err != nil {
		return err
	}

	if mode != "" {
		switch strings.ToLower(mode) {
		case string(config.CacheModeCache):
			settings.CacheMode = config.CacheModeCache
		case string(config.CacheModeDefault):
			settings.CacheMode = config.CacheModeDefault
		default:
			return fmt.Errorf("unknown cache mode %s (expected cache|never)", mode)
		}
		if err := config.SaveSettings(paths.Settings, settings); err != nil {
			return err
		}
	}

	if show || mode != "" {
		fmt.Printf("Cache mode: %s\n", settings.CacheMode)
	}
	return nil
}

func handleConfigExec(mode string, show bool) error {
	paths, settings, err := ensurePathsAndSettings("")
	if err != nil {
		return err
	}

	if mode != "" {
		switch strings.ToLower(mode) {
		case string(config.ExecModeIsolate):
			settings.ExecMode = config.ExecModeIsolate
		case string(config.ExecModeCWD):
			settings.ExecMode = config.ExecModeCWD
		default:
			return fmt.Errorf("unknown execution mode %s (expected isolate|cwd)", mode)
		}
		if err := config.SaveSettings(paths.Settings, settings); err != nil {
			return err
		}
	}

	if show || mode != "" {
		modeOut := settings.ExecMode
		if modeOut == "" {
			modeOut = config.ExecModeIsolate
		}
		fmt.Printf("Execution mode: %s\n", modeOut)
	}
	return nil
}
