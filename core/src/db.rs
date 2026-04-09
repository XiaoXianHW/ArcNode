use anyhow::Result;
use rusqlite::{params, Connection};
use serde::{Deserialize, Serialize};
use std::path::Path;
use std::sync::{Arc, Mutex};

use crate::events::TimelineEvent;

pub struct DbManager {
    conn: Arc<Mutex<Connection>>,
}

impl DbManager {
    pub fn new(db_path: &str) -> Result<Self> {
        if let Some(parent) = Path::new(db_path).parent() {
            std::fs::create_dir_all(parent)?;
        }
        
        let conn = Connection::open(db_path)?;
        Ok(Self {
            conn: Arc::new(Mutex::new(conn)),
        })
    }
    
    pub fn init(&self) -> Result<()> {
        let conn = self.conn.lock().unwrap();
        
        conn.execute(
            "CREATE TABLE IF NOT EXISTS app_events (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                timestamp INTEGER NOT NULL,
                event_type TEXT NOT NULL,
                process_name TEXT,
                window_title TEXT,
                pid INTEGER
            )",
            [],
        )?;
        
        conn.execute(
            "CREATE TABLE IF NOT EXISTS timeline_segments (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                app TEXT NOT NULL,
                start_time INTEGER,
                end_time INTEGER,
                duration_seconds INTEGER
            )",
            [],
        )?;
        
        conn.execute(
            "CREATE INDEX IF NOT EXISTS idx_timestamp ON app_events(timestamp)",
            [],
        )?;
        
        conn.execute(
            "CREATE INDEX IF NOT EXISTS idx_event_type ON app_events(event_type)",
            [],
        )?;
        
        Ok(())
    }
    
    pub fn insert_event(&self, event: &TimelineEvent) -> Result<()> {
        let conn = self.conn.lock().unwrap();
        
        conn.execute(
            "INSERT INTO app_events (timestamp, event_type, process_name, window_title, pid)
             VALUES (?1, ?2, ?3, ?4, ?5)",
            params![
                event.timestamp,
                event.event_type.as_str(),
                event.process_name(),
                event.window_title(),
                event.pid(),
            ],
        )?;
        
        self.update_timeline_segment(&conn, event)?;
        
        Ok(())
    }
    
    fn update_timeline_segment(&self, conn: &Connection, event: &TimelineEvent) -> Result<()> {
        let process_name = event.process_name().unwrap_or("unknown");
        let window_title = event.window_title().unwrap_or(process_name);
        let app_key = format!("{}|{}", process_name, window_title);
        
        let mut stmt = conn.prepare(
            "SELECT id, start_time, end_time FROM timeline_segments 
             WHERE app = ?1 
             ORDER BY end_time DESC LIMIT 1"
        )?;
        
        let result: Result<(i64, i64, i64), _> = stmt.query_row(params![&app_key], |row| {
            Ok((row.get(0)?, row.get(1)?, row.get(2)?))
        });
        
        match result {
            Ok((id, _start_time, end_time)) => {
                let time_gap = event.timestamp - end_time;
                
                if time_gap <= 60 {
                    conn.execute(
                        "UPDATE timeline_segments 
                         SET end_time = ?1, duration_seconds = ?1 - start_time 
                         WHERE id = ?2",
                        params![event.timestamp, id],
                    )?;
                } else {
                    conn.execute(
                        "INSERT INTO timeline_segments (app, start_time, end_time, duration_seconds)
                         VALUES (?1, ?2, ?2, 0)",
                        params![&app_key, event.timestamp],
                    )?;
                }
            }
            Err(_) => {
                conn.execute(
                    "INSERT INTO timeline_segments (app, start_time, end_time, duration_seconds)
                     VALUES (?1, ?2, ?2, 0)",
                    params![&app_key, event.timestamp],
                )?;
            }
        }
        
        Ok(())
    }
    
    pub fn clone_handle(&self) -> Self {
        Self {
            conn: Arc::clone(&self.conn),
        }
    }
}

impl Clone for DbManager {
    fn clone(&self) -> Self {
        self.clone_handle()
    }
}

#[derive(Debug, Serialize, Deserialize)]
struct ExportEvent {
    timestamp: i64,
    datetime: String,
    event_type: String,
    process_name: String,
    window_title: Option<String>,
    pid: u32,
}

#[derive(Debug, Serialize, Deserialize)]
struct ExportSegment {
    app: String,
    start_time: i64,
    start_datetime: String,
    end_time: i64,
    end_datetime: String,
    duration_seconds: i64,
    duration_readable: String,
}

#[derive(Debug, Serialize, Deserialize)]
struct ExportData {
    export_date: String,
    total_events: usize,
    total_segments: usize,
    events: Vec<ExportEvent>,
    segments: Vec<ExportSegment>,
}

pub fn export_to_json(db_path: &str, output_path: &str) -> Result<()> {
    use chrono::{DateTime, Local, Utc};
    
    let conn = Connection::open(db_path)?;
    
    let mut events = Vec::new();
    let mut stmt = conn.prepare(
        "SELECT timestamp, event_type, process_name, window_title, pid 
         FROM app_events 
         ORDER BY timestamp"
    )?;
    
    let event_iter = stmt.query_map([], |row| {
        let timestamp: i64 = row.get(0)?;
        let datetime = DateTime::<Utc>::from_timestamp(timestamp, 0)
            .map(|dt| dt.format("%Y-%m-%d %H:%M:%S").to_string())
            .unwrap_or_else(|| String::from("Invalid"));
        
        Ok(ExportEvent {
            timestamp,
            datetime,
            event_type: row.get(1)?,
            process_name: row.get(2)?,
            window_title: row.get(3)?,
            pid: row.get(4)?,
        })
    })?;
    
    for event in event_iter {
        events.push(event?);
    }
    
    let mut segments = Vec::new();
    let mut stmt = conn.prepare(
        "SELECT app, start_time, end_time, duration_seconds 
         FROM timeline_segments 
         ORDER BY start_time"
    )?;
    
    let segment_iter = stmt.query_map([], |row| {
        let start_time: i64 = row.get(1)?;
        let end_time: i64 = row.get(2)?;
        let duration: i64 = row.get(3)?;
        
        let start_datetime = DateTime::<Utc>::from_timestamp(start_time, 0)
            .map(|dt| dt.format("%Y-%m-%d %H:%M:%S").to_string())
            .unwrap_or_else(|| String::from("Invalid"));
        
        let end_datetime = DateTime::<Utc>::from_timestamp(end_time, 0)
            .map(|dt| dt.format("%Y-%m-%d %H:%M:%S").to_string())
            .unwrap_or_else(|| String::from("Invalid"));
        
        let hours = duration / 3600;
        let minutes = (duration % 3600) / 60;
        let seconds = duration % 60;
        let duration_readable = format!("{}h {}m {}s", hours, minutes, seconds);
        
        Ok(ExportSegment {
            app: row.get(0)?,
            start_time,
            start_datetime,
            end_time,
            end_datetime,
            duration_seconds: duration,
            duration_readable,
        })
    })?;
    
    for segment in segment_iter {
        segments.push(segment?);
    }
    
    let export_data = ExportData {
        export_date: Local::now().format("%Y-%m-%d").to_string(),
        total_events: events.len(),
        total_segments: segments.len(),
        events,
        segments,
    };
    
    std::fs::write(output_path, serde_json::to_string_pretty(&export_data)?)?;
    Ok(())
}
