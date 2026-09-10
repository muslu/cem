# cem'i IDE içinde kullan

PyCharm, IntelliJ ailesi, VS Code ve Antigravity IDE için sıfır plugin gerektiren entegrasyon yöntemleri.

English: [IDE.md](IDE.md)

---

## PyCharm / IntelliJ IDEA / GoLand / WebStorm

### Yol 1 — Dahili terminal (sıfır kurulum)

Alttaki **Terminal** sekmesini aç (`Alt+F12`) ve çalıştır:

```sh
cem "bu dosyayı açıkla"
cem -w "seçili fonksiyon için test yaz"
cem -p "satır 42'deki TODO'yu yap"
```

Working directory otomatik proje kökü → `.cem.yaml` ve API key rotasyonu sorunsuz çalışır.

### Yol 2 — External Tools (menü + kısayol)

**Settings → Tools → External Tools → `+`** ile üç giriş ekle:

| Ad | Program | Argumanlar | Working dir |
|---|---|---|---|
| `cem-think` | `cem` | `"$SelectedText$"` | `$ProjectFileDir$` |
| `cem-write` | `cem` | `-w "$SelectedText$"` | `$ProjectFileDir$` |
| `cem-pair` | `cem` | `-p "$SelectedText$"` | `$ProjectFileDir$` |

Sonra **Settings → Keymap → External Tools → cem-think** → sağ tık → **Add Keyboard Shortcut**:

| Aksiyon | Kısayol |
|---|---|
| `cem-think` | `Ctrl+Alt+I` (macOS: `⌥⌘I`) |
| `cem-write` | `Ctrl+Alt+W` |
| `cem-pair` | `Ctrl+Alt+P` |

**Kullanım:** bir kod bloğunu seç, kısayola bas. cem çıktısı alttaki **Run** panelinde belirir.

`$SelectedText$` boşsa cem stdin'i okur. Diyalog kutusu istiyorsan:

| Macro | Etkisi |
|---|---|
| `"$Prompt$"` | Her seferinde input kutusu açar |
| `"$SelectedText$"` | Mevcut editör seçimi |
| `"$FileText$"` | Tüm dosya içeriği |
| `"$SelectedText$" --in $FilePath$` | Seçim + dosya bağlamı |

### Yol 3 — Run Configuration (debug tarzı)

**Run → Edit Configurations → `+` → Shell Script**:

```
İsim:           cem pair
Script path:    /tam/yol/cem    (veya PATH'te ise sadece `cem`)
Script seçenekleri: -p
Execute in tty: ✓
Working dir:    $ProjectFileDir$
```

Script gövdesi boş — IDE `-p` flag'ini geçer, cem stdin'den prompt'u okur.

Sonuç: toolbar'da bir `▶` butonu, proje başına saklanır.

### Yol 4 — File Watcher (kayıtta otomatik çalıştır)

**Settings → Tools → File Watchers → `+` → custom**:

```
İsim:           save'de cem review
Dosya tipi:     Python (vs.)
Scope:          Open Files
Program:        cem
Argumanlar:     -p "bu dosyayı bug açısından gözden geçir"
Working dir:    $ProjectFileDir$
```

İstemediğin zaman watcher'ı kapat — yoksa her save token harcar.

---

## VS Code

### Yol 1 — Terminal

`` Ctrl+` `` aç, sonra `cem "..."`.

### Yol 2 — Tasks (`.vscode/tasks.json`)

```jsonc
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "cem: seçimi düşün",
      "type": "shell",
      "command": "cem",
      "args": ["\"${selectedText}\""],
      "presentation": { "reveal": "always", "panel": "shared" }
    },
    {
      "label": "cem: seçimde pair",
      "type": "shell",
      "command": "cem",
      "args": ["-p", "\"${selectedText}\""],
      "presentation": { "reveal": "always", "panel": "shared" }
    }
  ]
}
```

Kısayol için `.vscode/keybindings.json`:

```json
[
  { "key": "ctrl+alt+i", "command": "workbench.action.tasks.runTask", "args": "cem: seçimi düşün" },
  { "key": "ctrl+alt+p", "command": "workbench.action.tasks.runTask", "args": "cem: seçimde pair" }
]
```

### Yol 3 — Code Runner extension

**Code Runner** eklentisini kur, `settings.json`'a ekle:

```jsonc
"code-runner.customCommand": "cem -p \"$selectedText\""
```

Artık `Ctrl+Alt+J` seçili metni cem'e gönderir.

---

## Antigravity IDE

Antigravity zaten bir AI-agent IDE'si, iki entegrasyon kalıbın var:

### Yol 1 — Dahili terminal

Antigravity terminal panelinden cem'i direkt çalıştır:

```sh
cem -p "buildArgs fonksiyonunu refactor et"
```

### Yol 2 — Antigravity agent'a cem çağırt

Antigravity agent'ıyla konuşmada şöyle de:

> `cem -p "<görev>"` çalıştır ve çıktısını incele.

Agent cem'i alt-tool olarak çalıştırır, çıktıyı yakalar, kendi akıl yürütmesine besler. **İki farklı modelin birbirini cross-check etmesi** istediğinde gerçekten faydalı (Antigravity'nin Gemini'si + cem üzerinden Claude).

### Yol 3 — Custom slash command (araştırma)

Antigravity'nin `/goal`, `/schedule` gibi slash komutları var. Kullanıcı-tanımlı slash komut destekleyip desteklemediğini şu an doğrulamadık. Destekliyorsa `/cem` ile mevcut konuşma bağlamını cem'e geçirebiliriz. Bulgular için [docs/ANTIGRAVITY.md](ANTIGRAVITY.md).

---

## Vim / Neovim

```vim
function! CemThink()
  let l:selection = @v
  silent execute "!" . "cem \"" . l:selection . "\""
endfunction

vnoremap <leader>ct y:call CemThink()<CR>
vnoremap <leader>cp y:!cem -p "<C-r>""<CR>
```

Neovim'de floating window için: `:help nvim_open_term()`.

---

## Emacs

```elisp
(defun cem-think-region (start end)
  "Bölgeyi cem'e gönder, çıktıyı buffer'a aç."
  (interactive "r")
  (let ((prompt (buffer-substring-no-properties start end)))
    (shell-command (concat "cem \"" prompt "\"") "*cem*")))

(global-set-key (kbd "C-c c t") 'cem-think-region)
```

---

## Plugin geliştirme (native entegrasyon için)

Yukarıdaki tarifler sıfır eforlu ama sınırlı (streaming yok, inline diff yok, diagnostics entegrasyonu yok). Native plugin'ler:

- [docs/PYCHARM-PLUGIN.md](PYCHARM-PLUGIN.md) — IntelliJ Platform eklentisi: iç işleyiş ve geliştirme notları
- [docs/ANTIGRAVITY.md](ANTIGRAVITY.md) — Antigravity plugin modeli araştırması (C)

Streaming çıktı, inline kod önerisi veya özel tool window istiyorsan bu dosyalara bak.
