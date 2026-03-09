# gstash

A terminal UI for managing git stashes — browse, preview, apply, drop, and rename stashes without memorizing `stash@{n}` syntax.

## Features

- 📋 **List & navigate** stashes with branch and relative timestamp
- 👁️  **Diff preview** pane with syntax-colored output
- ✅ **Apply / Pop / Drop** with a single keypress
- ✏️  **Rename** stashes so "WIP" actually means something

## Keybindings

| Key         | Action                        |
|-------------|-------------------------------|
| `↑` / `k`   | Move up                       |
| `↓` / `j`   | Move down                     |
| `a`         | Apply stash (keep it)         |
| `p`         | Pop stash (apply and remove)  |
| `d`         | Drop stash (confirm required) |
| `r`         | Rename stash                  |
| `PgUp/PgDn` | Scroll diff preview           |
| `q`         | Quit                          |

## Installation

```bash
git clone https://github.com/user/gstash
cd gstash
go build -o gstash ./cmd/gstash
mv gstash /usr/local/bin/
```

## Usage

Run `gstash` from inside any git repository.

```bash
cd my-project
gstash
```

## Requirements

- Go 1.21+
- git
