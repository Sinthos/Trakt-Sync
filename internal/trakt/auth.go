package trakt

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// GetDeviceCode initiates the device code flow
func (c *Client) GetDeviceCode() (*DeviceCodeResponse, error) {
	var resp DeviceCodeResponse
	_, err := c.doRequest("POST", "/oauth/device/code", map[string]string{
		"client_id": c.clientID,
	}, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get device code: %w", err)
	}
	return &resp, nil
}

// PollForToken polls the token endpoint until the user authorizes or the code expires
func (c *Client) PollForToken(deviceCode string, interval int, expiresIn int) (*TokenResponse, error) {
	if interval <= 0 {
		interval = 5
	}
	if expiresIn <= 0 {
		expiresIn = 10 * 60
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	timeout := time.After(time.Duration(expiresIn) * time.Second)

	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("authorization timeout")
		case <-ticker.C:
			token, err := c.requestToken(deviceCode)
			if err != nil {
				var apiErr *APIError
				if errors.As(err, &apiErr) {
					switch apiErr.Code {
					case "authorization_pending":
						log.Debug().Msg("Still waiting for user authorization...")
						continue
					case "slow_down":
						interval += 5
						ticker.Reset(time.Duration(interval) * time.Second)
						log.Debug().Int("interval", interval).Msg("Slowing down device code polling")
						continue
					case "access_denied":
						return nil, fmt.Errorf("user denied authorization")
					case "expired_token":
						return nil, fmt.Errorf("device code expired")
					}

					if apiErr.Code == "" && apiErr.Status == http.StatusBadRequest {
						log.Debug().Str("detail", apiErr.Description).Msg("Device code not authorized yet")
						continue
					}
				}
				return nil, err
			}

			c.accessToken = token.AccessToken
			c.refreshToken = token.RefreshToken

			return token, nil
		}
	}
}

// requestToken requests an access token with the device code
func (c *Client) requestToken(deviceCode string) (*TokenResponse, error) {
	var resp TokenResponse
	_, err := c.doRequest("POST", "/oauth/device/token", map[string]string{
		"code":          deviceCode,
		"client_id":     c.clientID,
		"client_secret": c.clientSecret,
	}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// RefreshAccessToken refreshes the access token using the refresh token
func (c *Client) RefreshAccessToken() (*TokenResponse, error) {
	if c.refreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	var resp TokenResponse
	_, err := c.doRequest("POST", "/oauth/token", map[string]string{
		"refresh_token": c.refreshToken,
		"client_id":     c.clientID,
		"client_secret": c.clientSecret,
		"grant_type":    "refresh_token",
	}, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	c.accessToken = resp.AccessToken
	c.refreshToken = resp.RefreshToken

	if c.onTokenRefresh != nil {
		expiresAt := time.Unix(resp.CreatedAt, 0).Add(time.Duration(resp.ExpiresIn) * time.Second)
		c.onTokenRefresh(resp.AccessToken, resp.RefreshToken, expiresAt)
	}

	log.Info().Msg("Access token refreshed successfully")
	return &resp, nil
}
