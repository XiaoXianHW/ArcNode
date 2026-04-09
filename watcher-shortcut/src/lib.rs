use anyhow::Result;
use log::info;
#[cfg(not(any(windows, target_os = "linux", target_os = "macos")))]
use log::error;
use std::sync::atomic::AtomicBool;
use std::sync::Arc;
use core::Storage;

#[cfg(windows)]
mod windows;
#[cfg(target_os = "linux")]  
mod linux;
#[cfg(target_os = "macos")]
mod macos;

pub fn start_monitoring(device_id: String, storage: Arc<Storage>, running: Arc<AtomicBool>) -> Result<()> {
    info!("Starting shortcut monitoring...");
    
    #[cfg(windows)]
    windows::start_monitoring(device_id, storage, running)?;
    
    #[cfg(target_os = "linux")]
    linux::start_monitoring(device_id, storage, running)?;
    
    #[cfg(target_os = "macos")]
    macos::start_monitoring(device_id, storage, running)?;
    
    #[cfg(not(any(windows, target_os = "linux", target_os = "macos")))]
    {
        error!("Shortcut monitoring not supported on this platform");
        return Err(anyhow::anyhow!("Unsupported platform"));
    }
    
    Ok(())
}
