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
- [x] Request classifier accepts ASCII-written Turkish (olustur, duzelt,
      cevir): the missing variants silently produced a 49s run with no output.
- [x] Thinker no longer plans tests unless the task asked for them, and the
      writer is told to write the leanest code: same class of task went from
      8.2 KB across two files to 1.9 KB in one.
- [x] `memory/` moved into the repo (harness path is now a symlink) and
      refreshed with the project goal, the measure-before-shipping feedback and
      the user's environment constraints.
- [x] Writer cache: entries carry the produced files and restore them on a
      hit; edited files are never overwritten.
- [x] IntelliJ plugin: an empty selection no longer sends the whole open file
      as the prompt. README.md open + Ctrl+Alt+P used to spend 112s on a pair
      run that answered "nothing new to write"; a prompt dialog opens instead,
      as `plugin.xml` already claimed. Tab titles are stripped of markdown and
      emoji, and the output pane word-wraps instead of cutting lines off.
- [x] A question no longer produces a file: "lua'da hello world nasıl
      yazılır?" ran the writer (because the thinker's answer had a code block)
      and left an unwanted `hello.lua` in the working directory. Questions skip
      the writer; polite imperatives ("siler misin?") still don't.
- [x] IntelliJ plugin: the prompt no longer arrives in a modal dialog. A
      shortcut with no selection focuses a multi-line input box at the bottom
      of the tool window with the mode preselected (`pair`/`think`/`write`
      selector on its left); Enter sends, Shift+Enter adds a line, ↑/↓ still
      walk the history. "ask about file…" attaches the file as context and
      waits for the instruction in the same box.

## JetBrains Marketplace (open)

Repo side is done: signing + `verifyPlugin` wired into `build.gradle.kts`,
`CHANGELOG.md` added (the `changeNotes` link pointed at a missing file),
plugin name shortened to `cem` (Marketplace rejects punctuation used as a
separator and wants ≤20 characters), `pluginVersion` moved to CalVer so it
matches the release tags, and a dormant `publish-intellij-plugin` CI job that
turns on when the repo variable `PUBLISH_MARKETPLACE=true` is set.

Two latent build bugs surfaced while validating this:
- Bytecode target was Java 21 while `sinceBuild=233` IDEs run JBR 17 — the
  plugin could not load at all on 2023.3–2024.1 (`UnsupportedClassVersionError`).
  Now compiled with `--release 17` / `jvmTarget = 17` on a JDK 21 toolchain.
- `buildSearchableOptions { enabled = false }` left `prepareJarSearchableOptions`
  expecting a directory that a clean checkout never creates, so `clean
  buildPlugin` always failed; it only passed locally because an old build
  output kept the directory alive. The whole chain is disabled now.

Left for the user (cannot be automated):
- [ ] JetBrains account + Marketplace vendor profile.
- [ ] Generate the signing key (`openssl genpkey` → `private.pem`,
      `chain.crt`), keep it out of the repo (`.gitignore` covers `*.pem` /
      `*.crt`), then `./gradlew signPlugin -Pcem.signDir=$HOME/.cem-signing`.
- [ ] Upload the **ZIP** (not the JAR — `snakeyaml-engine` ships inside it)
      manually at plugins.jetbrains.com/plugin/add; the first publication must
      always be manual, and new plugins go through moderation.
- [ ] Real IDE screenshots at ≥1200×760 — `docs/img/cem-intellij.png` is
      900×410 and is a drawing, not a screenshot.
- [ ] Pick tags/categories and confirm the MIT license in the listing form.
- [ ] After approval: create the `PUBLISH_TOKEN`, `CERTIFICATE_CHAIN`,
      `PRIVATE_KEY`, `PRIVATE_KEY_PASSWORD` secrets and set
      `PUBLISH_MARKETPLACE=true`.

### Marketplace upload rejected the plugin ID (fixed)

`dev.cempw.intellij` was refused at upload: *"The plugin ID should not include
the word 'intellij'"*. Changed to **`dev.cempw.cem`** — in `plugin.xml` and in
the `updatePlugins.xml` the release workflow generates. The Kotlin package is
still `dev.cempw.intellij`; Marketplace does not care about package names.

**Consequence:** an already installed `dev.cempw.intellij` build is a different
plugin as far as the IDE is concerned. It will not auto-update to the new ID —
uninstall the old one once, then install the new build.

`plugin/intellij/yayinla.sh` now does the whole local release: installs the
missing packages (`nala`), finds or installs JDK 21, generates the signing key
on first run, then builds → signs → verifies the signature.
`--surum <tag>`, `--dogrula`, `--yayinla`, `--anahtar-yenile`.

- [x] doc:CLAUDE update — `Git & Release` now documents the one-command
      `surum-yayinla.sh` flow, the Marketplace listing (34196) and the CalVer
      tag scheme that was already in use but written up as semver.
- [x] Marketplace listing is live: id `dev.cempw.cem`, page 34196. The
      compatibility verifier's two warnings are fixed
      (`SimpleListCellRenderer.create(String, Function)` was scheduled for
      removal, `doWhenFocusSettlesDown(Runnable)` deprecated).
- [x] 1200×760 listing images: `docs/img/market-{pair,input,menu}[.tr].png`,
      regenerated by `docs/img/market-gorseller.py`. They are IDE mock-ups
      built from the real UI, not captures — a real screen capture still beats
      them if we ever stage the IDE for it.
- [x] README badges point at the Marketplace listing. GitHub strips `<iframe>`
      and `<script>`, so the JetBrains embeddable card/install widgets only
      work on cem.pw — snippets are in `OPERATIONS.md`.
- [x] `20260910.03` shipped: GitHub release (32 assets) + a signed zip uploaded
      to the Marketplace. The Marketplace still reports `approve: false` — a new
      plugin sits in moderation, and versions stay invisible until it clears.
- [x] Local / self-hosted models: `ollama`, `lmstudio`, `unsloth` are tools
      now. They are not subprocesses — cem speaks HTTP (OpenAI-compatible
      `/v1/chat/completions` and ollama's `/api/chat`) and streams the answer.
      `cem endpoint <tool> <ip:port> [--model X] [--key K] [--test]
      [--modeller] [--here]`. Install/remove/auto-update/effort/fast are
      skipped for them, and the model name is asked of the server, never
      guessed.
- [x] Setup is mandatory: cem refuses to run unconfigured. With a terminal it
      offers the wizard and re-verifies what was saved (a wizard interrupted
      with Ctrl+C left the same half state); without one (plugin, pipe, CI) it
      prints `cem setup` and exits.
- [x] Plugin: keep talking in a run tab. Each run tab now has an input box and
      carries the earlier turns as context (trimmed to 4000 characters —
      context is billed again every turn). A question from the tool had nowhere
      to be answered; every request was one-shot.
- [x] Plugin: Terminal tab. Commands run in the project root through the shell
      (pipes, `&&`, redirects work), `⏹` stops one. Not a full pty — the IDE's
      own terminal stays the place for interactive programs.
- [x] Plugin: multiple terminal tabs. `＋` opens another one — a long-running
      command (`go run`) used to block the single tab. The first tab is fixed;
      the extras close, and closing one kills its command. Numbering comes from
      the highest existing number, not the tab count, which would have produced
      a second "Terminal 3" after closing "Terminal 2".
