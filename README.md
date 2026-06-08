<div align="center">

# 🔥 tokenburning

**One dashboard for everything your AI coding tools cost you.**

Cost, tokens, activity and session analytics across Claude Code, Codex and Cursor — in a single static binary that installs in seconds and sends nothing to the network by default.

[![CI](https://github.com/rshatskiy/tokenburning/actions/workflows/ci.yml/badge.svg)](https://github.com/rshatskiy/tokenburning/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/rshatskiy/tokenburning?sort=semver)](https://github.com/rshatskiy/tokenburning/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-fb923c.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8.svg)](go.mod)

[Русская версия →](README.ru.md) · [tokenburning.online](https://tokenburning.online) · [tokenburning.ru](https://tokenburning.ru)

</div>

---

## Why

Your team runs Claude Code, Codex and Cursor every day, and nobody knows what it costs — the numbers live in three different local log formats on every laptop. `tokenburning` reads those logs **locally**, prices them, and shows you exactly where the money and the tokens go. Optionally, and only with explicit consent, it rolls anonymized aggregates up to a team dashboard.

- **Local-first.** The collector runs on your machine. The dashboard binds to `127.0.0.1` with a bearer token. Zero network egress by default.
- **Honest.** Unpriced models are flagged `~est`; session analytics are labeled "signal, not exact". No guessing dressed up as fact.
- **Private by construction.** What leaves your machine (if you opt in) is *derived aggregates only* — no source, no prompts, no project paths, no precise timestamps. The team server literally cannot receive your content.

## Install

**macOS / Linux:**
```sh
curl -fsSL https://raw.githubusercontent.com/rshatskiy/tokenburning/main/install.sh | sh
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/rshatskiy/tokenburning/main/install.ps1 | iex
```

Or grab a binary for your OS/arch from [Releases](https://github.com/rshatskiy/tokenburning/releases) (macOS, Linux, Windows × amd64/arm64).

## Use

```sh
tokenburning scan        # parse local logs and print cost by tool/model
tokenburning dashboard   # open the local web dashboard (127.0.0.1, token-gated)
tokenburning version
```

The dashboard shows total cost and tokens, a cost-over-time chart, breakdowns by tool/model/project, and per-tool session analytics (active duration, tokens, iterations) — dark theme, no telemetry.

## Background collection (optional)

By default `tokenburning` does nothing in the background. To enable periodic collection with login autostart:

```sh
tokenburning enable                 # local background collection, 15-min interval
tokenburning enable --interval-min 30
tokenburning disable                # turn it off
```

- **macOS:** LaunchAgent · **Linux:** systemd user unit · **Windows:** Scheduled Task — all without root.

## Teams — register and roll up (optional)

Want a team view? Sign up, create an org, invite developers, and each one gets a one-line install command with a personal token:

1. Go to **[tokenburning.online](https://tokenburning.online)** (or **[tokenburning.ru](https://tokenburning.ru)**) and sign in with an email code.
2. Create your organization, then share the invite link with your developers.
3. On the **Install** page, copy your personal command:
   ```sh
   tokenburning enable --to https://tokenburning.online --token <YOUR-TOKEN> --breadth
   ```

The collector then pushes **derived aggregates only**, on a schedule, over the consent categories you chose. Preview exactly what would be sent at any time:

```sh
tokenburning push --breadth --depth --dry-run
```

## Privacy model

```
┌─────────────────────────────┐         consent-gated,          ┌──────────────────────────┐
│  Your machine (collector)   │      derived aggregates only     │   Team server (optional) │
│                             │  ──────────────────────────────▶ │                          │
│  local logs → SQLite        │   no source · no prompts         │  org dashboard           │
│  127.0.0.1 dashboard        │   no project paths               │  cohort medians (≥5)     │
│  full detail stays here     │   no precise timestamps          │  per-person budget facts │
└─────────────────────────────┘                                  └──────────────────────────┘
```

- **Content never leaves your machine.** The push payload is aggregate numbers (cost, tokens, activity, day-granularity trend, session medians) — verifiable with `--dry-run`.
- **Cohort suppression.** Team distribution stats (medians/quartiles) are shown only when **5+ members** have reported. Below that, they are hidden.
- **Symmetry.** A developer's self-view shows exactly the aggregate that was sent up — no hidden upload.
- **Configurable.** Each org chooses how much of the team aggregate rank-and-file developers can see (`full` / `cohort_only` / `manager_only`).
- **Unsigned binaries (for now).** macOS Gatekeeper: `xattr -d com.apple.quarantine /path/to/tokenburning` (curl installs don't quarantine). Windows SmartScreen: "More info" → "Run anyway".

## Architecture

A single Go module, two binaries:

- **`tokenburning`** — the collector/CLI. Adapters for Claude Code (append-JSONL), Codex (hybrid), and Cursor (SQLite). Pure-Go SQLite (no CGO) → trivial cross-compilation.
- **`server`** — the team platform (Go + PostgreSQL, server-rendered, passwordless email login, geo-routed `.ru`/`.online`, RU+EN). See **[deploy/README.md](deploy/README.md)** to self-host with Docker.

## Build from source

```sh
go build ./cmd/tokenburning   # collector
go build ./cmd/server         # team server
go test ./...
```

## License

[MIT](LICENSE) © Roman Shatskiy
