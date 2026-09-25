package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/leolaurindo/gixt/internal/config"
	"github.com/leolaurindo/gixt/internal/gist"
	"github.com/leolaurindo/gixt/internal/known"
)

// ResolvedTarget preserves the gist identity and an exact filename inferred
// from the user's target, when there is one.
type ResolvedTarget struct {
	GistID        string
	RequestedFile string
}

// TargetNotFoundError identifies a target-resolution miss without conflating
// it with ambiguity, authentication, or network errors.
type TargetNotFoundError struct {
	Target  string
	Detail  string
	Suggest bool
}

func (e *TargetNotFoundError) Error() string {
	if e.Detail != "" {
		return e.Detail
	}
	return fmt.Sprintf("could not resolve %q", e.Target)
}

// AmbiguousTargetError reports all known candidates that can disambiguate a
// target. It is deliberately distinct from TargetNotFoundError.
type AmbiguousTargetError struct {
	Target     string
	Candidates []string
}

func (e *AmbiguousTargetError) Error() string {
	if len(e.Candidates) == 0 {
		return fmt.Sprintf("target %q is ambiguous", e.Target)
	}
	return fmt.Sprintf("target %q is ambiguous; use one of: %s", e.Target, strings.Join(e.Candidates, ", "))
}

// ResolveTarget resolves a target in the order defined by the CLI contract:
// gist ID/URL, alias, owner/name, exact filename, then filename stem.
func ResolveTarget(ctx context.Context, input string, paths config.Paths, allowNetwork bool) (ResolvedTarget, error) {
	input = strings.TrimSpace(input)
	id := gist.ExtractID(input)
	if gist.IsLikelyGistID(id) {
		return ResolvedTarget{GistID: id}, nil
	}

	st, err := known.Load(paths.KnownFile)
	if err != nil {
		return ResolvedTarget{}, err
	}

	if matches := entriesByAlias(st.Entries, input); len(matches) > 0 {
		return resolvedOrAmbiguous(input, matches, "")
	}

	if strings.Contains(input, "/") && !strings.Contains(input, "://") {
		parts := strings.SplitN(input, "/", 2)
		if parts[0] == "" || parts[1] == "" {
			return ResolvedTarget{}, targetNotFound(input, "target must use owner/name", true)
		}
		matches, requested := matchOwnerName(st.Entries, parts[0], parts[1])
		if len(matches) > 0 {
			return resolvedOrAmbiguous(input, matches, requested)
		}
		// A slash can also be part of an exact filename path. A known local
		// filename is unambiguous and avoids an unnecessary owner lookup.
		if matches, requested := matchFilename(st.Entries, input); len(matches) > 0 {
			return resolvedOrAmbiguous(input, matches, requested)
		}
		if !allowNetwork {
			return ResolvedTarget{}, targetNotFound(input,
				fmt.Sprintf("cannot resolve unknown owner/name %s while offline", input), false)
		}

		live, err := findOwnerNameLive(ctx, parts[0], parts[1], paths)
		if err != nil {
			return ResolvedTarget{}, err
		}
		if len(live.matches) > 0 {
			return resolvedOrAmbiguous(input, live.matches, live.requested)
		}
		return ResolvedTarget{}, targetNotFound(input,
			fmt.Sprintf("could not find %q among %s's gists", parts[1], parts[0]), true)
	}

	if matches, requested := matchFilename(st.Entries, input); len(matches) > 0 {
		return resolvedOrAmbiguous(input, matches, requested)
	}

	return ResolvedTarget{}, targetNotFound(input,
		fmt.Sprintf("could not resolve %q as a gist id, URL, owner/gist, or known name (run `gixt add <target> --as <name>` to remember it)", input), true)
}

// resolveTarget is retained for commands that only need the gist identity.
func resolveTarget(ctx context.Context, input string, paths config.Paths, allowNetwork bool) (string, error) {
	target, err := ResolveTarget(ctx, input, paths, allowNetwork)
	return target.GistID, err
}

func targetNotFound(target, detail string, suggest bool) *TargetNotFoundError {
	return &TargetNotFoundError{Target: target, Detail: detail, Suggest: suggest}
}

func resolvedOrAmbiguous(input string, matches []known.Entry, requested string) (ResolvedTarget, error) {
	matches = uniqueEntries(matches)
	if len(matches) != 1 {
		return ResolvedTarget{}, &AmbiguousTargetError{
			Target:     input,
			Candidates: disambiguators(input, matches),
		}
	}
	return ResolvedTarget{GistID: matches[0].ID, RequestedFile: requested}, nil
}

func entriesByAlias(entries []known.Entry, input string) []known.Entry {
	var matches []known.Entry
	for _, entry := range entries {
		if strings.EqualFold(entry.Alias, input) {
			matches = append(matches, entry)
		}
	}
	return uniqueEntries(matches)
}

func matchOwnerName(entries []known.Entry, owner, name string) ([]known.Entry, string) {
	var scoped []known.Entry
	for _, entry := range entries {
		if strings.EqualFold(entry.Owner, owner) {
			scoped = append(scoped, entry)
		}
	}
	if len(scoped) == 0 {
		return nil, ""
	}

	var aliases []known.Entry
	for _, entry := range scoped {
		if strings.EqualFold(entry.Alias, name) {
			aliases = append(aliases, entry)
		}
	}
	if len(aliases) > 0 {
		return uniqueEntries(aliases), ""
	}

	var exact []known.Entry
	for _, entry := range scoped {
		if hasExactFilename(entry, name) {
			exact = append(exact, entry)
		}
	}
	if len(exact) > 0 {
		return uniqueEntries(exact), name
	}

	var stems []known.Entry
	for _, entry := range scoped {
		if hasFilenameStem(entry, name) {
			stems = append(stems, entry)
		}
	}
	return uniqueEntries(stems), ""
}

type ownerLookup struct {
	matches   []known.Entry
	requested string
}

// findOwnerNameLive resolves owner/name against GitHub live.
func findOwnerNameLive(ctx context.Context, owner, name string, paths config.Paths) (ownerLookup, error) {
	client := gist.New(loadToken(paths.AuthFile))
	items, err := client.ListForOwner(ctx, owner, 100, 5)
	if err != nil {
		return ownerLookup{}, err
	}
	entries := make([]known.Entry, 0, len(items))
	for _, item := range items {
		entries = append(entries, toKnownEntryFromList(item))
	}
	matches, requested := matchOwnerName(entries, owner, name)
	return ownerLookup{matches: matches, requested: requested}, nil
}

func matchFilename(entries []known.Entry, input string) ([]known.Entry, string) {
	var exact []known.Entry
	for _, entry := range entries {
		if hasExactFilename(entry, input) {
			exact = append(exact, entry)
		}
	}
	if len(exact) > 0 {
		return uniqueEntries(exact), input
	}

	var stems []known.Entry
	for _, entry := range entries {
		if hasFilenameStem(entry, input) {
			stems = append(stems, entry)
		}
	}
	return uniqueEntries(stems), ""
}

func hasExactFilename(entry known.Entry, filename string) bool {
	for _, candidate := range entry.Filenames {
		if candidate == filename {
			return true
		}
	}
	return false
}

func hasFilenameStem(entry known.Entry, stem string) bool {
	stem = strings.ToLower(stem)
	for _, filename := range entry.Filenames {
		base := filepath.Base(filename)
		if strings.ToLower(strings.TrimSuffix(base, filepath.Ext(base))) == stem {
			return true
		}
	}
	return false
}

func uniqueEntries(entries []known.Entry) []known.Entry {
	seen := make(map[string]bool, len(entries))
	out := make([]known.Entry, 0, len(entries))
	for _, entry := range entries {
		if entry.ID == "" || seen[entry.ID] {
			continue
		}
		seen[entry.ID] = true
		out = append(out, entry)
	}
	return out
}

func disambiguators(input string, entries []known.Entry) []string {
	seen := map[string]bool{}
	var out []string
	add := func(value string) {
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	for _, entry := range entries {
		add(entry.Alias)
		if entry.Owner != "" && !strings.Contains(input, "/") {
			add(entry.Owner + "/" + input)
		}
		add(entry.ID)
	}
	sort.Strings(out)
	return out
}
