package ai.arcnode.agent

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent

/** Restart tracking after a reboot if the agent has been configured. */
class BootReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action != Intent.ACTION_BOOT_COMPLETED) return
        if (Prefs(context).isConfigured) {
            TrackingService.start(context)
        }
    }
}
