package cli

import (
	"context"
	"strings"

	"github.com/leolaurindo/gixt/internal/config"
	"github.com/leolaurindo/gixt/internal/gist"
)

func trustDecision(ctx context.Context, settings config.Settings, owner string, gistID string, yesFlag bool) bool {
	if yesFlag {
		return true
	}
	if settings.Mode == config.TrustAll {
		return true
	}
	if settings.TrustedGists[gistID] {
		return true
	}
	if settings.TrustedOwners[strings.ToLower(owner)] {
		return true
	}
	if settings.Mode == config.TrustMine && owner != "" {
		if login, err := gist.CurrentUser(ctx); err == nil && strings.EqualFold(login, owner) {
			return true
		}
	}
	return false
}
