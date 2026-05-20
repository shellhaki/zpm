# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Process spawning with fork and setsid
- Process registry with JSON persistence
- CLI commands: start, stop, list, purge
- PID tracking for running processes
- SIGTERM signal handling for graceful stops
- Daemon server (zpmd) with Unix socket support
- Test server (Bun + Hono) for integration testing
- Comprehensive documentation and contributing guidelines
- Package.json script support for `zpm start --name app` and `--script <name>`
- Release installer and Linux/macOS cross-build archives

### Fixed
- Memory management in process registry loading
- String lifetime issues with parsed JSON
- Socket initialization for cross-platform compatibility
- `zpm start` now stores the spawned process PID instead of `0`
- `zpm stop` now sends SIGTERM before marking a process as stopped

### Changed
- Simplified socket communication (direct registry access for now)
- Refactored main client to use pure registry operations
- README now focuses on install, build, usage, and next TODOs

## [0.1.0] - 2026-05-20

### Added
- Initial project structure
- Build system with Zig
- Basic process management skeleton
- Test server setup

---

## Release History

### Version 0.1.0 (Initial Release)
- Foundation for process management
- Core registry implementation
- CLI framework
- Project documentation

## Planned Features

### Version 0.2.0 (Next)
- [ ] Socket-based daemon communication
- [ ] Process auto-restart on crash
- [ ] Logs aggregation
- [ ] Resource limit configuration

### Version 0.3.0
- [ ] Process groups
- [ ] Process restart limits
- [ ] Event system (webhooks)
- [ ] HTTP REST API

### Version 0.4.0
- [ ] Web dashboard
- [ ] Multi-user support
- [ ] User authentication
- [ ] Role-based access control

### Version 1.0.0
- [ ] Cross-platform support (Windows, macOS)
- [ ] Clustering support
- [ ] Performance optimizations
- [ ] Comprehensive test suite

## Known Issues

- Socket communication not yet functional
- No log aggregation
- No process restart capability
- Single-user only

## Migration Guide

### From 0.1.0 to 0.2.0

No breaking changes expected.

## Contributors

See [CONTRIBUTORS.md](CONTRIBUTORS.md) for the list of contributors.

## How to Report Bugs

Please open an issue with:
- Steps to reproduce
- Expected vs actual behavior
- Environment (OS, Zig version)
- ZPM version

## How to Request Features

Open a discussion or issue describing:
- Problem statement
- Proposed solution
- Use cases
- Alternatives considered

---

**Latest Version**: 0.1.0
**Last Updated**: May 20, 2026
