//! Achievement presentation DLC.
//!
//! Polls the remote gateway for achievements newly unlocked by this tenant and
//! presents them locally in a configurable style: an OS notification, a
//! Steam-style popup toast, or a plain console line.

mod popup;

use anyhow::Result;
use serde::Deserialize;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::thread;
use std::time::{Duration, SystemTime, UNIX_EPOCH};

use core::AchievementsConfig;

#[derive(Debug, Clone, Deserialize)]
pub struct Unlock {
    pub id: String,
    pub name: String,
    #[serde(default)]
    pub description: String,
    #[serde(default)]
    pub tier: String,
    #[serde(default)]
    pub points: i64,
    pub unlocked_at: i64,
}

#[derive(Deserialize)]
struct UnlocksResponse {
    unlocks: Vec<Unlock>,
}

/// Starts the poller thread. Only unlocks that happen while the agent is
/// running are presented (the cursor starts at "now"), mirroring how Steam
/// pops achievements at unlock time instead of replaying history.
pub fn start(
    gateway_url: String,
    token: String,
    cfg: AchievementsConfig,
    running: Arc<AtomicBool>,
) -> Result<()> {
    let client = reqwest::blocking::Client::builder()
        .timeout(Duration::from_secs(10))
        .build()?;
    let interval = cfg.poll_interval_secs.max(5);
    let presentation = cfg.presentation.clone();

    thread::spawn(move || {
        let mut since = now_unix();
        log::info!(
            "achievement DLC started (presentation={}, poll every {}s)",
            presentation, interval
        );
        while running.load(Ordering::SeqCst) {
            match poll(&client, &gateway_url, &token, since) {
                Ok(unlocks) => {
                    for u in unlocks {
                        since = since.max(u.unlocked_at);
                        present(&presentation, &u);
                    }
                }
                Err(e) => log::warn!("achievement poll failed: {}", e),
            }
            let mut slept = 0;
            while slept < interval && running.load(Ordering::SeqCst) {
                thread::sleep(Duration::from_secs(1));
                slept += 1;
            }
        }
    });
    Ok(())
}

fn now_unix() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_secs() as i64)
        .unwrap_or(0)
}

fn poll(
    client: &reqwest::blocking::Client,
    gateway_url: &str,
    token: &str,
    since: i64,
) -> Result<Vec<Unlock>> {
    let url = format!(
        "{}/api/v1/agent/unlocks?since={}",
        gateway_url.trim_end_matches('/'),
        since
    );
    let resp = client.get(&url).bearer_auth(token).send()?;
    if !resp.status().is_success() {
        anyhow::bail!("gateway returned {}", resp.status());
    }
    Ok(resp.json::<UnlocksResponse>()?.unlocks)
}

fn present(style: &str, u: &Unlock) {
    log_unlock(u);
    match style {
        "popup" => {
            if let Err(e) = popup::show(u) {
                log::warn!("popup failed ({}), falling back to notification", e);
                notify(u);
            }
        }
        "notification" => notify(u),
        _ => {} // "console": the log line above is the presentation
    }
}

fn log_unlock(u: &Unlock) {
    log::info!(
        "🏆 Achievement unlocked: {} — {} [{}] +{} pts",
        u.name, u.description, u.tier, u.points
    );
}

fn notify(u: &Unlock) {
    let body = if u.points > 0 {
        format!("{}  (+{} pts)", u.description, u.points)
    } else {
        u.description.clone()
    };
    if let Err(e) = notify_rust::Notification::new()
        .summary(&format!("🏆 Achievement unlocked: {}", u.name))
        .body(&body)
        .appname("ArcNode")
        .timeout(notify_rust::Timeout::Milliseconds(6000))
        .show()
    {
        log::warn!("system notification failed: {}", e);
    }
}
