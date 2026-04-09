use core::{EventType, TimelineEvent, Storage};
use anyhow::Result;
use log::{info, error};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::ffi::CStr;
use std::ptr;
use std::thread;
use std::time::Duration;
use x11::xlib::*;
use std::sync::OnceLock;

static DEVICE_ID: OnceLock<String> = OnceLock::new();

pub fn start_monitoring(device_id: String, storage: Arc<Storage>, running: Arc<AtomicBool>) -> Result<()> {
    DEVICE_ID.set(device_id).ok();
    info!("Starting Linux window monitor");
    
    thread::spawn(move || {
        unsafe {
            let display = XOpenDisplay(ptr::null());
            if display.is_null() {
                error!("Failed to open X11 display");
                return;
            }
            
            let root = XDefaultRootWindow(display);
            let net_active_window = XInternAtom(display, b"_NET_ACTIVE_WINDOW\0".as_ptr() as *const i8, 0);
            let net_wm_name = XInternAtom(display, b"_NET_WM_NAME\0".as_ptr() as *const i8, 0);
            let wm_name = XInternAtom(display, b"WM_NAME\0".as_ptr() as *const i8, 0);
            let utf8_string = XInternAtom(display, b"UTF8_STRING\0".as_ptr() as *const i8, 0);
            let net_wm_pid = XInternAtom(display, b"_NET_WM_PID\0".as_ptr() as *const i8, 0);
            
            XSelectInput(display, root, PropertyChangeMask);
            
            let mut last_window: u64 = 0;
            let mut last_title = String::new();
            let mut last_pid: u32 = 0;
            
            info!("X11 window event hooks installed");
            
            while running.load(Ordering::SeqCst) {
                while XPending(display) > 0 {
                    let mut event: XEvent = std::mem::zeroed();
                    XNextEvent(display, &mut event);
                    
                    if event.get_type() == PropertyNotify {
                        let prop_event = event.property;
                        if prop_event.atom == net_active_window {
                            if let Some((window, title, pid)) = get_active_window_info(
                                display, root, net_active_window, net_wm_name, utf8_string, net_wm_pid
                            ) {
                                if window != last_window || title != last_title {
                                    handle_window_change(&storage, pid, &title, "Window changed");
                                    
                                    if last_window != 0 && last_window != window {
                                        XSelectInput(display, last_window, NoEventMask);
                                    }
                                    
                                    XSelectInput(display, window, PropertyChangeMask);
                                    
                                    last_window = window;
                                    last_title = title;
                                    last_pid = pid;
                                }
                            }
                        } else if last_window != 0 && prop_event.window == last_window 
                                  && (prop_event.atom == net_wm_name || prop_event.atom == wm_name) {
                            if let Some((title, pid)) = get_window_info(
                                display, last_window, net_wm_name, utf8_string, net_wm_pid
                            ) {
                                if title != last_title && pid == last_pid {
                                    handle_window_change(&storage, pid, &title, "Title changed");
                                    last_title = title;
                                }
                            }
                        }
                    }
                }
                
                thread::sleep(Duration::from_millis(10));
            }
            
            XCloseDisplay(display);
        }
    });
    
    Ok(())
}

unsafe fn get_active_window_info(
    display: *mut Display,
    root: u64,
    net_active_window: u64,
    net_wm_name: u64,
    utf8_string: u64,
    net_wm_pid: u64,
) -> Option<(u64, String, u32)> {
    let mut actual_type: u64 = 0;
    let mut actual_format: i32 = 0;
    let mut nitems: u64 = 0;
    let mut bytes_after: u64 = 0;
    let mut prop: *mut u8 = ptr::null_mut();
    
    if XGetWindowProperty(
        display,
        root,
        net_active_window,
        0,
        1,
        0,
        33,
        &mut actual_type,
        &mut actual_format,
        &mut nitems,
        &mut bytes_after,
        &mut prop,
    ) == 0 && !prop.is_null() {
        let window = *(prop as *const u64);
        XFree(prop as *mut _);
        
        if window != 0 {
            if let Some((title, pid)) = get_window_info(display, window, net_wm_name, utf8_string, net_wm_pid) {
                return Some((window, title, pid));
            }
        }
    }
    
    None
}

unsafe fn get_window_info(
    display: *mut Display,
    window: u64,
    net_wm_name: u64,
    utf8_string: u64,
    net_wm_pid: u64,
) -> Option<(String, u32)> {
    let mut actual_type: u64 = 0;
    let mut actual_format: i32 = 0;
    let mut nitems: u64 = 0;
    let mut bytes_after: u64 = 0;
    let mut prop: *mut u8 = ptr::null_mut();
    
    let mut title = String::new();
    
    if XGetWindowProperty(
        display,
        window,
        net_wm_name,
        0,
        1024,
        0,
        utf8_string,
        &mut actual_type,
        &mut actual_format,
        &mut nitems,
        &mut bytes_after,
        &mut prop,
    ) == 0 && !prop.is_null() {
        let c_str = CStr::from_ptr(prop as *const i8);
        title = c_str.to_string_lossy().to_string();
        XFree(prop as *mut _);
    }
    
    let mut pid: u32 = 0;
    prop = ptr::null_mut();
    
    if XGetWindowProperty(
        display,
        window,
        net_wm_pid,
        0,
        1,
        0,
        6,
        &mut actual_type,
        &mut actual_format,
        &mut nitems,
        &mut bytes_after,
        &mut prop,
    ) == 0 && !prop.is_null() {
        pid = *(prop as *const u32);
        XFree(prop as *mut _);
    }
    
    if !title.is_empty() && pid != 0 {
        Some((title, pid))
    } else {
        None
    }
}

fn handle_window_change(storage: &Arc<Storage>, pid: u32, title: &str, log_prefix: &str) {
    let process_name = get_process_name(pid);
    let device_id = DEVICE_ID.get().cloned().unwrap_or_default();
    
    let event = TimelineEvent::new_legacy(
        device_id,
        EventType::ForegroundChange,
        process_name.clone(),
        Some(title.to_string()),
        pid,
    );
    
    if let Err(e) = storage.insert_event(&event) {
        error!("Failed to insert event: {}", e);
    } else {
        info!("{}: {} - {}", log_prefix, process_name, title);
    }
}

fn get_process_name(pid: u32) -> String {
    std::fs::read_to_string(format!("/proc/{}/comm", pid))
        .unwrap_or_else(|_| String::from("Unknown"))
        .trim()
        .to_string()
}
