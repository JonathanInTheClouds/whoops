# whoops

Undo your last git action without googling the right incantation.

```bash
whoops
# Undo commit "wip: payment handler"? [y/N] y
# ✓ Undid: commit "wip: payment handler"
#   Ran: git reset --soft HEAD~1
```

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/JonathanInTheClouds/whoops/main/install.sh | bash
```

Or build from source:

```bash
git clone https://github.com/JonathanInTheClouds/whoops
cd whoops
make install
```

## Usage

**Undo the last thing you did:**

```bash
whoops
```

**Pick from recent git actions:**

```bash
whoops --history
```

**Preview what would be undone without doing it:**

```bash
whoops --dry-run
whoops --history --dry-run
```

**Check the version:**

```bash
whoops --version
```

## What it can undo

| Action          | What whoops does                                  |
| --------------- | ------------------------------------------------- |
| `git commit`    | `reset --soft HEAD~1` — changes go back to staged |
| `git stash`     | `stash pop` — brings your changes back            |
| `git stash pop` | `git stash` — re-stashes the changes              |
| `git merge`     | `reset --hard ORIG_HEAD`                          |
| `git rebase`    | `reset --hard ORIG_HEAD`                          |
| `git checkout`  | restores the previous state                       |

## History mode keybindings

| Key         | Action                |
| ----------- | --------------------- |
| `↑` / `k`   | Move up               |
| `↓` / `j`   | Move down             |
| `enter`     | Select action to undo |
| `y`         | Confirm undo          |
| `n` / `esc` | Cancel                |
| `q`         | Quit                  |

## Development

```bash
make build    # build binary
make test     # run tests
make release  # build all platform binaries into dist/
```

## Requirements

- git
- Go 1.21+ (to build from source)

## License

MIT
