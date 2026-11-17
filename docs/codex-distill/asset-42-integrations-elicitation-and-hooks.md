# 资产 42 · 集成面：schema 驱动表单、审批作为注入字段、连接器鉴权的结构化失败

> 证据根（只读）：`bottom_pane/mcp_server_elicitation.rs`（2884 行）、`bottom_pane/hooks_browser_view.rs`（1689）、
> `bottom_pane/{skills_toggle_view,memories_settings_view,app_link_view,feedback_view,feedback_note_view}.rs`（533/472/1757/574/441）。
> 对照根（只读）：`frontend/src/gaea/components/{CapabilitiesPanel,KnowledgePanel,MemoryPanel}.tsx`、
> `frontend/src/pages/MemoryHubPage.tsx`、`frontend/src/gaea/components/memoryhub/*`、`frontend/src/pages/ProgrammingPage.tsx`。
>
> 只出机制与落点；不改 gaea 源码。

---

## 一、MCP 请求表单：**schema 是唯一来源**，空 schema 有两种含义

`mcp_server_elicitation.rs:118-160` 有三层结构：`McpServerElicitationFieldInput`（字段输入控件）、
`required: bool`（必填）、`McpServerElicitationResponseMode`（响应模式）。
schema 直接用协议类型反序列化：`McpElicitationEnumSchema` / `McpElicitationPrimitiveSchema` / `McpElicitationSingleSelectEnumSchema`（`:7-9`）。

三种 schema 形态决定三种交互（`:229-252`）：

| 判定 | 证据 | 交互 |
|---|---|---|
| `is_message_only_schema`（`null` 或空 object） | `:237-244` | 只要一个"确认"，不渲染任何字段 |
| `is_tool_approval_action`（工具审批 + message-only） | `:245` | 走审批语义（见 §二），不渲染表单 |
| 带字段 object | `:252+` | 逐字段渲染：文本 / 枚举单选 / 布尔…（按 `required` 标必填） |

占位文案分必填/可选两条（`:65-66`）：`Type your answer` / `Type your answer (optional)`——
**必填与否写在占位里**，不靠红星。

`MIN_COMPOSER_HEIGHT = 3` / `MIN_OVERLAY_HEIGHT = 8`（`:68-69`）：表单弹层有最小高度，不会缩成一条缝。

## 二、审批不是"另开一套弹窗"，是**注入一个字段**

`:70-75`：

```
APPROVAL_FIELD_ID = "__approval"
APPROVAL_ACCEPT_ONCE_VALUE    = "accept"
APPROVAL_ACCEPT_SESSION_VALUE = "accept_session"
APPROVAL_ACCEPT_ALWAYS_VALUE  = "accept_always"
APPROVAL_DECLINE_VALUE        = "decline"
APPROVAL_CANCEL_VALUE         = "cancel"
```

字段 id 用双下划线前缀（`__approval`）——**协议层的保留命名空间**，避免与用户 schema 里的合法字段撞名；
五个取值覆盖"本次/本会话/始终/拒绝/取消"。

工具参数预览有明确上限（`:76-77`）：`APPROVAL_TOOL_PARAM_DISPLAY_LIMIT = 3`（只显示 3 个参数）、
`APPROVAL_TOOL_PARAM_VALUE_TRUNCATE_GRAPHEMES = 60`（值截断 60 grapheme）。
另有四个 meta key 常量（`:78-82`）：`tool_type` / `tool_id` / `suggest_type` / `suggest_reason` / `install_url`——
服务端"建议安装/启用/鉴权"是**结构化字段**，不是从文案里猜。

## 三、Hooks 浏览是两页 + 三列

`hooks_browser_view.rs:42-59`：

- `HooksBrowserPage::{Events, Handlers(HookEventName)}`——先看事件表，再进某个事件的处理链；
- 列宽 `EVENT_COLUMN_WIDTH = 22`、`COUNT_COLUMN_WIDTH = 12`；
- `MAX_COMMAND_DETAIL_LINES = 3`：命令详情最多 3 行（多出的走展开，不撑爆行高）；
- 视图同时持有 `warnings` 与 `errors`（`from_entry(hooks, warnings, errors, …)`）——**"有钩子但配置有问题"必须能在同一页看到**。

## 四、记忆设置只有两个开关 + 一个不可逆动作

`memories_settings_view.rs:36-71`：

- `MemoriesSetting{Use, Generate}`：**用**记忆 / **生成**记忆，拆成两个独立开关（不是一个大开关）；
- `MemoriesAction{Reset}` + `reset_confirmation: Option<ScrollState>`：重置**要二次确认**（确认本身是一个视图，不是一个 `confirm()`）；
- `MEMORIES_DOC_URL` 与 `docs_link`：解释链接就在页内。

## 五、应用链接：建议类型四态 + 鉴权失败的结构化

`app_link_view.rs:40-75`：

| 常量/类型 | 值/字段 | 用途 |
|---|---|---|
| `MCP_CODEX_APPS_SERVER_NAME` | `codex_apps` | 应用连接器所在 MCP server |
| `MCP_TOOL_CODEX_APPS_META_KEY` | `_codex_apps` | 工具 meta 里的应用段 |
| `CONNECTOR_AUTH_FAILURE_META_KEY` | `connector_auth_failure` | 鉴权失败段 |
| `…_IS_AUTH_FAILURE_KEY` | `is_auth_failure` | 是否真是鉴权失败（不是网络错） |
| `…_CONNECTOR_ID_KEY` / `…_CONNECTOR_NAME_KEY` | `connector_id` / `connector_name` | 哪个连接器 |
| `AppLinkScreen{Link, InstallConfirmation}` | 两屏 | 链接 → 安装确认 |
| `AppLinkSuggestionType{Install, Enable, Auth, ExternalAction}` | 四态 | 建议动作分四类，各自不同处理 |

关键：**"鉴权失败"是结构化判定**（`is_auth_failure` + 连接器身份），不是"看到 401 就提示重新登录"。

## 六、反馈：先分受众，再决定去哪

`feedback_view.rs:17-26`：`BASE_CLI_BUG_ISSUE_URL`、`CODEX_FEEDBACK_INTERNAL_URL`、`FeedbackAudience`——
内部员工与外部用户走不同入口；`feedback_note_view.rs:39-49` 还有 `FEEDBACK_POLICY_URL` /
`EMPLOYEE_FEEDBACK_FAQ_URL` / `IntroContent`（首屏说明内容）。即**采集前先把"拿去干什么"讲清楚**。

## 七、gaea 现状核对

| gaea | 现状 | 差在哪 |
|---|---|---|
| `gaea/components/CapabilitiesPanel.tsx`（11.6KB） | 能力面：工具/技能清单与开关 | 有清单；缺"按事件看处理链"（hooks 页）与"配置错误同页可见" |
| `pages/MemoryHubPage.tsx`（42.6KB）+ `memoryhub/*` + `MemoryPanel`（38KB） | 记忆库/仓库/工坊（比 codex 丰富） | 缺"用/生成"两开关的显式拆分与"重置二次确认"的就地表达（确认多走 Modal，可保留） |
| `pages/ProgrammingPage.tsx`（17.4KB）+ dsh-ops 插件 | MCP 工具接入（229 工具）、能力发现 | 无 elicitation 表单：MCP 若向用户要参数/确认，目前无通用渲染层 |
| 全仓 | 无连接器/应用链接面 | 缺；若接第三方应用需自带四态建议类型 |
| 全仓 | 无反馈入口 | 缺（可后置） |

## 八、三档判定

**档一（照搬语义）**

1. **schema 驱动表单**：字段类型/必填/枚举全部来自 schema，UI 不写死字段名；空 schema 走"仅确认"。
2. **保留命名空间**：注入字段用 `__` 前缀，避免与业务字段撞名。
3. **审批五值**（once/session/always/decline/cancel）——与 gaea 现有 `approval_mode` 语义对齐后再用。
4. **预览上限**：参数 3 条、值 60 grapheme。
5. **鉴权失败结构化**：`is_auth_failure + connector_id + connector_name` 三件套。

**档二（改造搬）**

6. **hooks 两页浏览**（事件 → 处理链）+ warnings/errors 同页。
7. **记忆双开关**（用/生成）——gaea 目前是"记忆总开关 + 多个库"，语义更粗。
8. **建议动作四态**（install/enable/auth/external）——将来接第三方能力时直接用。

**档三（不搬）**

9. `FOOTER_SEPARATOR = " | "`、`MIN_*_HEIGHT` 等终端排版常量；
10. 反馈入口的内部/外部受众分流（gaea 是私有产品，无内部域名）。

## 九、施工卡

| 卡 | 落点 | 改动 | 验收 |
|---|---|---|---|
| **F42-A schema 表单** | 新 `gaea/components/ui/SchemaForm.tsx`（`lib/schemaForm.ts` 纯函数解析） | 支持 object/string/enum/boolean/array + required；空 schema ⇒ 仅确认 | 单测：必填缺失不可提交；枚举渲染为选项；60 grapheme 截断生效 |
| **F42-B 审批注入字段** | 同上 + `ApprovalModal` 数据源 | `__approval` 五值映射既有 `approval_mode` | 单测：五值到 action 的映射一一对应；未知值拒绝 |
| **F42-C hooks 浏览** | `CapabilitiesPanel` 新增页 | 事件表（列宽 22/12）+ 处理链（详情 ≤3 行）+ warnings/errors | 单测：有 error 的钩子在事件行可见；详情超 3 行折叠 |
| **F42-D 记忆双开关** | `MemoryHubPage`/`MemoryPanel` | `useMemory` / `generateMemory` 分离 | 单测：关"生成"不影响"使用" |

**开关**：A/B 为新增件（无 MCP 请求时不挂载）；C/D 改现有面板，`FEATURE_CAPABILITIES_V2` 单闸。

## 十、风险

1. **与 `approval_mode` 对齐**：gaea 后端已有 `approval_mode.go`（approval/always/deny… 语义），
   `__approval` 五值必须先映射到它，不许在 UI 层另立一套决策枚举。
2. **schema 不可信**：schema 来自 MCP server（第三方），必须限深度/字段数/字符串长度，防超大 schema 拖死渲染。
3. **两壳**：`CapabilitiesPanel`、`MemoryPanel` 在 `gaea/` 与 `wenxue/` 都有同名件（`WenxueSettingsHub` 35.7KB），先定单源。

## 十一、证据自查

- 表单与 schema：`mcp_server_elicitation.rs:7-9,65-82,118-160,206-252`。
- 审批字段五值：`:70-75`；预览上限：`:76-77`；meta key：`:78-82`。
- hooks 浏览：`hooks_browser_view.rs:42-59`。
- 记忆设置：`memories_settings_view.rs:36-71`。
- 应用链接：`app_link_view.rs:40-75`；反馈受众：`feedback_view.rs:17-26`、`feedback_note_view.rs:39-49`。
