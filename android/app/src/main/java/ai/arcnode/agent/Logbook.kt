package ai.arcnode.agent

import android.util.Log
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

/**
 * Tiny in-memory ring buffer of human-readable log lines, shown in the UI so
 * connection / report status can be debugged without adb. Thread-safe.
 */
object Logbook {
    private const val MAX_LINES = 400
    private val lines = ArrayDeque<String>()
    private val fmt = SimpleDateFormat("HH:mm:ss", Locale.US)

    @Synchronized
    fun log(message: String) {
        val line = "${fmt.format(Date())}  $message"
        lines.addLast(line)
        while (lines.size > MAX_LINES) lines.removeFirst()
        Log.d("ArcNode", message)
    }

    @Synchronized
    fun snapshot(): String = lines.joinToString("\n")

    @Synchronized
    fun clear() {
        lines.clear()
        log("Log cleared")
    }
}
