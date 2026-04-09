use serde::{Deserialize, Serialize};
use serde_json::{Map, Value};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum EventType {
    ForegroundChange,
    ProcessStart,
    ProcessExit,
    IdleStart,
    IdleEnd,
    KeyboardShortcut,
}

impl EventType {
    pub fn as_str(&self) -> &str {
        match self {
            EventType::ForegroundChange => "foreground_change",
            EventType::ProcessStart => "process_start",
            EventType::ProcessExit => "process_exit",
            EventType::IdleStart => "idle_start",
            EventType::IdleEnd => "idle_end",
            EventType::KeyboardShortcut => "keyboard_shortcut",
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TimelineEvent {
    pub device_id: String,
    pub timestamp: i64,
    pub event_type: EventType,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub category: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub metadata: Option<Map<String, Value>>,
}

impl TimelineEvent {
    pub fn new(device_id: String, event_type: EventType) -> Self {
        Self {
            device_id,
            timestamp: chrono::Utc::now().timestamp(),
            event_type,
            category: None,
            metadata: None,
        }
    }

    pub fn with_metadata(mut self, metadata: Map<String, Value>) -> Self {
        self.metadata = Some(metadata);
        self
    }

    pub fn with_category(mut self, category: String) -> Self {
        self.category = Some(category);
        self
    }

    pub fn new_legacy(
        device_id: String,
        event_type: EventType,
        process_name: String,
        window_title: Option<String>,
        pid: u32,
    ) -> Self {
        let mut metadata = Map::new();
        metadata.insert("process_name".to_string(), Value::String(process_name));
        if let Some(title) = window_title {
            metadata.insert("window_title".to_string(), Value::String(title));
        }
        metadata.insert("pid".to_string(), Value::Number(serde_json::Number::from(pid)));
        
        Self {
            device_id,
            timestamp: chrono::Utc::now().timestamp(),
            event_type,
            category: None,
            metadata: Some(metadata),
        }
    }

    pub fn keyboard_shortcut(device_id: String, shortcut: String, application: Option<String>) -> Self {
        let mut metadata = Map::new();
        metadata.insert("shortcut".to_string(), Value::String(shortcut));
        if let Some(app) = application {
            metadata.insert("application".to_string(), Value::String(app));
        }
        metadata.insert("count".to_string(), Value::Number(serde_json::Number::from(1)));
        
        Self {
            device_id,
            timestamp: chrono::Utc::now().timestamp(),
            event_type: EventType::KeyboardShortcut,
            category: Some("input".to_string()),
            metadata: Some(metadata),
        }
    }

    pub fn idle(device_id: String, event_type: EventType) -> Self {
        Self {
            device_id,
            timestamp: chrono::Utc::now().timestamp(),
            event_type,
            category: Some("idle".to_string()),
            metadata: None,
        }
    }

    pub fn process_name(&self) -> Option<&str> {
        self.metadata
            .as_ref()?
            .get("process_name")?
            .as_str()
    }

    pub fn window_title(&self) -> Option<&str> {
        self.metadata
            .as_ref()?
            .get("window_title")?
            .as_str()
    }

    pub fn pid(&self) -> Option<u32> {
        self.metadata
            .as_ref()?
            .get("pid")?
            .as_u64()
            .map(|n| n as u32)
    }
}
