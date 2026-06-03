package ai.arcnode.agent

import org.json.JSONArray
import org.json.JSONObject
import java.io.BufferedReader
import java.net.HttpURLConnection
import java.net.URL
import java.util.UUID

/**
 * Minimal HTTP client that mirrors the Rust agent's contract with the gateway:
 *   POST /api/v1/init    — register / refresh device metadata
 *   POST /api/v1/events  — push a batch of TimelineEvents
 *
 * Uses HttpURLConnection so the app pulls in no heavy networking dependency.
 */
class GatewayClient(
    private val baseUrl: String,
    private val token: String,
    private val deviceId: String,
) {
    fun initDevice(name: String, platform: String, systemInfo: JSONObject): Boolean {
        val body = JSONObject()
            .put("device_id", deviceId)
            .put("name", name)
            .put("platform", platform)
            .put("system_info", systemInfo)
        return post("/api/v1/init", body.toString())
    }

    /** events is a list of objects already shaped like a TimelineEvent. */
    fun sendEvents(events: List<JSONObject>): Boolean {
        if (events.isEmpty()) return true
        val arr = JSONArray()
        events.forEach { arr.put(it) }
        val body = JSONObject()
            .put("device_id", deviceId)
            .put("events", arr)
        return post("/api/v1/events", body.toString())
    }

    private fun post(path: String, json: String): Boolean {
        var conn: HttpURLConnection? = null
        return try {
            conn = (URL("$baseUrl$path").openConnection() as HttpURLConnection).apply {
                requestMethod = "POST"
                connectTimeout = 15000
                readTimeout = 30000
                doOutput = true
                setRequestProperty("Content-Type", "application/json")
                setRequestProperty("Authorization", "Bearer $token")
            }
            conn.outputStream.use { it.write(json.toByteArray()) }
            val code = conn.responseCode
            if (code in 200..299) {
                Logbook.log("POST $path -> $code OK")
                true
            } else {
                val err = conn.errorStream?.bufferedReader()?.use(BufferedReader::readText)
                Logbook.log("POST $path -> $code FAIL ${err.orEmpty().take(120)}")
                false
            }
        } catch (e: Exception) {
            Logbook.log("POST $path error: ${e.message}")
            false
        } finally {
            conn?.disconnect()
        }
    }

    companion object {
        /** Build a TimelineEvent matching the gateway's expected JSON shape. */
        fun event(
            deviceId: String,
            eventType: String,
            timestamp: Long,
            category: String?,
            metadata: JSONObject,
        ): JSONObject {
            val obj = JSONObject()
                .put("event_id", UUID.randomUUID().toString())
                .put("device_id", deviceId)
                .put("timestamp", timestamp)
                .put("event_type", eventType)
                .put("metadata", metadata)
            if (category != null) obj.put("category", category)
            return obj
        }
    }
}
