# aistat

A fast, local CLI for monitoring active Claude Code and Codex sessions. It aggregates
status from Claude hooks/statusline and Codex rollout/notify logs, then renders a
clean live view (TUI) or a script-friendly table/JSON.

- macOS only (by design)
- Works out of the box; optional install command wires hooks/notify
- Redaction on by default (safer when sharing screens)

## Install

### Homebrew (recommended)

```sh
brew tap vburojevic/tap
brew install aistat
```

Upgrade:

```sh
brew update
brew upgrade aistat
```

### Go (from source)

```sh
go install github.com/vburojevic/aistat/cmd/aistat@latest
```

## Quick setup

### 1) Run it once

```sh
aistat
```

Tip: press `:` in the TUI for the command palette, `d` to toggle detail. Press `p` to open the project picker.
Press `tab` to open the projects dashboard.

### 2) Wire integrations (recommended)

This configures:
- Claude Code hooks + statusLine
- Codex notify integration

```sh
aistat install
```

Non-interactive install:

```sh
aistat install --skip-codex --force
```

### 3) Verify setup

```sh
aistat doctor
```

If you need to repair config/hook wiring:

```sh
aistat doctor --fix
```

## Usage guide

### Everyday usage

- Live TUI (default on a TTY):
  ```sh
  aistat
  ```
- Non-interactive table:
  ```sh
  aistat --no-tui
  ```
- Watch mode (table refresh):
  ```sh
  aistat --no-tui --watch
  ```
- Watch NDJSON stream (timestamps included):
  ```sh
  aistat --watch --json
  ```

### TUI quick guide

- `/` filter, `esc` clear
- `:` command palette
- `p` project picker (toggle projects)
- `tab` projects dashboard (active projects overview)
- `d` toggle detail pane (split view on wide screens)
- `b` toggle sidebar filters
- `s` sort, `g` group, `v` view
- `m` toggle last message snippets
- `P` pin, `space` select, `y` copy IDs
- `o` open log, `D` copy detail
- `1/2` provider filters, `R/W/E/S/Z/N` status filters

### CLI commands

```
aistat [flags]
aistat projects [flags]
aistat show <id> [flags]
aistat summary [flags]
aistat install [flags]
aistat config [--show|--init]
aistat doctor [--fix]
aistat tail <id> [flags]
```

### Common flags

- `--json` Output JSON instead of table/TUI
- `--watch` Continuously refresh output (non-TUI)
- `--no-tui` Force non-interactive output even on a TTY
- `--provider claude|codex` Filter by provider
- `--project <name>` Filter by project name (repeatable or comma-separated)
- `--status <status>` Filter by status (repeatable or comma-separated)
- `--fields <list>` Select output columns (comma-separated or repeatable)
- `--sort last_seen|status|provider|cost|project` Sort output
- `--group-by provider|project|status|day|hour` Group output (non-TUI only)
- `--include-last-msg` Include last user/assistant snippets when available
- `--all` Include ended/stale sessions (wider scan window)
- `--redact` Redact paths/IDs (default from config)
- `--active-window 30m` Define how long a session is considered active
- `--running-window 3s` Define how recent activity must be to show running
- `--refresh 1s` Refresh interval for watch/TUI
- `--max 50` Maximum sessions to show
- `--no-color` Disable color output (TUI + table)

### Examples

Agent-friendly help (structured JSON):

```sh
aistat help --format json
```

List all projects:

```sh
aistat projects
```

Include ended/stale projects:

```sh
aistat projects --all
```

Live TUI:

```sh
aistat
```

Search in the TUI:

```sh
# /  then type:
#   p:myproject
#   s:running
```

Table snapshot for scripts:

```sh
aistat --json
```

Watch NDJSON:

```sh
aistat --watch --json
```

Only Codex sessions:

```sh
aistat --provider codex
```

Filter multiple projects:

```sh
aistat --project alpha --project beta
```

Filter by status:

```sh
aistat --status approval
```

Custom fields:

```sh
aistat --fields provider,id,status,project
```

Grouped by day (non-TUI):

```sh
aistat --group-by day
```

Include last message snippets:

```sh
aistat --include-last-msg
```

Show a single session:

```sh
aistat show <id>
```

Summarize by project:

```sh
aistat summary --group-by project
```

Tail a session log:

```sh
aistat tail <id>
```

Auto-fix setup (same behavior as install):

```sh
aistat doctor --fix --force
```

`doctor --fix` prompts in a TTY and snapshots configs; if an error occurs it restores the backups.

## AI/agent integration

The `help` command outputs a machine-readable contract so agents can reason about
commands, flags, I/O, exit codes, and config:

```sh
aistat help --format json
```

## Configuration

`aistat` reads a JSON config file from:

```
~/Library/Application Support/aistat/config.json
```

Create a default config:

```sh
aistat config --init
```

Show current config:

```sh
aistat config --show
```

Example config:

```json
{
  "redact": true,
  "active_window": "30m",
  "running_window": "3s",
  "refresh_every": "1s",
  "max_sessions": 50,
  "all_scan_window": "168h",
  "statusline_min_write": "800ms"
}
```

## How it works

- Claude Code:
  - Hooks update session records in real time.
  - Statusline updates cost/model/context metrics.
  - Fallback scan reads recent transcript files.
- Codex:
  - Notify integration updates session records.
  - Rollout logs provide recent activity and metadata.

All records are stored locally under:

```
~/Library/Application Support/aistat/sessions
```

## Environment variables

- `AISTAT_HOME` Override the app data directory
- `CODEX_HOME` Override Codex home (where sessions live)
- `ACCESSIBLE` Use accessible UI mode in the install wizard

## Troubleshooting

- `aistat doctor` shows where aistat is reading from and what is configured.
- If nothing shows up, run `aistat install` and ensure Claude/Codex are writing
  events.
- For shared screens or logs, keep `redact` enabled (default).
- `aistat clean` removes spool data and invalid session records (use `--dry-run` to preview).

## Release process (maintainers)

Tag a version to publish a GitHub release and update the Homebrew formula:

```sh
git tag vX.Y.Z
git push --tags
```

The GitHub Actions release workflow will build and publish binaries and update
`vburojevic/homebrew-tap`.
