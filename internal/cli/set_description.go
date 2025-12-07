package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/leolaurindo/gixt/internal/alias"
	"github.com/leolaurindo/gixt/internal/cache"
	"github.com/leolaurindo/gixt/internal/gist"
)

func handleSetDescription(ctx context.Context, target string, desc string) error {
	desc = strings.TrimSpace(desc)
	if desc == "" {
		return errors.New("description cannot be empty")
	}

	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	aliases, err := alias.Load(paths.AliasFile)
	if err != nil {
		return err
	}

	id, ownerHint, _, err := resolveIdentifier(ctx, target, aliases, paths, false, true, normalizeUserPages(0))
	if err != nil {
		return err
	}

	g, err := gist.Fetch(ctx, id, "")
	if err != nil {
		return err
	}

	owner := ownerHint
	if owner == "" {
		owner = gist.GuessOwner(g)
	}
	if owner == "" {
		return errors.New("could not determine gist owner")
	}

	currentUser, err := gist.CurrentUser(ctx)
	if err != nil {
		return fmt.Errorf("detect current user: %w", err)
	}
	if !strings.EqualFold(owner, currentUser) {
		return fmt.Errorf("gist %s is owned by %s (you are %s)", cache.Shorten(id), owner, currentUser)
	}

	updated, err := gist.UpdateDescription(ctx, id, desc)
	if err != nil {
		return err
	}

	if err := refreshIndexAndCache(ctx, paths, updated, false); err != nil {
		return fmt.Errorf("refresh local cache/index: %w", err)
	}
	fmt.Printf("updated description for gist %s\n", cache.Shorten(id))
	return nil
}
