# ZPM
(README IS AI GENERATED !!!)

ZPM is a small Zig process manager for starting, listing, stopping, and removing local background processes.

Current support: Linux and macOS. Windows support is planned, but the current code uses POSIX process APIs.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/shellhaki/zpm/main/scripts/install.sh | sh
```

The installer downloads the latest GitHub release for your OS/CPU and installs `zpm` and `zpmd` into `~/.local/bin`.

To install a specific release:

```bash
curl -fsSL https://raw.githubusercontent.com/shellhaki/zpm/main/scripts/install.sh | ZPM_TAG=v0.1.0 sh
```

## Build From Source

```bash
git clone https://github.com/shellhaki/zpm.git
cd zpm
zig build
./zig-out/bin/zpm list
```

Cross-platform release archives can be built with:

```bash
scripts/build-cross.sh
```

## Commands

```bash
zpm start --name api "node server.js"
zpm start --name web              # uses package.json scripts.start
zpm start --name web --script dev # uses package.json scripts.dev
zpm list
zpm stop api
zpm purge api
```

ZPM stores process state in `zpm.json` in the directory where you run it.

## Releases

Pushing to `main` runs GitHub Actions, cross-compiles Linux/macOS archives, and publishes a rolling `main-latest` GitHub release.

Pushing a `v*` tag publishes that tag as a normal release:

```bash
git tag v0.1.0
git push origin v0.1.0
```

## Project Files

- `src/main.zig` - CLI entry point
- `src/registry.zig` - process registry, spawning, and stop logic
- `src/handlers/` - command handlers
- `src/daemon.zig` - daemon skeleton for the next IPC step
- `scripts/install.sh` - curl installer
- `scripts/build-cross.sh` - release archive builder
- `.github/workflows/release.yml` - automatic release pipeline

## More Docs

- [QUICKSTART.md](QUICKSTART.md)
- [INSTALL.md](INSTALL.md)
- [ARCHITECTURE.md](ARCHITECTURE.md)
- [ROADMAP.md](ROADMAP.md)

## License

MIT. See [LICENSE](LICENSE).

## TODO: Continue Here

- [x] Keep README short and focused on install/build/use.
- [x] Add `scripts/install.sh` for curl-based setup.
- [x] Add `scripts/build-cross.sh` for Linux/macOS release archives.
- [x] Add GitHub Actions release publishing on pushes to `main`.
- [x] Store real PIDs when `zpm start` spawns a command.
- [ ] Add tests for `start`, `stop`, `list`, and `purge`.
- [ ] Replace direct registry writes with `zpm` to `zpmd` socket IPC.
- [ ] Move registry storage out of the current working directory or make it configurable.
- [ ] Add log capture and restart-on-crash support.
- [ ] Add Windows support or clearly split POSIX-only code paths.

Last touched: May 20, 2026.
