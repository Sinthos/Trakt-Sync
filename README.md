# Trakt Sync

A Go-based tool to automatically synchronize Trakt.tv lists with trending, popular, and most-watched movies and shows.

## Features

- **OAuth2 Authentication** - Secure device code flow authentication with automatic token refresh
- **6 Auto-Synced Lists** - Tracks trending, popular, and streaming charts for both movies and shows
- **Daemon Mode** - Run continuously with configurable sync intervals
- **Smart Diffing** - Only adds/removes items that have changed
- **Rate Limiting** - Respects Trakt API limits with automatic backoff
- **Cross-Platform** - Builds for Linux (AMD64, ARM64) and other platforms
- **Lightweight** - Single binary with minimal resource usage

## Synced Lists

The tool maintains these 6 lists on your Trakt.tv account:

| List Slug | Description | Source API |
|-----------|-------------|------------|
| `trending-movies` | Top 20 trending movies | `/movies/trending` |
| `trending-shows` | Top 20 trending shows | `/shows/trending` |
| `popular-movies` | Top 20 popular movies | `/movies/popular` |
| `popular-shows` | Top 20 popular shows | `/shows/popular` |
| `streaming-charts-movies` | Top 20 most watched movies (weekly) | `/movies/watched/weekly` |
| `streaming-charts-shows` | Top 20 most watched shows (weekly) | `/shows/watched/weekly` |

## Installation

### Prerequisites

- Go 1.21 or higher
- A Trakt.tv account
- Trakt.tv API credentials (get them at https://trakt.tv/oauth/applications)

### Build from Source

```bash
# Clone the repository
git clone https://github.com/Sinthos/trakt-sync.git
cd trakt-sync

# Download dependencies
make deps

# Build for current platform
make build

# Or build for specific platforms
make build-linux      # Linux AMD64
make build-linux-arm  # Linux ARM64
make build-all        # All platforms

# Install to /usr/local/bin
make install
```

The binary will be in the `bin/` directory.

## Configuration

### Initial Setup

1. Create your config file:
   ```bash
   mkdir -p ~/.config/trakt-sync
   cp config.example.yaml ~/.config/trakt-sync/config.yaml
   ```

2. Edit the config file with your Trakt.tv credentials:
   ```yaml
   trakt:
     client_id: "your-client-id"
     client_secret: "your-client-secret"
     username: "your-username"
   ```

3. Get API credentials from https://trakt.tv/oauth/applications:
   - Click "New Application"
   - Fill in the details (Redirect URI can be `urn:ietf:wg:oauth:2.0:oob`)
   - Copy the Client ID and Client Secret

### Configuration Options

See `config.example.yaml` for all available options:

- **sync.limit** - Number of items per list (default: 20)
- **sync.list_privacy** - Privacy for auto-created lists (default: private)
- **sync.lists** - Enable/disable specific lists
- **logging.level** - Log level: debug, info, warn, error (default: info)
- **logging.format** - Log format: text or json (default: text)

## Usage

### Authenticate

First, authenticate with Trakt.tv:

```bash
trakt-sync auth
```

This will:
1. Display a URL and code
2. You visit the URL and enter the code
3. Tokens are automatically saved to your config

### Sync Lists

Run a one-time sync:

```bash
trakt-sync sync
```

Sync only specific lists:

```bash
trakt-sync sync --lists trending-movies,popular-shows
```

### Daemon Mode

Run continuously with automatic syncing:

```bash
# Sync every 6 hours (default)
trakt-sync daemon

# Custom interval
trakt-sync daemon --interval 3h
trakt-sync daemon --interval 30m
```

### Check Status

View authentication and configuration status:

```bash
trakt-sync status
```

### Validate Config

Check if your configuration is valid:

```bash
trakt-sync config validate
```

### Other Commands

```bash
# Show version
trakt-sync version

# Use custom config file
trakt-sync --config /path/to/config.yaml sync

# Verbose logging
trakt-sync --verbose sync

# Dry run (no API calls)
trakt-sync --dry-run sync

# Generate systemd service file
trakt-sync install-service
```

## Deployment

### systemd Service

Generate a systemd service file automatically:

```bash
sudo trakt-sync install-service

# Optional overrides
sudo trakt-sync install-service --user trakt-sync --interval 6h --path /etc/systemd/system/trakt-sync.service
```

Or create a systemd service file manually at `/etc/systemd/system/trakt-sync.service`:

```ini
[Unit]
Description=Trakt List Sync Service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=your-user
ExecStart=/usr/local/bin/trakt-sync daemon --interval 6h
Restart=on-failure
RestartSec=30

[Install]
WantedBy=multi-user.target
```

Then enable and start the service:

```bash
sudo systemctl enable trakt-sync
sudo systemctl start trakt-sync
sudo systemctl status trakt-sync
```

### Docker

Create a `Dockerfile`:

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o trakt-sync ./cmd/trakt-sync

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/trakt-sync .
COPY config.yaml /root/.config/trakt-sync/config.yaml
CMD ["./trakt-sync", "daemon"]
```

Build and run:

```bash
docker build -t trakt-sync .
docker run -d --name trakt-sync -v ~/.config/trakt-sync:/root/.config/trakt-sync trakt-sync
```

## Development

### Project Structure

```
trakt-sync/
├── cmd/trakt-sync/      # CLI entry point
│   └── main.go
├── internal/
│   ├── config/          # Configuration management
│   ├── trakt/           # Trakt API client
│   │   ├── client.go    # HTTP client
│   │   ├── auth.go      # OAuth2 device flow
│   │   ├── types.go     # API types
│   │   ├── movies.go    # Movie endpoints
│   │   ├── shows.go     # Show endpoints
│   │   └── lists.go     # List management
│   └── sync/            # Sync logic
│       └── sync.go
├── Makefile
├── go.mod
├── config.example.yaml
└── README.md
```

### Running Tests

```bash
make test
```

### Linting

```bash
# Install golangci-lint first
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
make lint
```

## API Documentation

Trakt API documentation: https://trakt.docs.apiary.io/

### Important Notes

- All API calls require headers:
  - `trakt-api-version: 2`
  - `trakt-api-key: {client_id}`
- Access tokens expire after 3 months
- Refresh tokens expire after 1 year
- Rate limit: 1000 calls per 5 minutes

## Troubleshooting

### Authentication Issues

If authentication fails:
1. Check your client ID and secret
2. Ensure you've created an application at https://trakt.tv/oauth/applications
3. Try authenticating again: `trakt-sync auth`

### Token Expired

Tokens are automatically refreshed, but if you see token errors:
1. Re-authenticate: `trakt-sync auth`
2. Check token expiry: `trakt-sync status`

### Rate Limiting

The tool automatically handles rate limiting with backoff. If you see many rate limit errors:
- Increase the daemon interval
- Reduce the number of enabled lists

### Logs

Enable verbose logging for debugging:
```bash
trakt-sync --verbose sync
```

Or set log level in config:
```yaml
logging:
  level: "debug"
```

## Contributing

Contributions are welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Run `make lint` and `make test`
6. Submit a pull request

## License

MIT License - see LICENSE file for details

## Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) for CLI
- Uses [Viper](https://github.com/spf13/viper) for configuration
- Logging with [Zerolog](https://github.com/rs/zerolog)
- Powered by the [Trakt.tv API](https://trakt.tv)
