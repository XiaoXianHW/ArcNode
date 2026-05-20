use anyhow::Result;
use core::{EventType, Storage, TimelineEvent};
use log::info;
use serde_json::{Map, Number, Value};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::thread;
use std::time::Duration;
use sysinfo::System;

pub fn start_monitoring(
    device_id: String,
    storage: Arc<Storage>,
    running: Arc<AtomicBool>,
    interval_secs: u64,
) -> Result<()> {
    let interval = if interval_secs == 0 { 60 } else { interval_secs };
    info!("Starting system sampler (interval={}s)", interval);
    thread::spawn(move || {
        let mut sys = System::new_all();
        sys.refresh_cpu();
        thread::sleep(Duration::from_millis(500));
        while running.load(Ordering::SeqCst) {
            sys.refresh_cpu();
            sys.refresh_memory();
            let cpu = sys.global_cpu_info().cpu_usage();
            let mem_total = sys.total_memory().max(1) as f64;
            let mem_used = sys.used_memory() as f64;
            let mem_pct = (mem_used / mem_total) * 100.0;

            let mut meta = Map::new();
            meta.insert(
                "cpu".to_string(),
                Value::Number(Number::from_f64(cpu as f64).unwrap_or_else(|| Number::from(0))),
            );
            meta.insert(
                "memory".to_string(),
                Value::Number(Number::from_f64(mem_pct).unwrap_or_else(|| Number::from(0))),
            );
            meta.insert(
                "memory_bytes".to_string(),
                Value::Number(Number::from(sys.used_memory())),
            );
            meta.insert(
                "memory_total_bytes".to_string(),
                Value::Number(Number::from(sys.total_memory())),
            );

            let ev = TimelineEvent::new(device_id.clone(), EventType::SystemSample)
                .with_metadata(meta);
            match storage.insert_event(&ev) {
                Ok(_) => info!(
                    "system sample cpu={:.1}% mem={:.1}%",
                    cpu, mem_pct
                ),
                Err(e) => log::warn!("system sample push failed: {}", e),
            }
            thread::sleep(Duration::from_secs(interval));
        }
    });
    Ok(())
}
