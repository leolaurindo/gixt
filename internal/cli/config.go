package cli

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/leolaurindo/gixt/internal/config"
)

func newConfigCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "config",
		Short: "manage gixt settings",
		Args:  cobra.NoArgs,
	}
	c.AddCommand(
		&cobra.Command{
			Use:   "get [key]",
			Short: "print a setting (default: all)",
			Args:  cobra.MaximumNArgs(1),
			RunE:  configGet,
		},
		&cobra.Command{
			Use:   "set <key> <value>",
			Short: "set a setting",
			Args:  cobra.ExactArgs(2),
			RunE:  configSet,
		},
		&cobra.Command{
			Use:   "unset <key>",
			Short: "reset a setting to its default",
			Args:  cobra.ExactArgs(1),
			RunE:  configUnset,
		},
	)
	return c
}

func configGet(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	s, err := config.LoadSettings(paths.Settings)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		fmt.Printf("trust.mine\t%v\n", s.Mine)
		return nil
	}
	switch args[0] {
	case "trust.mine":
		fmt.Printf("%v\n", s.Mine)
	default:
		return fmt.Errorf("unknown setting %q (known: trust.mine)", args[0])
	}
	return nil
}

func configSet(cmd *cobra.Command, args []string) error {
	key, value := args[0], args[1]
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	s, err := config.LoadSettings(paths.Settings)
	if err != nil {
		return err
	}
	switch key {
	case "trust.mine":
		v, err := strconv.ParseBool(strings.TrimSpace(value))
		if err != nil {
			return errors.New("trust.mine expects true or false")
		}
		s.Mine = v
	default:
		return fmt.Errorf("unknown setting %q (known: trust.mine)", key)
	}
	if err := config.SaveSettings(paths.Settings, s); err != nil {
		return err
	}
	logf("set %s to %v", key, s.Mine)
	return nil
}

func configUnset(cmd *cobra.Command, args []string) error {
	key := args[0]
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	s, err := config.LoadSettings(paths.Settings)
	if err != nil {
		return err
	}
	switch key {
	case "trust.mine":
		s.Mine = true
	default:
		return fmt.Errorf("unknown setting %q (known: trust.mine)", key)
	}
	if err := config.SaveSettings(paths.Settings, s); err != nil {
		return err
	}
	logf("reset %s to default", key)
	return nil
}
