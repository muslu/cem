package dev.cempw.intellij

import com.intellij.openapi.util.SystemInfo
import java.io.File

/**
 * Terminal sekmesinin Tab tamamlaması — kabuk tarzı, ama kabuğa sormadan.
 *
 * Neden burada: Terminal sekmesi tam bir pty değil, komut her Enter'da yeni
 * bir `/bin/sh -c` ile çalışıyor; dolayısıyla bash/zsh'ın kendi tamamlaması
 * yok. `python3 fe⇥` yazan kullanıcı dosya adını elle bitirmek ya da IDE'nin
 * Terminal'ine geçmek zorundaydı. Burası JVM'in dosya sistemi ve PATH
 * bilgisiyle aynı işi yapar: komut konumunda PATH'teki çalıştırılabilirler,
 * diğer konumlarda proje köküne göre dosya/dizin adları.
 *
 * Kural kabuktakiyle aynı: tek aday → tamamla (dizinse `/`, dosyaysa boşluk
 * ekle); çok aday → ortak öneke kadar tamamla, ilerleme yoksa adayları
 * `options` ile döndür — çağıran bunları seçilebilir bir listede gösterir ve
 * seçimi `Result.pick` ile kutuya yazar. (İlk sürüm adayları çıktı alanına
 * basıyordu: `python⇥` her basışta 30 adı bir daha döküyor, kutu ise olduğu
 * gibi kalıyordu — kullanıcı "tamamlamıyor, üstte yazıyor" dedi.)
 * Gizli dosyalar (`.git`) ancak `.` yazılmışsa gelir.
 *
 * Saf mantık `complete`'te: sözcük ayırma, PATH ve cwd parametre — IDE'siz
 * test edilebilsin diye.
 */
object CemCompletion {

    /** Listeye düşen aday sayısının üst sınırı — `/usr/bin` 3000 satır basmasın. */
    const val MAX_LISTED = 200

    /** Listede sunulan tek aday: `display` gösterilir, seçilince `insert` yazılır. */
    data class Option(val display: String, val insert: String, val isDir: Boolean)

    /**
     * Tamamlama sonucu. `text`/`caret` kutunun yeni hâli (değişmediyse
     * eskinin aynısı); `options` boş değilse kullanıcıya seçtirilir.
     * `start` = tamamlanan sözcüğün `text` içindeki başlangıcı — `pick`
     * seçimi oraya yazar.
     */
    data class Result(val text: String, val caret: Int, val options: List<Option>, val start: Int = caret) {
        /** Listedeki adların düz hâli (testler ve eski çağıranlar için). */
        val candidates: List<String> get() = options.map { it.display }

        /** Kullanıcının listeden seçtiği adayı sözcüğün yerine yazar. */
        fun pick(option: Option): Result =
            replaceWord(text, start, caret, option.insert, if (option.isDir) "/" else " ")
    }

    /**
     * `text` içinde `caret` konumundaki sözcüğü tamamlar.
     *
     * @param cwd      göreli yolların kökü (proje kökü).
     * @param pathDirs komut adaylarının arandığı dizinler (`PATH`).
     */
    fun complete(text: String, caret: Int, cwd: File, pathDirs: List<File>): Result {
        val start = wordStart(text, caret)
        val unchanged = Result(text, caret, emptyList(), start)
        val word = unescape(text.substring(start, caret))
        val isCommand = isCommandPosition(text, start) && !word.contains('/') && !word.startsWith("~")

        val matches: List<Candidate> =
            if (isCommand) commandCandidates(word, pathDirs) else pathCandidates(word, cwd)
        if (matches.isEmpty()) return unchanged

        val sorted = matches.sortedBy { it.insert }
        if (sorted.size == 1) {
            val only = sorted.single()
            val suffix = if (only.isDir) "/" else " "
            return replaceWord(text, start, caret, only.insert, suffix)
        }

        val common = commonPrefix(sorted.map { it.insert })
        // Ortak önek zaten yazılı olandan uzun değilse ilerleme yok → seçtir.
        // Eşit uzunlukta olabilir: "foo" yazılıyken "foo" ve "foo.txt" varsa.
        val typedBase = word.substringAfterLast('/')
        val commonBase = common.substringAfterLast('/')
        return if (commonBase.length > typedBase.length) replaceWord(text, start, caret, common, "")
        else Result(text, caret, sorted.map { Option(it.display, it.insert, it.isDir) }, start)
    }

    private data class Candidate(
        /** Sözcüğün yerine yazılacak tam metin (dizin öneki dahil, kaçırılmamış). */
        val insert: String,
        /** Listede gösterilen ad (yalnız son parça, dizinse `/` ekli). */
        val display: String,
        val isDir: Boolean,
    )

    /** Sözcüğü kaçırarak yerine yazar; `suffix` (boşluk ya da `/`) kaçırılmaz. */
    private fun replaceWord(text: String, start: Int, end: Int, word: String, suffix: String): Result {
        val escaped = escape(word) + suffix
        val newText = text.substring(0, start) + escaped + text.substring(end)
        return Result(newText, start + escaped.length, emptyList(), start)
    }

    // ── sözcük ayırma ────────────────────────────────────────────────────

    /**
     * Tamamlanacak sözcüğün başlangıcı: imleçten geriye ilk KAÇIRILMAMIŞ
     * boşluk ya da kabuk ayracı (`|`, `;`, `&`, `(`, `<`, `>`, `=`).
     * `=` ayraç: `--out=dos⇥` biçiminde bayrak değerleri de tamamlansın.
     */
    internal fun wordStart(text: String, caret: Int): Int {
        var i = caret
        while (i > 0) {
            val c = text[i - 1]
            val escaped = i >= 2 && text[i - 2] == '\\' && !SystemInfo.isWindows
            if (!escaped && (c.isWhitespace() || c in "|;&()<>=")) break
            i--
        }
        return i
    }

    /**
     * Sözcük komut konumunda mı? Satır başı ya da `|`, `||`, `&&`, `;`
     * sonrası ilk sözcükse evet. `VAR=deger komut` biçimindeki ortam
     * atamalarını atlar.
     */
    internal fun isCommandPosition(text: String, wordStart: Int): Boolean {
        val before = text.substring(0, wordStart)
        val segment = before.split('|', ';', '&', '\n', '(').last().trim()
        if (segment.isEmpty()) return true
        // `sudo ls⇥`, `FOO=1 ls⇥`, `time ls⇥` — komut hâlâ komut.
        return segment.split(Regex("\\s+")).all { it.matches(Regex("[A-Za-z_][A-Za-z0-9_]*=.*")) || it in commandPrefixes }
    }

    private val commandPrefixes = setOf("sudo", "time", "nohup", "exec", "env", "nice", "command", "builtin")

    // ── adaylar ──────────────────────────────────────────────────────────

    private fun commandCandidates(prefix: String, pathDirs: List<File>): List<Candidate> {
        if (prefix.isEmpty()) return emptyList()  // boş önekle tüm PATH'i dökmek anlamsız
        val seen = LinkedHashSet<String>()
        for (dir in pathDirs) {
            val entries = dir.listFiles() ?: continue
            for (f in entries) {
                if (!f.name.startsWith(prefix)) continue
                if (!f.isFile || !isExecutable(f)) continue
                seen.add(f.name)
            }
        }
        return seen.map { Candidate(it, it, isDir = false) }
    }

    private fun isExecutable(f: File): Boolean =
        if (SystemInfo.isWindows) windowsExecExt.any { f.name.endsWith(it, ignoreCase = true) }
        else f.canExecute()

    private val windowsExecExt: List<String> by lazy {
        (System.getenv("PATHEXT") ?: ".COM;.EXE;.BAT;.CMD;.PS1").split(';').filter { it.isNotBlank() }
    }

    private fun pathCandidates(word: String, cwd: File): List<Candidate> {
        val slash = word.lastIndexOf('/')
        val dirPart = if (slash >= 0) word.substring(0, slash + 1) else ""
        val base = word.substring(slash + 1)
        val dir = resolveDir(dirPart, cwd) ?: return emptyList()
        val entries = dir.listFiles() ?: return emptyList()
        val showHidden = base.startsWith(".")
        return entries
            .filter { it.name.startsWith(base) && (showHidden || !it.name.startsWith(".")) }
            .map { f ->
                val isDir = f.isDirectory
                Candidate(dirPart + f.name, if (isDir) f.name + "/" else f.name, isDir)
            }
    }

    private fun resolveDir(dirPart: String, cwd: File): File? {
        val expanded = when {
            dirPart == "~" || dirPart.startsWith("~/") ->
                System.getProperty("user.home") + dirPart.substring(1)
            else -> dirPart
        }
        val f = if (expanded.isEmpty()) cwd else File(expanded).let { if (it.isAbsolute) it else File(cwd, expanded) }
        return if (f.isDirectory) f else null
    }

    // ── yardımcılar ──────────────────────────────────────────────────────

    internal fun commonPrefix(items: List<String>): String {
        if (items.isEmpty()) return ""
        var p = items.first()
        for (s in items) {
            var n = 0
            while (n < p.length && n < s.length && p[n] == s[n]) n++
            p = p.substring(0, n)
            if (p.isEmpty()) break
        }
        return p
    }

    /** Boşluk ve kabuk-özel karakterleri kaçırır (`Yeni Dosya.txt` → `Yeni\ Dosya.txt`). */
    internal fun escape(s: String): String =
        if (SystemInfo.isWindows) s
        else buildString {
            for (c in s) {
                if (c.isWhitespace() || c in "|;&()<>$\"'`\\*?[]") append('\\')
                append(c)
            }
        }

    internal fun unescape(s: String): String =
        if (SystemInfo.isWindows) s
        else buildString {
            var i = 0
            while (i < s.length) {
                val c = s[i]
                if (c == '\\' && i + 1 < s.length) { append(s[i + 1]); i += 2 } else { append(c); i++ }
            }
        }

    /** `PATH` ortam değişkenindeki dizinler. */
    fun pathDirs(): List<File> =
        (System.getenv("PATH") ?: "").split(File.pathSeparatorChar)
            .filter { it.isNotBlank() }
            .map { File(it) }
}
