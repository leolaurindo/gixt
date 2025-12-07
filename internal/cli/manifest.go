package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/leolaurindo/gix/internal/alias"
	"github.com/leolaurindo/gix/internal/cache"
	"github.com/leolaurindo/gix/internal/config"
	"github.com/leolaurindo/gix/internal/gist"
	"github.com/leolaurindo/gix/internal/index"
	"github.com/leolaurindo/gix/internal/runner"
)

type manifestOpts struct {
	name    string
	create  bool
	edit    bool
	upload  bool
	view    bool
	gist    string
	run     string
	env     []string
	details string
	version string
	force   bool
}

func handleManifest(ctx context.Context, opts manifestOpts) error {
	filename := opts.name
	if filename == "" {
		filename = "gix.json"
	}
	targetPath := filename
	if !filepath.IsAbs(targetPath) {
		wd, _ := os.Getwd()
		targetPath = filepath.Join(wd, targetPath)
	}

	if !opts.create && !opts.edit && !opts.upload {
		if opts.view && opts.gist != "" {
			return viewRemoteManifest(ctx, opts.gist, filename)
		}
		return errors.New("usage: gix manifest [--create|--edit|--upload] [--name <file>] [--run ... --env KEY=VAL ... --details ... --version ...] [--gist <id|name>]")
	}
	if opts.view {
		if opts.create || opts.edit || opts.upload {
			return errors.New("--view cannot be combined with --create/--edit/--upload")
		}
		if opts.gist == "" {
			return errors.New("--view requires --gist <id|name>")
		}
		return viewRemoteManifest(ctx, opts.gist, filename)
	}
	if opts.create && opts.edit {
		return errors.New("choose either --create or --edit, not both")
	}

	manifest := runner.RunManifest{Env: map[string]string{}}
	exists := fileExists(targetPath)
	baseLoaded := false

	// Load base manifest when editing or uploading without create/edit.
	if exists && !opts.create && !opts.edit {
		if opts.upload {
			rm, err := runner.LoadRunManifest(targetPath)
			if err != nil {
				return err
			}
			manifest = rm
			baseLoaded = true
		} else {
			return fmt.Errorf("manifest %s already exists; use --edit or --upload", targetPath)
		}
	}

	if opts.edit {
		if exists {
			rm, err := runner.LoadRunManifest(targetPath)
			if err != nil {
				return err
			}
			manifest = rm
			baseLoaded = true
		} else if opts.upload && opts.gist != "" {
			rm, err := fetchRemoteManifest(ctx, opts.gist, filename)
			if err != nil {
				return err
			}
			manifest = rm
			baseLoaded = true
		} else if !opts.force {
			return fmt.Errorf("manifest %s does not exist (use --create or --force to write a new one)", targetPath)
		}
	}

	if manifest.Env == nil {
		manifest.Env = map[string]string{}
	}

	// Apply overrides
	if opts.run != "" {
		manifest.Run = opts.run
	}
	if len(opts.env) > 0 {
		manifest.Env = parseEnv(opts.env, manifest.Env)
	}
	if opts.details != "" {
		manifest.Details = opts.details
	}
	if opts.version != "" {
		manifest.Version = opts.version
	}
	if strings.TrimSpace(manifest.Details) == "" {
		manifest.Details = runner.DefaultDetails
	}

	// Decide whether to write locally
	shouldWrite := (opts.create || opts.edit) && !opts.upload
	if shouldWrite {
		if exists && !opts.force {
			ok, err := confirm(fmt.Sprintf("%s exists. Overwrite?", targetPath))
			if err != nil {
				return err
			}
			if !ok {
				return errors.New("aborted")
			}
		}
		if err := saveRunManifest(targetPath, manifest); err != nil {
			return err
		}
		fmt.Printf("wrote manifest to %s\n", targetPath)
	}

	if opts.upload {
		// If we haven't loaded anything yet, rely on the on-disk manifest.
		if !baseLoaded && !opts.create && !opts.edit {
			rm, err := runner.LoadRunManifest(targetPath)
			if err != nil {
				return err
			}
			manifest = rm
		}
		if err := uploadManifest(ctx, manifest, filename, opts.gist); err != nil {
			return err
		}
	}
	return nil
}

// applyManifestArgs allows positional overrides like:
// gix manifest --edit version 0.0.1
// gix manifest --create run "python app.py"
// gix manifest --edit env KEY=VAL env OTHER=VAL
func applyManifestArgs(args []string, opts *manifestOpts) error {
	for i := 0; i < len(args); {
		key := strings.ToLower(args[i])
		if i+1 >= len(args) {
			return fmt.Errorf("missing value for %q (use --%s <value>)", key, key)
		}
		val := args[i+1]
		switch key {
		case "version":
			opts.version = val
		case "run":
			opts.run = val
		case "details":
			opts.details = val
		case "name":
			opts.name = val
		case "env":
			opts.env = append(opts.env, val)
		default:
			return fmt.Errorf("unknown manifest argument %q (supported: version, run, details, name, env)", key)
		}
		i += 2
	}
	return nil
}

func saveRunManifest(path string, m runner.RunManifest) error {
	buf, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}

func uploadManifest(ctx context.Context, m runner.RunManifest, fileName string, target string) error {
	if target == "" {
		return errors.New("upload requires --gist <id|name|owner/name>")
	}
	baseName := filepath.Base(fileName)
	paths, err := ensurePaths("")
	if err != nil {
		return err
	}
	aliases, _ := alias.Load(paths.AliasFile)
	id, _, _, err := resolveIdentifier(ctx, target, aliases, paths, false, true, normalizeUserPages(0))
	if err != nil {
		return err
	}

	currentUser, err := gist.CurrentUser(ctx)
	if err != nil {
		return fmt.Errorf("detect current user: %w", err)
	}
	g, err := gist.Fetch(ctx, id, "")
	if err != nil {
		return err
	}
	if strings.TrimSpace(gist.GuessOwner(g)) == "" || !strings.EqualFold(gist.GuessOwner(g), currentUser) {
		return fmt.Errorf("gist %s is not owned by %s", id, currentUser)
	}

	ok, err := confirm(fmt.Sprintf("Upload manifest to gist %s (owner %s)? This will overwrite %s if it exists in the gist.", cache.Shorten(id), currentUser, baseName))
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("aborted")
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}

	files := map[string]string{
		baseName: string(data),
	}
	updated, err := gist.UpdateFiles(ctx, id, files)
	if err != nil {
		return err
	}

	if err := refreshIndexAndCache(ctx, paths, updated, true); err != nil {
		return err
	}
	fmt.Printf("uploaded %s to gist %s\n", baseName, id)
	return nil
}

func fetchRemoteManifest(ctx context.Context, target string, manifestName string) (runner.RunManifest, error) {
	paths, err := ensurePaths("")
	if err != nil {
		return runner.RunManifest{}, err
	}
	aliases, _ := alias.Load(paths.AliasFile)
	id, _, _, err := resolveIdentifier(ctx, target, aliases, paths, false, true, normalizeUserPages(0))
	if err != nil {
		return runner.RunManifest{}, err
	}
	g, err := gist.Fetch(ctx, id, "")
	if err != nil {
		return runner.RunManifest{}, err
	}
	want := strings.ToLower(filepath.Base(manifestName))
	var found *gist.File
	var rawURL string
	for name, f := range g.Files {
		if strings.ToLower(filepath.Base(name)) == want {
			// Need a copy to take address
			fileCopy := f
			found = &fileCopy
			rawURL = f.RawURL
			break
		}
	}
	if found == nil {
		return runner.RunManifest{}, fmt.Errorf("gist %s does not contain %s", id, manifestName)
	}

	data := []byte(found.Content)
	if found.Truncated || len(data) == 0 {
		resp, err := http.Get(rawURL)
		if err != nil {
			return runner.RunManifest{}, fmt.Errorf("download manifest %s: %w", manifestName, err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return runner.RunManifest{}, fmt.Errorf("read manifest %s: %w", manifestName, err)
		}
		if resp.StatusCode >= 300 {
			return runner.RunManifest{}, fmt.Errorf("download manifest %s: http %d", manifestName, resp.StatusCode)
		}
		data = body
	}
	return runner.LoadRunManifestBytes(data)
}

func viewRemoteManifest(ctx context.Context, target string, manifestName string) error {
	if strings.TrimSpace(manifestName) == "" {
		manifestName = "gix.json"
	}
	m, err := fetchRemoteManifest(ctx, target, manifestName)
	if err != nil {
		return err
	}
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func refreshIndexAndCache(ctx context.Context, paths config.Paths, g gist.Gist, forceUpdate bool) error {
	// refresh index entry
	idx, _ := index.Load(paths.IndexFile)
	found := false
	for i, e := range idx.Entries {
		if e.ID == g.ID {
			idx.Entries[i] = toIndexEntryFromGist(g)
			found = true
			break
		}
	}
	if !found {
		idx.Entries = append(idx.Entries, toIndexEntryFromGist(g))
	}
	sortIndexEntries(idx.Entries)
	idx.GeneratedAt = time.Now()
	if err := index.Save(paths.IndexFile, idx); err != nil {
		return err
	}

	// refresh cache with latest gist content if already cached or if forceUpdate requested
	sha := g.LatestVersion()
	if sha == "" {
		return nil
	}
	workDir := cache.Dir(paths.CacheDir, g.ID, sha)
	if forceUpdate || cache.PathExists(workDir) {
		if err := cache.EnsureDir(workDir); err != nil {
			return err
		}
		if _, _, err := materializeFiles(g, workDir, true); err != nil {
			return err
		}
		manifest := cache.Manifest{
			GistID:      g.ID,
			SHA:         sha,
			Description: g.Description,
			Owner:       gist.GuessOwner(g),
			Files:       mapFileNames(g.Files),
			Source:      g.HTMLURL,
			CreatedAt:   time.Now(),
		}
		if err := cache.SaveManifest(cache.ManifestPath(workDir), manifest); err != nil {
			return err
		}
	}
	return nil
}

func parseEnv(values []string, base map[string]string) map[string]string {
	if base == nil {
		base = map[string]string{}
	}
	for _, v := range values {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := parts[1]
		if key == "" {
			continue
		}
		base[key] = val
	}
	return base
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func confirm(prompt string) (bool, error) {
	fmt.Printf("%s%s%s [y/N]: ", clrPrompt, prompt, clrReset)
	var resp string
	if _, err := fmt.Scanln(&resp); err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	resp = strings.ToLower(strings.TrimSpace(resp))
	return resp == "y" || resp == "yes", nil
}
