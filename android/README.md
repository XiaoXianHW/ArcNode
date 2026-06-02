# ArcNode Android Agent

A lightweight Kotlin client that reports your phone's **foreground app** and
**system resources** (memory / battery) to an ArcNode gateway, mirroring the
desktop agent over the same REST surface (`/api/v1/init` + `/api/v1/events`).

## How it works

- A foreground `Service` polls `UsageStatsManager` for the most recent
  `MOVE_TO_FOREGROUND` app and emits a `ForegroundChange` event when it changes.
- Every *N* seconds it also emits a `SystemSample` (memory %, battery %).
- Events are POSTed to the gateway with a `Bearer` token, exactly like the
  Rust agent. No third-party networking dependency is used (`HttpURLConnection`).

## Build

Requires the Android SDK (platform 34, build-tools 34.0.0). With a local SDK:

```bash
echo "sdk.dir=/path/to/Android/Sdk" > local.properties
./gradlew :app:assembleDebug      # -> app/build/outputs/apk/debug/app-debug.apk
```

Or just open the `android/` folder in Android Studio.

## Use

1. Install the APK on the phone.
2. Open **ArcNode Agent**, enter the gateway URL (e.g. `http://192.168.1.10:8080`),
   the bearer token, a device name and a sample interval, then tap **Save**.
3. Tap **Grant usage access** and enable ArcNode Agent under
   *Settings → Usage access* (required to read the foreground app).
4. Tap **Start tracking**. A persistent notification confirms the service is
   running; the device then shows up in the gateway like any desktop agent.

The bottom **Logs** panel shows live status — device init result, detected
foreground apps, and each report's event count / HTTP status — so you can
confirm connectivity without adb.

## Notes

- `PACKAGE_USAGE_STATS` is a special access permission and must be granted from
  system settings — it cannot be requested at runtime.
- Tracking restarts automatically after reboot if the agent is configured.
- Android does not expose per-app CPU usage to third-party apps, so `cpu` is
  reported as `0`; memory and battery are populated.
