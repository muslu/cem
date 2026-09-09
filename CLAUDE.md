# CLAUDE.md — cem

> This file is project-specific and **overrides the global
> `~/.claude/CLAUDE.md`**. Where the two conflict, this file wins.
> Turkish version: [CLAUDE.tr.md](CLAUDE.tr.md).

---

## Project Summary

**CEM — Compose · Execute · Multiplex.** A Go orchestrator that drives
multiple AI CLIs (Claude, Antigravity, Aider, Gemini, Codex, Goose, Cody,
Continue, OpenHands, Cursor) from a single command. A "thinker" AI reasons,
a "writer" AI emits code; in `pair` mode the thinker's output is fed into
the writer.

- **Domain:** `cem.pw`
- **Git:** `https://github.com/muslu/cem.git` (canonical)
- **Language:** Go 1.25 (cobra + lipgloss + yaml.v3)
- **Binaries:** three names built from one source — `cem`, `cemi`
  (installer), `cemir` (remover). `main.go` dispatches based on the
  invoked binary name.

---

## Differences from the Global Guide

Global `~/.claude/CLAUDE.md` is centred on Python + PostgreSQL + FastAPI +
nginx. This project is a **Go CLI**, so the following global rules **do
not apply**:

| Global rule | Why it doesn't apply |
|---|---|
| Pydantic / `.env` / Fernet | No secrets at runtime; config is YAML |
| FastAPI middleware / Server-Timing | Not an API |
| PostgreSQL / MongoDB / Valkey | No data layer |
| Cache decorator / rate limit | Not relevant |
| Swagger UI / `src/swagger.py` | Not an API |

**Still applies:** Docker ban (bare-metal binaries), `nala` over `apt`.
The `cem.pw` web host runs **Apache2** on an existing shared box; the
nginx production standards from the global guide do not apply here.

---

## File Layout

```
cem/
├── main.go             — Binary-name dispatch + LDFLAGS version
├── config.go           — GlobalConfig + ProjectConfig + ResolvedConfig + KnownTools
├── config_test.go      — ActiveRoles override + KnownTools sanity
├── executor.go         — ModeThink/Write/Pair + Run + ReadStdin + effort/model resolve
├── executor_test.go    — stderr signature triage, effort args, trailing-echo dedupe
├── spinner.go          — TTY-aware single-line spinner (pair mode); Stop() is idempotent
├── spinner_test.go     — Stop() idempotency + concurrency (regression: closed-channel panic)
├── tool_update.go      — AI CLI updates: native `<tool> update` + daily background auto-update
├── detach_unix.go / detach_windows.go — platform split for detached background updates
├── history.go          — AppendHistory → ~/.cem/history.log (TSV)
├── wizard.go           — RunSetupWizard, InstallTool, RemoveTool, ShowRoles, askYN
├── banner.go           — ASCII art + lipgloss styles + ShowConfigSource
├── cmd_cem.go          — Cobra root + rolesCmd, setupCmd, initCmd, statusCmd
├── cmd_doctor.go       — `cem doctor`: diagnostic report
├── cmd_effort.go       — `cem effort`: show/change reasoning effort (global + --here)
├── cmd_history.go      — `cem history`: -n N, --clear
├── cmd_cemi.go         — `cemi`: install tools (4 known: claude, agy, gpt, cursor) + all + update
├── cmd_update.go       — `cem update`: cem.pw/r/'den son sürümü indir + yerine yaz
├── cmd_cemir.go        — `cemir`: remove tools (single + all)
├── cmd_uninstall.go    — `cem uninstall`: remove the binaries themselves
├── install.sh / .ps1   — User-facing installer via cem.pw/install
├── uninstall.sh / .ps1 — User-facing uninstaller via cem.pw/uninstall
├── Makefile            — build / dev / install / clean / tidy / test
├── go.mod / go.sum
├── .github/workflows/release.yml  — 7-platform binaries + SHA256SUMS
├── README.md / README.tr.md
├── CLAUDE.md / CLAUDE.tr.md
├── OPERATIONS.md / OPERATIONS.tr.md
└── todo.md / todo.tr.md
```

---

## Coding Conventions

1. **`package main`** — single package; do not split into subpackages
   unless required.
2. **No logger** — write to the user with `fmt.Println` + lipgloss styles
   (`styleSuccess`, `styleError`, `styleDim`, `styleBold`, declared in
   `wizard.go`).
3. **User-visible error messages are in Turkish** —
   `styleError.Render("✗ ...")`. Do not show stack traces.
   `fmt.Errorf` strings may be English for debugging.
4. **Subprocess for AI tools** — use `exec.Command`. Stdin: input,
   stdout/stderr: passthrough. In pair mode use `cmd.Output()` to capture.
5. **Config IO** — `~/.cem/config.yaml` file mode `0600`, directory
   `0755`.
6. **YAML marshalling** — `gopkg.in/yaml.v3`. Lowercase snake_case tags.
7. **Adding an AI tool** — append a `ToolMeta` to `KnownTools` and a key
   to `orderedToolKeys`. Do not touch the installer/remover/wizard; they
   read the map automatically. Fill `UpdateCmd` (the tool's own `update`
   subcommand) so auto-update covers it, and `EffortArgs`/`Efforts` if the
   CLI exposes a reasoning-effort knob — `EffortArgs` is a `fmt.Sprintf`
   template list (`{"--effort", "%s"}` vs `{"-c", "model_reasoning_effort=%s"}`).
8. **Adding a Cobra subcommand** — create `cmd_<name>.go` with its own
   `init()` that calls `rootCmd.AddCommand(...)`. Avoid touching
   `cmd_cem.go`'s init block.
9. **Banner** appears only on `help` / `setup` / `status` / `cemi` (no
   args) / `cemir` (no args) — never on a normal command run.
10. **Build** — always produce all three binaries. `make build` for local
    dev; `make install` (sudo) for system-wide.

---

## Runtime Gotchas

- **Never scan a subprocess's raw output for error signatures without
  `sanitizeStderr`.** AI CLIs dump HTTP response bodies into stderr as single
  100 KB lines; words like `unauthorized`, `429`, `quota`,
  `authentication failed` appear there as *data*. Matching them raw makes cem
  misreport a model error as an auth failure, or burn every API key on a
  phantom rate limit.
- **Diagnosis order in `runTool`/`captureToolWithSpinner`:** rate limit →
  model unsupported → auth failure → generic. Model errors must be checked
  before auth: several CLIs mention both in one message.
- **`Spinner.Stop()` is called from two places** — `stopWriter.Write` (on the
  subprocess's first stderr byte, a *different goroutine*) and `Run` in
  `ModePair`. It is `sync.Once`-guarded; keep it that way.
- **Effort and model flags must land before `-p`** for tools with
  `ModelBeforeRun: true` (claude, cursor): `-p` takes the prompt as its
  argument, so anything inserted between them swallows the prompt.
- **Auto-update runs detached** (`detachProcess`) and cannot write back to the
  config; version fields are refreshed on the *next* run in
  `maybeAutoUpdateTools`. Disable with `auto_update_tools: false`.

## Bans (for this project)

- **No Docker / docker-compose** — bare-metal binaries only.
- **No Python bridge** — pure Go. AI CLIs are invoked as subprocesses.
- **No separate version file** — version is injected at build time from
  `git describe --tags --always --dirty` via `LDFLAGS -X main.version`.
- **No `go run main.go`** — `main.go` depends on the rest of the package;
  use `go run .`.
- **Do not rename binaries** — `cem` / `cemi` / `cemir` are fixed.
  `main.go`'s dispatch and the install scripts depend on those names.
- **User-facing strings in Turkish; code-internal strings may be
  English.** Match the surrounding context.

---

## Git & Release

- **Canonical repo:** `https://github.com/muslu/cem.git` (only remote)
- **Binary downloads:**
  `install.sh` / `install.ps1` pull directly from
  `github.com/muslu/cem/releases/latest/download/...`. The `cem.pw/r/*`
  Apache rewrite proxies the same URL for older scripts.
- **Tags:** `vMAJOR.MINOR.PATCH` (semver). `LDFLAGS -X main.version=...`
  is set by both Makefile and CI.

---

## Validation Flow

```sh
make clean && make build
./build/cem --help      # banner + command list
./build/cem doctor      # diagnostic report
./build/cemi            # tool list (banner included)
./build/cemir           # installed-tools list (banner included)
./build/cem roles       # active roles + config source
go test -race ./...     # 28 tests, all should pass
```

The first run launches a wizard and creates `~/.cem/config.yaml`. Inside a
test directory, a `.cem.yaml` file overrides the global config.

---

## Known Gaps

`todo.md` is the live list. Open items:
- `.claude/agents/` and `.claude/skills/` are leftovers from the
  `autoinstalltrixie` project; the cleanup decision is still pending.
  `.claude/` is gitignored so they don't enter the repo.
- Broader test coverage: `history_test.go` still missing (`executor_test.go` and
  `spinner_test.go` now exist).
- No macOS/Linux/Windows integration tests.

---

*If this CLAUDE.md changes, add a `doc:CLAUDE update` line to `todo.md`.*
