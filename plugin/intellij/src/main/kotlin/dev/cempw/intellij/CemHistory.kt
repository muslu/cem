package dev.cempw.intellij

import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.application.PathManager
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.Paths
import java.nio.file.StandardOpenOption

/**
 * Girdi kutularının geçmişi — IDE oturumları arasında KALICI ve tüm sekmeler
 * arasında PAYLAŞIMLI.
 *
 * Neden: geçmiş eskiden `attachInput` içinde yerel bir listeydi. Her sekme
 * (Interactive, her çalıştırma sekmesinin "sohbete devam" kutusu, her Terminal)
 * kendi listesini tutuyordu; IDE kapanınca hepsi gidiyordu. Kullanıcı dünkü
 * sohbetini ↑ ile geri çağıramıyor, yeni açılan bir sekmede az önce yazdığı
 * isteği bulamıyordu.
 *
 * Sohbet ve komut AYRI tutulur (`Kind`): `go test ./...` satırının cem'e
 * gönderilecek prompt'ların arasına karışması, ↑ ile gezinmeyi kullanılmaz
 * yapardı.
 *
 * Depolama IDE'nin config dizini altında, cem'in `~/.cem/` dizinine
 * DOKUNULMAZ — `~/.cem/history.log` cem'in kendi TSV kaydıdır ve girdiyi
 * 80 karakterde kırpar (`history.go`); oradan geri yüklenen bir prompt
 * sessizce EKSİK gönderilirdi.
 */
object CemHistory {

    enum class Kind(val fileName: String) {
        /** cem'e gönderilen istekler: Interactive + "sohbete devam". */
        CHAT("history-chat.txt"),

        /** Terminal sekmelerinde çalıştırılan kabuk komutları. */
        COMMAND("history-command.txt"),
    }

    /** Geçmişte tutulan en fazla girdi sayısı. */
    const val MAX_ENTRIES = 200

    /**
     * Saklanan bir girdinin karakter üst sınırı. Daha uzun girdi geçmişe HİÇ
     * yazılmaz — kırpmak tehlikeli olurdu: kullanıcı ↑ ile çağırıp Enter'a
     * bastığında yarım prompt gider ve bu, gönderilene bakmadan fark edilmez.
     */
    const val MAX_ENTRY_LEN = 8000

    /** Dosya bu satır sayısını aşınca yeniden yazılır (append biriktirmesi). */
    private const val COMPACT_AT = MAX_ENTRIES * 3

    private val cache = HashMap<Kind, MutableList<String>>()

    /** Geçmişin anlık kopyası — en eskiden en yeniye. */
    @Synchronized
    fun entries(kind: Kind): List<String> = ArrayList(load(kind))

    /**
     * Yeni girdiyi ekler. Art arda aynı girdi tekrar yazılmaz (shell davranışı);
     * boş ve çok uzun girdiler atlanır.
     */
    @Synchronized
    fun add(kind: Kind, raw: String) {
        val entry = raw.trim()
        if (entry.isEmpty() || entry.length > MAX_ENTRY_LEN) return
        val list = load(kind)
        if (list.lastOrNull() == entry) return
        list.add(entry)
        while (list.size > MAX_ENTRIES) list.removeAt(0)
        persist(kind, entry)
    }

    /** Test/temizlik: belleği ve dosyayı sıfırlar. */
    @Synchronized
    fun clear(kind: Kind) {
        cache.remove(kind)
        val path = pathFor(kind) ?: return
        runCatching { Files.deleteIfExists(path) }
    }

    // ── biçim ────────────────────────────────────────────────────────────
    // Her girdi TEK satır: çok satırlı prompt'un satır sonları kaçırılır,
    // yoksa bir prompt dosyada birden çok geçmiş girdisi gibi okunurdu.

    fun encode(entry: String): String =
        entry.replace("\\", "\\\\").replace("\n", "\\n").replace("\r", "")

    fun decode(line: String): String {
        val sb = StringBuilder(line.length)
        var i = 0
        while (i < line.length) {
            val c = line[i]
            if (c == '\\' && i + 1 < line.length) {
                when (line[i + 1]) {
                    'n' -> { sb.append('\n'); i += 2; continue }
                    '\\' -> { sb.append('\\'); i += 2; continue }
                }
            }
            sb.append(c)
            i++
        }
        return sb.toString()
    }

    /** Dosyadan okunan satırları geçmiş listesine çevirir (son MAX_ENTRIES). */
    fun parse(lines: List<String>): MutableList<String> {
        val out = ArrayList<String>(minOf(lines.size, MAX_ENTRIES))
        for (line in lines) {
            if (line.isBlank()) continue
            val entry = decode(line)
            if (entry.isEmpty() || entry.length > MAX_ENTRY_LEN) continue
            out.add(entry)
        }
        while (out.size > MAX_ENTRIES) out.removeAt(0)
        return out
    }

    // ── disk ─────────────────────────────────────────────────────────────

    private fun load(kind: Kind): MutableList<String> =
        cache.getOrPut(kind) {
            val path = pathFor(kind)
            if (path == null || !Files.exists(path)) return@getOrPut ArrayList()
            runCatching { parse(Files.readAllLines(path)) }.getOrElse { ArrayList() }
        }

    /**
     * Diske yazma havuz thread'inde: Enter'a basılan an EDT'dir, dosya IO
     * orada arayüzü kilitler.
     */
    private fun persist(kind: Kind, entry: String) {
        val snapshot = cache[kind]?.let { ArrayList(it) } ?: return
        ApplicationManager.getApplication().executeOnPooledThread {
            val path = pathFor(kind) ?: return@executeOnPooledThread
            runCatching {
                Files.createDirectories(path.parent)
                val lineCount =
                    if (Files.exists(path)) Files.readAllLines(path).size else 0
                if (lineCount >= COMPACT_AT) {
                    Files.write(
                        path,
                        snapshot.map { encode(it) },
                        StandardOpenOption.CREATE,
                        StandardOpenOption.TRUNCATE_EXISTING,
                    )
                } else {
                    Files.write(
                        path,
                        listOf(encode(entry)),
                        StandardOpenOption.CREATE,
                        StandardOpenOption.APPEND,
                    )
                }
            }
        }
    }

    /** Test ortamında PathManager yoksa geçmiş sessizce bellek-içi kalır. */
    private fun pathFor(kind: Kind): Path? =
        runCatching { Paths.get(PathManager.getConfigPath(), "cem", kind.fileName) }.getOrNull()
}
