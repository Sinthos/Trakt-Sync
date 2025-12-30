package trakt

import "fmt"

// GetTrendingShows returns trending shows filtered by minimum rating
func (c *Client) GetTrendingShows(limit int, minRating int) ([]TrendingShow, error) {
	var shows []TrendingShow
	path := fmt.Sprintf("/shows/trending?limit=%d", limit)
	if minRating > 0 {
		path += fmt.Sprintf("&ratings=%d-100", minRating)
	}
	_, err := c.doRequest("GET", path, nil, &shows)
	if err != nil {
		return nil, fmt.Errorf("failed to get trending shows: %w", err)
	}
	return shows, nil
}

// GetPopularShows returns popular shows filtered by minimum rating
func (c *Client) GetPopularShows(limit int, minRating int) ([]Show, error) {
	var shows []Show
	path := fmt.Sprintf("/shows/popular?limit=%d", limit)
	if minRating > 0 {
		path += fmt.Sprintf("&ratings=%d-100", minRating)
	}
	_, err := c.doRequest("GET", path, nil, &shows)
	if err != nil {
		return nil, fmt.Errorf("failed to get popular shows: %w", err)
	}
	return shows, nil
}

// GetMostWatchedShows returns most watched shows weekly filtered by minimum rating
func (c *Client) GetMostWatchedShows(limit int, minRating int) ([]WatchedShow, error) {
	var shows []WatchedShow
	path := fmt.Sprintf("/shows/watched/weekly?limit=%d", limit)
	if minRating > 0 {
		path += fmt.Sprintf("&ratings=%d-100", minRating)
	}
	_, err := c.doRequest("GET", path, nil, &shows)
	if err != nil {
		return nil, fmt.Errorf("failed to get most watched shows: %w", err)
	}
	return shows, nil
}
