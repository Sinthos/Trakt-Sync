package trakt

import "fmt"

// GetTrendingMovies returns trending movies filtered by minimum rating
func (c *Client) GetTrendingMovies(limit int, minRating int) ([]TrendingMovie, error) {
	var movies []TrendingMovie
	path := fmt.Sprintf("/movies/trending?limit=%d", limit)
	if minRating > 0 {
		path += fmt.Sprintf("&ratings=%d-100", minRating)
	}
	_, err := c.doRequest("GET", path, nil, &movies)
	if err != nil {
		return nil, fmt.Errorf("failed to get trending movies: %w", err)
	}
	return movies, nil
}

// GetPopularMovies returns popular movies filtered by minimum rating
func (c *Client) GetPopularMovies(limit int, minRating int) ([]Movie, error) {
	var movies []Movie
	path := fmt.Sprintf("/movies/popular?limit=%d", limit)
	if minRating > 0 {
		path += fmt.Sprintf("&ratings=%d-100", minRating)
	}
	_, err := c.doRequest("GET", path, nil, &movies)
	if err != nil {
		return nil, fmt.Errorf("failed to get popular movies: %w", err)
	}
	return movies, nil
}

// GetMostWatchedMovies returns most watched movies weekly filtered by minimum rating
func (c *Client) GetMostWatchedMovies(limit int, minRating int) ([]WatchedMovie, error) {
	var movies []WatchedMovie
	path := fmt.Sprintf("/movies/watched/weekly?limit=%d", limit)
	if minRating > 0 {
		path += fmt.Sprintf("&ratings=%d-100", minRating)
	}
	_, err := c.doRequest("GET", path, nil, &movies)
	if err != nil {
		return nil, fmt.Errorf("failed to get most watched movies: %w", err)
	}
	return movies, nil
}
