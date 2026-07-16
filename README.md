# 🌊 DailyFlow

**A lightweight web interface for local Markdown journals.**

---

[![Go Version](https://img.shields.io/badge/Go-1.26.2+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![Build Status](https://img.shields.io/badge/Build-Success-4CAF50?style=for-the-badge&logo=github-actions)](https://github.com/Xarth-Mai/DailyFlow/actions)
[![License](https://img.shields.io/badge/License-MPL--2.0-brightgreen.svg?style=for-the-badge)](LICENSE)

DailyFlow turns a directory of Markdown files into a responsive, searchable journal. It keeps the source files as the source of truth: no database, frontend framework, bundler, or runtime package manager is required.

## ✨ Key Features

- 📁 **Filesystem Journals**: Recursively discovers and sorts `.md` files without importing them into a database.
- 📖 **Focused Timeline**: Expands the seven newest entries by default and keeps older entries behind `Read More`.
- 🔍 **Compact Search**: Returns titles and matching snippets with safe highlighting; results stay collapsed until expanded in place.
- 🗓️ **Month Navigation**: Filters the timeline with a compact month selector.
- 📱 **Responsive Reading**: Keeps the mobile toolbar on one row and automatically highlights the entry nearest the visual center while scrolling.
- 🔐 **Secure Access**: Bcrypt passwords, HMAC-signed sessions, hardened cookies, logout, and safe same-origin login returns.
- 🛡️ **Safe Markdown and Filesystem Access**: Escapes raw HTML, rejects unsafe link schemes, and prevents workspace traversal and symlink escapes.
- 📦 **Single-Binary Deployment**: Embeds the frontend and vendored Marked parser into the Go binary.
- 🎨 **Light and Dark Themes**: Uses one shared theme controller across the application and login page.

## 🚀 Quick Start

### 1. Build or Download

Download a binary from [GitHub Actions](https://github.com/Xarth-Mai/DailyFlow/actions), or build from source:

```bash
git clone https://github.com/Xarth-Mai/DailyFlow.git
cd DailyFlow
go build -o dailyflow .
```

### 2. Configure

Create or update the configuration file through the CLI:

```bash
./dailyflow -setdir "path/to/Daily"
./dailyflow -setuser "MyUsername"
./dailyflow -setpass
```

Use `-c` when the configuration is not `./config.conf`:

```bash
./dailyflow -c /etc/dailyflow/config.conf -setpass
```

For an HTTPS deployment behind a reverse proxy, add:

```ini
COOKIE_SECURE=true
```

DailyFlow generates and persists a cryptographically random `SESSION_SECRET` when it is missing. An explicitly configured secret must be at least 32 bytes. The configuration file is secured to mode `0600` and updates are written atomically.

### 3. Run

```bash
./dailyflow
```

The default address is `http://localhost:5330`.

## ⚙️ CLI Usage

| Flag | Description | Default |
| :--- | :--- | :--- |
| `-c` | Path to the configuration file | `./config.conf` |
| `-setdir` | Set the workspace directory | - |
| `-setuser` | Set the authentication username | - |
| `-setpass` | Interactively set and confirm a password | `false` |
| `-setbind` | Set the listen address, such as `0.0.0.0` | - |
| `-h` | Show command help | - |

If no credentials are configured, DailyFlow starts with clearly marked unsafe defaults and prints a warning. Replace them before exposing the service to a network.

## 🗂️ Journal Layout

DailyFlow accepts Markdown files anywhere below the configured workspace. Date-based paths make month navigation and sorting predictable:

```text
Daily/
└── 2026/
    ├── 06/
    │   └── 2026-06-30.md
    └── 07/
        └── 2026-07-17.md
```

## 🏗️ Architecture

- `main.go`: CLI, embedded web assets, routes, and server startup.
- `internal/api`: JSON and entry-content HTTP handlers.
- `internal/auth`: Password hashing, signed sessions, cookies, and authentication middleware.
- `internal/config`: Configuration parsing, atomic updates, and permission hardening.
- `internal/scanner`: Rooted Markdown listing, search, snippets, and month filtering.
- `web`: Vanilla JavaScript, CSS, shared theme logic, favicon, and vendored Marked.

## ✅ Verification

```bash
go test ./...
go test -race ./...
go vet ./...
go mod verify
node --test web/app-core.test.js
node --check web/app.js
node --check web/app-core.js
node --check web/theme.js
```

Node.js is only used for frontend tests and syntax checks; it is not required to run DailyFlow.

## 🛡️ Security Notes

- Keep `config.conf` private. It contains the password hash and session secret.
- Enable `COOKIE_SECURE=true` when users access DailyFlow through HTTPS.
- Session signatures use constant-time HMAC verification.
- Workspace reads are rooted and reject traversal, non-Markdown entry paths, and symlink escapes.
- Markdown raw HTML is rendered as text, and unsafe link schemes do not receive clickable `href` attributes.

## 📄 License

DailyFlow is licensed under the MPL-2.0 License. See [LICENSE](LICENSE).
