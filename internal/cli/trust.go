package cli

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/leolaurindo/gixt/internal/cache"
	"github.com/leolaurindo/gixt/internal/gist"
	"github.com/leolaurindo/gixt/internal/trust"
)

func newTrustCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "trust",
		Short: "manage trust approvals",
		Args:  cobra.NoArgs,
	}
	c.AddCommand(
		&cobra.Command{
			Use:   "mine",
			Short: "approve the current commit of every gist you own",
			Args:  cobra.NoArgs,
			RunE:  trustMine,
		},
		&cobra.Command{
			Use:   "list",
			Short: "show approved gists and commits",
			Args:  cobra.NoArgs,
			RunE:  trustList,
		},
		&cobra.Command{
			Use:   "remove <target>",
			Short: "revoke an approval",
			Args:  cobra.ExactArgs(1),
			RunE:  trustRemove,
		},
		&cobra.Command{
			Use:   "clear",
			Short: "revoke every approval",
			Args:  cobra.NoArgs,
			RunE:  trustClear,
		},
	)
	return c
}

const trustMineWorkers = 3

func trustMine(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths()
	if err != nil {
		return err
	}
	client := gist.New(loadToken(paths.AuthFile))
	items, err := client.ListMine(cmd.Context(), 100)
	if err != nil {
		return fmt.Errorf("trust mine: %w", err)
	}
	store, err := trust.Load(paths.TrustFile)
	if err != nil {
		return err
	}

	snapshots, err := snapshotTrust(cmd.Context(), items, func(ctx context.Context, id string) (gist.Gist, error) {
		return fetchTrustGist(ctx, client, id)
	}, func(done, total int) {
		logf("Snapshotting gists: %d/%d", done, total)
	})
	if err != nil {
		return err
	}
	for _, snapshot := range snapshots {
		store.Trust(snapshot.ID, snapshot.SHA, snapshot.Owner)
	}
	if err := trust.Save(paths.TrustFile, store); err != nil {
		return err
	}
	logf("approved %d gists at their current commits", len(snapshots))
	return nil
}

type trustSnapshot struct {
	ID    string
	SHA   string
	Owner string
}

type trustSnapshotResult struct {
	index int
	item  trustSnapshot
	err   error
}

func snapshotTrust(ctx context.Context, items []gist.ListItem, fetch func(context.Context, string) (gist.Gist, error), progress func(done, total int)) ([]trustSnapshot, error) {
	items = uniqueListItems(items)
	if len(items) == 0 {
		return nil, nil
	}

	workCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan int)
	results := make(chan trustSnapshotResult, len(items))
	workers := trustMineWorkers
	if len(items) < workers {
		workers = len(items)
	}

	var wg sync.WaitGroup
	wg.Add(workers)
	for worker := 0; worker < workers; worker++ {
		go func() {
			defer wg.Done()
			for index := range jobs {
				item, err := fetch(workCtx, items[index].ID)
				result := trustSnapshotResult{index: index}
				if err != nil {
					result.err = fmt.Errorf("snapshot gist %s: %w", items[index].ID, err)
					results <- result
					cancel()
					continue
				}
				sha := item.LatestVersion()
				if sha == "" {
					result.err = fmt.Errorf("could not determine the current revision of %s", items[index].ID)
					results <- result
					cancel()
					continue
				}
				result.item = trustSnapshot{ID: item.ID, SHA: sha, Owner: item.Owner.Login}
				results <- result
			}
		}()
	}

	go func() {
		defer close(jobs)
		for index := range items {
			select {
			case jobs <- index:
			case <-workCtx.Done():
				return
			}
		}
	}()
	go func() {
		wg.Wait()
		close(results)
	}()

	snapshots := make([]trustSnapshot, len(items))
	errs := make([]error, len(items))
	completed := 0
	for result := range results {
		if result.err != nil {
			errs[result.index] = result.err
			continue
		}
		snapshots[result.index] = result.item
		completed++
		if progress != nil {
			progress(completed, len(items))
		}
	}
	for _, err := range errs {
		if err != nil && !errors.Is(err, context.Canceled) {
			return nil, err
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return snapshots, nil
}

func uniqueListItems(items []gist.ListItem) []gist.ListItem {
	seen := make(map[string]bool, len(items))
	unique := make([]gist.ListItem, 0, len(items))
	for _, item := range items {
		if item.ID == "" || seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		unique = append(unique, item)
	}
	return unique
}

func fetchTrustGist(ctx context.Context, client *gist.Client, id string) (gist.Gist, error) {
	const attempts = 3
	for attempt := 0; attempt < attempts; attempt++ {
		item, err := client.Fetch(ctx, id, "")
		if err == nil {
			return item, nil
		}
		var rateLimit *gist.RateLimitError
		if !errors.As(err, &rateLimit) || attempt == attempts-1 {
			return gist.Gist{}, err
		}
		delay := rateLimit.RetryDelay(time.Now())
		if err := waitForTrustRetry(ctx, delay); err != nil {
			return gist.Gist{}, err
		}
	}
	return gist.Gist{}, errors.New("trust snapshot retry limit reached")
}

func waitForTrustRetry(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func trustList(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths()
	if err != nil {
		return err
	}
	store, err := trust.Load(paths.TrustFile)
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(store.Entries))
	for id := range store.Entries {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		e := store.Entries[id]
		fmt.Printf("%s\t%s\t%s\n", id, cache.Shorten(e.SHA), e.Owner)
	}
	return nil
}

func trustRemove(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths()
	if err != nil {
		return err
	}
	id, err := resolveTarget(cmd.Context(), args[0], paths, true)
	if err != nil {
		return err
	}
	store, err := trust.Load(paths.TrustFile)
	if err != nil {
		return err
	}
	if _, ok := store.Entries[id]; !ok {
		return fmt.Errorf("gist %s has no trust approval", id)
	}
	delete(store.Entries, id)
	return trust.Save(paths.TrustFile, store)
}

func trustClear(cmd *cobra.Command, args []string) error {
	paths, err := ensurePaths()
	if err != nil {
		return err
	}
	store, err := trust.Load(paths.TrustFile)
	if err != nil {
		return err
	}
	store.Entries = map[string]trust.Entry{}
	return trust.Save(paths.TrustFile, store)
}
