package ai.arcnode.agent

import android.content.Context
import java.util.UUID

/** Lightweight persistent settings backed by SharedPreferences. */
class Prefs(context: Context) {
    private val sp = context.getSharedPreferences("arcnode", Context.MODE_PRIVATE)

    var gatewayUrl: String
        get() = sp.getString(KEY_URL, "") ?: ""
        set(value) = sp.edit().putString(KEY_URL, value.trimEnd('/')).apply()

    var token: String
        get() = sp.getString(KEY_TOKEN, "") ?: ""
        set(value) = sp.edit().putString(KEY_TOKEN, value).apply()

    var deviceName: String
        get() = sp.getString(KEY_NAME, "") ?: ""
        set(value) = sp.edit().putString(KEY_NAME, value).apply()

    var sampleIntervalSecs: Int
        get() = sp.getInt(KEY_INTERVAL, 60)
        set(value) = sp.edit().putInt(KEY_INTERVAL, value.coerceAtLeast(5)).apply()

    /** Stable per-install device id; generated once on first access. */
    val deviceId: String
        get() {
            val existing = sp.getString(KEY_DEVICE_ID, null)
            if (existing != null) return existing
            val generated = UUID.randomUUID().toString()
            sp.edit().putString(KEY_DEVICE_ID, generated).apply()
            return generated
        }

    val isConfigured: Boolean
        get() = gatewayUrl.isNotEmpty() && token.isNotEmpty()

    companion object {
        private const val KEY_URL = "gateway_url"
        private const val KEY_TOKEN = "token"
        private const val KEY_NAME = "device_name"
        private const val KEY_INTERVAL = "sample_interval"
        private const val KEY_DEVICE_ID = "device_id"
    }
}
