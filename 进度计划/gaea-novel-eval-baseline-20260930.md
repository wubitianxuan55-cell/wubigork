# 长篇刀7：评测基线（v4.435.0，七刀收官）

> 用户指令「继续」（次刀序开刀7，长篇七刀收官）。规格依据长篇规格 §4（评测闭环）
> 与刀表刀7 行「指标快照+回归集+A/B——让前六刀『变好』可证」。按 §4 原则裁剪为
> 单刀可完成的最小闭环：**确定性指标聚合快照 + 基线落盘可比 + 逐指标 Δ**。

## 落地

### 指标快照 `NovelEvalSnapshot(persist bool)`（零 LLM，全部确定性聚合）
单次调用聚合既有指标为一份 JSON：
- **文本层**：全书 AI 味（每章 `ScoreTextNoRef` → mean/max/P90/最差章号）+ 每章
  确定性质量信号计数（`ChapterQualityIssues` S1/S2/S3 汇总与 worst 章）。
- **伏笔**：`LintForeshadows` 在产报告 → items/planted/hinted/revealed/
  回收率（revealed/items）+ findings 计数。
- **结构**：`NovelStoryHealth`（刀3）findings 计数（按 code 分桶）。
- **文风**：有 fingerprint 参考档时全书 `Delta`（无档标 skipped）。
- **上下文**：刀6 `NovelContextInventory` 合计 rune（prompt 体积基线）。
- **元数据**：章数/非空白字数/`promptSetHash`（prompts/*.json 内容 sha256，
  按文件名稳定序）——**任一哈希变更即 stale，禁止直接对比**（§4 原则）。

### 基线与历史（`eval/` 目录）
- `eval/baseline.json`：当前基线（NovelEvalBaselineSet 把最近一次快照设为基线）。
- `eval/snapshots/<RFC3339Compact>.json`：persist=true 时留史。
- **确定性验收**：同一状态两次产出**除时间戳（仅文件名）外逐字节相同**——
  body 不含时钟，全部稳定序聚合。

### Δ 对比 `NovelEvalCompare()`（A/B 最小形态）
最近快照 vs 基线：逐数值指标给 Δ 与方向（better/worse/flat——按指标语义定方向，
如 AI 味降=better、回收率升=better）；promptSetHash 或模型档不一致 → 整份标
`stale`（如实拒绝对比语义）。真 A/B 双生成对照留池。

### 绑定（+3）
`NovelEvalSnapshot` / `NovelEvalBaselineSet` / `NovelEvalCompare`（739→742）。
前端不做专门 UI（API/DSH 消费面，与刀6 Inventory 同口径）。

## 验收

快照确定性（两次逐字节相同）；聚合正确（造数：最差章=实际最高分章；回收率=
planted/revealed 造值）；Δ 方向正确；promptSetHash 随 prompts 变更；零章项目
诚实空快照。

## 明确不做

回归夹具 3×30 章人工病历（自建工程，独立立项）；LLM 盲评/bootstrap 置信区间；
张力曲线聚合（依赖分析 V2 覆盖率，随真机数据另刀）。

## 落地情况

（发版时回填）
