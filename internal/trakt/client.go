package trakt

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	BaseURL    = "https://api.trakt.tv"
	APIVersion = "2"

	maxRetries  = 3
	baseBackoff = 500 * time.Millisecond
	maxBackoff  = 5 * time.Second
)

// Client is a Trakt API client
type Client struct {
	httpClient     *http.Client
	clientID       string
	clientSecret   string
	accessToken    string
	refreshToken   string
	onTokenRefresh func(accessToken, refreshToken string, expiresAt time.Time)

	rateLimitRemaining int
	rateLimitReset     time.Time
	rateLimitMu        sync.Mutex
}

// NewClient creates a new Trakt API client
func NewClient(clientID, clientSecret, accessToken, refreshToken string) *Client {
	return &Client{
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		clientID:     clientID,
		clientSecret: clientSecret,
		accessToken:  accessToken,
		refreshToken: refreshToken,
	}
}

// SetTokenRefreshCallback sets the callback function called when tokens are refreshed
func (c *Client) SetTokenRefreshCallback(callback func(accessToken, refreshToken string, expiresAt time.Time)) {
	c.onTokenRefresh = callback
}

// doRequest performs an HTTP request with proper headers and retries
func (c *Client) doRequest(method, path string, body interface{}, result interface{}) (*http.Response, error) {
	var bodyBytes []byte
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyBytes = jsonData
	}

	var resp *http.Response
	var err error
	var retryAfter time.Duration

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := backoffDuration(attempt)
			if retryAfter > delay {
				delay = retryAfter
			}
			if delay > 0 {
				log.Warn().Int("attempt", attempt+1).Dur("delay", delay).Msg("Retrying request")
				time.Sleep(delay)
			}
		}

		retryAfter = 0
		c.waitForRateLimit()

		resp, err = c.doRequestOnce(method, path, bodyBytes, result)
		if err == nil {
			return resp, nil
		}

		var apiErr *APIError
		if errors.As(err, &apiErr) {
			if apiErr.RetryAfter > 0 {
				retryAfter = apiErr.RetryAfter
			}
			if apiErr.Status == http.StatusTooManyRequests || apiErr.Status >= 500 {
				continue
			}
			return resp, err
		}

		if isRetryableError(err) {
			continue
		}

		return resp, err
	}

	return resp, err
}

func (c *Client) doRequestOnce(method, path string, body []byte, result interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if len(body) > 0 {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, BaseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("trakt-api-version", APIVersion)
	req.Header.Set("trakt-api-key", c.clientID)

	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	c.updateRateLimit(resp.Header)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Error != "" {
			return resp, &APIError{
				Status:      resp.StatusCode,
				Code:        errResp.Error,
				Description: errResp.ErrorDescription,
				RetryAfter:  retryAfterDuration(resp.Header),
			}
		}
		return resp, &APIError{
			Status:      resp.StatusCode,
			Description: string(respBody),
			RetryAfter:  retryAfterDuration(resp.Header),
		}
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return resp, fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return resp, nil
}

func (c *Client) waitForRateLimit() {
	c.rateLimitMu.Lock()
	remaining := c.rateLimitRemaining
	reset := c.rateLimitReset
	c.rateLimitMu.Unlock()

	if remaining == 0 && !reset.IsZero() {
		now := time.Now()
		if now.Before(reset) {
			sleep := time.Until(reset)
			log.Warn().Dur("delay", sleep).Msg("Rate limit reached, waiting for reset")
			time.Sleep(sleep)
		}
	}
}

func (c *Client) updateRateLimit(headers http.Header) {
	remainingHeader := headers.Get("X-Ratelimit-Remaining")
	resetHeader := headers.Get("X-Ratelimit-Reset")

	var remaining int
	remainingSet := false
	if remainingHeader != "" {
		value, err := strconv.Atoi(remainingHeader)
		if err == nil {
			remaining = value
			remainingSet = true
		}
	}

	reset, resetSet := parseRateLimitReset(resetHeader, time.Now())

	c.rateLimitMu.Lock()
	if remainingSet {
		c.rateLimitRemaining = remaining
	}
	if resetSet {
		c.rateLimitReset = reset
	}
	c.rateLimitMu.Unlock()
}

func retryAfterDuration(headers http.Header) time.Duration {
	retryAfter := headers.Get("Retry-After")
	if retryAfter != "" {
		if seconds, err := strconv.Atoi(retryAfter); err == nil {
			return time.Duration(seconds) * time.Second
		}
		if t, err := http.ParseTime(retryAfter); err == nil {
			return time.Until(t)
		}
	}

	if reset, ok := parseRateLimitReset(headers.Get("X-Ratelimit-Reset"), time.Now()); ok {
		return time.Until(reset)
	}

	return 0
}

func parseRateLimitReset(value string, now time.Time) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}, false
	}

	if parsed > now.Unix()+60 {
		return time.Unix(parsed, 0), true
	}

	return now.Add(time.Duration(parsed) * time.Second), true
}

func isRetryableError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	return false
}

func backoffDuration(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	delay := baseBackoff * time.Duration(1<<uint(attempt-1))
	if delay > maxBackoff {
		delay = maxBackoff
	}
	return delay
}

// SetAccessToken updates the access token
func (c *Client) SetAccessToken(token string) {
	c.accessToken = token
}

// SetRefreshToken updates the refresh token
func (c *Client) SetRefreshToken(token string) {
	c.refreshToken = token
}
