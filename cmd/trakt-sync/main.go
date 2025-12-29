package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/maximilian/trakt-sync/internal/config"
	syncpkg "github.com/maximilian/trakt-sync/internal/sync"
	"github.com/maximilian/trakt-sync/internal/trakt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	cfgFile string
	verbose bool
	dryRun  bool
	cfg     *config.Config

	servicePath     string
	serviceUser     string
	serviceInterval time.Duration
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "trakt-sync",
	Short: "Sync Trakt.tv lists with trending and streaming charts",
	Long:  "A tool to automatically synchronize Trakt.tv lists with top trending and most watched movies and shows.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		setupLogging()

		if cmd.Name() == "version" {
			return
		}

		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to load config")
		}

		setupLogging()
	},
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Trakt.tv",
	Long:  "Initiates OAuth2 device flow to authenticate with Trakt.tv and stores tokens.",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runAuth(); err != nil {
			log.Fatal().Err(err).Msg("Authentication failed")
		}
	},
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync lists once",
	Long:  "Performs a one-time sync of all enabled lists.",
	Run: func(cmd *cobra.Command, args []string) {
		lists, _ := cmd.Flags().GetString("lists")
		result, err := runSync(lists)
		if err != nil {
			log.Error().Err(err).Msg("Sync failed")
		}
		exitCode := syncExitCode(result, err)
		if exitCode != 0 {
			os.Exit(exitCode)
		}
	},
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run as daemon with periodic syncing",
	Long:  "Runs continuously and syncs lists at the specified interval.",
	Run: func(cmd *cobra.Command, args []string) {
		interval, _ := cmd.Flags().GetDuration("interval")
		if err := runDaemon(interval); err != nil {
			log.Fatal().Err(err).Msg("Daemon failed")
		}
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication and configuration status",
	Long:  "Displays current authentication status and configuration.",
	Run: func(cmd *cobra.Command, args []string) {
		runStatus()
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration",
	Long:  "Validates the configuration file.",
	Run: func(cmd *cobra.Command, args []string) {
		if err := cfg.Validate(); err != nil {
			log.Error().Err(err).Msg("Configuration is invalid")
			os.Exit(1)
		}
		log.Info().Msg("Configuration is valid")
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration commands",
	Long:  "Commands for managing configuration.",
}

var installServiceCmd = &cobra.Command{
	Use:   "install-service",
	Short: "Install systemd service file",
	Long:  "Generates a systemd service file for running trakt-sync in daemon mode.",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runInstallService(servicePath, serviceUser, serviceInterval); err != nil {
			log.Fatal().Err(err).Msg("Failed to install systemd service")
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version",
	Long:  "Displays the version of trakt-sync.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("trakt-sync version %s\n", Version)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default: ~/.config/trakt-sync/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "show what would happen without making changes")

	syncCmd.Flags().String("lists", "", "comma-separated list slugs to sync (e.g., trakt-sync-filme,trakt-sync-serien)")

	daemonCmd.Flags().Duration("interval", 6*time.Hour, "sync interval")

	installServiceCmd.Flags().StringVar(&servicePath, "path", "/etc/systemd/system/trakt-sync.service", "systemd service file path")
	installServiceCmd.Flags().StringVar(&serviceUser, "user", "trakt-sync", "systemd service user")
	installServiceCmd.Flags().DurationVar(&serviceInterval, "interval", 6*time.Hour, "sync interval for the service")

	configCmd.AddCommand(configValidateCmd)

	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(installServiceCmd)
	rootCmd.AddCommand(versionCmd)
}

func setupLogging() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "2006-01-02 15:04:05"})

	level := zerolog.InfoLevel
	format := "text"

	if cfg != nil {
		switch strings.ToLower(cfg.Logging.Level) {
		case "debug":
			level = zerolog.DebugLevel
		case "info":
			level = zerolog.InfoLevel
		case "warn":
			level = zerolog.WarnLevel
		case "error":
			level = zerolog.ErrorLevel
		}

		if strings.ToLower(cfg.Logging.Format) == "json" {
			format = "json"
		}
	}

	if verbose {
		level = zerolog.DebugLevel
	}

	zerolog.SetGlobalLevel(level)

	if format == "json" {
		log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}
}

func runAuth() error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	client := trakt.NewClient(cfg.Trakt.ClientID, cfg.Trakt.ClientSecret, "", "")

	deviceResp, err := client.GetDeviceCode()
	if err != nil {
		return err
	}

	fmt.Println("\nPlease authenticate by visiting:")
	fmt.Printf("\n  %s\n\n", deviceResp.VerificationURL)
	fmt.Printf("And enter this code: %s\n\n", deviceResp.UserCode)
	fmt.Println("Waiting for authorization...")

	tokenResp, err := client.PollForToken(deviceResp.DeviceCode, deviceResp.Interval, deviceResp.ExpiresIn)
	if err != nil {
		return err
	}

	cfg.Trakt.AccessToken = tokenResp.AccessToken
	cfg.Trakt.RefreshToken = tokenResp.RefreshToken
	cfg.Trakt.TokenExpires = time.Unix(tokenResp.CreatedAt, 0).Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	configPath := cfgFile
	if configPath == "" {
		configPath = config.DefaultConfigPath()
	}

	if err := config.Save(cfg, configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	log.Info().Msg("Authentication successful! Tokens saved to config.")
	return nil
}

func runSync(listsFilter string) (syncpkg.SyncResult, error) {
	if err := cfg.Validate(); err != nil {
		return syncpkg.SyncResult{}, fmt.Errorf("config validation failed: %w", err)
	}

	if !dryRun && !cfg.IsAuthenticated() {
		return syncpkg.SyncResult{}, fmt.Errorf("not authenticated. Run 'trakt-sync auth' first")
	}

	client := trakt.NewClient(
		cfg.Trakt.ClientID,
		cfg.Trakt.ClientSecret,
		cfg.Trakt.AccessToken,
		cfg.Trakt.RefreshToken,
	)

	if !dryRun {
		client.SetTokenRefreshCallback(func(accessToken, refreshToken string, expiresAt time.Time) {
			cfg.Trakt.AccessToken = accessToken
			cfg.Trakt.RefreshToken = refreshToken
			cfg.Trakt.TokenExpires = expiresAt

			configPath := cfgFile
			if configPath == "" {
				configPath = config.DefaultConfigPath()
			}

			if err := config.Save(cfg, configPath); err != nil {
				log.Error().Err(err).Msg("Failed to save refreshed tokens")
			}
		})

		if cfg.NeedsRefresh() {
			log.Info().Msg("Access token expired, refreshing...")
			if _, err := client.RefreshAccessToken(); err != nil {
				return syncpkg.SyncResult{}, fmt.Errorf("failed to refresh token: %w", err)
			}
		}
	}

	if listsFilter != "" {
		requestedLists := strings.Split(listsFilter, ",")
		cfg.Sync.Lists = config.ListSyncConfig{}
		for _, listSlug := range requestedLists {
			listSlug = strings.TrimSpace(listSlug)
			switch listSlug {
			case "trakt-sync-filme":
				cfg.Sync.Lists.Movies = true
			case "trakt-sync-serien":
				cfg.Sync.Lists.Shows = true
			default:
				log.Warn().Str("list", listSlug).Msg("Unknown list slug")
			}
		}
	}

	syncer := syncpkg.NewSyncer(client, cfg)

	if dryRun {
		log.Info().Msg("DRY RUN: No API calls will be made")
		result := syncpkg.SyncResult{}
		for _, listDef := range syncer.GetListDefinitions() {
			if !listDef.Enabled {
				continue
			}
			result.Total++
			result.Successful++
			log.Info().Str("list", listDef.Slug).Int("limit", cfg.Sync.Limit).Msg("DRY RUN: would sync list")
		}
		return result, nil
	}

	result, err := syncer.SyncAll()

	if !dryRun && syncer.ConfigDirty() {
		configPath := cfgFile
		if configPath == "" {
			configPath = config.DefaultConfigPath()
		}

		if saveErr := config.Save(cfg, configPath); saveErr != nil {
			log.Error().Err(saveErr).Msg("Failed to save sync state")
		}
	}

	return result, err
}

func runDaemon(interval time.Duration) error {
	if !dryRun && !cfg.IsAuthenticated() {
		return fmt.Errorf("not authenticated. Run 'trakt-sync auth' first")
	}

	log.Info().Dur("interval", interval).Msg("Starting daemon mode")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	if _, err := runSync(""); err != nil {
		log.Error().Err(err).Msg("Initial sync failed")
	}

	for range ticker.C {
		if _, err := runSync(""); err != nil {
			log.Error().Err(err).Msg("Sync failed")
		}
	}

	return nil
}

func runStatus() {
	configPath := cfgFile
	if configPath == "" {
		configPath = config.DefaultConfigPath()
	}

	fmt.Println("Trakt Sync Status")
	fmt.Println("=================")
	fmt.Printf("Config file: %s\n", configPath)
	fmt.Printf("Username: %s\n", cfg.Trakt.Username)
	fmt.Printf("Authenticated: %v\n", cfg.IsAuthenticated())

	if cfg.IsAuthenticated() {
		fmt.Printf("Token expires: %s\n", cfg.Trakt.TokenExpires.Format(time.RFC3339))
		if cfg.NeedsRefresh() {
			fmt.Println("Token needs refresh: YES")
		} else {
			fmt.Println("Token needs refresh: NO")
		}
	}

	fmt.Println("\nEnabled Lists:")
	if cfg.Sync.Lists.Movies {
		fmt.Println("  - trakt-sync-filme")
	}
	if cfg.Sync.Lists.Shows {
		fmt.Println("  - trakt-sync-serien")
	}

	fmt.Printf("\nSync limit: %d items per source\n", cfg.Sync.Limit)
	fmt.Printf("List privacy: %s\n", cfg.Sync.ListPrivacy)
	fmt.Printf("Full refresh: every %d days\n", cfg.Sync.FullRefreshDays)
}

func runInstallService(path, user string, interval time.Duration) error {
	if strings.TrimSpace(user) == "" {
		return fmt.Errorf("service user must not be empty")
	}
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("service path must not be empty")
	}

	serviceFile := fmt.Sprintf(`[Unit]
Description=Trakt List Sync Service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=%s
ExecStart=/usr/local/bin/trakt-sync daemon --interval %s
Restart=on-failure
RestartSec=30

[Install]
WantedBy=multi-user.target
`, user, interval.String())

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create service directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(serviceFile), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	log.Info().Str("path", path).Msg("Systemd service installed")
	return nil
}

func syncExitCode(result syncpkg.SyncResult, err error) int {
	if err != nil {
		if errors.Is(err, syncpkg.ErrAllFailed) {
			return 2
		}
		return 2
	}

	if result.Total == 0 {
		return 0
	}

	if result.Failed > 0 {
		if result.Successful == 0 {
			return 2
		}
		return 1
	}

	return 0
}
