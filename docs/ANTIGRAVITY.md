# cem — Antigravity IDE Integration (C)

Research notes on integrating cem with Google Antigravity IDE. This document tracks findings, since Antigravity's plugin/extensibility story is still evolving and not yet documented like JetBrains' IntelliJ Platform.

Turkish: [IDE.tr.md](IDE.tr.md) covers the IDE integrations, this page has no Turkish version yet.

Last updated: 2026-05-25.

---

## TL;DR

As of May 2026, Antigravity does **not have a published plugin API**. The integration patterns available today:

1. **Terminal-based use** — Antigravity ships with a terminal panel; `cem ...` runs natively. *(Done — see [IDE.md](IDE.md).)*
2. **Agent tool delegation** — Tell the Antigravity agent in chat to invoke `cem` as a sub-tool; it captures output and reasons over it. *(Done.)*
3. **Future: official plugin or MCP** — Track this document for updates.

---

## What we know about Antigravity's surfaces

| Surface | Available now | Notes |
|---|---|---|
| Terminal | ✅ | Full shell access; cem runs fine |
| Agent chat | ✅ | Agent can shell out; results returned in-chat |
| Slash commands (`/goal`, `/schedule`, `/cem` etc.) | ⚠ Built-in only | Custom slash commands not yet user-definable per public docs |
| External tool registration | ❓ | Possibly behind feature flags; nothing public |
| MCP (Model Context Protocol) | ❓ | If Antigravity adopts MCP servers, cem could expose itself as one |
| Extension API (JS, WASM, etc.) | ❌ | No SDK published |

---

## Strategy: MCP server wrapper

If Antigravity (or any MCP-compatible host) wants programmatic access to cem, the cleanest path is to wrap `cem` as an [MCP](https://modelcontextprotocol.io/) server. That gives:

- Tool-call interface: `think(prompt)`, `write(prompt)`, `pair(prompt)`
- Streaming responses
- Auth + key rotation stays inside cem
- Any MCP host (Claude Desktop, Codex CLI, future Antigravity?) gets cem for free

Scaffold target: `plugin/mcp/` — Go service that exec's `cem` per tool call. ~200 lines of Go using the official `modelcontextprotocol/sdk-go` package.

**Status:** not started. Will be opened as a separate tracking issue once cem v0.2.0 lands.

---

## Strategy: HTTP shim

If MCP isn't supported, the next best is a tiny HTTP server:

```
POST /think → { "prompt": "..." } → { "response": "..." }
POST /write → ...
POST /pair  → ...
```

Antigravity's agent (or any tool) can hit `http://localhost:7878/pair` and read the result. Trivial in Go: ~50 lines wrapping `exec.Command`. Lower priority than MCP because it's not standardized.

---

## Strategy: file-based protocol

A poor-man's integration: write a prompt to `~/.cem/inbox/<uuid>.txt`, watch the file with cem in daemon mode, write response to `~/.cem/outbox/<uuid>.txt`. Antigravity (or any tool) reads from outbox.

Awkward but works in any environment that has a filesystem. Use only if MCP and HTTP both unviable.

---

## Practical recommendation

Until Antigravity publishes a plugin API:

1. Use the **terminal** for direct invocation.
2. Use **agent delegation** for "have cem think about this, then come back to me" workflows.
3. Watch [antigravity.google/docs](https://antigravity.google/docs) for plugin/MCP/extension announcements.
4. When (if) Antigravity adopts MCP, cem will ship an MCP server in `plugin/mcp/`.

---

## Useful Antigravity docs to monitor

- [antigravity.google/docs/cli-getting-started](https://antigravity.google/docs/cli-getting-started) — covers `agy`, the terminal CLI
- [code.google.com/antigravity](https://antigravity.google/) — IDE landing
- Google I/O 2026 announcements — Antigravity 2.0 launch was where the CLI was unveiled; the desktop IDE plugin story may evolve

---

## Open questions

1. Does Antigravity respect an `mcp.json` file in the project root (like Claude Code)?
2. Can the desktop IDE invoke external binaries with structured prompts (not just terminal)?
3. Is there a way to register a custom side-panel? (PyCharm-style tool window.)

If you have insight from inside Google or from the Antigravity community, please open a GitHub issue: <https://github.com/muslu/cem/issues>.
