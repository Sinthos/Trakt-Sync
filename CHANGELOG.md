# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- **Rate limit handling**: Fixed edge case where rate limit wait logic could fail if reset time is zero or in the past
- **Flag parsing**: Fixed unhandled error when parsing `--lists` flag in sync command (now properly fails with error message)
- **Exit codes**: Improved exit code logic to distinguish between different error types (exit code 3 for config/auth errors, 2 for all lists failed, 1 for partial failure)
- **Backoff calculation**: Added overflow protection for exponential backoff calculation to prevent panic with large retry attempts
- **Config persistence**: Changed config save error from ERROR to WARN level with explicit message about implications (prevents misleading success status)
- **Logging setup**: Removed redundant double logging setup in PersistentPreRun (now only sets up once after config load)
- **Path validation**: Added security validation for service installation path to prevent directory traversal attacks
- **HTTP timeout**: Increased HTTP client timeout from 30s to 60s for improved reliability on slow networks or large API responses

### Added
- Docker support with multi-platform builds (linux/amd64, linux/arm64)
- Docker Compose configuration for easy deployment
- Harbor registry deployment documentation

### Security
- Service installation now validates paths to prevent directory traversal (requires absolute paths, rejects `..` in paths)
