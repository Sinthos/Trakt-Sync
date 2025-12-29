package trakt

import (
	"fmt"
	"time"
)

// DeviceCodeResponse represents the response from the device code endpoint
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// TokenResponse represents the OAuth token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	CreatedAt    int64  `json:"created_at"`
}

// Movie represents a Trakt movie
type Movie struct {
	Title string   `json:"title"`
	Year  int      `json:"year"`
	IDs   MediaIDs `json:"ids"`
}

// Show represents a Trakt show
type Show struct {
	Title string   `json:"title"`
	Year  int      `json:"year"`
	IDs   MediaIDs `json:"ids"`
}

// MediaIDs contains various IDs for media items
type MediaIDs struct {
	Trakt int    `json:"trakt"`
	Slug  string `json:"slug"`
	IMDB  string `json:"imdb,omitempty"`
	TMDB  int    `json:"tmdb,omitempty"`
}

// TrendingMovie wraps a movie with trending data
type TrendingMovie struct {
	Watchers int   `json:"watchers"`
	Movie    Movie `json:"movie"`
}

// TrendingShow wraps a show with trending data
type TrendingShow struct {
	Watchers int  `json:"watchers"`
	Show     Show `json:"show"`
}

// WatchedMovie wraps a movie with watch count
type WatchedMovie struct {
	WatcherCount   int   `json:"watcher_count"`
	PlayCount      int   `json:"play_count"`
	CollectedCount int   `json:"collected_count"`
	Movie          Movie `json:"movie"`
}

// WatchedShow wraps a show with watch count
type WatchedShow struct {
	WatcherCount   int  `json:"watcher_count"`
	PlayCount      int  `json:"play_count"`
	CollectedCount int  `json:"collected_count"`
	Show           Show `json:"show"`
}

// List represents a Trakt list
type List struct {
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Privacy        string    `json:"privacy"`
	DisplayNumbers bool      `json:"display_numbers"`
	AllowComments  bool      `json:"allow_comments"`
	SortBy         string    `json:"sort_by"`
	SortHow        string    `json:"sort_how"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	ItemCount      int       `json:"item_count"`
	CommentCount   int       `json:"comment_count"`
	Likes          int       `json:"likes"`
	IDs            ListIDs   `json:"ids"`
}

// ListIDs contains IDs for a list
type ListIDs struct {
	Trakt int    `json:"trakt"`
	Slug  string `json:"slug"`
}

// ListItem represents an item in a list
type ListItem struct {
	Rank     int       `json:"rank"`
	ListedAt time.Time `json:"listed_at"`
	Type     string    `json:"type"`
	Movie    *Movie    `json:"movie,omitempty"`
	Show     *Show     `json:"show,omitempty"`
}

// AddToListRequest represents items to add to a list
type AddToListRequest struct {
	Movies []AddMovie `json:"movies,omitempty"`
	Shows  []AddShow  `json:"shows,omitempty"`
}

// AddMovie represents a movie to add to a list
type AddMovie struct {
	IDs MediaIDs `json:"ids"`
}

// AddShow represents a show to add to a list
type AddShow struct {
	IDs MediaIDs `json:"ids"`
}

// RemoveFromListRequest represents items to remove from a list
type RemoveFromListRequest struct {
	Movies []RemoveMovie `json:"movies,omitempty"`
	Shows  []RemoveShow  `json:"shows,omitempty"`
}

// RemoveMovie represents a movie to remove from a list
type RemoveMovie struct {
	IDs MediaIDs `json:"ids"`
}

// RemoveShow represents a show to remove from a list
type RemoveShow struct {
	IDs MediaIDs `json:"ids"`
}

// CreateListRequest represents a request to create a new list
type CreateListRequest struct {
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	Privacy        string `json:"privacy"`
	DisplayNumbers bool   `json:"display_numbers"`
	AllowComments  bool   `json:"allow_comments"`
}

// ErrorResponse represents an error from the Trakt API
type ErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// APIError provides structured errors for Trakt API responses
type APIError struct {
	Status      int
	Code        string
	Description string
	RetryAfter  time.Duration
}

func (e *APIError) Error() string {
	if e == nil {
		return "API error: <nil>"
	}
	if e.Code != "" {
		return fmt.Sprintf("API error: %s - %s", e.Code, e.Description)
	}
	return fmt.Sprintf("API error: status %d", e.Status)
}
