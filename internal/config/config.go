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
	Limit       int            `mapstructure:"limit"`
	ListPrivacy string         `mapstructure:"list_privacy"`
	Lists       ListSyncConfig `mapstructure:"lists"`
}

// ListSyncConfig defines which lists to sync
type ListSyncConfig struct {
	TrendingMovies  bool `mapstructure:"trending_movies"`
	TrendingShows   bool `mapstructure:"trending_shows"`
	PopularMovies   bool `mapstructure:"popular_movies"`
	PopularShows    bool `mapstructure:"popular_shows"`
	StreamingMovies bool `mapstructure:"streaming_movies"`
	StreamingShows  bool `mapstructure:"streaming_shows"`
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
	v.Set("sync.list_privacy", privacy)
	v.Set("sync.lists.trending_movies", cfg.Sync.Lists.TrendingMovies)
	v.Set("sync.lists.trending_shows", cfg.Sync.Lists.TrendingShows)
	v.Set("sync.lists.popular_movies", cfg.Sync.Lists.PopularMovies)
	v.Set("sync.lists.popular_shows", cfg.Sync.Lists.PopularShows)
	v.Set("sync.lists.streaming_movies", cfg.Sync.Lists.StreamingMovies)
	v.Set("sync.lists.streaming_shows", cfg.Sync.Lists.StreamingShows)

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
	v.SetDefault("sync.limit", 20)
	v.SetDefault("sync.list_privacy", "private")
	v.SetDefault("sync.lists.trending_movies", true)
	v.SetDefault("sync.lists.trending_shows", true)
	v.SetDefault("sync.lists.popular_movies", true)
	v.SetDefault("sync.lists.popular_shows", true)
	v.SetDefault("sync.lists.streaming_movies", true)
	v.SetDefault("sync.lists.streaming_shows", true)
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
			Limit:       20,
			ListPrivacy: "private",
			Lists: ListSyncConfig{
				TrendingMovies:  true,
				TrendingShows:   true,
				PopularMovies:   true,
				PopularShows:    true,
				StreamingMovies: true,
				StreamingShows:  true,
			},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}
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
