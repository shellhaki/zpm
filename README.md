# ZPM

[![Go](https://img.shields.io/badge/Go-1.22-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![Linux](https://img.shields.io/badge/Linux-ready-FCC624?style=for-the-badge&logo=linux&logoColor=000)](#)
[![macOS](https://img.shields.io/badge/macOS-ready-000000?style=for-the-badge&logo=apple)](#)
[![Windows](https://img.shields.io/badge/Windows-ready-0078D4?style=for-the-badge&logo=windows)](#)

**ZPM is a small, sharp process manager for apps you want to keep alive.**

It gives you a daemon, named processes, crash restart, log following, clusters, environment profiles, health checks, log rotation, startup on boot, and ecosystem config files without turning your terminal into a carnival.

## Install From Source

```bash
go build -o daemon/zpmd ./daemon
go build -o src/zpm ./src
ln -sf "$PWD/src/zpm" ~/.local/bin/zpp
```

Keep `zpmd` beside the CLI or set:

```bash
export ZPMD_PATH=/path/to/zpmd
```

## Daemon

```bash
zpp daemon start
zpp daemon stop
zpp daemon reload
```

Start on login:

```bash
zpp startup install
zpp startup uninstall
```

Linux uses user systemd, macOS uses LaunchAgent, Windows uses Task Scheduler.

## Run Apps

```bash
zpp start "bun index" --name api --follow
zpp status
zpp stop api
zpp restart api
zpp start api
zpp purge api
```

ZPM starts commands from the directory where you run `zpp start`, so package scripts work naturally.

```json
{
  "scripts": {
    "serve:zpm": "zpp start \"bun index\" --name api --follow"
  }
}
```

## PM2-Style Features

```bash
zpp start "bun index" --name api --env production
zpp start "bun index" --name api --env PORT=3000
zpp start "bun index" --name api --instances 4
zpp start "bun index" --name api --restart-delay 1000 --max-restarts 10
zpp start "bun index" --name api --health "curl -fsS http://127.0.0.1:3000/health"
zpp start "bun index" --name api --log-max-size 20mb --log-backups 7
```

Cluster instances are named `api-0`, `api-1`, etc. Group commands work:

```bash
zpp stop api
zpp restart api
zpp purge api
```

## Ecosystem Config

Create `zpm.config.json`:

```json
{
  "apps": [
    {
      "name": "api",
      "command": "bun index",
      "cwd": ".",
      "instances": 2,
      "auto_restart": true,
      "restart_delay": 1000,
      "max_restarts": -1,
      "health_command": "curl -fsS http://127.0.0.1:3000/health",
      "log_max_bytes": 10485760,
      "log_backups": 5,
      "env": {
        "PORT": "3000"
      },
      "env_production": {
        "NODE_ENV": "production"
      }
    }
  ]
}
```

Run it:

```bash
zpp ecosystem start zpm.config.json --env production
```

## Storage

ZPM stores state in your user config directory:

```text
registry.json
daemon.pid
daemon.log
logs/<process>.log
```

On Linux that is usually `~/.config/zpm`.

## Release

Push a tag to publish cross-platform builds:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The release pipeline builds:

- Linux amd64, arm64
- macOS amd64, arm64
- Windows amd64
