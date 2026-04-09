use anyhow::Result;
use log::{error, info};
use std::ffi::{CStr, CString};
use std::os::raw::{c_char, c_int, c_uchar, c_ulong};
use std::ptr;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{Arc, OnceLock};
use std::thread;
use std::time::Duration;
use core::{Storage, TimelineEvent};
use x11::xlib::*;
use x11::xrecord::*;

static STORAGE: OnceLock<Arc<Storage>> = OnceLock::new();
static DEVICE_ID: OnceLock<String> = OnceLock::new();
static RUNNING: OnceLock<Arc<AtomicBool>> = OnceLock::new();

fn get_key_name(keycode: u32) -> String {
    match keycode {
        9 => "Esc".to_string(),
        10..=19 => format!("{}", keycode - 9),
        24 => "Q".to_string(),
        25 => "W".to_string(),
        26 => "E".to_string(),
        27 => "R".to_string(),
        28 => "T".to_string(),
        29 => "Y".to_string(),
        30 => "U".to_string(),
        31 => "I".to_string(),
        32 => "O".to_string(),
        33 => "P".to_string(),
        38 => "A".to_string(),
        39 => "S".to_string(),
        40 => "D".to_string(),
        41 => "F".to_string(),
        42 => "G".to_string(),
        43 => "H".to_string(),
        44 => "J".to_string(),
        45 => "K".to_string(),
        46 => "L".to_string(),
        52 => "Z".to_string(),
        53 => "X".to_string(),
        54 => "C".to_string(),
        55 => "V".to_string(),
        56 => "B".to_string(),
        57 => "N".to_string(),
        58 => "M".to_string(),
        23 => "Tab".to_string(),
        36 => "Enter".to_string(),
        65 => "Space".to_string(),
        22 => "Backspace".to_string(),
        119 => "Delete".to_string(),
        50 => "Shift".to_string(),
        62 => "RightShift".to_string(),
        37 => "Ctrl".to_string(),
        105 => "RightCtrl".to_string(),
        64 => "Alt".to_string(),
        108 => "RightAlt".to_string(),
        133 => "Super".to_string(),
        134 => "RightSuper".to_string(),
        67..=76 => format!("F{}", keycode - 66),
        95 => "F11".to_string(),
        96 => "F12".to_string(),
        111 => "Up".to_string(),
        116 => "Down".to_string(),
        113 => "Left".to_string(),
        114 => "Right".to_string(),
        110 => "Home".to_string(),
        115 => "End".to_string(),
        112 => "PageUp".to_string(),
        117 => "PageDown".to_string(),
        _ => format!("Key{}", keycode),
    }
}

fn get_active_window_class(display: *mut Display) -> Option<String> {
    unsafe {
        let mut focus_window: Window = 0;
        let mut revert_to: c_int = 0;
        
        XGetInputFocus(display, &mut focus_window, &mut revert_to);
        
        if focus_window == 0 || focus_window == 1 {
            return None;
        }
        
        let mut actual_type: Atom = 0;
        let mut actual_format: c_int = 0;
        let mut nitems: c_ulong = 0;
        let mut bytes_after: c_ulong = 0;
        let mut prop: *mut c_uchar = ptr::null_mut();
        
        let wm_class = XInternAtom(display, CString::new("WM_CLASS").unwrap().as_ptr(), False);
        
        if XGetWindowProperty(
            display,
            focus_window,
            wm_class,
            0,
            1024,
            False,
            AnyPropertyType as c_ulong,
            &mut actual_type,
            &mut actual_format,
            &mut nitems,
            &mut bytes_after,
            &mut prop,
        ) == Success as c_int && !prop.is_null() {
            
            let class_str = CStr::from_ptr(prop as *const c_char);
            let result = class_str.to_string_lossy().into_owned();
            XFree(prop as *mut _);
            
            if !result.is_empty() {
                return Some(result);
            }
        }
        
        let mut window_name: *mut c_char = ptr::null_mut();
        if XFetchName(display, focus_window, &mut window_name) != 0 && !window_name.is_null() {
            let name_str = CStr::from_ptr(window_name);
            let result = name_str.to_string_lossy().into_owned();
            XFree(window_name as *mut _);
            
            if !result.is_empty() {
                return Some(result);
            }
        }
    }
    
    None
}

unsafe extern "C" fn event_callback(
    _closure: XPointer,
    rec_data: *mut XRecordInterceptData,
) {
    if rec_data.is_null() {
        return;
    }
    
    let data = &*rec_data;
    
    if let Some(running) = RUNNING.get() {
        if !running.load(Ordering::SeqCst) {
            return;
        }
    }
    
    if data.category == XRecordFromServer {
        let event_data = data.data as *const c_uchar;
        if !event_data.is_null() {
            let event_type = *event_data;
            
            if event_type == KeyPress as u8 {
                let keycode = *event_data.offset(1) as u32;
                
                let display = XOpenDisplay(ptr::null());
                if display.is_null() {
                    return;
                }
                
                let mut keys: [c_char; 32] = [0; 32];
                XQueryKeymap(display, keys.as_mut_ptr());
                
                let shift_pressed = (keys[6] & 0x01) != 0 || (keys[7] & 0x40) != 0;
                let ctrl_pressed = (keys[4] & 0x20) != 0 || (keys[13] & 0x02) != 0;
                let alt_pressed = (keys[8] & 0x01) != 0 || (keys[13] & 0x10) != 0;
                let super_pressed = (keys[16] & 0x20) != 0 || (keys[16] & 0x40) != 0;
                
                if ctrl_pressed || alt_pressed || super_pressed || 
                   (shift_pressed && ![50, 62].contains(&keycode)) {
                    
                    let mut shortcut_parts = Vec::new();
                    
                    if ctrl_pressed { shortcut_parts.push("Ctrl"); }
                    if alt_pressed { shortcut_parts.push("Alt"); }
                    if shift_pressed { shortcut_parts.push("Shift"); }
                    if super_pressed { shortcut_parts.push("Super"); }
                    
                    let key_name = get_key_name(keycode);
                    shortcut_parts.push(&key_name);
                    
                    let shortcut = shortcut_parts.join("+");
                    let app_name = get_active_window_class(display);
                    
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
                
                XCloseDisplay(display);
            }
        }
    }
    
    XRecordFreeData(rec_data);
}

pub fn start_monitoring(device_id: String, storage: Arc<Storage>, running: Arc<AtomicBool>) -> Result<()> {
    info!("Starting shortcut monitoring on Linux...");
    
    STORAGE.set(storage).map_err(|_| anyhow::anyhow!("Failed to set storage"))?;
    DEVICE_ID.set(device_id).map_err(|_| anyhow::anyhow!("Failed to set device ID"))?;
    RUNNING.set(running.clone()).map_err(|_| anyhow::anyhow!("Failed to set running flag"))?;
    
    unsafe {
        let display = XOpenDisplay(ptr::null());
        if display.is_null() {
            error!("Cannot open X11 display");
            return Err(anyhow::anyhow!("Cannot open X11 display"));
        }
        
        let mut major_version: c_int = 0;
        let mut minor_version: c_int = 0;
        
        if XRecordQueryVersion(display, &mut major_version, &mut minor_version) == 0 {
            XCloseDisplay(display);
            error!("XRecord extension not available");
            return Err(anyhow::anyhow!("XRecord extension not available"));
        }
        
        info!("XRecord version: {}.{}", major_version, minor_version);
        
        let clients = XRecordAllClients;
        let mut ranges = XRecordRange {
            core_requests: XRecordRange8 {
                first: 0,
                last: 0,
            },
            core_replies: XRecordRange8 {
                first: 0,
                last: 0,
            },
            ext_requests: XRecordExtRange {
                ext_major: XRecordRange8 {
                    first: 0,
                    last: 0,
                },
                ext_minor: XRecordRange16 {
                    first: 0,
                    last: 0,
                },
            },
            ext_replies: XRecordExtRange {
                ext_major: XRecordRange8 {
                    first: 0,
                    last: 0,
                },
                ext_minor: XRecordRange16 {
                    first: 0,
                    last: 0,
                },
            },
            delivered_events: XRecordRange8 {
                first: KeyPress as u8,
                last: KeyRelease as u8,
            },
            device_events: XRecordRange8 {
                first: KeyPress as u8,
                last: KeyRelease as u8,
            },
            errors: XRecordRange8 {
                first: 0,
                last: 0,
            },
            client_started: False,
            client_died: False,
        };
        
        let context = XRecordCreateContext(
            display,
            0,
            &clients,
            1,
            &mut &mut ranges as *mut *mut XRecordRange,
            1,
        );
        
        if context == 0 {
            XCloseDisplay(display);
            error!("Failed to create XRecord context");
            return Err(anyhow::anyhow!("Failed to create XRecord context"));
        }
        
        info!("Starting XRecord monitoring...");
        let record_display = XOpenDisplay(ptr::null());
        if record_display.is_null() {
            XRecordFreeContext(display, context);
            XCloseDisplay(display);
            error!("Cannot open second X11 display for recording");
            return Err(anyhow::anyhow!("Cannot open second display"));
        }
        
        let running_clone = running.clone();
        let record_thread = thread::spawn(move || {
            unsafe {
                XRecordEnableContext(record_display, context, Some(event_callback), ptr::null_mut());
                XCloseDisplay(record_display);
            }
        });
        
        while running.load(Ordering::SeqCst) {
            thread::sleep(Duration::from_millis(100));
        }
        
        XRecordDisableContext(display, context);
        XRecordFreeContext(display, context);
        XCloseDisplay(display);
        if let Err(e) = record_thread.join() {
            error!("Record thread join error: {:?}", e);
        }
    }
    
    info!("Shortcut monitoring stopped on Linux");
    Ok(())
}
