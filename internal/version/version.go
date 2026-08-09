package version

import "runtime/debug"

// Version is set at build time via -ldflags "-X github.com/leolaurindo/gixt/internal/version.Version=vX.Y.Z".
// Falls back to the module version recorded by `go install` (e.g. v0.2.1).
var Version = "dev"

func init() {
	if Version != "dev" {
		return
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			Version = v
		}
	}
}
