# CEM — Advanced Guide

> Start with [README.md](README.md) if you've never used cem before. This document is for users who want every knob.

Turkish version: [ADVANCED.tr.md](ADVANCED.tr.md)

---

## Architecture in 30 seconds

cem is **not** an LLM client; it is a process orchestrator. Every "thinker" / "writer" call is `exec.Command(<tool>, --flags..., <prompt>)`. cem decides:

- Which binary to call (resolved against PATH + a per-tool fallback list)
- Which model flag to append (per tool's `ModelFlag`)
- Which environment variable to inject the API key into (per tool's `Provider` + `APIKeyEnv`)
- Whether to retry with the next key on rate-limit signals

If you understand `exec.Command` + flag ordering, you understand cem.

---

## Tool registry

```go
type ToolMeta struct {
    Name             string   // display name
    Binary           string   // PATH name if != toolKey (e.g., cursor → cursor-agent)
    Description      string
    Deprecated       string   // shows a warning line in `cemi`

    InstallCmd       []string // exec.Command form, e.g. ["npm", "install", "-g", "@openai/codex"]
    InstallShellUnix string   // shell command for sh -c (Linux/macOS)
    InstallShellWin  string   // shell command for powershell (Windows)

    VersionFlag      string   // e.g. "--version"
    RunFlags         []string // appended on every call, e.g. ["-p"] or ["exec", "--skip-git-repo-check"]
    PromptAsArg      bool     // if true, input is the LAST positional arg; else piped via stdin
    ModelFlag        string   // e.g. "--model" — empty if the CLI doesn't support model selection
    ModelBeforeRun   bool     // if true, --model is placed BEFORE RunFlags (needed when a -p flag consumes the next arg)
    Models           []string // suggestions for the wizard model picker

    Provider         string   // "anthropic", "openai" — empty disables API key rotation
    APIKeyEnv        string   // "ANTHROPIC_API_KEY", "OPENAI_API_KEY"

    AuthCmd          []string // subcommand for `cem auth <tool>` (e.g. ["login"])
}
```

Current 4-tool lineup:

| Key | Binary | Install source | Model flag | API key rotation |
|---|---|---|---|---|
| `claude` | `claude` | [claude.ai/install.sh](https://code.claude.com/docs/en/quickstart) | `--model opus\|sonnet\|haiku` | anthropic (ANTHROPIC_API_KEY) |
| `agy` | `agy` | [antigravity.google/cli/install.sh](https://antigravity.google/docs/cli-getting-started) | — (CLI has no model flag) | — (OAuth only) |
| `gpt` | `codex` | `npm i -g @openai/codex` | `--model gpt-5.5\|...` (after `exec`) | openai (OPENAI_API_KEY) |
| `cursor` | `cursor-agent` | [cursor.com/install](https://cursor.com/cli) | `--model claude-4.6\|gpt-5.2\|...` | — (OAuth only) |

Notes:
- **Antigravity has no `--model` flag** — `agy --help` lists `-p`, `-c`, `--sandbox`, `--print-timeout`, but no model selector. cem's `ModelFlag` is intentionally empty for agy.
- **Cursor binary is `cursor-agent`** (legacy symlink); installer also drops `agent.*` aliases.
- **Codex binary is `codex`** (npm package is `@openai/codex` but PATH name is `codex`).

---

## Resolution order (when cem invokes a tool)

```
1. cfg.Tools[toolKey].Command  (absolute path stored after install) — used if file still exists
2. exec.LookPath(meta.Binary or toolKey)
3. fallbackInstallPath(toolKey) — hand-maintained list of known install locations:
     agy    → %LOCALAPPDATA%\agy\bin\agy.exe         ~/.local/bin/agy
     claude → %LOCALAPPDATA%\Claude\claude.exe       ~/.claude/local/claude  ~/.local/bin/claude
     cursor → %LOCALAPPDATA%\cursor-agent\{cursor-agent,agent}.{cmd,ps1,exe}
              %APPDATA%\npm\cursor-agent.{cmd,ps1}   ~/.local/bin/cursor-agent
4. Give up → "✗ <tool> not found"
```

If a native installer puts the binary somewhere unexpected, add a candidate to `fallbackInstallPath` in `executor.go`. PRs welcome.

---

## Command-line argument order

cem builds args as follows:

```
ModelBeforeRun = true   →  [--model X, RunFlags..., promptArg?]
ModelBeforeRun = false  →  [RunFlags..., --model X, promptArg?]
```

Why this matters: `agy -p` and `cursor-agent -p` consume the *next* arg as the prompt. If `--model X` lands between `-p` and the prompt, the tool reads "--model" as the prompt. Set `ModelBeforeRun = true` for any tool whose run flag eats its argument.

Codex uses `exec` as a subcommand, so `--model` must follow it: `codex exec --model gpt-5.5 --skip-git-repo-check "prompt"`. Hence `ModelBeforeRun = false` (default) plus `RunFlags: ["exec", "--skip-git-repo-check"]`.

---

## Pair mode skip rules

`cem -p "..."` runs thinker → writer. cem skips the writer call when it would be wasteful:

| Condition | Behaviour |
|---|---|
| `thinker == writer` (both `claude` etc.) | Writer skipped — identical model would just duplicate |
| Prompt has no code intent **and** thinker output has no ` ``` ` block | Writer skipped — nothing to write |
| Otherwise | Writer runs with thinker's analysis prepended to original prompt |

Code-intent detection: regex on `yaz|kod|script|fonksiyon|class|method|implement|kodla|oluştur|üret|döndür|export|function|code|write|build|generate|refactor|debug|fix` (case-insensitive, word boundary).

---

## HTTP model servers (ollama · LM Studio · unsloth)

Three tools in the registry are not subprocesses. `ToolMeta.HTTPAPI` decides:

| Value | Request |
|---|---|
| `"openai"` | `POST {base}/v1/chat/completions` — LM Studio, unsloth/vLLM |
| `"ollama"` | `POST {base}/api/chat` |
| `""` | not an HTTP tool; cem execs a binary |

`isHTTPTool(key)` gates every path that assumes a binary:
`installableToolKeys()` keeps them out of `cemi` / `cemir` / auto-update, effort
and fast mode are skipped, and `cem doctor` probes reachability instead of PATH.

Address resolution: project `.cem.yaml` → global `endpoints.<tool>` →
`ToolMeta.DefaultBaseURL`. A missing scheme becomes `http://` — a LAN server
rarely speaks TLS — and `/v1` is not appended twice if the address already ends
in it. A key, when set, travels as `Authorization: Bearer`; ollama and LM Studio
need none locally, so no header is sent when it is empty.

The answer is streamed: OpenAI's SSE (`data: {...}`, `data: [DONE]`) and
ollama's newline-delimited JSON are both line-based, so one scanner handles
them. Lines that don't parse are skipped rather than raised — servers interleave
warm-up and statistics objects. An empty answer *is* an error: a server with no
model loaded returns a contentless stream, and reporting success there would
leave the user with nothing and no reason.

Timeouts are split: 30 minutes for the request (a local 70B model can spend
minutes on a long reply) but 5 seconds to connect, so a closed port fails
immediately instead of hanging for half an hour.

The model name is never guessed. `endpointModel` returns an error pointing at
`cem endpoint <tool> --model`, because a wrong name on a local server either
404s or silently runs a different model than the user intended.

---

## Setup without a terminal

`loadAndCheckSetup` refuses to run unconfigured — a half configuration means
sending the request to a tool the user never chose. With a TTY it offers the
wizard and re-checks the saved config afterwards (a wizard interrupted with
Ctrl+C used to leave the same half state). Without a TTY it prints `cem setup`
and exits, because the questions would scroll past with nobody to answer them.

That leaves GUIs and scripts, which is what the flags are for:

```sh
cem setup --thinker gpt --writer claude
cem setup --thinker ollama --endpoint-thinker 1.2.3.4:11434 \
          --model-thinker qwen3-coder --writer claude
cem status --json
```

`ApplySetup` validates before saving: unknown tool (with a fuzzy suggestion), an
HTTP tool with no model name, an effort level the tool does not have. A value
that is not passed is preserved, so `cem setup --lang en` does not clear the
roles.

`cem status --json` prints the config plus, per tool, whether it is installed,
its path, model, effort, endpoint and the known model/effort lists. Nothing else
may reach stdout: `machineReadableOutput()` scans `os.Args` for `--json` before
cobra parses and skips the update notice and the auto-update line, either of
which used to land on top of the JSON and break the caller's parser.

---

## API key rotation

Storage (`~/.cem/config.yaml`):

```yaml
api_keys:
  anthropic:
    - value: sk-ant-...
      label: personal
    - value: sk-ant-...
      label: company-backup
  openai:
    - value: sk-proj-...
```

Runtime: on each call cem sets `<provider>.APIKeyEnv = <key>` and execs the tool. If the tool exits non-zero **and** stderr matches:

```
rate.?limit | quota | 429 | too many requests | usage limit | overloaded
```

cem retries with the next key (preserves order, full rotation per call). If stderr matches an auth-failure pattern instead:

```
401 | unauthorized | missing bearer | invalid api key | not.?logged.?in | please run /login | please log in | authentication failed
```

cem does not rotate (different keys won't fix bad auth); it prints a helpful hint pointing at `cem keys list/remove/add` or the tool's interactive login.

---

## Interactive login and the OAuth code

For tools that take the prompt as an argument (agy, claude, cursor) stdin is
free, and it used to be left unset — that is, `/dev/null`. When such a tool
asked for an OAuth code ("Or, paste the authorization code here and press
Enter") it had nothing to read: it waited 60 seconds and died, while the code
the user pasted went to the shell instead. On Windows PowerShell read
`4/0ATs…` as a command and answered *"You must provide a value expression
following the '/' operator"* three times in a row.

`attachStdin` hands the terminal to the tool when cem's own stdin is a TTY. Not
when it is a pipe — that data was already consumed by `ReadStdin`, and the tool
would block waiting for EOF — and not in quiet mode, where the raw stream is
hidden and the user would be typing blind. When the interactive prompt appears
in the stream, one dim line says the code goes *here*, not into the shell.

### PSReadline paste helper

PowerShell's PSReadline silently truncates long pastes (Google OAuth codes routinely break). Workaround:

```powershell
cem auth agy --code 4/0AY29...long-google-oauth-code
```

`--code` writes the code to the system clipboard (`Set-Clipboard` on Windows, `pbcopy` on macOS, `wl-copy`/`xclip`/`xsel` on Linux). Then cem launches `<bin> login` interactively. At the CLI prompt, **right-click to paste** — bypasses PSReadline entirely.

Without `--code`, the command just runs `<bin> login` with TTY passthrough (no clipboard touch).

---

## Project-local config

`.cem.yaml` at any project root overrides global settings for that subtree:

```yaml
roles:
  thinker: claude
  writer:  agy
models:
  claude: opus
  agy: gemini-3-pro    # ignored (agy has no --model), but kept for documentation
```

Resolution order for model: project `models.<key>` → global `tools.<key>.model` → empty (CLI default).

`cem init` creates this file interactively. After every install cem appends `.cem.yaml` to `.gitignore` in the current git repo (or warns if no `.gitignore` exists) — your project-local config doesn't leak into the repo.

`endpoints` works the same way: the same project opened on two machines can
point at different model servers.

---

## Linux Node.js — why nvm?

NodeSource (the official Node apt/dnf source) shipped a "nodistro" pivot in 2024: all builds now require `libc6 >= 2.28`. Ubuntu 18.04 (bionic) has `libc6 2.27`. Result: `apt install nodejs` from NodeSource fails with "unmet dependencies" on bionic and similar older distros.

cem's `ensureDep("npm")` therefore uses **nvm** on Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh | bash
. ~/.nvm/nvm.sh
nvm install --lts
```

After install cem prepends `~/.nvm/versions/node/v*/bin` to its own PATH so the new npm is usable in the same session.

**Caveat:** nvm's prebuilt Node binaries from nodejs.org/dist also link against modern glibc. On Ubuntu 18.04 we've observed `GLIBC_2.28 not found` even for nvm Node 24. If you must run cem-installed npm-based tools (Codex CLI) on bionic, install Node yourself from source or pin to an older Node version that supports glibc 2.27. Easier: upgrade the OS.

---

## `cem doctor`

Runs a system + roles + tools + PATH diagnostic. Use it whenever something looks off:

```
✓ linux/amd64 · Go runtime: go1.25.10
✓ ~/.cem directory → /home/muslu/.cem
✓ global config → /home/muslu/.cem/config.yaml
✓ no project config (using global)

✓ thinker → claude
✓ writer → agy

✓ Claude   /home/muslu/.local/bin/claude
⚠ Antigravity on PATH but not in config → cemi agy
✗ Cursor   in config but not on PATH
```

---

## History log

`~/.cem/history.log` records every cem invocation as TSV:

```
2026-05-25T17:30:14Z    think   claude    "explain this codebase"
```

`cem history -n 50` shows the last 50; `cem history --clear` empties the log.

---

## Update check

cem hits `api.github.com/repos/muslu/cem/releases/latest` at most once per hour and caches the result at `~/.cem/update-check.json`:

```json
{"last_check":"2026-05-25T19:00:00Z","latest_version":"v0.1.33"}
```

If `latest_version != version`, the next cem/cemi/cemir invocation prints a colored notice:

```
🔔 new version: v0.1.32 → v0.1.33 · run cem update
```

`cem update` itself queries the API synchronously before downloading, so the latest tag is always shown first ("ⓘ latest: v0.1.33 (current: v0.1.32)").

---

## Build from source

```sh
git clone https://github.com/muslu/cem.git
cd cem
make build               # produces ./build/{cem,cemi,cemir}
make install             # sudo cp to /usr/local/bin
go test ./...
```

Build flags inject `git describe --tags --always --dirty` into `main.version` via `LDFLAGS -X main.version=...`. A `dev` build (no git tag context) suppresses the update-check notice — useful when iterating locally.

---

## Self-update internals

`cem update` is implemented in pure Go (`cmd_update.go`). Steps:

1. Query GitHub API for latest tag (sync, ~200ms).
2. Compare with `main.version`; abort with "already up to date" if equal.
3. Pre-flight `canWriteDir` on the install directory. If not writable on Unix → re-exec self via `sudo`. On Windows → tell user to launch an Admin PowerShell.
4. Download `cem.pw/r/<binary>` to `os.TempDir()`.
5. Replace the running binary:
   - **Unix**: `os.Rename(tmp, dst)` — atomic, the kernel allows replacing a running executable.
   - **Windows**: rename `dst → dst.old` first (allowed even for executing files), then `os.Rename(tmp, dst)`. The `.old` file is left in place — `cem uninstall` cleans it up.
6. Repeat for `cemi`, `cemir`.

---

## Self-uninstall internals

`cem uninstall`:

- Iterates `[cem, cemi, cemir]` and deletes each from PATH.
- On Windows the running `cem.exe` can't delete itself; cem schedules a detached `cmd /c "ping 127.0.0.1 -n 2 >nul & del /f /q <path>"` that runs after the parent exits.
- Optionally removes `~/.cem/` (config, history log).
- Optionally removes a project's `.cem.yaml`.

`cemir <tool>` deletes the AI tool's binary. For shell-installed tools (claude, agy, cursor) the parent directory is checked: if `dirname` matches the toolKey or the binary name, the whole directory tree is removed (`%LOCALAPPDATA%\cursor-agent\` including `versions/`). Otherwise just the single file is removed and empty parent dirs are cleaned up.

`cemir all` also wipes orphan entries from `cfg.Tools` that are no longer in the current `KnownTools` map (e.g., the v0.1.13 lineup trim left `gemini` behind in some users' configs).

Flags: `--yes` (no questions), `--config` (`~/.cem` and the local `.cem.yaml`),
`--plugin` (the IDE plugins), `--all` (everything, no questions). Without a
terminal it refuses rather than deleting unconfirmed.

`ideEklentiDizinleri()` walks the JetBrains config root and looks for
`cem-intellij` under every product directory (`GoLand2026.2`,
`PyCharm2026.1`, …) plus `~/.vscode/extensions/*cem-*`. Removing them with the
binary matters: a plugin left behind loads on every IDE start only to report
that cem is missing, and the per-product paths are tedious to find by hand.
The AI CLIs are never touched — `cemir` is for those.

---

## Fuzzy command matching

```
PS> cemi cluade
  'cluade' is unknown — did you mean 'claude'? (y/N): y
  ⏳ Claude installing...
```

`suggestTool` uses Levenshtein distance ≤ 2 against `KnownTools` keys. Exact match wins; otherwise the closest neighbour is suggested. Implemented in `fuzzy.go`.

---

## Server side (cem.pw)

The install/update URLs are served by an Apache vhost on the cem.pw host. Routing:

- `/install` and `/uninstall` are User-Agent-aware: PowerShell or WindowsPowerShell → `.ps1`; everything else → `.sh`
- Both scripts ship with `Content-Type: text/plain; charset=utf-8` (for Turkish chars in script output) and `Cache-Control: no-cache, max-age=300`
- `/r/<asset>` is a 302 redirect to `github.com/muslu/cem/releases/latest/download/<asset>` — so install scripts never need to know GitHub directly

The vhost config + UA rules + Apache rewrite rules live on the production host (not in this repo); the relevant snippets are in `OPERATIONS.md`.

---

## Known limitations

- **Ubuntu 18.04 + Codex**: glibc 2.27 < 2.28 → modern Node won't run → npm-installed Codex CLI fails. No path forward without OS upgrade or building Node from source.
- **Antigravity on Windows + PSReadline**: long OAuth codes get mangled by bracketed paste. Use `cem auth agy --code <kod>` clipboard helper.
- **Antigravity `--model`**: not supported by the CLI. Model selection happens outside cem.
- **Cursor login from CLI**: a Cursor desktop install is required for the OAuth handshake to succeed.

---

## File layout

```
cem/
├── main.go                — binary-name dispatch (cem/cemi/cemir from one binary)
├── banner.go              — ASCII banner + OpenSourceNotice
├── config.go              — GlobalConfig + ProjectConfig + ToolMeta + KnownTools
├── config_test.go         — KnownTools sanity + Roles resolution
├── executor.go            — runTool, captureTool, withKeyRotation, resolveModel, fallbackInstallPath
├── wizard.go              — RunSetupWizard, InstallTool, RemoveTool, ensureDep, askYN, askModel
├── spinner.go             — TTY-aware spinner
├── history.go             — AppendHistory → ~/.cem/history.log
├── fuzzy.go / fuzzy_test  — Levenshtein-based typo suggester
├── update_notice.go       — Hourly GitHub Releases polling + cached colored banner
├── cmd_cem.go             — root cobra: cem [input] + roles + setup + init + status
├── cmd_cemi.go            — cemi: install (with -y)
├── cmd_cemir.go           — cemir: uninstall (with -y, orphan cleanup)
├── cmd_update.go          — cem update (with sudo escalation on Unix)
├── cmd_uninstall.go       — cem uninstall (Windows self-delete via detached cmd)
├── cmd_doctor.go          — cem doctor
├── cmd_history.go         — cem history
├── cmd_keys.go            — cem keys add/list/remove
├── cmd_auth.go            — cem auth <tool> [--code]
├── cmd_endpoint.go        — cem endpoint (address / key / model of HTTP servers)
├── http_tool.go           — HTTP model servers: streaming chat, probe, model list
├── setup_apply.go         — flag-driven setup + cem status --json
├── surum-yayinla.sh       — one command: tests → tag → push → Marketplace
├── plugin/intellij/       — JetBrains plugin (docs/PYCHARM-PLUGIN.md)
└── .github/workflows/     — release.yml (7 platforms × 3 binaries + SHA256SUMS)
```
