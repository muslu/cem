# cem in JetBrains IDEs

Works in **PyCharm, IntelliJ IDEA, GoLand, WebStorm, RubyMine, PhpStorm, Rider, DataGrip, CLion, AppCode, RustRover, AquaIDE** — anything built on the IntelliJ Platform 2023.3+ (build 233+).

Turkish: [INTELLIJ.tr.md](INTELLIJ.tr.md)

---

## Install

1. Make sure `cem` is on your PATH (`curl -fsSL cem.pw/install | sh` or PowerShell equivalent — see [README](../README.md)).
2. IDE → **Settings → Plugins → Marketplace** → search **cem** → *Install*
   ([listing](https://plugins.jetbrains.com/plugin/34196)).
3. Restart the IDE.

**From disk instead** (a build that is not on the Marketplace yet, or an
air-gapped machine):

```
https://github.com/muslu/cem/releases/latest/download/cem-intellij-<version>.zip
```

IDE → **Settings → Plugins → ⚙ (gear icon) → Install Plugin from Disk** → pick
the zip. Note that a plugin installed from disk is never updated on its own.

You'll now see **Tools → cem** with three actions, and a `cem` submenu in the editor right-click menu.

## Automatic updates

Installed from the Marketplace, the IDE updates the plugin itself — nothing to
configure.

For a **disk install**, updates never arrive on their own. Either reinstall
from the Marketplace, or point the IDE at the release feed once:

**Settings → Plugins → ⚙ → Manage Plugin Repositories → `+`**

```
https://github.com/muslu/cem/releases/latest/download/updatePlugins.xml
```

The IDE then polls that address; when a new version ships it shows up under
**Plugins → Updates** and installs itself if automatic updates are on. The URL
is stable — its contents are refreshed on every cem release.

## Usage

| Action | Default shortcut | Effect |
|---|---|---|
| `cem: think on selection` | `Ctrl+Alt+I` (`⌥⌘I` macOS) | Runs `cem "<selection>"` (thinker only) |
| `cem: write on selection` | `Ctrl+Alt+W` (`⌥⌘W`) | Runs `cem -w "<selection>"` (writer only) |
| `cem: pair on selection`  | `Ctrl+Alt+P` (`⌥⌘P`) | Runs `cem -p "<selection>"` (thinker → writer) |

### Keep talking in a run tab

Every run opens its own tab, and each tab has an input box underneath it. Type
there and the same tab starts another turn with the previous ones carried along
as context — that is how you answer a question the tool asked ("do you want to
revert all of them?") or add to what it just did.

The context is trimmed to the last few thousand characters. cem has no session:
each call is a fresh process, so continuing means re-sending the earlier turns,
and every turn is billed again. Trimming keeps that from growing silently.

### Terminal tab

The `Terminal` tab runs commands in the project root — `go test ./...`,
`git diff`, `npm run build` — so the answer and the command output stay in one
window. It goes through the shell, so pipes, redirects and `&&` work. It is not
a full pty: use the IDE's own terminal for `vim`, `top` or anything that expects
a real terminal. `⏹` stops a running command, and `＋` opens another terminal
tab — a `go run` left serving on :8080 would otherwise block everything else.
The first tab stays open; the extra ones can be closed, and closing one kills
whatever it was running.

### Selection and shortcuts

If nothing is selected, the cursor lands in the input box at the bottom of the
**cem** tool window with the mode preselected — no modal dialog, and the open
file is not sent by mistake. Output streams to that same tool window, one tab
per run.

Working directory = project root, so `.cem.yaml` (project-local config) and API keys from `~/.cem/config.yaml` work as expected.

## Settings

**Settings → Tools → cem** now also manages (all written to
`~/.cem/config.yaml`, the same file the terminal `cem` uses):

| Field | What it does |
|---|---|
| Thinker / Writer | which AI thinks, which one writes |
| Model | per-role model (blank = the CLI's own default) |
| **Reasoning effort** | suggested: thinker `high`, writer `low` |
| **Fast mode** | runs the tool without loading its hooks/permission rules — measured 8s instead of 124s. Turn it off to keep your own hooks. |

## Settings

**Settings → Tools → cem**:

| Field | Default | Notes |
|---|---|---|
| `cem binary path` | `cem` | Absolute path required if the binary isn't on the IDE's PATH (common when the IDE was launched from a non-shell environment on macOS/Linux). |

## Customising shortcuts

**Settings → Keymap → Plug-ins → cem** → right-click an action → **Add Keyboard Shortcut**. The defaults conflict with nothing in the standard JetBrains keymap (as of 2026.1).

## Troubleshooting

### `0x80004002` "Interface not supported" (Windows)

Plugin compiled against a newer IntelliJ Platform than your IDE supports. Update the IDE (Help → Check for Updates) or, if you can't, open an issue with your build number (Help → About → Build #).

### Tool window shows nothing

1. Is `cem` actually on PATH? Run `where cem` (Windows) / `which cem` (Unix) in the IDE terminal.
2. If `cem` is somewhere unusual (e.g., `C:\Users\Foo\AppData\Local\cem\bin\cem.exe`), set the absolute path in **Settings → Tools → cem**.
3. Check the **cem** tool window for stderr lines.

### Action appears greyed out

Make sure the editor (code area) has focus. The actions are bound to `editorTextFocus`.

### `idea.log` errors

**Help → Show Log in Explorer / Finder** → search for `dev.cempw.cem`. Open a GitHub issue with the stack trace.

## Build from source

```sh
cd plugin/intellij
./gradlew buildPlugin
# build/distributions/cem-<version>.zip → Install Plugin from Disk
```

Requires JDK 21.

See also: [docs/PYCHARM-PLUGIN.md](PYCHARM-PLUGIN.md) for the plugin's internal architecture and roadmap.
