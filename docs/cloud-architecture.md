# ArcNode：自部署 + 中心化云端（类 Rybbit）架构方案

目标：同一份代码既能像现在这样单机/家庭自部署，也能做成统一云端版（SaaS），
为每个用户/团队分配独立的数据空间，并能扛住活动日志这类「写多、量大、按时间查」的负载。

---

## 0. 现状（基线）

- 网关：Go + `modernc.org/sqlite`，单文件 SQLite（`Store{ DB *sql.DB }`，`SetMaxOpenConns(1)`）。
- 表：`devices` / `events`(id, device_id, timestamp, event_type, category, process_name,
  window_title, pid, metadata TEXT) / `segments` / `custom_keywords`。
- 鉴权：单一 Bearer token（`devtoken123`）——**没有用户概念**，所有数据共享一个库。
- 前端内嵌、按 device_id + 时间区间查询。

要做云端版，核心要补三块：**多租户模型**、**存储分层（元数据 vs 时序日志）**、**摄取/配额/保留**。

---

## 1. 双形态：一套代码、两种部署 profile

参考 Rybbit/Plausible 的做法——**不要 fork 两套**，用「部署模式」开关 + 存储接口隔离差异：

```
ARCNODE_MODE = "single" | "cloud"
```

- `single`（自部署默认）：嵌入式存储，零外部依赖，开箱即用。
- `cloud`：外置 Postgres（元数据/账号）+ ClickHouse（事件时序），可水平扩展。

关键是把存储抽象成接口，业务代码不关心底下是 SQLite 还是 ClickHouse：

```go
type EventStore interface {
    InsertEvents(ctx context.Context, tenantID string, evts []Event) error
    QueryTimeline(ctx context.Context, q TimelineQuery) ([]Segment, error)
    Summary(ctx context.Context, q SummaryQuery) (Summary, error)
    SystemSamples(ctx context.Context, q RangeQuery) ([]Sample, error)
}

type MetaStore interface {           // 账号/设备/配置/配额
    CreateAccount(...) ; CreateDevice(...) ; Quota(tenantID) ...
}
```

- `single` 模式：`EventStore` 与 `MetaStore` 都走 SQLite（或 DuckDB，见 §3）。
- `cloud` 模式：`MetaStore` → Postgres，`EventStore` → ClickHouse。

现在 `gateway/storage/*.go` 里直接拼 SQL 的逻辑，第一步先收敛到这两个接口后面，
后面换库就只是加一个实现，而不是改动 handler。

---

## 2. 多租户模型（每个用户独立空间）

### 2.1 实体层级

```
Account（计费主体 / 个人或团队）
 └─ User（登录身份，多对多到 Account）
     └─ Device（agent，归属某个 Account）
         └─ Event / Sample / Segment（带 account_id 列）
```

引入稳定的 `tenant_id`（= account_id）。**所有时序行都带 `tenant_id`**，这是隔离与配额的基石。

### 2.2 隔离策略三选一

| 方案 | 隔离强度 | 成本/运维 | 适用 |
|---|---|---|---|
| **A. 共享表 + tenant_id 行级隔离** | 中（靠代码/RLS 保证） | 最低，最易扩展 | **云端推荐**（海量小租户） |
| B. schema/database per tenant | 高 | 中（连接数、迁移成本上升） | 大客户/合规隔离 |
| C. 物理实例 per tenant | 最高 | 最高 | 私有化大单 |

推荐 **A 为默认**，对企业大客户再叠加 B/C。

- Postgres 侧：用 **Row Level Security (RLS)**，会话设 `SET app.tenant_id = ...`，
  策略 `USING (tenant_id = current_setting('app.tenant_id'))`，从 DB 层兜底防越权。
- ClickHouse 侧：CH 没有 RLS，但有 **Row Policy**（`CREATE ROW POLICY ... USING tenant_id = ...`）。
  实务上更稳的是**强制在网关层给每条查询注入 `WHERE tenant_id = ?`**，并集中在 EventStore 实现里，
  绝不让上层手拼 SQL（避免漏加租户条件 = 数据串户）。

### 2.3 鉴权升级

- 用户：邮箱/OAuth 登录（Postgres 存身份），签发短期 JWT 给前端。
- 设备 agent：每台设备一把 **per-device API key / ingest token**（可单独吊销），
  替换现在写死的 `devtoken123`。token → tenant_id 映射缓存在内存/Redis。

---

## 3. 数据库选型：把数据拆成两类

活动/日志数据的特征：**写多读少、追加为主、几乎不更新、按时间+维度聚合查询、量随时间线性膨胀**。
这正是**列式 / 时序数据库**的主场，行存 OLTP 库（SQLite/Postgres 原生表）会越来越吃力。

**核心建议：元数据用 Postgres（行存、强一致、好做账号/配额/RLS），事件时序用 ClickHouse（列存、高压缩、聚合快）。**

### 3.1 候选对比

| 引擎 | 模型 | 压缩 | 聚合查询 | 运维 | 在 ArcNode 的角色 |
|---|---|---|---|---|---|
| **ClickHouse** | 列存 OLAP | **极好（5–15×，低基数列更高）** | **极快**（向量化、物化视图） | 中（需独立部署/运维） | **云端事件主库 ✅** |
| TimescaleDB | Postgres 扩展（hypertable） | 好（原生压缩 ~90%） | 好 | 低（就是 Postgres） | 想少一个组件时的折中方案 |
| DuckDB | 嵌入式列存 | 好 | 好（单机分析） | 极低（嵌入进程） | **自部署版替代 SQLite ✅** |
| SQLite（现状） | 嵌入式行存 | 一般 | 小数据量够用 | 极低 | 保留给最小自部署 |
| Postgres 原生 | 行存 OLTP | 一般 | 中 | 低 | **元数据/账号/配额 ✅** |

结论：
- **云端版**：ClickHouse（事件） + Postgres（账号、设备、配置、配额、计费）。
  这正是 Rybbit / Plausible / PostHog 的同款组合，已被验证能扛十亿级事件。
- **想省一个组件**：用 **TimescaleDB**（一个 Postgres 同时放元数据 + 时序），
  规模中等时运维最省心，后续真撑不住再迁 ClickHouse。
- **自部署版**：默认仍可 SQLite；想要更好的压缩/分析速度，把嵌入式引擎换成 **DuckDB**，
  零外部依赖却有列存的压缩与聚合优势——非常契合「单机但数据量大」的个人用户。

### 3.2 ClickHouse 事件表设计（关键在压缩与分区）

```sql
CREATE TABLE events
(
    tenant_id     UInt64,
    device_id     UUID,
    ts            DateTime64(3),                 -- 毫秒
    event_type    LowCardinality(String),        -- 枚举值少 → 极致压缩
    category      LowCardinality(String),
    process_name  LowCardinality(String),
    window_title  String           CODEC(ZSTD(3)),
    pid           UInt32,
    metadata      String           CODEC(ZSTD(3))  -- 或拆成具体列/JSON 类型
)
ENGINE = MergeTree
PARTITION BY (tenant_id, toYYYYMMDD(ts))           -- 按租户+天分区，便于按租户删/导出
ORDER BY (tenant_id, device_id, ts)                -- 主键即排序键，时间范围查询走 PK
TTL toDateTime(ts) + INTERVAL 90 DAY               -- 自动过期（按套餐分层）
SETTINGS index_granularity = 8192;
```

压缩与提速要点：
- `LowCardinality(String)`：event_type / category / process_name 取值有限，列存字典编码后体积极小。
- `CODEC(ZSTD)`：长文本（window_title / metadata）显式 ZSTD，吞吐与压缩比平衡好；
  时间戳可用 `DoubleDelta`、单调列用 `Delta` 编码。
- `ORDER BY (tenant_id, device_id, ts)`：把「某设备某段时间」做成顺序扫描，I/O 最小。
- **物化视图做 rollup**：把秒级原始事件预聚合成「按小时/天 × category 的时长」，
  前端的 Dashboard/Categories/Timeline 直接查聚合表，毫秒级返回，原始明细按 TTL 过期：

```sql
CREATE MATERIALIZED VIEW mv_daily_category
ENGINE = SummingMergeTree
PARTITION BY (tenant_id, toYYYYMM(day))
ORDER BY (tenant_id, device_id, day, category) AS
SELECT tenant_id, device_id, toDate(ts) AS day, category,
       count() AS events, sum(duration) AS total_secs
FROM segments GROUP BY tenant_id, device_id, day, category;
```

### 3.3 元数据（Postgres）

`accounts / users / memberships / devices / api_keys / quotas / plans / billing`，
开 RLS，常规迁移工具（goose/atlas）管 schema。这些数据小、要强一致、要事务——Postgres 最合适。

---

## 4. 摄取管线（云端要顶住写入峰值）

现在是 agent 直接 `POST /events` → 同步写库。云端规模下建议：

```
agent → [Ingest API（鉴权、限流、校验、按 token 解析 tenant_id）]
      → 缓冲队列（Kafka / Redis Streams / NATS JetStream）
      → 批量 writer → ClickHouse（批量 INSERT，几千~几万行/批）
```

- **批量写**：ClickHouse 最忌小批量高频 INSERT；用队列攒批（按时间/条数触发），
  也天然削峰、与查询解耦。
- **幂等**：事件带 `(device_id, ts, seq)` 或客户端生成的 `event_id`，重发不重复（CH 用 ReplacingMergeTree 或入库前去重）。
- **背压/限流**：按 tenant 限速，保护多租户互不影响。
- 自部署 `single` 模式可跳过队列，agent → 直接批量写嵌入库，保持简单。

> 你现在已经做的 agent 端「后台定时 flush + 失败重入队 + 批量上报」正是这条路的雏形，方向对。

---

## 5. 每用户存储配额与数据保留

- **配额**：`quotas(tenant_id, max_events, max_storage_bytes, retention_days, max_devices)`。
  ClickHouse 可用 `system.parts` 按 `partition`(含 tenant_id) 统计每租户实际占用字节，用于计费/限额。
- **分层保留（省空间的关键）**：
  - 原始明细短 TTL（如 30/90 天），到期自动删。
  - rollup 聚合长留（按天的统计很小，可留 1~数年）。
  - 套餐越高，retention 越长——既是产品分层也是成本控制。
- **冷归档**：旧分区可导出到对象存储（S3/MinIO）做 Parquet 冷备，ClickHouse 支持 S3 表引擎按需查。

---

## 6. 落地路线（分阶段，避免一步到位翻车）

1. **抽接口**：把 `gateway/storage` 收敛到 `EventStore` / `MetaStore` 接口，现有 SQLite 作为第一个实现，
   所有查询强制带 `tenant_id`（单机模式 tenant 固定为 0）。— *无行为变化，纯重构，风险低*。
2. **加多租户元数据**：引入 accounts/users/devices/api_keys，登录 + per-device token，替换写死 token。
3. **加 ClickHouse 实现**：`EventStore` 增加 CH 实现 + 物化视图 rollup；`cloud` 模式启用。
4. **加摄取队列**：Ingest API + Kafka/Redis Streams + 批量 writer。
5. **配额/保留/计费**：TTL 分层、用量统计、套餐。
6. **（可选）自部署升级**：把嵌入式从 SQLite 换/可选 DuckDB，给数据量大的单机用户更好压缩与查询。

---

## 7. 一句话结论

- **形态**：一套代码 + `mode=single|cloud` + 存储接口隔离，自部署与 SaaS 共存（Rybbit 同思路）。
- **隔离**：account 级 `tenant_id` 行级隔离为主（Postgres RLS + 网关强制注入），大客户再上 schema/实例隔离。
- **数据库**：**Postgres 管账号/配额（强一致），ClickHouse 管海量事件（列存、ZSTD+LowCardinality、按租户+天分区、物化视图 rollup、TTL 分层）**；
  想少运维一个组件就用 TimescaleDB 折中；自部署单机版用 DuckDB 替代 SQLite 兼顾省空间与查询速度。
