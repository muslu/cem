# CEM — full guide

Everything beyond the [README](README.md). Türkçe:
[README_DETAILS.tr.md](README_DETAILS.tr.md).

- [1. Install](#1-install)
- [2. First run](#2-first-run)
- [3. The three commands](#3-the-three-commands)
- [4. Language](#4-language)
- [5. Models and reasoning effort](#5-models-and-reasoning-effort)
- [6. Login and API keys](#6-login-and-api-keys)
- [7. Output](#7-output)
- [8. Update / uninstall](#8-update--uninstall)
- [9. IDE integrations](#9-ide-integrations)
- [10. Downloads](#10-downloads)
- [11. Troubleshooting](#11-troubleshooting)

---

## 0. Defaults that keep the bill down

You do not have to configure anything. Out of the box cem is set up so the
expensive model decides and the cheap one types:

| Default | Why |
|---|---|
| In pair mode the thinker **does not write code** | otherwise both models solve the task and the same work is billed twice |
| Thinker effort **high**, writer effort **low** | the thinker produces the plan; the writer only implements it |
| The thinker's answers are **cached** | asking the same thing twice does not pay for the same reasoning twice |
| Both roles are **cached** | the same request gives the same answer; the writer's entry also carries the files it produced, and they are restored |
| **Fast mode on** | the tool's hooks/permission rules are not reloaded on every call — measured 124s → 8s |
| The writer is **skipped** when there is nothing to write | no code task, or the thinker asked for missing information |
| Tool banners and logs are **filtered** | they also inflate the prompt handed to the writer |

`cem doctor` checks this setup and says where you are paying more than you
need to. Every default can be overridden — see the sections below.

## 1. Install

**macOS / Linux / WSL:**
```sh
curl -fsSL cem.pw/install | sh
```

**Windows (PowerShell — not CMD, not Git Bash):**
```powershell
irm cem.pw/install | iex
```

You get three commands on your PATH:

| Command | What it does |
|---|---|
| `cem` | the orchestrator |
| `cemi` | installs AI CLIs (claude, agy, gpt/codex, cursor) |
| `cemir` | removes them |

## 2. First run

```sh
cem "what is fibonacci?"
```

A short wizard opens the first time and asks, in order:

1. **Language** — Turkish or English (everything after this is in that language)
2. **Which AI thinks** and **which one writes**
3. **Model** for each (Enter = the CLI's own default)
4. **Reasoning effort** for each, where the CLI supports it

If a tool isn't installed, cem offers to install it. If Node.js or another
prerequisite is missing, cem offers to install that too — via `winget`
(Windows), `brew` (macOS) or `nvm` (Linux).

Re-run it any time: `cem setup`.

## 3. The three commands

```sh
cem "explain this codebase"         # THINKER — one AI
cem -w "write a quicksort in Go"    # WRITER — one AI
cem -p "build me a CLI tool"        # PAIR — thinker plans, writer codes
```

Pair mode is the point of the tool. The thinker is told **not to write the
code**: it produces a short plan (which file, which signature, which approach,
which edge cases) and that plan is handed to the writer with an explicit "do
not re-analyse, just implement" instruction.

That split is what saves tokens. Without it both models solve the whole task
and you pay for the same work twice — the expensive one included.

The writer is skipped when there is nothing to write: if the request isn't a
code task and the thinker produced no code block, cem stops after the thinker.

Input can also come from a file or a pipe:

```sh
cem -f main.go -p "make this context-aware"
git diff | cem -p "review this change"
```

## 4. Language

```sh
cem lang           # show the current language and where it came from
cem lang tr        # Turkish
cem lang en        # English
```

Priority: `CEM_LANG` env var → `~/.cem/config.yaml` → system locale (`LANG`) →
English.

## 5. Models and reasoning effort

```sh
cem model                        # active model per tool + where it comes from
cem model gpt                    # known models for that tool
cem model gpt gpt-5.6-terra      # set globally
cem model --here claude sonnet   # this project only (.cem.yaml)
cem model gpt default            # clear it — the CLI decides
```

```sh
cem effort                       # active reasoning effort per tool
cem effort gpt xhigh             # set globally
cem effort --here claude low     # this project only
cem effort gpt default           # clear it
```

| Tool | Model flag | Effort levels |
|---|---|---|
| claude | `--model` | low, medium, high, xhigh, max |
| gpt (codex) | `--model` | low, medium, high, xhigh, max |
| cursor | `--model` | — |
| agy | — (no flag yet) | — |

Effort levels are suggestions: the accepted set depends on the model. If the
tool rejects one, cem tells you what it accepts instead of failing silently.

**The recommended setup for pair mode** — expensive model thinks, cheap model
types:

```sh
cem model  gpt    gpt-5.6-terra
cem effort gpt    xhigh
cem model  claude sonnet
cem effort claude low
```

`cem roles` then shows the whole setup at a glance:

```
┌── Active Roles ────────────────────────────────────────┐
│ 🧠 thinker   gpt · gpt-5.6-terra · xhigh   cem "task"  │
│ ✍️  writer   claude · sonnet · low         cem -w "…"  │
│ 🤝 pair      gpt → claude                  cem -p "…"  │
└────────────────────────────────────────────────────────┘
```

Project overrides live in a `.cem.yaml` next to your code and win over the
global config:

```yaml
roles:
  thinker: gpt
  writer: claude
models:
  gpt: gpt-5.6-terra
  claude: sonnet
efforts:
  gpt: xhigh
  claude: low
```

Create one with `cem init` (wizard) or `cem init gpt claude` (direct).

## 6. Login and API keys

After installing an AI tool, cem asks how to authenticate:

```
  auth for Claude:
    [1] Store an API key  (multiple keys + auto-rotation on rate limit)
    [2] Subscription / OAuth login  (run 'claude' to start the browser flow)
    [3] Skip
```

Pick **1** for an API key (Anthropic Console / OpenAI Platform), **2** for a
subscription (Claude Pro, ChatGPT Plus, Antigravity, Cursor).

```sh
cem keys add anthropic     # paste your sk-ant-… key
cem keys list              # masked view
cem keys remove openai 2   # delete the 2nd OpenAI key
cem auth gpt               # re-run a tool's login flow
```

Multiple keys are tried in order; when one hits a rate limit cem rotates to the
next automatically, so a long session isn't interrupted.

> ChatGPT **free** plan cannot use Codex models — codex returns
> "model … does not exist or you do not have access to it". A paid plan or an
> OpenAI API key is required.

## 7. Output

cem filters what the AI CLIs print: banners, session ids, internal logs and
repeated lines are stripped so the answer is what you see. Tools that can write
their final message to a file (codex) are run quietly — a spinner runs while
they work and the answer is printed once, instead of the tool echoing every
command it ran and every diff it produced.

```sh
cem --raw -p "…"       # turn filtering off: raw tool output, banners and all
cem --no-cache "…"     # ignore the stored answer, ask again and store the new one
```

Both roles are cached. The writer's entry also stores the files it created, so
a cache hit restores them:

```
  ♻ from cache (4m 12s ago) · re-run it with: --no-cache
  ↺ 2 file(s) restored
```

A file that changed in the meantime is **never overwritten** — cem reports it
and leaves your edit alone. Turn the writer cache off with
`cache_writer: false`.

Each role prints how long it took, and pair mode adds the total:

```
  ⏱ thinking 8.9s
  ────────────────────────────────────────────────────
  ⏱ writing 1m 30s
  ⏱ total 1m 39s   (thinking 8.9s + writing 1m 30s)
```

Every run is recorded:

```sh
cem history           # last 20 runs
cem history -n 100    # last 100
cem history --clear
```

## 8. Update / uninstall

```sh
cem update         # fetch the latest cem release
cemi update        # update the installed AI CLIs (claude, codex, agy, cursor)
cemi update gpt    # just one
cem uninstall      # remove cem itself
cemir all          # remove the AI tools
```

cem also updates the installed AI CLIs **by itself**, once a day, in the
background — it runs each tool's own `update` command detached from cem, so
your command never waits. The log is `~/.cem/auto-update.log`. Turn it off with
`auto_update_tools: false` in `~/.cem/config.yaml`.

> **Versions:** calendar versioning `YYYYMMDD.MINOR` since 2026-05-25 (e.g.
> `20260909.06`). Older `v0.1.x` semver tags still work; `cem update`
> understands both and only offers an update when the remote tag is newer.

## 9. IDE integrations

| Editor | Quick install | Guide |
|---|---|---|
| **PyCharm / IntelliJ IDEA / GoLand / WebStorm / RubyMine / PhpStorm / Rider / DataGrip / CLion / RustRover** | plugin zip from disk | [docs/INTELLIJ.md](docs/INTELLIJ.md) |
| **VS Code** | `code --install-extension cem-vscode.vsix` | [docs/VSCODE.md](docs/VSCODE.md) |
| **Cursor** | same vsix, plus optional MCP | [docs/CURSOR.md](docs/CURSOR.md) |
| **Claude Desktop** | `cem-mcp` MCP server | [docs/CLAUDE-DESKTOP.md](docs/CLAUDE-DESKTOP.md) |
| **Continue.dev** | `cem-mcp` MCP server | [docs/CONTINUE.md](docs/CONTINUE.md) |
| **Antigravity IDE** | built-in terminal | [docs/ANTIGRAVITY.md](docs/ANTIGRAVITY.md) |
| **Vim / Neovim** | shell function recipes | [docs/VIM.md](docs/VIM.md) |
| **Emacs** | elisp recipes | [docs/EMACS.md](docs/EMACS.md) |

![cem in a JetBrains IDE](docs/img/cem-intellij.svg)

### JetBrains plugin auto-update

The plugin is not on the Marketplace, and a plugin installed from disk is never
updated. Add this repository once and the IDE finds updates itself:

**Settings → Plugins → ⚙ → Manage Plugin Repositories → `+`**

```
https://github.com/muslu/cem/releases/latest/download/updatePlugins.xml
```

Mind the order of the path segments: `releases/latest/download/…`, not
`releases/download/latest/…`.

### Slash command

```sh
cem install-slash          # installs /cem into the supported AI CLIs
```

## 10. Downloads

Always-current URLs (they redirect to the latest release):

| Asset | URL |
|---|---|
| IntelliJ plugin (all JetBrains IDEs) | https://github.com/muslu/cem/releases/latest/download/cem-intellij.zip |
| JetBrains update repository | https://github.com/muslu/cem/releases/latest/download/updatePlugins.xml |
| VS Code extension (also Cursor) | https://github.com/muslu/cem/releases/latest/download/cem-vscode.vsix |
| MCP server — Linux x86_64 | https://github.com/muslu/cem/releases/latest/download/cem-mcp-linux-amd64 |
| MCP server — Linux arm64 | https://github.com/muslu/cem/releases/latest/download/cem-mcp-linux-arm64 |
| MCP server — macOS Intel | https://github.com/muslu/cem/releases/latest/download/cem-mcp-darwin-amd64 |
| MCP server — macOS Apple Silicon | https://github.com/muslu/cem/releases/latest/download/cem-mcp-darwin-arm64 |
| MCP server — Windows | https://github.com/muslu/cem/releases/latest/download/cem-mcp-windows-amd64.exe |
| cem binary — Linux x86_64 | https://github.com/muslu/cem/releases/latest/download/cem-linux-amd64 |
| cem binary — macOS Apple Silicon | https://github.com/muslu/cem/releases/latest/download/cem-darwin-arm64 |
| cem binary — Windows | https://github.com/muslu/cem/releases/latest/download/cem-windows-amd64.exe |

All releases (versioned filenames + changelog):
https://github.com/muslu/cem/releases

## 11. Troubleshooting

```sh
cem doctor      # tools, config, PATH, roles — one report
cem status      # installation status
cem roles       # who thinks, who writes, with model and effort
```

| Symptom | Cause / fix |
|---|---|
| `✗ … not found — install it with: cemi <tool>` | the AI CLI isn't installed |
| `⚠ … is not authorized` | login missing → `cem auth <tool>` or `cem keys add <provider>` |
| `⚠ model '…' is not available on this account/plan` | wrong model for your plan → `cem model <tool> <name>` |
| `⚠ reasoning effort '…' was rejected` | that level isn't valid for this model → `cem effort <tool> high` |
| `⚠ all … keys are rate limited` | every key hit its limit → add another with `cem keys add` |
| `(nothing to write, writer skipped)` | the request wasn't a code task and the thinker produced no code |
| Answer looks truncated or odd | try `cem --raw` to see the tool's unfiltered output |
| IDE plugin can't find `cem` | set the absolute path in **Settings → Tools → cem** |

Config files:

| Path | Contents |
|---|---|
| `~/.cem/config.yaml` | tools, roles, models, efforts, API keys, language (mode `0600`) |
| `.cem.yaml` | per-project roles/models/efforts — overrides the global config |
| `~/.cem/history.log` | run history (TSV) |
| `~/.cem/auto-update.log` | background AI CLI updates |

Deeper material — project configs, OAuth code helper, build from source,
internals — is in [ADVANCED.md](ADVANCED.md).
