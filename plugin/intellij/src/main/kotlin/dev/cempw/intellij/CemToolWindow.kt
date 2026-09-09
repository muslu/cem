package dev.cempw.intellij

import com.intellij.openapi.Disposable
import com.intellij.openapi.project.Project
import com.intellij.openapi.wm.ToolWindow
import com.intellij.openapi.wm.ToolWindowFactory
import com.intellij.ui.JBColor
import com.intellij.ui.components.JBScrollPane
import com.intellij.ui.components.JBTextField
import com.intellij.ui.content.Content
import com.intellij.ui.content.ContentFactory
import java.awt.BorderLayout
import java.awt.event.ActionEvent
import javax.swing.AbstractAction
import javax.swing.JLabel
import javax.swing.JPanel
import javax.swing.JTextPane
import javax.swing.KeyStroke
import javax.swing.text.SimpleAttributeSet
import javax.swing.text.StyleConstants

/**
 * cem output tool window. Each invocation (think/write/pair) opens a NEW
 * closable tab. A persistent "Welcome" tab gives usage tips on first open.
 */
class CemToolWindowFactory : ToolWindowFactory {
    override fun createToolWindowContent(project: Project, toolWindow: ToolWindow) {
        // İlk açılış: terminal benzeri etkileşimli sekme. Alttaki input'a yazıp
        // Enter'a basınca cem (think mode) çalışır, çıktı üstte birikir.
        val interactive = CemTab.interactive(project, toolWindow)
        val content = ContentFactory.getInstance()
            .createContent(interactive.component, "Interactive", false)
        content.isCloseable = false   // kapanmaz — kullanıcı her seferinde input'a yazar
        content.isPinned = true
        toolWindow.contentManager.addContent(content)
        toolWindow.contentManager.setSelectedContent(content)
    }
}

/**
 * Tek bir cem çıktısının container'ı. Action'lar her seferinde yeni tab
 * ister: CemTab.newRun(project, mode, snippet) → ContentFactory ile content
 * oluşturup ToolWindow'a addContent.
 */
class CemTab {
    val component: JPanel
    /**
     * Uzun satırlar sağda kesilmesin: JTextPane ancak viewport genişliğini
     * takip ettiğinde word-wrap yapar. Yatay scrollbar da kapatılır, yoksa
     * pane kendini genişletip satırı yine kesiyor.
     */
    private val textPane = object : JTextPane() {
        override fun getScrollableTracksViewportWidth(): Boolean = true
    }.apply {
        isEditable = false
        background = JBColor.background()
    }
    /** Alt status bar — dönen spinner + geçen süre. Text pane spam'i yok. */
    val statusLabel = JLabel(" ")
    /** Çalışan subprocess. Tab kapanırsa öldürülür. */
    @Volatile var process: Process? = null
    /** Tab kapatıldı mı (idempotency için). */
    @Volatile var cancelled = false

    init {
        component = JPanel(BorderLayout()).apply {
            val scroll = JBScrollPane(textPane).apply {
                horizontalScrollBarPolicy =
                    javax.swing.ScrollPaneConstants.HORIZONTAL_SCROLLBAR_NEVER
            }
            add(scroll, BorderLayout.CENTER)
            add(statusLabel, BorderLayout.SOUTH)
        }
    }

    /** Status bar metnini (EDT-safe) güncelle. */
    fun setStatus(text: String) {
        com.intellij.openapi.application.ApplicationManager.getApplication()
            .invokeLater { statusLabel.text = " $text" }
    }

    /** Tab kapatıldığında çağırılır: subprocess'i öldür. */
    fun cancel() {
        if (cancelled) return
        cancelled = true
        process?.let { p ->
            if (p.isAlive) {
                p.destroy()
                // Grace period — sonra zorla
                Thread {
                    try { Thread.sleep(1500) } catch (_: InterruptedException) {}
                    if (p.isAlive) p.destroyForcibly()
                }.start()
            }
        }
    }

    fun appendHeader(mode: String, snippet: String) {
        appendStyled(
            "─── cem $mode · ${promptSnippet(snippet, 80)} ───\n",
            bold = true, color = JBColor.GRAY,
        )
    }

    fun appendLine(s: String) {
        appendStyled(s + "\n", bold = false, color = JBColor.foreground())
    }

    fun appendDim(s: String) {
        appendStyled(s + "\n", bold = false, color = JBColor.GRAY)
    }

    fun appendError(s: String) {
        appendStyled(s + "\n", bold = true, color = JBColor.RED)
    }

    fun appendRaw(s: String) {
        appendStyled(s, bold = false, color = JBColor.foreground())
    }

    private fun appendStyled(text: String, bold: Boolean, color: java.awt.Color) {
        val doc = textPane.styledDocument
        val attrs = SimpleAttributeSet().apply {
            StyleConstants.setBold(this, bold)
            StyleConstants.setForeground(this, color)
            StyleConstants.setFontFamily(this, "Monospaced")
        }
        doc.insertString(doc.length, text, attrs)
        textPane.caretPosition = doc.length
    }

    companion object {
        /**
         * Sekme adı / başlık için prompt özeti.
         *
         * Ham prompt olduğu gibi başlığa yazılınca markdown ve emoji başlığı
         * okunmaz yapıyordu (README gönderildiğinde sekme adı
         * "pair · # CEM ```" + kutucuklar oluyordu). Burada satırlar tek
         * satıra indirilir, markdown işaretleri ve sembol/emoji kod noktaları
         * atılır, boşluklar sadeleşir.
         */
        fun promptSnippet(raw: String, max: Int): String {
            val sb = StringBuilder()
            for (cp in raw.codePoints().toArray()) {
                when {
                    cp < 0x20 -> sb.append(' ')            // kontrol karakterleri
                    isDecorative(cp) -> sb.append(' ')     // emoji / sembol / ok
                    else -> sb.appendCodePoint(cp)
                }
            }
            val clean = sb.toString()
                .replace(markdownNoise, " ")
                .replace(whitespaceRun, " ")
                .trim()
            if (clean.isEmpty()) return "prompt"
            return if (clean.length <= max) clean else clean.take(max - 1).trim() + "…"
        }

        /** Başlıkta yeri olmayan süs karakterleri (emoji, ok, dingbat, VS16). */
        private fun isDecorative(cp: Int): Boolean =
            cp in 0x2190..0x2BFF ||        // oklar, çeşitli semboller, dingbat
            cp in 0xFE00..0xFE0F ||        // variation selector
            cp == 0x200D || cp == 0x200B || cp == 0xFEFF ||
            cp in 0x1F000..0x1FAFF         // emoji blokları

        private val markdownNoise = Regex("""[`*_#>~|\[\]]""")
        private val whitespaceRun = Regex("""\s+""")

        /** Welcome tab (legacy — şimdi interactive ile değiştirildi). */
        fun welcome(): CemTab {
            val tab = CemTab()
            tab.appendStyled(welcomeText(), bold = false, color = JBColor.GRAY)
            return tab
        }

        private fun welcomeText(): String = """
            ⚡ cem — Compose · Execute · Multiplex
            One command, many AIs.

            Aşağıdaki input'a sorunu yaz, Enter'a bas → cem -p "..." (pair)
            Düşünen plan yapar, yazan kodu yazar; soru kod istemiyorsa
            yazan otomatik atlanır — sekme adı yine "pair" görünür.

            Editör shortcut'ları:
              Ctrl+Alt+I  →  cem: think on selection
              Ctrl+Alt+W  →  cem: write on selection
              Ctrl+Alt+P  →  cem: pair on selection (thinker → writer)
              Ctrl+Alt+A  →  cem: ask freely (custom prompt)

            Tip: editör seçimi olmadan kısayol → input dialog açılır.
            Settings → Tools → cem ile thinker/writer/model değiştirilir.

            ─── geçmiş ───

        """.trimIndent()

        /**
         * Etkileşimli "Interactive" tab — minimal REPL:
         *  ┌──────────────────────────────┐
         *  │ Welcome + history (scroll)   │
         *  ├──────────────────────────────┤
         *  │ cem ›  [input...]          ⏎ │
         *  ├──────────────────────────────┤
         *  │ status bar (spinner + süre)  │
         *  └──────────────────────────────┘
         *
         * Tek input, Enter → cem -p (pair mode). cem'in kendi skip mantığı:
         *   - Soru kod istemiyorsa writer otomatik atlanır → sadece thinker
         *   - Kod istiyorsa thinker → writer zinciri çalışır
         * Bu sayede kullanıcı mode seçmek zorunda değil.
         */
        fun interactive(project: Project, toolWindow: ToolWindow): CemTab {
            val tab = CemTab()
            tab.appendStyled(welcomeText(), bold = false, color = JBColor.GRAY)

            val input = JBTextField()
            input.toolTipText = "Sorunu yaz, Enter ↵   ·   ↑/↓ önceki/sonraki prompt"

            val inputPanel = JPanel(BorderLayout()).apply {
                add(JLabel(" cem ›  "), BorderLayout.WEST)
                add(input, BorderLayout.CENTER)
            }
            // statusLabel zaten SOUTH; input'u onun ÜZERİNE koy (south wrap içinde)
            tab.component.remove(tab.statusLabel)
            val southWrap = JPanel(BorderLayout()).apply {
                add(inputPanel, BorderLayout.NORTH)
                add(tab.statusLabel, BorderLayout.SOUTH)
            }
            tab.component.add(southWrap, BorderLayout.SOUTH)

            // Shell-style prompt history. Bash gibi: Enter → ekle; ↑/↓ gez.
            // historyIndex = history.size → "henüz tarihçeye dönmedi", input düzenleniyor.
            val history = mutableListOf<String>()
            var historyIndex = 0
            var draft = ""

            input.actionMap.put("submit", object : AbstractAction() {
                override fun actionPerformed(e: ActionEvent) {
                    val prompt = input.text.trim()
                    if (prompt.isEmpty()) return
                    input.text = ""
                    // Duplikatları arka arkaya eklemeyiz (bash HIST_IGNORE_DUPS).
                    if (history.isEmpty() || history.last() != prompt) history.add(prompt)
                    historyIndex = history.size
                    draft = ""
                    tab.appendDim("→ yeni tab: $prompt")
                    CemAction.launchCem(project, CemAction.Mode.PAIR, prompt)
                }
            })
            input.inputMap.put(KeyStroke.getKeyStroke("ENTER"), "submit")

            input.actionMap.put("history.prev", object : AbstractAction() {
                override fun actionPerformed(e: ActionEvent) {
                    if (history.isEmpty()) return
                    // İlk ↑ basışında mevcut taslağı kaydet
                    if (historyIndex == history.size) draft = input.text
                    if (historyIndex > 0) {
                        historyIndex--
                        input.text = history[historyIndex]
                        input.caretPosition = input.text.length
                    }
                }
            })
            input.inputMap.put(KeyStroke.getKeyStroke("UP"), "history.prev")

            input.actionMap.put("history.next", object : AbstractAction() {
                override fun actionPerformed(e: ActionEvent) {
                    if (historyIndex >= history.size) return
                    historyIndex++
                    input.text = if (historyIndex == history.size) draft else history[historyIndex]
                    input.caretPosition = input.text.length
                }
            })
            input.inputMap.put(KeyStroke.getKeyStroke("DOWN"), "history.next")

            return tab
        }

        /**
         * Interactive tab için legacy — yeni tab AÇMADAN, mevcut tab'a stream et.
         * Yeni davranış: Interactive submit handler artık CemAction.launchCem'i
         * çağırıyor (yeni tab açar). Bu fonksiyon ileride farklı use-case için
         * tutuluyor (örn. test, manuel inline run).
         */
        @Suppress("unused")
        private fun launchInline(project: Project, mode: String, prompt: String, tab: CemTab) {
            com.intellij.openapi.application.ApplicationManager.getApplication()
                .executeOnPooledThread {
                try {
                    val cemPath = CemAction.resolveCemBinary()
                    val args = mutableListOf(cemPath)
                    when (mode) {
                        "write" -> args.add("-w")
                        "pair"  -> args.add("-p")
                        // think: no flag
                    }
                    args.add(prompt)
                    val pb = ProcessBuilder(args).redirectErrorStream(true)
                    project.basePath?.let { pb.directory(java.io.File(it)) }
                    val startTime = System.currentTimeMillis()
                    val process = pb.start()
                    try { process.outputStream.close() } catch (_: Exception) {}
                    tab.process = process

                    // Status spinner — interactive tab da spinner kullansın
                    val frames = listOf("⠋","⠙","⠹","⠸","⠼","⠴","⠦","⠧","⠇","⠏")
                    Thread {
                        var i = 0
                        while (!tab.cancelled && process.isAlive) {
                            val secs = (System.currentTimeMillis() - startTime) / 1000
                            tab.setStatus("${frames[i++ % frames.size]}  ${secs}s · running")
                            try { Thread.sleep(100) } catch (_: InterruptedException) { break }
                        }
                    }.apply { isDaemon = true }.start()
                    val reader = java.io.InputStreamReader(process.inputStream, Charsets.UTF_8)
                    val buf = CharArray(2048)
                    while (!tab.cancelled) {
                        val n = reader.read(buf)
                        if (n < 0) break
                        val chunk = String(buf, 0, n)
                        com.intellij.openapi.application.ApplicationManager.getApplication()
                            .invokeLater { tab.appendRaw(chunk) }
                    }
                    val exit = process.waitFor()
                    com.intellij.openapi.application.ApplicationManager.getApplication()
                        .invokeLater {
                            tab.appendDim(if (exit == 0) "─── done ───\n" else "─── exit $exit ───\n")
                        }
                } catch (e: Exception) {
                    com.intellij.openapi.application.ApplicationManager.getApplication()
                        .invokeLater { tab.appendError("cem hata: ${e.message}\n") }
                }
            }
        }

        /**
         * Yeni bir run-tab ekler, ToolWindow'u açıp seçili yapar.
         * Çağıran sonra tab.appendXxx ile çıktı yazar.
         *
         * Content disposer'ı tab.cancel()'a bağlanır → kullanıcı sekmeyi
         * X ile kapattığında subprocess öldürülür.
         */
        fun newRun(toolWindow: ToolWindow, mode: String, snippet: String): CemTab {
            val tab = CemTab()
            tab.appendHeader(mode, snippet)
            val title = "$mode · ${promptSnippet(snippet, 30)}"
            val content: Content = ContentFactory.getInstance()
                .createContent(tab.component, title, true)
            content.isCloseable = true
            content.isPinned = false
            content.setDisposer(Disposable { tab.cancel() })
            toolWindow.contentManager.addContent(content)
            toolWindow.contentManager.setSelectedContent(content)
            toolWindow.show()
            return tab
        }
    }
}
