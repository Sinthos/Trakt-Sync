package trakt

import "fmt"

// GetTrendingMovies returns trending movies
func (c *Client) GetTrendingMovies(limit int) ([]TrendingMovie, error) {
	var movies []TrendingMovie
	path := fmt.Sprintf("/movies/trending?limit=%d", limit)
	_, err := c.doRequest("GET", path, nil, &movies)
	if err != nil {
		return nil, fmt.Errorf("failed to get trending movies: %w", err)
	}
	return movies, nil
}

// GetPopularMovies returns popular movies
func (c *Client) GetPopularMovies(limit int) ([]Movie, error) {
	var movies []Movie
	path := fmt.Sprintf("/movies/popular?limit=%d", limit)
	_, err := c.doRequest("GET", path, nil, &movies)
	if err != nil {
		return nil, fmt.Errorf("failed to get popular movies: %w", err)
	}
	return movies, nil
}

// GetMostWatchedMovies returns most watched movies weekly
func (c *Client) GetMostWatchedMovies(limit int) ([]WatchedMovie, error) {
	var movies []WatchedMovie
	path := fmt.Sprintf("/movies/watched/weekly?limit=%d", limit)
	_, err := c.doRequest("GET", path, nil, &movies)
	if err != nil {
		return nil, fmt.Errorf("failed to get most watched movies: %w", err)
	}
	return movies, nil
}
