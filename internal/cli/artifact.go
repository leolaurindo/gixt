package cli

import (
	"context"
	"os"

	"github.com/leolaurindo/gixt/internal/cache"
	"github.com/leolaurindo/gixt/internal/config"
	"github.com/leolaurindo/gixt/internal/gist"
)

type acquiredArtifact struct {
	resolved  ResolvedTarget
	workDir   string
	meta      cache.Meta
	fromCache bool
	pin       string
	cleanup   func()
}

func acquireArtifact(ctx context.Context, paths config.Paths, client *gist.Client, target string, offline, noCache bool, requestedRef string) (acquiredArtifact, error) {
	resolved, err := ResolveTarget(ctx, target, paths, !offline)
	if err != nil {
		return acquiredArtifact{}, err
	}

	pin, err := pinnedRef(paths, resolved.GistID)
	if err != nil {
		return acquiredArtifact{}, err
	}
	ref := requestedRef
	if ref == "" {
		ref = pin
	}

	workDir, meta, fromCache, err := obtain(ctx, client, paths, resolved.GistID, ref, offline, noCache)
	if err != nil {
		return acquiredArtifact{}, err
	}
	artifact := acquiredArtifact{
		resolved:  resolved,
		workDir:   workDir,
		meta:      meta,
		fromCache: fromCache,
		pin:       pin,
	}
	if noCache {
		artifact.cleanup = func() { _ = os.RemoveAll(workDir) }
	}
	return artifact, nil
}

func (a acquiredArtifact) close() {
	if a.cleanup != nil {
		a.cleanup()
	}
}
