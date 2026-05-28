use anyhow::Result;
use log::{error, info};
use serde::{Deserialize, Serialize};
use std::collections::VecDeque;
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
        };
        
        if let Err(e) = storage.init_device() {
            error!("Failed to initialize device: {}", e);
        }
        
        storage.spawn_background_flusher();
        
        Ok(storage)
    }
    
    // Periodically drains and ships the buffer so low-frequency events (e.g.
    // system samples on an otherwise idle machine) are delivered reliably and
    // are never left stranded between insert_event calls.
    fn spawn_background_flusher(&self) {
        let buffer = self.buffer.clone();
        let last_flush = self.last_flush.clone();
        let client = self.client.clone();
        let url = self.gateway_url.clone();
        let token = self.token.clone();
        let device_id = self.device_id.clone();
        let interval = self.max_flush_interval;
        thread::spawn(move || loop {
            thread::sleep(interval);
            let events: Vec<TimelineEvent> = {
                let mut buf = match buffer.lock() {
                    Ok(b) => b,
                    Err(_) => continue,
                };
                if buf.is_empty() {
                    continue;
                }
                buf.drain(..).collect()
            };
            match Self::post_batch(&client, &url, &token, &device_id, events.clone()) {
                Ok(_) => {
                    if let Ok(mut last) = last_flush.lock() {
                        *last = Instant::now();
                    }
                }
                Err(e) => {
                    error!("Background flush failed, re-queueing {} events: {}", events.len(), e);
                    if let Ok(mut buf) = buffer.lock() {
                        for ev in events.into_iter().rev() {
                            buf.push_front(ev);
                        }
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
    
    fn should_flush(&self, buffer_len: usize) -> bool {
        if buffer_len == 0 {
            return false;
        }
        
        if buffer_len >= self.batch_size {
            return true;
        }
        
        let last_flush_elapsed = self.last_flush.lock()
            .map(|last| last.elapsed())
            .unwrap_or(Duration::from_secs(0));
        
        let last_event_elapsed = self.last_event_time.lock()
            .map(|last| last.elapsed())
            .unwrap_or(Duration::from_secs(0));
        
        if buffer_len <= 5 && last_event_elapsed >= Duration::from_millis(500) {
            return true;
        }
        
        if last_flush_elapsed >= self.min_flush_interval && buffer_len > 0 {
            return true;
        }
        
        if last_flush_elapsed >= self.max_flush_interval {
            return true;
        }
        
        false
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
    fn insert_event(&self, event: &TimelineEvent) -> Result<()> {
        let mut buffer = self.buffer.lock().unwrap();
        buffer.push_back(event.clone());
        
        if let Ok(mut last_event) = self.last_event_time.lock() {
            *last_event = Instant::now();
        }
        
        let buffer_len = buffer.len();
        if self.should_flush(buffer_len) {
            let events: Vec<_> = buffer.drain(..).collect();
            drop(buffer);
            
            if let Err(e) = self.send_batch(events) {
                error!("Failed to send batch: {}", e);
                return Err(e);
            }
            
            if let Ok(mut last) = self.last_flush.lock() {
                *last = Instant::now();
            }
        }
        
        Ok(())
    }
    
    fn flush(&self) -> Result<()> {
        let mut buffer = self.buffer.lock().unwrap();
        if buffer.is_empty() {
            return Ok(());
        }
        
        let events: Vec<_> = buffer.drain(..).collect();
        drop(buffer);
        
        self.send_batch(events)?;
        
        if let Ok(mut last) = self.last_flush.lock() {
            *last = Instant::now();
        }
        
        Ok(())
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
