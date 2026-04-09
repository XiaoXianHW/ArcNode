use anyhow::Result;
use log::{error, info};
use std::ptr;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::thread;
use std::time::Duration;
use core::{EventType, IdleConfig, Storage, TimelineEvent};
use x11::xlib::*;
use x11::xss::*;

pub fn start_monitoring(device_id: String, storage: Arc<Storage>, running: Arc<AtomicBool>, idle_config: IdleConfig) -> Result<()> {
    info!("Starting Linux idle monitor (threshold: {}s)", idle_config.threshold_seconds);
    
    thread::spawn(move || {
        monitor_idle_time(device_id, storage, running, idle_config);
    });
    
    Ok(())
}

fn monitor_idle_time(device_id: String, storage: Arc<Storage>, running: Arc<AtomicBool>, idle_config: IdleConfig) {
    unsafe {
        let display = XOpenDisplay(ptr::null());
        if display.is_null() {
            error!("Failed to open X11 display for idle monitoring");
            return;
        }
        
        let mut is_idle = false;
        
        while running.load(Ordering::SeqCst) {
            thread::sleep(Duration::from_secs(idle_config.check_interval_seconds));
            
            let idle_seconds = get_idle_time_seconds(display);
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
        
        XCloseDisplay(display);
    }
    
    info!("Idle monitor stopped");
}

unsafe fn get_idle_time_seconds(display: *mut Display) -> u64 {
    let mut info = XScreenSaverInfo {
        window: 0,
        state: 0,
        kind: 0,
        til_or_since: 0,
        idle: 0,
        eventMask: 0,
    };
    
    let screen = XDefaultScreen(display);
    let root = XRootWindow(display, screen);
    
    if XScreenSaverQueryInfo(display, root, &mut info) != 0 {
        return (info.idle / 1000) as u64;
    }
    
    0
}
