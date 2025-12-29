package trakt

import "fmt"

// GetTrendingShows returns trending shows
func (c *Client) GetTrendingShows(limit int) ([]TrendingShow, error) {
	var shows []TrendingShow
	path := fmt.Sprintf("/shows/trending?limit=%d", limit)
	_, err := c.doRequest("GET", path, nil, &shows)
	if err != nil {
		return nil, fmt.Errorf("failed to get trending shows: %w", err)
	}
	return shows, nil
}

// GetPopularShows returns popular shows
func (c *Client) GetPopularShows(limit int) ([]Show, error) {
	var shows []Show
	path := fmt.Sprintf("/shows/popular?limit=%d", limit)
	_, err := c.doRequest("GET", path, nil, &shows)
	if err != nil {
		return nil, fmt.Errorf("failed to get popular shows: %w", err)
	}
	return shows, nil
}

// GetMostWatchedShows returns most watched shows weekly
func (c *Client) GetMostWatchedShows(limit int) ([]WatchedShow, error) {
	var shows []WatchedShow
	path := fmt.Sprintf("/shows/watched/weekly?limit=%d", limit)
	_, err := c.doRequest("GET", path, nil, &shows)
	if err != nil {
		return nil, fmt.Errorf("failed to get most watched shows: %w", err)
	}
	return shows, nil
}
