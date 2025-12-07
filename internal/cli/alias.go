package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/leolaurindo/gixt/internal/alias"
	"github.com/leolaurindo/gixt/internal/gist"
)

func handleAlias(args []string) error {
	if len(args) == 0 {
		fmt.Println("alias commands: add <name> <gist-id>, list, remove <name>")
		return nil
	}

	paths, err := ensurePaths("")
	if err != nil {
		return err
	}

	aliases, err := alias.Load(paths.AliasFile)
	if err != nil {
		return err
	}

	switch args[0] {
	case "add":
		if len(args) < 3 {
			return errors.New("usage: gixt alias add <name> <gist-id>")
		}
		name := args[1]
		id := gist.ExtractID(args[2])
		aliases[name] = id
		if err := alias.Save(paths.AliasFile, aliases); err != nil {
			return err
		}
		fmt.Printf("alias %s -> %s saved\n", name, id)
		return nil
	case "list":
		alias.PrintList(os.Stdout, aliases)
		return nil
	case "remove":
		if len(args) < 2 {
			return errors.New("usage: gixt alias remove <name>")
		}
		name := args[1]
		if _, ok := aliases[name]; !ok {
			return fmt.Errorf("alias %s not found", name)
		}
		delete(aliases, name)
		if err := alias.Save(paths.AliasFile, aliases); err != nil {
			return err
		}
		fmt.Printf("alias %s removed\n", name)
		return nil
	default:
		return fmt.Errorf("unknown alias command: %s", args[0])
	}
}
