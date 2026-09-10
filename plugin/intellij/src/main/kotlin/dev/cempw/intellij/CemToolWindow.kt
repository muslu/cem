package dev.cempw.intellij

import com.intellij.openapi.Disposable
import com.intellij.openapi.project.Project
import com.intellij.openapi.wm.ToolWindow
import com.intellij.openapi.wm.ToolWindowFactory
import com.intellij.openapi.ui.ComboBox
import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.application.ModalityState
import com.intellij.openapi.util.SystemInfo
import com.intellij.openapi.wm.IdeFocusManager
import com.intellij.openapi.wm.ToolWindowManager
import com.intellij.ui.JBColor
import com.intellij.ui.SimpleListCellRenderer
import com.intellij.ui.components.JBScrollPane
import com.intellij.ui.content.Content
import com.intellij.ui.content.ContentFactory
import java.awt.BorderLayout
import java.awt.event.ActionEvent
import java.util.WeakHashMap
import javax.swing.AbstractAction
import javax.swing.JButton
import javax.swing.JComponent
import javax.swing.JLabel
import javax.swing.JPanel
import javax.swing.JTextArea
import javax.swing.JTextPane
import javax.swing.KeyStroke
import javax.swing.ScrollPaneConstants
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
        // Komut çalıştırmak için ikinci sabit sekme: yazan rol dosya üretince
        // testi/derlemeyi aynı pencerede koşturmak için.
        CemTab.commandTab(project, toolWindow)
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
    /** Interactive sekmesinin girdi alanı — yalnız o sekmede dolu. */
    private var inputArea: JTextArea? = null
    /** Interactive sekmesinin mod seçici (pair / think / write). */
    private var modeBox: ComboBox<CemAction.Mode>? = null
    /**
     * Girdinin ALTINA iliştirilecek bağlam (ör. dosya içeriği). Kullanıcı
     * talimatını yazıp Enter'a basınca prompt'a eklenir ve TEMİZLENİR —
     * yoksa bir sonraki soruya da yapışır ve iki kere faturalanır.
     */
    @Volatile private var pendingContext: String? = null
    /** Tab kapatıldı mı (idempotency için). */
    @Volatile var cancelled = false
    /**
     * Bu sekmedeki konuşma dökümü — "sohbete devam" bunu bağlam olarak
     * gönderiyor. cem her çağrıda yeni bir süreç: aracın kendi oturumu yok,
     * bu yüzden devam eden istekte önceki tur METİN olarak taşınmak zorunda.
     *
     * Sınırlı tutuluyor: bağlam her turda yeniden faturalanıyor. Son
     * transcriptLimit karakter yeterli — asıl soru ve son cevap orada.
     */
    private val transcript = StringBuilder()
    /** Çalıştırma sekmesindeki takip girdisi (sohbete devam). */
    private var followUp: JTextArea? = null

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

    /**
     * Girdi kutusunu bir istek için hazırla: modu seç, bağlamı iliştir,
     * imleci içine al. Kısayollar artık modal dialog yerine burayı kullanıyor.
     */
    fun prepareInput(mode: CemAction.Mode, context: String?, hint: String?) {
        val area = inputArea ?: return
        modeBox?.selectedItem = mode
        pendingContext = context
        hint?.let { appendDim(it) }
        IdeFocusManager.getGlobalInstance().doWhenFocusSettlesDown({
            area.requestFocusInWindow()
            area.caretPosition = area.document.length
        }, ModalityState.defaultModalityState())
    }

    /** Bekleyen bağlamı alıp temizler (tek kullanımlık). */
    private fun takePendingContext(): String? {
        val c = pendingContext
        pendingContext = null
        return c
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

    /** Bir turu (istek + cevap) döküme ekler; baştan kırpar. */
    fun noteExchange(request: String, answer: String) {
        transcript.append("KULLANICI: ").append(request.trim()).append("\n")
            .append("CEM: ").append(answer.trim()).append("\n\n")
        if (transcript.length > transcriptLimit) {
            transcript.delete(0, transcript.length - transcriptLimit)
        }
    }

    /**
     * Devam isteğini önceki turların bağlamıyla sarar.
     *
     * Neden düz metin: cem'in (ve altındaki araçların) oturum kavramı yok;
     * "önceki cevabına göre şunu yap" demek için önceki cevabı yeniden
     * göndermek gerekiyor. Kırpma sondan yapılır — kullanıcının son sorusu ve
     * aracın son cevabı, ilk turdan daha önemli.
     */
    fun followUpPrompt(message: String): String {
        if (transcript.isEmpty()) return message
        return buildString {
            append("Bu bir devam isteği. Önceki konuşma (kısaltılmış):\n")
            append("---\n").append(transcript).append("---\n\n")
            append("Yeni istek: ").append(message)
        }
    }

    /** Takip girdisini kilitle/aç — süreç çalışırken yeni istek gönderilmesin. */
    fun setInputEnabled(enabled: Boolean) {
        val area = followUp ?: return
        com.intellij.openapi.application.ApplicationManager.getApplication().invokeLater {
            area.isEnabled = enabled
            if (enabled) area.requestFocusInWindow()
        }
    }

    /**
     * Çalıştırma sekmesine "sohbete devam" kutusu ekler: cevabın altına yazıp
     * Enter'a basınca aynı sekmede yeni tur başlar ve önceki turlar bağlam
     * olarak gider. Araç soru sorduğunda (ör. "tümünü geri almak mı
     * istiyorsunuz?") cevap verebilmenin tek yolu buydu — eskiden her istek
     * tek seferlikti, kullanıcı cevabı yazacak yer bulamıyordu.
     */
    fun enableFollowUp(project: Project, mode: CemAction.Mode) {
        followUp = attachInput(
            this,
            west = null,
            tooltip = "Enter ↵ devam et · Shift+Enter satır atla · ↑/↓ önceki istekler",
        ) { typed ->
            appendDim("→ ${mode.name.lowercase()}: ${promptSnippet(typed, 80)}")
            setInputEnabled(false)
            CemAction.continueInTab(project, this, mode, typed)
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
         * Devam bağlamının üst sınırı (karakter). Bağlam her turda yeniden
         * gönderiliyor, yani her tur yeniden faturalanıyor: sınırsız döküm
         * uzun bir sohbette maliyeti sessizce katlar.
         */
        private const val transcriptLimit = 4000

        /**
         * Alt girdi kutusu + shell tarzı geçmiş. Üç yerde kullanılıyor:
         * Interactive sekmesi, çalıştırma sekmesinde "sohbete devam" ve komut
         * sekmesi. Enter gönderir, Shift+Enter satır atlar, ↑/↓ geçmişi gezer.
         */
        private fun attachInput(
            tab: CemTab,
            west: JComponent?,
            tooltip: String,
            onSubmit: (String) -> Unit,
        ): JTextArea {
            val input = JTextArea(2, 20).apply {
                lineWrap = true
                wrapStyleWord = true
                toolTipText = tooltip
            }
            val inputPanel = JPanel(BorderLayout()).apply {
                if (west != null) {
                    add(JPanel(BorderLayout()).apply {
                        add(west, BorderLayout.WEST)
                        add(JLabel("  › "), BorderLayout.EAST)
                    }, BorderLayout.WEST)
                } else {
                    add(JLabel("  › "), BorderLayout.WEST)
                }
                add(
                    JBScrollPane(
                        input,
                        ScrollPaneConstants.VERTICAL_SCROLLBAR_AS_NEEDED,
                        ScrollPaneConstants.HORIZONTAL_SCROLLBAR_NEVER,
                    ),
                    BorderLayout.CENTER,
                )
            }
            // statusLabel zaten SOUTH; girdiyi onun ÜZERİNE koy.
            tab.component.remove(tab.statusLabel)
            tab.component.add(
                JPanel(BorderLayout()).apply {
                    add(inputPanel, BorderLayout.NORTH)
                    add(tab.statusLabel, BorderLayout.SOUTH)
                },
                BorderLayout.SOUTH,
            )

            // Shell tarzı geçmiş. historyIndex = history.size → taslak düzenleniyor.
            val history = mutableListOf<String>()
            var historyIndex = 0
            var draft = ""

            // JTextArea'da ↑/↓ imleci satır atlatır. Çok satırlı taslakta
            // geçmişe atlamak yazılanı çöpe atardı: metin tek satırsa geçmiş,
            // değilse aracın kendi imleç hareketi çalışır.
            val caretUp = input.getActionForKeyStroke(KeyStroke.getKeyStroke("UP"))
            val caretDown = input.getActionForKeyStroke(KeyStroke.getKeyStroke("DOWN"))

            input.actionMap.put("cem.submit", object : AbstractAction() {
                override fun actionPerformed(e: ActionEvent) {
                    val typed = input.text.trim()
                    if (typed.isEmpty()) return
                    input.text = ""
                    if (history.isEmpty() || history.last() != typed) history.add(typed)
                    historyIndex = history.size
                    draft = ""
                    onSubmit(typed)
                }
            })
            input.inputMap.put(KeyStroke.getKeyStroke("ENTER"), "cem.submit")

            input.actionMap.put("cem.newline", object : AbstractAction() {
                override fun actionPerformed(e: ActionEvent) {
                    input.insert("\n", input.caretPosition)
                }
            })
            input.inputMap.put(KeyStroke.getKeyStroke("shift ENTER"), "cem.newline")

            input.actionMap.put("cem.history.prev", object : AbstractAction() {
                override fun actionPerformed(e: ActionEvent) {
                    if (input.text.contains('\n')) { caretUp?.actionPerformed(e); return }
                    if (history.isEmpty()) return
                    if (historyIndex == history.size) draft = input.text
                    if (historyIndex > 0) historyIndex--
                    input.text = history[historyIndex]
                    input.caretPosition = input.document.length
                }
            })
            input.inputMap.put(KeyStroke.getKeyStroke("UP"), "cem.history.prev")

            input.actionMap.put("cem.history.next", object : AbstractAction() {
                override fun actionPerformed(e: ActionEvent) {
                    if (input.text.contains('\n')) { caretDown?.actionPerformed(e); return }
                    if (history.isEmpty() || historyIndex >= history.size) return
                    historyIndex++
                    input.text = if (historyIndex == history.size) draft else history[historyIndex]
                    input.caretPosition = input.document.length
                }
            })
            input.inputMap.put(KeyStroke.getKeyStroke("DOWN"), "cem.history.next")

            return input
        }

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

            Aşağıdaki kutuya sorunu yaz, Enter'a bas. Soldaki seçici modu
            belirler: pair (düşünen → yazan) · think · write.
            Enter gönderir, Shift+Enter satır atlar, ↑/↓ önceki promptlar.
            Soru kod istemiyorsa yazan otomatik atlanır.

            Editör shortcut'ları:
              Ctrl+Alt+I  →  cem: think on selection
              Ctrl+Alt+W  →  cem: write on selection
              Ctrl+Alt+P  →  cem: pair on selection (thinker → writer)
              Ctrl+Alt+A  →  cem: ask freely (custom prompt)

            Seçim yokken kısayola basmak dialog açmaz: modu ayarlayıp
            imleci aşağıdaki kutuya getirir.

            Her çalıştırma kendi sekmesini açar ve o sekmenin altındaki kutuya
            yazarak SOHBETE DEVAM edilir: önceki tur bağlam olarak gider, araç
            bir soru sorduysa cevabını orada yazarsın.
            Terminal sekmesi: komutları proje kökünde çalıştırır (go test, git diff).

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

            // Mod seçici: dialog'lu akışta mod kısayolla sabitleniyordu
            // (Ctrl+Alt+W = write). Girdi kutuya taşınınca modun da burada
            // görünür ve değiştirilebilir olması gerekiyor.
            val modeBox = ComboBox(arrayOf(CemAction.Mode.PAIR, CemAction.Mode.THINK, CemAction.Mode.WRITE)).apply {
                renderer = SimpleListCellRenderer.create<CemAction.Mode> { label, value, _ ->
                    label.text = value.name.lowercase()
                }
                toolTipText = "pair: düşünen → yazan · think: sadece düşünen · write: sadece yazan"
            }

            val input = attachInput(
                tab,
                west = modeBox,
                tooltip = "Enter ↵ gönder · Shift+Enter satır atla · ↑/↓ önceki prompt",
            ) { typed ->
                val mode = modeBox.selectedItem as? CemAction.Mode ?: CemAction.Mode.PAIR
                val context = tab.takePendingContext()
                val prompt = if (context != null) "$typed\n\n$context" else typed
                tab.appendDim("→ ${mode.name.lowercase()}: ${promptSnippet(typed, 80)}")
                CemAction.launchCem(project, mode, prompt)
            }
            tab.inputArea = input
            tab.modeBox = modeBox

            interactiveTabs[project] = tab
            return tab
        }

        /**
         * "Terminal" sekmesi — cem penceresinden komut çalıştırma.
         *
         * Neden burada: yazan rol dosya üretiyor, sonra kullanıcı `go test`,
         * `git diff`, `npm run build` çalıştırmak istiyor ve bunun için başka
         * bir araç penceresine geçmek gerekiyordu. Aynı pencerede kalmak,
         * cevabı ve komut çıktısını yan yana tutuyor.
         *
         * IDE'nin kendi Terminal'inin yerine geçmez (tam bir pty değil,
         * interaktif programlar — vim, top — burada çalışmaz). Amaç tek
         * seferlik komutlar: kabuk üzerinden çalıştırıldığı için pipe,
         * yönlendirme ve && zincirleri geçerli.
         */
        fun commandTab(project: Project, toolWindow: ToolWindow): CemTab {
            val tab = CemTab()
            tab.appendStyled(commandWelcomeText(project), bold = false, color = JBColor.GRAY)

            val stop = JButton("⏹").apply {
                toolTipText = "çalışan komutu durdur"
                isEnabled = false
            }
            val input = attachInput(
                tab,
                west = stop,
                tooltip = "Enter ↵ çalıştır · Shift+Enter satır atla · ↑/↓ önceki komutlar",
            ) { line ->
                if (tab.process?.isAlive == true) {
                    tab.appendError("önceki komut hâlâ çalışıyor — ⏹ ile durdur")
                } else {
                    tab.appendStyled("\n$ $line\n", bold = true, color = JBColor.foreground())
                    runShellCommand(project, tab, line, stop)
                }
            }
            tab.inputArea = input
            stop.addActionListener {
                tab.process?.let { p ->
                    if (p.isAlive) {
                        p.destroy()
                        tab.appendDim("─── durduruldu ───")
                    }
                }
            }

            val content = ContentFactory.getInstance().createContent(tab.component, "Terminal", false)
            content.isCloseable = false
            content.isPinned = true
            // Sekme kapanmıyor ama pencere kapanınca çalışan komut da ölsün.
            content.setDisposer(Disposable { tab.cancel() })
            toolWindow.contentManager.addContent(content)
            return tab
        }

        private fun commandWelcomeText(project: Project): String =
            """
            Terminal — komutlar proje kökünde ve kabuk üzerinden çalışır.
            Dizin: ${project.basePath ?: "?"}

            Enter ↵ çalıştır · Shift+Enter satır atla · ↑/↓ önceki komutlar · ⏹ durdur
            Not: tam bir pty değil — vim/top gibi interaktif programlar için IDE'nin
            kendi Terminal penceresini kullan.

            """.trimIndent() + "\n"

        /**
         * Komutu kabukla çalıştırır ve çıktıyı sekmeye akıtır.
         *
         * Kabuk şart: kullanıcı "go build ./... && go test" ya da
         * "grep -rn foo | head" yazabiliyor. Windows'ta PowerShell, diğer
         * yerlerde /bin/sh. stderr stdout'a katılıyor (redirectErrorStream):
         * hata mesajı çıktının neresinde olduğunu görmek gerekiyor.
         */
        private fun runShellCommand(project: Project, tab: CemTab, line: String, stop: JButton) {
            ApplicationManager.getApplication().executeOnPooledThread {
                val başlangıç = System.currentTimeMillis()
                try {
                    val cmd =
                        if (SystemInfo.isWindows) listOf("powershell.exe", "-NoProfile", "-Command", line)
                        else listOf("/bin/sh", "-lc", line)
                    val pb = ProcessBuilder(cmd).redirectErrorStream(true)
                    project.basePath?.let { pb.directory(java.io.File(it)) }
                    val process = pb.start()
                    // stdin'i kapat: girdi bekleyen komut sessizce asılı kalmasın.
                    try { process.outputStream.close() } catch (_: Exception) {}
                    tab.process = process
                    ApplicationManager.getApplication().invokeLater { stop.isEnabled = true }

                    val reader = java.io.InputStreamReader(process.inputStream, Charsets.UTF_8)
                    val buf = CharArray(2048)
                    while (!tab.cancelled) {
                        val n = reader.read(buf)
                        if (n < 0) break
                        val chunk = String(buf, 0, n)
                        ApplicationManager.getApplication().invokeLater { tab.appendRaw(chunk) }
                    }
                    val exit = process.waitFor()
                    val süre = (System.currentTimeMillis() - başlangıç) / 1000
                    ApplicationManager.getApplication().invokeLater {
                        stop.isEnabled = false
                        if (exit == 0) {
                            tab.appendDim("─── bitti (${süre}s) ───")
                            tab.setStatus("✓ ${süre}s")
                        } else {
                            tab.appendError("─── çıkış $exit (${süre}s) ───")
                            tab.setStatus("✗ çıkış $exit · ${süre}s")
                        }
                    }
                } catch (e: Exception) {
                    ApplicationManager.getApplication().invokeLater {
                        stop.isEnabled = false
                        tab.appendError("komut çalıştırılamadı: ${e.message}")
                    }
                }
            }
        }

        /** Proje başına Interactive sekmesi — aksiyonlar girdi kutusunu buradan bulur. */
        private val interactiveTabs = WeakHashMap<Project, CemTab>()

        /**
         * Prompt'u modal dialog yerine araç penceresinin ALTINDAKİ kutuda ister.
         *
         * Dialog her seferinde ekranın ortasını kapatıyor, fare gerektiriyor ve
         * kapanınca yazılan metin kayboluyordu; ayrıca aynı iş için iki ayrı
         * girdi yeri (dialog + Interactive sekmesi) vardı. Artık tek yer var:
         * kısayol kutuyu hazırlar, kullanıcı yazıp Enter'a basar.
         *
         * false dönerse araç penceresi yok — çağıran dialog'a düşer.
         */
        fun askInInput(
            project: Project,
            mode: CemAction.Mode,
            context: String? = null,
            hint: String? = null,
        ): Boolean {
            val tw = ToolWindowManager.getInstance(project).getToolWindow("cem") ?: return false
            tw.activate({
                val tab = interactiveTabs[project] ?: return@activate
                tw.contentManager.contents
                    .firstOrNull { it.component === tab.component }
                    ?.let { tw.contentManager.setSelectedContent(it) }
                tab.prepareInput(mode, context, hint)
            }, true, true)
            return true
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
