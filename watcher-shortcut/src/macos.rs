use anyhow::Result;
use core_foundation::runloop::{kCFRunLoopDefaultMode, CFRunLoop, CFRunLoopRunResult};
use core_graphics::event::{
    CGEvent, CGEventFlags, CGEventTap, CGEventTapLocation, CGEventTapOptions,
    CGEventTapPlacement, CGEventType, EventField,
};
use log::{error, info};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{Arc, OnceLock};
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
    use cocoa::foundation::{NSAutoreleasePool, NSString};
    use objc::{class, msg_send, sel, sel_impl};

    unsafe {
        let pool = NSAutoreleasePool::new(nil);
        let workspace: id = msg_send![class!(NSWorkspace), sharedWorkspace];
        let active_app: id = msg_send![workspace, frontmostApplication];

        if active_app != nil {
            let app_name: id = msg_send![active_app, localizedName];
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

fn handle_key_event(event: &CGEvent) {
    let keycode = event.get_integer_value_field(EventField::KEYBOARD_EVENT_KEYCODE) as u16;
    let flags = event.get_flags();

    let cmd_pressed = flags.contains(CGEventFlags::CGEventFlagCommand);
    let ctrl_pressed = flags.contains(CGEventFlags::CGEventFlagControl);
    let alt_pressed = flags.contains(CGEventFlags::CGEventFlagAlternate);
    let shift_pressed = flags.contains(CGEventFlags::CGEventFlagShift);

    if !(cmd_pressed
        || ctrl_pressed
        || alt_pressed
        || (shift_pressed && ![56, 60].contains(&keycode)))
    {
        return;
    }

    let mut shortcut_parts: Vec<&str> = Vec::new();
    if ctrl_pressed { shortcut_parts.push("Ctrl"); }
    if alt_pressed { shortcut_parts.push("Option"); }
    if shift_pressed { shortcut_parts.push("Shift"); }
    if cmd_pressed { shortcut_parts.push("Cmd"); }

    let key_name = get_key_name(keycode);
    shortcut_parts.push(&key_name);
    let shortcut = shortcut_parts.join("+");
    let app_name = get_active_application_name();

    if let (Some(storage), Some(device_id)) = (STORAGE.get(), DEVICE_ID.get()) {
        let ev = TimelineEvent::keyboard_shortcut(device_id.clone(), shortcut, app_name);
        if let Err(e) = storage.insert_event(&ev) {
            error!("Failed to insert keyboard shortcut event: {}", e);
        }
    }
}

pub fn start_monitoring(
    device_id: String,
    storage: Arc<Storage>,
    running: Arc<AtomicBool>,
) -> Result<()> {
    info!("Starting shortcut monitoring on macOS...");

    STORAGE
        .set(storage)
        .map_err(|_| anyhow::anyhow!("Failed to set storage"))?;
    DEVICE_ID
        .set(device_id)
        .map_err(|_| anyhow::anyhow!("Failed to set device ID"))?;
    RUNNING
        .set(running.clone())
        .map_err(|_| anyhow::anyhow!("Failed to set running flag"))?;

    let event_tap = CGEventTap::new(
        CGEventTapLocation::HID,
        CGEventTapPlacement::HeadInsertEventTap,
        CGEventTapOptions::Default,
        vec![CGEventType::KeyDown],
        |_proxy, _event_type, event| {
            if let Some(r) = RUNNING.get() {
                if r.load(Ordering::SeqCst) {
                    handle_key_event(event);
                }
            }
            None
        },
    )
    .map_err(|_| {
        anyhow::anyhow!("Failed to create event tap (accessibility permissions required?)")
    })?;

    let run_loop_source = event_tap
        .mach_port
        .create_runloop_source(0)
        .map_err(|_| anyhow::anyhow!("Failed to create runloop source"))?;
    let run_loop = CFRunLoop::get_current();

    unsafe {
        run_loop.add_source(&run_loop_source, kCFRunLoopDefaultMode);
    }

    event_tap.enable();
    info!("Shortcut monitoring started on macOS");

    while running.load(Ordering::SeqCst) {
        let result = unsafe {
            CFRunLoop::run_in_mode(kCFRunLoopDefaultMode, Duration::from_millis(100), false)
        };
        if result == CFRunLoopRunResult::Stopped {
            break;
        }
    }

    unsafe {
        run_loop.remove_source(&run_loop_source, kCFRunLoopDefaultMode);
    }

    info!("Shortcut monitoring stopped on macOS");
    Ok(())
}
