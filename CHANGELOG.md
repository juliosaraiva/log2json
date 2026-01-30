# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- `make lint` target for local linting
- `make vet` target for go vet
- `make check` target for pre-push validation (lint + vet + test)
- `make release` target for streamlined tag creation
- `make version` target showing git-derived version
- Git-derived version in Makefile (replaces hardcoded version)
- CI uploads cross-compile artifacts with 7-day retention
- CONTRIBUTING.md with development workflow and guidelines

### Fixed
- CI cache warnings ("go.sum not found") by disabling Go module caching

## [0.1.0] - 2025-01-29

### Added
- Initial release
- Streaming log-to-JSON converter reading from stdin, writing to stdout
- Auto-detection of log formats with priority-based parser registry
- Supported formats: syslog, Apache combined, JSON (NDJSON), key-value (logfmt), generic
- Custom regex patterns with named capture groups (`--pattern`)
- Adaptive mode for mixed-format log streams (`--adaptive`)
- Field filtering (`--fields`), pretty-print (`--pretty`)
- Metadata injection: `--add-timestamp`, `--add-line-number`, `--add-raw`
- Parse error handling with `--omit-empty`
- Startup validation for unknown format names
- Comprehensive test suite (96-100% coverage on library packages)
- CI pipeline: 3 OS x 3 Go versions + lint + cross-compile
- Release automation with GoReleaser (6 platform/arch binaries)
- MIT License

[Unreleased]: https://github.com/juliosaraiva/log2json/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/juliosaraiva/log2json/releases/tag/v0.1.0
