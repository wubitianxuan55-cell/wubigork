# gaea 电脑操控 · 施工单（2026-11-19）

> **性质**：施工单，不含任何代码改动。执行时一刀一提交、每刀自带验收与回退。
> **依据**：`docs/gaea-computer-use-landing-research-2026-11.md`（调研）、`docs/codex-distill/asset-62-computer-use-sky-windows.md`（协议规格）、
> `docs/codex-distill/asset-63-computer-use-full-reverse.md`（代码级还原）。
> **坐标**：file:line 均以本文写作时的工作树为准，施工前请按符号名重新定位（行号会漂）。

---

## 0. 一句话

gaea 的电脑操控分成 **8 刀**：先用 W0 把「图片进主对话」的通道打通（0.5 人日，独立可并行），再按 **骨架 → 观察 → 工具 → 授权 → 输入 → 观感 → 门禁** 顺序落，每刀都是"默认关 + 关掉即回到现状"。

| 刀 | 内容 | 依赖 | 估时 | 默认状态 |
|---|---|---|---|---|
| **W0** | 多模态通道（消息带图 + 工具结果带图 + 缩图预算） | 无 | 0.5 人日 | 关（`Message.Images` 为空时逐字节不变） |
| **W1** | `internal/gaea/computer` 包骨架 + 会话/版次/TTL | 无 | 0.5 人日 | — |
| **W2** | 窗口枚举 + UIA 树渲染（离线 golden） | W1 | 1 人日 | — |
| **W3** | 只读三工具 + 设置项（默认关） | W1,W2 | 0.5 人日 | **关** |
| **W4** | 按 app 授权（`subjectKeys` + `AlwaysAsk`） | W3 | 0.5 人日 | 关（工具未注册即不可达） |
| **W5** | 输入注入（click/type/key）+ 双闸门 + DPI + deny 表 | W3,W4 | 1.5 人日 | **关** |
| **W6** | 事件状态条 + 截图给人看 | W3 | 0.5 人日 | 跟随 W3 开关 |
| **W7** | 门禁与验收（单测清单 / build / smoke） | 全部 | 0.5 人日 | — |

合计 ≈ **5.5 人日**（不含真机调优与视觉模型联调）。

---

## 1. 全局纪律（每刀都适用）

1. **默认关**：所有新工具不进 `cfg.Tools.Enabled` 默认值；关闭时 `Registry.Schemas()` 与现状**逐字节一致**。
2. **无图路径零字节变动**：W0 的任何改动都不得影响现有纯文本请求体（用 golden 断言锁死）。
3. **不引第三方依赖**：UIA/窗口/输入全部 `syscall.NewLazyDLL` + COM 手写（与 `internal/screen`、`secure_windows.go` 同风格）。
4. **平台隔离**：所有 Windows 实现放 `*_windows.go`，其余平台 `*_other.go` 返回 `ErrUnsupported`（对齐 `internal/screen/screen_other.go`）。
5. **不碰既有语义**：`screen_capture` 工具、`vision` 工具、审批五决策、`subjectKeys` 既有 6 键语义都不改，只做加法。
6. **一刀一提交**，提交信息带刀号；每刀必须能单独回退。

---

## 2. 每刀明细

### W0 · 多模态通道（0.5 人日）

**目标**：让"图片"能进入主对话请求，同时保证无图路径逐字节不变。

| 改点 | 文件（定位） | 动作 |
|---|---|---|
| W0-1 结构 | `internal/gaea/provider/provider.go:31`（`type Message`） | 加 `Images []string \`json:"images,omitempty"\``（值为本地绝对路径，序列化时才转 data URL，减少常驻内存） |
| W0-2 序列化 | `internal/ai/types.go:6`（`type ChatMessage`） | 加 `Images []string \`json:"-"\`` + 实现 `func (m ChatMessage) MarshalJSON() ([]byte, error)`：`len(m.Images)==0` → 走原结构体（保证逐字节不变）；否则 `content` 输出 `[{type:"text",text:…},{type:"image_url",image_url:{url:…}}]` |
| W0-3 转换 | `internal/gaea/provider/bridge/bridge.go:225`（`toChatMessages`） | `cm.Images = m.Images` |
| W0-4 工具结果带图 | `internal/gaea/agent/batch_executor.go:484`（`type toolOutcome`） | 加 `images []string` |
| W0-5 落库 | `internal/gaea/agent/agent_run.go:469`（tool 消息构造；同类还有 :435、:564） | 三处 `provider.Message{…}` 各加 `Images: resultImages[i]`；`resultImages` 与 `results` 同长同序 |
| W0-6 编码件 | 新文件 `internal/gaea/provider/imagepart.go` | `func EncodeImagePart(path string, maxEdge int) (string, error)`（长边缩放、JPEG q=70、返回 data URL）+ `func ImageTokenEstimate(w, h int) int`（保守常数） |

**验收**
1. **无图 golden**：`json.Marshal(ChatMessage{Role:"user",Content:"hi"})` 与改动前**逐字节相等**（新测试 `TestChatMessageMarshal_NoImagesUnchanged`）。
2. **有图 golden**：`Images=["x.png"]` → body 含 `"content":[{...text...},{...image_url...}]`，且 `images` 键不出现在 JSON 里。
3. 压缩摘要不吃图：`BuildCompactSummary` 对带 `Images` 的消息**不出现 base64**（现有实现只读 `Content`，测试确认即可）。
4. live 冒烟（`GAEA_LIVE_VISION_TEST=1` 才跑）：本地 herdsman 视觉模型能描述截图里的按钮文字。

**禁区**：不改 `Content` 的类型；不在无图路径改任何键序/空格；`SanitizeToolPairing` 一行不动。

---

### W1 · `computer` 包骨架（0.5 人日）

**新增文件**

```
internal/gaea/computer/
  session.go            // Default() 单例 + Options{IdleTTL} + 版次/绑定状态
  computer_other.go     // !windows：全部返回 ErrUnsupported
  errors.go             // ErrUnsupported / ErrStaleObservation / ErrScreenshotIDRequired / ErrForbiddenKey / ErrWindowGone
```

**签名草案**

```go
type Options struct{ IdleTTL time.Duration }        // 照 internal/gaea/browser/manager.go:52 的 Options/NewManager
func Default() *Session                             // 单例，IdleTTL 默认 10m
type Session struct{ /* windowID, revision, shotID, lastState, mu, lastUsed */ }
func (s *Session) Observe(windowID int, opt ObserveOpt) (*Snapshot, error)  // 每次调用 revision++
func (s *Session) CheckRevision(rev uint64) error   // 不匹配 → ErrStaleObservation
```

**验收**
1. `go build ./...` 在 windows 与 linux 交叉编译（`GOOS=linux go build ./internal/gaea/...`）都过。
2. 非 Windows：`Observe` 返回 `ErrUnsupported`。
3. TTL 单测：注入假时钟，超时后 `Session` 自动释放绑定（用例照抄 `browser/manager_test.go:567-636`）。

---

### W2 · 窗口枚举 + UIA 树渲染（1 人日）

**新增文件**：`window_windows.go`、`tree_windows.go`、夹具 `testdata/tree_*.json`

```go
type Window struct{ ID int; App string /*process:<exe 全路径>*/; Title string; X, Y, W, H int }
func ListWindows() ([]Window, error)                       // EnumWindows + DwmGetWindowAttribute + QueryFullProcessImageNameW
type Snapshot struct{ Window Window; Revision uint64; Text string; Shot *Shot }
func ObserveWindow(w Window) (*Snapshot, error)            // UIA → 文本（格式见下）
```

**树文本格式**（唯一实现处，`tree_windows.go` 一个函数产出）：
首行 `Window: "<title>", App: <displayName>.`；节点行 `\t×深度 + 序号 + 角色 + 名字 + [(状态)] + Description/Value/ID/Secondary Actions`；
长值 **75 + `... [truncated]` + 73**（合计 163）；角色取 UIA `LocalizedControlType`（**不硬编码英文表**）。

**验收**
1. golden 三断言：首行正则、节点行正则、截断几何（`TestTreeFormat_*`，输入是 `testdata/tree_*.json` 夹具，**不依赖真实屏幕**）。
2. 树 > 32 KB 时按行截断并追加 `... [accessibility tree truncated]`。
3. 真机冒烟（手动）：对记事本 / 资源管理器 / gaea 自身各取一棵树，人工核对角色与层级。

**禁区**：不引入 UIA 第三方库；不在渲染函数里做 IO。

---

### W3 · 只读三工具 + 设置项（0.5 人日）

**新增文件**：`internal/gaea/tool/builtin/computer_tools.go`（薄封装，照 `browser_tools.go` 的形态）

| 工具 | Schema 要点 | ReadOnly |
|---|---|---|
| `computer_windows` | `filter?: string`（标题/进程模糊） | `true` |
| `computer_state` | `window_id:int`、`include_text?:bool=true`、`include_screenshot?:bool`（W0 未开时忽略并回提示） | `true` |
| `computer_launch` | `app:string`（exe 路径或 `process:` 标识） | `false` |

**配置**（`internal/config/config_keys.go` 加常量，默认值取"关"）

```go
KeyComputerUseEnabled = "computer_use_enabled"      // bool，默认 false
KeyComputerUseApps    = "computer_use_apps"         // []string 白名单（process: 前缀）
KeyComputerUseIdleTTL = "computer_use_idle_ttl_sec" // int，默认 600
```

**接线**：`boot.go:339` 的 `cfg.Tools.Enabled` 不含这三件时不注册（`addBuiltins` 已按名字过滤，无需改装配逻辑）；显式开启时追加注册。

**验收**
1. 默认关：`Registry.Schemas()` 里无 `computer_*`；`boot` 装配计数与现状一致。
2. 开启后：三件出现在 schema 面，`ReadOnly()` 与表一致。
3. 非 Windows：三件注册但 `Execute` 回 `ErrUnsupported` 的信封错误（不 panic）。

---

### W4 · 按 app 授权（0.5 人日）

| 改点 | 文件（定位） | 动作 |
|---|---|---|
| W4-1 subject 扩键 | `internal/gaea/permission/permission.go:198` | `subjectKeys = []string{"app", "window", "command", …}`（**app 放最前**） |
| W4-2 强制逐条确认 | `internal/gaea/control/*`（构造 `Gate.AlwaysAsk` 处） | 把 `computer_key` 放进 `AlwaysAsk`（键盘可触发任意快捷键，必须逐条确认） |
| W4-3 前端 | `frontend/src/gaea/components/ApprovalModal.tsx:73` | `HARD_ASK_TOOLS` 追加 `computer_key`；卡片标题按 `tool` 前缀显示"应用名"（取值来自 `approval.subject`，**无需改数据结构**） |

**为什么零结构改动**：`event.Approval.Subject`（`event.go:195`）已经是 `permission.Subject(args)` 的结果；`app` 进 `subjectKeys` 后，审批卡的 subject 天然就是 `process:C:\…\notepad.exe`。
「始终允许」落盘走既有 `rememberRule`（`permission.go:416`），格式自动是 `computer_click(process:C:\…\notepad.exe)`。

**验收**
1. `Policy{Allow:[{Tool:"computer_click", Subject:"process:C:\\…\\notepad.exe"}]}` 对同 app 放行，对不同 app 仍 `Ask`。
2. `permLevel=never`（`ApprovalMode`）下 → `Deny` 且理由可读（既有 `NormalizeAsk` 路径）。
3. 审批卡：`computer_click` 卡片 subject 显示进程路径（真机手验）。

---

### W5 · 输入注入（1.5 人日）

**新增文件**：`input_windows.go`、`guard_windows.go`

```go
func Click(w Window, p Point, button string, count int) error   // 坐标已换算为屏幕绝对坐标
func ClickElement(w Window, index int) error                    // 优先 UIA Invoke/SelectionItem，失败才落坐标
func TypeText(w Window, text string) error                      // SendInput KEYEVENTF_UNICODE
func PressKey(w Window, chord string) error                     // keysym 名 + 别名映射表
func Scroll(w Window, p Point, dx, dy int) error
```

**三条硬约束（缺一不可）**
1. **双闸门**：`Click/Scroll/Drag` 前校验 `Session.revision` 与坐标动作的 `shotID`；不匹配直接 `ErrStaleObservation`，**不得落到 SendInput**。
2. **DPI**：操作前 `SetThreadDpiAwarenessContext`，`defer` 复原；复原失败 → 置 `coordinatesDisabled=true`，此后所有坐标动作返回明确错误（语义照抄 asset-62 §6）。
3. **deny 表**（本地硬拒，先于任何策略）：
   - 键名含 `Win`/`Meta`/`Cmd`/`Super`/`OS` → `ErrForbiddenKey`；
   - 目标窗口标题/进程命中 终端、认证框、密码管理器、安全软件 → 拒绝并说明原因。

**验收**
1. 旧 `element_index` → `ErrStaleObservation`，**且注入一个 fake sender 断言 0 次调用**。
2. 记事本真机：`Click`（元素）→ `TypeText` → `PressKey("Return")`，动作返回体附带新树（顺序见 W3 的 `computer_state`）。
3. `PressKey("Win+r")` → `ErrForbiddenKey`，无 syscall。
4. 多显示器/125% 缩放：点击落点误差 ≤ 2px（手验）；DPI 复原失败用例用注入错误模拟。

---

### W6 · 状态条 + 截图给人看（0.5 人日）

- 事件：`core.emit("computer-status", {state:"using|idle", app, window, turnID})`（`internal/app/app.go:317` 现成通道，网页端 `httpbridge` 也能收）。
- 截图对人：落 `.gaea/uploads/computer-<ts>.jpg`（照 `screenshot.go:106` 的 `saveScreenshot` 范式），在工具返回体里给 markdown 图片链接；**不走事件传 base64**。
- 前端：状态条挂在现有会话头/输入框上方，复用 `notice` 的样式；设置页新增 `computer_use` 分组（开关 / app 白名单 / 空闲 TTL）。

**验收**
1. 事件载荷字段齐全（单测断言 map 的键）。
2. 关闭开关后无事件发射、无文件落盘。

---

### W7 · 门禁（0.5 人日）

```powershell
# 后端
go build ./...
go vet ./internal/gaea/...
go test ./internal/gaea/computer/... ./internal/gaea/permission/... ./internal/gaea/tool/... -count=1
go test ./internal/ai/... ./internal/gaea/provider/... -count=1      # W0 golden
# 前端
cd frontend; npm.cmd run build
# 冒烟（构建指南要求的正式路径）
scripts\smoke.ps1 -ExePath build\bin\gaea.exe                        # /api/health 200
# 文档守卫
node scripts\check-docs.mjs
```

**提交纪律**：W0~W7 各自独立提交；W0 可与 W1 并行；W3 之前的任何提交都不得让 `computer_*` 出现在默认 schema 面。

---

## 3. 回退矩阵

| 刀 | 回退方式 | 回退后行为 |
|---|---|---|
| W0 | `Message.Images` 不赋值 | 请求体与现状逐字节一致（golden 保证） |
| W1~W6 | `computer_use_enabled=false`（默认） | 三件工具不注册，`Schemas()` 无新增，无事件、无落盘 |
| W4 | `subjectKeys` 去掉 `app` | 审批退回按工具名授权（既有语义） |
| W5 | 只注册 W3 三件只读工具 | 无输入能力，纯观察 |

---

## 4. 风险与红线

| 风险 | 处置 |
|---|---|
| 误点误输（OS 级副作用） | 工具默认关 + `computer_key` 进 `AlwaysAsk` + deny 表 + 双闸门 + 动作后强制回树 |
| 与 `screen_capture` / `vision` 语义重叠 | 三者在描述里互相指路；`computer_state` 只服务"主模型决策"，不复用 `screen_capture` 的落盘语义 |
| 上下文爆炸 | 树 32 KB 行级截断 + 节点值 163 字符；图片每回合上限 4 张、长边 ≤1280 |
| DPI/多屏坐标漂移 | 坐标系一律"窗口相对逻辑像素"，注入前统一换算；复原失败即禁用坐标 |
| 锁屏/会话切换 | 观察失败即停并提示解锁；不尝试绕过 `LockApp` |
| 规划漂移 | 本单的 file:line 会漂；施工前按符号名重新定位，禁止按行号盲改 |

---

## 5. 明确不在本单范围

JS REPL / `cua` 高层对象 / native pipe 双传输 / 自绘 overlay 浮窗 / 音频录制 / cloud computer /
多区域 `screenshotID` / 树 diff / 键盘 Esc 全局钩子（P2 再议）。

---

## 6. 施工前置确认（需拍板）

1. **W0 先行**：是否同意先花 0.5 人日打通多模态，再开 W1？（否则 W5 的坐标点击价值大打折扣）
2. **试点板块**：先在 `office` 板块露这三件工具，还是聊天主线程直接开？
3. **图片上限**：每回合 4 张、长边 1280 是否照此执行？（token 成本与识别率的折中）
