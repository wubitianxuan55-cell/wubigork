# gaea · revolution 刀8 收官：跨模块验收 + 回退演练（规格书）

> 2026-09-17 · 用户指令「继续优化迭代 gaea」。小刀单线主代理+三线并发子代理。
> 前置核对：race 项已收口（v4.331 CI race job 六包）；本刀收掉革命规划最后
> 一项未勾——跨模块验收+回退演练，合入即 **revolution 刀 1–8 全线收官**。
> 规划权威=`docs/gaea-novel-revolution-2026.md` §5 刀8 行+验收口径。

## 0. 缺口事实（2026-09-17 摸底，全部 file:line 可查）

各模块单测在位，但**跨模块串联零覆盖**：

1. 生成主轴端到端（CreateChapter→done+aiTaste→自动门→分析 V2 落盘→记忆回填→
   标注→伏笔同步→场景物化）全仓无一个贯通测试；自动门**成功路径**
   （analysisDone=true/foreshadowSync 有值）从未被测（现有 novel_book_health_test
   只测 nil-engine 降级形态）。
2. 叙事审批链 app 级 0 测试（BuildNovelStatePatch/SettleNovelState 无任何 _test
   引用；journal 本体 Replay 有包级测试）。
3. 回退路径无测试：app 级 RestoreSnapshot（仅 snapshot 包级）；MigrateV3ToV4
   `_v3_backup` 非破坏备份零断言。
4. novelstyle 无性能钉（AI 味检测 <1s 判据）；DeSlopRewrite 无「分降且篇幅/
   实体不变」断言。

已钉不重做：POV 掩码语义（novelcontext）、人写 vs AI 方向（TestScoreText_NoRef）、
rewrite 版本库 Restore 三条 stub-LLM e2e、promptstore Reset 三层、homeLayout 守卫。

## 1. 顺手根治：版本三处漂移（独立于刀8 的新发现）

**事实**：app_info.go=4.323.0、versioninfo.rc=4,323,0,0，而 wails.json/README=4.333.0
——v4.324 起十版未跑 sync-version.ps1，exe 内嵌版本号落后十版；check-version-drift
闸门存在但不在 ci.ps1 内，从未拦截。

**修**：①`scripts/sync-version.ps1 -Version 4.334.0` 三处对齐；②ci.ps1 增加
check-version-drift.ps1 段（防再犯）；③CHANGELOG 本版条目如实记漂移区间
v4.324~v4.333（内嵌版本停 4.323.0，不影响功能面）。

## 2. 三线实现契约（文件足迹互斥）

### 线A `internal/novelstyle/acceptance_knife8_test.go`（新建）

- `TestKnife8_AiTasteScorer_PerfUnder1s`：≥5000 rune 真实感章节 ×3（句长参差/
  混合标点/对话），`ScoreTextNoRef` 单次耗时 ≤1s（判据「确定性检测<1s」）。
- `TestKnife8_DeSlopRewrite_Fidelity`：构造含已知高频词命中的 AI 味文本（含
  两个实体名+对话），`DeSlopRewrite` 后断言：分数下降；rune 长度漂移 ≤10%
  （篇幅不变）；实体名逐字保留；句数不减少（不丢内容）。

### 线B `internal/app/novel_revolution_axis_test.go`（新建，helper 前缀 axis）

- fixture `newAxisTestApp`：herdsman→httptest SSE 双路桩——请求体含
  「小说分析编辑」（analysis-chapter 系统提示词标记）回分析 JSON，否则回章节
  正文；`a.analysisAgent = analysis.New(桩client, pm, cfg, eng)`（成功路径）。
- `TestKnife8_GenerationAxis_EndToEnd`：CreateChapter（正文 ≥1200 rune、目标
  1000）→ 章文件落盘（含正文）→ waitFor 自动门过水：analysis-v2.json 有该章
  条目 / memories/001-*.json 非空 / annotations/1.json ≥1 条 pos≥0（正文埋
  8-25 字锚点保证命中）/ foreshadows.json 出现桩 JSON 的 planted 条目 /
  WriteChapter 触发 rebuildScenesFromBlob 场景物化（场景 ≥1）。
- `TestKnife8_BookHealth_FiftyChapters`：合成 50 章（干净/脏交替，脏=超长段落
  等触发 novelgate），预落部分 V2 → RunBookHealthCheck 断言 TotalChapters=50、
  聚合计数与设计一致、WorstAiTaste 落在脏章集合、AnalyzedChapters=预落数、
  全程确定性零 LLM（判据「书级重检」长文口径）。

### 线C `internal/app/novel_state_acceptance_test.go`（新建，helper 前缀 state）

- `TestKnife8_ApprovalChain_NoAutoSettleAndReplay`：落章+摘要 →
  BuildNovelStatePatch(1)（patch ID ch001-ai、upsert 来自摘要角色）→
  SettleNovelState(approved=false)：版本 0/实体空/GetNovelState 空（**AI 建议
  无一自动入库**）→ approved=true：版本 1+实体在账 → narrative.Open+Replay
  重放快照与 GetNovelState 一致（append-only 可回放）→ 坏 JSON 拒绝。
- `TestKnife8_SnapshotRestore_Drill`（app 级）：v4 工程 → 改场景 →
  CreateSnapshot → 再改 → RestoreSnapshot → 内容回到快照态。
- `TestKnife8_MigrateV3ToV4_NonDestructive`：v3 blob 章（内容 X）→
  MigrateV3ToV4 → 场景 ≥1、Stitch==X（内容无损）、`_v3_backup/` 内保留原文。

**铁律**：三线零生产代码改动（缝合面全部现成）；同包新 helper 名带前缀防撞；
各自定向 `go test` 绿 + gofmt 后交回。

## 3. 验收口径对照（revolution §5）

| 判据 | 本刀落点 |
|---|---|
| 长文一致性（死亡不复活等） | 深检纯函数既有测试 + 线B 50 章书级体检聚合 |
| AI 味方向正确 | 既有 TestScoreText_NoRef（不重做） |
| 去味分降且字数/实体不变 | 线A Fidelity |
| 作者是上帝（无一自动入库） | 线C approved=false 全链 |
| journal 可回放审计 | 线C Replay 一致性 |
| 性能 <1s / 书级窗口 | 线A Perf / 线B FiftyChapters |
| 每刀可回退 | 线C 三条回退演练 + 既有 rewrite Reset 三条 e2e（引用） |

## 4. 门禁与出口

零新绑定（707 不动）；go build/vet/test 全量、tsc -b、eslint、vitest 3185、
bindings drift OK@707、**版本三处 4.334.0（含漂移修复）**、ci.ps1 新增
version-drift 段全绿；产物 exe+SHA256SUMS+冒烟 /api/health 200。
文档=本规格书+releases/v4.334.0.md+CHANGELOG/README+AGENTS 迁1插1+revolution
刀8 勾选+progress/todos。**出口=revolution 刀 1–8 全勾，规划收官。**
