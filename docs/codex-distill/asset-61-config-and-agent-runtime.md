# asset-61 · 配置层与多智能体运行面（B 批收口 · 施工级）

> 证据：`codex-rs/core/src/config/{mod.rs(4818), edit.rs(911), edit/document_helpers.rs(338), permissions.rs(830),
> network_proxy_spec.rs(514), managed_features.rs(351), runtime_refresh.rs(202), requirements.rs(169),
> windows_sandbox_config.rs(180), network_config.rs(163)}`、
> `core/src/agent/{control/spawn.rs(1535), control.rs(734), control/user_authorization.rs(420), role.rs(401),
> registry.rs(379), child_config.rs(344), control/residency.rs(318), control/api.rs(309), control/runtime.rs(309),
> root_handoff.rs, api.rs}`（`rust-v0.165.0`）。
> 本篇补齐 `core` 最后两个未覆盖子系统；`core/src/{tools→asset-50, session→asset-49, context/guardian→asset-02/07/08}`。

---

## 1. 一句话结论

配置层的价值在**"默认值即口径"**：git 忽略大目录 `200` 项 / 大文件 `10 MiB`、异步压缩阈值 `65%`、
协作工具命名空间 `collaboration`、code-mode 让出时间 `30s`、三处提示词模板各 **≤2000 B**——
每个数字都写在 `config/mod.rs` 常量里，可被单测钉住。
多智能体运行面的价值在**"资源上限 + 身份"**：子代理关停失败只保留最近 `64` 次、根上下文窗口 `3` 条、
授权消息上限 `16` 条、角色名有 `default` 兜底、agent 名称来自静态词表（`include_str!`）。

---

## 2. 机制地图

| 关注点 | 文件 | 关键行 |
|---|---|---|
| 默认值口径 | `config/mod.rs` | 226 `DEFAULT_IGNORE_LARGE_UNTRACKED_DIRS=200`；227 `…UNTRACKED_FILES=10 MiB`；228 `DEFAULT_ASYNC_COMPACTION_THRESHOLD_PERCENT=65`；263 `DEFAULT_MULTI_AGENT_V2_TOOL_NAMESPACE="collaboration"`；1211 `DEFAULT_CODE_MODE_EXEC_YIELD_TIME_MS=30_000` |
| 提示词模板上限 | `config/mod.rs` | 1240–1242 三处各 `2000` 字节（reminder / guidance / auto-compact fallback） |
| 配置文件 | `config/mod.rs` | 271 `CONFIG_TOML_FILE="config.toml"`；272 `CONFIG_PROFILE_V2_SUFFIX=".config.toml"`；269 `LOCAL_DEV_BUILD_VERSION="0.0.0"` |
| 编辑（保留注释） | `config/edit.rs` + `edit/document_helpers.rs` | 26 `NOTICE_TABLE_KEY="notice"`；`document_helpers` 用 `toml_edit` 做**保留注释的改写** |
| 权限模型 | `config/permissions.rs`(830) | 权限 profile 的解析与合并 |
| 出网规格 | `config/network_proxy_spec.rs`(514)、`network_config.rs`(163) | 代理规格 → 运行时配置 |
| 托管特性 | `config/managed_features.rs`(351)、`requirements.rs`(169) | 管理员层可**钉住**特性开关（见 `asset-53`/`asset-57` 的 requirements） |
| 运行期刷新 | `config/runtime_refresh.rs`(202) | 配置变更 → 运行期生效路径 |
| 子代理派生 | `agent/control/spawn.rs`(1535) | 子代理拉起（角色、配置、上下文继承） |
| 角色 | `agent/role.rs` | 33 `DEFAULT_ROLE_NAME="default"`；34 `AGENT_TYPE_UNAVAILABLE_ERROR`（不可用要明说，不静默降级） |
| 注册表 | `agent/registry.rs`(379) | agent 类型注册与查询 |
| 名称词表 | `agent/control/spawn.rs:47` | `AGENT_NAMES = include_str!("assets/agent/agent_names.txt")`（**静态词表，不用随机生成**） |
| 关停失败保留 | `agent/control/runtime.rs` | 34 `MAX_RETAINED_SHUTDOWN_FAILURES=64` |
| 根上下文窗口 | `agent/root_handoff.rs` | 15 `ROOT_CONTEXT_WINDOW=3` |
| 授权消息 | `agent/control/user_authorization.rs` | 44 `MAX_ROOT_MESSAGES=16` |
| 子代理配置 | `agent/child_config.rs`(344)、`control/residency.rs`(318) | 子代理的配置派生与驻留 |

---

## 3. 配置层：默认值即口径

```rust
const DEFAULT_IGNORE_LARGE_UNTRACKED_DIRS: i64 = 200;
const DEFAULT_IGNORE_LARGE_UNTRACKED_FILES: i64 = 10 * 1024 * 1024;
const DEFAULT_ASYNC_COMPACTION_THRESHOLD_PERCENT: u8 = 65;
const DEFAULT_MULTI_AGENT_V2_TOOL_NAMESPACE: &str = "collaboration";
const DEFAULT_CODE_MODE_EXEC_YIELD_TIME_MS: u64 = 30_000;
```

四条观察：

1. **默认值写成具名常量而不是散落字面量**——于是"我们为什么忽略 200 个以上文件的未跟踪目录"这类问题
   有唯一答案，也能被单测覆盖。
2. **阈值类默认值都有"百分比/字节"单位后缀**（`_PERCENT`、`_BYTES`、`_MS`），消灭单位歧义。
3. **提示词模板有硬字节上限**（三处各 2000）——防止配置里塞进超长提示词把上下文吃掉。
4. **配置编辑走 `toml_edit` 文档模型**（`edit/document_helpers.rs`）：改写时**保留用户注释与格式**，
   而不是"读→结构体→整体重写"。这是"配置文件也是用户资产"的态度。

`managed_features.rs` + `requirements.rs` 是**托管层**：管理员可以钉住某个特性（开或关），
用户的本地覆盖不能越过它——与 `asset-53`（网络策略）、`asset-57`（市场来源白名单）是同一套模式。

---

## 4. 多智能体运行面

### 4.1 身份与可用性

- `DEFAULT_ROLE_NAME = "default"`：角色缺失时用兜底名，**不是拒绝**。
- `AGENT_TYPE_UNAVAILABLE_ERROR = "agent type is currently not available"`：类型不可用时**明确报错**，
  不静默降级成别的 agent（降级会改变行为语义，报错只损失一次调用）。
- agent 名称来自**静态词表**（`include_str!("assets/agent/agent_names.txt")`）：
  可复现、可本地化、不依赖随机数——便于日志对齐与用户记忆。

### 4.2 资源上限（三条）

| 上限 | 值 | 位置 | 语义 |
|---|---|---|---|
| 关停失败保留 | 64 | `agent/control/runtime.rs:34` | 只留最近 64 次关停失败记录（防日志吃内存） |
| 根上下文窗口 | 3 | `agent/root_handoff.rs:15` | 交给子代理的根上下文只带 3 条（防上下文爆炸） |
| 授权消息上限 | 16 | `agent/control/user_authorization.rs:44` | 用户授权相关的消息最多 16 条 |

三者都是"子代理不能把父会话拖垮"的具体体现：**上下文/日志/授权**三个方向各设一道闸。

### 4.3 派生与驻留

`control/spawn.rs`(1535) 是子代理派生的真身：角色 → 配置派生（`child_config.rs`）→ 上下文继承
（`root_handoff.rs` 的窗口）→ 驻留（`residency.rs`）→ 注册（`registry.rs`）。
`control.rs`(734) 是控制面（暂停/恢复/取消/查询），`control/api.rs` 是给扩展用的 trait。

---

## 5. 失败路径

1. **配置无法解析**：报错并保留原文件（`toml_edit` 改写失败不动盘）。
2. **托管特性与本地冲突**：托管层赢，本地覆盖被忽略并给 warning（不静默）。
3. **agent 类型不可用**：明确错误串（`AGENT_TYPE_UNAVAILABLE_ERROR`），不换类型重试。
4. **子代理关停失败**：记录但只留 64 条；父会话继续（不因一个子代理卡住）。
5. **配置运行期刷新失败**：保留旧配置（`runtime_refresh` 的语义是"要么全生效，要么不生效"）。

---

## 6. gaea 对照（只读实测）

| 维度 | codex | gaea | 结论 |
|---|---|---|---|
| 默认值 | 具名常量 + 单位后缀 + 单测 | `internal/gaea/config`（未逐文件读）；已知散落常量（如 `sendQueueLimit=8`、`wakeMaxConsecutive=3`） | 可借"具名 + 单位 + 单测"三件套 |
| 配置编辑 | `toml_edit` 保留注释；原子写（见 asset-47 §4 的 `write_atomically`） | 未读到（`config` 包写法未读） | **可借**：配置文件是用户资产，改写要保注释 |
| 托管层 | requirements 钉住特性/策略 | 未读到托管层概念 | 单用户本地可不要；若做企业版必须补 |
| 子代理上限 | 64 失败 / 3 上下文 / 16 授权 | `wakeMaxConsecutive=3`（自激励预算）已有 | 方向一致；缺"关停失败保留上限"与"授权消息上限" |
| agent 名称 | 静态词表 | 未读到 | 可借（可复现、可本地化） |

---

## 7. 施工卡（2 刀小改）

### 刀 1 · 配置常量具名化 + 单位后缀（约 2 小时）

- 把 gaea 散落的魔法数（`sendQueueLimit=8`、`wakeMaxConsecutive=3` 一类）迁到
  `internal/gaea/config/defaults.go`，命名带单位（`_BYTES`/`_MS`/`_PERCENT`），并为每个常量加一条单测断言取值。

### 刀 2 · 配置改写保注释（约 2–3 小时，视现状）

- 若 gaea 现在是"读结构体 → 整体重写 TOML"，改为文档模型改写（保留注释与键序），失败不动盘。

---

## 8. 验收断言

| # | 断言 |
|---|---|
| C1 | 每个默认值常量有单测，改动常量会导致测试失败（防误改） |
| C2 | 配置改写后，原文件注释与键序保持不变；改写失败时文件字节不变 |
| C3 | 托管层与本地冲突：以托管为准，且产生一条 warning |
| C4 | agent 类型不可用：返回固定错误串，不自动换类型 |
| C5 | 子代理关停失败超过 64 次：只保留最近 64 条，内存不增长 |
| C6 | 子代理根上下文只带 3 条（超出丢弃并计数） |

---

## 9. 不搬 / 未读

- **不搬**：`windows_sandbox_config.rs`、`network_proxy_spec.rs` 的平台细节（沙箱与代理实现由 OS/宿主承担）。
- **不搬**：`managed_features.rs` 的托管层（单用户本地无对应），但**保留其"托管优先"的设计原则**备将来用。
- **未读（诚实边界）**：`config/mod.rs`(4818) 只读常量与结构；`config/permissions.rs`、`edit/document_helpers.rs`
  全文；`agent/control/spawn.rs`(1535) 只读结构；`agent/registry.rs`、`child_config.rs`、`residency.rs` 全文。
