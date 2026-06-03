# ArcNode 中心化（SaaS）+ 成就系统 方案

> 配套文档：自部署强化见 `selfhost-roadmap.md`，整体架构与数据库选型见 `cloud-architecture.md`。
> 本文聚焦两件事：**（A）中心化接入与数据隔离**、**（B）计算机成就系统**。

核心原则贯穿全文：
**自部署与中心化使用完全相同的上报协议（wire protocol）。Agent 从自部署迁移到中心化，
只需改两样东西：上报地址 + 凭据。** 其余逻辑、事件 schema、批量/重试机制全部不变。

---

## A. 中心化接入

### A.1 接入模型：frpc/s 式 vs client_id/key —— 怎么选

你提到两种思路，先对比：

| 模型 | 形态 | 适合 | 不适合 |
|---|---|---|---|
| **frpc/s 式（长连接隧道）** | agent 与中心服务器维持一条持久连接（TCP/WebSocket/gRPC stream），server 端用 token 鉴别 client | 需要**服务端反向下发指令**、实时双向、穿透 NAT 直连设备 | 海量设备时长连接成本高；活动日志本就是「批量异步上报」，不需要隧道 |
| **client_id / client_key（无状态 HTTP）** | agent 用 `client_id + client_key` 换取/携带凭据，**批量 HTTPS POST** 上报，无需常驻连接 | **本场景最佳**：写多、可容忍秒级延迟、天然水平扩展、对 NAT/防火墙友好 | 纯被动，服务端要主动触达需轮询/推送通道 |

**建议：主通道用 client_id/client_key 的无状态 HTTPS 上报**（与当前自部署的 `POST /api/v1/events` 完全同构，这正是「无缝迁移」的关键）。
**未来若需要实时能力**（远程下发配置、实时在线状态、Live 时间线），再叠加一条可选的 frpc/s 式持久通道（WebSocket/gRPC），但它只承载控制信令，不承载主数据流。

这样：自部署（agent → 自己的 gateway）和中心化（agent → cloud ingest）走的是**同一套 HTTP 协议**，迁移成本趋近于零。

### A.2 凭据体系（frp token 思路的多租户版）

借鉴 frp 的「一个 token 认 client」，扩展成多租户分层：

```
用户在云端控制台
  └─ 创建「项目/空间」(= tenant)        → 得到 client_id + client_secret（项目级，像 frp token）
       └─ agent 用 client_id+secret 注册一台设备  → 换发 per-device token（短期可吊销）
            └─ 之后所有上报用 per-device token
```

- **client_id / client_secret**：项目级长期凭据，相当于 frp 的 server token，用于「把新设备登记进这个租户」。
- **per-device token**：每台设备一把，**哈希存储、可单独吊销、可设过期**。日常上报用它，泄露一台不影响全局。
- 自部署版同样支持这套（client_secret 自动生成、per-device token），所以**模型在两端一致**。

### A.3 Agent 无缝迁移流程

自部署配置（现状）：
```toml
[storage]
mode = "remote"
gateway_url = "https://my-home-server:8080"
token = "devtoken123"
```

迁移到中心化，只改地址与凭据（其余完全不动）：
```toml
[storage]
mode = "remote"
gateway_url = "https://ingest.arcnode.app"
client_id = "proj_xxx"          # 控制台创建项目时得到
client_secret = "sk_live_xxx"   # 首次注册换 per-device token 后可不再保存
```

提供 `arcnode enroll`（或 `arcnode login`）一条命令完成：用 client_id/secret 调 `/api/v1/devices/enroll` → 拿到 per-device token 落本地 → 之后照常上报。
**Agent 二进制完全相同**，single / cloud 只是配置差异。

### A.4 用户数据隔离（强隔离 + 安全）

- **全程 `tenant_id`**：所有时序行带 tenant_id（自部署恒为 local）；网关层**强制**在每条查询注入 `WHERE tenant_id=?`，集中在存储实现里，上层永远拿不到跨租户数据。
- **Postgres RLS**：元数据（账号/设备/配额）开行级安全，DB 层兜底防越权。
- **ClickHouse 隔离**：用 Row Policy + 网关强制注入双保险；分区 `PARTITION BY (tenant_id, 天)`，按租户删除/导出/计量都干净。
- **凭据隔离**：per-device token → 只能写自己 tenant 的数据；token 解析出的 tenant_id 不可被请求体覆盖（防伪造 tenant_id 越权写）。

### A.5 传输与接口安全

- **TLS 全程**（HTTPS）；可选 mTLS 给高安全客户。
- **请求签名 / 防重放**：上报带 `timestamp + nonce`，HMAC(client_key) 签名；服务端校验时间窗 + nonce 去重，防重放与篡改。
- **幂等**：事件携带 `event_id`，重发不重复入库（与自部署一致）。
- **限流/配额**：按 tenant + 按 device 限速；请求体大小上限；超额按套餐拒绝或降级。
- **密钥轮换/吊销**：client_secret 可轮换，per-device token 可即时吊销（吊销表/短 TTL）。
- **审计**：登录、enroll、token 吊销、配额变更留审计日志。

### A.6 中心化落地阶段

1. **元数据与账号**：Postgres + accounts/users/projects/devices/api_keys/quotas；登录（邮箱/OAuth）。
2. **Ingest 服务**：`/api/v1/events` 云端版（鉴权解析 tenant → 校验签名 → 入队）。
3. **事件存储**：ClickHouse 实现 `EventStore`（建表/物化视图 rollup，见 `cloud-architecture.md`）。
4. **队列削峰**：Kafka/Redis Streams/NATS → 批量 writer → ClickHouse。
5. **enroll/迁移命令**：`arcnode enroll`，控制台「添加设备」向导（可出二维码）。
6. **配额/计费/保留**：用量统计、套餐、TTL 分层。
7. **（可选）实时控制通道**：WebSocket/gRPC 下发配置、在线状态。

---

## B. 计算机成就系统（Steam 式）

> 自部署版可跑「内置成就 + 本地解锁」；中心化版独有「**全局稀有度 + 社区成就库 + 成就分享**」。
> 成就本质 = **在事件流上做规则匹配**，是现有事件的派生层，不改采集端。

### B.1 成就分类（成就库结构）

- **累计型 (cumulative)**：某程序累计使用 N 小时 / 累计敲 N 次快捷键 / 累计上线 N 天
- **里程碑型 (milestone)**：首次使用某类程序、首次连续工作 X 小时
- **行为/时段型 (behavioral)**：凌晨 3–5 点仍在编码（夜猫子）、连续 7 天打开 IDE（坚持）
- **组合型 (combo)**：一天内用 ≥10 个不同程序（多面手）、同日既写代码又长时间娱乐（劳逸结合）
- **稀有/彩蛋型 (rare)**：极低全局解锁比例的隐藏成就
- 每个成就带：`category`、`tier`（铜/银/金/铂）、`points`、`icon`、`secret(bool)`、`rarity%`（云端算）

### B.2 声明式规则 DSL（关键：安全 + 可社区扩展）

社区成就**绝不能跑用户提交的任意代码**。用**声明式 JSON 条件树**，由可信评估器解释执行：

```jsonc
{
  "id": "night_owl",
  "name": "夜猫子",
  "category": "behavioral",
  "tier": "silver",
  "points": 20,
  "secret": false,
  "rule": {
    "event": "ForegroundChange",
    "all": [
      { "field": "category", "op": "eq", "value": "Development" },
      { "field": "local_hour", "op": "between", "value": [3, 5] }
    ],
    "accumulate": { "metric": "duration_secs", "op": ">=", "value": 1800 },
    "window": "per_day"
  }
}
```

- 支持算子：`eq/neq/gt/gte/lt/lte/between/in/contains`，逻辑 `all/any/not`，聚合 `count/sum/duration`，窗口 `per_day/per_week/rolling_Nd/lifetime`。
- 评估器是白名单解释器 → 安全；社区只能提交 DSL，服务端审核后上架。

### B.3 评估器（实时 + 回算）

- **实时增量**：事件入库时过一遍匹配，命中则写 `user_achievements`（解锁）。
- **批量回算**：成就定义新增/修改时，对历史事件跑批补发（ClickHouse 上按 tenant 扫描），并对解锁状态做版本化，避免重复发/漏发。
- **进度跟踪**：未解锁成就显示进度条（如「87/100 小时」），靠聚合表/物化视图实时取数。

### B.4 数据模型

```
achievements(id, name, category, tier, points, icon, secret, rule_json, version, source)   -- 定义（内置+社区）
user_achievements(tenant_id, achievement_id, device_id, unlocked_at, progress)             -- 解锁/进度
achievement_unlock_counts(achievement_id, unlocked_users)                                   -- 全局稀有度（云端）
community_achievements(id, author, status, rule_json, reviewed_at)                          -- 社区投稿与审核
```

- **全局稀有度** = `unlocked_users / 活跃用户`，只有云端能算 → 中心化的核心吸引力。
- 自部署版用内置 `achievements`，无 `unlock_counts`（或显示「本地」徽章）。

### B.5 前端体验

- 时间线/仪表盘解锁瞬间弹 **成就 Toast**（带音效/动画，Steam 风）。
- **成就墙**页面：按类型分类、显示稀有度%、已解锁/进行中/隐藏，悬停看条件与进度。
- 个人主页可分享成就卡片（**默认私密，显式 opt-in**，分享成就名而非原始活动数据）。

### B.6 风险与红线

- **隐私**：活动数据比游戏私密得多，成就分享默认关闭、可控粒度、脱敏。
- **避免反向激励**：成就向「健康/平衡/技能成长」倾斜，慎用「连续在线 X 小时」这类鼓励沉迷的设计。
- **防作弊**：云端校验事件合理性（时间连续性、设备指纹、签名/防重放），否则稀有度失真。
- **规则版本化**：成就改动可回溯、可重算、可标注「赛季」。

### B.7 成就系统落地阶段

1. **本地 MVP（自部署即可用）**：内置 10–20 个成就 + DSL 评估器（实时匹配）+ 成就墙页面 + 解锁 Toast。先验证「看时间线顺便解锁」体验。
2. **回算与进度**：历史事件批量回算、进度条、隐藏成就。
3. **云端增强**：全局稀有度、跨设备合并成就、成就分享。
4. **社区成就库**：DSL 投稿 → 审核 → 上架，按稀有度/热度排行。

---

## 兼容性总览（一句话）

- **同协议**：自部署与中心化共用 `POST /api/v1/events` + per-device token + event_id 幂等，agent 迁移 = 改 URL + 凭据。
- **同 schema**：所有表带 `tenant_id`（自部署=local），存储走统一接口，云端只是换实现（ClickHouse/Postgres）。
- **接入**：client_id/client_secret 注册（frp token 的多租户版）→ per-device token 上报；未来按需叠加 frpc/s 式实时控制通道。
- **成就**：声明式 DSL + 安全评估器，自部署跑内置成就，中心化加全局稀有度与社区库。
