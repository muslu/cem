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

class CemSettingsConfigurable : Configurable {
    private var panel: JPanel? = null
    private var cemPathField: JBTextField? = null
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
        thinkerCombo = ComboBox(CemConfig.knownTools.toTypedArray()).apply {
            selectedItem = initial.thinker.ifBlank { "claude" }
            addActionListener { refreshModelCombos() }
        }
        writerCombo = ComboBox(CemConfig.knownTools.toTypedArray()).apply {
            selectedItem = initial.writer.ifBlank { "agy" }
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
            .addLabeledComponent(JBLabel("cem binary path:"), cemPathField!!, 1, false)
            .addComponent(JBLabel("Leave as 'cem' if it's on PATH. Otherwise full path."))
            .addSeparator()
            .addComponent(JBLabel("Roles — cem setup ile aynı sorular:"))
            .addLabeledComponent(JBLabel("🧠 Thinker:"), thinkerCombo!!, 1, false)
            .addLabeledComponent(JBLabel("    Model:"), thinkerModelCombo!!, 1, false)
            .addLabeledComponent(JBLabel("    Düşünme seviyesi:"), thinkerEffortCombo!!, 1, false)
            .addLabeledComponent(JBLabel("✍️  Writer:"), writerCombo!!, 1, false)
            .addLabeledComponent(JBLabel("    Model:"), writerModelCombo!!, 1, false)
            .addLabeledComponent(JBLabel("    Düşünme seviyesi:"), writerEffortCombo!!, 1, false)
            .addComponent(JBLabel("<html><i>Öneri: thinker <b>high</b> (planı o çıkarıyor), writer <b>low</b> (planı uyguluyor).</i></html>"))
            .addSeparator()
            .addComponent(fastCheck!!)
            .addComponent(JBLabel("<html><i>Ölçüldü: aynı görev 124s yerine 8s. Kapatırsan kendi hook'ların ve izin kuralların çalışır.</i></html>"))
            .addComponent(JBLabel("<html><i>Boş model = CLI default. Apply ile ~/.cem/config.yaml güncellenir.</i></html>"))
            .addComponentFillVertically(JPanel(), 0)
            .panel
        panel = form
        return form
    }

    private val defaultLabel = "(CLI default)"

    private fun refreshModelCombos() {
        val tk = thinkerCombo?.selectedItem as? String ?: return
        val wk = writerCombo?.selectedItem as? String ?: return
        thinkerModelCombo?.let { it.model = javax.swing.DefaultComboBoxModel(modelsFor(tk).toTypedArray()) }
        writerModelCombo?.let { it.model = javax.swing.DefaultComboBoxModel(modelsFor(wk).toTypedArray()) }
        thinkerModelCombo?.selectedItem = initial.models[tk]?.ifBlank { defaultLabel } ?: defaultLabel
        writerModelCombo?.selectedItem  = initial.models[wk]?.ifBlank { defaultLabel } ?: defaultLabel

        thinkerEffortCombo?.let {
            it.model = javax.swing.DefaultComboBoxModel(effortsFor(tk).toTypedArray())
            it.selectedItem = initial.efforts[tk]?.ifBlank { defaultLabel } ?: defaultLabel
            it.isEnabled = CemConfig.effortsByTool.containsKey(tk)
        }
        writerEffortCombo?.let {
            it.model = javax.swing.DefaultComboBoxModel(effortsFor(wk).toTypedArray())
            it.selectedItem = initial.efforts[wk]?.ifBlank { defaultLabel } ?: defaultLabel
            it.isEnabled = CemConfig.effortsByTool.containsKey(wk)
        }
        // Hızlı mod yalnızca destekleyen araç rollerden birindeyse anlamlı.
        fastCheck?.isEnabled = CemConfig.fastCapable.contains(tk) || CemConfig.fastCapable.contains(wk)
    }

    /** '(CLI default)' + aracın desteklediği seviyeler; desteklemiyorsa sadece default. */
    private fun effortsFor(tool: String): List<String> {
        val list = mutableListOf(defaultLabel)
        list += CemConfig.effortsByTool[tool] ?: emptyList()
        return list
    }

    /** '(CLI default)' (default) + tool'un bilinen modelleri. */
    private fun modelsFor(tool: String): List<String> {
        val list = mutableListOf(defaultLabel)
        list += CemConfig.modelsByTool[tool] ?: emptyList()
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
        return pathChanged || rolesChanged || modelsChanged || effortsChanged || fastChanged
    }

    override fun apply() {
        CemSettings.instance.cemPath = cemPathField?.text ?: "cem"
        val cur = currentUiState()
        if (cur.thinker.isNotBlank() && cur.writer.isNotBlank()) {
            try {
                CemConfig.save(cur)
                initial = cur
            } catch (e: Exception) {
                com.intellij.openapi.ui.Messages.showErrorDialog(
                    "~/.cem/config.yaml yazılamadı: ${e.message}",
                    "cem",
                )
            }
        }
    }

    override fun reset() {
        cemPathField?.text = CemSettings.instance.cemPath
        initial = CemConfig.load()
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
    }
}
