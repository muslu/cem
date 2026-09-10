package dev.cempw.intellij

import com.intellij.openapi.diagnostic.Logger
import java.io.File
import java.util.concurrent.TimeUnit

/**
 * cem CLI ile makine-okur konuşma katmanı.
 *
 * Neden: eklenti bir dönem `~/.cem/config.yaml`'i KENDİ yazıyordu. İki ayrı
 * yazıcı iki ayrı doğruluk kaynağı demek — ve kurulum zorunlu hale gelince
 * (setup_done) GUI'den yapılan ayar cem'i çalışabilir duruma getirmiyordu,
 * çünkü YAML'a yazmak "kurulum yapıldı" anlamına gelmiyor.
 *
 * Artık yazan tek yer cem: GUI `cem setup --thinker ... --writer ...`
 * çağırıyor, durumu da `cem status --json` ile okuyor. Aynı yüzey terminalde
 * ve betiklerde de kullanılabilir.
 */
object CemCli {
    private val LOG = Logger.getInstance(CemCli::class.java)

    /** `cem status --json` çıktısındaki araç kaydı. */
    data class Tool(
        val key: String,
        val name: String,
        val http: Boolean,
        val installed: Boolean,
        val path: String,
        val model: String,
        val effort: String,
        val endpoint: String,
        val models: List<String>,
        val efforts: List<String>,
    )

    data class Status(
        val version: String,
        val setup: Boolean,
        val lang: String,
        val thinker: String,
        val writer: String,
        val configPath: String,
        val projectConfig: Boolean,
        val tools: List<Tool>,
    ) {
        fun tool(key: String): Tool? = tools.firstOrNull { it.key == key }
    }

    class CliException(message: String) : Exception(message)

    /**
     * Durumu okur. cem bulunamazsa ya da çıktı ayrıştırılamazsa null döner —
     * çağıran taraf GUI'yi yine açabilsin (yol alanı düzeltilebilir olmalı).
     */
    fun status(workDir: String? = null): Status? {
        val (exit, out) = run(listOf("status", "--json"), workDir, timeoutSec = 20)
        if (exit != 0) {
            LOG.info("cem status --json exit=" + exit + ": " + out.take(300))
            return null
        }
        return try {
            parseStatus(out)
        } catch (e: Exception) {
            LOG.warn("cem status --json parse failed: " + out.take(300), e)
            null
        }
    }

    /**
     * Kurulumu kaydeder. Boş alanlar gönderilmez: cem verilmeyen değeri
     * mevcut config'ten koruyor.
     */
    fun setup(
        thinker: String,
        writer: String,
        thinkerModel: String = "",
        writerModel: String = "",
        thinkerEffort: String = "",
        writerEffort: String = "",
        thinkerEndpoint: String = "",
        writerEndpoint: String = "",
        workDir: String? = null,
    ) {
        val args = mutableListOf("setup", "--thinker", thinker, "--writer", writer)
        fun add(flag: String, value: String) {
            if (value.isNotBlank()) {
                args.add(flag)
                args.add(value)
            }
        }
        add("--model-thinker", thinkerModel)
        add("--model-writer", writerModel)
        add("--effort-thinker", thinkerEffort)
        add("--effort-writer", writerEffort)
        add("--endpoint-thinker", thinkerEndpoint)
        add("--endpoint-writer", writerEndpoint)

        val (exit, out) = run(args, workDir, timeoutSec = 60)
        if (exit != 0) {
            // cem hatayı tek satırda ve okunur veriyor ("✗ ollama için model
            // adı gerekli …"); olduğu gibi göstermek en faydalısı.
            throw CliException(out.trim().ifBlank { "cem setup exit " + exit })
        }
    }

    /** Hızlı mod — setup bayraklarında değil, kendi komutunda. */
    fun fast(toolKey: String, on: Boolean, workDir: String? = null) {
        val (exit, out) = run(listOf("fast", toolKey, if (on) "on" else "off"), workDir, timeoutSec = 20)
        if (exit != 0) throw CliException(out.trim().ifBlank { "cem fast exit " + exit })
    }

    /** Sunucudaki modelleri sorar (ollama/LM Studio/unsloth). */
    fun endpointModels(toolKey: String, address: String, workDir: String? = null): List<String> {
        if (address.isNotBlank()) {
            val (exit, out) = run(listOf("endpoint", toolKey, address), workDir, timeoutSec = 20)
            if (exit != 0) throw CliException(out.trim())
        }
        val (exit, out) = run(listOf("endpoint", toolKey, "--modeller"), workDir, timeoutSec = 30)
        if (exit != 0) throw CliException(out.trim())
        return out.lines()
            .map { it.trim() }
            .filter { it.isNotEmpty() && !it.startsWith("cem ") && !it.startsWith("✗") }
    }

    /**
     * Ham çalıştırma: (exit, stdout+stderr). stdin kapatılıyor — cem'in
     * interaktif sorulara düşmesi burada işe yaramaz, kilitlenir.
     */
    private fun run(args: List<String>, workDir: String?, timeoutSec: Long): Pair<Int, String> {
        val cmd = mutableListOf(CemAction.resolveCemBinary())
        cmd.addAll(args)
        return try {
            val pb = ProcessBuilder(cmd).redirectErrorStream(true)
            workDir?.let { pb.directory(File(it)) }
            val p = pb.start()
            try {
                p.outputStream.close()
            } catch (_: Exception) {
            }
            val out = p.inputStream.readBytes().toString(Charsets.UTF_8)
            if (!p.waitFor(timeoutSec, TimeUnit.SECONDS)) {
                p.destroyForcibly()
                return -1 to ("cem " + (args.firstOrNull() ?: "") + " zaman aşımına uğradı (" + timeoutSec + "s)")
            }
            p.exitValue() to out
        } catch (e: Exception) {
            -1 to (e.message ?: "cem çalıştırılamadı")
        }
    }

    /**
     * JSON ayrıştırma — elle, bağımlılık eklemeden.
     *
     * Eklenti snakeyaml dışında kütüphane taşımıyor; durum çıktısı sabit ve
     * sığ bir şema olduğu için küçük bir ayrıştırıcı yeterli. IntelliJ'in
     * kendi JSON API'lerine bağlanmıyoruz: sinceBuild 233'ten bugüne kadar
     * çalışması gerekiyor.
     */
    internal fun parseStatus(json: String): Status {
        val root = MiniJson.parse(json) as? Map<*, *> ?: throw CliException("beklenmeyen çıktı")
        root["error"]?.let { throw CliException(it.toString()) }
        val tools = (root["tools"] as? List<*> ?: emptyList<Any>()).mapNotNull { raw ->
            val m = raw as? Map<*, *> ?: return@mapNotNull null
            Tool(
                key = m.str("key"),
                name = m.str("name"),
                http = m["http"] == true,
                installed = m["installed"] == true,
                path = m.str("path"),
                model = m.str("model"),
                effort = m.str("effort"),
                endpoint = m.str("endpoint"),
                models = m.strList("models"),
                efforts = m.strList("efforts"),
            )
        }
        return Status(
            version = root.str("version"),
            setup = root["setup"] == true,
            lang = root.str("lang"),
            thinker = root.str("thinker"),
            writer = root.str("writer"),
            configPath = root.str("config_path"),
            projectConfig = root["project_config"] == true,
            tools = tools,
        )
    }

    private fun Map<*, *>.str(key: String): String = this[key]?.toString() ?: ""

    private fun Map<*, *>.strList(key: String): List<String> =
        (this[key] as? List<*>)?.map { it.toString() } ?: emptyList()
}

/** Küçük JSON okuyucu — yalnız cem'in durum şemasını çözmek için. */
internal object MiniJson {
    fun parse(s: String): Any? = Parser(s).value()

    private class Parser(val s: String) {
        var i = 0

        fun skipWhitespace() {
            while (i < s.length && s[i].isWhitespace()) i++
        }

        fun value(): Any? {
            skipWhitespace()
            if (i >= s.length) return null
            return when (s[i]) {
                '{' -> obj()
                '[' -> arr()
                '"' -> str()
                't' -> { expect("true"); true }
                'f' -> { expect("false"); false }
                'n' -> { expect("null"); null }
                else -> num()
            }
        }

        fun obj(): Map<String, Any?> {
            val m = LinkedHashMap<String, Any?>()
            i++ // {
            skipWhitespace()
            if (i < s.length && s[i] == '}') { i++; return m }
            while (i < s.length) {
                skipWhitespace()
                val k = str()
                skipWhitespace()
                if (i < s.length && s[i] == ':') i++
                m[k] = value()
                skipWhitespace()
                when {
                    i < s.length && s[i] == ',' -> i++
                    i < s.length && s[i] == '}' -> { i++; return m }
                    else -> return m
                }
            }
            return m
        }

        fun arr(): List<Any?> {
            val l = mutableListOf<Any?>()
            i++ // [
            skipWhitespace()
            if (i < s.length && s[i] == ']') { i++; return l }
            while (i < s.length) {
                l.add(value())
                skipWhitespace()
                when {
                    i < s.length && s[i] == ',' -> i++
                    i < s.length && s[i] == ']' -> { i++; return l }
                    else -> return l
                }
            }
            return l
        }

        fun str(): String {
            val sb = StringBuilder()
            if (i < s.length && s[i] == '"') i++
            while (i < s.length) {
                val c = s[i++]
                when {
                    c == '"' -> return sb.toString()
                    c == '\\' && i < s.length -> {
                        when (val e = s[i++]) {
                            'n' -> sb.append('\n')
                            't' -> sb.append('\t')
                            'r' -> sb.append('\r')
                            'b' -> sb.append('\b')
                            'f' -> sb.append('')
                            'u' -> {
                                val hex = s.substring(i, (i + 4).coerceAtMost(s.length))
                                i += hex.length
                                sb.append(hex.toInt(16).toChar())
                            }
                            else -> sb.append(e)
                        }
                    }
                    else -> sb.append(c)
                }
            }
            return sb.toString()
        }

        fun num(): Any {
            val start = i
            while (i < s.length && (s[i].isDigit() || s[i] in "-+.eE")) i++
            val t = s.substring(start, i)
            return t.toLongOrNull() ?: t.toDoubleOrNull() ?: t
        }

        fun expect(word: String) {
            if (s.startsWith(word, i)) i += word.length else i = s.length
        }
    }
}
