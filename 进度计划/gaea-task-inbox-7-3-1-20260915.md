# gaea 7.3-1 多入口统一任务收件箱 · 实施规格（v4.318.0）

> 来源：`docs/gaea-stage7-plan-2026-09.md` §3 7.3-1（7.1/7.2 四刀已收官，形态层首刀）。
> 刀法：契约先行 → A/B/C 三线并行（足迹互斥）→ 主代理收口（gen_bindings / 门禁 / 发版）。
> 红线：纯同步按需（无常驻、无轮询、无定时）；任务实体 space 维度必带
> （work/play 隔离沿 boards/space.ts，跨空间仅显式）；不加新板块（收件箱挂
> 既有首页两空间 + 既有入口，不动 pageRegistry/manifest 结构）；不重写不换框架。

## 0. 论点与范围

意图中枢/微信/语音/Ctrl+K 四入口已汇同一内核（v4.5 起在册：intent.Parse →
routeIntent 能力执行层）。本刀补「任务」为持久实体与统一收件箱视图：**四入口的
指令除即时执行外可「存为任务」→ 状态机追踪（待处理/进行中/已完成/已放弃）→
任务详情跳回来源板块与源会话**。任务只在被打开/被操作时读写（零常驻判据）。

范围裁决（V1）：
- **任务=用户意图的持久卡**（title + 来源 + space + 状态机），不是后台执行队列
  ——后台任务表（GaeaTaskList/tasks 包）与会话待办（todo_write）语义不变、零改动。
- **不自动执行任务**：收件箱是清单不是调度器（昼夜运转拍板）；「去板块」是
  导航不是触发。
- 源会话跳转粒度：V1 落库 Session 字段（审计链），首页面板跳转=来源板块级；
  精确回源会话若 gaea 工作台有现成 seam 则接（见 §3.4），否则欠账如实入档。
- 不做跨空间移动/复制（跨门仅显式=本刀不做，只做隔离）。

## 1. 线 A · 纯函数包 `internal/taskinbox`（零 IO，先例 skilldistill/routesuggest）

```go
package taskinbox // import "github.com/gaea/gaea/internal/taskinbox"

const (
    MaxTitleRunes = 120 // 标题上限（超长截断不报错——存任务不该被长度卡死）
    MaxTasks      = 500 // 状态文件条目上限（防膨胀；超限 Save 报错）
)

type Status string // pending | doing | done | abandoned

type Task struct {
    ID        string `json:"id"`        // "ti-"+12hex（crypto/rand，实体非重算候选）
    Title     string `json:"title"`     // NormalizeTitle 产物
    Space     string `json:"space"`     // "work"|"play"（唯一合法值）
    Status    Status `json:"status"`
    Source    string `json:"source"`    // ctrlk|palette|voice|weixin|inbox
    Action    string `json:"action,omitempty"` // 意图动作（navigate/…；手动空）
    Target    string `json:"target,omitempty"` // 意图目标（板块 id 等）
    Session   string `json:"session,omitempty"`
    Note      string `json:"note,omitempty"`
    CreatedAt int64  `json:"createdAt"`
    UpdatedAt int64  `json:"updatedAt"`
}

// ParseStatus 归一；CanTransition 状态机：
// pending→doing→done；pending|doing→abandoned；终态（done/abandoned）不接受
// 任何迁移；同值迁移（无变化）恒 false（由调用方短路）。
func ParseStatus(s string) (Status, bool)
func CanTransition(from, to Status) bool

// NormalizeTitle：trim；空 → error；>MaxTitleRunes rune 截断。
func NormalizeTitle(s string) (string, error)
// ValidSpace：仅 work|play。
func ValidSpace(s string) bool
// ValidSource：仅 ctrlk|palette|voice|weixin|inbox。
func ValidSource(s string) bool
// ParseTaskID 残渣防御（"ti-"+12hex；先例 ParseSuggestionID）。
func ParseTaskID(id string) bool
// FilterBySpace：space "" = 全部（GaeaTaskList 变参先例口径）。
func FilterBySpace(tasks []Task, space string) []Task
// Sort：状态组序 pending<doing<done<abandoned，组内 UpdatedAt 降序，
// 再 ID 升序稳定（收件箱=行动优先：待办在前、最近变化优先）。
func Sort(tasks []Task) []Task
```

测试（表驱动）：状态机全矩阵（合法 3 条 + 全非法含终态/同值）/ ParseStatus
未知值 / NormalizeTitle 空/纯空白/超长截断/正常 / ValidSpace·ValidSource 边界 /
ParseTaskID 残渣 / FilterBySpace 含 '' 全量 / Sort 组序+组内时间+稳定 / JSON
标签钉死（camelCase，与状态文件契约一致）。

## 2. 线 B · 状态文件 + 执行层接线 `internal/app/gaea_task_inbox.go` + intent 扩展

### 2.1 绑定（绑定面 691→695，OfficeB +4）

```go
type TaskInboxView struct { /* 与 taskinbox.Task 同字段同 json 标签 */ }

func (a *App) GaeaTaskInboxList(space string) []TaskInboxView
    // 读状态文件 → taskinbox.FilterBySpace+Sort → 视图；文件缺失/损坏=空表。
func (a *App) GaeaTaskInboxSave(reqJSON string) (TaskInboxView, error)
    // reqJSON={id?,title,space,source?,action?,target?,session?,note?}
    // id 空=新建（status=pending，CreatedAt/UpdatedAt=now，ID=ti-+12hex）；
    // id 非空=更新 title/note（Source/Action/Target/Session/CreatedAt 不可变
    // ——来源审计链）；全部经纯包校验；超 MaxTasks 拒绝新建。
func (a *App) GaeaTaskInboxSetStatus(id, status string) (TaskInboxView, error)
    // ParseStatus + CanTransition（同值短路成功）；UpdatedAt=now。
func (a *App) GaeaTaskInboxDelete(id string) error
```

- 状态文件 `<DataRoot>/task_inbox.json`：`{version:1, tasks:[…]}` camelCase，
  route_suggestions 同款容错读（缺失/损坏回空表）+ 原子写（temp+rename）。
- 门面：bindings_office.go OfficeB +4 纯委托。**不跑 gen_bindings**（主代理收口统一跑）。
- 事件：不 emit（零常驻；前端在每次变更后重拉——清单量级 ≤500，读即所得）。

### 2.2 intent 扩展 `internal/intent`：ActionSaveTask

```go
ActionSaveTask Action = "save_task" // 存为任务（Target = 任务标题文本）
```

规则（首尾锚定 + 宁漏勿误；不命中样例入测试）：
`^(?:请|麻烦|帮我)?(?:存为|存个|记个|记一条|添加|新建)(?:一个|一条|个)?任务[:：]?\s*(.{1,120})$`
`|^(?:任务|待办)[:：]\s*(.{1,120})$`
- 不命中：「这个任务不错」（无动词锚）、「打开任务中心」（导航动词在前）、「存为任务」
  （空标题）、「提醒我存个任务」（提醒让位，与 reWxReminderish 同款守卫——save_task
  检查排在 reminder 之后）。
- Parse 顺序：save_task 排在 reminder 之后、其他动作之前/后均可（短语互斥，测试钉死）。

### 2.3 执行层 `internal/app/intent_router.go`：save_task case

- `case intent.ActionSaveTask: reply, ok := a.execSaveTask(it)`：
  space=`gaeaEffectiveSpace()`（后端入口无前端空间上下文——语音/微信的诚实取法）；
  source：routeIntentModeForAssistant 的 assistantID 非空 → "weixin"，否则 "voice"；
  title=it.Target 经 NormalizeTitle；落库走 §2.1 同一 Save 内核；reply=
  「已存入任务收件箱：<title>」（语音 TTS/微信回推天然可用）。
- **Ctrl+K/命令面板不走此执行路径落库**：dry-run 预览照常命中（action=save_task），
  前端点「执行/存为任务」直接调 GaeaTaskInboxSave（source=ctrlk|palette、
  space=前端当前空间——见 §3）。两路落同一状态文件，来源可区分（审计判据）。

### 2.4 测试（internal/app/gaea_task_inbox_test.go + intent 增例）

- 状态文件往返（新建→列表→状态迁移→删除；损坏文件回空表；camelCase 形状钉死）。
- Save 新建/编辑（id 非空只改 title/note，来源字段不可变）/非法 space·source·
  空标题拒绝/超上限拒绝。
- SetStatus 合法迁移与非法迁移拒绝（纯包矩阵的集成面）。
- intent.Parse save_task 命中/不命中表（含提醒让位）。
- execSaveTask：space 取 gaeaEffectiveSpace + assistantID 判 source（voice/weixin
  双例）；GaeaRouteIntent("存个任务 X", true) 预览 Handled=true 且零落盘。
- 只跑定向：`go test ./internal/taskinbox ./internal/intent ./internal/app -run
  'TestTaskInbox|TestIntent.*SaveTask|TestSaveTask' -count=1`；**不跑全量、不跑
  ci.ps1、不 commit**。

## 3. 线 C · 前端（收件箱视图 + 四入口接线）

### 3.1 `TaskInboxPanel.tsx`（frontend/src/gaea/components/，+test）

- props：`{open, onClose, space: ShellSpace, onNavigate?: (boardId: string) => void}`；
  antd Modal + antd 组件，风格对齐 TaskCenter/MemoryPanel（v3-panel-head 细条头）。
- open 时拉 `GaeaTaskInboxList(space)`（经 bridge app.*），每次变更（新建/迁移/
  删除）后重拉；**无轮询无定时器**（零常驻判据）。
- 顶部手动新建：Input + 「添加」（source='inbox'，space=props.space，回车提交）。
- 状态 tab 四档（待处理/进行中/已完成/已放弃）带计数；行=标题 + 来源 Tag +
  相对时间 + note 折叠 + 动作（按状态机：pending→〔开始〕〔放弃〕、doing→
  〔完成〕〔放弃〕、终态→〔删除〕；全部可〔删除〕）；禁用态走 CanTransition
  前端镜像（纯展示逻辑，测试钉死）。
- 跳转：Target 非空且 action=navigate → 〔去板块〕onNavigate(Target)；Session
  非空 → 〔回会话〕onNavigate('gaea')（V1 板块粒度，精确会话见 §3.4）。
- 空态：V3Empty 两分（无任务/加载失败重试）。
- data-testid：task-inbox-panel / task-inbox-row / task-inbox-add。

### 3.2 双空间首页挂点（ModuleLauncher.tsx）

- 书斋（work）：w-vitals 增「任务」节（w-vital section，图标章 + 待处理计数 +
  「打开收件箱」，data-testid=desk-task-inbox）；计数=open 时拉取？否——首页
  渲染时一次性 `GaeaTaskInboxList('work')` 取 pending 数（与 desk-recent-docs
  同款首屏一次读，零轮询）；点击开 §3.1 面板（space='work'）。
- 闲庭（play）：p-foot 园底信息带增第五节（data-testid=garden-task-inbox，
  待处理计数 + 打开）；若 p-foot CSS 为固定四列网格则该节跨两列或降级为
  garden-chips 区头部一枚 chip（实现时以 styles.css 实况为准，**不破坏既有
  garden-progress/sessions/memory/meters 四 testid**）；点击开面板 space='play'。
- 面板单例挂 ModuleLauncher 顶层，space 跟随当前 home 空间；onNavigate 用
  ModuleLauncher 既有 onNavigate 回调（判据②：跳回来源板块）。

### 3.3 四入口接线

- Ctrl+K（SearchModal.tsx）：指令预览卡加次按钮〔存为任务〕
  （data-testid=intent-save-task）→ `GaeaTaskInboxSave({title:query, space:
  props.space, source:'ctrlk', action:preview.action, target:preview.target})`，
  成功后 intentReply 位内联回执「已存入收件箱」；preview 未命中指令时不出按钮
  （搜索词≠指令，宁漏勿误——手动新建走收件箱面板）。
- preview.action==='save_task' 时「执行」直接调 Save（source='ctrlk'，同上）。
- 意图中枢（gaea/App.tsx CommandPalette）：paletteItems 末尾 query 非空时加
  动态项「存为任务『query』」（group=命令）→ Save（source='palette'，
  space='work'——办公工作台内；session=当前会话路径若有）。
- 语音/微信：零前端改动（§2.3 后端闭环；回复文本即回执）。

### 3.4 源会话接缝（调研后二选一，如实报告）

- 若 gaea App 有按路径打开会话的现成 seam（palette sessionItems 的 run 动作）：
  TaskInboxPanel 增可选 prop `onOpenSession?(path)`，面板挂 gaea 工作台侧时
  〔回会话〕精确打开；首页挂点无 seam 则降级 onNavigate('gaea')。
- 若无干净 seam：V1 一律板块粒度，「精确回源会话」欠账如实入 releases 清单。

### 3.5 类型与字典

- lib/types 本地重述 TaskInboxView（herdsman 防环先例，不依赖 AppModels 再生）。
- 三语 locales 新键 ~22（shell.launcher.taskInbox / tasks.inbox.* 标题、四状态、
  动作、来源、空态、回执），三语键集相等（zh/en/zh-TW）。
- **不碰** bridge/*、bindingNames.ts、spaceBindings*、mock/*、wailsjs（主代理收口）；
  因此 tsc -b 在收口前必红——**只跑定向 vitest（vi.mock bridge）**，如实报告。

## 4. 主代理收口清单（本刀版本 v4.318.0）

1. 线间对账（API 漂移修正）；bridge/core +4、mappings +4、bindingNames 再生
   691→695、mock 四桩（含样本）、spaceBindings GaeaTaskInbox* → shared
   （两空间首页都调；work/play 锁数各 +4，锁数测试更新）、`go run ./scripts/gen_bindings`。
2. 定向：tsc -b、eslint 改动文件、go/vitest 定向；全量 `scripts/ci.ps1` 恰一次
   （≈5-7 分钟）。
3. 版本四处 4.318.0（sync-version.ps1 口径）→ 前端 build → `wails build -s` →
   exe/SUMS/桌面副本/冒烟 → releases/v4.318.0.md + CHANGELOG/README →
   .gaea AGENTS（迁 1 插 1）+ progress + todos → commit+tag。

## 5. 足迹互斥表

| 线 | 独占足迹 |
|---|---|
| A | internal/taskinbox/** |
| B | internal/intent/intent.go(+intent_test 增例)、internal/app/gaea_task_inbox.go(+test)、internal/app/intent_router.go、internal/app/bindings_office.go(+4) |
| C | frontend/src/gaea/components/TaskInboxPanel.tsx(+test)、ModuleLauncher.tsx(+test 增量)、SearchModal.tsx(+test 增量)、gaea/App.tsx、gaea/lib/types.ts 重述、locales 三语 |
| 主 | 规格书（本档）、契约/生成文件（bridge/core、mappings、mock、spaceBindings、bindingNames、gen_bindings 产物）、版本四处、releases/CHANGELOG/README/.gaea 记忆 |

## 6. 出口对照与观察池

- 判据①（四入口任一创建并在收件箱追踪）：Ctrl+K 按钮/面板动态项/语音短语/
  微信短语四路 source 落库可区分，测试钉死 ✅（面板手动新建=第五来源兜底）。
- 判据②（跳回来源板块与源会话）：板块级 ✅；会话级按 §3.4 调研结论（无 seam
  则欠账入档）。
- 判据③（空间隔离零泄漏）：space 必带 + FilterBySpace 严格相等 + 两首页面板
  各查各空间，测试钉死 ✅。
- 判据④（零常驻）：无定时器/无轮询/无后台事件订阅，读写均由用户动作触发 ✅。
- 观察池：任务模板联动（GaeaTaskTemplates 与收件箱「从模板建任务」）；收件箱
  首屏化（7.3-2 板块降视图的前置数据）；跨空间显式移动；LLM 兜底分类器对
  save_task 的覆盖；真机走查（四入口各建一条→收件箱状态机走全→跳回板块）。
