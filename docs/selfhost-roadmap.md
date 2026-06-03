# ArcNode 自部署强化路线图

目标顺序：**先把「无中心化、无成就」的自部署版做扎实**（多设备数据处理 + 安全 + 传输加密 + 一键跑起来），
同时在设计上为未来中心化预留无缝兼容。然后单独宣传 ArcNode 自部署版，再做中心化另行推广。

> 命名统一：项目中残留的 `arcowo` 文本/二进制名全部改为 `arcnode`。

---

## Phase 0 — 命名统一 arcowo → arcnode（先做，低风险）

- [ ] `app/Cargo.toml`：crate `name = "arcowo"` → `"arcnode-agent"`（或 `arcnode`），同步二进制产物名
- [ ] 网关二进制 `arcowo-gateway` → `arcnode-gateway`（Makefile、CI、release.yml、.gitignore）
- [ ] `.github/workflows/ci.yml`、`release.yml`：产物名、artifact 路径、压缩包内文件名
- [ ] `Makefile`：build/clean 目标
- [ ] `README.md`（根 + android）、`docs/*`：命令示例、目录树、架构图里的 `arcowo`
- [ ] 全仓 `grep -ri arcowo` 复查归零
- [ ] 本地 `cargo build --workspace` + `make build` 验证，推分支过 CI

---

## Phase 1 — 自部署核心：agent + gateway 多设备数据处理

**A. 设备身份与注册**
- [ ] 设备 ID 生成/持久化稳定（重装不丢身份；冲突检测）
- [ ] `init-config` 生成稳定 UUID + 主机名，注册到网关 `devices`
- [ ] 网关侧设备元数据更新（last_seen、平台、配置）健壮化

**B. 摄取正确性（多设备并发）**
- [ ] 批量事件端点校验：字段必填、类型、时间戳范围、单批上限
- [ ] **幂等/去重**：客户端生成 `event_id`（或 device_id+ts+seq），重发不重复入库
- [ ] 事件顺序与时钟偏移处理（用服务端接收时间兜底 + 保留客户端时间）
- [ ] 部分失败处理：批中坏事件不拖垮整批，返回明确错误

**C. 可靠传输（agent 端）**
- [ ] **本地持久化重试队列**：离线/网关宕机时事件落盘，恢复后补传（agent 重启不丢）
- [ ] 退避重连 + 上限，避免风暴
- [ ] （已做的后台 flush + 失败重入队 在此基础上固化）

**D. 存储与查询**
- [ ] SQLite 调优复查：WAL、busy_timeout、`SetMaxOpenConns(1)` 对并发写的影响评估
- [ ] 索引复查（device_id+timestamp 已有；补 event_type/category 组合查询）
- [ ] **数据保留/清理**：可配置 retention，定时 prune + VACUUM，控制库体积
- [ ] 多设备合并查询正确性回归（已实现的 merged summary / timeline 加测试）

---

## Phase 2 — 安全性与传输加密

- [ ] **传输加密**：网关内置 TLS（证书配置）或文档化反代(Caddy/Nginx)方案；提供自签 + Let's Encrypt 指引
- [ ] **per-device API token**：废弃单一写死 token；每设备一把，**哈希存储**、可单独吊销、可命名
- [ ] token → device/权限 映射；agent 配置改用各自 token
- [ ] **前端/管理面登录鉴权**：当前 dashboard 仅靠 bearer，加管理员账号登录（自部署本地账号即可）
- [ ] 限流（按 token/IP）、请求体大小上限、超时
- [ ] 安全响应头、CORS 收紧（仅放行配置的前端来源）
- [ ] 配置脱敏：token 不进日志；支持环境变量注入；配置文件权限提示
- [ ] （可选）静态加密/文件权限说明，备份指引

---

## Phase 3 — 一键自部署可用

- [ ] **docker-compose**：网关（内嵌前端）+ 数据卷一条命令起；`.env` 配置 token/端口
- [ ] 单 Dockerfile 多阶段构建（前端 build → 嵌入 Go 二进制）
- [ ] `arcnode init` 体验：自动生成网关 token + 打印「设备接入命令」（可附二维码给移动端）
- [ ] **跨平台 agent 安装/服务化脚本**：Linux systemd、macOS launchd、Windows 服务（开机自启）
- [ ] mac/Linux 真机/CI 验证采集→上报→展示全链路
- [ ] Quickstart 文档：3 步内从零到看见数据；常见问题（端口、证书、权限）

---

## Phase 4 — 中心化前向兼容（仅设计/预留，不实装）

> 不做中心化，但现在埋好接口，未来加云端实现是「加文件」而非「改 handler」。

- [ ] 存储抽象：把 `gateway/storage/*.go` 收敛到 `EventStore` / `MetaStore` 接口，SQLite 作首个实现
- [ ] **`tenant_id` 列**：所有时序表加上，自部署恒为 `0`/local——schema 与云端一致，未来零迁移
- [ ] 所有查询强制带 `tenant_id`（本地固定值），杜绝未来串户隐患
- [ ] `event_id` 幂等贯穿（Phase 1 已做，确保云端可直接复用）
- [ ] **API 版本化**（`/api/v1/...` 已是，固化契约 + 稳定事件 schema）
- [ ] `ARCNODE_MODE=single|cloud` 脚手架（当前仅 single 生效）
- [ ] per-device token 模型与云端一致（Phase 2 已对齐）

---

## Phase 5 — 验证与发布

- [ ] 多设备 E2E：≥2 模拟设备并发上报 → 合并视图/各页正确
- [ ] 断线/重启/坏数据/高并发 压力与故障注入测试
- [ ] 安全自查：未授权访问被拒、TLS 生效、token 吊销即时失效、限流生效
- [ ] README/docs/截图 全部改名 arcnode、更新一键部署说明
- [ ] release 产物改名（arcnode-gateway / arcnode-agent），校验包内文件
- [ ] CI 全绿，打 tag 出 release

---

## 落地建议

- **顺序**：Phase 0（改名，半天）→ Phase 1（数据处理稳固，核心）→ Phase 2（安全/加密）→ Phase 3（一键部署）→ Phase 4（预留接口，多为重构）→ Phase 5（验证发布）。
- Phase 1/2/4 可拆成多个小 PR，逐个过 CI，降低风险。
- Phase 4 的「tenant_id + 存储接口 + per-device token」是连接自部署与中心化的三根钢筋，越早埋越省后续迁移成本。
