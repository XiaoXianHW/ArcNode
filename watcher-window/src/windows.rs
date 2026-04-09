use anyhow::Result;
use log::{error, info};
use std::collections::HashMap;
use std::ffi::OsString;
use std::os::windows::ffi::OsStringExt;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{Arc, Mutex, OnceLock};
use std::thread;
use std::time::Duration;
use core::{EventType, Storage, TimelineEvent};
use windows::Win32::Foundation::HWND;
use windows::Win32::System::Diagnostics::ToolHelp::{
    CreateToolhelp32Snapshot, Process32FirstW, Process32NextW, PROCESSENTRY32W, TH32CS_SNAPPROCESS,
};
use windows::Win32::UI::Accessibility::{SetWinEventHook, UnhookWinEvent, HWINEVENTHOOK};
use windows::Win32::UI::WindowsAndMessaging::{
    DispatchMessageW, GetForegroundWindow, GetWindowTextW, GetWindowThreadProcessId, PeekMessageW, 
    TranslateMessage, EVENT_OBJECT_NAMECHANGE, EVENT_SYSTEM_FOREGROUND, MSG, PM_REMOVE, 
    WINEVENT_OUTOFCONTEXT, WINEVENT_SKIPOWNPROCESS,
};

static STORAGE: OnceLock<Arc<Storage>> = OnceLock::new();
static DEVICE_ID: OnceLock<String> = OnceLock::new();
static LAST_STATE: OnceLock<Mutex<(String, String, u32)>> = OnceLock::new();
static PROCESS_NAME_CACHE: OnceLock<Mutex<HashMap<u32, String>>> = OnceLock::new();

pub fn start_monitoring(device_id: String, storage: Arc<Storage>, running: Arc<AtomicBool>) -> Result<()> {
    STORAGE.set(storage).ok();
    DEVICE_ID.set(device_id).ok();
    LAST_STATE.set(Mutex::new((String::new(), String::new(), 0))).ok();
    PROCESS_NAME_CACHE.set(Mutex::new(HashMap::new())).ok();
    
    info!("Starting Windows window monitor");
    
    thread::spawn(move || {
        unsafe {
            let hook_foreground = SetWinEventHook(
                EVENT_SYSTEM_FOREGROUND,
                EVENT_SYSTEM_FOREGROUND,
                None,
                Some(foreground_change_callback),
                0,
                0,
                WINEVENT_OUTOFCONTEXT | WINEVENT_SKIPOWNPROCESS,
            );
            
            if hook_foreground.is_invalid() {
                error!("Failed to set foreground event hook");
                return;
            }
            
            let hook_namechange = SetWinEventHook(
                EVENT_OBJECT_NAMECHANGE,
                EVENT_OBJECT_NAMECHANGE,
                None,
                Some(name_change_callback),
                0,
                0,
                WINEVENT_OUTOFCONTEXT | WINEVENT_SKIPOWNPROCESS,
            );
            
            if hook_namechange.is_invalid() {
                error!("Failed to set name change event hook");
                UnhookWinEvent(hook_foreground);
                return;
            }
            
            info!("Window event hooks installed (foreground + title change)");
            
            let mut msg = MSG::default();
            while running.load(Ordering::SeqCst) {
                while PeekMessageW(&mut msg, None, 0, 0, PM_REMOVE).as_bool() {
                    TranslateMessage(&msg);
                    DispatchMessageW(&msg);
                }
                thread::sleep(Duration::from_millis(10));
            }
            
            UnhookWinEvent(hook_foreground);
            UnhookWinEvent(hook_namechange);
            info!("Window event hooks uninstalled");
        }
    });
    
    Ok(())
}

unsafe extern "system" fn foreground_change_callback(
    _h_win_event_hook: HWINEVENTHOOK,
    _event: u32,
    hwnd: HWND,
    _id_object: i32,
    _id_child: i32,
    _id_event_thread: u32,
    _dwms_event_time: u32,
) {
    handle_window_event(hwnd, "Window changed");
}

unsafe extern "system" fn name_change_callback(
    _h_win_event_hook: HWINEVENTHOOK,
    _event: u32,
    hwnd: HWND,
    id_object: i32,
    _id_child: i32,
    _id_event_thread: u32,
    _dwms_event_time: u32,
) {
    if id_object != 0 || hwnd.0 == 0 {
        return;
    }
    
    let foreground = GetForegroundWindow();
    if foreground.0 != hwnd.0 {
        return;
    }
    
    handle_window_event(hwnd, "Title changed");
}

fn handle_window_event(hwnd: HWND, log_prefix: &str) {
    unsafe {
        if hwnd.0 == 0 {
            return;
        }
        
        let mut window_title = [0u16; 512];
        let len = GetWindowTextW(hwnd, &mut window_title);
        if len == 0 {
            return;
        }
        
        let title = OsString::from_wide(&window_title[..len as usize])
            .to_string_lossy()
            .to_string();
        
        let mut pid: u32 = 0;
        GetWindowThreadProcessId(hwnd, Some(&mut pid));
        if pid == 0 {
            return;
        }
        
        let process_name = get_process_name_cached(pid);
        
        if let Some(last_state) = LAST_STATE.get() {
            if let Ok(mut state) = last_state.lock() {
                if state.0 == title && state.1 == process_name && state.2 == pid {
                    return;
                }
                *state = (title.clone(), process_name.clone(), pid);
            }
        }
        
        let event = if let Some(device_id) = DEVICE_ID.get() {
            TimelineEvent::new_legacy(
                device_id.clone(),
                EventType::ForegroundChange,
                process_name.clone(),
                Some(title.clone()),
                pid,
            )
        } else {
            return;
        };
        
        if let Some(storage) = STORAGE.get() {
            if let Err(e) = storage.insert_event(&event) {
                error!("Failed to insert event: {}", e);
            } else {
                info!("{}: {} - {}", log_prefix, process_name, title);
            }
        }
    }
}

fn get_process_name_cached(pid: u32) -> String {
    if let Some(cache) = PROCESS_NAME_CACHE.get() {
        if let Ok(cache_map) = cache.lock() {
            if let Some(name) = cache_map.get(&pid) {
                return name.clone();
            }
        }
    }
    
    let name = get_process_name(pid).unwrap_or_else(|| String::from("Unknown"));
    
    if let Some(cache) = PROCESS_NAME_CACHE.get() {
        if let Ok(mut cache_map) = cache.lock() {
            cache_map.insert(pid, name.clone());
            if cache_map.len() > 100 {
                cache_map.clear();
            }
        }
    }
    
    name
}

fn get_process_name(pid: u32) -> Option<String> {
    unsafe {
        let snapshot = CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0).ok()?;
        let mut entry = PROCESSENTRY32W {
            dwSize: std::mem::size_of::<PROCESSENTRY32W>() as u32,
            ..Default::default()
        };
        
        Process32FirstW(snapshot, &mut entry).ok()?;
        
        loop {
            if entry.th32ProcessID == pid {
                let name_len = entry.szExeFile.iter().position(|&c| c == 0)?;
                return Some(
                    OsString::from_wide(&entry.szExeFile[..name_len])
                        .to_string_lossy()
                        .to_string()
                );
            }
            
            if Process32NextW(snapshot, &mut entry).is_err() {
                break;
            }
        }
        None
    }
}
