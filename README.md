# 🌊 DailyFlow

**A seamless, high-performance web interface for managing and viewing your daily markdown journals.**

---

[![Go Version](https://img.shields.io/badge/Go-1.26.2+-00ADD8?style=for-the-badge&logo=go)](https://golang.org)
[![Build Status](https://img.shields.io/badge/Build-Success-4CAF50?style=for-the-badge&logo=github-actions)](https://github.com/Xarth-Mai/DailyFlow/actions)
[![License](https://img.shields.io/badge/License-MPL--2.0-brightgreen.svg?style=for-the-badge)](LICENSE)

DailyFlow is a lightweight, modular server that transforms your local directory of Markdown files into a beautiful, searchable web experience. Perfect for developers, writers, and anyone who uses Markdown for daily logs.

## 🌟 Why DailyFlow?

Managing daily journals in flat folders can be tedious. DailyFlow bridges the gap between simple text files and a distraction-free reading environment. It combines the simplicity of Markdown with a modern, high-performance web interface—no database required.

## ✨ Key Features

- 📁 **Effortless Scanning**: Automatically index and sort all `.md` files in your workspace.
- 🔍 **Full-Text Search**: Blazing fast search across all your journal entries.
- 🔐 **Secure Access**: Modern authentication with Bcrypt password hashing and token-based sessions.
- 📄 **Dynamic Navigation**: Responsive sidebar for quick access to your history.
- 📦 **Zero-Dependency Core**: Single binary deployment with all static assets embedded.
- 🚀 **Performance First**: Built in Go for rapid indexing and minimal resource footprint.
- 🎨 **Premium UI**: Clean, responsive frontend designed for focused reading.
- 💻 **Cross-Platform**: Native support for Windows and Linux (AMD64 & ARM64).

## 🚀 Quick Start

### 1. Download
Grab the latest binary for your architecture from the [GitHub Actions artifacts](https://github.com/Xarth-Mai/DailyFlow/actions).

### 2. Configure
Initialize your workspace directory:
```bash
./dailyflow -setdir "path/to/Daily"
```

Set your secure credentials:
```bash
./dailyflow -setuser "MyUsername"
./dailyflow -setpass
```

### 3. Run
Start the server:
```bash
./dailyflow
```
The server will start at `http://localhost:5330` by default. Visit the page and log in to start viewing your journals.

## ⚙️ CLI Usage

DailyFlow comes with a robust CLI for easy configuration without manually editing files.

| Flag | Description | Default |
| :--- | :--- | :--- |
| `-c` | Path to the configuration file | `./config.conf` |
| `-setdir` | Set the workspace directory path | - |
| `-setuser` | Set the authentication username | `DailyFlowUser` |
| `-setpass` | Interactively set a new password | `DailyFlowUnsafePasswd` |
| `-setbind` | Set the listen address (e.g., `0.0.0.0`) | `localhost` |
| `-h` | Show help message | - |

## 🏗️ Technical Architecture

DailyFlow is built with a modular approach for better maintainability:

- `internal/api`: RESTful API handling search, listing, and asset serving.
- `internal/auth`: Secure session management and Bcrypt authentication.
- `internal/scanner`: High-performance filesystem crawler and indexer.
- `internal/config`: Adaptive configuration management.
- `web`: Modern frontend with vanilla JS and CSS.

## 🛠️ Build from Source

Ensure you have [Go](https://golang.org/dl/) installed.

```bash
git clone https://github.com/Xarth-Mai/DailyFlow.git
cd DailyFlow
go build -o dailyflow .
```

## 🛡️ Security
- **Authentication**: DailyFlow requires a username and password to prevent unauthorized access.
- **Passwords**: We use **Bcrypt** for industry-standard password hashing, ensuring your credentials are never stored in plain text.
- **Sessions**: Token-based session management ensures secure interaction between the frontend and backend.
- **Warning**: Never share your `config.conf` file, as it contains your private configuration and hashed password.

## 📄 License

This project is licensed under the MPL-2.0 License. See the [LICENSE](LICENSE) file for details.

---

<p align="center">
  Made with ❤️ for the Markdown community.
</p>
