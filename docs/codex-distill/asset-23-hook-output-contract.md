# 资产 23 · Hooks 输出契约与匹配器：结构化决策替代退出码

> 源码：`hooks/src/{engine/output_parser.rs 21 KB, engine/matcher.rs 1.5 KB, config_rules.rs 9 KB, schema.rs 48 KB}`。
> 对照：gaea `internal/gaea/hook/{hook.go, runner.go}`——目前靠**退出码**判定阻断（非零/exit 2）、
> 靠 stdout 注入上下文（`PreCompact` 的 stdout 作为额外摘要指引）。

---

## 一、钩子输出是 JSON，不是退出码

### 通用字段（每个事件都认）

```rust
pub struct UniversalOutput {
    pub continue_processing: bool,     // JSON: continue
    pub stop_reason: Option<String>,   // stopReason
    pub suppress_output: bool,         // suppressOutput
    pub system_message: Option<String>,// systemMessage
}
```

### 每个事件额外字段

| 事件 | 额外字段 | 语义 |
|---|---|---|
| `SessionStart` / `SubagentStart` | `additionalContext` | 注入到会话起始的上下文 |
| `PreToolUse` | `permissionDecision`（allow/deny/ask）、`permissionDecisionReason`、`additionalContext`、`updatedInput` | **可以改写工具入参**、可以要求询问 |
| `PermissionRequest` | `decision` = `allow` \| `deny{message}` | 对审批请求直接裁决 |
| `PostToolUse` | `shouldBlock`、`reason`、`additionalContext` | 事后阻断 + 补上下文 |

**两个关键能力是退出码模型做不到的**：

1. **`ask`**：钩子可以要求「这条要问用户」，而不是只能放行/拦截；
2. **`updatedInput`**：钩子可以**改写工具的入参**再执行（例如把危险路径重写成安全路径），
   而不是只能允许或拒绝。

## 二、两套决策词汇并存（兼容历史）

```rust
enum PreToolUsePermissionDecisionWire { Allow, Deny, Ask }   // 新（hook-specific）
enum PreToolUseDecisionWire          { Approve, Block }      // 旧（legacy）
```

解析优先级（`parse_pre_tool_use`）：

```
若 hook_specific_output 里出现 permissionDecision 或 permissionDecisionReason
    → 用新词汇（allow / deny / ask）
否则
    → 回落到 legacy（approve / block），并记一条 unsupported_pre_tool_use_legacy_decision
```

另有 `invalid_reason` 字段：输出**结构合法但取值不合法**时，把原因带出来而不是静默失败。

## 三、匹配器：一个字段，三种语义

```rust
enum HookMatcher { All(String), Exact(String), Regex(regex::Regex) }

new(pattern):
  空 或 "*"                                → All
  只含 [A-Za-z0-9_|]                       → Exact（"a|b|c" 按 | 拆分多值匹配）
  其它                                      → 按正则编译
matches(input): All→true / Exact→按 | 拆分后逐个相等 / Regex→正则匹配
```

**这个设计很省事**：配置里只写一个字符串字段，按内容特征自动决定是通配、枚举还是正则；
`PartialEq` 比较的是原始字符串（配置身份），不是编译后的对象。

**注意**：`Exact` 分支允许 `|` 分隔多值（`"read_file|write_file"`），
所以「多个精确值」不需要写正则。

## 四、信任规则：能发现 ≠ 能启用

```rust
// config_rules.rs 注释原文意思
// 只读 User 与 SessionFlags 两层；Project / Managed / Plugin 层
// 可以发现钩子，但**不能写用户的 hook 状态**。
```

这条很重要：插件或项目配置里带了一个钩子，它可以被**发现**（出现在列表里），
但**是否启用**由用户层决定。否则「下载一个项目 = 自动获得执行权限」。

## 五、钩子输入（stdin JSON）

```rust
#[serde(deny_unknown_fields)]
pub(crate) struct PreToolUseCommandInput {
    pub session_id: String,
    pub turn_id: String,        // codex 扩展：把当前回合 id 暴露给钩子
    ...
}
```

输入是**严格模式**（`deny_unknown_fields`）：多一个未知字段就拒绝，避免拼错字段名后静默无效。

---

## 六、gaea 施工方案（1–1.5 天，改 `hook` 包）

1. **定义输出结构**（`hook/output.go`）：
   ```go
   type Output struct {
       Continue      *bool   `json:"continue,omitempty"`
       StopReason    string  `json:"stopReason,omitempty"`
       SuppressOut   bool    `json:"suppressOutput,omitempty"`
       SystemMessage string  `json:"systemMessage,omitempty"`
       // PreToolUse
       PermissionDecision       string `json:"permissionDecision,omitempty"` // allow|deny|ask
       PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
       UpdatedInput             json.RawMessage `json:"updatedInput,omitempty"`
       // PostToolUse
       ShouldBlock bool   `json:"shouldBlock,omitempty"`
       Reason      string `json:"reason,omitempty"`
       // 通用
       AdditionalContext string `json:"additionalContext,omitempty"`
   }
   ```
2. **双模式**：`stdout` 能解析成 JSON 就用结构化决策；解析失败回落到现有退出码语义
   （保持向后兼容，不破坏已有钩子脚本）；
3. **`ask` 与 `updatedInput`** 接到审批闸门与工具派发：`ask` → 走审批；`updatedInput` → 替换参数后重校验；
4. **匹配器**：照 §三 的三级编译实现 `Match(pattern string)`；
5. **状态来源**：钩子「可发现」与「已启用」分开存，只有用户配置层（与项目级显式确认）能改启用状态。

## 七、验收（可断言）

1. 钩子输出 `{"permissionDecision":"ask","permissionDecisionReason":"..."}` → 触发审批且理由可见；
2. 输出 `{"updatedInput":{...}}` → 工具收到改写后的参数，且**重写后仍过一遍策略校验**（不得绕过）；
3. 输出合法 JSON 但 `permissionDecision` 取值非法 → 记 `invalid_reason` 并回落到退出码语义（不静默放行）；
4. 匹配器：`"*"` 全匹配、`"read_file|write_file"` 只匹配这两个、`"^read_.*"` 走正则——三条单测；
5. 插件/项目层声明的钩子出现在列表但默认不启用；只有用户层能改启用状态。

## 八、不建议照搬

- **legacy `approve/block` 兼容层**：gaea 没有历史包袱，直接上 `allow/deny/ask` 一套；
- **`schema.rs` 48 KB 的 JSON Schema 生成**：那是为了给编辑器做补全与校验，gaea 用结构体 + 一份文档即可；
- **Claude Code 兼容的输入字段名**（`hook_event_name` 等）：gaea 用自己的命名，别背兼容。

## 九、一句话

把钩子从「退出码 + stdout」升级成「JSON 决策」，换来两个能力：**`ask`（要求询问）**与
**`updatedInput`（改写入参后再执行）**；而「能发现 ≠ 能启用」这条信任规则，
是防止「下载项目即获执行权」的关键一句。
