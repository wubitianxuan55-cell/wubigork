# 资产 17 · 插件生态：插件不是代码，是「资源包」

> 源码：`plugin/src/{manifest.rs,load_outcome.rs,bundled_hooks.rs,provider.rs}`（5 文件）、
> `core-plugins/src/`（`manager.rs` 153 KB / `loader.rs` 72 KB / `remote.rs` 89 KB / `marketplace.rs` 39 KB /
> `manifest.rs` 35 KB / `store.rs` 29 KB）、`core-plugin-common/src/{installed.rs,plugin_id.rs}`。
> 对照：gaea `internal/gaea/plugin/`（`plugin.go` 24 KB 做 MCP 客户端 + `transport_stdio/http` + `ssrf.go`）。

---

## 一、核心认知：插件是资源包，不是可执行代码

```rust
pub struct PluginManifest<Resource> {
    pub name: String,
    pub version: Option<String>,
    pub description: Option<String>,
    pub keywords: Vec<String>,
    pub paths: PluginManifestPaths<Resource>,     // ★ 资源清单一
    pub interface: Option<PluginManifestInterface<Resource>>,  // 面向模型/UI 的元数据
}

pub struct PluginManifestPaths<Resource> {
    pub skills: Vec<Resource>,                    // 技能（目录或文件）
    pub onboarding_skill: Option<Resource>,       // 首次使用引导技能
    pub mcp_servers: Option<PluginManifestMcpServers<Resource>>,  // Path | Object(内联)
    pub apps: Option<Resource>,                   // 连接器/应用声明
    pub hooks: Option<PluginManifestHooks<Resource>>,  // Paths(Vec) | Inline(Vec<HooksFile>)
}
```

**插件里没有一条可执行代码**——它只是把宿主**已经会加载的东西**打成一个包：
技能、MCP 服务声明、hooks、应用声明，外加一份给模型和 UI 看的元数据
（`display_name` / `capabilities` / `default_prompt` / `brand_color` / `logo` / `screenshots`…）。

这个设计对 gaea 极其重要：gaea 的 `plugin` 包现在只做 **MCP 客户端**，
而 skills 与 hooks 各自独立加载。若把三者收拢成「插件 = 资源包」，
用户装一个包就能同时得到「技能 + 工具 + 钩子」，而**不需要信任任何新代码**。

## 二、运行时模型（`LoadedPlugin` 的字段就是全部语义）

```rust
pub struct LoadedPlugin<M> {
    pub config_name: String,              // 配置里的名字
    pub remote_plugin_id: Option<String>, // 来自远端的 id（本地包为 None）
    pub manifest_name: Option<String>,
    pub plugin_namespace: Option<String>, // 命名空间（工具名前缀来源）
    pub manifest_description: Option<String>,
    pub root: AbsolutePathBuf,
    pub enabled: bool,                    // ★ 可单独禁用
    pub skill_roots: Vec<AbsolutePathBuf>,
    pub skill_discovery_mode: SkillDiscoveryMode,
    pub disabled_skill_paths: HashSet<PathUri>,  // ★ 可按路径禁单个技能
    pub has_enabled_skills: bool,
    pub mcp_servers: HashMap<String, M>,  // ★ 已合并的 MCP 定义
    pub apps: Vec<AppDeclaration>,
    pub hook_sources: Vec<PluginHookSource>,
}
```

三个可抄的设计：

1. **`enabled` 与 `disabled_skill_paths` 并存**：整包禁用，和「包内某个技能单独禁用」是两件事。
2. **`has_enabled_skills` 是派生态而非配置项**——UI 用它决定「这个插件有没有可用能力」，
   避免渲染一个「装了但什么都没开」的空卡片。
3. **`mcp_servers` 是 `HashMap`（已合并）**：插件声明的 MCP 与用户自己配的同名时，
   由加载期决定覆盖关系，运行期只看到一张表。

另有 `MAX_CAPABILITY_SUMMARY_DESCRIPTION_LEN = 1024`：给模型看的「插件能力摘要」有硬上限。

## 三、安装态与版本（小但值得抄的校验）

```rust
pub const DEFAULT_PLUGIN_VERSION: &str = "local";
pub const PLUGINS_CACHE_DIR: &str = "plugins/cache";
pub const REMOTE_PLUGIN_INSTALL_METADATA_FILE: &str = ".codex-remote-plugin-install.json";
pub const REMOTE_PLUGIN_INSTALL_METADATA_SCHEMA_VERSION: u8 = 1;

// 版本段校验：路径穿越防御
validate_plugin_version_segment(v):
  - 非空
  - 不是 "." 或 ".."
  - 只允许 ASCII 字母数字与 - _ . +
```

**这段校验是安全要点**：插件版本号会被拼进缓存目录路径，
如果允许 `../` 就能写到缓存目录之外。`DEFAULT_PLUGIN_VERSION = "local"` 表示本地包没有远程版本号。

`schema_version` 单独存一份（`.json` 元数据文件），便于将来迁移。

## 四、明确不做（gaea 无对应）

| 不做 | 理由 |
|---|---|
| `marketplace.rs` / `remote.rs` / `remote_bundle.rs`（合计 ~156 KB） | 云端插件市场与远端分发，单机应用没有对手方 |
| `marketplace_policy.rs`（22 KB） | 企业侧「允许哪些来源」的合规策略 |
| `startup_sync.rs`（40 KB） | 启动时与云端同步已装插件 |
| `script_attribution.rs`（26 KB） | 插件脚本的归属审计（云端计费/合规用） |

## 五、gaea 施工方案（1.5–2 天，改造现有 plugin 包）

1. **新增 `plugin/manifest.go`**：
   ```go
   type Manifest struct {
       Name, Version, Description string
       Keywords []string
       Paths    ManifestPaths      // Skills / MCPServers / Hooks / Apps
       Interface *Interface        // 展示元数据（可空）
   }
   type ManifestPaths struct {
       Skills     []string
       MCPServers *MCPServers      // Path 或内联对象
       Hooks      *Hooks           // Paths 或内联
       Apps       []string
   }
   ```
2. **加载器**：解析 manifest → 逐类资源交给已有加载器（`skill` 包、`plugin` 的 MCP、`hook` 包），
   产出 `LoadedPlugin{Enabled, SkillRoots, DisabledSkillPaths, MCPServers, HookSources}`；
3. **失败逐项上报**：某类资源加载失败不拖垮整包（返回 `LoadOutcome`，含每类的 ok/err）；
4. **版本段校验**：照抄 `validate_plugin_version_segment` 四条规则（空 / `.` / `..` / 字符集）；
5. **禁用两级**：包级 `enabled` + 技能级 `disabled_skill_paths`。

## 六、验收（可断言）

1. 一个只含 skills 的插件加载后，技能出现在技能列表且来源标为该插件；
2. 一个含 MCP 声明的插件与用户手工配置同名 server → 启动不冲突，且 UI 能看出谁覆盖谁；
3. 插件里某个技能目录丢失 → 其余资源正常加载，`LoadOutcome` 里该项为 err（整包不失败）；
4. `version = "../../etc"` → 拒绝加载并给出「无效版本号」错误（路径穿越防御）；
5. 包级 `enabled=false` → 其技能/MCP/hooks **都不出现**在上下文与工具面里；
6. 技能级禁用 → 只有该技能消失，同包其它技能仍在。

## 七、一句话

codex 的插件模型值得借的只有一句话：**插件是「资源包的清单」，不是「要执行的程序」**。
把 gaea 现在分散的 skills / MCP / hooks 收拢成这一层，用户装包不增加代码信任面，
而 gaea 的 `plugin` 包也能从「MCP 客户端」升级成「能力包管理器」。
