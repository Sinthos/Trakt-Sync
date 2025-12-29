package sync

import (
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/maximilian/trakt-sync/internal/config"
	"github.com/maximilian/trakt-sync/internal/trakt"
)

func TestCalculateDiffMovies(t *testing.T) {
	syncer := &Syncer{}
	current := []trakt.ListItem{
		{Movie: &trakt.Movie{IDs: trakt.MediaIDs{Trakt: 1}}},
		{Movie: &trakt.Movie{IDs: trakt.MediaIDs{Trakt: 2}}},
	}
	newItems := []trakt.MediaIDs{{Trakt: 2}, {Trakt: 3}}

	toAdd, toRemove := syncer.calculateDiff(current, newItems)

	assertIDs(t, toAdd, []int{3})
	assertIDs(t, toRemove, []int{1})
}

func TestCalculateDiffShows(t *testing.T) {
	syncer := &Syncer{}
	current := []trakt.ListItem{
		{Show: &trakt.Show{IDs: trakt.MediaIDs{Trakt: 10}}},
	}
	newItems := []trakt.MediaIDs{}

	toAdd, toRemove := syncer.calculateDiff(current, newItems)

	assertIDs(t, toAdd, []int{})
	assertIDs(t, toRemove, []int{10})
}

func TestUniqueIDs(t *testing.T) {
	items := []trakt.MediaIDs{{Trakt: 1}, {Trakt: 2}, {Trakt: 1}}
	unique := uniqueIDs(items)
	assertIDs(t, unique, []int{1, 2})
}

func TestShouldFullRefresh(t *testing.T) {
	now := time.Now()
	cfg := &config.Config{
		Sync: config.SyncConfig{
			FullRefreshDays: 7,
			LastFullRefresh: config.FullRefreshState{
				Movies: now.Add(-8 * 24 * time.Hour),
				Shows:  now.Add(-5 * 24 * time.Hour),
			},
		},
	}

	syncer := &Syncer{config: cfg}

	if !syncer.shouldFullRefresh(true) {
		t.Fatal("expected movies to require full refresh")
	}

	if syncer.shouldFullRefresh(false) {
		t.Fatal("did not expect shows to require full refresh")
	}
}

func assertIDs(t *testing.T, got []trakt.MediaIDs, want []int) {
	t.Helper()
	if want == nil {
		want = []int{}
	}
	gotIDs := extractIDs(got)
	sort.Ints(gotIDs)
	sort.Ints(want)
	if !reflect.DeepEqual(gotIDs, want) {
		t.Fatalf("expected IDs %v, got %v", want, gotIDs)
	}
}

func extractIDs(items []trakt.MediaIDs) []int {
	ids := make([]int, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.Trakt)
	}
	return ids
}
