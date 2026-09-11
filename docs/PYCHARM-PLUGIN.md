# cem — JetBrains plugin internals

Development notes for the IntelliJ Platform plugin. For *using* it, read
[docs/INTELLIJ.md](INTELLIJ.md).

Source: [`plugin/intellij/`](../plugin/intellij/)

---

## Status

Published. The plugin is listed on the JetBrains Marketplace as
[**cem**](https://plugins.jetbrains.com/plugin/34196), plugin id
`dev.cempw.cem`, and every cem release also attaches
`cem-intellij-<version>.zip` plus an `updatePlugins.xml` feed to the GitHub
release.

Two details worth remembering:

- **The plugin id cannot contain `intellij`.** The first upload as
  `dev.cempw.intellij` was rejected for exactly that. The id is immutable now
  that the plugin is published — the Kotlin package is still
  `dev.cempw.intellij`, which Marketplace does not care about.
- **The name is `cem`, not `cem — multi-AI orchestrator`.** Marketplace's
  listing rules reject punctuation used as a separator and want ≤20 characters.

## Build

JDK 21 is required to *compile*, but the bytecode target is **17**:
`sinceBuild=233` IDEs run JBR 17, and Java 21 bytecode fails to load there with
`UnsupportedClassVersionError`. `build.gradle.kts` sets `--release 17` and
`jvmTarget = 17` on a JDK 21 toolchain; `verifyPluginProjectConfiguration`
reports it if that ever drifts.

```sh
cd plugin/intellij
./gradlew buildPlugin                       # build/distributions/cem-intellij-<version>.zip
./gradlew verifyPluginProjectConfiguration  # local checks, no downloads
./gradlew verifyPlugin                      # IntelliJ Plugin Verifier (downloads IDEs)
./gradlew runIde                            # sandbox IDE with the plugin loaded
./gradlew test --no-configuration-cache     # see below
```

`./gradlew test` on its own fails while writing Gradle's configuration cache
(KotlinCompile's classpath snapshot), and `--offline` fails because
`test-framework` and junit are not in the local cache.

`buildSearchableOptions` is disabled, and so are `prepareJarSearchableOptions`
and `jarSearchableOptions` — with only the first task off, the second expected a
directory that a clean checkout never creates, so `clean buildPlugin` always
failed. It passed locally only because an old build output kept the directory
alive.

## Sign and publish

Marketplace requires a signed plugin. `plugin/intellij/yayinla.sh` does the
whole local release: installs missing packages, finds or installs JDK 21,
generates the signing key on first run, then builds → signs → verifies the
signature.

```sh
./yayinla.sh                       # build + sign
./yayinla.sh --surum 20260911.01   # force a version
./yayinla.sh --yayinla             # upload to the Marketplace
./yayinla.sh --anahtar-yenile      # regenerate the signing key
```

Keys live outside the repo in `~/.cem-signing/` (`private.pem`, `chain.crt`,
optionally `parola` and `token`, all `chmod 600`). A wrong passphrase used to
surface as a 20-line BouncyCastle stack trace (`pad block corrupted`); the
script now checks the key with `openssl` first and says what to do.

From the repo root, `./surum-yayinla.sh` releases everything at once: tests →
next `YYYYMMDD.NN` tag → push (CI builds the binaries, the plugin zip and the
GitHub release) → sign and upload the plugin.

## Layout

```
plugin/intellij/
├── build.gradle.kts                — IntelliJ Platform Gradle Plugin 2.x, signing, verification
├── gradle.properties               — pluginVersion (CI syncs it to the tag)
├── gradlew                         — wrapper, Gradle 9.3.1 (pinned; `gradle` may not be on PATH)
├── updateplugins_uret.py           — updatePlugins.xml generator; description comes from plugin.xml
├── yayinla.sh                      — local build + sign + publish
└── src/main/
    ├── kotlin/dev/cempw/intellij/
    │   ├── CemAction.kt            — Think / Write / Pair / Ask actions, run + stream, notifications
    │   ├── CemToolWindow.kt        — tool window: Interactive, Terminal and run tabs
    │   ├── CemCompletion.kt        — Tab completion for the Terminal box (PATH commands + project files)
    │   ├── CemHistory.kt           — persistent, shared ↑/↓ history (chat and command kept apart)
    │   ├── CemCli.kt               — cem status --json / cem setup / cem fast + a small JSON reader
    │   ├── CemConfig.kt            — legacy YAML reader (fallback when cem cannot be run)
    │   └── CemSettings.kt          — binary path + the setup page (Settings → Tools → cem)
    └── resources/META-INF/plugin.xml
```

## Tool window

Three kinds of tab, all sharing one input widget (`attachInput`: Enter sends,
Shift+Enter adds a line, ↑/↓ walk the history — ↑ on the first line and ↓ on
the last line of a multi-line draft, the caret moves on the lines in between;
an entry recalled from the history and not yet edited always walks on):

The history is **persistent and shared** (`CemHistory`): every chat box draws on
the same list, every Terminal box on another, and both survive an IDE restart —
a per-box list started empty in each newly opened tab, so yesterday's prompt (or
the one sent two tabs ago) could not be recalled. Chat and shell entries stay
apart: `go test ./...` between the prompts would make ↑ useless. Storage is
`<IDE config>/cem/history-{chat,command}.txt`, one escaped line per entry, the
last 200 entries; cem's own `~/.cem/history.log` is *not* read back, because it
truncates the input at 80 characters and a restored prompt would silently be
half a prompt. An entry longer than 8000 characters is not stored at all for the
same reason.

| Tab | Behaviour |
|---|---|
| `Interactive` | Fixed tab. Mode selector on the left, one run per Enter. |
| `Terminal` | Commands in the project root through the shell. `⏹` stops one, `＋` opens another tab (a `go run` serving on :8080 blocks its own tab). Not a pty. **Tab completes** like a shell (`CemCompletion`): the first word against the executables on `PATH`, every other word against files and directories under the project root (`~/` and absolute paths work too). One match is completed — a directory gets `/`, a file gets a space; several matches advance to the common prefix, and when that is no progress the candidates open in a chooser popup above the input (200 at most; ↑/↓ + Enter or a click inserts the pick, typing filters, Esc closes). Hidden files only when a `.` is typed; names with spaces are backslash-escaped. No shell is asked — the tab runs `sh -c` per command, so bash's own completion is never available. |
| run tabs | One per invocation, closeable — closing kills the process. The input box underneath continues the conversation. |

**Continuing a conversation** re-sends the earlier turns as text: cem has no
session, every call is a fresh process. The transcript is capped at 4000
characters because context is billed again on every turn, and what goes into it
is the user's request, not the prompt that was sent — otherwise the context
would double each turn.

## Action flow

1. The user selects text and presses `Ctrl+Alt+I` / `W` / `P` (or `Ctrl+Alt+A`,
   or picks the action from the editor / project-tree context menu).
2. With **no selection** the cursor lands in the tool window's input box with the
   mode preselected. It does *not* send the whole open file: doing that spent
   112 s on a pair run that answered "nothing new to write".
3. A run tab opens and the header is printed before the process starts, so the
   user can see who is talking.
4. A pooled thread runs `ProcessBuilder(cemPath, [-w|-p], prompt)` in
   `project.basePath`, with the child's stdin closed so cem sees EOF.
5. Output is read in chunks (not lines) and appended via `invokeLater`; the
   status bar carries the spinner and elapsed time, so the text pane stays clean.
6. On a non-zero exit the output is checked — first for a missing setup, then for
   an OAuth URL — and the matching notification is shown.

## Settings page

`Settings → Tools → cem` is the setup page: roles, models, reasoning effort, the
binary path, and for HTTP tools (ollama / LM Studio / unsloth) the server
address.

It reads `cem status --json` and writes through `cem setup` / `cem fast` /
`cem endpoint`. The panel used to write `~/.cem/config.yaml` itself, which broke
once setup became mandatory: roles in the YAML are not `setup_done`, so a user
who configured cem from the GUI still could not run it. Reading from cem also
means the tool list, models and effort levels are no longer a second hardcoded
copy — `ollama`, `lmstudio` and `unsloth` appeared in the panel on their own.

`CemCli` parses JSON with a small hand-written reader (`MiniJson`): the plugin
carries no JSON dependency, and IntelliJ's own JSON APIs have moved between the
versions this plugin must support (233 → current).

## Constraints

- **Prompt size.** The prompt is a positional argument; Windows caps a command
  line at ~32 KB. Don't run cem on multi-megabyte selections.
- **The Terminal tab is not a pty.** `vim`, `top` and anything else expecting a
  real terminal belong in the IDE's own terminal.
- **A stale installed jar looks like a code bug.** When a fix "isn't working",
  check the installed jar before the source:
  `ls ~/.local/share/JetBrains/<IDE>/cem-intellij/lib`. Back up an old plugin
  directory *outside* the IDE's plugin path — a backup left inside it caused
  `ClassNotFoundException` on startup.

## Release feed

CI attaches `cem-intellij-<tag>.zip`, a version-less `cem-intellij.zip` and
`updatePlugins.xml` to every GitHub release. The feed's description is generated
from `plugin.xml` by `updateplugins_uret.py`; it used to be a hand-written
one-liner, which showed up in the IDE's plugin screen as a single line next to
the 3200-character Marketplace description.

Marketplace publishing from CI is a separate job (`publish-intellij-plugin`),
dormant until the repository variable `PUBLISH_MARKETPLACE=true` is set. It is
deliberately not part of the plugin build job, which runs with
`continue-on-error: true` — a publish step there would fail silently.
