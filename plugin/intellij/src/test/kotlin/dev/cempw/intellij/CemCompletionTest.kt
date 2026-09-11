package dev.cempw.intellij

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.rules.TemporaryFolder
import java.io.File

/**
 * Tab tamamlamanın saf tarafı: sözcük ayırma, dosya/dizin/komut adayları,
 * ortak önek ve listeleme. Gerçek dosya sistemi (geçici dizin) kullanılır,
 * PATH sahte bir dizinle verilir — makinenin PATH'ine bağlı test kırılgan olur.
 */
class CemCompletionTest {

    @get:Rule
    val tmp = TemporaryFolder()

    private lateinit var cwd: File
    private lateinit var bin: File

    @Before
    fun kur() {
        cwd = tmp.newFolder("proje")
        File(cwd, "fetch.py").writeText("")
        File(cwd, "feed.py").writeText("")
        File(cwd, "main.go").writeText("")
        File(cwd, ".gitignore").writeText("")
        File(cwd, "docs").mkdir()
        File(cwd, "docs/README.md").writeText("")
        bin = tmp.newFolder("bin")
        File(bin, "python3").apply { writeText(""); setExecutable(true) }
        File(bin, "pytest").apply { writeText(""); setExecutable(true) }
        File(bin, "python3-notes.txt").writeText("")   // çalıştırılabilir değil
    }

    private fun tamamla(text: String, caret: Int = text.length) =
        CemCompletion.complete(text, caret, cwd, listOf(bin))

    @Test
    fun `tek dosya adayi tamamlanir ve bosluk eklenir`() {
        val r = tamamla("python3 fet")
        assertEquals("python3 fetch.py ", r.text)
        assertEquals(r.text.length, r.caret)
        assertTrue(r.candidates.isEmpty())
    }

    @Test
    fun `tek dizin adayi bolu ile biter`() {
        val r = tamamla("cat do")
        assertEquals("cat docs/", r.text)
    }

    @Test
    fun `dizin icindeki dosya tamamlanir`() {
        val r = tamamla("cat docs/RE")
        assertEquals("cat docs/README.md ", r.text)
    }

    @Test
    fun `cok aday ortak oneke kadar tamamlanir`() {
        val r = tamamla("python3 f")
        assertEquals("python3 fe", r.text)
        assertTrue("ilerleme varken liste basılmaz", r.candidates.isEmpty())
    }

    @Test
    fun `ilerleme yoksa adaylar listelenir`() {
        val r = tamamla("python3 fe")
        assertEquals("python3 fe", r.text)
        assertEquals(listOf("feed.py", "fetch.py"), r.candidates)
    }

    @Test
    fun `listeden secilen aday sozcugun yerine yazilir`() {
        // Kutu "python3 fe" iken listeden "fetch.py" seçilince kutu değişir;
        // seçim ortadaki sözcüğe de yazılabilmeli (sonrası korunur).
        val r = tamamla("python3 fe")
        val picked = r.pick(r.options.first { it.display == "fetch.py" })
        assertEquals("python3 fetch.py ", picked.text)
        assertEquals(picked.text.length, picked.caret)

        val mid = tamamla("python3 fe --x", 10)
        val p2 = mid.pick(mid.options.first { it.display == "feed.py" })
        assertEquals("python3 feed.py  --x", p2.text)
        assertEquals("python3 feed.py ".length, p2.caret)

        // Dizin seçilince boşluk değil `/` gelir.
        val d = tamamla("ls ", 3)
        val dirOpt = d.options.first { it.isDir }
        assertTrue(d.pick(dirOpt).text.endsWith("/"))
    }

    @Test
    fun `gizli dosya ancak nokta yazilinca gelir`() {
        assertTrue(tamamla("cat gi").candidates.isEmpty())
        assertEquals("cat gi", tamamla("cat gi").text)
        assertEquals("cat .gitignore ", tamamla("cat .gi").text)
    }

    @Test
    fun `komut konumunda PATH taranir ve calistirilamayan atlanir`() {
        val r = tamamla("pyth")
        assertEquals("python3 ", r.text)
    }

    @Test
    fun `komut konumunda birden cok aday listelenir`() {
        // "py" → ortak önek "pyt"e ilerler; ikinci Tab'da liste gelir.
        assertEquals("pyt", tamamla("py").text)
        val r = tamamla("pyt")
        assertEquals("pyt", r.text)
        assertEquals(listOf("pytest", "python3"), r.candidates)
    }

    @Test
    fun `boru ve ampersand sonrasi yine komut konumu`() {
        assertEquals("ls | pytest ", tamamla("ls | pyte").text)
        assertEquals("go build && pytest ", tamamla("go build && pyte").text)
        assertEquals("sudo python3 ", tamamla("sudo pyth").text)
        assertEquals("FOO=1 python3 ", tamamla("FOO=1 pyth").text)
    }

    @Test
    fun `komut konumunda bolu varsa dosya tamamlanir`() {
        assertEquals("./main.go ", tamamla("./ma").text)
    }

    @Test
    fun `imlec ortadayken yalniz oradaki sozcuk degisir`() {
        val text = "python3 fet --verbose"
        val r = tamamla(text, caret = "python3 fet".length)
        assertEquals("python3 fetch.py  --verbose", r.text)
        assertEquals("python3 fetch.py ".length, r.caret)
    }

    @Test
    fun `aday yoksa metin degismez`() {
        val r = tamamla("cat zzz")
        assertEquals("cat zzz", r.text)
        assertTrue(r.candidates.isEmpty())
    }

    @Test
    fun `bosluklu ad kacirilir`() {
        File(cwd, "Yeni Dosya.txt").writeText("")
        assertEquals("cat Yeni\\ Dosya.txt ", tamamla("cat Ye").text)
        // Kaçırılmış sözcük yeniden tamamlanınca da tek sözcük sayılır.
        assertEquals("cat Yeni\\ Dosya.txt ", tamamla("cat Yeni\\ Do").text)
    }

    @Test
    fun `ortak onek`() {
        assertEquals("fe", CemCompletion.commonPrefix(listOf("feed", "fetch")))
        assertEquals("", CemCompletion.commonPrefix(listOf("a", "b")))
        assertEquals("", CemCompletion.commonPrefix(emptyList()))
    }
}
