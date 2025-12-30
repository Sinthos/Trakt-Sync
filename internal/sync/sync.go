package sync

import (
	"errors"
	"fmt"
	"time"

	"github.com/maximilian/trakt-sync/internal/config"
	"github.com/maximilian/trakt-sync/internal/trakt"
	"github.com/rs/zerolog/log"
)

var ErrAllFailed = errors.New("all lists failed to sync")

// ListDefinition defines a list to sync
type ListDefinition struct {
	Slug        string
	Name        string
	Description string
	Enabled     bool
	FetchFunc   func(*trakt.Client, int) ([]trakt.MediaIDs, error)
	IsMovie     bool
}

// SyncResult captures the summary of a sync run
type SyncResult struct {
	Successful int
	Failed     int
	Total      int
	Duration   time.Duration
}

// Syncer handles syncing lists
type Syncer struct {
	client      *trakt.Client
	config      *config.Config
	configDirty bool
}

// NewSyncer creates a new syncer
func NewSyncer(client *trakt.Client, cfg *config.Config) *Syncer {
	return &Syncer{
		client: client,
		config: cfg,
	}
}

// ConfigDirty reports whether sync updated persisted config values.
func (s *Syncer) ConfigDirty() bool {
	return s.configDirty
}

// GetListDefinitions returns all list definitions based on config
func (s *Syncer) GetListDefinitions() []ListDefinition {
	return []ListDefinition{
		{
			Slug:        "trakt-sync-filme",
			Name:        "Trakt Sync Filme",
			Description: "Top 20 trending and top 20 streaming charts movies",
			Enabled:     s.config.Sync.Lists.Movies,
			FetchFunc:   s.fetchCombinedMovies,
			IsMovie:     true,
		},
		{
			Slug:        "trakt-sync-serien",
			Name:        "Trakt Sync Serien",
			Description: "Top 20 trending and top 20 streaming charts shows",
			Enabled:     s.config.Sync.Lists.Shows,
			FetchFunc:   s.fetchCombinedShows,
			IsMovie:     false,
		},
	}
}

// SyncAll syncs all enabled lists
func (s *Syncer) SyncAll() (SyncResult, error) {
	startTime := time.Now()
	lists := s.GetListDefinitions()

	result := SyncResult{}

	log.Info().Msg("Starting sync...")

	for _, listDef := range lists {
		if !listDef.Enabled {
			log.Debug().Str("list", listDef.Slug).Msg("List disabled, skipping")
			continue
		}

		result.Total++

		if err := s.SyncList(listDef); err != nil {
			log.Error().Err(err).Str("list", listDef.Slug).Msg("Failed to sync list")
			result.Failed++
			continue
		}

		result.Successful++
	}

	result.Duration = time.Since(startTime)

	if result.Total == 0 {
		log.Warn().Msg("No lists enabled for sync")
		return result, nil
	}

	log.Info().
		Int("successful", result.Successful).
		Int("failed", result.Failed).
		Int("total", result.Total).
		Dur("duration", result.Duration).
		Msg("Sync complete")

	if result.Failed > 0 && result.Successful == 0 {
		return result, ErrAllFailed
	}

	return result, nil
}

// SyncList syncs a single list
func (s *Syncer) SyncList(listDef ListDefinition) error {
	startTime := time.Now()

	log.Info().Str("list", listDef.Slug).Msg("Starting list sync")

	if err := s.client.EnsureListExists(
		s.config.Trakt.Username,
		listDef.Slug,
		listDef.Name,
		listDef.Description,
		s.config.Sync.ListPrivacy,
	); err != nil {
		return fmt.Errorf("failed to ensure list exists: %w", err)
	}

	newItems, err := listDef.FetchFunc(s.client, s.config.Sync.Limit)
	if err != nil {
		return fmt.Errorf("failed to fetch items: %w", err)
	}
	newItems = uniqueIDs(newItems)

	log.Info().Str("list", listDef.Slug).Int("count", len(newItems)).Msg("Fetched items from API")

	currentItems, err := s.client.GetListItems(s.config.Trakt.Username, listDef.Slug)
	if err != nil {
		return fmt.Errorf("failed to get current list items: %w", err)
	}

	if s.shouldFullRefresh(listDef.IsMovie) {
		toRemove := listItemIDs(currentItems)
		if len(toRemove) > 0 {
			if err := s.removeItems(listDef.Slug, toRemove, listDef.IsMovie); err != nil {
				return fmt.Errorf("failed to remove items: %w", err)
			}
		}

		if len(newItems) > 0 {
			if err := s.addItems(listDef.Slug, newItems, listDef.IsMovie); err != nil {
				return fmt.Errorf("failed to add items: %w", err)
			}
		}

		s.markFullRefresh(listDef.IsMovie)

		duration := time.Since(startTime)
		log.Info().
			Str("list", listDef.Slug).
			Bool("full_refresh", true).
			Int("added", len(newItems)).
			Int("removed", len(toRemove)).
			Int("unchanged", 0).
			Dur("duration", duration).
			Msg("List sync complete")
		return nil
	}

	toAdd, toRemove := s.calculateDiff(currentItems, newItems)

	if len(toRemove) > 0 {
		if err := s.removeItems(listDef.Slug, toRemove, listDef.IsMovie); err != nil {
			return fmt.Errorf("failed to remove items: %w", err)
		}
	}

	if len(toAdd) > 0 {
		if err := s.addItems(listDef.Slug, toAdd, listDef.IsMovie); err != nil {
			return fmt.Errorf("failed to add items: %w", err)
		}
	}

	unchanged := len(currentItems) - len(toRemove)
	duration := time.Since(startTime)

	log.Info().
		Str("list", listDef.Slug).
		Int("added", len(toAdd)).
		Int("removed", len(toRemove)).
		Int("unchanged", unchanged).
		Dur("duration", duration).
		Msg("List sync complete")

	return nil
}

func (s *Syncer) shouldFullRefresh(isMovie bool) bool {
	days := s.config.Sync.FullRefreshDays
	if days <= 0 {
		days = 7
	}

	last := s.lastFullRefresh(isMovie)
	if last.IsZero() {
		return true
	}

	return time.Since(last) >= time.Duration(days)*24*time.Hour
}

func (s *Syncer) lastFullRefresh(isMovie bool) time.Time {
	if isMovie {
		return s.config.Sync.LastFullRefresh.Movies
	}
	return s.config.Sync.LastFullRefresh.Shows
}

func (s *Syncer) markFullRefresh(isMovie bool) {
	now := time.Now().UTC()
	if isMovie {
		s.config.Sync.LastFullRefresh.Movies = now
	} else {
		s.config.Sync.LastFullRefresh.Shows = now
	}
	s.configDirty = true
}

// calculateDiff calculates which items to add and remove
func (s *Syncer) calculateDiff(current []trakt.ListItem, new []trakt.MediaIDs) (toAdd, toRemove []trakt.MediaIDs) {
	currentMap := make(map[int]bool)
	for _, item := range current {
		if item.Movie != nil {
			currentMap[item.Movie.IDs.Trakt] = true
		} else if item.Show != nil {
			currentMap[item.Show.IDs.Trakt] = true
		}
	}

	newMap := make(map[int]trakt.MediaIDs)
	for _, ids := range new {
		newMap[ids.Trakt] = ids
	}

	for _, ids := range new {
		if !currentMap[ids.Trakt] {
			toAdd = append(toAdd, ids)
		}
	}

	for _, item := range current {
		var traktID int
		var ids trakt.MediaIDs

		if item.Movie != nil {
			traktID = item.Movie.IDs.Trakt
			ids = item.Movie.IDs
		} else if item.Show != nil {
			traktID = item.Show.IDs.Trakt
			ids = item.Show.IDs
		}

		if _, exists := newMap[traktID]; !exists {
			toRemove = append(toRemove, ids)
		}
	}

	return toAdd, toRemove
}

// addItems adds items to a list
func (s *Syncer) addItems(listSlug string, items []trakt.MediaIDs, isMovie bool) error {
	req := trakt.AddToListRequest{}

	if isMovie {
		for _, ids := range items {
			req.Movies = append(req.Movies, trakt.AddMovie{IDs: ids})
		}
	} else {
		for _, ids := range items {
			req.Shows = append(req.Shows, trakt.AddShow{IDs: ids})
		}
	}

	return s.client.AddItemsToList(s.config.Trakt.Username, listSlug, req)
}

// removeItems removes items from a list
func (s *Syncer) removeItems(listSlug string, items []trakt.MediaIDs, isMovie bool) error {
	req := trakt.RemoveFromListRequest{}

	if isMovie {
		for _, ids := range items {
			req.Movies = append(req.Movies, trakt.RemoveMovie{IDs: ids})
		}
	} else {
		for _, ids := range items {
			req.Shows = append(req.Shows, trakt.RemoveShow{IDs: ids})
		}
	}

	return s.client.RemoveItemsFromList(s.config.Trakt.Username, listSlug, req)
}

func listItemIDs(items []trakt.ListItem) []trakt.MediaIDs {
	ids := make([]trakt.MediaIDs, 0, len(items))
	for _, item := range items {
		if item.Movie != nil {
			ids = append(ids, item.Movie.IDs)
		} else if item.Show != nil {
			ids = append(ids, item.Show.IDs)
		}
	}
	return ids
}

func uniqueIDs(items []trakt.MediaIDs) []trakt.MediaIDs {
	seen := make(map[int]struct{}, len(items))
	unique := make([]trakt.MediaIDs, 0, len(items))
	for _, ids := range items {
		if _, ok := seen[ids.Trakt]; ok {
			continue
		}
		seen[ids.Trakt] = struct{}{}
		unique = append(unique, ids)
	}
	return unique
}

// Fetch functions for different list types
func (s *Syncer) fetchCombinedMovies(client *trakt.Client, limit int) ([]trakt.MediaIDs, error) {
	trending, err := s.fetchTrendingMovies(client, limit)
	if err != nil {
		return nil, err
	}

	streaming, err := s.fetchStreamingMovies(client, limit)
	if err != nil {
		return nil, err
	}

	return uniqueIDs(append(trending, streaming...)), nil
}

func (s *Syncer) fetchCombinedShows(client *trakt.Client, limit int) ([]trakt.MediaIDs, error) {
	trending, err := s.fetchTrendingShows(client, limit)
	if err != nil {
		return nil, err
	}

	streaming, err := s.fetchStreamingShows(client, limit)
	if err != nil {
		return nil, err
	}

	return uniqueIDs(append(trending, streaming...)), nil
}

func (s *Syncer) fetchTrendingMovies(client *trakt.Client, limit int) ([]trakt.MediaIDs, error) {
	movies, err := client.GetTrendingMovies(limit, s.config.Sync.MinRating)
	if err != nil {
		return nil, err
	}

	var ids []trakt.MediaIDs
	for _, m := range movies {
		ids = append(ids, m.Movie.IDs)
	}
	return ids, nil
}

func (s *Syncer) fetchTrendingShows(client *trakt.Client, limit int) ([]trakt.MediaIDs, error) {
	shows, err := client.GetTrendingShows(limit, s.config.Sync.MinRating)
	if err != nil {
		return nil, err
	}

	var ids []trakt.MediaIDs
	for _, sh := range shows {
		ids = append(ids, sh.Show.IDs)
	}
	return ids, nil
}

func (s *Syncer) fetchStreamingMovies(client *trakt.Client, limit int) ([]trakt.MediaIDs, error) {
	movies, err := client.GetMostWatchedMovies(limit, s.config.Sync.MinRating)
	if err != nil {
		return nil, err
	}

	var ids []trakt.MediaIDs
	for _, m := range movies {
		ids = append(ids, m.Movie.IDs)
	}
	return ids, nil
}

func (s *Syncer) fetchStreamingShows(client *trakt.Client, limit int) ([]trakt.MediaIDs, error) {
	shows, err := client.GetMostWatchedShows(limit, s.config.Sync.MinRating)
	if err != nil {
		return nil, err
	}

	var ids []trakt.MediaIDs
	for _, sh := range shows {
		ids = append(ids, sh.Show.IDs)
	}
	return ids, nil
}
