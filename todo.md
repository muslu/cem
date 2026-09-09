# CEM — TODO

> Turkish version: [todo.tr.md](todo.tr.md)

## 1. Folder & File Layout
- [x] Move `uninstall/cmd_uninstall.go` → root (`cmd_uninstall.go`)
- [x] Move `uninstall/uninstall.sh` → root
- [x] Move `uninstall/uninstall.ps1` → root
- [x] Delete the now-empty `uninstall/` directory
- [x] Reorganize `nginx/` (snippets/, sites-available/, fail2ban/)

## 2. Missing Core Go Files
- [x] `main.go` — dispatch by binary name (cem / cemi / cemir)
- [x] `config.go` — GlobalConfig + ProjectConfig + ResolvedConfig + Roles
      + InstalledTool + KnownTools + LoadConfig / saveGlobalConfig
      / loadGlobalConfig / SaveProjectConfig
- [x] `executor.go` — ModeThink/ModeWrite/ModePair + Run + ReadStdin

## 3. Build & Dependencies
- [x] Populate `go.mod` (cobra + lipgloss + yaml.v3)
- [x] `go mod tidy`
- [x] `Makefile` (build / dev / install / clean / tidy / test)
- [x] `go build` smoke test — 3 binaries (3.9 MB each with ldflags)

## 4. CI/CD
- [x] `.github/workflows/release.yml` (7 platforms + SHA256SUMS)
- [x] `.gitlab-ci.yml` mirror (kept in tree; canonical is GitHub)

## 5. Verification
- [x] `./build/cem --help` works
- [x] `./build/cemi --version` → 1.0.0
- [x] `./build/cemir --version` → 1.0.0

## 6. Extras (initial session)
- [x] `.claude/` audit (agents/skill/hooks)
- [x] `CLAUDE.md` rewritten for the project (English) + `CLAUDE.tr.md`
- [x] `install.sh` + `install.ps1` URLs → GitHub canonical

## 7. New features (follow-up session)
- [x] `cem doctor` command (system + roles + tools + PATH)
- [x] `cemir all` bulk uninstall (with confirmation + failure summary)
- [x] LDFLAGS version injection from `git describe`
- [x] `config_test.go` — 8 unit tests including deprecation and order
- [x] `.gitlab-ci.yml` release stage — 21 binary asset links + SHA256SUMS
- [x] `.gitignore` rewritten for CEM
- [x] Git init + GitHub origin

## 8. Persistence & UX (follow-up session)
- [x] `~/.cem/history.log` + `cem history` (-n / --clear)
- [x] `cem -p` spinner — TTY-aware, no bubbletea dep
- [x] nginx `/r/` proxy → GitHub Releases
- [x] CLAUDE.md / README updates

## 9. Brand + tool catalogue (current session)
- [x] Slogan: **CEM — Compose · Execute · Multiplex**
      ("One command, many AIs.")
- [x] Canonical repo flipped to `https://github.com/muslu/cem.git`
- [x] `agy` description corrected → **Antigravity (Google)**
- [x] `gemini` deprecation note (personal use ends 2026-06-16)
- [x] 5 new AI CLIs added: goose, cody, continue, openhands, cursor
- [x] `orderedToolKeys` introduced; 4 duplicated `[]string{...}` lists
      reduced to one source of truth
- [x] All MD docs split English/Turkish (README/CLAUDE/OPERATIONS/todo)

## 10. v0.1.x deployment session (live tags v0.1.0 → v0.1.15)
- [x] cem.pw → Apache vhost (production), `/install` `/uninstall` `/r/*`
      UA-aware (PowerShell → .ps1, curl → .sh)
- [x] GitLab tracks removed, GitHub muslu canonical
- [x] PowerShell UTF-8: BOM dropped, charset=utf-8 header,
      `[Console]::OutputEncoding=UTF8` in scripts
- [x] install.sh escape sequences (`$(printf '\033')`) — colors render
- [x] "döküman" → "doküman", docs link → github.com/muslu/cem
- [x] `cem update` — GitHub API ile son sürüm önizlemesi, Windows
      self-replace via .old rename, Linux sudo escalation
- [x] `cem uninstall` Windows self-delete via detached cmd
- [x] Tool lineup trimmed 10 → 4: claude, agy, gpt(codex), cursor
- [x] Native installers: claude.ai/install.sh, antigravity.google/cli/...,
      cursor.com/install
- [x] `ToolMeta.Binary` — toolKey ≠ PATH binary (cursor→cursor-agent, gpt→codex)
- [x] `ToolMeta.RunFlags` + `PromptAsArg` (codex exec, agy/cursor -p arg)
- [x] `fallbackInstallPath` for installer's PATH-unaware drops (agy, cursor)
- [x] Pair mode skip logic (thinker == writer; no code in input/output)
- [x] `ensureDep` Linux node via **nvm** (NodeSource glibc 2.28+ kills bionic);
      `depVersionOK` rejects ancient npm 3.x
- [x] `cemi -y` + `cemir -y` flags; askYN honors autoYes
- [x] `cemir all` shell-install path support (curl|bash binaries deleted)
- [x] **API key management** — `cem keys add/list/remove` + provider
      rotation on rate-limit (anthropic, openai)
- [x] Silent install: cmd output captured, only last 12-15 lines on error
- [x] Codex: `--skip-git-repo-check` so it runs outside git repos

## 11. Open (deferred / decision pending)
- [ ] Ubuntu 18.04 (glibc 2.27) — Codex unreachable; nvm Node 24 still
      requires 2.28. No clear path without OS upgrade. Document as
      known-incompatible.
- [ ] Antigravity `iwr | iex` Windows output sometimes hides installer
      progress — investigate or accept.
- [ ] Cursor + Antigravity API key rotation: providers don't publish CLI
      env-var docs yet (OAuth only). Skipped from rotation.
- [ ] `history_test.go` — coverage gap remains (`executor_test.go`,
      `spinner_test.go` done).
- [ ] LICENSE file (README references MIT but no LICENSE in the tree).
- [ ] `.claude/agents/` cleanup decision still pending.

## 12. Reliability + effort/auto-update (2026-09-09)
- [x] `Spinner.Stop()` panic: `close of closed channel` when both
      `stopWriter.Write` and `Run`/`ModePair` stopped it — now `sync.Once`,
      nil-safe, race-tested.
- [x] Error triage no longer matches signatures inside dumped HTTP bodies
      (`sanitizeStderr`); codex's 105 KB model JSON was reported as an auth
      failure while the real error was "model not supported".
- [x] Model errors get their own hint (`hintModel`) instead of sending the
      user into a pointless login flow.
- [x] `captureToolWithSpinner` no longer exits silently — pair mode used to
      return exit 1 with no cem message at all.
- [x] Writer prompt strips the thinker's duplicated final message
      (`dedupeTrailingEcho`) — codex exec prints it twice, doubling tokens.
- [x] Reasoning effort is selectable: `cem effort`, wizard step, `cem init`
      step, `.cem.yaml > efforts`, shown in the run header.
- [x] `cemi update` uses each tool's own `update` subcommand; daily detached
      auto-update for installed CLIs (`auto_update_tools: false` to disable).
- [x] `executor_test.go` + `spinner_test.go` added (28 tests, `-race` clean).
- [ ] doc:CLAUDE update — File Layout, Runtime Gotchas, conventions #7,
      validation flow refreshed for the above.

## 13. Language + output clarity (2026-09-09)
- [x] Bilingual UI (tr/en): `L(tr, en)` helper, `cem lang`, wizard asks the
      language first, `CEM_LANG`/`LANG` detection, `applyLang()` for cobra
      Short/Long (package-init cycle otherwise).
- [x] Output noise filter (`noise.go`) + `--raw` escape hatch.
- [x] codex runs through `--output-last-message`: no more exec/apply-patch/diff
      spam and no more duplicated final answer.
- [x] Clearer run headers: `🧠 THINKING · gpt` (blue) / `✍️ WRITING · claude`
      (green); the "Open source" line only shows on the banner screen and the
      global-config line is silent (project config still announced).
- [ ] README / README.tr: document `cem lang`, `cem effort`, `--raw`.
- [x] Quiet-mode double print: the URL passthrough leaked whole answers; removed.
- [x] `cem model` command (symmetric with `cem effort`); plugin no longer
      shows an auth balloon for every URL in the output.
- [x] Pair mode: thinker plans, writer codes (was: both wrote the full code).
      `cem model` + roles table now shows model · effort per role.
- [x] Per-role timings (`⏱ thinking 18.4s` / `⏱ writing 9.2s` / total) and a
      separator line between the thinker's and the writer's output.
- [x] IntelliJ plugin auto-update via a custom plugin repository
      (`updatePlugins.xml` published with every release).
- [x] `minimal` is no longer a valid codex effort; effort rejections get their
      own hint instead of being reported as a model problem.
- [x] Answer cache (`cem cache`, `--no-cache`), live elapsed time in the
      spinner, per-directory trust prompt, simplified README + README_DETAILS,
      SVG diagrams, richer plugin Overview.
- [x] Fast mode (`cem fast`): skips the tool's user settings — measured
      124s → 8s on the same task. Off by default.
- [x] Writer now shows a live spinner with elapsed time; hook noise filtered.
- [x] Language-specific diagrams (`*.svg` English, `*.tr.svg` Turkish).
- [x] Fast mode is now the default; role-based effort defaults
      (`applyRoleDefaults`); `cem doctor` audits the cost setup; the writer is
      skipped when the thinker asked for information instead of planning.
- [x] `--no-cache` now refreshes the stored answer instead of leaving the old
      one in place (it skips reading, not writing).
- [x] Cache correctness: key includes the working directory; clarification
      answers ('file not found, please share it') are never stored.
- [x] Output filter corrupted code: the "drop a repeated line" rule ate the
      second `}` of nested blocks. Removed, with tests. Blank-line collapsing
      relaxed to two so Python spacing survives.
- [x] A malformed timestamp in the config no longer bricks every command.
- [x] Writer no longer tries to run tests and ask for approval; the plan is
      capped at 15 lines. Same task: 1m 38s → 1m 06s.
- [x] Run headers carry a timestamp; pair total shows the full date.
- [x] IntelliJ settings gained reasoning effort + fast mode, and the model list
      was refreshed (gpt-5.6-terra).
