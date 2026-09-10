# Changelog

All notable changes to **cem** and its IntelliJ Platform plugin.
Versions follow `YYYYMMDD.MINOR` (calendar versioning); tags on
[github.com/muslu/cem](https://github.com/muslu/cem/releases) are the source of
truth. Turkish version: [CHANGELOG.tr.md](CHANGELOG.tr.md).

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
