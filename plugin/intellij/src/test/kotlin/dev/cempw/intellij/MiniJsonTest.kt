package dev.cempw.intellij

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * `cem status --json` çıktısını çözen küçük ayrıştırıcının testleri.
 *
 * Neden test: panel araç listesini, modelleri ve endpoint adresini buradan
 * alıyor. Ayrıştırma sessizce boş dönerse GUI "hiç araç yok" gibi görünür ve
 * kullanıcı kurulumu yapamaz.
 */
class MiniJsonTest {
    @Test
    fun `nesne ve dizi cozulur`() {
        val json = """
            {
              "setup": true,
              "thinker": "gpt",
              "tools": [
                {"key": "claude", "http": false, "models": ["opus", "sonnet"]},
                {"key": "ollama", "http": true, "endpoint": "http://127.0.0.1:11434"}
              ]
            }
        """.trimIndent()
        @Suppress("UNCHECKED_CAST")
        val root = MiniJson.parse(json) as Map<String, Any?>
        assertEquals(true, root["setup"])
        assertEquals("gpt", root["thinker"])

        @Suppress("UNCHECKED_CAST")
        val tools = root["tools"] as List<Map<String, Any?>>
        assertEquals(2, tools.size)
        assertEquals("claude", tools[0]["key"])
        assertEquals(false, tools[0]["http"])
        assertEquals(listOf("opus", "sonnet"), tools[0]["models"])
        assertEquals("http://127.0.0.1:11434", tools[1]["endpoint"])
    }

    @Test
    fun `kacisli karakterler ve unicode`() {
        @Suppress("UNCHECKED_CAST")
        val root = MiniJson.parse("""{"a":"satır\nsonu","b":"tirnak: \""}""") as Map<String, Any?>
        assertEquals("satır\nsonu", root["a"])
        assertEquals("tirnak: \"", root["b"])
    }

    @Test
    fun `bos ve bozuk girdi patlamaz`() {
        // Ayrıştırıcı istisna atarsa panel hiç açılmıyordu; boş/kırık girdide
        // null ya da eksik veri dönmesi yeterli.
        assertEquals(null, MiniJson.parse(""))
        val yarim = MiniJson.parse("""{"a": [1, 2""")
        assertTrue(yarim is Map<*, *>)
    }

    @Test
    fun `sayilar ve null`() {
        @Suppress("UNCHECKED_CAST")
        val root = MiniJson.parse("""{"n": 42, "f": 1.5, "z": null}""") as Map<String, Any?>
        assertEquals(42L, root["n"])
        assertEquals(1.5, root["f"] as Double, 0.0001)
        assertEquals(null, root["z"])
    }
}
