# gaea 7.1-2 路由学习：任务级账目 + 成本归因 + 建议制改绑（2026-09-15）

> 阶段七规划 `docs/gaea-stage7-plan-2026-09.md` §1.2 的实施规格。
> 前置调研：三线子代理调研结论已入档（.gaea/todos.md 阶段七行与 AGENTS v4.315.0 速览「未做（下刀）」）。
> 版本：v4.316.0。红线：routeModel 决策链零改动；不自动改绑（建议制，采纳=用户动作）；
> 离线模式/办公本地优先等策略路由优先级高于建议层；领域规则不进代码（通用机制）。

## 0. 调研结论（实施依据，file:line 相对仓库根）

- **R1 遥测**：`internal/ai/client.go` 两处 `ModelCallUsage` 构造（recordUsage≈:1029、流式 defer≈:805）→ `modelengine.Manager.RecordCall`（stats.go:614）是**唯一全量无旁路收口**（成功/失败/故障转移全过）；现聚合 `statsKey=engineID|model`（stats.go:345）无 feature 维度；`ChatRequest`（internal/ai/types.go:54-71）已有 `EngineID string json:"-"` 带外字段先例。
- **R2 绑定**：功能绑定键 8 个（chat/whisper=chat 别名/novel/office/gaea/characterlib/routine/sin，枚举点 `internal/app/feature_model_handler.go:17-41`）；改绑走 `App.SetFeatureModel`（internal/app/gaea_handler.go:349，gaea/office 域即时 applyOfficeFeatureModel）；本地引擎（EngineType.IsLocal：ollama/herdsman/cosyvoice/modelhub）今天即可绑定功能域（SetFeatureModel 校验无 IsLocal 排除）；成本估算 `estimatePrice/EstimateCostCNY`（stats.go:212/254，本地恒 0）；质量代理=stats 成功率/延迟+herdsman benchmark（TTFT/TPS，仅本地模型）。
- **R3 前端**：落点=模型中心新「成本归因」tab（候选 A，零右栏契约/零三语字典触碰）；testid 惯例 `mc-*`；⚠️ 已有 `GaeaCostAttribution`（造价归因对标，bindingNames.ts:115）重名不同义——新绑定命名避开 CostAttribution 字样；统计类先例走 `frontend/src/api/engines.ts` 主 App 通道（getModelCallStats :449）。

## 1. 契约（三线共同遵守）

### 1.1 数据模型

```go
// internal/modelengine/stats.go 扩展（Line A）
type ModelCallUsage struct { /* 既有字段 */ Feature string }          // 新增；空串=历史/未标记
// statsKey: "feature|engine|model"（feature 为空时归 ""，展示层标「未标记」）
// statsFile 版本 2→3：读 v2 时全量按 feature="" 兼容读入，首次落盘升 v3
// GetModelCallStats 汇总结构新增 per_feature 段：[]FeatureUsageSummary{Feature, Calls, Tokens, Cost, SuccessRate, AvgMs}
```

### 1.2 App 绑定方法（Line A 提供 ledger，Line B 提供 suggestions；主代理统一 gen_bindings）

```go
// Line A：internal/app/gaea_route_ledger.go（新文件）
func (a *App) GaeaRouteLedger() (RouteLedgerView, error)
// RouteLedgerView{GeneratedAt, Total{Calls,TokensIn,TokensOut,CostCNY,AvgMs,SuccessRate}, Features []FeatureUsageSummary}
// 数据源：GetModelCallStats 的 per-feature 段（A 线产出）；成本口径=既有估算（EstimateCostCNY），本地引擎恒 0 属实呈现

// Line B：internal/app/gaea_route_suggestions.go（新文件）
func (a *App) GaeaRouteSuggestions() (RouteSuggestionsView, error)   // 现算+合并忽略表
func (a *App) GaeaRouteSuggestionApply(id string) error              // 内部必须调 a.SetFeatureModel（保即时重建），绝不自动调用
func (a *App) GaeaRouteSuggestionIgnore(id string) error             // 写忽略状态，被忽略建议不再出现
```

存储：`<DataRoot>/route_suggestions.json`——`{version:1, records: map[suggestionID]{status:"ignored"|"applied", decidedAt}}`（原子写；先例 pricefeed SetFetchStatus / engines.json 同目录惯例）。

### 1.3 纯函数包 internal/routesuggest（Line B）

```go
type Input struct {
    Features []FeatureBinding // {Feature, EngineID, Model}——当前绑定（routeModel 解析后的生效值）
    Candidates []Candidate    // 全部 enabled 引擎的 kind=llm 模型（本地云端同池，天然满足判据④）
                               // {EngineID, Model, IsLocal, SuccessRate(0-1, 无样本=nil), AvgMs, CostCNY(本地 0), TTFTMs/TPS(可选, herdsman)}
}
type Suggestion struct { ID, Feature string; From, To {EngineID,Model}; ScoreGap float64; Reason, Evidence string }
func Suggest(in Input, ignored map[string]struct{}) []Suggestion
```

评分与阈值（**宁少勿扰**）：score = quality × costFactor；quality=SuccessRate（无样本不给建议——证据不足不出声）；costFactor=归一化成本（max(cost, ε) 倒数归一）；同 feature 下候选 score − 当前绑定 score ≥ **ScoreGapThreshold=0.25**（包级导出常量，memoryeval.Threshold 先例）才出建议；单 feature 至多 1 条（取分差最大者）；样本量 < MinCalls=20 的候选不出建议；被忽略 ID 静默。ID 确定性：`feature:fromEngine/fromModel>toEngine/toModel`（重算幂等，忽略状态不漂移）。

### 1.4 前端（Line C）

- 落点：模型中心左栏新分类「成本归因」（Category 联合 + 'attribution'；TABS +1；渲染分支 +1）。
- `AttributionSection.tsx`：KPI 行（总成本 CNY/总 token/调用数/成功率，复用 KpiTile）+ 按功能绑定键分组表（功能/引擎/模型/调用/token/成本/均时长/成功率）+ 每组上方建议卡（分差/理由/证据 + 采纳/忽略按钮；采纳调 GaeaRouteSuggestionApply 后既有 `feature-model-changed` 事件自动刷新绑定面）。
- 数据通道：`frontend/src/api/engines.ts` 加 `getRouteLedger()` / `getRouteSuggestions()` / `applyRouteSuggestion(id)` / `ignoreRouteSuggestion(id)`（贴 getModelCallStats 先例）。
- testid：`mc-attribution` / `mc-attribution-kpi` / `mc-attribution-row-${feature}` / `mc-suggest-${feature}` / `mc-suggest-${feature}-apply` / `mc-suggest-${feature}-ignore`；zh 单语（模型中心惯例，不进三语字典）；空态用 modelcenter EmptyState。

## 2. 文件足迹互斥表

| 线 | 足迹（只许改这些） |
|---|---|
| A 账目维度 | internal/ai/types.go、internal/ai/client.go、internal/modelengine/stats.go(+stats_test)、internal/gaea/provider/bridge/bridge.go、internal/app/*_handler.go 等**既有调用点一行式透传**（Feature: feature）、新 internal/app/gaea_route_ledger.go |
| B 建议回路 | 新 internal/routesuggest/*（包+测试）、新 internal/app/gaea_route_suggestions.go(+测试) |
| C 归因视图 | frontend/src/pages/modelcenter/utils.tsx、ModelCenterPage.tsx、新 AttributionSection.tsx、新 hooks/useAttributionState.ts、新 AttributionSection.test.tsx、frontend/src/api/engines.ts |
| 主代理收口 | scripts/gen_bindings 产物（bindingNames.ts、bindings_*.go 门面、wails.d.ts）、frontend/src/gaea/lib/spaceBindings.ts(+test 锁 506→510)、frontend/src/gaea/lib/mock/*（如走 mock）、drift/ci/build/发版 |

**铁律**：三线互不触碰对方足迹；生成类文件全部主代理统一处理；B/C 不等 A 也可开工（接口已在本契约钉死）。

## 3. 出口判据对账（阶段七 §1.2）

1. 任务级账目覆盖全部功能绑定键 ← A（8 键全透传+stats 三维桶）
2. 归因视图可查任意会话/任务成本 ← C（模型中心 tab，per-feature 聚合；「任意会话」= 全局账本视图，会话级切片已有 ContextView SessionStats 承接，本刀不重复建）
3. 建议仅在评分差超阈值时出现且可一键采纳/忽略 ← B+C（ScoreGap≥0.25 + Apply/Ignore）
4. 本地引擎出现在建议候选中（有真实被采纳样本） ← B 同池 + 真机验收时人工采纳一次留档

## 4. 验证

- 定向：`go test ./internal/routesuggest/ ./internal/modelengine/ ./internal/ai/ -count=1`；`go test ./internal/app/ -run 'Route|Feature' -count=1`；`cd frontend && npx vitest run src/pages/modelcenter/AttributionSection.test.tsx`
- 收口（主代理）：`go run ./scripts/gen_bindings` → bindingNames/drift（684→688）→ spaceBindings 锁 506→510 → ci.ps1 全量 → 发版 v4.316.0 全流程。
