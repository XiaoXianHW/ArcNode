use anyhow::Result;
use chrono::Local;
use clap::{Parser, Subcommand};
use log::info;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::thread;
use std::time::Duration;
use core::{export_to_json, Config, Storage};

#[derive(Parser)]
#[command(name = "ArcLM Node Agent")]
#[command(about = "跨平台用户行为时间线采集器", long_about = None)]
struct Cli {
    #[command(subcommand)]
    command: Option<Commands>,
}

#[derive(Subcommand)]
enum Commands {
    Export {
        #[arg(short, long)]
        date: Option<String>,

        #[arg(short, long)]
        output: Option<String>,
    },
    
    InitConfig,
}

fn main() -> Result<()> {
    env_logger::init();
    
    let cli = Cli::parse();
    
    match &cli.command {
        Some(Commands::Export { date, output }) => {
            let export_date = date.clone()
                .unwrap_or_else(|| Local::now().format("%Y-%m-%d").to_string());
            
            let db_path = format!("data/{}.db", export_date);
            let output_path = output.clone()
                .unwrap_or_else(|| format!("export_{}.json", export_date));
            
            export_to_json(&db_path, &output_path)?;
            info!("Data exported to: {}", output_path);
            Ok(())
        }
        Some(Commands::InitConfig) => {
            let config = Config::default();
            let content = toml::to_string_pretty(&config)?;
            std::fs::write("config.toml", content)?;
            info!("Config file created: config.toml");
            Ok(())
        }
        None => {
            info!("ArcLM Node Agent starting...");
            
            let config = Config::load_or_create("config.toml")?;
            info!("Device ID: {}", config.device.id);
            info!("Device Name: {}", config.device.name);
            info!("Platform: {}", Config::get_platform());
            
            let running = Arc::new(AtomicBool::new(true));
            let r = running.clone();
            
            ctrlc::set_handler(move || {
                info!("Received shutdown signal");
                r.store(false, Ordering::SeqCst);
            }).expect("Error setting Ctrl-C handler");
            
            let storage = Arc::new(Storage::new(config.device.id.clone(), &config.storage)?);
            
            let mut started_modules = Vec::new();
            
            if config.modules.window {
                watcher_window::start_monitoring(config.device.id.clone(), storage.clone(), running.clone())?;
                started_modules.push("window");
            }
            
            if config.modules.process {
                watcher_process::start_monitoring(config.device.id.clone(), storage.clone(), running.clone())?;
                started_modules.push("process");
            }
            
            if config.modules.idle {
                watcher_idle::start_monitoring(config.device.id.clone(), storage.clone(), running.clone(), config.idle.clone())?;
                started_modules.push("idle");
            }
            
            if config.modules.shortcut {
                watcher_shortcut::start_monitoring(config.device.id.clone(), storage.clone(), running.clone())?;
                started_modules.push("shortcut");
            }

            if config.modules.system {
                watcher_system::start_monitoring(
                    config.device.id.clone(),
                    storage.clone(),
                    running.clone(),
                    config.modules.system_interval_secs,
                )?;
                started_modules.push("system");
            }

            if started_modules.is_empty() {
                info!("No monitoring modules enabled, exiting...");
                return Ok(());
            } else {
                info!("Started monitoring modules: {}", started_modules.join(", "));
            }
            
            while running.load(Ordering::SeqCst) {
                thread::sleep(Duration::from_secs(1));
            }
            
            info!("Flushing remaining events...");
            storage.flush()?;
            info!("Collector stopped");
            Ok(())
        }
    }
}
