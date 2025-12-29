package sync

import (
	"reflect"
	"sort"
	"testing"

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
