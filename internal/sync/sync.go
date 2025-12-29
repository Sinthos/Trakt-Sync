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
	client *trakt.Client
	config *config.Config
}

// NewSyncer creates a new syncer
func NewSyncer(client *trakt.Client, cfg *config.Config) *Syncer {
	return &Syncer{
		client: client,
		config: cfg,
	}
}

// GetListDefinitions returns all list definitions based on config
func (s *Syncer) GetListDefinitions() []ListDefinition {
	return []ListDefinition{
		{
			Slug:        "trending-movies",
			Name:        "Trending Movies",
			Description: "Top 20 trending movies on Trakt",
			Enabled:     s.config.Sync.Lists.TrendingMovies,
			FetchFunc:   s.fetchTrendingMovies,
		},
		{
			Slug:        "trending-shows",
			Name:        "Trending Shows",
			Description: "Top 20 trending shows on Trakt",
			Enabled:     s.config.Sync.Lists.TrendingShows,
			FetchFunc:   s.fetchTrendingShows,
		},
		{
			Slug:        "popular-movies",
			Name:        "Popular Movies",
			Description: "Top 20 popular movies on Trakt",
			Enabled:     s.config.Sync.Lists.PopularMovies,
			FetchFunc:   s.fetchPopularMovies,
		},
		{
			Slug:        "popular-shows",
			Name:        "Popular Shows",
			Description: "Top 20 popular shows on Trakt",
			Enabled:     s.config.Sync.Lists.PopularShows,
			FetchFunc:   s.fetchPopularShows,
		},
		{
			Slug:        "streaming-charts-movies",
			Name:        "Streaming Charts Movies",
			Description: "Top 20 most watched movies this week",
			Enabled:     s.config.Sync.Lists.StreamingMovies,
			FetchFunc:   s.fetchStreamingMovies,
		},
		{
			Slug:        "streaming-charts-shows",
			Name:        "Streaming Charts Shows",
			Description: "Top 20 most watched shows this week",
			Enabled:     s.config.Sync.Lists.StreamingShows,
			FetchFunc:   s.fetchStreamingShows,
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

	log.Info().Str("list", listDef.Slug).Int("count", len(newItems)).Msg("Fetched items from API")

	currentItems, err := s.client.GetListItems(s.config.Trakt.Username, listDef.Slug)
	if err != nil {
		return fmt.Errorf("failed to get current list items: %w", err)
	}

	toAdd, toRemove := s.calculateDiff(currentItems, newItems)

	if len(toRemove) > 0 {
		if err := s.removeItems(listDef.Slug, toRemove); err != nil {
			return fmt.Errorf("failed to remove items: %w", err)
		}
	}

	if len(toAdd) > 0 {
		if err := s.addItems(listDef.Slug, toAdd); err != nil {
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
func (s *Syncer) addItems(listSlug string, items []trakt.MediaIDs) error {
	isMovieList := listSlug == "trending-movies" || listSlug == "popular-movies" || listSlug == "streaming-charts-movies"

	req := trakt.AddToListRequest{}

	if isMovieList {
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
func (s *Syncer) removeItems(listSlug string, items []trakt.MediaIDs) error {
	isMovieList := listSlug == "trending-movies" || listSlug == "popular-movies" || listSlug == "streaming-charts-movies"

	req := trakt.RemoveFromListRequest{}

	if isMovieList {
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

// Fetch functions for different list types
func (s *Syncer) fetchTrendingMovies(client *trakt.Client, limit int) ([]trakt.MediaIDs, error) {
	movies, err := client.GetTrendingMovies(limit)
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
	shows, err := client.GetTrendingShows(limit)
	if err != nil {
		return nil, err
	}

	var ids []trakt.MediaIDs
	for _, s := range shows {
		ids = append(ids, s.Show.IDs)
	}
	return ids, nil
}

func (s *Syncer) fetchPopularMovies(client *trakt.Client, limit int) ([]trakt.MediaIDs, error) {
	movies, err := client.GetPopularMovies(limit)
	if err != nil {
		return nil, err
	}

	var ids []trakt.MediaIDs
	for _, m := range movies {
		ids = append(ids, m.IDs)
	}
	return ids, nil
}

func (s *Syncer) fetchPopularShows(client *trakt.Client, limit int) ([]trakt.MediaIDs, error) {
	shows, err := client.GetPopularShows(limit)
	if err != nil {
		return nil, err
	}

	var ids []trakt.MediaIDs
	for _, s := range shows {
		ids = append(ids, s.IDs)
	}
	return ids, nil
}

func (s *Syncer) fetchStreamingMovies(client *trakt.Client, limit int) ([]trakt.MediaIDs, error) {
	movies, err := client.GetMostWatchedMovies(limit)
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
	shows, err := client.GetMostWatchedShows(limit)
	if err != nil {
		return nil, err
	}

	var ids []trakt.MediaIDs
	for _, s := range shows {
		ids = append(ids, s.Show.IDs)
	}
	return ids, nil
}
