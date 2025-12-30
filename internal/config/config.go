package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Trakt   TraktConfig   `mapstructure:"trakt"`
	Sync    SyncConfig    `mapstructure:"sync"`
	Logging LoggingConfig `mapstructure:"logging"`
}

// TraktConfig holds Trakt.tv API credentials and tokens
type TraktConfig struct {
	ClientID     string    `mapstructure:"client_id"`
	ClientSecret string    `mapstructure:"client_secret"`
	Username     string    `mapstructure:"username"`
	AccessToken  string    `mapstructure:"access_token"`
	RefreshToken string    `mapstructure:"refresh_token"`
	TokenExpires time.Time `mapstructure:"token_expires_at"`
}

// SyncConfig defines sync behavior
type SyncConfig struct {
	Limit           int              `mapstructure:"limit"`
	MinRating       int              `mapstructure:"min_rating"`
	ListPrivacy     string           `mapstructure:"list_privacy"`
	FullRefreshDays int              `mapstructure:"full_refresh_days"`
	LastFullRefresh FullRefreshState `mapstructure:"last_full_refresh"`
	Lists           ListSyncConfig   `mapstructure:"lists"`
}

// FullRefreshState keeps track of weekly full refresh timestamps.
type FullRefreshState struct {
	Movies time.Time `mapstructure:"movies"`
	Shows  time.Time `mapstructure:"shows"`
}

// ListSyncConfig defines which lists to sync
type ListSyncConfig struct {
	Movies bool `mapstructure:"movies"`
	Shows  bool `mapstructure:"shows"`
}

// LoggingConfig defines logging behavior
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// DefaultConfigPath returns the default config file path
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.yaml"
	}
	return filepath.Join(home, ".config", "trakt-sync", "config.yaml")
}

// Load reads and parses the config file
func Load(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = DefaultConfigPath()
	}

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	setDefaults(v)

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if err := createDefaultConfig(configPath); err != nil {
				return nil, fmt.Errorf("failed to create default config: %w", err)
			}
			if err := v.ReadInConfig(); err != nil {
				return nil, fmt.Errorf("failed to read config: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	var cfg Config
	decodeHook := mapstructure.ComposeDecodeHookFunc(stringToTimeHook())
	if err := v.Unmarshal(&cfg, viper.DecodeHook(decodeHook)); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// Save writes the config to disk
func Save(cfg *Config, configPath string) error {
	if configPath == "" {
		configPath = DefaultConfigPath()
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")
	setDefaults(v)

	privacy := strings.TrimSpace(cfg.Sync.ListPrivacy)
	if privacy == "" {
		privacy = "private"
	}

	v.Set("trakt.client_id", cfg.Trakt.ClientID)
	v.Set("trakt.client_secret", cfg.Trakt.ClientSecret)
	v.Set("trakt.username", cfg.Trakt.Username)
	v.Set("trakt.access_token", cfg.Trakt.AccessToken)
	v.Set("trakt.refresh_token", cfg.Trakt.RefreshToken)
	if cfg.Trakt.TokenExpires.IsZero() {
		v.Set("trakt.token_expires_at", "")
	} else {
		v.Set("trakt.token_expires_at", cfg.Trakt.TokenExpires.Format(time.RFC3339))
	}

	v.Set("sync.limit", cfg.Sync.Limit)
	v.Set("sync.min_rating", cfg.Sync.MinRating)
	v.Set("sync.list_privacy", privacy)
	v.Set("sync.full_refresh_days", cfg.Sync.FullRefreshDays)
	v.Set("sync.last_full_refresh.movies", formatTimeOrEmpty(cfg.Sync.LastFullRefresh.Movies))
	v.Set("sync.last_full_refresh.shows", formatTimeOrEmpty(cfg.Sync.LastFullRefresh.Shows))
	v.Set("sync.lists.movies", cfg.Sync.Lists.Movies)
	v.Set("sync.lists.shows", cfg.Sync.Lists.Shows)

	v.Set("logging.level", cfg.Logging.Level)
	v.Set("logging.format", cfg.Logging.Format)

	return v.WriteConfigAs(configPath)
}

// Validate checks if the config is valid
func (c *Config) Validate() error {
	if c.Trakt.ClientID == "" {
		return fmt.Errorf("trakt.client_id is required")
	}
	if c.Trakt.ClientSecret == "" {
		return fmt.Errorf("trakt.client_secret is required")
	}
	if c.Trakt.Username == "" {
		return fmt.Errorf("trakt.username is required")
	}
	if c.Sync.Limit <= 0 {
		return fmt.Errorf("sync.limit must be greater than 0")
	}
	if strings.TrimSpace(c.Sync.ListPrivacy) == "" {
		return fmt.Errorf("sync.list_privacy is required")
	}
	if c.Sync.FullRefreshDays <= 0 {
		return fmt.Errorf("sync.full_refresh_days must be greater than 0")
	}
	return nil
}

// IsAuthenticated checks if we have valid tokens
func (c *Config) IsAuthenticated() bool {
	return c.Trakt.AccessToken != "" && c.Trakt.RefreshToken != ""
}

// NeedsRefresh checks if the access token needs to be refreshed
func (c *Config) NeedsRefresh() bool {
	if c.Trakt.AccessToken == "" {
		return false
	}
	return time.Now().Add(1 * time.Hour).After(c.Trakt.TokenExpires)
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("sync.limit", 30)
	v.SetDefault("sync.min_rating", 60)
	v.SetDefault("sync.list_privacy", "private")
	v.SetDefault("sync.full_refresh_days", 7)
	v.SetDefault("sync.lists.movies", true)
	v.SetDefault("sync.lists.shows", true)
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "text")
}

func createDefaultConfig(path string) error {
	cfg := defaultConfig()
	return Save(cfg, path)
}

func defaultConfig() *Config {
	return &Config{
		Trakt: TraktConfig{},
		Sync: SyncConfig{
			Limit:           30,
			MinRating:       60,
			ListPrivacy:     "private",
			FullRefreshDays: 7,
			Lists: ListSyncConfig{
				Movies: true,
				Shows:  true,
			},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

func formatTimeOrEmpty(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func stringToTimeHook() mapstructure.DecodeHookFunc {
	return func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
		if from.Kind() == reflect.String && to == reflect.TypeOf(time.Time{}) {
			value := strings.TrimSpace(data.(string))
			if value == "" {
				return time.Time{}, nil
			}
			parsed, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return data, err
			}
			return parsed, nil
		}
		return data, nil
	}
}
