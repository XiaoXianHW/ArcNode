use core::{EventType, TimelineEvent, Storage};
use anyhow::Result;
use log::{info, error};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{Arc, Mutex, OnceLock};
use std::collections::HashMap;
use std::thread;
use std::time::Duration;
use sysinfo::{System, ProcessRefreshKind};

static TRACKED_PROCESSES: OnceLock<Mutex<HashMap<u32, String>>> = OnceLock::new();
static DEVICE_ID: OnceLock<String> = OnceLock::new();

pub fn start_monitoring(device_id: String, storage: Arc<Storage>, running: Arc<AtomicBool>) -> Result<()> {
    TRACKED_PROCESSES.set(Mutex::new(HashMap::new())).ok();
    DEVICE_ID.set(device_id).ok();
    
    info!("Starting macOS process monitor");
    
    thread::spawn(move || {
        monitor_processes(storage, running);
    });
    
    Ok(())
}

fn monitor_processes(storage: Arc<Storage>, running: Arc<AtomicBool>) {
    let mut sys = System::new_all();
    sys.refresh_processes_specifics(ProcessRefreshKind::everything());
    
    let initial_processes: HashMap<u32, String> = sys.processes()
        .iter()
        .filter(|(_, proc)| is_gui_app(proc))
        .map(|(pid, proc)| (pid.as_u32(), proc.name().to_string()))
        .collect();
    
    let initial_count = initial_processes.len();
    
    if let Some(tracked) = TRACKED_PROCESSES.get() {
        if let Ok(mut tracked_map) = tracked.lock() {
            *tracked_map = initial_processes;
        }
    }
    
    info!("Initialized with {} existing user processes", initial_count);
    
    while running.load(Ordering::SeqCst) {
        sys.refresh_processes_specifics(ProcessRefreshKind::everything());
        
        let current_processes: HashMap<u32, String> = sys.processes()
            .iter()
            .filter(|(_, proc)| is_gui_app(proc))
            .map(|(pid, proc)| (pid.as_u32(), proc.name().to_string()))
            .collect();
        
        if let Some(tracked) = TRACKED_PROCESSES.get() {
            if let Ok(mut tracked_map) = tracked.lock() {
                for (pid, process_name) in current_processes.iter() {
                    if !tracked_map.contains_key(pid) {
                        let device_id = DEVICE_ID.get().cloned().unwrap_or_default();
                        let event = TimelineEvent::new_legacy(
                            device_id,
                            EventType::ProcessStart,
                            process_name.clone(),
                            None,
                            *pid,
                        );
                        
                        if let Err(e) = storage.insert_event(&event) {
                            error!("Failed to insert process start event: {}", e);
                        } else {
                            info!("Process started: {} (PID: {})", process_name, pid);
                        }
                        
                        tracked_map.insert(*pid, process_name.clone());
                    }
                }
                
                let exited_pids: Vec<(u32, String)> = tracked_map.iter()
                    .filter(|(pid, _)| !current_processes.contains_key(pid))
                    .map(|(pid, name)| (*pid, name.clone()))
                    .collect();
                
                for (pid, process_name) in exited_pids {
                    let device_id = DEVICE_ID.get().cloned().unwrap_or_default();
                    let event = TimelineEvent::new_legacy(
                        device_id,
                        EventType::ProcessExit,
                        process_name.clone(),
                        None,
                        pid,
                    );
                    
                    if let Err(e) = storage.insert_event(&event) {
                        error!("Failed to insert process exit event: {}", e);
                    } else {
                        info!("Process exited: {} (PID: {})", process_name, pid);
                    }
                    
                    tracked_map.remove(&pid);
                }
            }
        }
        
        thread::sleep(Duration::from_millis(1000));
    }
    
    info!("Process monitor stopped");
}

fn is_gui_app(process: &sysinfo::Process) -> bool {
    let exe = process.exe();
    if let Some(exe_path) = exe {
        let path_str = exe_path.to_string_lossy();
        return path_str.contains("/Applications/") || 
               path_str.contains(".app/") ||
               path_str.ends_with(".app");
    }
    false
}
