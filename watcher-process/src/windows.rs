use anyhow::Result;
use log::{error, info};
use std::collections::HashMap;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{Arc, Mutex, OnceLock};
use std::thread;
use std::time::Duration;
use sysinfo::{ProcessRefreshKind, System};
use core::{EventType, Storage, TimelineEvent};
use windows::Win32::Foundation::{BOOL, HWND, LPARAM};
use windows::Win32::UI::WindowsAndMessaging::{
    EnumWindows, GetWindow, GetWindowTextW, GetWindowThreadProcessId, IsWindowVisible, GW_OWNER,
};

static TRACKED_PROCESSES: OnceLock<Mutex<HashMap<u32, String>>> = OnceLock::new();
static DEVICE_ID: OnceLock<String> = OnceLock::new();

pub fn start_monitoring(device_id: String, storage: Arc<Storage>, running: Arc<AtomicBool>) -> Result<()> {
    TRACKED_PROCESSES.set(Mutex::new(HashMap::new())).ok();
    DEVICE_ID.set(device_id).ok();
    
    info!("Starting Windows process monitor");
    
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
        .filter(|(_, proc)| has_user_visible_window(proc.pid().as_u32()))
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
            .filter(|(_, proc)| has_user_visible_window(proc.pid().as_u32()))
            .map(|(pid, proc)| (pid.as_u32(), proc.name().to_string()))
            .collect();
        
        if let Some(tracked) = TRACKED_PROCESSES.get() {
            if let Ok(mut tracked_map) = tracked.lock() {
                for (pid, process_name) in current_processes.iter() {
                    if !tracked_map.contains_key(pid) {
                        let event = if let Some(device_id) = DEVICE_ID.get() {
                            TimelineEvent::new_legacy(
                                device_id.clone(),
                                EventType::ProcessStart,
                                process_name.clone(),
                                None,
                                *pid,
                            )
                        } else {
                            continue;
                        };
                        
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
                    let event = if let Some(device_id) = DEVICE_ID.get() {
                        TimelineEvent::new_legacy(
                            device_id.clone(),
                            EventType::ProcessExit,
                            process_name.clone(),
                            None,
                            pid,
                        )
                    } else {
                        continue;
                    };
                    
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

struct ProcessWindowInfo {
    target_pid: u32,
    has_main_window: bool,
}

fn has_user_visible_window(pid: u32) -> bool {
    unsafe {
        let mut info = ProcessWindowInfo {
            target_pid: pid,
            has_main_window: false,
        };
        
        EnumWindows(
            Some(check_process_window),
            LPARAM(&mut info as *mut ProcessWindowInfo as isize)
        ).ok();
        
        info.has_main_window
    }
}

unsafe extern "system" fn check_process_window(hwnd: HWND, lparam: LPARAM) -> BOOL {
    let info = &mut *(lparam.0 as *mut ProcessWindowInfo);
    
    if !IsWindowVisible(hwnd).as_bool() {
        return BOOL(1);
    }
    
    if GetWindow(hwnd, GW_OWNER).0 != 0 {
        return BOOL(1);
    }
    
    let mut window_pid: u32 = 0;
    GetWindowThreadProcessId(hwnd, Some(&mut window_pid));
    
    if window_pid == info.target_pid {
        let mut title = [0u16; 512];
        let len = GetWindowTextW(hwnd, &mut title);
        
        if len > 0 {
            info.has_main_window = true;
            return BOOL(0);
        }
    }
    
    BOOL(1)
}
