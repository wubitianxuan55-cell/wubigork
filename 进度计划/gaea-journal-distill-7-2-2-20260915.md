# gaea 7.2-2 journal 历史蒸馏 · 实施规格（v4.317.0）

> 来源：`docs/gaea-stage7-plan-2026-09.md` §2 7.2-2（依赖 7.1-1 评测已解锁，7.2-1 审阅管道复用）。
> 刀法：契约先行 → A/B/C 三线并行（足迹互斥）→ 主代理收口（gen_bindings / 门禁 / 发版）。
> 红线：纯同步按需（无常驻）；work 空间一期（journal 本身恒 work，天然满足）；
> 只提示不打扰（用户拒绝的模式静默）；审计链完整（哪次历史、谁确认、哪版）。

## 0. 论点与范围

journal 证据链（`.gaea/work/journal/*.jsonl`，ChangeRecord 每卡=一次已应用变更）是
书斋审批链的既有沉淀。本刀把它变成程序性记忆的矿床：**确定性挖掘跨会话重复 ≥3 次的
工具应用模式 → 建议卡只提示 → 用户点「结晶为技能」→ LLM 从多次执行的回放蒸馏草稿 →
复用 7.2-1 审阅管道（编辑→GaeaSkillDraftSave 落盘）→ 决定与结晶全程落审计状态**。

范围裁决：
- **V1 模式=归一化后会话级变更序列的精确匹配**（连续同形步骤折叠后全序列相等）。
  滑窗/子序列挖掘不进 V1（误报风险高，真实数据先行）——观察池。
- 单会话流程折叠后步数 ∉ [2,12] 不参评（单步不成流程；超长序列过于特异）。
- 结晶技能调用 ≥5 次且成功率可见（出口判据②的机制面）**不在本刀**：需技能调用
  计数机制（statsFile 先例另刀），本刀先落建议+审阅+审计三件，欠账如实入档。

## 1. 线 A · 纯函数包 `internal/skilldistill`（零 IO，先例 routesuggest/novelgate）

```go
package skilldistill // import "github.com/gaea/gaea/internal/skilldistill"

const (
    MinRepeat     = 3  // 跨会话重复门槛（plan「重复 ≥ N 次」）
    MaxCandidates = 3  // 同时至多提议条数（宁少勿扰）
    MinFlowSteps  = 2  // 折叠后最少步骤
    MaxFlowSteps  = 12 // 折叠后最长步骤
)

// Record 是 evidence.ChangeRecord 的本地投影（防包环：不 import evidence）。
type Record struct{ SessionID, Tool, Target string; At int64 }

// FlowStep 一次工具应用；Flow 一个会话内的序列（At 升序）。
type FlowStep struct{ Tool, Target string; At int64 }
type Flow struct{ SessionID string; Steps []FlowStep }

// PatternStep 归一化步骤：Tool + Shape（目标扩展名小写含点；无扩展名 "*"）。
type PatternStep struct{ Tool, Shape string }

// Candidate 重复模式候选。
type Candidate struct {
    ID       string   // "jd-"+sha256(pattern 串接)[:8hex]，确定性幂等
    Pattern  []PatternStep
    Repeat   int      // 出现该模式的**不同会话数**
    Sessions []string // 证据会话（最近优先，≤5）
    Evidence []string // 人类可读证据行（≤5，行 ≤120 rune："会话 s1（日期）：tool 文件名 → …"）
    FirstAt, LastAt int64
}

// MineFlows 聚合：按 SessionID 分组、(At,Tool,Target) 排序；Tool/SessionID 空跳过。
func MineFlows(records []Record) []Flow

// Distill 挖掘：每会话流 → 折叠连续同 (Tool,Shape) → 全序列精确匹配分组
// → repeat≥MinRepeat 且步数∈[2,12] → ignored 静默跳过 → 排序（repeat 降序、
// 同数按 ID 升序）→ 截 MaxCandidates。
func Distill(flows []Flow, ignored map[string]struct{}) []Candidate

// PatternLine 步骤行渲染："edit_file .md"（视图层直显）。
func PatternLine(p PatternStep) string
```

测试（表驱动）：聚合排序与空值跳过 / 连续折叠 / 精确匹配分组与跨会话计数 /
同会话多次不计 repeat / 门槛与步数区间过滤 / ignored 静默 / ID 确定性（同输入同 ID）/
排序与截断 / Evidence 行数与截断。

## 2. 线 B · 绑定与蒸馏管道 `internal/app/gaea_skill_distill.go` + 模板

### 2.1 前端契约（绑定面 688→691，OfficeB +3）

```go
type SkillDistillCandidateView struct {
    ID       string   `json:"id"`
    Pattern  []string `json:"pattern"` // PatternLine 产物
    Repeat   int      `json:"repeat"`
    Sessions []string `json:"sessions"`
    Evidence []string `json:"evidence"`
    LastAt   int64    `json:"lastAt"`
}
type SkillDistillView struct {
    Candidates  []SkillDistillCandidateView `json:"candidates"`
    Available   bool                        `json:"available"` // journal 目录可读且有卡
    GeneratedAt string                      `json:"generatedAt"`
}

func (a *App) GaeaSkillDistillCandidates() SkillDistillView          // 现算只读，零 LLM
func (a *App) GaeaSkillDistillDraft(patternID string) (SkillRecordResult, error) // LLM 草稿，只回不落盘
func (a *App) GaeaSkillDistillDecide(patternID, decision, skillName string) error
```

- Candidates：读 journal（GaeaJournalList 同款 ReadDir+OpenJournal+List）→ Record
  投影 → MineFlows → Distill(ignored ∪ crystallized)；目录缺失=Available false 空表。
- Draft：重算候选按 ID 找（找不到报错）→ buildJournalReplay（纯函数：证据流程
  ≤3 个最近会话，「── 第 k 次执行（会话 s…，日期）──」+ 逐步「【工具】tool target」
  + AfterSummary 前 200 rune）→ `prompts/skill-from-journal.json` →
  routeOfficeLocal("office") + RetryJSON 2 + parseSkillDraft →
  SkillRecordResult{Draft, Replay, Preview}（复用 7.2-1 全套结构与渲染）。
- Decide：decision ∈ {"ignore","crystallized"}；crystallized 必须带 skillName（落审计：
  哪些会话、何时、结晶成哪个技能）；ID 形状校验（jd-+8hex，先例 ParseSuggestionID）。
- 状态：`<DataRoot>/skill_distill_state.json`，route_suggestions.json 同款容错读 +
  原子写（temp+rename）。结构
  `{version:1, ignored:{id:{decidedAt}}, crystallized:[{id,skill,sessions[],decidedAt}]}`。
- 门面：bindings_office.go OfficeB +3 纯委托。**不跑 gen_bindings**（主代理收口统一跑）。

### 2.2 模板 `prompts/skill-from-journal.json`（skill-from-session 同构）

input_sections：replay（P0，多次执行回放）/ pattern（P1，模式步骤行）/ repeat（P1）。
纪律：只依据回放不虚构；**合并多次执行的共性为步骤，差异写进 cautions**；文件名
通用化占位（<主题>.md）；name 英文 kebab-case；output JSON 同 7.2-1 五字段。

### 2.3 测试（internal/app/gaea_skill_distill_test.go）

回放构建（多会话段/截断/≤3 会话）/ 状态往返（ignore 静默复算验证 + crystallized
审计记录）/ Candidates 端到端（临时工作区 journal 3 会话同模式 → repeat 3；
isolateWorkspaceTo 先例）/ 守卫（无 ctrl：Candidates Available false、Draft 报错）。
只跑定向 `go test ./internal/app -run TestSkillDistill -count=1` 与
`go test ./internal/skilldistill -count=1`；**不跑全量、不跑 ci.ps1、不 commit**。

## 3. 线 C · 前端（建议卡 + 审阅复用）

- `components/SkillDistillSection.tsx`（+test）：props
  `{view, loading?, onDraft(id), onIgnore(id)}`；候选卡=模式步骤序（PatternLine 行）
  + 「重复 N 次」徽章 + 证据折叠 + 动作〔结晶为技能〕〔不再提示〕；空态两分
  （无候选/不可用）。antd，风格对齐 MemoryPanel 建议区。
- `MemoryPanel.tsx`：建议 tab 增「流程蒸馏」分区，新 props 全部可选（既有测试零改动）。
- `SkillRecordModal.tsx`：加可选 `preload?: SkillRecordResult`——传入则跳过内部蒸馏
  直接回填表单（7.2-1 行为零变化，既有测试不动）。
- `useSessionHandlers.ts`：distill 视图 + 三个 handler（Candidates 拉取 / Draft→
  开 SkillRecordModal(preload) / Decide(id,"ignore","")）。
- 类型：lib/types 本地重述 SkillDistillCandidateView/SkillDistillView（herdsman 防环
  先例，不依赖 AppModels 再生）。
- 三语字典：memory.distill.* 键集三语相等（zh/en/zh-TW）。
- **不碰** bridge/*、bindingNames.ts、spaceBindings*、mock/*、wailsjs（主代理收口）；
  因此 `tsc -b`/build 在收口前必红——**只跑定向 vitest（vi.mock bridge）**，如实报告。

## 4. 主代理收口清单（本刀版本 v4.317.0）

1. 线间对账（API 漂移修正）；bridge/core.ts +3、mappings.ts +3、bindingNames.ts
   688→691、mock 桩、spaceBindings.ts work 锁 510→513 + 锁数测试、wails models 再生、
   `go run ./scripts/gen_bindings`。
2. 定向：tsc -b、eslint 改动文件、go/vitest 定向；全量 `scripts/ci.ps1` 恰一次
   （≈5-7 分钟）。
3. 版本四处 4.317.0（sync-version.ps1 口径）→ 前端 build → `wails build -s` →
   exe/SUMS/桌面副本/冒烟 → releases/v4.317.0.md + CHANGELOG/README →
   .gaea AGENTS（迁 1 插 1）+ progress + todos → commit+tag。

## 5. 足迹互斥表

| 线 | 独占足迹 |
|---|---|
| A | internal/skilldistill/** |
| B | internal/app/gaea_skill_distill.go(+test)、internal/app/bindings_office.go、prompts/skill-from-journal.json |
| C | frontend/src/gaea/components/SkillDistillSection.tsx(+test)、MemoryPanel.tsx、SkillRecordModal.tsx、useSessionHandlers.ts、lib/types 本地重述、locales 三语 |
| 主 | 规格书（本档）、契约/生成文件、版本四处、releases/CHANGELOG/README/.gaea 记忆 |

## 6. 出口对照与观察池

- 判据①（只提示不打扰、拒绝即静默）：Decide ignore → 状态文件 → 复算跳过，测试钉死 ✅
- 判据③（审计链完整）：crystallized 记录 id/skill/sessions/decidedAt 落状态文件 ✅
- 判据②（调用 ≥5 次且成功率可见）：**欠账**——需技能调用计数（statsFile v3 先例另刀），
  如实写入 releases 欠账清单与 todos。
- 观察池：滑窗/子序列模式挖掘（V2，等真实 journal 数据）；journal 回放纳入 verdicts
  （复核结论）过滤；蒸馏模板温度/上限调参；真机走查（建议卡→结晶→新会话命中一条龙）。
