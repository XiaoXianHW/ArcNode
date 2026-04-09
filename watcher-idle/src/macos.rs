use anyhow::Result;
use core_graphics::event_source::{CGEventSource, CGEventSourceStateID};
use log::{error, info};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::thread;
use std::time::Duration;
use core::{EventType, IdleConfig, Storage, TimelineEvent};

pub fn start_monitoring(device_id: String, storage: Arc<Storage>, running: Arc<AtomicBool>, idle_config: IdleConfig) -> Result<()> {
    info!("Starting macOS idle monitor (threshold: {}s)", idle_config.threshold_seconds);
    
    thread::spawn(move || {
        monitor_idle_time(device_id, storage, running, idle_config);
    });
    
    Ok(())
}

fn monitor_idle_time(device_id: String, storage: Arc<Storage>, running: Arc<AtomicBool>, idle_config: IdleConfig) {
    let mut is_idle = false;
    
    while running.load(Ordering::SeqCst) {
        thread::sleep(Duration::from_secs(idle_config.check_interval_seconds));
        
        let idle_seconds = get_idle_time_seconds();
        let now_idle = idle_seconds >= idle_config.threshold_seconds;
        
        if now_idle && !is_idle {
            let event = TimelineEvent::idle(device_id.clone(), EventType::IdleStart);
            if let Err(e) = storage.insert_event(&event) {
                error!("Failed to insert idle start event: {}", e);
            } else {
                info!("User became idle ({}s of inactivity)", idle_seconds);
            }
            is_idle = true;
        } else if !now_idle && is_idle {
            let event = TimelineEvent::idle(device_id.clone(), EventType::IdleEnd);
            if let Err(e) = storage.insert_event(&event) {
                error!("Failed to insert idle end event: {}", e);
            } else {
                info!("User became active again");
            }
            is_idle = false;
        }
    }
    
    info!("Idle monitor stopped");
}

fn get_idle_time_seconds() -> u64 {
    let source = CGEventSource::new(CGEventSourceStateID::CombinedSessionState)
        .expect("Failed to create event source");
    
    let idle_time_nanos = source.seconds_since_last_event_type(
        core_graphics::event::CGEventType::Null
    );
    
    idle_time_nanos as u64
}
