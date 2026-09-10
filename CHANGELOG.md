# Changelog

All notable changes to **cem** and its IntelliJ Platform plugin.
Versions follow `YYYYMMDD.MINOR` (calendar versioning); tags on
[github.com/muslu/cem](https://github.com/muslu/cem/releases) are the source of
truth. Turkish version: [CHANGELOG.tr.md](CHANGELOG.tr.md).

## 20260910.06

- `cem uninstall` gained `--yes`, `--config`, `--plugin` and `--all`, and it now
  finds and removes the IDE plugins. JetBrains keeps one copy per product and
  version, and a plugin left behind loads on every IDE start only to report that
  cem is missing. Without a terminal it refuses rather than deleting unconfirmed.

## 20260910.05

- Setup can be done without a terminal: `cem setup --thinker X --writer Y`
  (plus `--model-*`, `--effort-*`, `--endpoint-*`, `--lang`) and
  `cem status --json`. Making setup mandatory had left the IDE plugin and CI
  with no way to configure cem.
- **Plugin:** Settings → Tools → cem is now the setup page. It reads
  `cem status --json` and Apply runs `cem setup`, so cem validates the choices
  and stays the only writer of `~/.cem/config.yaml`. HTTP tools get an address
  field; a run that fails for want of setup offers to open the page.

## 20260910.04

- Local and self-hosted models: `ollama`, `lmstudio` and `unsloth` are tools
  now. cem speaks HTTP to them (OpenAI-compatible `/v1/chat/completions` and
  ollama's `/api/chat`) and streams the answer. `cem endpoint <tool> <ip:port>`
  sets the address; the model name is asked of the server, never guessed.
- Setup is mandatory: cem refuses to run unconfigured instead of sending the
  request to a tool the user never chose.
- The OAuth code now reaches the tool. For tools that take the prompt as an
  argument, the child process had no stdin, so an interactive login prompt had
  nothing to read: on Windows the pasted code went to PowerShell, which read
  `4/0ATs…` as a command.
- **Plugin:** keep talking in a run tab (the earlier turns travel as trimmed
  context), a Terminal tab that runs commands in the project root, and `＋` for
  more terminal tabs.

## 20260910.03

- The plugin is listed on the JetBrains Marketplace (id `dev.cempw.cem`). The
  first upload was rejected because a plugin ID may not contain the word
  `intellij`.
- Two compatibility warnings from the Marketplace verifier are fixed:
  `SimpleListCellRenderer.create(String, Function)` was scheduled for removal
  and `doWhenFocusSettlesDown(Runnable)` was deprecated.
- The plugin is compiled for Java 17 instead of 21: `sinceBuild=233` IDEs run
  JBR 17, so the 21 bytecode could not load there at all.

## 20260910.02

- **Plugin:** the `-p` / `-w` prompt is typed in an input box at the bottom of the
  cem tool window instead of a modal dialog. All four action paths (think / write /
  pair / ask) route through it.

## 20260910.01

- The writer role no longer creates files when the request is a question. Asking
  "how do I write hello world in lua?" used to leave a `hello.lua` in the working
  directory and cost a second AI call.

## 20260909.20

- Fixed the pair-mode spinner overwriting the first lines of the writer's answer.

## 20260909.19

- **Plugin:** with no editor selection, the whole open file is no longer sent as
  the task; the input box is focused instead.

## 20260909.18

- The writer role is cacheable too: a cache hit restores the files the run
  produced, and a file changed in the meantime is never overwritten — the
  conflict is reported instead.

## 20260909.09

- Fast mode (on by default): skips the AI CLI's own user settings — hooks,
  permission rules, MCP — and auto-approves edits. Measured on the same task
  with claude 2.1.266: **124s → 8s**.
- Live spinner during the writer stage, per-stage timings, language-specific
  diagrams.

## Earlier

See the [release list](https://github.com/muslu/cem/releases) for tags before
`20260909.09`.
