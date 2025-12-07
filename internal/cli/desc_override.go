package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/leolaurindo/gixt/internal/alias"
	"github.com/leolaurindo/gixt/internal/indexdesc"
)

func handleDescOverride(ctx context.Context, args []string) error {
	if len(args) == 0 {
		args = []string{"list"}
	}
	action := strings.ToLower(args[0])

	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	aliases, _ := alias.Load(paths.AliasFile)
	overrides, err := indexdesc.Load(paths.IndexDescFile)
	if err != nil {
		return err
	}

	switch action {
	case "list":
		names := indexdesc.Sorted(overrides)
		if len(names) == 0 {
			fmt.Println("no description overrides set")
			return nil
		}
		for _, id := range names {
			fmt.Printf("%s -> %s\n", id, overrides[id])
		}
		return nil
	case "add":
		if len(args) < 3 {
			return errors.New("usage: gixt index-description add <id|name> <description>")
		}
		target := args[1]
		desc := indexdesc.Normalize(strings.Join(args[2:], " "))
		if desc == "" {
			return errors.New("description cannot be empty")
		}
		id, _, _, err := resolveIdentifier(ctx, target, aliases, paths, false, true, normalizeUserPages(0))
		if err != nil {
			return err
		}
		overrides[id] = desc
		if err := indexdesc.Save(paths.IndexDescFile, overrides); err != nil {
			return err
		}
		fmt.Printf("set description override for %s\n", id)
		return nil
	case "remove":
		if len(args) < 2 {
			return errors.New("usage: gixt index-description remove <id|name>")
		}
		target := args[1]
		id, _, _, err := resolveIdentifier(ctx, target, aliases, paths, false, true, normalizeUserPages(0))
		if err != nil {
			return err
		}
		delete(overrides, id)
		if err := indexdesc.Save(paths.IndexDescFile, overrides); err != nil {
			return err
		}
		fmt.Printf("removed description override for %s\n", id)
		return nil
	default:
		return errors.New("usage: gixt index-description [list|add|remove] ...")
	}
}
