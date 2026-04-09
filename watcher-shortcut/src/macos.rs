use anyhow::Result;
use core_foundation::machport::CFMachPort;
use core_foundation::runloop::{CFRunLoop, CFRunLoopMode, CFRunLoopRef};
use core_graphics::event::{
    CGEvent, CGEventRef, CGEventTap, CGEventTapLocation, CGEventTapOptions, 
    CGEventTapPlacement, CGEventType,
};
use core_graphics::event_source::CGEventSource;
use log::{error, info};
use std::ffi::c_void;
use std::ptr;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{Arc, OnceLock};
use std::thread;
use std::time::Duration;
use core::{Storage, TimelineEvent};

static STORAGE: OnceLock<Arc<Storage>> = OnceLock::new();
static DEVICE_ID: OnceLock<String> = OnceLock::new();
static RUNNING: OnceLock<Arc<AtomicBool>> = OnceLock::new();

fn get_key_name(keycode: u16) -> String {
    match keycode {
        0 => "A".to_string(),
        1 => "S".to_string(),
        2 => "D".to_string(),
        3 => "F".to_string(),
        4 => "H".to_string(),
        5 => "G".to_string(),
        6 => "Z".to_string(),
        7 => "X".to_string(),
        8 => "C".to_string(),
        9 => "V".to_string(),
        11 => "B".to_string(),
        12 => "Q".to_string(),
        13 => "W".to_string(),
        14 => "E".to_string(),
        15 => "R".to_string(),
        16 => "Y".to_string(),
        17 => "T".to_string(),
        18 => "1".to_string(),
        19 => "2".to_string(),
        20 => "3".to_string(),
        21 => "4".to_string(),
        22 => "6".to_string(),
        23 => "5".to_string(),
        24 => "=".to_string(),
        25 => "9".to_string(),
        26 => "7".to_string(),
        27 => "-".to_string(),
        28 => "8".to_string(),
        29 => "0".to_string(),
        30 => "]".to_string(),
        31 => "O".to_string(),
        32 => "U".to_string(),
        33 => "[".to_string(),
        34 => "I".to_string(),
        35 => "P".to_string(),
        36 => "Enter".to_string(),
        37 => "L".to_string(),
        38 => "J".to_string(),
        39 => "'".to_string(),
        40 => "K".to_string(),
        41 => ";".to_string(),
        42 => "\\".to_string(),
        43 => ",".to_string(),
        44 => "/".to_string(),
        45 => "N".to_string(),
        46 => "M".to_string(),
        47 => ".".to_string(),
        48 => "Tab".to_string(),
        49 => "Space".to_string(),
        50 => "`".to_string(),
        51 => "Delete".to_string(),
        53 => "Esc".to_string(),
        55 => "Cmd".to_string(),
        56 => "Shift".to_string(),
        57 => "CapsLock".to_string(),
        58 => "Option".to_string(),
        59 => "Ctrl".to_string(),
        60 => "RightShift".to_string(),
        61 => "RightOption".to_string(),
        62 => "RightCtrl".to_string(),
        63 => "Fn".to_string(),
        96 => "F5".to_string(),
        97 => "F6".to_string(),
        98 => "F7".to_string(),
        99 => "F3".to_string(),
        100 => "F8".to_string(),
        101 => "F9".to_string(),
        103 => "F11".to_string(),
        105 => "F13".to_string(),
        107 => "F14".to_string(),
        109 => "F10".to_string(),
        111 => "F12".to_string(),
        113 => "F15".to_string(),
        114 => "Help".to_string(),
        115 => "Home".to_string(),
        116 => "PageUp".to_string(),
        117 => "ForwardDelete".to_string(),
        118 => "F4".to_string(),
        119 => "End".to_string(),
        120 => "F2".to_string(),
        121 => "PageDown".to_string(),
        122 => "F1".to_string(),
        123 => "Left".to_string(),
        124 => "Right".to_string(),
        125 => "Down".to_string(),
        126 => "Up".to_string(),
        _ => format!("Key{}", keycode),
    }
}

fn get_active_application_name() -> Option<String> {
    use cocoa::base::{id, nil};
    use cocoa::foundation::{NSString, NSAutoreleasePool};
    use cocoa::appkit::NSWorkspace;
    
    unsafe {
        let pool = NSAutoreleasePool::new(nil);
        let workspace = NSWorkspace::sharedWorkspace(nil);
        let active_app = NSWorkspace::frontmostApplication(workspace);
        
        if active_app != nil {
            let app_name: id = unsafe { objc::msg_send![active_app, localizedName] };
            if app_name != nil {
                let name = NSString::UTF8String(app_name);
                if !name.is_null() {
                    let result = std::ffi::CStr::from_ptr(name).to_string_lossy().into_owned();
                    pool.drain();
                    return Some(result);
                }
            }
        }
        
        pool.drain();
    }
    
    None
}

extern "C" fn event_tap_callback(
    _proxy: core_graphics::event::CGEventTapProxy,
    event_type: CGEventType,
    event: CGEventRef,
    _user_info: *mut c_void,
) -> CGEventRef {
    if let Some(running) = RUNNING.get() {
        if !running.load(Ordering::SeqCst) {
            return event;
        }
    }
    
    if event_type == CGEventType::KeyDown {
        unsafe {
            let cg_event = CGEvent::from_ptr(event);
            let keycode = cg_event.get_integer_value_field(core_graphics::event::CGEventField::KeyboardEventKeycode) as u16;
            let flags = cg_event.get_flags();
            
            let cmd_pressed = flags.contains(core_graphics::event::CGEventFlags::CGEventFlagCommand);
            let ctrl_pressed = flags.contains(core_graphics::event::CGEventFlags::CGEventFlagControl);
            let alt_pressed = flags.contains(core_graphics::event::CGEventFlags::CGEventFlagAlternate);
            let shift_pressed = flags.contains(core_graphics::event::CGEventFlags::CGEventFlagShift);
            
            if cmd_pressed || ctrl_pressed || alt_pressed || 
               (shift_pressed && ![56, 60].contains(&keycode)) {
                
                let mut shortcut_parts = Vec::new();
                
                if ctrl_pressed { shortcut_parts.push("Ctrl"); }
                if alt_pressed { shortcut_parts.push("Option"); }
                if shift_pressed { shortcut_parts.push("Shift"); }
                if cmd_pressed { shortcut_parts.push("Cmd"); }
                
                let key_name = get_key_name(keycode);
                shortcut_parts.push(&key_name);
                
                let shortcut = shortcut_parts.join("+");
                let app_name = get_active_application_name();
                
                if let (Some(storage), Some(device_id)) = (STORAGE.get(), DEVICE_ID.get()) {
                    let event = TimelineEvent::keyboard_shortcut(
                        device_id.clone(),
                        shortcut,
                        app_name,
                    );
                    
                    if let Err(e) = storage.insert_event(&event) {
                        error!("Failed to insert keyboard shortcut event: {}", e);
                    }
                }
            }
        }
    }
    
    event
}

pub fn start_monitoring(device_id: String, storage: Arc<Storage>, running: Arc<AtomicBool>) -> Result<()> {
    info!("Starting shortcut monitoring on macOS...");
    
    STORAGE.set(storage).map_err(|_| anyhow::anyhow!("Failed to set storage"))?;
    DEVICE_ID.set(device_id).map_err(|_| anyhow::anyhow!("Failed to set device ID"))?;
    RUNNING.set(running.clone()).map_err(|_| anyhow::anyhow!("Failed to set running flag"))?;
    
    let trusted = unsafe {
        core_graphics::access::ax_is_process_trusted()
    };
    
    if !trusted {
        error!("Application needs accessibility permissions to monitor keyboard events");
        return Err(anyhow::anyhow!("Accessibility permissions required"));
    }
    
    let event_mask = (1 << CGEventType::KeyDown as u64);
    let event_tap = unsafe {
        CGEventTap::new(
            CGEventTapLocation::HID,
            CGEventTapPlacement::HeadInsertEventTap,
            CGEventTapOptions::Default,
            event_mask,
            event_tap_callback,
            ptr::null_mut(),
        )
    };
    
    match event_tap {
        Ok(tap) => {
            let run_loop_source = tap.mach_port.create_runloop_source(0)?;
            let run_loop = unsafe { CFRunLoop::get_current() };
            
            let default_mode = unsafe { 
                core_foundation::string::CFString::wrap_under_get_rule(kCFRunLoopDefaultMode) 
            };
            
            unsafe {
                run_loop.add_source(&run_loop_source, default_mode);
            }
            
            tap.enable();
            info!("Shortcut monitoring started on macOS");
            while running.load(Ordering::SeqCst) {
                unsafe {
                    let result = CFRunLoop::run_in_mode(
                        default_mode,
                        0.1,
                        false,
                    );
                    
                    if result == core_foundation::runloop::kCFRunLoopRunStopped {
                        break;
                    }
                }
            }
            
            tap.disable();
            unsafe {
                run_loop.remove_source(&run_loop_source, default_mode);
            }
        }
        Err(e) => {
            error!("Failed to create event tap: {:?}", e);
            return Err(anyhow::anyhow!("Failed to create event tap"));
        }
    }
    
    info!("Shortcut monitoring stopped on macOS");
    Ok(())
}

extern "C" {
    static kCFRunLoopDefaultMode: core_foundation::string::CFStringRef;
}
