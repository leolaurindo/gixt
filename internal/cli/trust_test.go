package cli

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/leolaurindo/gixt/internal/gist"
)

func TestSnapshotTrustLimitsWorkersAndReportsProgress(t *testing.T) {
	items := make([]gist.ListItem, 10)
	for i := range items {
		items[i].ID = fmt.Sprintf("gist-%d", i)
	}

	var active, maximum atomic.Int32
	var readyOnce sync.Once
	ready := make(chan struct{})
	release := make(chan struct{})
	fetch := func(ctx context.Context, id string) (gist.Gist, error) {
		current := active.Add(1)
		for {
			old := maximum.Load()
			if current <= old || maximum.CompareAndSwap(old, current) {
				break
			}
		}
		if current == trustMineWorkers {
			readyOnce.Do(func() { close(ready) })
		}
		select {
		case <-release:
		case <-ctx.Done():
			active.Add(-1)
			return gist.Gist{}, ctx.Err()
		}
		active.Add(-1)
		return gist.Gist{ID: id, History: []gist.HistoryEntry{{Version: "sha-" + id}}}, nil
	}

	type result struct {
		snapshots []trustSnapshot
		progress  []int
		err       error
	}
	finished := make(chan result, 1)
	var progress []int
	go func() {
		snapshots, err := snapshotTrust(context.Background(), items, fetch, func(done, total int) {
			progress = append(progress, done, total)
		})
		finished <- result{snapshots: snapshots, progress: progress, err: err}
	}()

	select {
	case <-ready:
	case <-time.After(time.Second):
		t.Fatal("worker pool did not start three requests")
	}
	close(release)

	got := <-finished
	if got.err != nil {
		t.Fatalf("snapshotTrust error: %v", got.err)
	}
	if len(got.snapshots) != len(items) || maximum.Load() != trustMineWorkers {
		t.Fatalf("got %d snapshots with maximum concurrency %d", len(got.snapshots), maximum.Load())
	}
	if len(got.progress) != len(items)*2 || got.progress[len(got.progress)-2] != len(items) || got.progress[len(got.progress)-1] != len(items) {
		t.Fatalf("unexpected progress: %v", got.progress)
	}
}

func TestSnapshotTrustReturnsErrorsInInputOrder(t *testing.T) {
	items := []gist.ListItem{{ID: "first"}, {ID: "second"}, {ID: "third"}}
	_, err := snapshotTrust(context.Background(), items, func(ctx context.Context, id string) (gist.Gist, error) {
		if id == "first" || id == "third" {
			return gist.Gist{}, errors.New(id + " failed")
		}
		return gist.Gist{ID: id, History: []gist.HistoryEntry{{Version: "ok"}}}, nil
	}, nil)
	if err == nil || err.Error() != "snapshot gist first: first failed" {
		t.Fatalf("expected first input error, got %v", err)
	}
}
