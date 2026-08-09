package cli

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leolaurindo/gixt/internal/config"
)

func TestResolveTargetOfflineSkipsLiveLookup(t *testing.T) {
	paths := config.Paths{KnownFile: filepath.Join(t.TempDir(), "missing.json")}
	_, err := resolveTarget(context.Background(), "owner/tool", paths, false)
	if err == nil || !strings.Contains(err.Error(), "offline") {
		t.Fatalf("expected offline resolution error, got %v", err)
	}
}
