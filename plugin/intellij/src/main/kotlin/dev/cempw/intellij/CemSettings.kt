package dev.cempw.intellij

import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.components.PersistentStateComponent
import com.intellij.openapi.components.State
import com.intellij.openapi.components.Storage
import com.intellij.openapi.ui.ComboBox
import com.intellij.openapi.options.Configurable
import com.intellij.ui.components.JBLabel
import com.intellij.ui.components.JBTextField
import com.intellij.util.ui.FormBuilder
import javax.swing.JComponent
import javax.swing.JPanel

// NOT: @Service annotation kullanmıyoruz — plugin.xml'deki
// <applicationService> tek kayıt noktası. Çift kayıt 2024.2+ IDE'lerde
// "no interface supported" (0x80004002) hatasına yol açıyor.
@State(name = "CemSettings", storages = [Storage("cem.xml")])
class CemSettings : PersistentStateComponent<CemSettings.State> {
    data class State(
        var cemPath: String = "cem",
    )

    private var state = State()
    override fun getState() = state
    override fun loadState(s: State) {
        state = s
    }

    var cemPath: String
        get() = state.cemPath
        set(v) {
            state.cemPath = v
        }

    companion object {
        val instance: CemSettings
            get() = ApplicationManager.getApplication().getService(CemSettings::class.java)
    }
}

/**
 * Ayar paneli — kurulumu GUI'den yapmanın yolu.
 *
 * Yazma işini cem yapıyor: Apply, `cem setup --thinker ... --writer ...`
 * çalıştırıyor. Panel bir dönem `~/.cem/config.yaml`'i kendisi yazıyordu;
 * kurulum zorunlu hale gelince (setup_done) bu yetmemeye başladı — YAML'a rol
 * yazmak "kurulum yapıldı" demek değil, dolayısıyla GUI'den ayar yapan
 * kullanıcı cem'i yine çalıştıramıyordu.
 *
 * Araç listesi, modeller, effort seviyeleri ve endpoint adresleri
 * `cem status --json`'dan okunuyor: yeni bir araç cem'e eklendiğinde
 * (ollama/lmstudio/unsloth böyle geldi) panelde elle güncelleme gerekmiyor.
 */
class CemSettingsConfigurable : Configurable {
    private var panel: JPanel? = null
    private var cemPathField: JBTextField? = null
    private var statusLabel: JBLabel? = null
    private var thinkerEndpointField: JBTextField? = null
    private var writerEndpointField: JBTextField? = null
    private var status: CemCli.Status? = null
    private var thinkerCombo: ComboBox<String>? = null
    private var writerCombo: ComboBox<String>? = null
    private var thinkerModelCombo: ComboBox<String>? = null
    private var writerModelCombo: ComboBox<String>? = null
    private var thinkerEffortCombo: ComboBox<String>? = null
    private var writerEffortCombo: ComboBox<String>? = null
    private var fastCheck: javax.swing.JCheckBox? = null
    private var initial: CemConfig.State = CemConfig.State()

    override fun getDisplayName() = "cem"

    override fun createComponent(): JComponent {
        cemPathField = JBTextField(CemSettings.instance.cemPath, 40)

        initial = CemConfig.load()
        status = CemCli.status()
        val toolKeys = (status?.tools?.map { it.key } ?: CemConfig.knownTools).toTypedArray()

        statusLabel = JBLabel(statusText())
        thinkerEndpointField = JBTextField(20)
        writerEndpointField = JBTextField(20)

        thinkerCombo = ComboBox(toolKeys).apply {
            selectedItem = (status?.thinker ?: initial.thinker).ifBlank { "claude" }
            addActionListener { refreshModelCombos() }
        }
        writerCombo = ComboBox(toolKeys).apply {
            selectedItem = (status?.writer ?: initial.writer).ifBlank { "agy" }
            addActionListener { refreshModelCombos() }
        }
        thinkerModelCombo = ComboBox<String>()
        writerModelCombo = ComboBox<String>()
        thinkerEffortCombo = ComboBox<String>()
        writerEffortCombo = ComboBox<String>()
        // Hızlı mod cem tarafında varsayılan AÇIK; kutu işaretli başlar ve
        // yalnızca kapatıldığında config'e yazılır.
        fastCheck = javax.swing.JCheckBox(
            "Hızlı mod — aracın hook/izin kurallarını yüklemeden çalıştır",
            initial.fast["claude"] ?: true,
        )
        refreshModelCombos()

        val form = FormBuilder.createFormBuilder()
            .addComponent(statusLabel!!)
            .addSeparator()
            .addLabeledComponent(JBLabel("cem binary path:"), cemPathField!!, 1, false)
            .addComponent(JBLabel("Leave as 'cem' if it's on PATH. Otherwise full path."))
            .addSeparator()
            .addComponent(JBLabel("Roles — cem setup ile aynı sorular:"))
            .addLabeledComponent(JBLabel("🧠 Thinker:"), thinkerCombo!!, 1, false)
            .addLabeledComponent(JBLabel("    Model:"), thinkerModelCombo!!, 1, false)
            .addLabeledComponent(JBLabel("    Düşünme seviyesi:"), thinkerEffortCombo!!, 1, false)
            .addLabeledComponent(JBLabel("    Sunucu adresi:"), thinkerEndpointField!!, 1, false)
            .addLabeledComponent(JBLabel("✍️  Writer:"), writerCombo!!, 1, false)
            .addLabeledComponent(JBLabel("    Model:"), writerModelCombo!!, 1, false)
            .addLabeledComponent(JBLabel("    Düşünme seviyesi:"), writerEffortCombo!!, 1, false)
            .addLabeledComponent(JBLabel("    Sunucu adresi:"), writerEndpointField!!, 1, false)
            .addComponent(JBLabel("<html><i>Sunucu adresi yalnız ollama / LM Studio / unsloth için — ör. 192.168.1.10:11434. Model adı zorunlu, tahmin edilmiyor.</i></html>"))
            .addComponent(JBLabel("<html><i>Öneri: thinker <b>high</b> (planı o çıkarıyor), writer <b>low</b> (planı uyguluyor).</i></html>"))
            .addSeparator()
            .addComponent(fastCheck!!)
            .addComponent(JBLabel("<html><i>Ölçüldü: aynı görev 124s yerine 8s. Kapatırsan kendi hook'ların ve izin kuralların çalışır.</i></html>"))
            .addComponent(JBLabel("<html><i>Boş model = CLI default. Apply, <b>cem setup</b> çalıştırır — config'i yazan tek yer cem.</i></html>"))
            .addComponentFillVertically(JPanel(), 0)
            .panel
        panel = form
        return form
    }

    private val defaultLabel = "(CLI default)"

    /** Panelin üstündeki tek satır: kurulum yapılmış mı, config nerede. */
    private fun statusText(): String {
        val st = status ?: return "<html>⚠ <b>cem çalıştırılamadı</b> — aşağıdaki yolu düzeltip Apply'a bas.</html>"
        return if (st.setup) {
            "<html>✓ kurulum tamam · cem ${st.version} · ${st.configPath}</html>"
        } else {
            "<html>⚠ <b>kurulum yapılmadı</b> — cem çalışmaz. Rolleri seçip Apply'a bas.</html>"
        }
    }

    /** Aracın cem'den gelen kaydı (yoksa null: eski YAML listelerine düşülür). */
    private fun toolOf(key: String): CemCli.Tool? = status?.tool(key)

    private fun refreshModelCombos() {
        val tk = thinkerCombo?.selectedItem as? String ?: return
        val wk = writerCombo?.selectedItem as? String ?: return

        // Sunucu adresi yalnız HTTP araçlarda anlamlı; diğerlerinde alan
        // kapatılıyor ki kullanıcı doldurup hiçbir etkisi olmadığını
        // sanmasın.
        thinkerEndpointField?.let { f ->
            val t = toolOf(tk)
            f.isEnabled = t?.http == true
            f.text = if (t?.http == true) t.endpoint else ""
            f.toolTipText = if (t?.http == true) "ör. 192.168.1.10:11434" else "bu araç bir sunucu değil"
        }
        writerEndpointField?.let { f ->
            val t = toolOf(wk)
            f.isEnabled = t?.http == true
            f.text = if (t?.http == true) t.endpoint else ""
            f.toolTipText = if (t?.http == true) "ör. 127.0.0.1:1234" else "bu araç bir sunucu değil"
        }
        thinkerModelCombo?.let { it.model = javax.swing.DefaultComboBoxModel(modelsFor(tk).toTypedArray()) }
        writerModelCombo?.let { it.model = javax.swing.DefaultComboBoxModel(modelsFor(wk).toTypedArray()) }
        thinkerModelCombo?.selectedItem = initial.models[tk]?.ifBlank { defaultLabel } ?: defaultLabel
        writerModelCombo?.selectedItem  = initial.models[wk]?.ifBlank { defaultLabel } ?: defaultLabel

        thinkerEffortCombo?.let {
            it.model = javax.swing.DefaultComboBoxModel(effortsFor(tk).toTypedArray())
            it.selectedItem = initial.efforts[tk]?.ifBlank { defaultLabel } ?: defaultLabel
            it.isEnabled = effortsFor(tk).size > 1
        }
        writerEffortCombo?.let {
            it.model = javax.swing.DefaultComboBoxModel(effortsFor(wk).toTypedArray())
            it.selectedItem = initial.efforts[wk]?.ifBlank { defaultLabel } ?: defaultLabel
            it.isEnabled = effortsFor(wk).size > 1
        }
        // Hızlı mod yalnızca destekleyen araç rollerden birindeyse anlamlı.
        fastCheck?.isEnabled = CemConfig.fastCapable.contains(tk) || CemConfig.fastCapable.contains(wk)
    }

    /**
     * '(CLI default)' + aracın desteklediği seviyeler.
     *
     * Liste cem'den geliyor; cem çalıştırılamadıysa eklentideki eski sabit
     * listeye düşülüyor (panel yine açılabilsin, yol alanı düzeltilebilsin).
     */
    private fun effortsFor(tool: String): List<String> {
        val list = mutableListOf(defaultLabel)
        list += toolOf(tool)?.efforts ?: CemConfig.effortsByTool[tool] ?: emptyList()
        return list
    }

    /** '(CLI default)' + aracın bilinen modelleri (cem'den). */
    private fun modelsFor(tool: String): List<String> {
        val list = mutableListOf(defaultLabel)
        list += toolOf(tool)?.models ?: CemConfig.modelsByTool[tool] ?: emptyList()
        // HTTP sunucularda model adı zorunlu: '(CLI default)' anlamsız, ama
        // listede kalıyor ki kullanıcı elle yazabilsin — cem boş modeli
        // reddedip ne yapılacağını söylüyor.
        return list
    }

    /** Display label ↔ saklanacak değer (default → "" boş string). */
    private fun normalizeModel(displayed: String?): String =
        if (displayed.isNullOrBlank() || displayed == defaultLabel) "" else displayed

    private fun currentUiState(): CemConfig.State {
        val tk = thinkerCombo?.selectedItem as? String ?: ""
        val wk = writerCombo?.selectedItem as? String ?: ""
        val tm = normalizeModel(thinkerModelCombo?.selectedItem as? String)
        val wm = normalizeModel(writerModelCombo?.selectedItem as? String)
        // Mevcut models map'ini kopyala, sadece seçili tool'ların entry'lerini güncelle.
        val models = initial.models.toMutableMap()
        if (tk.isNotBlank()) models[tk] = tm
        if (wk.isNotBlank()) models[wk] = wm

        val efforts = initial.efforts.toMutableMap()
        if (tk.isNotBlank() && CemConfig.effortsByTool.containsKey(tk)) {
            efforts[tk] = normalizeModel(thinkerEffortCombo?.selectedItem as? String)
        }
        if (wk.isNotBlank() && CemConfig.effortsByTool.containsKey(wk)) {
            efforts[wk] = normalizeModel(writerEffortCombo?.selectedItem as? String)
        }

        val fast = initial.fast.toMutableMap()
        val fastOn = fastCheck?.isSelected ?: true
        for (key in CemConfig.fastCapable) {
            if (key == tk || key == wk) fast[key] = fastOn
        }
        return CemConfig.State(tk, wk, models, efforts, fast)
    }

    override fun isModified(): Boolean {
        val pathChanged = cemPathField?.text != CemSettings.instance.cemPath
        val cur = currentUiState()
        val rolesChanged = cur.thinker != initial.thinker || cur.writer != initial.writer
        val modelsChanged = cur.models != initial.models
        val effortsChanged = cur.efforts != initial.efforts
        val fastChanged = cur.fast != initial.fast
        // Sunucu adresi değişimi: alan yalnız HTTP araçta etkin.
        val endpointChanged =
            (thinkerEndpointField?.takeIf { it.isEnabled }?.text?.trim() ?: "") !=
                (toolOf(cur.thinker)?.endpoint ?: "") ||
                (writerEndpointField?.takeIf { it.isEnabled }?.text?.trim() ?: "") !=
                (toolOf(cur.writer)?.endpoint ?: "")
        val setupMissing = status?.setup == false
        return pathChanged || rolesChanged || modelsChanged || effortsChanged ||
            fastChanged || endpointChanged || setupMissing
    }

    /**
     * Apply — kurulumu cem'e yaptırır.
     *
     * Eskiden YAML doğrudan yazılıyordu; artık `cem setup` çağırılıyor. Fark
     * sadece temizlik değil: setup_done bayrağını ancak cem koyabiliyor ve
     * doğrulamayı (bilinmeyen araç, HTTP araçta eksik model, geçersiz effort)
     * cem yapıyor — hata mesajı kullanıcıya olduğu gibi gösteriliyor.
     */
    override fun apply() {
        CemSettings.instance.cemPath = cemPathField?.text ?: "cem"
        val cur = currentUiState()
        if (cur.thinker.isBlank() || cur.writer.isBlank()) return

        try {
            CemCli.setup(
                thinker = cur.thinker,
                writer = cur.writer,
                thinkerModel = cur.models[cur.thinker] ?: "",
                writerModel = cur.models[cur.writer] ?: "",
                thinkerEffort = cur.efforts[cur.thinker] ?: "",
                writerEffort = cur.efforts[cur.writer] ?: "",
                thinkerEndpoint = thinkerEndpointField?.takeIf { it.isEnabled }?.text?.trim() ?: "",
                writerEndpoint = writerEndpointField?.takeIf { it.isEnabled }?.text?.trim() ?: "",
            )
            // Hızlı mod setup bayraklarında değil; kendi komutu var.
            val fastOn = fastCheck?.isSelected ?: true
            for (key in setOf(cur.thinker, cur.writer)) {
                if (CemConfig.fastCapable.contains(key) && (initial.fast[key] ?: true) != fastOn) {
                    CemCli.fast(key, fastOn)
                }
            }
            status = CemCli.status()
            statusLabel?.text = statusText()
            initial = CemConfig.load()
        } catch (e: Exception) {
            com.intellij.openapi.ui.Messages.showErrorDialog(
                e.message ?: "cem setup başarısız",
                "cem",
            )
        }
    }

    override fun reset() {
        cemPathField?.text = CemSettings.instance.cemPath
        initial = CemConfig.load()
        status = CemCli.status()
        statusLabel?.text = statusText()
        thinkerCombo?.selectedItem = initial.thinker.ifBlank { "claude" }
        writerCombo?.selectedItem  = initial.writer.ifBlank { "agy" }
        fastCheck?.isSelected = initial.fast["claude"] ?: true
        refreshModelCombos()
    }

    override fun disposeUIResources() {
        panel = null
        cemPathField = null
        thinkerCombo = null
        writerCombo = null
        thinkerModelCombo = null
        writerModelCombo = null
        thinkerEffortCombo = null
        writerEffortCombo = null
        fastCheck = null
        statusLabel = null
        thinkerEndpointField = null
        writerEndpointField = null
    }
}
