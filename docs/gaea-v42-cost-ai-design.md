# gaea v4.2「智慧」工位造价包 · AI 组价 + 询价飞轮 + 五算对比

> 2026-08-29 定稿。权威路线图 `docs/gaea-nextgen-roadmap-2026.md` §10.4/§14 阶段 3+；
> 本文件为该 Step 的执行契约（数据模型/包接口/绑定面/前端面板/验收）。
> 纪律沿用：每 Step 独立提交可回退、旧数据只读兼容、不做新板块、不堆功能。

## 1. 背景与目标

v3.1.0 已建成造价数据库（成本库 + 测算项目 + 造价参考 + 复盘笔记 + 价格源抓取 +
AI/OCR 报价单导入 + 供应商比价）。v4.2 把「数据」升级为「智慧」：

1. **AI 组价**：清单描述 → 分类定位 → 相似清单检索（语义+关键词）→ 价格带推荐
   （P25–P75 + 置信度 + 证据链）→ LLM 人材机拆解 → 一键确认回写。现有溯源字段
   （region/price_date/price_type/source）天然是证据格式——对手没有的先天优势。
2. **询价飞轮**：信息价 + OCR 报价 + 供应商比价 + 手动询价四源归一为「询价库数据点」；
   每张 OCR 报价单自动变数据点；到期预警（valid_until）+ 调差建议（旧价 vs 最新询价）。
3. **五算对比**：测算容器/版本快照延伸到五算（估/概/预/结/决），阶段金额对比 +
   相邻阶段偏差特征提取（供复盘笔记 AI 诊断复用）。

**验收（红线）**：① 组价建议必须带证据链（相似条目+来源+期数+口径）与置信度；
② 询价数据点无确认不写成本库（沿用「无确认不落库」纪律）；③ 五算对比纯本地计算，
LLM 仅做偏差诊断文案；④ 全链路离线可用（语义/重排走本地 Herdsman，LLM 走办公功能
模型 `routeSensitiveLocal("office")`，失败降级规则化兜底）。

## 2. Step 拆分与发布节奏

- **v4.2a「组价底座」**：`cost.PriceBand` 价格带推荐纯函数（本 Step 即出，供 AI 组价
  与前端「价格带参考」共用）。
- **v4.2b「询价飞轮」**：`costinquiry` 包（数据点存储 + 到期预警 + 调差建议）+
  `coststage` 包（五算阶段值 + 对比 + 偏差特征）——两个自包含存储包，包内
  `CREATE TABLE IF NOT EXISTS` 自建表（父代理集成时统一收编进 db/schema.go 正式迁移）。
- **v4.2c「AI 组价」**：`GaeaCostCompose` 绑定（分类定位→相似检索→价格带→LLM 拆解→
  建议视图）+ 前端「AI 组价」面板（ComposeModal）。
- **v4.2d「面板收口」**：前端询价面板（询价库/到期预警/调差）+ 五算对比视图 +
  复盘笔记 AI 诊断接线；绑定面漂移检查 + 版本统一 + 发布。

> v4.2a/b 纯后端可并行；c/d 依赖 a/b 产出与绑定面，父代理集成。

## 3. 数据模型（SchemaV15，父代理收编）

```sql
-- 询价库数据点（四源归一：信息价/OCR报价/供应商比价/手动询价）
CREATE TABLE IF NOT EXISTS cost_inquiry_records (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  title       TEXT NOT NULL,             -- 品名/清单描述
  spec        TEXT NOT NULL DEFAULT '',  -- 规格型号
  unit        TEXT NOT NULL DEFAULT '',
  price       REAL NOT NULL,             -- 元/单位
  source      TEXT NOT NULL DEFAULT '手动询价', -- 信息价/OCR报价/供应商比价/手动询价
  supplier    TEXT NOT NULL DEFAULT '',  -- 供应商/来源点名（信息价=期数名）
  region      TEXT NOT NULL DEFAULT '',
  price_date  TEXT NOT NULL DEFAULT '',  -- 价格时间/期数
  valid_until TEXT NOT NULL DEFAULT '',  -- 有效期至 YYYY-MM-DD（空=长期）
  note        TEXT NOT NULL DEFAULT '',
  status      TEXT NOT NULL DEFAULT '现行', -- 现行/已过期
  created_at  TEXT NOT NULL,
  updated_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_cost_inquiry_title ON cost_inquiry_records(title);

-- 五算阶段值（project_id 复用 cost_projects.id，stage 取 估算/概算/预算/结算/决算）
CREATE TABLE IF NOT EXISTS cost_stage_values (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  project_id TEXT NOT NULL,
  stage      TEXT NOT NULL,              -- 估算/概算/预算/结算/决算
  amount     REAL NOT NULL,              -- 阶段金额（元）
  date       TEXT NOT NULL DEFAULT '',   -- 阶段日期 YYYY-MM-DD
  note       TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(project_id, stage)
);
```

`cost_entries.valid_until`（SchemaV9 已有）为到期预警数据源，不加列。

## 4. 后端包契约（v4.2a/b，子代理实现）

### 4.1 `internal/gaea/cost/priceband.go`（纯函数，零依赖新增）

```go
// BandSource 证据链一条：相似条目快照（不携带 Body）。
type BandSource struct {
    Name, Title, Category, Unit, Spec, Source, Region, PriceDate, PriceType string
    Price float64
    UpdatedAt time.Time
}

// PriceBand 相似清单的价格带推荐（对标造价参考分位数口径）。
type PriceBand struct {
    Samples    int         // 参与统计的样本数
    Min, Max   float64
    Mean       float64
    Median     float64     // P50
    P25, P75   float64     // R-7 线性插值（与 costref.percentile 同口径）
    SpreadPct  float64     // 离散度 (P75-P25)/Median*100
    Outliers   int         // 超出 P25-1.5IQR / P75+1.5IQR 的样本数
    Confidence string      // 高(>=8)/中(4-7)/低(1-3)/—(0)
    Sources    []BandSource // 全部参与样本（含离群，标记由 Outliers 数承担）
}

// ComputePriceBand 由相似条目计算价格带：
//   - 按 unit 过滤（非空且与条目 Unit 不一致则排除；空 unit 不过滤）；
//   - 过滤无效价格（<=0）；
//   - 无有效样本返回 nil；
//   - 分位数口径必须与 costref 一致（R-7 线性插值）。
func ComputePriceBand(entries []Summary, unit string) *PriceBand

// RecommendPrice 推荐价：mode = median(默认)/mean/p25/p75/conservative（保守=min）。
// 返回推荐价与理由文案（含样本数与置信度）。
func RecommendPrice(b *PriceBand, mode string) (float64, string)
```

规则约束：单样本时 P25=Median=P75=该值；离群用标准 1.5×IQR；`SpreadPct` 在
Median=0 时为 0。新增文件仅 `priceband.go` / `priceband_test.go`，不动既有文件。

### 4.2 `internal/gaea/costinquiry/`（新包，自建表）

```go
type Record struct { // 对应 cost_inquiry_records，json 标签驼峰
    ID int64; Title, Spec, Unit string; Price float64
    Source, Supplier, Region, PriceDate, ValidUntil, Note, Status string
    CreatedAt, UpdatedAt time.Time
}
type Store struct{ db *sql.DB }
func Open(gdb *sql.DB) *Store            // 不可用时 Available()=false
func (s *Store) Save(r Record) (int64, error)  // id<=0 新建
func (s *Store) List(query string, limit int) []Record
func (s *Store) Get(id int64) (*Record, error)
func (s *Store) Delete(id int64) error
// ListExpiring 到期预警：valid_until 非空且 <= today+days 的现行记录（新→旧）。
func (s *Store) ListExpiring(days int) []Record

// AdjustSuggestion 调差建议：条目 vs 最新询价数据点。
type AdjustSuggestion struct {
    EntryName, EntryTitle string; EntryPrice float64
    LatestPrice float64; LatestDate string; LatestSource string
    Diff float64; DiffPct float64
    Unit string
}
// SuggestAdjustments 对 cost_entries 逐条按标题归一化匹配（matchTitle 纯函数：
// 去空白/全半角/大小写/括号内规格），命中且 DiffPct 显著(>2%) 时产出建议；
// 无数据点或库空返回 nil。entries 由调用方传入（cost.Store.List）。
func (s *Store) SuggestAdjustments(entries []cost.Summary) []AdjustSuggestion
```

包内 `Open` 执行 `CREATE TABLE IF NOT EXISTS cost_inquiry_records ...`（幂等，内容见
§3）——**不许改 db/schema.go**（父代理收编正式迁移）。测试用 `t.TempDir()` sqlite
（`db.OpenFile` 或 modernc 内存库，看既有包怎么测——参照 `cost_test.go`）。

### 4.3 `internal/gaea/coststage/`（新包，自建表）

```go
// Stage 五算阶段常量。
const (
    StageEstimate   = "估算" // 投资估算
    StageDesign     = "概算" // 设计概算
    StageBudget     = "预算" // 施工图预算
    StageSettlement = "结算" // 竣工结算
    StageFinal      = "决算" // 竣工决算
)
var StageOrder = []string{StageEstimate, StageDesign, StageBudget, StageSettlement, StageFinal}

type StageValue struct { // 对应 cost_stage_values
    ID int64; ProjectID, Stage string; Amount float64; Date, Note string
    CreatedAt, UpdatedAt time.Time
}
type Store struct{ db *sql.DB }
func Open(gdb *sql.DB) *Store
func (s *Store) SaveStage(v StageValue) error          // UNIQUE(project_id,stage) UPSERT
func (s *Store) ListStages(projectID string) []StageValue // 按 StageOrder 排序
func (s *Store) DeleteStage(projectID, stage string) error

// CompareRow 对比行：阶段/金额/环比(相对上一阶段)/累计差(相对估算)/环比差幅%。
type CompareRow struct {
    Stage string; Amount float64
    PrevAmount float64; HasPrev bool; ChainDiff float64; ChainDiffPct float64
    BaseDiff float64; BaseDiffPct float64
}
// ComputeComparison 纯函数：按 StageOrder 补全缺阶段（Amount=0 标记缺失），
// 产出对比表。少于 2 个有值阶段返回 nil。
func ComputeComparison(values []StageValue) []CompareRow

// Deviation 相邻阶段偏差特征（供前端展示/复盘笔记 AI 诊断输入）。
type Deviation struct {
    FromStage, ToStage string
    FromAmount, ToAmount float64
    Diff, DiffPct float64
    Direction string // 上升/下降
    Level     string // 正常(<5%)/关注(5-15%)/异常(>15%)，阈值常量可调
    Suggestion string // 规则文案，如「预算较概算 +18.2%，建议核查工程量或单价差异」
}
func ExtractDeviations(rows []CompareRow) []Deviation
```

同样包内自建表，不改 db/schema.go；测试同 4.2。

## 5. 绑定面新增（v4.2c/d，父代理集成）

`CostB` 门面新增（gen_bindings 自动归入，绑定面 506 → 约 514）：

- `GaeaCostInquirySave(r costinquiry.Record) (int64, error)`
- `GaeaCostInquiryList(query string, limit int) []costinquiry.Record`
- `GaeaCostInquiryDelete(id int64) error`
- `GaeaCostInquiryExpiring(days int) []costinquiry.Record`
- `GaeaCostInquiryAdjust() []costinquiry.AdjustSuggestion`（内部遍历 cost_entries）
- `GaeaCostStageSave(v coststage.StageValue) error`
- `GaeaCostStages(projectID string) []coststage.StageValue`
- `GaeaCostStageCompare(projectID string) []coststage.CompareRow`（含偏差标注或单独）
- `GaeaCostStageDeviations(projectID string) []coststage.Deviation`
- `GaeaCostCompose(desc string) (CostComposeView, error)`（v4.2c）
- `GaeaCostComposeApply(view CostComposeView) (string, error)`（确认回写 cost_entries）

`CostComposeView`（app 层视图类型）：`Description/CategoryPath/Unit/Quantity/`
`Band *cost.PriceBand/Components []CostComponentView/RecommendedPrice/Confidence/`
`Evidence []CostComposeEvidence`（相似条目：名称/标题/单价/单位/来源/地区/期数/口径/
相似度）。绑定门面与视图文件放 `internal/app/`，子代理不做（父代理集成）。

## 6. 前端面板（v4.2c/d，父代理集成）

- **AI 组价**（CostProjectsView 明细行「AI 组价」按钮 + ComposeModal）：输入描述 →
  展示价格带（P25/P50/P75 迷你统计）+ 证据链相似条目表 + LLM 人材机拆解组件预览 →
  「应用为明细行」/「沉淀成本库」两档确认。复用 `CostEntryModal` 的组件编辑。
- **询价**（CostLibraryPanel 右侧「询价」子 Tab）：数据点列表（四源徽标）+ 新增/编辑 +
  到期预警横幅（红）+ 调差建议表（一键应用到成本库=走 GaeaCostSave）。
- **五算对比**（CostProjectsView 项目详情「五算对比」区）：五阶段金额输入/保存 +
  对比表（环比/累计差/差幅着色）+ 偏差卡片（异常红/关注黄）+ 「生成复盘笔记」按钮
  （调 GaeaCostNoteSave 预填偏差诊断文案——LLM 生成放 v4.2d 若时间允许，否则规则文案）。

前端页统一走既有 gaea/** 组件风格（i18n 三语：壳层 chrome 已有字典，面板内 zh 单语
按 i18n 决策）。空间标签：cost 全为 work 域（spaceBindings 分类表同步）。

## 7. 验收清单

1. `go test ./internal/gaea/cost/ ./internal/gaea/costinquiry/ ./internal/gaea/coststage/`
   全绿（新测试覆盖：单样本/多样本/单位过滤/离群/空输入/到期边界/调差匹配/对比补全/偏差阈值）。
2. 绑定面漂移 PASS（新增绑定同步 bindingNames.ts + spaceBindings 分类）。
3. 前端 tsc/eslint 0 errors + vitest 全绿（ComposeModal/询价/五算对比交互用例）。
4. 旧数据只读兼容：既有 cost_entries/测算项目零改动可读；新表独立建。
5. 回退：v4.2a/b/c/d 各自独立提交，可单独 revert。
6. 全链路离线：语义走本地 Herdsman；LLM 拆解走 `routeSensitiveLocal("office")`，
   不可用时返回规则兜底（中位数价格 + 顶部相似条目组件结构），不阻塞组价。
