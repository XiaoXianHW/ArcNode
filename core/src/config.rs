use serde::{Deserialize, Serialize};
use std::fs;
use std::path::Path;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    pub device: DeviceConfig,
    pub storage: StorageConfig,
    pub idle: IdleConfig,
    pub modules: ModulesConfig,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeviceConfig {
    pub id: String,
    pub name: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IdleConfig {
    pub threshold_seconds: u64,
    pub check_interval_seconds: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ModulesConfig {
    pub window: bool,
    pub process: bool,
    pub idle: bool,
    pub shortcut: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "type")]
pub enum StorageConfig {
    #[serde(rename = "local")]
    Local { data_dir: String },
    
    #[serde(rename = "remote")]
    Remote {
        gateway_url: String,
        token: String,
        batch_size: usize,
        flush_interval_secs: u64,
    },
}

impl Default for Config {
    fn default() -> Self {
        Config {
            device: DeviceConfig {
                id: uuid::Uuid::new_v4().to_string(),
                name: hostname::get()
                    .unwrap_or_else(|_| "unknown".into())
                    .to_string_lossy()
                    .to_string(),
            },
            storage: StorageConfig::Local {
                data_dir: "data".to_string(),
            },
            idle: IdleConfig {
                threshold_seconds: 300,
                check_interval_seconds: 10,
            },
            modules: ModulesConfig {
                window: true,
                process: true,
                idle: true,
                shortcut: true,
            },
        }
    }
}

impl Config {
    pub fn load_or_create<P: AsRef<Path>>(path: P) -> anyhow::Result<Self> {
        let path = path.as_ref();
        
        if path.exists() {
            let content = fs::read_to_string(path)?;
            let mut config: Config = toml::from_str(&content)?;
            let mut needs_update = false;
            
            if !Self::is_valid_uuid(&config.device.id) {
                config.device.id = uuid::Uuid::new_v4().to_string();
                needs_update = true;
                log::info!("Generated new device ID: {}", config.device.id);
            }
            
            if config.device.name.is_empty() || config.device.name == "my-computer" || config.device.name == "device-uuid-here" {
                config.device.name = hostname::get()
                    .unwrap_or_else(|_| "unknown".into())
                    .to_string_lossy()
                    .to_string();
                needs_update = true;
                log::info!("Updated device name: {}", config.device.name);
            }
            
            if needs_update {
                let updated_content = toml::to_string_pretty(&config)?;
                fs::write(path, updated_content)?;
                log::info!("Configuration file updated with auto-generated values");
            }
            
            Ok(config)
        } else {
            let config = Config::default();
            let content = toml::to_string_pretty(&config)?;
            
            if let Some(parent) = path.parent() {
                fs::create_dir_all(parent)?;
            }
            
            fs::write(path, content)?;
            log::info!("Created new configuration file with auto-generated values");
            Ok(config)
        }
    }
    
    fn is_valid_uuid(s: &str) -> bool {
        uuid::Uuid::parse_str(s).is_ok()
    }
    
    pub fn get_platform() -> &'static str {
        std::env::consts::OS
    }
}
