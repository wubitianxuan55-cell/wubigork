# WPS 灵犀 app-server (5.24.0) 后端逆向报告

目标：`C:\Users\wubi\AppData\Roaming\WPS 灵犀\app_server_5.24.0\service\*.exe`（只读，未运行）
调用方：`re-lingxi/out/x/out/main/index.js`（Electron 主进程，minified）
标记：`[观]`=字符串命中 / `[推]`=结构推断 / `[未]`=未知

## 0. 身份与构建

```
ksogit.kingsoft.net/lingxi-ai/lingxi-app-server/internal/nativehost.(*Server).ListenAndServe
git.kingsoft.net/lingxi-ai/lingxi-app-server/internal/wpsclibroker
H:/release/cmd/wpscli-broker-poc/main_windows.go
```
- Go 模块 `lingxi-app-server`，含 Go 类型元数据（可读出 struct 名与 `json:"..."` tag，共 2634 个 tag）`[观]`
- **架构结论**：app-server 本体走 **stdio JSON-RPC**，不开 TCP 端口 `[观+推]`

## 1. HTTP 路由（`grep -a -o`，950 条去重，按前缀分组）

噪声：`/api/v3/etcdserverpb`(1919)、`/internal/*`(Go 标准库包路径)、`/v2/protoc-gen-openapiv2/*`。真实路由如下 `[观]`：

| 前缀 | 资源（样例） | 归属 |
|---|---|---|
| `/api/public/v1/` | `sessions`、`sessions/{session_id}`、`{session_id}/title`、`agents/`、`agents/fork_from_session`、`agents/run_records`、`agents/{agent_id}/memory/{search,pages,status}`、`skills`、`skills/batch`、`skills/inspect`、`connector-skills/{name}/manifest`、`projects/`、`credits/{balance,consume,runs,transaction_summary}`、`dev/credits/*`、`dev/agents/batch`、`invite_codes/redeemed`、`account/profile`、`agent-runtime/*` | 云 Public API（app-server 作客户端） |
| `/api/aioffice/v1/` | `sessions/{session_id}/messages`、`sessions/{session_id}/pin`、`sessions/plan`、`sessions/rollout`、`agent/parallel-tools/policy`、`immemory/compact`、`short_link`、`trace/config`、`web_search`、`cowork/sessions` | WPS 文档侧（aioffice） |
| `/api/agent-runtime/` | `chat/completions`、`chat/recommendations`、`search/{web,image,academic,data-sources}`、`fetch/urls`、`image/{generate,edit,upload}`、`asr/transcribe`、`vision/queryDesktop`、`ppt/templates[/recommended]`、`search/macro` | 本地 agent 能力编排 |
| `/api/logs/v1/` | `app-server/logs`、`app-server/dw`、`app-server/dw/events`、`metrics` | 遥测上报（`https://lingxi.wps.cn/api/logs/v1/logs`） |
| `/api/v2/` `/api/v3/` | `v2/llm/{chat,multimodal,videobackend,embedding}`、`v2/asr/audio2text`、`v2/tts/base`、`v2/aigc/*`、`v2/mcp`、`v2/ws/audio2text`、`v3/chat/completions`、`v3/responses`、`v3/images/{generations,edits}` | AI Gateway |
| `/api/v5/files/`、`/api/v7/drives/`、`/api/v7/longtask/drives/`、`/v1/sessions/sync`、`/v1/label/items/query`、`/office/v5/dev/ai/crawler/crawl` | 文件/云盘/老 WPS 接口 | 云 |

方法前缀路由标签（Go mux 风格）：
```
GET_/api/public/v1/sessions/   POST_/api/public/v1/sessions        PUT_/api/public/v1/sessions/
DELETE_/api/public/v1/sessions/  POST_/api/public/v1/agents/fork_from_session
GET_/api/public/v1/skills      POST_/api/public/v1/skills/batch    PATCH_/api/public/v1/agents/
```

## 2. 本机接口：JSON-RPC 方法 + 事件（非 HTTP）

`index.js` 与 `lingxi.exe` 双向命中的方法名 `[观]`：

| 类 | 方法 |
|---|---|
| 会话 | `session/start`、`session/list`、`session/messages`、`session/subscribe`、`session/unsubscribe`、`session/delete`、`session/title_updated`、`session/pinswitch` |
| 运行 | `run/start`、`run/steer`、`run/interrupt`、`run/error` |
| 工具/技能 | `tool.call`、`skill/list_context`、`skill/confirm`、`resources/skills` |
| 配置/鉴权 | `config/read`、`config/write`、`auth/refresh`、`account/logout`、`bash/auth` |
| 其他 | `client/operation`、`connector/{check,manage,resolve}`、`canvas/edit`、`model/test`、`webapi/model`、`immemory/compact`、`timer/task`、`pet/host_event` |
| Go 侧生命周期 | `app_server.start` / `.stop` / `.shutdown` / `.run` |

SSE/流式：`Content-Type: application/json, text/event-stream`、`text/event-stream`、`data:` 分帧、`content_block_delta` / `char_delta` / `line_delta` / `patch_delta` / `title_updated` `[观]`。
WS：`/v2/ws/audio2text`、`/ws/audio2text`（语音）；native-host 侧 `Sec-WebSocket`、`[NH-MSG] cdp.event: method=%s session=%s` `[观]`。

## 3. 工具名（证据强度）

| 强（工具 prompt 模板存在，`tools/*.tmpl`） | 强（代码内固有标识） |
|---|---|
| `ask_user_question` `await` `bash` `connector_exec` `connector_manage` `create_plan` `creative` `deliver_result` `exec` `exec_auth` `execute_jupyter` `file` `generate_image` `group_chat` `imagesearch` `manage_skills` `memory` `pip_install` `project_insight` `render_widget` `search` `share_file` `switch_mode` `timer_task` `todo_write` `url_fetch` `view_image` `web` `wiki_memory` | `fs_read_file` `fs_write_file` `fs_edit_file` `fs_list_dir` `fs_download_file` `bash_exec` `bash_exec_sandbox` `bash_auth` `pip_install` `jupyter` `websearch` `await_process` `tool.call` |

中：`create_plan`/`switch_mode` 与 `plan`/`mode` 状态机绑定；`mcp/*` 包存在（`mcp/tool`、`mcp/elicit`、`mcp/streamable`）→ MCP 工具运行时动态命名 `[推]`。
弱/否定：`mcp__*`=0、`str_replace`=0、`spawn_agent`=0、`web_fetch`=0、`function_call`=3（仅透传字段）、`input_schema`=1。→ 该版本**不使用** `mcp__` 前缀与 `str_replace`/`spawn_agent` 命名 `[观]`。

## 4. 启动链（Electron → app-server）

```js
// index.js: 解析可执行文件
async function uc(t,e=process.platform){const n=e==="win32"?".exe":"",
  r=[y.join(t,"service",`lingxi${n}`),y.join(t,`lingxi${n}`)];...}     // service/lingxi.exe

// index.js: 参数 + 管道
function vde(t){return["--stdin","--auth-storage=runtime","--config",t.configFile,"--data-dir",t.dataDir]}
const a=n(t.executablePath,o,{cwd:t.cwd,env:t.env,windowsHide:!0,stdio:["pipe","pipe","pipe"]});
```
启动参数只有这 4 个：`--stdin`、`--auth-storage=runtime`、`--config <file>`、`--data-dir <dir>`；**无 `--port`/`--listen`** `[观]`。stdin/stdout 走 **NDJSON JSON-RPC 2.0**（`jsonrpc:"2.0"` 校验函数 `G3/K3/Y3/Wle`，逐行 `JSON.parse`）`[观]`。

健康/版本：`app_server_link`（userData 下文件，缓存版本号）、`bundled-version.json`、`app_server_` 前缀目录；退出码 `ERROR_EXIT_CODE=493`；错误码族 `local_app_server_{not_ready,startup_failed,request_timeout,auth_failed,connection_failed,runtime_restarted}`（`v7code` 400449020…400449029）`[观]`。
依赖安装：`service/requirements.txt` → `python -m pip install -r` `[观]`（该文件确实存在，1224 B）。

## 5. 旁路进程角色

| 进程 | 语言 | 通道 | 角色 |
|---|---|---|---|
| `lingxi-native-host.exe` | Go | stdio 原生消息（4 字节长度帧，`frame too large: %d bytes`）+ 回环 HTTP/WS | 浏览器扩展 **native messaging host**（宿主名 `com.wps.lingxi`，`registerNativeMessagingHost`）+ 内置 `nativehost.Server/EmbeddedServer`，监听 `127.0.0.1:%d` 并把端口写入 `<userData>/serverdir/nativehost.port`；同时是 **CDP 代理**（`CDPRequest/Response`、`[NH-CDP]`、`Browser.getVersion`）。类型：`NMInbound/NMOutbound/StdinIdleMonitor/PortCleanupResult` `[观]` |
| `lingxi-wpscli-proxy.exe` | Go | 命名管道 `\\.\pipe\lingxi-wps-broker-%d-%s` | **wpscli 代理**：`internal/wpsclibroker`，`brokerRequest/brokerResponse`，`broker frame is too large`，`wpscli arguments are too large`，子进程用 `go-winio` 管道通信；由 app-server 以 `lingxi-wpscli-proxy.exe` + `LINGXI_WPS_BROKER_TOKEN` 拉起，代理 `wpscli`（KDocs CLI，`KDOCS_CLI_APP_ID`）`[观]` |
| `lingxi-sandbox-runner.exe` | **Rust** | 命名管道 `\\.\pipe\lingxi-runner-%x` | 源自 **Codex CLI** 的 Windows 沙箱执行器：`codex_project`、`codex_execpolicy`、`.codex.sandbox`、`codex_home/cap_sids/network_proxy_restricting_sid/use_private_desktop`。协议帧：`command/cwd/env/permission_profile/workspace_roots/timeout_ms/tty/stdin_open` + `ExitPayload{exit_code,timed_out}`、`StdinPayload{data_b64}`，`runner: unsupported protocol version` `[观]` |
| `lingxi-sandbox-setup.exe` | Go | 无长连接 | 沙箱**账号与网络**一次性配置：建沙箱用户组/能力 SID、`SetEntriesInAclW` deny-read ACL、**WFP** 过滤器、`For sandbox users/for sandbox_group` `[观]` |

## 6. 鉴权与沙箱

- 鉴权模式：`--auth-storage` 仅接受 `file` / `runtime`（`lingxi: invalid --auth-storage %q: must be file or runtime`）；`file` 模式用 `auth.json`（`unmarshal/marshal auth.json`、`mkdir for auth file`）与 `LINGXI_AUTH_FILE`；`runtime` 由 Electron 注入内存 `[观]`。
- Token 传递：`X-Kso-AccessToken`、`Authorization`（114 次）、`LINGXI_WPS_SID`/`wps_sid`、`CryptProtectData`(DPAPI)、`X-Lingxi-AppServer-Version`、`X-Client-Language`、`X-Client-Request-Id`；日志脱敏正则 `(?i)(authorization|cookie|set-cookie|wps_?sid|access_?token|refresh_?token|api_?key|secret|request_?id)` + `[REDACTED]` `[观]`。
- 沙箱策略：环境变量 `LINGXI_SANDBOX_POLICY_MODE`（值 `apply-acl` / `static`，内嵌 Python 检查）、`LINGXI_SANDBOX_REGISTERED_CORE`、`sandbox_permissions`；字段 `permission_profile`、`read_roots/write_roots`、`deny_read_paths/deny_write_paths`、`managed_read_root_blacklist`；工具 `bash_exec` vs `bash_exec_sandbox` 双通道 `[观]`。
- Windows 原语：`CreateRestrictedToken`、`CreateWellKnownSid`、`CreateJobObjectW`/`AssignProcessToJobObject`/`TerminateJobObject`、私有桌面（`SetEntriesInAclW failed for private desktop`）、`CreateProcessWithLogonW`（沙箱账号）。`AppContainer` 字符串 0 次 → 未用 AppContainer `[观]`。
- macOS 分支存在但不可用：`seatbelt_unavailable`、`seatbelt_preparation`、`seatbelt sandbox is only available on macOS`、`lingxi-linux-sandbox` `[观]`。

## 7. 端口/协议线索汇总

| 线索 | 值 | 证据 |
|---|---|---|
| app-server | 无端口，stdio JSON-RPC | `--stdin` + `stdio:["pipe","pipe","pipe"]` |
| native-host HTTP | `http://127.0.0.1:%d`，端口文件 `serverdir/nativehost.port`，JS 回退 **38127** | `const G3e=38127` + `readFile(.../nativehost.port)` |
| CDP | `http://127.0.0.1:%d/cdp` | 字符串 |
| 命名管道 | `\\.\pipe\lingxi-runner-%x`、`\\.\pipe\lingxi-wps-broker-%d-%s` | 字符串 |
| MCP 客户端 | `http://127.0.0.1:${port}/mcp`，`Authorization: Bearer`，client `musa-desktop/1.0.0` | index.js `StreamableHTTPClientTransport` |
| 云基址 | `https://lingxi.wps.cn`、`api.wps.cn`、`encs-inner.wps.cn`、`personal-ai-bus.internal.wps.cn`、`aigc-gateway-test.ksord.com`、`http://120.92.124.158`、`api-mcp.51ifind.com:8643` | URL 扫描 |

## 8. 未确定项

- `[未]` lingxi.exe 自身是否有回环 HTTP 路由表：只命中 `nativehost.(*Server).ListenAndServe`（同包内嵌），无法区分是 app-server 主动监听还是仅链接了该包。
- `[未]` `X-Lingxi-AppServer-Version` 的校验端；`auth.json` 的加密结构与派生密钥（仅见 `derive_key`/`chacha20poly1305`）。
- `[未]` MCP 工具的运行时命名规范（包存在但无 `mcp__` 字面量）。
- `[未]` 全部结论为静态字符串与交叉比对，未经运行验证（按要求未启动 exe）。
