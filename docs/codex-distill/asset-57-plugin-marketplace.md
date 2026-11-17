# asset-57 · 插件市场与远程分发（B7 · 施工级）

> 证据：`codex-rs/core-plugins/src/{lib.rs, manager.rs, manifest.rs, agent_plugin_manifest.rs, loader.rs,
> marketplace.rs, marketplace_add.rs, marketplace_policy.rs, marketplace_upgrade.rs, remote.rs,
> remote_bundle.rs, remote/remote_installed_plugin_sync.rs, remote/share.rs, startup_sync.rs,
> command_migration.rs, store.rs, installed_marketplaces.rs, discoverable.rs}`（`rust-v0.165.0`）。
> `asset-17-plugins.md` 已覆盖"插件是资源包（skills/hooks/mcp/apps + interface 元数据）"，
> 本篇只写**市场、来源策略、远程分发、清单加载与迁移**。
> 判定：**只在 gaea 决定做插件生态时才落地**（拍板门 1）；不做则本篇作为"为什么不做的依据"。

---

## 1. 一句话结论

插件市场不是一个"下载器"，而是**四道闸门叠加**：
① **清单闸**——清单只在插件根目录下解析，路径必须是 `./...` 形式且**先校验语法再解析**；
② **来源闸**——允许的来源是显式枚举（`GitUrl{url,ref}` / `GitHostPattern(regex)` / `Local(path)`），
由 managed requirements 决定，**不是用户随便填**；
③ **策略闸**——`MarketplacePolicy` 必须在构造时就被验证，注释写明*"默认值 + 后设 setter 会 fail open"*；
④ **同步闸**——策展库同步只做一次（`AtomicBool`），远程已安装插件同步有 **120s 等待上限**。
外加两条容量纪律：已加载插件缓存 **8** 条、featured 插件 id 缓存 **3 小时**。

---

## 2. 机制地图

| 关注点 | 文件 | 关键行 |
|---|---|---|
| 市场名单 | `lib.rs` | 47–49 `openai-curated` / `openai-api-curated` / `openai-bundled`；另有 `openai-primary-runtime`、`openai-bundled-alpha`（`artifact_operation.rs:4`、`marketplace_policy.rs:2`） |
| 管理器与缓存 | `manager.rs` | 151 策展同步只做一次的 `AtomicBool`；152–153 `FEATURED_PLUGIN_IDS_CACHE_TTL = 3h`；154 `REMOTE_INSTALLED_PLUGIN_SYNC_WAIT_TIMEOUT = 120s`；155 `LOADED_PLUGINS_CACHE_CAPACITY = 8`；173 注释"构造即需已验证策略" |
| 清单 | `manifest.rs` | 16–17 `MAX_DEFAULT_PROMPT_COUNT=3` / `MAX_DEFAULT_PROMPT_LEN=128`；37–41 `PluginManifestFormat::{Legacy, AgentPlugin}`；59–60 路径保留原始串以**先校验 `./...` 语法** |
| 清单字段 | `agent_plugin_manifest.rs` | 17 命名空间 `com.openai`；18 字段白名单 |
| 加载器 | `loader.rs` | 72–77 `skills` / `hooks/hooks.json` / `.mcp.json` / `.app.json` / `config.toml`；77 缓存版本 sha 前缀 8 位 |
| 来源策略 | `marketplace_policy.rs` | 36–43 `AllowedMarketplaceSource::{GitUrl, GitHostPattern, Local}`；19–24 与 requirements/config 类型对接；27 路径**规范化后比较** |
| 安装目录 | `installed_marketplaces.rs` | 12 `.tmp/marketplaces` |
| 安装/升级 | `marketplace_add.rs`(440)、`marketplace_upgrade.rs`(489) | 来源解析与升级路径 |
| 远程 | `remote.rs`(2261)、`remote_bundle.rs`(1024)、`remote/share.rs`(513) | 远程插件、bundle、分享/检出 |
| 策展同步 | `startup_sync.rs`(1031) | 策展仓库 URL / API 市场路径 / 仓库路径 |
| 命令迁移 | `command_migration.rs` | 14 `COMMAND_SKILL_PREFIX="source-command"`；15 `MAX_SKILL_NAME_LEN=64` |
| 探测并发 | `executor_provider.rs` | 24 `MAX_CONCURRENT_MANIFEST_PROBES=256` |
| 产物 | `artifact_operation.rs` | 5 `MAX_EXPECTED_OUTPUT_COUNT=100` |

---

## 3. 清单闸：先语法、后解析

```rust
// manifest.rs:59–60（注释）
// Keep manifest paths as raw strings so we can validate the required `./...` syntax before
// resolving them under the plugin root.
```
- 清单里的路径**保持为字符串**，先做 `./...` 语法校验，**再**在插件根下解析。
- 好处：`../` 逃逸与绝对路径在**字符串阶段**就被拒，不需要依赖"解析后再检查父目录"。
- 清单有**两种格式**：`Legacy` 与 `AgentPlugin`（命名空间 `com.openai`），由 `parse_agent_plugin_manifest_uri` 判别——
  兼容老插件但新插件走新形状。
- 默认提示词有上限：最多 3 条、每条 128 字符（`MAX_DEFAULT_PROMPT_COUNT/LEN`）——防止清单变成提示词注入载体。

---

## 4. 来源闸：白名单是"类型"而不是"字符串"

```rust
enum AllowedMarketplaceSource {
    GitUrl { url: String, ref_name: Option<String> },
    GitHostPattern(Regex),
    Local(AbsolutePathBuf),
}
```
- 三种来源形态互斥且显式：**具体 git 地址**（可钉 ref）、**git 主机正则**（如"公司 GitLab"）、**本地绝对路径**。
- 判定链：`ConfigRequirements`（托管层）→ `MarketplaceAllowedSourceToml` → `AllowedMarketplaceSource`；
  即"谁能装市场"由**策略层**决定，不是用户在 UI 上点一下。
- 本地路径比较用 `paths_match_after_normalization`：**规范化后再比**，否则 `\` 与 `/`、`..`、大小写差异都能绕过。

---

## 5. 策略闸：构造即验证（fail-closed）

```rust
// manager.rs:173（注释）
// Require validated policy at construction; a default followed by a setter could fail open.
```
- `PluginsConfigInput::new(...)` 要求**传入已验证的策略**；不提供"先 new 默认再 set"的路径。
- 这是"安全开关不允许两步初始化"的通用范式：**任何"默认宽、后来收紧"的接口都是 fail-open 隐患**。

---

## 6. 同步闸与容量纪律

| 纪律 | 值 | 说明 |
|---|---|---|
| 策展库同步 | 只做一次（`AtomicBool`，进程级） | 避免每次会话都拉一次仓库 |
| 远程已安装插件同步等待 | 120s | 超时即"用现有状态继续"，不阻塞启动 |
| featured 插件 id 缓存 | 3h | 目录类数据用长 TTL，避免频繁请求 |
| 已加载插件缓存 | 8 条 | 上界明确，防"插件多到吃内存" |
| 清单探测并发 | 256 | 扫描插件目录时的并发上限 |
| 单插件预期产物数 | 100 | 产物类插件（artifact）的输出上界 |
| 迁移后技能名 | ≤64 字符 | 命令→技能迁移时的长度约束 |

---

## 7. 失败路径

1. **来源不在白名单**：策略判定阶段即拒（`MarketplacePolicy`），不发起网络请求。
2. **清单路径非法**（`../`、绝对路径、非 `./` 前缀）：字符串校验阶段拒绝，**不解析**。
3. **清单版本/格式不识别**：走 `Legacy` 兼容分支；两者都不匹配则报"清单格式不支持"。
4. **策展同步失败**：只 warn，市场列表退化为"已安装 + 内置"；不阻塞会话。
5. **远程同步超时（120s）**：以现有状态继续，后台重试；不在启动路径上硬等。
6. **升级失败**：保留旧版本目录（升级是"新目录 + 切换"，不是原地覆盖——由 `marketplace_upgrade` 的路径结构体现）。

---

## 8. gaea 对照

| 维度 | codex | gaea | 结论 |
|---|---|---|---|
| 插件形态 | 资源包（skills/hooks/mcp/apps），asset-17 已覆盖 | `internal/gaea/plugin`（未读细节）、`internal/gaea/skill` | 形态接近，**无市场** |
| 来源策略 | 三态白名单 + 托管层决策 | 未读到 | 若做市场必须先有来源策略，**否则等于开放任意代码执行** |
| 清单校验 | 先语法后解析 + 提示词条数/长度上限 | 未读到 | 可借：清单里的一切字符串都要有上限与形态校验 |
| 构造即验证 | 明确禁止"默认 + setter"的 fail-open | 未读到 | **强烈建议作为通用纪律**引入 |
| 缓存 TTL | 目录 3h / 远程同步 120s / 已加载 8 条 | 未读到统一 TTL 口径 | 可借 TTL 分层口径 |

---

## 9. 施工卡（仅在做插件生态时；3 刀）

### 刀 1 · 来源策略与清单校验（约 1 天）

- `internal/gaea/plugin/source.go`：`AllowedSource{GitURL{URL,Ref}, GitHostPattern, LocalPath}`，
  由配置层（托管优先）给定；本地路径**规范化后比较**。
- 清单校验顺序固定：① 形态（`./` 前缀、禁绝对/父级）② 字段上限（提示词 ≤3 条、每条 ≤128 字符）
  ③ 体积上限 ④ 才解析。
- 禁区：不做"用户任意填 URL"的自由市场；策略缺失时**默认拒绝**（fail-closed）。

### 刀 2 · 同步与缓存（约半天）

- 策展同步进程内只做一次；远程同步 120s 上限；目录缓存 3h；已加载插件缓存 8 条。
- 禁用/降级：网络失败只 warn，市场列表退化为"已安装 + 内置"。

### 刀 3 · 升级与回滚（约半天）

- 升级 = 新目录 + 原子切换 + 保留旧目录；失败回滚到旧目录（不原地覆盖）。
- 迁移（命令→技能）带长度上限（≤64）与前缀（`source-command` 一类）标记，便于识别来源。

---

## 10. 验收断言

| # | 断言 |
|---|---|
| P1 | 清单路径 `../evil` 与绝对路径在**解析前**被拒（无 IO 发生） |
| P2 | 来源不在白名单：不发起任何网络请求，直接拒绝并给出可读原因 |
| P3 | 本地路径比较：`a/./b` 与 `a/b` 视为同一来源；`\` 与 `/` 混用不绕过 |
| P4 | 策展同步并发调用 N 次，实际只发生一次网络请求 |
| P5 | 远程同步卡住 120s 后继续启动，市场列表为"已安装 + 内置" |
| P6 | 升级失败后旧版本仍可用（目录未被破坏） |
| P7 | 清单提示词 4 条或单条 200 字符 → 拒绝 |

---

## 11. 不搬 / 风险 / 未读

- **不搬**：`remote/share.rs`（分享/检出）、`remote_bundle.rs` 的 bundle 打包格式、`startup_sync.rs` 的
  OpenAI 策展仓库地址与 API 路径、`artifact_operation.rs` 产物技能清单。
- **不搬**：多市场命名空间（`openai-curated` / `-api-curated` / `-bundled` / `-primary-runtime`）——
  那是上游产品分层，gaea 若做只需一个来源白名单。
- **风险**：① 做市场 = 引入"下载并执行第三方资源"的攻击面（技能即提示词、hooks 即命令），
  必须先有来源策略 + 清单校验 + 用户显式确认三层，否则不做更安全；
  ② 升级的"新目录 + 切换"要求磁盘上同时存在两版，需算清空间与清理策略（上游用 `release_pruning` 一类机制）。
- **未读（诚实边界）**：`manager.rs`(3634) 全文、`remote.rs`(2261)、`loader.rs`(1825)、`marketplace.rs`(1070)、
  `remote_bundle.rs`(1024)、`startup_sync.rs`(1031)、`store.rs`(694)、`script_attribution.rs`(659)、
  `discoverable.rs`（可发现性白名单）全文。
