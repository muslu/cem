package dev.cempw.intellij

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Geçmişin saf (IDE'siz) tarafı: satır kaçırma ve ayrıştırma.
 *
 * Kaçırma testi regresyondur: çok satırlı bir prompt dosyaya olduğu gibi
 * yazılsaydı, sonraki açılışta TEK istek birden çok geçmiş girdisi olarak
 * okunur ve ↑ ile yarım prompt geri gelirdi.
 */
class CemHistoryTest {

    @Test
    fun `cok satirli girdi tek satira kacirilir ve geri cozulur`() {
        val entry = "ilk satır\nikinci satır\\son"
        val line = CemHistory.encode(entry)
        assertTrue("kaçırılmış satır newline içermemeli", !line.contains("\n"))
        assertEquals(entry, CemHistory.decode(line))
    }

    @Test
    fun `parse bos satiri atlar ve sirayi korur`() {
        val lines = listOf("bir", "", "iki\\nüç")
        assertEquals(listOf("bir", "iki\nüç"), CemHistory.parse(lines))
    }

    @Test
    fun `parse son MAX_ENTRIES girdiyi tutar`() {
        val lines = (1..CemHistory.MAX_ENTRIES + 5).map { "istek-$it" }
        val parsed = CemHistory.parse(lines)
        assertEquals(CemHistory.MAX_ENTRIES, parsed.size)
        assertEquals("istek-6", parsed.first())
        assertEquals("istek-${CemHistory.MAX_ENTRIES + 5}", parsed.last())
    }

    @Test
    fun `asiri uzun girdi gecmise alinmaz`() {
        val uzun = "x".repeat(CemHistory.MAX_ENTRY_LEN + 1)
        assertTrue(CemHistory.parse(listOf(uzun)).isEmpty())
    }
}
