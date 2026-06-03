use anyhow::Result;
use log::{error, info, warn};
use serde::{Deserialize, Serialize};
use std::collections::VecDeque;
use std::fs;
use std::path::PathBuf;
use std::sync::{Arc, Mutex};
use std::thread;
use std::time::{Duration, Instant};

use crate::config::StorageConfig;
use crate::db::DbManager;
use crate::events::TimelineEvent;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EventBatch {
    pub device_id: String,
    pub events: Vec<TimelineEvent>,
}

pub trait StorageBackend: Send + Sync {
    fn insert_event(&self, event: &TimelineEvent) -> Result<()>;
    fn flush(&self) -> Result<()>;
}

/// Upper bound on how many events we keep spooled on disk while the gateway is
/// unreachable. Older events beyond this are dropped to keep disk usage bounded.
const SPOOL_MAX_EVENTS: usize = 50_000;

/// Disk-backed overflow queue for events that could not be delivered to the
/// gateway. Persisted as newline-delimited JSON so undelivered activity
/// survives an agent crash or restart and is replayed once the gateway is back.
/// Combined with the per-event `event_id`, replaying is safe: the gateway
/// deduplicates, so re-sending never double-counts.
struct Spool {
    path: PathBuf,
}

impl Spool {
    fn new(path: PathBuf) -> Self {
        Self { path }
    }

    fn load(&self) -> Vec<TimelineEvent> {
        let data = match fs::read_to_string(&self.path) {
            Ok(d) => d,
            Err(_) => return Vec::new(),
        };
        data.lines()
            .filter(|l| !l.trim().is_empty())
            .filter_map(|l| serde_json::from_str::<TimelineEvent>(l).ok())
            .collect()
    }

    fn replace(&self, events: &[TimelineEvent]) {
        if events.is_empty() {
            self.clear();
            return;
        }
        let start = events.len().saturating_sub(SPOOL_MAX_EVENTS);
        let dropped = start;
        if dropped > 0 {
            warn!("spool over {} events, dropping {} oldest", SPOOL_MAX_EVENTS, dropped);
        }
        let mut out = String::new();
        for e in &events[start..] {
            if let Ok(s) = serde_json::to_string(e) {
                out.push_str(&s);
                out.push('\n');
            }
        }
        // Write to a temp file then rename so a crash mid-write can't corrupt
        // the spool.
        let tmp = self.path.with_extension("jsonl.tmp");
        if fs::write(&tmp, out.as_bytes()).is_ok() {
            let _ = fs::rename(&tmp, &self.path);
        }
    }

    fn clear(&self) {
        let _ = fs::remove_file(&self.path);
    }
}

fn spool_path() -> PathBuf {
    if let Ok(p) = std::env::var("ARCNODE_SPOOL") {
        return PathBuf::from(p);
    }
    PathBuf::from(".arcnode-spool.jsonl")
}

/// Decides whether the in-memory buffer should be drained and shipped now,
/// trading latency for batching efficiency.
fn should_flush(
    buffer_len: usize,
    batch_size: usize,
    last_flush: &Arc<Mutex<Instant>>,
    last_event_time: &Arc<Mutex<Instant>>,
    min_flush_interval: Duration,
    max_flush_interval: Duration,
) -> bool {
    if buffer_len == 0 {
        return false;
    }
    if buffer_len >= batch_size {
        return true;
    }
    let last_flush_elapsed = last_flush.lock().map(|l| l.elapsed()).unwrap_or_default();
    let last_event_elapsed = last_event_time.lock().map(|l| l.elapsed()).unwrap_or_default();
    if buffer_len <= 5 && last_event_elapsed >= Duration::from_millis(500) {
        return true;
    }
    if last_flush_elapsed >= min_flush_interval {
        return true;
    }
    last_flush_elapsed >= max_flush_interval
}

pub struct LocalStorage {
    db_manager: DbManager,
}

impl LocalStorage {
    pub fn new(data_dir: &str) -> Result<Self> {
        let db_path = format!("{}/{}.db", data_dir, chrono::Local::now().format("%Y-%m-%d"));
        let db_manager = DbManager::new(&db_path)?;
        db_manager.init()?;
        Ok(Self { db_manager })
    }
}

impl StorageBackend for LocalStorage {
    fn insert_event(&self, event: &TimelineEvent) -> Result<()> {
        self.db_manager.insert_event(event)
    }
    
    fn flush(&self) -> Result<()> {
        Ok(())
    }
}

pub struct RemoteStorage {
    device_id: String,
    gateway_url: String,
    token: String,
    batch_size: usize,
    min_flush_interval: Duration,
    max_flush_interval: Duration,
    buffer: Arc<Mutex<VecDeque<TimelineEvent>>>,
    last_flush: Arc<Mutex<Instant>>,
    last_event_time: Arc<Mutex<Instant>>,
    client: reqwest::blocking::Client,
    spool: Arc<Mutex<Spool>>,
}

impl RemoteStorage {
    pub fn new(
        device_id: String,
        gateway_url: String,
        token: String,
        batch_size: usize,
        flush_interval_secs: u64,
    ) -> Result<Self> {
        let client = reqwest::blocking::Client::builder()
            .timeout(Duration::from_secs(30))
            .build()?;
        
        let now = Instant::now();
        let storage = Self {
            device_id: device_id.clone(),
            gateway_url: gateway_url.clone(),
            token: token.clone(),
            batch_size,
            min_flush_interval: Duration::from_secs(1),
            max_flush_interval: Duration::from_secs(flush_interval_secs.max(3)),
            buffer: Arc::new(Mutex::new(VecDeque::new())),
            last_flush: Arc::new(Mutex::new(now)),
            last_event_time: Arc::new(Mutex::new(now)),
            client,
            spool: Arc::new(Mutex::new(Spool::new(spool_path()))),
        };
        
        if let Err(e) = storage.init_device() {
            error!("Failed to initialize device: {}", e);
        }

        if let Ok(s) = storage.spool.lock() {
            let pending = s.load();
            if !pending.is_empty() {
                info!("recovered {} spooled events from previous run", pending.len());
            }
        }
        
        storage.spawn_background_flusher();
        
        Ok(storage)
    }
    
    // Owns all network delivery. Every ~1s it considers the in-memory buffer
    // plus anything previously spooled to disk, ships them as one batch, and on
    // failure persists the undelivered events to disk and backs off
    // exponentially. Low-frequency events (e.g. system samples on an otherwise
    // idle machine) are therefore never stranded, and a crash/restart or an
    // offline gateway never loses data.
    fn spawn_background_flusher(&self) {
        let buffer = self.buffer.clone();
        let last_flush = self.last_flush.clone();
        let last_event_time = self.last_event_time.clone();
        let client = self.client.clone();
        let url = self.gateway_url.clone();
        let token = self.token.clone();
        let device_id = self.device_id.clone();
        let spool = self.spool.clone();
        let batch_size = self.batch_size;
        let min_flush = self.min_flush_interval;
        let max_flush = self.max_flush_interval;
        let tick = Duration::from_secs(1);
        let max_backoff = Duration::from_secs(60);
        thread::spawn(move || {
            let mut backoff = Duration::ZERO;
            loop {
                thread::sleep(tick + backoff);

                let spooled: Vec<TimelineEvent> =
                    spool.lock().map(|s| s.load()).unwrap_or_default();
                let buf_len = buffer.lock().map(|b| b.len()).unwrap_or(0);

                let due = !spooled.is_empty()
                    || should_flush(buf_len, batch_size, &last_flush, &last_event_time, min_flush, max_flush);
                if !due {
                    backoff = Duration::ZERO;
                    continue;
                }

                // Spooled (older) events first, then the live buffer, so order
                // is preserved across a reconnect.
                let mut events = spooled;
                if let Ok(mut buf) = buffer.lock() {
                    events.extend(buf.drain(..));
                }
                if events.is_empty() {
                    backoff = Duration::ZERO;
                    continue;
                }

                match Self::post_batch(&client, &url, &token, &device_id, events.clone()) {
                    Ok(_) => {
                        if let Ok(s) = spool.lock() {
                            s.clear();
                        }
                        if let Ok(mut last) = last_flush.lock() {
                            *last = Instant::now();
                        }
                        backoff = Duration::ZERO;
                    }
                    Err(e) => {
                        error!(
                            "flush failed, spooling {} events to disk: {}",
                            events.len(),
                            e
                        );
                        if let Ok(s) = spool.lock() {
                            s.replace(&events);
                        }
                        backoff = if backoff.is_zero() {
                            Duration::from_secs(2)
                        } else {
                            (backoff * 2).min(max_backoff)
                        };
                    }
                }
            }
        });
    }
    
    fn init_device(&self) -> Result<()> {
        let device_name = hostname::get()
            .unwrap_or_else(|_| "unknown".into())
            .to_string_lossy()
            .to_string();
        let platform = std::env::consts::OS;
        
        #[derive(serde::Serialize)]
        struct InitRequest {
            device_id: String,
            name: String,
            platform: String,
            system_info: crate::sysinfo::SystemInfo,
        }
        
        let system_info = crate::sysinfo::SystemInfo::collect();
        
        let url = format!("{}/api/v1/init", self.gateway_url);
        let response = self.client
            .post(&url)
            .header("Authorization", format!("Bearer {}", self.token))
            .json(&InitRequest {
                device_id: self.device_id.clone(),
                name: device_name,
                platform: platform.to_string(),
                system_info,
            })
            .send()?;
        
        if !response.status().is_success() {
            let error_text = response.text().unwrap_or_else(|_| "Unknown error".to_string());
            return Err(anyhow::anyhow!("Init device error: {}", error_text));
        }
        
        info!("Device initialized successfully");
        Ok(())
    }
    
    fn send_batch(&self, events: Vec<TimelineEvent>) -> Result<()> {
        Self::post_batch(&self.client, &self.gateway_url, &self.token, &self.device_id, events)
    }
    
    fn post_batch(
        client: &reqwest::blocking::Client,
        gateway_url: &str,
        token: &str,
        device_id: &str,
        events: Vec<TimelineEvent>,
    ) -> Result<()> {
        let count = events.len();
        let batch = EventBatch {
            device_id: device_id.to_string(),
            events,
        };
        
        let url = format!("{}/api/v1/events", gateway_url);
        let response = client
            .post(&url)
            .header("Authorization", format!("Bearer {}", token))
            .json(&batch)
            .send()?;
        
        if !response.status().is_success() {
            let error_text = response.text().unwrap_or_else(|_| "Unknown error".to_string());
            return Err(anyhow::anyhow!("Gateway error: {}", error_text));
        }
        
        info!("Sent {} events to gateway", count);
        Ok(())
    }
}

impl StorageBackend for RemoteStorage {
    // Buffering only; the background flusher owns delivery, spooling, and
    // backoff. This keeps the hot path (event capture) lock-light and never
    // blocks watchers on network I/O.
    fn insert_event(&self, event: &TimelineEvent) -> Result<()> {
        if let Ok(mut buffer) = self.buffer.lock() {
            buffer.push_back(event.clone());
        }
        if let Ok(mut last_event) = self.last_event_time.lock() {
            *last_event = Instant::now();
        }
        Ok(())
    }
    
    // Best-effort synchronous drain (used on shutdown): combine the disk spool
    // and the in-memory buffer, try once, and on failure persist everything
    // back to the spool so it is replayed on next launch.
    fn flush(&self) -> Result<()> {
        let mut events: Vec<TimelineEvent> =
            self.spool.lock().map(|s| s.load()).unwrap_or_default();
        if let Ok(mut buffer) = self.buffer.lock() {
            events.extend(buffer.drain(..));
        }
        if events.is_empty() {
            return Ok(());
        }

        match self.send_batch(events.clone()) {
            Ok(_) => {
                if let Ok(s) = self.spool.lock() {
                    s.clear();
                }
                if let Ok(mut last) = self.last_flush.lock() {
                    *last = Instant::now();
                }
                Ok(())
            }
            Err(e) => {
                if let Ok(s) = self.spool.lock() {
                    s.replace(&events);
                }
                Err(e)
            }
        }
    }
}

pub struct Storage {
    backend: Box<dyn StorageBackend>,
}

impl Storage {
    pub fn new(device_id: String, config: &StorageConfig) -> Result<Self> {
        let backend: Box<dyn StorageBackend> = match config {
            StorageConfig::Local { data_dir } => {
                info!("Using local storage: {}", data_dir);
                Box::new(LocalStorage::new(data_dir)?)
            }
            StorageConfig::Remote {
                gateway_url,
                token,
                batch_size,
                flush_interval_secs,
            } => {
                info!("Using remote storage: {}", gateway_url);
                Box::new(RemoteStorage::new(
                    device_id,
                    gateway_url.clone(),
                    token.clone(),
                    *batch_size,
                    *flush_interval_secs,
                )?)
            }
        };
        
        Ok(Self { backend })
    }
    
    pub fn insert_event(&self, event: &TimelineEvent) -> Result<()> {
        self.backend.insert_event(event)
    }
    
    pub fn flush(&self) -> Result<()> {
        self.backend.flush()
    }
}

impl Clone for Storage {
    fn clone(&self) -> Self {
        Self {
            backend: Box::new(ClonableBackend),
        }
    }
}

struct ClonableBackend;

impl StorageBackend for ClonableBackend {
    fn insert_event(&self, _event: &TimelineEvent) -> Result<()> {
        Ok(())
    }
    
    fn flush(&self) -> Result<()> {
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::events::EventType;

    fn sample(n: usize) -> Vec<TimelineEvent> {
        (0..n)
            .map(|_| TimelineEvent::new("dev".into(), EventType::SystemSample))
            .collect()
    }

    #[test]
    fn spool_round_trips_events() {
        let dir = std::env::temp_dir().join(format!("arcnode-spool-test-{}", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        let path = dir.join("spool.jsonl");
        let spool = Spool::new(path.clone());

        assert!(spool.load().is_empty());

        let events = sample(3);
        spool.replace(&events);
        let loaded = spool.load();
        assert_eq!(loaded.len(), 3);
        // event_id must survive the round-trip so the gateway can deduplicate.
        assert_eq!(loaded[0].event_id, events[0].event_id);

        spool.clear();
        assert!(spool.load().is_empty());
        let _ = std::fs::remove_dir_all(&dir);
    }

    #[test]
    fn spool_caps_at_max_events() {
        let dir = std::env::temp_dir().join(format!("arcnode-spool-cap-{}", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        let path = dir.join("spool.jsonl");
        let spool = Spool::new(path.clone());

        let events = sample(SPOOL_MAX_EVENTS + 10);
        spool.replace(&events);
        assert_eq!(spool.load().len(), SPOOL_MAX_EVENTS);
        let _ = std::fs::remove_dir_all(&dir);
    }
}
