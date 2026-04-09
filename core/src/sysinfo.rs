use serde::{Deserialize, Serialize};
use sysinfo::{Disks, Networks, System};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SystemInfo {
    pub cpu_brand: String,
    pub cpu_cores: usize,
    pub total_memory: u64,
    pub total_disk: u64,
    pub os_name: String,
    pub os_version: String,
    pub architecture: String,
    pub boot_time: u64,
    pub uptime: u64,
    pub network_upload: u64,
    pub network_download: u64,
}

impl SystemInfo {
    pub fn collect() -> Self {
        let mut sys = System::new_all();
        sys.refresh_all();
        
        let networks = Networks::new_with_refreshed_list();
        let (total_rx, total_tx) = networks.iter().fold((0, 0), |(rx, tx), (_, data)| {
            (rx + data.total_received(), tx + data.total_transmitted())
        });
        
        let cpu_brand = sys
            .cpus()
            .first()
            .map(|cpu| cpu.brand().to_string())
            .unwrap_or_else(|| "Unknown".to_string());
        
        let os_name = System::name().unwrap_or_else(|| "Unknown".to_string());
        let os_version = System::os_version().unwrap_or_else(|| "Unknown".to_string());
        
        let disks = Disks::new_with_refreshed_list();
        let total_disk = disks
            .list()
            .iter()
            .map(|disk| disk.total_space())
            .sum();
        
        Self {
            cpu_brand,
            cpu_cores: sys.cpus().len(),
            total_memory: sys.total_memory(),
            total_disk,
            os_name,
            os_version,
            architecture: std::env::consts::ARCH.to_string(),
            boot_time: System::boot_time(),
            uptime: System::uptime(),
            network_upload: total_tx,
            network_download: total_rx,
        }
    }
}
