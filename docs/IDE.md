# cem in your IDE

Recipes for invoking `cem` from PyCharm, IntelliJ family, VS Code, and Antigravity IDE — without writing any plugins.

Turkish version: [IDE.tr.md](IDE.tr.md)

---

## PyCharm / IntelliJ IDEA / GoLand / WebStorm

### Option 1 — Built-in terminal (zero setup)

Open the **Terminal** tab at the bottom (`Alt+F12`) and run:

```sh
cem "explain this file"
cem -w "write tests for the selected function"
cem -p "implement the TODO at line 42"
```

The working directory is your project root automatically, so `.cem.yaml` and API key rotation just work.

### Option 2 — External Tools (menu entry + keyboard shortcut)

**Settings → Tools → External Tools → `+`** and add three entries:

| Name | Program | Arguments | Working dir | Output filter |
|---|---|---|---|---|
| `cem-think` | `cem` | `"$SelectedText$"` | `$ProjectFileDir$` | leave blank |
| `cem-write` | `cem` | `-w "$SelectedText$"` | `$ProjectFileDir$` | leave blank |
| `cem-pair` | `cem` | `-p "$SelectedText$"` | `$ProjectFileDir$` | leave blank |

Then **Settings → Keymap → External Tools → cem-think** → right-click → **Add Keyboard Shortcut**. Recommended bindings:

| Action | Shortcut |
|---|---|
| `cem-think` | `Ctrl+Alt+I` (macOS: `⌥⌘I`) |
| `cem-write` | `Ctrl+Alt+W` |
| `cem-pair` | `Ctrl+Alt+P` |

**Usage:** select a code block, press the shortcut. cem output appears in the bottom **Run** panel.

If `$SelectedText$` is empty (no selection), cem reads stdin — combine with a prompt dialog by switching to `Prompt for arguments`:

| Argument macro | Effect |
|---|---|
| `"$Prompt$"` | Pops up an input box every time |
| `"$SelectedText$"` | Uses current editor selection |
| `"$FileText$"` | Sends the entire current file |
| `"$SelectedText$" --in $FilePath$` | Selection plus filename context |

### Option 3 — Run Configuration (debugger-style)

**Run → Edit Configurations → `+` → Shell Script**:

```
Name:               cem pair
Script path:        /absolute/path/to/cem    (or use `cem` if it's on PATH)
Script options:     -p
Execute in tty:     ✓
Working directory:  $ProjectFileDir$
```

Leave the script body empty — the IDE will pass `-p` to cem and cem will read your prompt from stdin (or you can edit the configuration to pass a fixed prompt).

This gives you a `▶` button in the toolbar that you can save per-project.

### Option 4 — File Watcher (auto-run on save)

**Settings → Tools → File Watchers → `+` → custom**:

```
Name:         cem review on save
File type:    Python (or whatever)
Scope:        Open Files
Program:      cem
Arguments:    -p "review this file for bugs"
Working dir:  $ProjectFileDir$
Output paths: leave blank
```

Toggle the watcher off when you don't want it — otherwise every save will burn tokens.

---

## VS Code

### Option 1 — Built-in terminal

`` Ctrl+` `` to open, then `cem "..."`.

### Option 2 — Tasks (`.vscode/tasks.json`)

```jsonc
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "cem: think on selection",
      "type": "shell",
      "command": "cem",
      "args": ["\"${selectedText}\""],
      "presentation": { "reveal": "always", "panel": "shared" }
    },
    {
      "label": "cem: pair on selection",
      "type": "shell",
      "command": "cem",
      "args": ["-p", "\"${selectedText}\""],
      "presentation": { "reveal": "always", "panel": "shared" }
    }
  ]
}
```

Bind with `.vscode/keybindings.json`:

```json
[
  { "key": "ctrl+alt+i", "command": "workbench.action.tasks.runTask", "args": "cem: think on selection" },
  { "key": "ctrl+alt+p", "command": "workbench.action.tasks.runTask", "args": "cem: pair on selection" }
]
```

### Option 3 — Code runner extension

Install the **Code Runner** extension, then add to `settings.json`:

```jsonc
"code-runner.customCommand": "cem -p \"$selectedText\""
```

Now `Ctrl+Alt+J` runs cem on the selected text.

---

## Antigravity IDE

Antigravity is itself an AI-agent IDE, so you have two integration patterns:

### Option 1 — Built-in terminal

Antigravity exposes a terminal panel. Run cem there just like any other shell command:

```sh
cem -p "refactor the buildArgs function"
```

### Option 2 — Ask the Antigravity agent to invoke cem

Tell the Antigravity agent in a conversation:

> Run `cem -p "<task>"` and review the output.

The agent treats `cem` as a sub-tool — it executes, captures the output, and feeds it into its own reasoning. This is genuinely useful when you want **two different models cross-checking each other** (Antigravity's Gemini + Claude via cem).

### Option 3 — Custom slash command (research)

Antigravity supports slash commands (`/goal`, `/schedule`, etc). Whether user-defined slash commands are possible needs verification against current Antigravity docs. If supported, a `/cem` command can be configured to invoke cem with the current conversation context. See [docs/ANTIGRAVITY.md](ANTIGRAVITY.md) for findings.

---

## Vim / Neovim

```vim
" In your .vimrc or init.vim
function! CemThink()
  let l:selection = @v
  silent execute "!" . "cem \"" . l:selection . "\""
endfunction

vnoremap <leader>ct y:call CemThink()<CR>
vnoremap <leader>cp y:!cem -p "<C-r>""<CR>
```

For Neovim with floating windows, see `:help nvim_open_term()` to run cem inside a popup.

---

## Emacs

```elisp
(defun cem-think-region (start end)
  "Send region to cem and show output in a buffer."
  (interactive "r")
  (let ((prompt (buffer-substring-no-properties start end)))
    (shell-command (concat "cem \"" prompt "\"") "*cem*")))

(global-set-key (kbd "C-c c t") 'cem-think-region)
```

---

## Plugin development (for native integration)

The recipes above are zero-effort but limited (no streaming, no inline diff, no diagnostics integration). Native plugins are tracked in:

- [docs/PYCHARM-PLUGIN.md](PYCHARM-PLUGIN.md) — IntelliJ Platform plugin: internals and development notes
- [docs/ANTIGRAVITY.md](ANTIGRAVITY.md) — research into Antigravity's plugin model (C)

If you'd rather see streaming output, inline edit suggestions, or a dedicated tool window, see those documents.
