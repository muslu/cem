package dev.cempw.intellij

import org.snakeyaml.engine.v2.api.Dump
import org.snakeyaml.engine.v2.api.DumpSettings
import org.snakeyaml.engine.v2.api.Load
import org.snakeyaml.engine.v2.api.LoadSettings
import org.snakeyaml.engine.v2.common.FlowStyle
import java.io.File
import java.nio.file.Files
import java.nio.file.Path

/**
 * Plugin-side view of ~/.cem/config.yaml.
 *
 * The cem CLI manages this file authoritatively; we only mutate the
 * roles + tools.<key>.model fields and leave everything else
 * (api_keys, version, setup_done, command paths) untouched.
 */
object CemConfig {

    /** İçerik şeması; eksik alanlar opsiyonel. */
    data class State(
        var thinker: String = "",
        var writer: String = "",
        /** toolKey → model (örn. "claude" → "opus"). Boş model = CLI default. */
        var models: MutableMap<String, String> = mutableMapOf(),
        /** toolKey → düşünme seviyesi (low..max). Boş = CLI default. */
        var efforts: MutableMap<String, String> = mutableMapOf(),
        /** toolKey → hızlı mod. Yok = CLI varsayılanı (açık). */
        var fast: MutableMap<String, Boolean> = mutableMapOf(),
    )

    /** Tüm 4 tool için bilinen model önerileri. cem core'la senkron tut. */
    val knownTools: List<String> = listOf("claude", "agy", "gpt", "cursor")

    val modelsByTool: Map<String, List<String>> = mapOf(
        "claude" to listOf("opus", "sonnet", "haiku"),
        "agy"    to listOf("gemini-3-pro", "gemini-3-flash"),
        "gpt"    to listOf("gpt-5.6-terra", "gpt-5.5", "gpt-5-mini", "gpt-5"),
        "cursor" to listOf("claude-4.6", "gpt-5.2", "gemini-3-pro"),
    )

    /**
     * Düşünme seviyesi destekleyen araçlar. cem core'daki ToolMeta.Efforts ile
     * senkron tut; desteklemeyen araç listede yok (combo devre dışı kalır).
     */
    val effortsByTool: Map<String, List<String>> = mapOf(
        "claude" to listOf("low", "medium", "high", "xhigh", "max"),
        "gpt"    to listOf("low", "medium", "high", "xhigh", "max"),
    )

    /** Hızlı mod tanımlı araçlar (cem core: ToolMeta.FastArgs). */
    val fastCapable: Set<String> = setOf("claude")

    /** ~/.cem/config.yaml yolu. */
    fun configPath(): Path {
        val home = System.getProperty("user.home")
        return Path.of(home, ".cem", "config.yaml")
    }

    /** Dosyadan oku — yoksa boş State döner. */
    @Suppress("UNCHECKED_CAST")
    fun load(): State {
        val path = configPath()
        if (!Files.exists(path)) return State()
        return try {
            val text = Files.readString(path)
            val raw = Load(LoadSettings.builder().build()).loadFromString(text) as? Map<String, Any?>
                ?: return State()
            val roles = raw["roles"] as? Map<String, Any?>
            val thinker = (roles?.get("thinker") as? String).orEmpty()
            val writer  = (roles?.get("writer")  as? String).orEmpty()
            val tools = raw["tools"] as? Map<String, Any?>
            val models = mutableMapOf<String, String>()
            val efforts = mutableMapOf<String, String>()
            val fast = mutableMapOf<String, Boolean>()
            tools?.forEach { (key, value) ->
                val v = value as? Map<String, Any?>
                val m = v?.get("model") as? String
                if (!m.isNullOrBlank()) models[key] = m
                val e = v?.get("effort") as? String
                if (!e.isNullOrBlank()) efforts[key] = e
                (v?.get("fast") as? Boolean)?.let { fast[key] = it }
            }
            State(thinker, writer, models, efforts, fast)
        } catch (_: Exception) {
            State()
        }
    }

    /**
     * Kaydet. Dosya yoksa minimal bir dosya oluşturur; varsa mevcut alanları
     * (api_keys, command path'leri, version) korur, sadece roles + tools.model'i
     * günceller.
     */
    @Suppress("UNCHECKED_CAST")
    fun save(newState: State) {
        val path = configPath()
        Files.createDirectories(path.parent)

        val existing: MutableMap<String, Any?> = if (Files.exists(path)) {
            try {
                val text = Files.readString(path)
                val raw = Load(LoadSettings.builder().build()).loadFromString(text) as? Map<String, Any?>
                raw?.toMutableMap() ?: mutableMapOf()
            } catch (_: Exception) {
                mutableMapOf()
            }
        } else {
            mutableMapOf()
        }

        // roles
        val roles = (existing["roles"] as? Map<String, Any?>)?.toMutableMap() ?: mutableMapOf()
        if (newState.thinker.isNotBlank()) roles["thinker"] = newState.thinker
        if (newState.writer.isNotBlank())  roles["writer"]  = newState.writer
        existing["roles"] = roles

        // tools.<key>.model
        val tools = (existing["tools"] as? Map<String, Any?>)?.toMutableMap() ?: mutableMapOf()
        val touched = newState.models.keys + newState.efforts.keys + newState.fast.keys
        for (key in touched) {
            val current = (tools[key] as? Map<String, Any?>)?.toMutableMap() ?: mutableMapOf()

            newState.models[key]?.let { model ->
                if (model.isBlank()) current.remove("model") else current["model"] = model
            }
            newState.efforts[key]?.let { effort ->
                if (effort.isBlank()) current.remove("effort") else current["effort"] = effort
            }
            // Hızlı mod cem tarafında VARSAYILAN AÇIK ve config'te *bool:
            // alanın hiç olmaması "varsayılan" demek. Kullanıcı kapatmadıkça
            // yazmıyoruz ki CLI varsayılanı değişirse buradaki eski değer
            // onu sabitlemesin.
            newState.fast[key]?.let { on ->
                if (on) current.remove("fast") else current["fast"] = false
            }

            // command field boş kalmasın; tool kuruluysa CLI yazmıştır.
            if (current.isNotEmpty() || tools.containsKey(key)) {
                tools[key] = current
            }
        }
        if (tools.isNotEmpty()) existing["tools"] = tools

        val dumpSettings = DumpSettings.builder()
            .setDefaultFlowStyle(FlowStyle.BLOCK)
            .setIndent(2)
            .build()
        val yaml = Dump(dumpSettings).dumpToString(existing)
        Files.writeString(path, yaml)
        // 0600 — kullanıcı-only (API key'ler de aynı dosyada)
        try {
            File(path.toString()).setReadable(false, false)
            File(path.toString()).setReadable(true, true)
            File(path.toString()).setWritable(false, false)
            File(path.toString()).setWritable(true, true)
        } catch (_: Exception) {
            // Windows izin modeli farklı; sessizce geç
        }
    }
}
