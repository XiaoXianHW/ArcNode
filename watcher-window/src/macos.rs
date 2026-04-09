use core::{EventType, TimelineEvent, Storage};
use anyhow::Result;
use log::{info, error};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::thread;
use std::time::Duration;
use cocoa::base::{id, nil};
use cocoa::foundation::NSAutoreleasePool;
use objc::{msg_send, sel, sel_impl};
use std::sync::OnceLock;

static DEVICE_ID: OnceLock<String> = OnceLock::new();

pub fn start_monitoring(device_id: String, storage: Arc<Storage>, running: Arc<AtomicBool>) -> Result<()> {
    DEVICE_ID.set(device_id).ok();
    info!("Starting macOS window monitor");
    
    thread::spawn(move || {
        unsafe {
            let mut last_app_name = String::new();
            let mut last_window_title = String::new();
            let mut last_pid: u32 = 0;
            
            info!("macOS window monitoring started");
            
            while running.load(Ordering::SeqCst) {
                let pool = NSAutoreleasePool::new(nil);
                
                let workspace: id = msg_send![class!(NSWorkspace), sharedWorkspace];
                let frontmost_app: id = msg_send![workspace, frontMostApplication];
                
                if frontmost_app != nil {
                    let app_name: id = msg_send![frontmost_app, localizedName];
                    let pid: i32 = msg_send![frontmost_app, processIdentifier];
                    
                    if app_name != nil {
                        let app_name_str = nsstring_to_string(app_name);
                        let pid_u32 = pid as u32;
                        
                        let window_title = get_frontmost_window_title();
                        let title_for_event = if !window_title.is_empty() {
                            window_title.clone()
                        } else {
                            app_name_str.clone()
                        };
                        
                        if app_name_str != last_app_name {
                            handle_window_change(&storage, pid_u32, &app_name_str, &title_for_event, "Application changed");
                            
                            last_app_name = app_name_str;
                            last_window_title = window_title;
                            last_pid = pid_u32;
                        } else if window_title != last_window_title && pid_u32 == last_pid {
                            handle_window_change(&storage, pid_u32, &app_name_str, &title_for_event, "Title changed");
                            last_window_title = window_title;
                        }
                    }
                }
                
                let _: () = msg_send![pool, drain];
                thread::sleep(Duration::from_millis(200));
            }
        }
    });
    
    Ok(())
}

unsafe fn nsstring_to_string(ns_string: id) -> String {
    use std::ffi::CStr;
    use std::os::raw::c_char;
    
    let utf8: *const c_char = msg_send![ns_string, UTF8String];
    if utf8.is_null() {
        return String::new();
    }
    
    CStr::from_ptr(utf8)
        .to_string_lossy()
        .to_string()
}

unsafe fn get_frontmost_window_title() -> String {
    let workspace: id = msg_send![class!(NSWorkspace), sharedWorkspace];
    let frontmost_app: id = msg_send![workspace, frontMostApplication];
    
    if frontmost_app == nil {
        return String::new();
    }
    
    let bundle_id: id = msg_send![frontmost_app, bundleIdentifier];
    if bundle_id != nil {
        let bundle_str = nsstring_to_string(bundle_id);
        
        if bundle_str.contains("Safari") || bundle_str.contains("Chrome") || 
           bundle_str.contains("Firefox") || bundle_str.contains("Edge") {
            if let Some(title) = get_browser_window_title() {
                return title;
            }
        }
    }
    
    String::new()
}

unsafe fn get_browser_window_title() -> Option<String> {
    use core_foundation::base::TCFType;
    use core_foundation::string::CFString;
    use core_graphics::window::{kCGWindowListOptionOnScreenOnly, kCGNullWindowID};
    
    let window_list = core_graphics::window::CGWindowListCopyWindowInfo(
        kCGWindowListOptionOnScreenOnly,
        kCGNullWindowID,
    );
    
    if window_list.is_null() {
        return None;
    }
    
    let cf_array = core_foundation::array::CFArray::<_>::wrap_under_create_rule(window_list as _);
    
    for i in 0..cf_array.len() {
        if let Some(window_info) = cf_array.get(i) {
            let dict = window_info as core_foundation::dictionary::CFDictionaryRef;
            
            let layer_key = CFString::new("kCGWindowLayer");
            let layer_value = core_foundation::dictionary::CFDictionaryGetValue(
                dict, 
                layer_key.as_concrete_TypeRef() as *const _
            );
            
            if !layer_value.is_null() {
                let name_key = CFString::new("kCGWindowName");
                let name_value = core_foundation::dictionary::CFDictionaryGetValue(
                    dict,
                    name_key.as_concrete_TypeRef() as *const _
                );
                
                if !name_value.is_null() {
                    let cf_string = CFString::wrap_under_get_rule(name_value as _);
                    let title = cf_string.to_string();
                    if !title.is_empty() {
                        return Some(title);
                    }
                }
            }
        }
    }
    
    None
}

fn handle_window_change(storage: &Arc<Storage>, pid: u32, app_name: &str, title: &str, log_prefix: &str) {
    let device_id = DEVICE_ID.get().cloned().unwrap_or_default();
    let event = TimelineEvent::new_legacy(
        device_id,
        EventType::ForegroundChange,
        app_name.to_string(),
        Some(title.to_string()),
        pid,
    );
    
    if let Err(e) = storage.insert_event(&event) {
        error!("Failed to insert event: {}", e);
    } else {
        info!("{}: {} - {}", log_prefix, app_name, title);
    }
}
