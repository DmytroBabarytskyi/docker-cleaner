# 🧹 Docker Cleaner

A blazingly fast, interactive **Terminal User Interface (TUI)** for managing and cleaning up your Docker environment.  
Built with Go and the awesome Bubble Tea framework.

---

## ✨ Features

- **Interactive TUI**  
  Say goodbye to long, unreadable terminal outputs. Navigate your Docker resources with ease.

- **Multi-Select & Bulk Delete**  
  Select multiple containers, images, volumes, or networks and delete them all at once.

- **Smart Prune**  
  Quickly select all exited containers or dangling images with a single keystroke (`p`).

- **Deep Dive**  
  Inspect container configurations (`i`) or view real-time logs (`L`) right inside the app.

- **Docker Compose Aware**  
  Filters and displays Docker Compose project names automatically.

- **Fuzzy Search**  
  Filter resources instantly by name, ID, or Compose project.

---

## 🚀 Installation

Ensure you have Go installed. Then run:

```bash
go install github.com/DmytroBabarytskyi/docker-cleaner@latest
```

---

### Alternatively, clone the repository and build it manually:

```bash
git clone https://github.com/DmytroBabarytskyi/docker-cleaner.git
cd docker-cleaner
go build -o docker-cleaner .
./docker-cleaner
```

---

## ⌨️ Keybindings

### Global Navigation

| Key | Action |
|-----|--------|
| `Tab` / `→` / `←` | Switch between tabs (Containers, Images, Volumes, Networks) |
| `↑` / `↓` / `k` / `j` | Navigate the list |
| `/` | Focus search bar to filter items |
| `q` / `Ctrl+C` | Quit application |

---

### Selection & Actions

| Key | Action |
|-----|--------|
| `Space` | Toggle selection for the highlighted item |
| `a` | Select / Deselect ALL items in the current tab |
| `p` | Select all "prunable" items (exited containers, dangling images) |
| `Enter` | Delete all currently selected items across ALL tabs |

---

### Container Specific (Containers Tab)

| Key | Action |
|-----|--------|
| `s` | Stop the highlighted container |
| `r` | Restart the highlighted container |
| `L` | View logs of the highlighted container (scrollable) |
| `i` | Inspect (view JSON configuration) of the highlighted container |

---

## 🛠️ Built With

- **Bubble Tea** — The fun, functional, and stateful way to build terminal apps.
- **Lip Gloss** — Style definitions for nice terminal layouts.
- **Docker Engine API** — For interacting with the Docker daemon.

---

## 📄 License

This project is licensed under the MIT License - see the `LICENSE` file for details.