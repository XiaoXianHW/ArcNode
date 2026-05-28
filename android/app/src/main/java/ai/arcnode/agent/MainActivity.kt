package ai.arcnode.agent

import android.Manifest
import android.app.AppOpsManager
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import android.os.Process
import android.provider.Settings
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import androidx.core.app.ActivityCompat
import ai.arcnode.agent.databinding.ActivityMainBinding

class MainActivity : AppCompatActivity() {

    private lateinit var binding: ActivityMainBinding
    private lateinit var prefs: Prefs

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityMainBinding.inflate(layoutInflater)
        setContentView(binding.root)
        prefs = Prefs(this)

        binding.gatewayUrl.setText(prefs.gatewayUrl)
        binding.token.setText(prefs.token)
        binding.deviceName.setText(prefs.deviceName.ifEmpty { Build.MODEL ?: "" })
        binding.interval.setText(prefs.sampleIntervalSecs.toString())

        binding.saveButton.setOnClickListener { save() }
        binding.usageAccessButton.setOnClickListener {
            startActivity(Intent(Settings.ACTION_USAGE_ACCESS_SETTINGS))
        }
        binding.startButton.setOnClickListener { startTracking() }
        binding.stopButton.setOnClickListener {
            TrackingService.stop(this)
            toast(getString(R.string.stopped))
        }
    }

    override fun onResume() {
        super.onResume()
        binding.usageStatus.text = getString(
            if (hasUsageAccess()) R.string.usage_granted else R.string.usage_missing,
        )
    }

    private fun save() {
        val url = binding.gatewayUrl.text.toString().trim()
        val token = binding.token.text.toString().trim()
        if (url.isEmpty() || token.isEmpty()) {
            toast(getString(R.string.need_url_token))
            return
        }
        prefs.gatewayUrl = url
        prefs.token = token
        prefs.deviceName = binding.deviceName.text.toString().trim()
        prefs.sampleIntervalSecs = binding.interval.text.toString().toIntOrNull() ?: 60
        toast(getString(R.string.saved))
    }

    private fun startTracking() {
        if (!prefs.isConfigured) {
            toast(getString(R.string.need_url_token))
            return
        }
        if (!hasUsageAccess()) {
            toast(getString(R.string.usage_missing))
            startActivity(Intent(Settings.ACTION_USAGE_ACCESS_SETTINGS))
            return
        }
        requestNotificationPermissionIfNeeded()
        TrackingService.start(this)
        toast(getString(R.string.started))
    }

    private fun hasUsageAccess(): Boolean {
        val appOps = getSystemService(Context.APP_OPS_SERVICE) as AppOpsManager
        val mode = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            appOps.unsafeCheckOpNoThrow(
                AppOpsManager.OPSTR_GET_USAGE_STATS,
                Process.myUid(),
                packageName,
            )
        } else {
            @Suppress("DEPRECATION")
            appOps.checkOpNoThrow(
                AppOpsManager.OPSTR_GET_USAGE_STATS,
                Process.myUid(),
                packageName,
            )
        }
        return mode == AppOpsManager.MODE_ALLOWED
    }

    private fun requestNotificationPermissionIfNeeded() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU &&
            checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED
        ) {
            ActivityCompat.requestPermissions(
                this,
                arrayOf(Manifest.permission.POST_NOTIFICATIONS),
                1,
            )
        }
    }

    private fun toast(msg: String) = Toast.makeText(this, msg, Toast.LENGTH_SHORT).show()
}
