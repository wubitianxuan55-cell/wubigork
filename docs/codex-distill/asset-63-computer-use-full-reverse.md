# asset-63 · 电脑操控全量逆向还原（Electron 前端 + JS 客户端 + 原生 helper + launcher）

> 与 `asset-62` 的关系：62 是**协议与行为规格**（够用来重写）；63 是**代码级还原清单**（够用来抄实现/对照）。
> 证据包（仓库外，约 74 MB）：`C:\AI\re\codex-computer-use\`，索引 `data\INDEX.md`。
> 本轮新增证据源：Electron 主程序 `%USERPROFILE%\.codex\zh-cn-patched\<hash>\app\resources\app.asar`（761 MB / 19 791 文件）、
> `app.asar` 内主进程 bundle `.vite\build\early-bootstrap-715-*.js`（5.8 MB）、renderer `webview\assets\async\*.js`、
> 原生 helper `codex-computer-use-swift.exe`（56.7 MB，sha256 `f672c4ef…`）、启动器 `codex-computer-use.exe`（1.55 MB，Rust）。

---

## 1. 一句话结论

codex 的"控制电脑"没有一行是模型直接调用的专用工具代码，而是**六层拼起来的**：

| # | 层 | 载体 | 还原等级 |
|---|---|---|---|
| 1 | 模型指令 | 插件 `SKILL.md` / `docs/*.md` / `@oai/cua docs/*.md` | **源码级**（原文） |
| 2 | 模型运行面 | `node_repl` / `cua_repl` MCP server（`@oai/cua-repl`）+ `banner.js` | **源码级** |
| 3 | 客户端 SDK | `@oai/sky` 0.7.17（未混淆 dist JS + `.d.ts`）、`@oai/cua` 0.2.6 | **源码级** |
| 4 | Electron 前端 | 主进程 bundle（computer-use 桥 + overlay 配置 + executor env）+ renderer（设置/审批/PiP/i18n） | **切片级**（整函数已从 minified 抽出） |
| 5 | 原生 helper | `codex-computer-use-swift.exe`（UIA / WGC / SendInput / overlay） | **符号级 + 汇编级**（无源码，无符号表；结构靠 Swift 反射表，行为靠 xref 反汇编） |
| 6 | 启动器 | `codex-computer-use.exe`（Rust） | **汇编级**（5 个函数已拆） |

**唯一没有源码的是第 5 层**——它是 Swift 静态编译产物、符号已 strip；但 `.sw5rfst` 反射串表留下了 **5 134 个字段/枚举名**，
加上 PE 导入表（90 DLL / 680 函数）与 xref 反汇编，足以把它的状态机、数据模型与调用顺序逐条写死（§5）。

---

## 2. Electron 前端：`openai-codex-electron@26.1103.31037`

### 2.1 包结构（`app.asar` 文件表全量已导出）

```
.vite/build/…            主进程 bundle（early-bootstrap*.js、worker.js、browser-page-preload.js …）
webview/…                渲染进程（Vite 产物：assets/async/<chunkid>.<hash>.js，最大单块 ~12 MB）
native-menu-locales/     本地化字符串（overlay 用它取文案）
node_modules/…           运行时依赖（含 @oai/cua-repl workspace 包）
package.json             49 个依赖，含 @oai/cua-repl、browser-api、app-server-manager、node-pty
```

- asar 头 5.3 MB（JSON 文件表 + 每文件 offset/size/SHA256）→ `data\frontend\asar-header.json`、`asar-files.csv`（19 791 行）。
- 抽取方式：按 `offset` 直接切（data 基址 = `8 + headerSize` = 5 336 136）→ `data\frontend\asar\`（129 文件 / 26.7 MB，仅保留与 computer-use 相关的块）。

### 2.2 主进程：computer-use 桥（`carve-computer_use_openai_bundled-01.js`，2.9 KB，整函数已还原）

Windows 上 helper 走 **native pipe**，主进程在这里做 JSON-RPC 2.0 服务端：

```js
// 每 socket 一条：method ∈ {ping → "pong", close, request}
// request 载荷 = {codexTurnMetadata, method, params}（helper 的方法名）
// 真正转发： (await getHelperTransport()).request(method, params,
//              { codexTurnMetadata, createElicitation: e => approvalBridge.requestApprovalForSender(sendResponse, e) })
// 错误统一包成 {error:{code:-32000, message}}
// 每个请求发 analytics {type:"tool-called", pluginId:"computer-use@openai-bundled",
//   mcpServerName:"node_repl", invocationSource:"code_mode", threadId, turnId, threadSource, itemId,
//   terminalStatus:"completed"|"failed", mcpErrorPresent, durationMs}
// end_turn 特判：只允许当前活跃 turn 的 session，否则直接 result:null（不转发）
// 并发：同一 socket 有在飞请求时，第二个 request 直接回 "Computer Use helper already has an active request"
```

要点（gaea 可借）：**end_turn 只对当前活跃回合生效**、**单 socket 串行**、**analytics 与错误包不掺进业务返回**。

### 2.3 主进程：overlay 配置下发（`~/.codex/computer-use/config.json` 的真正写者）

```js
{ accentColor, direction, locale,
  strings: { usingComputer: sd(settings,"computerUseOverlay.usingComputer","ChatGPT is using your computer"),
             escToCancel:   sd(settings,"computerUseOverlay.escToCancel","Esc to cancel") } }
```

- `accentColor`：取主题 accent，正则 `/^#[0-9a-fA-F]{6}$/` 校验，非法回落 `#339cff`。
- `direction`：`new Intl.Locale(locale.replaceAll("_","-")).getTextInfo().direction === "rtl" ? "rtl" : "ltr"`。
- `locale`：`app.getLocale()`；文案目录 `native-menu-locales`（打包）/`webview/src/locales`（开发）。
- 失败只记 warning（`Failed to write Computer Use config`），不阻断。

**结论**：那句 "ChatGPT is using your computer / Esc to cancel" 的浮窗**不是 Electron 画的**，是 Swift helper 自己的 Win32+D2D 窗口；
Electron 只负责把**文案、语言方向、强调色**写进 config 文件。

### 2.4 主进程：`cua_repl` 的进程环境（`carve-CUA_REPL_ENABLED_SURFACES-01.js` / `carve-SKY_CUA_NATIVE_PIPE-01.js`）

```js
CUA_REPL_ENABLED_SURFACES = "browser,computer"         // 缺/空/未知 → 启动即抛
NODE_REPL_TRUSTED_SERVICES = { sky: "@oai/sky/service" } // 可信服务白名单（computer 面）
NODE_REPL_JS_BANNER        = <instructions/banner.js>   // 默认 `await import("@oai/cua/tinyskyAlt")`
NODE_REPL_TOOL_OVERRIDES   = { server_instructions, tools: {
    js: { description: <description+browser|computer+output+audio 拼接>, field_descriptions:{code,title} },
    js_reset: { description } } }
NODE_REPL_JS_COMPLETION_SUMMARY = { instructions: task_completion, description: completion_summary }
NODE_REPL_UNTRUSTED_ENV_ALLOWLIST += "CUA_REPL_ENABLED_SURFACES,CUA_REPL_BROWSER_ENV"
SKY_CUA_NATIVE_PIPE="1" + SKY_CUA_NATIVE_PIPE_DIRECTORY=<dir>   // 仅 win32
SKY_ENABLE_AUDIO = "1" 且 NODE_REPL_ENABLE_AUDIO="1" 才开音频
CODEX_INTERNAL_ORIGINATOR_OVERRIDE = "codex_work_desktop"
```

- 指令是**按平台 + 按启用面拼 Markdown**：`instructions/<platform>/{description,computer,output,audio,browser}.md`；
  未启用 computer 就塞 `computer-disabled.md`。这就是 `@oai/cua-repl\instructions\` 那堆文件的用途。
- 命令默认 `process.execPath`（即 cua_node 的 node），非 Windows 用 `/bin/sh -c '… exec "$@"'` 包一层做 access executor。

### 2.5 Renderer（前端 UI 面）

从 chunk 里抓到的**键名即功能清单**（368 条 i18n 命中，节选）：

| 键 | 默认文案 / 用途 |
|---|---|
| `settings.nav.computer-use` | "Computer use"（设置页导航项） |
| `computerUse.label` | "Computer use" |
| `computerUseDescription` / `computerUseFollowUp` / `computerUsePreview` | 设置页说明、跟进、预览 |
| `computerUseAppApproval` | 单 app 审批界面（对应 `approvalRequest.app/displayName`） |
| `computerUseAccessGranted` / `computerUseAvailable` | 授权/可用状态 |
| `computerUseAlwaysHidePictureInPicture` | 画中画始终隐藏（**存在 PiP 浮层**） |
| `computerUseOverlay.usingComputer` / `.escToCancel` | 写进 overlay config 的两条文案 |
| `restricted.cloudComputer.{title,approvals,cookies}` | 云端电脑（另一形态）的审批与 cookie 管理 |

- 块级证据：`11951`（设置描述符）、`107421`（设置导航 + cloud computer）、`224153`（computer-use 相关组件）、
  `159406`（`computerUseAvailable/AccessGranted`）、`99588`（i18n 目录）、`611468`（18 处 `computer-use`）。
- 设置项统一走描述符 `{agentAccess, default, description, key, schema}`（例：`computerUseAlwaysHidePictureInPicture`，
  `agentAccess:"hidden"`）——**同一个设置项同时喂给 UI、agent 与默认值**。

---

## 3. JS 客户端（源码级，全部就位）

| 包 | 版本 | 关键文件 | 内容 |
|---|---|---|---|
| `@oai/sky` | 0.7.17 | `dist\project\cua\sky_js\src\targets\windows\internal\{helper_transport,computer_use_client,computer_use_client_base}.js`、`desktop_client.js`、`service.js`、`sky.js` | 行 JSON / native pipe 双传输、方法表、参数规整、截图落盘、超时 |
| `@oai/cua` | 0.2.6 | `dist\lib\js\oai_js_cua\src\tinysky_alt\{bind_windows_app,create_tinysky_alt,bind_tab}.js`、`core\*` | `cua` 高层 API（App/Tab/Target）、状态拼装、文档注入 |
| `@oai/cua-repl` | — | `bin\cua-repl.mjs`、`dist\...\instructions.js`、`instructions\<platform>\*.md` | NodeREPL + `cua` API 的装配与平台指令 |
| 文档 | — | `@oai/cua\docs\tinysky-alt-core-*-windows.md`（350 行）等 11 篇 | 模型契约原文 |

`bind_windows_app.js` 里的**状态拼装公式**（gaea 直接抄）：

```js
state = [tree, focused_element && `Focused element: ${…}`, selected_text && `Selected text: ${…}`,
         selected_elements?.length && `Selected elements:\n${join}`, document_text && `Document text:\n${…}`]
        .filter(Boolean).join("\n\n");
// tree 为 null → `Accessibility state unavailable for window <id>.`
// 多 region 截图 → "Screenshot regions (display order):\nImage N: screenshotId=\"…\"\nPass the matching screenshotId for coordinate input."
```

---

## 4. 原生 helper：结构（Swift 反射表 5 134 名）

`.sw5rfst`（reflection strings，123 KB）导出全部字段/枚举名 → `data\native\swift-reflection-strings.txt`。分组要点：

| 组 | 名字（节选） |
|---|---|
| 故障/拒绝枚举 | `accessibilitySnapshotRequired` `windowChanged` `elementStateChanged` `accessibilityElementChanged` `boundsChanged` `boundsUnavailable` `emptyBounds` `activationRejected` `activationIncomplete` `foregroundChangedAfterRestoration` `restorationFailed` `keyboardCompletionFailed` `ownerChanged` `identityChanged` `browserDialogChanged` `elementPreparationSuperseded` `invalidParentProcessID` `missingParentProcessID` |
| 会话/请求 | `requestTask` `requestTaskFinished` `responseTask` `waiter` `deadline` `queued` `queuedIndex` `queuedByteCount` `userStopWasObserved` `terminate` `continuation` |
| 抓屏状态机 | `creatingCaptureItem` → `creatingFramePool` → `configuringCaptureSession` → `startingCaptureSession` → `waitingForFrameArrival` → `readingFrameSize` → `queryingFrame` → `recreatingFramePool` → `closingFrame` |
| 抓屏数据 | `captureSessionID` `captureSequence` `frameTimes` `frameOwnerIdentity` `staleIdentityBeforeCapture` `staleIdentityAfterCapture` `maskScreenshotTop/Bottom` `rootScreenshotID` `screenshotSpaces` `ancestorAllowsTextCapture` |
| 无障碍 | `tree` `elements` `treeElement` `accessibilityRevision` `accessibilityTimeoutMilliseconds` `elementIndexOutOfBounds` `textSegment{Grapheme,Word}Mode` `scrollIntoView` `scroll{Up,Down,Left,Right}` |
| UIA 接口名 | `CreateTreeWalker` `get_{Control,Content,Raw}ViewWalker` `GetRootElement{,BuildCache}` `ElementFrom{Handle,Point}{,BuildCache}` `GetFocusedElement{,BuildCache}` `Get{Parent,FirstChild,LastChild,NextSibling,PreviousSibling}Element` `{get,put}_Tree{Scope,Filter}` `{get,put}_AutomationElementMode` `{get,Current,Cached}_Is{Control,Content}Element` |
| overlay/光标 | `overlay` `overlayWindow` `overlayEntryAnimation` `idleOverlay` `isOverlayPresented` `cursorOverlay` `cursorPresenter` `cursorRasters` `backgroundCursors` `nextCursorID` `cursorBefore` `cursorAfter` `activeOverlayTeardownCount` `overlayTeardownWaiters` |
| 输入 | `keyboard` `keyboardAndMouse` `mouseButton` `clickCount` `clickAngle` `click` `clickElement` `scroll` `scrollElement` `drag` `dragID` `paste` `text` `textField` `keyNotFound` `keyCollision` `keyPath` `textSelectionUnavailable` `textNotFound` `textWithPrefix` |
| WinRT/WGC 接口 | `get_IsCursorCaptureEnabled` `CreateFastMetadataEncoderFromFrameDecode` `GetFrameCount` `GetFrame` `DoesSupportMultiframe` |

配套：PE 导入表（`data\native\swift-helper.pe-imports.json`）、全量串表（`swift-helper.strings.txt`，2.03 MB）、
xref 反汇编输出（`data\native\decomp-helper\asm\*.asm`，见 §5）。

---

## 5. 原生 helper：行为（xref 反汇编法）

工具：`scripts\xref_disasm.py`（pefile + capstone 5.0.7，**不需要 Ghidra**）。流程：
字符串 VA → 线性扫 `.text` 找 RIP-relative 引用 → 回溯函数头（`55 48 89 e5` / `48 83 ec` / 填充字节）→ 整函数反汇编 → `.asm`。

已定位的目标串（每串对应一条实现路径）：

| 串 | 命中 | 说明 |
|---|---|---|
| `Secondary Actions: ` | ✅ 1 处 RIP 引用 → 函数 `0x14029d3cc`（≥600 指令，被 600 指令上限截断） | 树渲染器（节点行后缀） |
| `data:image/jpeg;base64,` | ✅ 5 处 → 函数 `0x140040128` / `0x1400404dc` / `0x140040ba8` / `0x1400b3efa`(73 指令) / `0x1400b422a`(96 指令) | 截图编码（前三个是同一族的重载，后两个是短包装） |
| `[accessibility tree truncated` | ⚠️ 串在（3 处），**未找到 RIP/指针引用** | 树截断器 |
| `window changed; call get_window_state before using an accessibility element` | ⚠️ 串在（1 处），未找到引用 | 索引失效闸门 |
| `element index is out of bounds` | ⚠️ 串在，未找到引用 | 索引越界 |
| `was not found. Current windows: ` | ⚠️ 串在，未找到引用 | 窗口查找失败 |
| `requires approval to use ` | ⚠️ 串在（2 处），未找到引用 | 授权拒绝 |
| `requestComputerUseApproval` | ❌ 本二进制里没有该串 | 反向请求方法名由**客户端侧**（`computer_use_client.js`）处理，helper 侧用别的编码 |

反汇编产物：`data\native\decomp-helper\asm\*.asm`（6 个整函数，带 `; -> "<literal>"` 注释）。

**为什么有的串找不到引用（方法论边界）**：capstone 只认 `RIP-relative` 操作数与 data 里指向该串的绝对/相对指针（两跳都扫过，命中 0）。
未命中的那几条显然不是这两条路径可达——最可能是 Swift 把描述符放在 `__TEXT,__const` 的一层间接表、
或走 `movabs` 立即数；这两类本脚本没覆盖。**但这不影响行为结论**：这些文案本来就是通过 §3 的实测探针
（`probe2` / `probe3` 的 JSON 响应）拿到的**运行时证据**，比反汇编更强。

启动器 `codex-computer-use.exe`（Rust + serde_json）同理已拆 5 个函数（`data\native\decomp-launcher\asm\`）：
`failed to launch codex app-server`（`0x14007a53c`）、`--parent-pid`（`0x1400815bb`）、三处 `get_window_state`
（`0x14008966b` / `0x140097821` / `0x14009f9f4`）。它的职责就是**拉起 app-server（批准 UI 由它承担）后把 stdio 交给 swift helper**；
所以单独跑它必然报 `failed to launch codex app-server: program not found`（实测）。

---

## 6. 还原度与缺口（诚实清单）

| 面 | 状态 |
|---|---|
| 模型指令 / 技能文档 | ✅ 原文 |
| `@oai/sky` / `@oai/cua` / `@oai/cua-repl` | ✅ 全量 dist（含 `.d.ts` 与 `docs/`） |
| Electron 主进程 computer-use 桥 / env 组装 / overlay 配置 | ✅ 整函数切片（15 段） |
| Electron renderer UI | ◑ 键名 + 片段 + i18n 目录（368 条）；未做整应用模块级重排 |
| 原生 helper 数据模型 | ✅ 5 134 反射名 + 错误枚举族 |
| 原生 helper 控制流 | ◑ xref 反汇编（目标函数已拆，非全量伪代码） |
| 原生 helper 全量伪代码 | ❌ 无符号表 + 无 Ghidra/IDA 环境；如需，装 Ghidra 12.2 + JDK21 后 `analyzeHeadless` 可补 |
| `codex.exe`（357 MB Rust 主程序）里的调度/策略 | ◑ 串级：`computer_use` 策略模板、`ComputerUseConfigToml`、`computer-use@openai-bundled` 引用；未反编译 |

---

## 7. gaea 落点（在 asset-62 三档之上，新增两条）

1. **overlay 文案/主题下发**照抄：一个 `config.json`（`accentColor/direction/locale/strings`）+ 正则校验 + 失败只 warning。
   比自己起一个 Electron 浮窗便宜得多，且能复用系统本地化目录。
2. **单 socket 串行 + end_turn 只认活跃回合**：把 `Computer Use helper already has an active request` 这条闸门照搬，
   避免"两个回合同时驱动鼠标"这类最难查的错。
3. 前端只做**三处**就够：设置页开关（`computerUse*` 键族）、单 app 审批卡（`approvalRequest` 三字段 + 持久化档位）、
   回合内状态条（Using/Used computer + Esc 停止）。

---

## 8. 复现命令（全部可重跑）

```powershell
# asar 文件表 + 指定块抽取（data 基址 = 5336136）
#   见 scripts\（asar 解析已内联在会话里；核心：读 u32@12 为 header 长度，JSON 表里取 offset/size）
# 主进程/渲染进程反编译切片（最小化 JS → 整函数）
node scripts\carve.mjs <bundle.js> <outDir> "escToCancel" "computer-use@openai-bundled" "CUA_REPL_ENABLED_SURFACES"
# 原生反汇编（不需要 Ghidra）
python scripts\xref_disasm.py <binary.exe> <outDir> "Secondary Actions: " "data:image/jpeg;base64,"
# Swift 反射表
python -c "import pefile;pe=pefile.PE(sys.argv[1]);[print(s.Name, len(s.get_data())) for s in pe.sections if b'sw5rfst' in s.Name]" <helper.exe>
```

---

## 9. 验收断言（新增）

1. overlay config 的 `accentColor` 非法值必须回落 `#339cff`；`direction` 只允许 `ltr|rtl`。
2. 同一 socket 并发第二个 `request` → `Computer Use helper already has an active request`。
3. `end_turn` 对非活跃 turn → 直接 `result:null`，不落到 helper。
4. helper 拒绝授权时返回体必须同时含 `error` 与 `approvalRequest{app,displayName,riskLevel}`。
5. 反汇编脚本对任一目标串必须能在 `.text` 找到 ≥1 处 RIP-relative 引用（否则说明串在别处/编译期合并）。
