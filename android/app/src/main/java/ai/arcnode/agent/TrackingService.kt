package ai.arcnode.agent

import android.app.ActivityManager
import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.app.usage.UsageEvents
import android.app.usage.UsageStatsManager
import android.content.Context
import android.content.Intent
import android.os.BatteryManager
import android.os.Build
import android.os.IBinder
import org.json.JSONObject
import java.util.concurrent.atomic.AtomicBoolean

/**
 * Foreground service that mirrors the desktop agent on Android: it polls
 * UsageStatsManager for foreground-app changes and periodically samples
 * memory / battery, then reports both to the gateway as TimelineEvents.
 */
class TrackingService : Service() {

    private val running = AtomicBoolean(false)
    private var worker: Thread? = null

    private lateinit var prefs: Prefs
    private lateinit var client: GatewayClient

    override fun onCreate() {
        super.onCreate()
        prefs = Prefs(this)
        client = GatewayClient(prefs.gatewayUrl, prefs.token, prefs.deviceId)
        startForeground(NOTIF_ID, buildNotification())
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        if (running.compareAndSet(false, true)) {
            worker = Thread { runLoop() }.also { it.start() }
        }
        return START_STICKY
    }

    override fun onDestroy() {
        running.set(false)
        worker?.interrupt()
        super.onDestroy()
    }

    override fun onBind(intent: Intent?): IBinder? = null

    private fun runLoop() {
        val deviceName = prefs.deviceName.ifEmpty { Build.MODEL ?: "android-device" }
        client.initDevice(deviceName, "android", systemInfo())

        var lastPackage = ""
        var lastQuery = System.currentTimeMillis() - 60_000L

        while (running.get()) {
            val intervalMs = prefs.sampleIntervalSecs * 1000L
            val batch = mutableListOf<JSONObject>()

            val now = System.currentTimeMillis()
            val fg = latestForegroundPackage(lastQuery, now)
            lastQuery = now
            if (fg != null && fg != lastPackage) {
                lastPackage = fg
                batch += GatewayClient.event(
                    deviceId = prefs.deviceId,
                    eventType = "ForegroundChange",
                    timestamp = now / 1000,
                    category = null,
                    metadata = JSONObject()
                        .put("process_name", fg)
                        .put("window_title", appLabel(fg))
                        .put("pid", 0),
                )
            }

            batch += GatewayClient.event(
                deviceId = prefs.deviceId,
                eventType = "SystemSample",
                timestamp = now / 1000,
                category = null,
                metadata = systemSample(),
            )

            client.sendEvents(batch)

            try {
                Thread.sleep(intervalMs)
            } catch (_: InterruptedException) {
                break
            }
        }
    }

    /** Most recent package moved to foreground within [from, to). */
    @Suppress("DEPRECATION") // MOVE_TO_FOREGROUND == ACTIVITY_RESUMED; kept for minSdk 24
    private fun latestForegroundPackage(from: Long, to: Long): String? {
        val usm = getSystemService(Context.USAGE_STATS_SERVICE) as? UsageStatsManager ?: return null
        val events = usm.queryEvents(from, to)
        val ev = UsageEvents.Event()
        var pkg: String? = null
        while (events.hasNextEvent()) {
            events.getNextEvent(ev)
            if (ev.eventType == UsageEvents.Event.MOVE_TO_FOREGROUND) {
                pkg = ev.packageName
            }
        }
        return pkg
    }

    private fun appLabel(pkg: String): String = try {
        val pm = packageManager
        pm.getApplicationLabel(pm.getApplicationInfo(pkg, 0)).toString()
    } catch (_: Exception) {
        pkg
    }

    private fun systemSample(): JSONObject {
        val am = getSystemService(Context.ACTIVITY_SERVICE) as ActivityManager
        val mem = ActivityManager.MemoryInfo()
        am.getMemoryInfo(mem)
        val memPct = if (mem.totalMem > 0) {
            ((mem.totalMem - mem.availMem).toDouble() / mem.totalMem.toDouble()) * 100.0
        } else 0.0

        val bm = getSystemService(Context.BATTERY_SERVICE) as? BatteryManager
        val battery = bm?.getIntProperty(BatteryManager.BATTERY_PROPERTY_CAPACITY) ?: -1

        return JSONObject()
            .put("cpu", 0)
            .put("memory", memPct)
            .put("memory_bytes", mem.totalMem - mem.availMem)
            .put("memory_total_bytes", mem.totalMem)
            .apply { if (battery in 0..100) put("battery_pct", battery) }
    }

    private fun systemInfo(): JSONObject {
        val am = getSystemService(Context.ACTIVITY_SERVICE) as ActivityManager
        val mem = ActivityManager.MemoryInfo()
        am.getMemoryInfo(mem)
        return JSONObject()
            .put("cpu_brand", "${Build.MANUFACTURER} ${Build.HARDWARE}")
            .put("cpu_cores", Runtime.getRuntime().availableProcessors())
            .put("total_memory", mem.totalMem)
            .put("total_disk", 0)
            .put("os_name", "Android")
            .put("os_version", Build.VERSION.RELEASE ?: "")
            .put("architecture", Build.SUPPORTED_ABIS.firstOrNull() ?: "")
    }

    private fun buildNotification(): Notification {
        val nm = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                getString(R.string.channel_name),
                NotificationManager.IMPORTANCE_LOW,
            )
            nm.createNotificationChannel(channel)
        }
        val pi = PendingIntent.getActivity(
            this,
            0,
            Intent(this, MainActivity::class.java),
            PendingIntent.FLAG_IMMUTABLE,
        )
        val builder = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            Notification.Builder(this, CHANNEL_ID)
        } else {
            @Suppress("DEPRECATION")
            Notification.Builder(this)
        }
        return builder
            .setContentTitle(getString(R.string.app_name))
            .setContentText(getString(R.string.notif_text))
            .setSmallIcon(R.drawable.ic_stat)
            .setContentIntent(pi)
            .setOngoing(true)
            .build()
    }

    companion object {
        private const val CHANNEL_ID = "arcnode_tracking"
        private const val NOTIF_ID = 1001

        fun start(context: Context) {
            val intent = Intent(context, TrackingService::class.java)
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                context.startForegroundService(intent)
            } else {
                context.startService(intent)
            }
        }

        fun stop(context: Context) {
            context.stopService(Intent(context, TrackingService::class.java))
        }
    }
}
