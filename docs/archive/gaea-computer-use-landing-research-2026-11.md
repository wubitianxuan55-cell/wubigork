# gaea 落地「电脑操控」调研（2026-11-19）

> 性质：**只调研，不改代码**。上游对照物 = `asset-62`（协议规格）/ `asset-63`（代码级还原），证据包 `C:\AI\re\codex-computer-use\`。
> 本篇回答三件事：**gaea 现在缺什么、按什么路线落、每一刀的验收是什么**。

---

## 0. 结论卡

| 问题 | 结论 |
|---|---|
| 能不能落 | **能，且可以走 codex 同款"截图给主模型"路线**——模型本身是多模态的，gaea 也已经会发多模态请求（`vision.go:199-204` 拼 `image_url` + base64 data URL），缺的只是**主对话消息结构没带图**（1 处结构 + 1 处转换，见 §2.7） |
| 三处"缺"（不是"不行"） | ① 主对话通道：`provider.Message.Content` / `ai.ChatMessage.Content` 都是 `string`（`provider.go:31`、`ai/types.go:8`），**改造成本约 0.5 人日**；② 全仓**无输入注入**（无 `SendInput`/`SetCursorPos`/UIA），只有全屏截屏（`internal/screen`）——这块才是真工作量；③ 审批是 **subject-glob** 模型（`permission.go:198` 的 `subjectKeys` 无 `app`），没有 codex 那种「按 app 授权」（加 1 个 key 即可） |
| 最短可用路径 | 新增 `internal/gaea/computer` 包 + **4~6 个工具**（不是 codex 的 17 个方法）+ 把 `app` 加进 `subjectKeys` 复用现成 `Gate`/`ApprovalModal` + 会话 TTL 照抄 `browser.Manager` |
| 抄什么 | ① 索引/坐标**一次性**闸门 ② 按 app 授权 + 持久化档位 ③ 树文本文法与 75/73/163 截断 ④ Esc 中断语义 ⑤ DPI 复原失败即禁用坐标 |
| 明确不抄 | JS REPL / `cua` 高层对象 / native pipe 双传输 / 自绘 overlay 浮窗 / 音频录制 / cloud computer |
| 试点建议 | 先 `office`（办公）板块，`permLevel=ask` + 工具**默认关**，只在显式开启后可观察、可点击 |
| 预估 | P0 只读 1~1.5 人日；P1 输入 2~3 人日；P2 完善（per-window 抓屏/多区域）2~4 人日 |

**最容易想到的偷懒路径要直接否掉**：把 codex 的 `codex-computer-use.exe` / `codex-computer-use-swift.exe` 打进 gaea 当后端。
技术上可行（协议已被我们摸清：行 JSON + `--parent-pid` + `meta['x-oai-cua-approved-app']`），但它是 OpenAI 的私有产物
（插件许可 `Proprietary`）、版本随 codex 桌面端漂移、审批环节还依赖 codex 的 app-server；一旦上游升级，gaea 会**整体失效且无法自行修**。
只可作为**对照验证器**（拿它跑同一窗口比对我方树文本），不可作为运行时依赖。

---

## 1. gaea 现状盘点（可复用件，file:line）

| 能力 | 现状 | 位置 | 复用度 |
|---|---|---|---|
| 截屏（全屏，GDI BitBlt） | ✅ 有，含「成功但全黑」的防呆 | `internal/screen/screen_windows.go:15-26`（`user32/gdi32` 懒加载）、`:186-196`（`SelectObject`/`BitBlt` 返回码检查） | **高**（P0 直接用） |
| 截图落盘 + 给模型看 | ✅ 有（存 `.gaea/uploads/screenshot-*.png`，再调 `vision`） | `tool/builtin/screenshot.go:60,106`；`tool/builtin/vision.go:20` | **高**（P0 直接复用范式） |
| 工具抽象 | ✅ `Name/Description/Schema/Execute/ReadOnly` + 6 个可选能力接口 | `internal/gaea/tool/tool.go:18-136` | 高 |
| 工具注册 | ✅ `RegisterBuiltin`（init）+ boot 装配 + 隐藏/预算 | `tool.go:144`；`boot.go:1089`（`addBuiltins`）、`:1237-1240`（`HideUnlessOnly`）、`:689`（`SetSchemaBudget`） | 高 |
| 执行闸门 | ✅ 单一收口：dispatcher.Check → gate.Check → 路径访问 → 预检 → 执行 | `internal/gaea/agent/execute_one.go:61-134` | **高**（新工具自动继承） |
| 权限策略 | ✅ `Policy{Allow/Ask/Deny/ArgvAllow}` + `Gate.Check` + `Approver` + `AlwaysAsk` + `ApprovalMode=never` + `DenyUnapproved` | `permission/permission.go:92-155,255-411` | **高**（改 1 处即可支持按 app） |
| subject 提取 | ⚠️ 只认 `command/file_path/path/pattern/source/url` | `permission.go:198` | **需扩 `app`/`window`** |
| 审批 UI | ✅ 5 决策（deny/allow_once/allow_session/persist_allow/abort）+ 键盘 + hardAsk 三钮 | `frontend/src/gaea/components/ApprovalModal.tsx:37-80` | 高（加个 app 分支即可） |
| 事件到前端 | ✅ `core.emit` → Wails `EventsEmit` + `httpbridge.Publish`（网页端也能收） | `internal/app/app.go:317-329` | 高 |
| 会话单例 + 空闲回收 | ✅ `browser.Manager`（`Options.IdleTTL`，0=禁用） | `internal/gaea/browser/manager.go`、`manager_test.go:567-636` | **高（照抄范式）** |
| 权限级别 | ✅ `ask / auto / yolo` 三档 + 回合内禁改 | `control/controller.go:173-175,1033`；`controller_step.go:41-61` | 高 |
| 图像消息（主对话） | ⚠️ **结构未带图**：`Message.Content string`（`provider.go:33`）、`ai.ChatMessage.Content string`（`ai/types.go:8`）；但模型是多模态的 | `provider/provider.go:31-44`、`ai/types.go:6-12` | **中（改 3 处，见 §2.7）** |
| 多模态请求能力（旁路） | ✅ **已有**：`vision` 包自己拼 `content:[{type:text},{type:image_url}]` + base64 data URL | `internal/gaea/vision/vision.go:104-105,199-204,256-263` | **高（照抄即可）** |
| 用户贴图 | ✅ 前端可粘贴图片落盘（`GaeaSavePastedImage`），但落盘后交给识图工具，不进主对话 | `frontend/src/**/image.ts:21,278-282` | 中（同一改造一并解决） |
| 窗口枚举 / 激活 | ❌ 无 | — | 0（新写） |
| UIA 无障碍树 | ❌ 无 | — | 0（新写，最大工作量） |
| 输入注入 | ❌ 无（`SysProcAttr.HideWindow` 只是隐藏控制台，不是输入） | `tool/builtin/hide_window_windows.go` | 0（新写） |

---

## 2. 落地架构（包边界 + 工具面）

### 2.1 新包 `internal/gaea/computer`

```
internal/gaea/computer/
  computer_other.go      // 非 Windows：所有能力返回 ErrUnsupported（对齐 internal/screen/screen_other.go 范式）
  session.go             // Default() 单例 + IdleTTL + 会话状态（窗口绑定/revision/截图ID/上次观察）
  window_windows.go      // EnumWindows + DwmGetWindowAttribute + 进程/身份（对齐 codex 的 Window{app,id,title,frame}）
  tree_windows.go        // UIA：CreateTreeWalker/RawViewWalker → 文本树（格式见 §2.3）
  capture_windows.go     // P0 复用 internal/screen；P2 换 Windows.Graphics.Capture（遮挡也抓）
  input_windows.go       // SendInput（鼠标绝对坐标/键盘 Unicode/按键映射）+ DPI 上下文切换与复原
  guard_windows.go       // 默认 deny 表（终端/认证/安全软件/Win 键）+ Esc 低级键盘钩子
  model.go               // 与 §2.3 文法绑定的事件/快照结构（便于单测）
```

**为什么单独成包而不是塞进 `builtin`**：① 需要跨工具共享会话状态（光标/索引/截图缓存）；② 有平台文件与大量 Windows syscall；
③ `browser` 包已验证这个形态（包内 `Manager` + `builtin/browser_tools.go` 薄封装）。

### 2.2 工具面（收敛版：4 + 3）

| 阶段 | 工具 | 参数要点 | 只读 |
|---|---|---|---|
| P0 | `computer_windows` | `filter?`（按标题/进程名模糊） | ✅ true |
| P0 | `computer_state` | `window_id`, `include{text,screenshot}`（P0 阶段 `text=true, screenshot=false`——多模态未通前图只给人看；P1.5 打通后 screenshot 默认真） | ✅ true |
| P0 | `computer_launch` | `app`(exe 路径或已知 id) | ❌ |
| P1 | `computer_click` | `window_id`, `element_index`\|`x,y`, `button?`, `count?` | ❌ |
| P1 | `computer_type` | `window_id`, `text`（走 `SendInput` Unicode，不走剪贴板） | ❌ |
| P1 | `computer_key` | `window_id`, `key`（keysym 名 + 别名，见 asset-62 §7） | ❌ |
| P2 | `computer_scroll` / `computer_drag` | 坐标 + 方向/端点 | ❌ |

**不做 17 个方法**：codex 的细粒度方法面是为了让模型在 JS 里自由编排；gaea 的工具面有 schema 字节预算（`boot.go:689`、
`asset-11`/`asset-50` 的结论），且模型看到的是"一次调用一个动作"。**动作间必须重观察**这条纪律靠工具**语义强制**：
`computer_click/type/key` 执行完直接把「新树」塞进返回体（省一次往返，同时天然保证"动作后必须刷新"）。

### 2.3 树文本格式（与 codex 对齐，gaea 侧唯一定义处）

直接照 `asset-62` §4：首行 `Window: "<title>", App: <displayName>.`；节点行 `\t×深度 + 序号 + 角色 + 名字 + [(状态)] + Description/Value/ID/Secondary Actions`；
长文本 **75 + `... [truncated]` + 73**；角色取 UIA `LocalizedControlType`（中文系统自然出中文，**不要硬编码英文表**）。
落地位置：`tree_windows.go` 的单一渲染函数 + 一个 golden 测试（用固定 UIA 快照夹具，不看真实屏幕）。

### 2.4 会话状态机（照抄 codex 的两条闸门）

```go
type Session struct {
    windowID  int
    revision  uint64          // 每次 get_state 自增；动作前比对
    shotID    string          // 最近一次截图 id（P2 多区域用）
    lastState *Snapshot
}
// 闸门 1（索引一次性）：动作携带的 element_index 只在 revision 匹配时有效，否则回
//   "window changed; call computer_state before using an element index"（文案自写，语义照抄）
// 闸门 2（坐标一次性）：坐标动作必须带本次截图的 shotID
```
TTL 复用 `browser.Manager` 的 `IdleTTL` 范式（`manager_test.go:567-636` 的用例可直接改造成 `computer` 的用例）。

### 2.5 审批（改一处即可按 app 授权）

```go
// permission.go:198
var subjectKeys = []string{"app", "window", "command", "file_path", "path", "pattern", "source", "url"}
```
把 `app` 放**最前**（computer 工具的 args 里 `app` 唯一稳定标识），这样：
- 用户勾「始终允许」→ 落盘 `computer_click(process:C:\…\notepad.exe)`（`rememberRule` 天然产出这个格式）；
- 会话级放行 → 复用 `SessionArgv`/会话规则（同构，换个容器即可）；
- 高危动作（`computer_key` 含 `Win`/`Meta`/`Cmd`/`Super`/`OS`）→ 走 `Gate.AlwaysAsk` + 本地硬拒（对齐 asset-62 §8.3 的 12 条 deny）。

`ApprovalModal` 只需加一个「app」角标分支（`approval.tool` 前缀 `computer_` 时显示 `args.app` 的进程名），复用现有 5 决策。

### 2.6 事件与前端

- **状态条**：`core.emit("computer-status", {state:"using|idle", app, window, turnId})`（`app.go:317` 路径现成，网页端也能收）。
- **截图给用户看**：落盘 + 走现有 markdown 图片渲染（`screenshot.go:106` 的 `.gaea/uploads/` 范式），**不走事件传 base64**。
- **设置页**：新增 `computer_use` 分组（默认关、默认 `ask`、白名单 app 编辑），复用 `SettingsSection.tsx`。
- **不做自绘浮窗**：codex 的 "ChatGPT is using your computer / Esc to cancel" 是它自己画的 Win32 浮窗；gaea 用主窗口状态条 + `notice`（`controller_submit.go:192` 同款）即可，省掉一整套 D2D/DWrite/overlay 生命周期。

---

### 2.7 打通"截图给主模型"（多模态通道）

**先澄清一个事实**：gaea 的模型是多模态的，而且**代码里已经有能发图的请求实现**——`vision` 包直接拼
`content:[{type:"text"},{type:"image_url",image_url:{url:"data:image/png;base64,…"}}]`
（`vision.go:199-204`），只是这条通道**封在 vision 内部**，主对话链路没接。

主对话的瓶颈只有一处：消息结构不带图。

```go
// provider/provider.go:31
type Message struct {
    Role    Role   `json:"role"`
    Content string `json:"content,omitempty"`   // ← 只有文本
    ...
}
// ai/types.go:6
type ChatMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`             // ← 只有文本
    ...
}
```

**三个改点（不改 Content 的类型，改最小）**：

| # | 改点 | 位置 | 做法 |
|---|---|---|---|
| 1 | 给消息加图片旁路字段 | `provider.Message` | 加 `Images []string json:"images,omitempty"`（data URL 或本地路径，落盘路径更省内存） |
| 2 | 序列化成 OpenAI 多模态 parts | `ai.ChatMessage` | 加 `Images []string json:"-"` + 自定义 `MarshalJSON`：有图 → `content:[{type:text},{type:image_url},…]`；无图 → 保持原字符串（**逐字节不变**） |
| 3 | 转换透传 | `bridge.go:225 toChatMessages` | `cm.Images = m.Images` |

**为什么这是最小改法**：
- 全仓 `provider.Message{...}` 构造点 **237 处**都在造 `Content`，加可选字段**零改动**；
- `ai.ChatMessage{...}` 构造点 **27 处**同理；
- 读取 `Content` 的地方只有响应解析（`client_stream.go` 4 处）+ 测试，**请求侧没人读**，所以不用改类型；
- 压缩摘要 `BuildCompactSummary`（`compact_summary.go:32,35,202`）只读 `Content` → **天然忽略图片**，不用写剔除逻辑；
- 会话存储走 JSON（`omitempty`），旧会话回放不受影响。

**必须配套的两件事**（不做就会翻车）：
1. **缩图**：1920×1032 的 JPEG（实测 180 KB）直接进上下文，单张按 OpenAI/Anthropic 的 patch 换算约 1.1~1.5k tokens；
   发之前统一缩到长边 ≤1280、JPEG q≈70，并把"这张图来自哪次观察"写进文本 part（否则模型分不清新旧帧）。
2. **token 预算**：图片 token 要么估个保守常数计入预算，要么在 `MaxTokens` 之外单独设"每回合最多 N 张图"（建议 4）。

**成本**：结构+转换+序列化 ≈ 150 行；缩图 + 预算 ≈ 80 行；golden 测试（有图/无图两种 JSON 逐字节断言）≈ 60 行。
约 **0.5 人日**，风险在"是否影响现有纯文本请求的字节稳定性"——用无图路径的 golden 对比即可锁死。

---

## 3. 与 codex 逐项对照（抄 / 改 / 不抄）

| 机制 | codex 做法 | gaea 落法 | 判定 |
|---|---|---|---|
| 模型面 | `node_repl` JS + `@oai/sky` 17 方法 | 4~7 个 Go 工具，动作后自动回树 | **改** |
| 看图 | JPEG 作为 image content 发给多模态主模型 | **同款**：打通消息图片旁路（§2.7，0.5 人日）后，截图作为 `image_url` 直接进主对话；未打通前退化为 `vision` 转文本 | **改（成本可控）** |
| 元素索引 | `element_index` + `accessibilityRevision` | 同（`session.revision`） | **抄** |
| 坐标 | `screenshotId` + 多 region | P0 单窗口单区域；P2 再上多区域 | **抄（分期）** |
| 授权 | 按 app + `approvalRequest{app,displayName,riskLevel}` + 持久化 `session/always` | `subjectKeys` 加 `app`，复用 Policy 的 `allow_once/allow_session/persist_allow` | **抄语义，改载体** |
| 截断 | 75 / marker / 73 | 同（含 golden 测试） | **抄** |
| Esc 中断 | 低级键盘钩子 → 退出码 130 → 回合级中断标记 | P1：先做 UI 中断（复用现有 abort）；P2 再上键盘钩子 | **改（分期）** |
| DPI | 切线程上下文 + 复原失败则禁用坐标 | 同 | **抄** |
| 遮挡抓屏 | Windows.Graphics.Capture | P0 GDI 已够；P2 视需要 | **分期** |
| overlay 浮窗 | 自绘 Win32+D2D | 不做（用状态条） | **不抄** |
| JS REPL / native pipe / 音频 / cloud computer | 有 | 不做 | **不抄** |

---

## 4. 三阶段落地与验收

### P0 · 只读观察（1~1.5 人日）
- 交付：`computer` 包（session/window/tree/capture）+ 3 个只读工具 + 设置项（默认关）。
- 验收：
  1. `computer_windows` 能列出当前可见顶层窗口（含 `app` 进程标识、frame、title）；
  2. `computer_state` 对记事本/资源管理器/gaea 自身返回树文本，且**首行/节点行/截断几何**三条断言通过（§6.1）；
  3. 非 Windows 平台编译通过且返回 `ErrUnsupported`（`go build ./...` 交叉验证不破）；
  4. 树文本 > 32 KB 时按行截断并标注（防止上下文爆掉，参考 `asset-48`）。

### P0.5 · 多模态通道（0.5 人日，可与其他阶段并行）
- 交付：§2.7 的三处改点 + 缩图/张数预算。
- 验收：
  1. **无图路径 golden**：现有纯文本请求序列化结果逐字节不变（防止回归）；
  2. **有图路径 golden**：`Images=[x]` 时 body 为 `content:[{type:"text"},{type:"image_url",image_url:{url:"data:image/…"}}]`；
  3. 真实多模态模型（本地 herdsman 视觉 / 云端任选一个）能准确描述一张截图里的按钮文字（跑一次 live 冒烟，默认 skip）；
  4. 压缩摘要、`SanitizeToolPairing`、会话回放对新字段**零感知**（不 panic、不把 base64 写进摘要）。

### P1 · 低风险输入（2~3 人日）
- 交付：`computer_click/type/key` + `app` subject + 坐标/索引双闸门 + 默认 deny 表 + UI 中断。
- 验收：
  1. 记事本：点击 + 打字 + `Enter`，动作后返回体自带新树且序号刷新；
  2. 用旧 `element_index` 再点 → 必须回"需要重新观察"类错误（不落到系统调用）；
  3. `Win`/`Meta`/`Cmd` 组合键 → 本地拒绝；`permLevel=yolo` 下仍逐条确认（`AlwaysAsk`）；
  4. 点击落在**非目标窗口**上时报告并提示 `activate`（对齐 asset-62 §6 的"点在别人窗口上"处理）；
  5. 截屏/输入过程中的 DPI 复原失败 → 坐标动作整体禁用并告知原因。

### P2 · 完善（2~4 人日，按需）
- per-window 抓屏（遮挡可用）、多区域 `shotID`、键盘 Esc 钩子、`scroll/drag`、树 diff（对齐 codex 的"无变化不重发"）。

---

## 5. 风险与红线

| 风险 | 处置 |
|---|---|
| 误点/误输（最危险） | 工具默认关 + `AlwaysAsk` + 默认 deny（终端、认证框、密码管理器、安全软件、Win 键）+ 每次动作后强制回树 |
| 与现有 `screen_capture` 语义重叠 | `screen_capture` 保留为"给视觉模型看的图"；`computer_state` 是"给主模型的树"。两者在描述里互相指路，**不合并** |
| 上下文爆炸 | 树截断（32 KB 行级 + 节点级 163 字符），状态默认不带图 |
| 锁屏/会话切换 | 检测失败即停（`OpenInputDesktop` 失败 = 锁屏），要求用户解锁后再继续（对齐 asset-62 §8 恢复段） |
| 多模态通道不通时的退化 | 若暂不做 §2.7：截图只能靠 `vision` 转文字描述 → 坐标点击精度掉一个档（图标按钮、canvas、无树界面基本不可用）。**明确写进产品口径**：未打通期间 gaea 的电脑操控是"无障碍优先"，纯图形界面软件不在支持范围 |
| 上游升级失效 | 不依赖 codex 二进制；`asset-63` 的还原仅作规格与对照 |

---

## 6. 验收断言（可直接写成 Go 单测）

```go
// 6.1 树文法（golden，不碰真实屏幕）
regexp.MustCompile(`^Window: ".*", App: .+\.$`)                       // 首行
regexp.MustCompile(`^\t*\d+ \S+( \(.*\))?( .*)?$`)                    // 节点行
// 长值截断：head 75 + "... [truncated]" + tail 73 == 163

// 6.2 闸门
// 旧 revision 的 element_index → errors.Is(err, ErrStaleObservation)
// 坐标动作缺 shotID（多区域时）→ ErrScreenshotIDRequired

// 6.3 授权
// permission.Policy{Allow: []Rule{{Tool:"computer_click", Subject:"process:C:\\…\\notepad.exe"}}} 命中放行
// 未授权 app → Ask；permLevel=never → Deny 且理由可读

// 6.4 红线
// key 含 Win/Meta/Cmd/Super/OS → ErrForbiddenKey（本地拒绝，不产生 syscall）
```

---

## 7. 待拍板（落地前需要定 5 件事）

1. **路线**：先打通多模态（§2.7，0.5 人日）再上电脑操控，还是先树文本跑通再补？（推荐前者：打通后 P1 的坐标点击才有意义）
2. **工具粒度**：4 个（`windows/state/click/type`）够不够，还是按 codex 拆 7 个？
3. **试点板块**：办公（`office`）单板块试点，还是聊天主线程直接开（影响默认关的开关位置）？
4. **输入注入默认态**：默认关 + `ask`；`auto` 下是否允许输入（建议仍 `AlwaysAsk`）。
5. **审批载体**：复用 `ApprovalModal` 加 app 角标，还是新做一个 `AppApprovalCard`？

---

## 8. 复现入口（调研证据）

```powershell
# codex 侧协议与原文（对照用）
Get-Content C:\AI\re\codex-computer-use\data\INDEX.md
Get-Content C:\AI\re\codex-computer-use\data\api-surface.json
# gaea 侧现状
Select-String -Path internal\gaea\tool\tool.go -Pattern 'type Tool interface' -Context 0,20
Select-String -Path internal\gaea\permission\permission.go -Pattern 'subjectKeys|func \(g \*Gate\) Check'
Select-String -Path internal\screen\screen_windows.go -Pattern 'procBitBlt|GetSystemMetrics'
Select-String -Path internal\gaea\agent\execute_one.go -Pattern 'dispatcher.Check|gate.Load'
```
