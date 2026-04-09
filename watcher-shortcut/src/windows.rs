use anyhow::Result;
use log::{error, info};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{Arc, OnceLock};
use core::{Storage, TimelineEvent};
use windows::core::PCWSTR;
use windows::Win32::Foundation::*;
use windows::Win32::System::LibraryLoader::GetModuleHandleW;
use windows::Win32::System::ProcessStatus::*;
use windows::Win32::System::Threading::{OpenProcess, PROCESS_QUERY_INFORMATION};
use windows::Win32::UI::Input::KeyboardAndMouse::*;
use windows::Win32::UI::WindowsAndMessaging::*;

static STORAGE: OnceLock<Arc<Storage>> = OnceLock::new();
static DEVICE_ID: OnceLock<String> = OnceLock::new();
static LAST_SHORTCUT: OnceLock<std::sync::Mutex<String>> = OnceLock::new();

fn get_key_name(vk_code: u32) -> String {
    match vk_code {
        0x08 => "Backspace".to_string(),
        0x09 => "Tab".to_string(),
        0x0D => "Enter".to_string(),
        0x10 => "Shift".to_string(),
        0x11 => "Ctrl".to_string(),
        0x12 => "Alt".to_string(),
        0x1B => "Esc".to_string(),
        0x20 => "Space".to_string(),
        0x21 => "PageUp".to_string(),
        0x22 => "PageDown".to_string(),
        0x23 => "End".to_string(),
        0x24 => "Home".to_string(),
        0x25 => "Left".to_string(),
        0x26 => "Up".to_string(),
        0x27 => "Right".to_string(),
        0x28 => "Down".to_string(),
        0x2E => "Delete".to_string(),
        0x30..=0x39 => (vk_code as u8 as char).to_string(),
        0x41..=0x5A => (((vk_code - 0x41) as u8 + b'A') as char).to_string(),
        0x70..=0x87 => format!("F{}", vk_code - 0x6F),
        0x5B => "Win".to_string(),
        0x5C => "Win".to_string(),
        _ => format!("Key{:02X}", vk_code),
    }
}

fn get_foreground_process_name() -> Option<String> {
    unsafe {
        let hwnd = GetForegroundWindow();
        if hwnd.0 == 0 {
            return None;
        }
        
        let mut process_id = 0u32;
        GetWindowThreadProcessId(hwnd, Some(&mut process_id));
        
        if process_id == 0 {
            return None;
        }
        
        let process_handle = OpenProcess(
            PROCESS_QUERY_INFORMATION,
            BOOL(0),
            process_id,
        );
        
        if let Ok(handle) = process_handle {
            let mut buffer = [0u16; 1024];
            
            if GetModuleBaseNameW(
                handle,
                HMODULE(0),
                &mut buffer,
            ) != 0 {
                let end = buffer.iter().position(|&c| c == 0).unwrap_or(buffer.len());
                return Some(String::from_utf16_lossy(&buffer[..end]));
            }
        }
    }
    
    None
}


unsafe extern "system" fn low_level_keyboard_proc(
    n_code: i32,
    w_param: WPARAM,
    l_param: LPARAM,
) -> LRESULT {
    if n_code >= 0 {
        let kbd_struct = *(l_param.0 as *const KBDLLHOOKSTRUCT);
        
        if w_param.0 == WM_KEYDOWN as usize {
            let vk_code = kbd_struct.vkCode;
            
            let ctrl_pressed = (GetAsyncKeyState(VK_CONTROL.0 as i32) as u16) & 0x8000 != 0;
            let shift_pressed = (GetAsyncKeyState(VK_SHIFT.0 as i32) as u16) & 0x8000 != 0;
            let alt_pressed = (GetAsyncKeyState(VK_MENU.0 as i32) as u16) & 0x8000 != 0;
            let win_pressed = (GetAsyncKeyState(VK_LWIN.0 as i32) as u16) & 0x8000 != 0 ||
                             (GetAsyncKeyState(VK_RWIN.0 as i32) as u16) & 0x8000 != 0;
            
            let modifier_keys = [VK_LCONTROL.0 as u32, VK_RCONTROL.0 as u32, 
                                VK_LSHIFT.0 as u32, VK_RSHIFT.0 as u32,
                                VK_LMENU.0 as u32, VK_RMENU.0 as u32,
                                VK_LWIN.0 as u32, VK_RWIN.0 as u32];
            let is_modifier_key = modifier_keys.contains(&vk_code);
            
            let has_modifier = ctrl_pressed || alt_pressed || win_pressed || shift_pressed;
            if has_modifier && !is_modifier_key {
                
                let mut shortcut_parts = Vec::new();
                
                if ctrl_pressed { shortcut_parts.push("Ctrl"); }
                if shift_pressed { shortcut_parts.push("Shift"); }
                if alt_pressed { shortcut_parts.push("Alt"); }
                if win_pressed { shortcut_parts.push("Win"); }
                
                let key_name = get_key_name(vk_code);
                shortcut_parts.push(&key_name);
                
                let shortcut = shortcut_parts.join("+");
                
                if let Some(last_shortcut_mutex) = LAST_SHORTCUT.get() {
                    if let Ok(mut last_shortcut) = last_shortcut_mutex.lock() {
                        if *last_shortcut == shortcut {
                            return CallNextHookEx(HHOOK(0), n_code, w_param, l_param);
                        }
                        *last_shortcut = shortcut.clone();
                    }
                }
                
                let app_name = get_foreground_process_name();
                
                if let (Some(storage), Some(device_id)) = (STORAGE.get(), DEVICE_ID.get()) {
                    let event = TimelineEvent::keyboard_shortcut(
                        device_id.clone(),
                        shortcut.clone(),
                        app_name.clone(),
                    );
                    
                    if let Err(e) = storage.insert_event(&event) {
                        error!("Failed to insert keyboard shortcut event: {}", e);
                    } else {
                        info!("Keyboard shortcut detected: {} in {:?}", shortcut, app_name);
                    }
                }
            }
        } else if w_param.0 == WM_KEYUP as usize {
            if let Some(last_shortcut_mutex) = LAST_SHORTCUT.get() {
                if let Ok(mut last_shortcut) = last_shortcut_mutex.lock() {
                    last_shortcut.clear();
                }
            }
        }
    }
    
    CallNextHookEx(HHOOK(0), n_code, w_param, l_param)
}

pub fn start_monitoring(device_id: String, storage: Arc<Storage>, running: Arc<AtomicBool>) -> Result<()> {
    STORAGE.set(storage).map_err(|_| anyhow::anyhow!("Failed to set storage"))?;
    DEVICE_ID.set(device_id).map_err(|_| anyhow::anyhow!("Failed to set device ID"))?;
    LAST_SHORTCUT.set(std::sync::Mutex::new(String::new())).map_err(|_| anyhow::anyhow!("Failed to set last shortcut"))?;
    
    unsafe {
        let hook = SetWindowsHookExW(
            WH_KEYBOARD_LL,
            Some(low_level_keyboard_proc),
            GetModuleHandleW(PCWSTR::null()).unwrap(),
            0,
        )?;
        
        info!("Shortcut monitoring started");
        
        let mut msg = MSG::default();
        while running.load(Ordering::SeqCst) {
            let ret = PeekMessageW(&mut msg, HWND(0), 0, 0, PM_REMOVE);
            if ret.as_bool() {
                if msg.message == WM_QUIT {
                    break;
                }
                TranslateMessage(&msg);
                DispatchMessageW(&msg);
            } else {
                std::thread::sleep(std::time::Duration::from_millis(10));
            }
        }
        
        UnhookWindowsHookEx(hook)?;
    }
    
    info!("Shortcut monitoring stopped");
    Ok(())
}
