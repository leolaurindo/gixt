package cli

import (
	"context"
	"testing"

	"github.com/leolaurindo/gixt/internal/config"
)

func TestTrustDecisionOrdering(t *testing.T) {
	settings := config.Settings{
		Mode:          config.TrustNever,
		TrustedOwners: map[string]bool{"owner1": true},
		TrustedGists:  map[string]bool{"gist1": true},
	}

	if !trustDecision(context.Background(), settings, "any", "any", true) {
		t.Fatalf("expected yesFlag to trust immediately")
	}

	settings.Mode = config.TrustAll
	if !trustDecision(context.Background(), settings, "any", "any", false) {
		t.Fatalf("expected mode=all to trust")
	}

	settings.Mode = config.TrustNever
	if !trustDecision(context.Background(), settings, "owner1", "other", false) {
		t.Fatalf("expected trusted owner to trust")
	}
	if !trustDecision(context.Background(), settings, "other", "gist1", false) {
		t.Fatalf("expected trusted gist to trust")
	}

	if trustDecision(context.Background(), settings, "other", "other", false) {
		t.Fatalf("expected untrusted inputs to require prompt")
	}
}
