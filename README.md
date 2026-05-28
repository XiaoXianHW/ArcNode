<div align="center">

# ArcNode

**Self-hosted personal activity timeline.**
Track every window, project, language, game, focus block and idle stretch across all your devices — then query it with SQL, the web UI, or any MCP-aware AI assistant.

[![CI](https://github.com/XiaoXianHW/ArcNode/actions/workflows/ci.yml/badge.svg)](https://github.com/XiaoXianHW/ArcNode/actions/workflows/ci.yml)
[![Release](https://github.com/XiaoXianHW/ArcNode/actions/workflows/release.yml/badge.svg)](https://github.com/XiaoXianHW/ArcNode/actions/workflows/release.yml)
![rust](https://img.shields.io/badge/agent-Rust%201.95+-orange)
![go](https://img.shields.io/badge/gateway-Go%201.23+-00ADD8)
![react](https://img.shields.io/badge/frontend-React%2018-61dafb)
![platform](https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey)

[简体中文](#chinese) · [Features](#features) · [Quick Start](#quick-start) · [REST API](#rest-api) · [MCP](#mcp-for-ai-clients) · [Architecture](#architecture)

</div>

---

## What is ArcNode?

ArcNode is a **personal time-machine** for your computer. A tiny native agent watches what window is in front, what process is running, and whether you've gone idle. A self-hosted Go gateway collects the stream from one or many devices into SQLite, auto-categorizes everything, and ships a React dashboard plus a JSON-RPC **MCP endpoint** so any AI assistant can answer questions like *"how much Rust did I write last week and which repo took the most time?"*.

Everything runs locally. No cloud. No telemetry. No analytics SaaS in the loop.

```
┌────────────┐   /api/v1/events    ┌────────────┐    /api/*     ┌──────────────┐
│  agent     │ ──────────────────▶ │  gateway   │ ────────────▶ │  frontend    │
│  (Rust)    │   bearer + batch    │  (Go +     │   bearer      │  (React)     │
│  arcowo    │                     │   SQLite)  │               │  embedded    │
└────────────┘                     └────────────┘               └──────────────┘
   one per                            single                       served by
   device                             binary                       gateway
                                          │
                                          └── /mcp  (JSON-RPC) ──▶  AI clients
                                                                    (Claude, Cursor…)
```

---

## Features

### Capture (agent)

| Module          | Source                                                   | Granularity    |
|-----------------|----------------------------------------------------------|----------------|
| Foreground      | Win32 `SetWinEventHook` / X11 `_NET_ACTIVE_WINDOW` / NSWorkspace | every window change |
| Process         | `EnumWindows` + sysinfo                                  | start / stop   |
| Idle            | `GetLastInputInfo` / X11 idle / `CGEventSourceSecondsSinceLastEventType` | configurable threshold |
| Shortcut        | `WH_KEYBOARD_LL` / XRecord / `CGEventTap`                | modifier+key combos |
| System          | sysinfo CPU + memory                                     | configurable interval (default 60s) |

### Analytics dashboard (frontend)

```
Overview        ▸  Dashboard · Profile · Timeline · Categories
Activities      ▸  Focus · Coding · Gaming · Insights · Shortcuts
Wellbeing       ▸  Wellness
System          ▸  System · Devices · Settings
```

**Focus** — deep-focus blocks (consecutive ≥25min same category), flow score (0–100), 28-day weekday×hour switch grid, session-length histogram, recent block list.
**Coding** — daily heatmap, ranked IDEs / windows / files (parsed from window titles), language breakdown (~50 langs), 14-day project/repo stacked bars.
**Gaming** — per-game annual report (sessions, avg, longest, first/last played), 90-day session distribution.
**Wellness** — active vs idle stacked bars, daily sedentary stretches (>60min red-flagged), video/livestream time per platform.
**System** — CPU + memory time series, app-pair co-occurrence ranking.
**Profile** — live device card with status dot, current window/app/category, recent apps, polled every 15s.
**Insights** — 24h×weekday heatmap, balance stacked area, language distribution.
**Settings** — light/dark theme, English/中文 i18n, custom-keyword editor, AI weekly report, CSV/JSON export, smart classification suggestions.

### Storage & query (gateway)

- **SQLite** (pure-Go `modernc.org/sqlite`, no CGO required)
- **Auto-segmenting**: adjacent foreground events <60s apart collapse into one timeline segment
- **Keyword classifier**: ~11 built-in categories (coding, gaming, video, music, communication, browsing, design, ai_tools, reading, social, languages…)
- **Custom keywords**: add / remove rules from the UI; merged live on top of defaults
- **CSV / JSON export** of any time range, scoped per device
- **AI-generated weekly report** with focus, switches, top categories / apps / languages / games

### MCP integration

The gateway exposes a single `/mcp` JSON-RPC endpoint with **27 tools** so any MCP-compatible AI client (Claude Desktop, Cursor, Continue, …) can answer questions about your own data.

---

## Quick Start

### 1. Pre-built binaries

Grab the latest `arcnode-<platform>-<arch>.zip` (or `.tar.gz`) from the [Releases page](https://github.com/XiaoXianHW/ArcNode/releases) — each archive contains both `arcowo-gateway` (server + embedded frontend) and `arcowo` (agent), with example configs and SHA256 checksums.

CI also publishes per-commit binaries on every push to main — see the [Actions tab](https://github.com/XiaoXianHW/ArcNode/actions/workflows/ci.yml).

### 2. Run the gateway

```bash
cp gateway/config.example.toml gateway/config.toml
# edit the token, listen address, db path...
./arcowo-gateway --config gateway/config.toml
```

The frontend is embedded in the binary — open `http://localhost:8080` in any browser.

### 3. Run an agent

```bash
./arcowo init-config        # writes config.toml with a UUID + hostname
# edit [storage] to point at your gateway
./arcowo                    # starts monitoring + uploading
```

### 4. Connect an AI client (optional)

Point Claude / Cursor / Continue at `https://<gateway>/mcp` with `Authorization: Bearer <token>` — see [MCP](#mcp-for-ai-clients) below.

---

## Build from source

```bash
# Single binary (frontend embedded into gateway)
make build           # → ./arcowo-gateway

# Or component-by-component:
(cd frontend && npm install && npm run build)   # outputs to gateway/web/dist/
(cd gateway  && go build -o ../arcowo-gateway .)
cargo build --workspace --release               # → target/release/arcowo
```

Requirements: **Go 1.23+**, **Node 22+**, **Rust 1.95+**. Cross-platform release archives are produced by `.github/workflows/release.yml` for Linux/macOS/Windows × amd64/arm64.

---

## Configuration

### Agent — `config.toml`

```toml
[device]
id   = "auto-generated-uuid"     # auto-filled on first run
name = "auto-detected-hostname"

[idle]
threshold_seconds      = 300     # >5min no input → idle_start
check_interval_seconds = 10

[modules]
window               = true
process              = true
idle                 = true
shortcut             = true
system               = true      # CPU / RAM sampling
system_interval_secs = 60

# Local SQLite (one db per day in data/YYYY-MM-DD.db)
[storage]
type     = "local"
data_dir = "data"

# OR remote gateway
# [storage]
# type                = "remote"
# gateway_url         = "http://localhost:8080"
# token               = "your-gateway-token-here"
# batch_size          = 50
# flush_interval_secs = 3
```

### Gateway — `config.toml`

```toml
listen              = ":8080"
token               = "change-me-to-a-secure-token"
db_path             = "./gateway.db"
segment_gap_seconds = 60         # foreground events <this collapse into one segment

[categories]
coding   = ["code", "cursor", "intellij", "vim", ...]
gaming   = ["steam", "league of legends", "genshin", ...]
video    = ["bilibili", "youtube", "netflix", ...]
# ...full list in gateway/config.example.toml
```

You can also add / delete categories at runtime from **Settings → Custom keywords** in the UI; these are stored in SQLite and merged on top of the file rules.

---

## REST API

All endpoints sit under `/api/v1` and require `Authorization: Bearer <token>`. Time params are unix seconds. Most stat endpoints accept `device_id` (omit for all devices), `days` (default 7/30 depending on endpoint), or explicit `start` / `end`.

### Ingest

| Method | Path                  | Description                          |
|--------|-----------------------|--------------------------------------|
| POST   | `/api/v1/init`        | Register a device with sysinfo       |
| POST   | `/api/v1/events`      | Push a batch of timeline events      |

### Inventory

| Method | Path                       | Description                  |
|--------|----------------------------|------------------------------|
| GET    | `/api/v1/devices`          | List devices                 |
| GET    | `/api/v1/devices/:id`      | Single device + sysinfo      |
| GET    | `/api/v1/devices/:id/live` | Latest window / idle / recent apps (for the profile card) |
| GET    | `/api/v1/events`           | Raw events (paged)           |
| GET    | `/api/v1/segments`         | Merged timeline segments     |

### Basic stats

| Path                          | Description                              |
|-------------------------------|------------------------------------------|
| `/api/v1/stats/categories`    | Category time totals                     |
| `/api/v1/stats/apps`          | Top processes                            |
| `/api/v1/stats/shortcuts`     | Top shortcut combos                      |
| `/api/v1/stats/summary`       | Daily summary card                       |
| `/api/v1/stats/daily`         | 30-day bar series for a category         |
| `/api/v1/stats/heatmap`       | GitHub-style year heatmap                |
| `/api/v1/stats/hourly`        | 24h × weekday heatmap                    |
| `/api/v1/stats/balance`       | Stacked categories per day               |
| `/api/v1/stats/projects`      | Top window titles (per category)         |
| `/api/v1/stats/languages`     | Programming language distribution        |

### Advanced analytics

| Path                              | Description                                                |
|-----------------------------------|------------------------------------------------------------|
| `/api/v1/stats/focus`             | Deep-focus blocks (≥`min_duration` same category)          |
| `/api/v1/stats/switches`          | Window-switch frequency (hour + day)                       |
| `/api/v1/stats/flow`              | Flow score 0–100 per day                                   |
| `/api/v1/stats/sessions`          | Session-length histogram                                   |
| `/api/v1/stats/files`             | Top files (parsed from IDE titles)                         |
| `/api/v1/stats/projects-daily`    | Per-day project / repo time                                |
| `/api/v1/stats/app-pairs`         | App co-occurrence (Sankey / matrix)                        |
| `/api/v1/stats/video`             | YouTube / Bilibili / Twitch / Netflix time                 |
| `/api/v1/stats/idle-ratio`        | Active vs idle per day                                     |
| `/api/v1/stats/sedentary`         | Stretches > `threshold` seconds with no idle               |
| `/api/v1/stats/suggestions`       | Frequent un-categorized processes                          |
| `/api/v1/stats/system`            | CPU + memory samples                                       |
| `/api/v1/stats/games`             | Per-game annual report                                     |
| `/api/v1/stats/weekly-report`     | Pre-computed AI weekly summary inputs                      |

### Configuration

| Method  | Path                              | Description                       |
|---------|-----------------------------------|-----------------------------------|
| GET     | `/api/v1/categories`              | Active classifier rules           |
| GET     | `/api/v1/custom-keywords`         | List custom keywords              |
| POST    | `/api/v1/custom-keywords`         | Add `{category, keyword}`         |
| DELETE  | `/api/v1/custom-keywords/:id`     | Remove                            |

### Export

| Path                            | Description                          |
|---------------------------------|--------------------------------------|
| `/api/v1/export/segments.csv`   | Filtered segments as CSV             |
| `/api/v1/export/events.json`    | Raw events as JSON (paged)           |

---

## MCP (for AI clients)

ArcNode ships a JSON-RPC 2.0 MCP server at `POST /mcp`. Any MCP-compatible client can connect over HTTP with a bearer token. List tools:

```bash
curl -sX POST http://localhost:8080/mcp \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

Call one:

```bash
curl -sX POST http://localhost:8080/mcp \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call",
       "params":{"name":"get_weekly_report","arguments":{"days":7}}}'
```

### Available tools

```
list_devices         get_summary           get_categories       get_apps
get_segments         get_heatmap           get_daily            get_projects
get_languages        get_hourly            get_balance          get_rules
list_custom_keywords get_focus_blocks      get_flow             get_switches
get_sessions         get_files             get_app_pairs        get_video_time
get_idle_ratio       get_sedentary         get_suggestions      get_system_samples
get_games            get_live_status       get_weekly_report
```

### Claude Desktop example

```json
{
  "mcpServers": {
    "arcnode": {
      "url": "http://localhost:8080/mcp",
      "headers": { "Authorization": "Bearer <token>" }
    }
  }
}
```

Then ask: *"Using arcnode, summarize last week — top languages, longest focus block, and total gaming time."*

---

## Architecture

```
ArcNode/
├── core/              # Rust shared lib — events, db, storage, sysinfo, config
├── app/               # Rust CLI (arcowo) — default monitor / export / init-config
├── watcher-window/    # foreground window watcher (per-OS impl)
├── watcher-process/   # process lifecycle watcher
├── watcher-idle/      # idle detection
├── watcher-shortcut/  # global shortcut hook
├── watcher-system/    # CPU / RAM sampler (new in v3)
│
├── gateway/           # Go service
│   ├── api/           # ingest, query, analytics, mcp, export handlers
│   ├── storage/       # SQLite store, analytics store
│   ├── category/      # classifier (file rules + custom-keywords overlay)
│   ├── config/        # TOML loader
│   ├── middleware/    # bearer auth
│   └── web/           # //go:embed frontend dist
│
├── frontend/          # React + Vite + TS + Tailwind, xAI/Vercel-style UI
│   └── src/pages/     # Dashboard · Profile · Timeline · Categories · Focus
│                        Coding · Gaming · Insights · Shortcuts · Wellness
│                        System · Devices · Settings
│
├── .github/workflows/ # ci.yml (per-PR build + artifacts), release.yml (tagged)
└── Makefile           # `make build` = frontend + gateway, single binary
```

**Why two languages?** Rust gives the agent native OS hooks (Win32, X11, NSWorkspace) with zero overhead. Go gives the gateway a single static binary with embedded frontend, easy HTTP / SQLite / MCP plumbing, and effortless cross-compilation. The agent talks to the gateway over a tiny REST surface (`/init` + `/events`) — they can also run on the same machine or across a LAN.

---

## Privacy

- 100% self-hosted. The agent only talks to the gateway you configure.
- No third-party telemetry, no analytics SDK, no remote logging.
- All data lives in `gateway.db` (SQLite) on your machine.
- Bearer token authentication on every endpoint.
- Export everything to CSV / JSON at any time; delete the DB to wipe.

---

## Roadmap

- [x] **agent-android** — Kotlin foreground service polling `UsageStatsManager` (see [`android/`](android/))
- [ ] **agent-ios** — Shortcuts / Screen Time integration
- [ ] **Browser extension** — domain-level breakdown for Chrome / Firefox
- [ ] **Local LLM RAG** — `ask("what did I do Wednesday afternoon?")` via on-device model
- [ ] **Git activity hook** — correlate commits with focus blocks
- [ ] **Public sharing cards** — opt-in "Year Wrapped" image generator

---

## Contributing

PRs welcome. Keep code minimal, follow existing patterns (modular folders, camelCase Go files, kebab/PascalCase TS files), and avoid dragging in new heavy deps. Lint commands:

```bash
(cd gateway  && go vet ./... && go build ./...)
(cd frontend && npm run build)        # tsc strict + vite
cargo build --workspace
```

---

## License

MIT — see [LICENSE](LICENSE) if present, otherwise the repo defaults apply.

---

<a id="chinese"></a>

## 简体中文

ArcNode 是一个**完全本地、自托管的个人电脑活动时间线**：原生 Agent 监控前台窗口 / 进程 / 空闲 / 快捷键 / 系统资源，Go 网关把数据落到 SQLite 并提供分类 / 查询 / 统计 / 导出 API，React 前端给出可视化看板，外加一个 `/mcp` JSON-RPC 端点让 Claude / Cursor 等 AI 客户端直接查询你的数据。

### 主要特性

- 🪟 跨平台 Agent（Windows / macOS / Linux），用原生 API 监控前台、进程、空闲、快捷键
- 🧠 ~250 条游戏关键词 + ~50 种 IDE / AI IDE / 编程语言识别，可在前端自定义关键词
- ⏱️ **Focus / Flow / 切换频率 / Session 直方图** 等深度专注力分析
- 📁 IDE 标题反推**文件 / 项目 / 仓库**维度时长
- 🎮 **单游戏年度报告** + Session 分布
- 🩺 **屏幕时间 / Idle 比例 / 久坐警报** Wellness 看板
- 🖥️ **CPU / RAM 时间线** + App pair 共现
- 👤 **设备实时资料卡**（在线状态 / 当前窗口 / 最近活动）
- 🤖 **AI 周报** + 智能分类建议（一键加规则）
- 📤 CSV / JSON 任意时段导出
- 🔌 27 个 **MCP 工具**，让 AI 助手直接读你的数据
- 🌗 浅 / 深主题 + 中英文 i18n
- 📦 单一二进制（前端 embed 进网关）+ Release 包含全平台 Agent

### 快速上手

1. 从 [Releases](https://github.com/XiaoXianHW/ArcNode/releases) 下载对应平台的压缩包
2. `cp gateway/config.example.toml gateway/config.toml` 改 token / 端口 / db 路径
3. `./arcowo-gateway --config gateway/config.toml` 启动网关，访问 `http://localhost:8080`
4. `./arcowo init-config` 生成 Agent 配置，把 `[storage]` 切到远端网关并运行 `./arcowo`
5. AI 客户端可指向 `https://<gateway>/mcp`，Header `Authorization: Bearer <token>`

详细 API / MCP / 架构请看上方英文部分对应章节。
