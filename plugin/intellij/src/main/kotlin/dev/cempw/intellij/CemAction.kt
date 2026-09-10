package dev.cempw.intellij

import com.intellij.ide.BrowserUtil
import com.intellij.notification.NotificationAction
import com.intellij.notification.NotificationGroupManager
import com.intellij.openapi.options.ShowSettingsUtil
import com.intellij.notification.NotificationType
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.actionSystem.CommonDataKeys
import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.diagnostic.Logger
import com.intellij.openapi.editor.Editor
import com.intellij.openapi.fileEditor.FileDocumentManager
import com.intellij.openapi.fileEditor.FileEditorManager
import com.intellij.openapi.ide.CopyPasteManager
import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.Messages
import com.intellij.openapi.vfs.VirtualFile
import com.intellij.openapi.wm.ToolWindowManager
import java.awt.datatransfer.StringSelection
import java.io.InputStreamReader

/**
 * cem editor actions (think / write / pair).
 *
 * Seçim varsa onu gönderir; seçim yoksa kullanıcıdan prompt ister.
 */
sealed class CemAction(val mode: Mode) : AnAction() {

    enum class Mode(val flag: String?) {
        THINK(null),
        WRITE("-w"),
        PAIR("-p"),
    }

    class Think : CemAction(Mode.THINK)
    class Write : CemAction(Mode.WRITE)
    class Pair : CemAction(Mode.PAIR)

    override fun actionPerformed(event: AnActionEvent) {
        val project = event.project ?: return
        val editor: Editor? = event.getData(CommonDataKeys.EDITOR)
            ?: FileEditorManager.getInstance(project).selectedTextEditor
        val text = pickPrompt(project, editor) ?: return
        launchCem(project, mode, text)
    }

    /** Seçim yokken gösterilecek ipucu — açık dosya varsa adıyla. */
    private fun noSelectionHint(fileName: String?): String =
        if (fileName != null) {
            "  ↓ ${mode.name.lowercase()} · seçim yok (açık dosya: $fileName) — sorunu aşağıya yaz, Enter ↵"
        } else {
            "  ↓ ${mode.name.lowercase()} · sorunu aşağıya yaz, Enter ↵"
        }

    /**
     * Seçim yoksa AÇIK DOSYAYI PROMPT SANMA.
     *
     * Eskiden seçim yokken editördeki dosyanın tamamı prompt olarak
     * gönderiliyordu: kullanıcı README.md açıkken Ctrl+Alt+P'ye basınca
     * tüm README görev sanılıp pair modunda iki modele birden yollanıyor,
     * 112 saniye ve para harcanıp "yapılacak yeni kod yok" cevabı
     * dönüyordu (ölçüldü, 2026-09-09). plugin.xml'deki açıklama da zaten
     * "Empty selection → input dialog" diyor; kod ondan sapmıştı.
     *
     * Dosyayı bağlam olarak göndermek isteyen için proje ağacındaki
     * "cem: review file… / ask about file…" aksiyonları var; cem zaten
     * proje dizininde çalıştığı için dosyayı adıyla da isteyebilir.
     */
    private fun pickPrompt(project: Project, editor: Editor?): String? {
        val sel = editor?.selectionModel?.selectedText?.takeIf { it.isNotBlank() }
        if (sel != null) return sel
        val fileName = editor?.let {
            FileDocumentManager.getInstance().getFile(it.document)?.name
        }
        // Modal dialog yerine araç penceresinin altındaki girdi kutusu:
        // dialog ekranın ortasını kapatıyor, fare istiyor ve iptal edilince
        // yazılan metin kayboluyordu. Kutu kalıcı — mod da orada görünür.
        // Araç penceresi hiç yoksa (kayıt edilmemişse) dialog'a düşeriz.
        if (CemTab.askInInput(project, mode, null, noSelectionHint(fileName))) return null
        val message = if (fileName != null) {
            "cem'e ne sormak istiyorsun? (seçim yok — açık dosya: $fileName)"
        } else {
            "cem'e ne sormak istiyorsun?"
        }
        return promptUser(project, message)
    }

    companion object {
        internal val LOG = Logger.getInstance(CemAction::class.java)

        /** Çok-satırlı input dialog. Kullanıcı iptal ederse null. */
        fun promptUser(project: Project, message: String, default: String = ""): String? {
            val resp = Messages.showMultilineInputDialog(
                project, message, "cem", default, Messages.getQuestionIcon(), null,
            )
            return resp?.takeIf { it.isNotBlank() }
        }

        private val urlRegex = Regex("""https?://[^\s<>"']+""")

        /** URL'in kendisi bir login akışına mı işaret ediyor? */
        private val authUrlRegex =
            Regex("""(?i)(accounts\.|/oauth|/auth|/login|/signin|/device|/verify)""")

        /** Metinde açık bir "giriş yap" ifadesi var mı? */
        private val authHintRegex = Regex(
            """(?i)(sign\s?in|log\s?in|please login|authenticate|authorization code|""" +
                """authorize|paste the code|open this url|oturum aç|giriş yap|yetkilendir)"""
        )

        /** Yerel adresler login linki değildir (MCP endpoint'leri, dev sunucular). */
        private val localUrlRegex =
            Regex("""(?i)https?://(127\.0\.0\.1|localhost|0\.0\.0\.0|\[::1\])""")

        /**
         * Çıktıdan LOGIN url'ini çıkar — yoksa null.
         *
         * Eskiden buradaki `extractUrls(...).firstOrNull()` çıktıda geçen HERHANGİ
         * bir bağlantı için "authentication needed" balonu gösteriyordu. Kullanıcının
         * seçtiği dosyada bir URL varsa (ör. .mcp.json içindeki
         * http://127.0.0.1:64462/stream) yanlış uyarı çıkıyordu. Artık ya URL'in
         * kendisi bir auth adresi olmalı, ya da metinde açık bir giriş ifadesi.
         */
        fun extractAuthUrl(text: String): String? {
            val urls = urlRegex.findAll(text).map { it.value }.toList()
                .filter { !localUrlRegex.containsMatchIn(it) }
                .filter { url ->
                    val u = url.lowercase()
                    "github.com/muslu/cem" !in u && "cem.pw" !in u
                }
            if (urls.isEmpty()) return null
            urls.firstOrNull { authUrlRegex.containsMatchIn(it) }?.let { return it }
            return if (authHintRegex.containsMatchIn(text)) urls.first() else null
        }

        /** OAuth URL'i bulunca IDE balloon notification göster + tarayıcı/kopyala butonu. */
        /**
         * Kurulum yapılmamış: eklentiden sihirbaz çalıştırılamaz (TTY yok),
         * o yüzden kullanıcıyı GUI'ye yönlendiriyoruz.
         *
         * cem kurulum olmadan çalışmayı reddediyor — yarım yapılandırmayla
         * çalıştırmak, kullanıcının seçmediği araca istek göndermek demek.
         * Eklentide bunun karşılığı bu bildirim: tek tıkla ayar sayfası.
         */
        fun notifySetupNeeded(project: Project) {
            val group = NotificationGroupManager.getInstance()
                .getNotificationGroup("cem.setup") ?: return
            group.createNotification(
                "cem: kurulum gerekli",
                "Düşünen ve yazan rolü seçilmeden cem çalışmıyor.",
                NotificationType.WARNING,
            ).addAction(NotificationAction.createSimple("Ayarları aç") {
                ShowSettingsUtil.getInstance()
                    .showSettingsDialog(project, CemSettingsConfigurable::class.java)
            }).addAction(NotificationAction.createSimple("Terminalde: cem setup") {
                CopyPasteManager.getInstance().setContents(StringSelection("cem setup"))
            }).notify(project)
        }

        fun notifyAuthUrl(project: Project, url: String) {
            val group = NotificationGroupManager.getInstance()
                .getNotificationGroup("cem.auth") ?: return
            group.createNotification(
                "cem: authentication needed",
                "Tarayıcıda aç veya kodu kopyalayıp 'cem auth <tool> --code ...' kullan.",
                NotificationType.WARNING,
            ).addAction(NotificationAction.createSimple("Open in Browser") {
                BrowserUtil.browse(url)
            }).addAction(NotificationAction.createSimple("Copy URL") {
                CopyPasteManager.getInstance().setContents(StringSelection(url))
            }).notify(project)
        }

        /**
         * cem binary'sini bul. Sıra:
         *   1. Settings'te abs path verilmişse ve dosya varsa → onu kullan
         *   2. ProcessBuilder PATH'ı kontrol et (where/which)
         *   3. Bilinen kurulum konumları (Win: %LOCALAPPDATA%\cem\bin\cem.exe,
         *      Unix: /usr/local/bin/cem, ~/.local/bin/cem)
         *   4. Hiçbiri yoksa Settings değerini olduğu gibi döndür (anlamlı hata için)
         */
        fun resolveCemBinary(): String {
            val configured = CemSettings.instance.cemPath.ifBlank { "cem" }
            val isWindows = System.getProperty("os.name").startsWith("Windows", ignoreCase = true)
            val ext = if (isWindows) ".exe" else ""

            // 1) Absolute path verilmişse direkt kontrol et
            if (configured != "cem" && configured != "cem.exe") {
                val f = java.io.File(configured)
                if (f.exists() && f.canExecute()) return configured
            }

            // 2) PATH üzerinden ara
            val pathEnv = System.getenv("PATH") ?: ""
            for (dir in pathEnv.split(java.io.File.pathSeparator)) {
                if (dir.isBlank()) continue
                val candidate = java.io.File(dir, "cem$ext")
                if (candidate.exists() && candidate.canExecute()) return candidate.absolutePath
            }

            // 3) Bilinen kurulum konumları
            val home = System.getProperty("user.home")
            val candidates = if (isWindows) {
                listOfNotNull(
                    System.getenv("LOCALAPPDATA")?.let { "$it\\cem\\bin\\cem.exe" },
                    "$home\\.local\\bin\\cem.exe",
                    "C:\\Program Files\\cem\\cem.exe",
                )
            } else {
                listOf(
                    "/usr/local/bin/cem",
                    "$home/.local/bin/cem",
                    "/opt/homebrew/bin/cem",
                    "/home/linuxbrew/.linuxbrew/bin/cem",
                )
            }
            for (path in candidates) {
                val f = java.io.File(path)
                if (f.exists() && f.canExecute()) return path
            }

            // 4) Bulunamadı; configured'u döndür, ProcessBuilder anlamlı hata atacak
            return configured
        }

        /** Background thread'de cem'i çalıştır + sonucu yeni tab'a stream et. */
        fun launchCem(project: Project, mode: Mode, prompt: String) {
            val toolWindow = ToolWindowManager.getInstance(project).getToolWindow("cem")
                ?: return
            val tab = CemTab.newRun(toolWindow, mode.name.lowercase(), prompt.take(80))
            // Sohbete devam kutusu: araç soru sorduğunda ya da kullanıcı
            // "şunu da ekle" demek istediğinde aynı sekmede yeni tur açılır.
            tab.enableFollowUp(project, mode)
            ApplicationManager.getApplication().executeOnPooledThread {
                try {
                    runCem(project, mode, prompt, tab)
                } catch (e: Exception) {
                    LOG.warn("cem invocation failed", e)
                    ApplicationManager.getApplication().invokeLater {
                        tab.appendError("Failed to invoke cem: ${e.message}")
                    }
                }
            }
        }

        /**
         * Aynı sekmede devam turu: önceki turlar bağlam olarak gönderilir.
         *
         * cem'in oturumu yok — her çağrı yeni bir süreç. Devam etmenin tek
         * yolu önceki konuşmayı metin olarak taşımak (CemTab.followUpPrompt),
         * o yüzden bağlam kırpılıyor: her tur yeniden faturalanıyor.
         */
        fun continueInTab(project: Project, tab: CemTab, mode: Mode, message: String) {
            val prompt = tab.followUpPrompt(message)
            ApplicationManager.getApplication().executeOnPooledThread {
                try {
                    runCem(project, mode, prompt, tab, requestForTranscript = message)
                } catch (e: Exception) {
                    LOG.warn("cem follow-up failed", e)
                    ApplicationManager.getApplication().invokeLater {
                        tab.appendError("Failed to invoke cem: ${e.message}")
                    }
                } finally {
                    tab.setInputEnabled(true)
                }
            }
        }

        private fun runCem(
            project: Project,
            mode: Mode,
            prompt: String,
            tab: CemTab,
            requestForTranscript: String = prompt,
        ) {
            val cemPath = resolveCemBinary()
            val workDir = project.basePath
            val cmd = mutableListOf(cemPath)
            mode.flag?.let { cmd.add(it) }
            cmd.add(prompt)
            val pb = ProcessBuilder(cmd).redirectErrorStream(true)
            if (workDir != null) pb.directory(java.io.File(workDir))
            tab.appendDim("  → $cemPath${mode.flag?.let { " $it" } ?: ""}")
            val startTime = System.currentTimeMillis()
            val process = pb.start()
            // cem io.ReadAll(stdin) hang fix: child'ın stdin'ini hemen kapat ki
            // EOF görsün. Redirect.DISCARD sadece output için geçerli, stdin'de
            // 'Invalid for reading: WRITE' hatası veriyor.
            try { process.outputStream.close() } catch (_: Exception) {}
            tab.process = process

            // Status bar spinner — text pane'i kirletmez. 100ms'de bir
            // dönen unicode karakter + geçen süre. Süreç bitince temizlenir.
            val spinnerThread = Thread {
                val frames = listOf("⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏")
                var i = 0
                while (!tab.cancelled && process.isAlive) {
                    val secs = (System.currentTimeMillis() - startTime) / 1000
                    tab.setStatus("${frames[i % frames.size]}  ${secs}s · running (PID ${process.pid()})")
                    i++
                    try { Thread.sleep(100) } catch (_: InterruptedException) { break }
                }
            }
            spinnerThread.isDaemon = true
            spinnerThread.start()

            // Char-by-char oku — line buffering ile bekleme yok.
            val reader = InputStreamReader(process.inputStream, Charsets.UTF_8)
            val buf = CharArray(2048)
            val fullOutput = StringBuilder()
            while (!tab.cancelled) {
                val n = reader.read(buf)
                if (n < 0) break
                val chunk = String(buf, 0, n)
                fullOutput.append(chunk)
                ApplicationManager.getApplication().invokeLater { tab.appendRaw(chunk) }
            }
            val exit = process.waitFor()
            // OAuth/login URL'lerini yakala, kullanıcıya notification göster:
            // PSReadline'da uzun URL kopyalamak/yapıştırmak zor olabiliyor.
            // Sadece BAŞARISIZ çalıştırmalarda: auth gerçekten eksikse cem
            // sıfırdan farklı bir kodla çıkar. Başarılı bir cevabın içinde
            // geçen bağlantı login uyarısı değildir.
            if (exit != 0) {
                val çıktı = fullOutput.toString()
                // Kurulum uyarısı auth'tan ÖNCE bakılıyor: kurulum yoksa araç
                // hiç çalıştırılmadı, dolayısıyla auth mesajı da olmaz.
                if (çıktı.contains("cem setup")) {
                    ApplicationManager.getApplication().invokeLater {
                        tab.appendDim("Kurulum için: Settings → Tools → cem (ya da terminalde: cem setup)")
                        notifySetupNeeded(project)
                    }
                } else {
                    extractAuthUrl(çıktı)?.let { url ->
                        ApplicationManager.getApplication().invokeLater {
                            notifyAuthUrl(project, url)
                        }
                    }
                }
            }
            val totalSecs = (System.currentTimeMillis() - startTime) / 1000
            // Turu döküme yaz: bir sonraki "sohbete devam" isteği bunu bağlam
            // olarak gönderiyor. Gönderilen prompt DEĞİL, kullanıcının yazdığı
            // istek kaydediliyor — devam turunda prompt zaten önceki dökümü
            // içeriyor, onu tekrar eklemek bağlamı her turda katlardı.
            tab.noteExchange(requestForTranscript, fullOutput.toString())
            // Tab kapatıldıysa final mesajı yazmıyoruz — content zaten gitti
            if (tab.cancelled) return
            ApplicationManager.getApplication().invokeLater {
                if (exit != 0) {
                    tab.appendError("─── exit $exit (after ${totalSecs}s) ───")
                    tab.setStatus("✗ exit $exit  ·  ${totalSecs}s")
                } else {
                    tab.appendDim("─── done in ${totalSecs}s ───")
                    tab.setStatus("✓ done  ·  ${totalSecs}s")
                }
                tab.setInputEnabled(true)
            }
        }
    }
}

/**
 * cem "ask freely" — editor state'inden bağımsız, her zaman input dialog
 * açıp serbest prompt alır. Hiçbir dosya açık değilken bile çalışır.
 */
class CemAskAction : AnAction() {
    override fun actionPerformed(event: AnActionEvent) {
        val project = event.project ?: return
        if (CemTab.askInInput(
                project, CemAction.Mode.PAIR, null,
                "  ↓ pair · sorunu aşağıya yaz, Enter ↵",
            )
        ) return
        val prompt = CemAction.promptUser(project, "What do you want to ask cem?") ?: return
        CemAction.launchCem(project, CemAction.Mode.PAIR, prompt)
    }
}

/**
 * Project tree'de dosyaya sağ-tık ile çağrılan file-level aksiyonlar.
 *
 * Her aksiyon dosya içeriğini okuyup önceden tanımlı bir talimat prefix'iyle
 * cem -p'ye yollar (pair modu, çünkü hem analiz hem üretim faydalı).
 */
sealed class CemFileAction(private val instruction: String) : AnAction() {

    class Review : CemFileAction(
        "Aşağıdaki dosyayı kod kalitesi, olası bug'lar, eksikler ve iyileştirme " +
                "önerileri için incele. Kısa ve eyleme dönüştürülebilir maddelerle yaz.",
    )
    class FixErrors : CemFileAction(
        "Aşağıdaki dosyada hata, bug veya yanlış kullanım varsa tespit et ve " +
                "düzeltilmiş kodu üret. Sadece değişen kısmı belirt, açıklamayı kısa tut.",
    )
    class Explain : CemFileAction(
        "Aşağıdaki dosyanın ne yaptığını, ana akışını ve önemli detaylarını sadece " +
                "düz prozayla açıkla. Kod tekrar etme.",
    )
    class Ask : CemFileAction("__ASK__") // kullanıcı talimatı interaktif girer

    override fun actionPerformed(event: AnActionEvent) {
        val project = event.project ?: return
        val files = event.getData(CommonDataKeys.VIRTUAL_FILE_ARRAY)
            ?: event.getData(CommonDataKeys.VIRTUAL_FILE)?.let { arrayOf(it) }
            ?: run {
                Messages.showWarningDialog(project, "Hiç dosya seçilmedi.", "cem")
                return
            }
        // Çoklu seçim → tek prompt'ta birleştir (dosya başına başlık ile).
        val sb = StringBuilder()
        for (f in files) {
            if (f.isDirectory) continue
            val text = readFile(f) ?: continue
            sb.append("===== ").append(f.path).append(" =====\n")
            sb.append(text).append("\n\n")
        }
        if (sb.isEmpty()) {
            Messages.showWarningDialog(project, "Seçili dosyaların hiçbiri okunamadı.", "cem")
            return
        }
        val context = sb.toString().trim()
        if (instruction == "__ASK__") {
            // Talimat girdi kutusunda alınır; dosya içeriği bağlam olarak
            // bekletilir ve Enter'a basıldığında prompt'un altına eklenir.
            val names = files.filter { !it.isDirectory }.joinToString(", ") { it.name }
            if (CemTab.askInInput(
                    project, CemAction.Mode.PAIR, context,
                    "  ↓ pair · $names eklendi — talimatını aşağıya yaz, Enter ↵",
                )
            ) return
            val typed = CemAction.promptUser(
                project,
                "Bu dosya(lar) için cem'e ne sormak istiyorsun?",
            ) ?: return
            CemAction.launchCem(project, CemAction.Mode.PAIR, "$typed\n\n$context")
            return
        }
        CemAction.launchCem(project, CemAction.Mode.PAIR, "$instruction\n\n$context")
    }

    private fun readFile(file: VirtualFile): String? = try {
        file.inputStream.bufferedReader(Charsets.UTF_8).use { it.readText() }
    } catch (_: Exception) {
        null
    }
}
