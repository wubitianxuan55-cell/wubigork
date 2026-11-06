# gaea t7 前端统一接线·收官刀（分析 V2 消费面板 + 标注高亮视图）规格书

> 2026-09-16 立项。来源：用户指令「继续」。阶段七规划 §4（docs/gaea-stage7-
> plan-2026-09.md）：「小说 t7：前端统一接线（分析 V2/伏笔面板/标注高亮/
> 重写建议的消费面）」。前置核对：伏笔面板 v4.299 已消费、重写建议 v4.305
> 已消费——**本刀清掉最后两块：分析 V2 九维零消费、标注层零消费**。
> 本刀合入即「小说·MuMu 蒸馏六域」前端接线全清（t7 收官）。
> 目标版本：v4.325.0。

## 1. 论点

后端资产早已在位而 UI 面零消费：

- **分析 V2**（v4.301）：analysis-v2.json 按章落盘九维（hooks/foreshadows/
  conflict/emotional_arc/character_states/organization_states/plot_points/
  scenes/pacing+文白比）+ scores/suggestions/summary，但 `AnalyzeChapter`
  绑定至今在 Legacy 面（Go 存在、AppBindings 不声明、前端零调用）——
  用户没有任何入口触发分析，也没有任何面看到结果。
- **标注层**（v4.306）：`NovelChapterAnnotations`（keyword→正文 rune 偏移）
  绑定已上、按需重建已接，前端零消费——「前端高亮随 t7」的欠账即本刀。

## 2. 裁决

| # | 裁决 |
|---|---|
| Q1 | 新绑定 `NovelChapterAnalysisV2(chapterNum)` 回 `types.ChapterAnalysisResult` **直连**（PromptTemplateDetail 回 prompt.Template 直连先例）；顶层 snake_case（chapter_num/analyzed_at…）如实透传，前端视图镜像声明——不为只读面板复制九维子类型树 |
| Q2 | 缺该章 V2 分析 → error「第 N 章尚未分析」（诚实拒绝；面板空态接「分析本章」按钮闭环） |
| Q3 | `AnalyzeChapter` 从 drift.ts `LegacySurfaceNames` 移入 AppBindings（novel 面）+ mock 桩——t7「接线」的字面义；其 V1 返回 wire 面板忽略（分析落盘 V2 后重拉 V2 即真相源） |
| Q4 | 标注高亮 V1 = **面板内只读高亮视图**（正文 prop 由 CreatePage 传入，rune 偏移→code-unit 换算后按段 `<mark>` 渲染），不动编辑器 textarea（overlay 高亮对 antd 样式同步脆弱，观察池） |
| Q5 | 高亮段：跳过 pos<0/length≤0；按 pos 排序；越界/交叠裁剪（后续段起点钳到前段终点）；rune→code-unit 换算（代理对按 codePoint 步进，EditorPanel toRune 逆函数） |
| Q6 | type 徽标配色：hook=blue/foreshadow=purple/plot_point=green/conflict=red/character=orange/suggestion=default（antd Tag 色名，非 hex） |
| Q7 | 「分析本章」按钮调 AnalyzeChapter（LLM 慢路径 loading 态），完成后重拉 V2+标注；触发即顺带伏笔同步/记忆回填（既有 agent 链路，本刀零新后端逻辑） |

## 3. 线 B：internal/app/analysis_handler.go +1 绑定

```go
// NovelChapterAnalysisV2 读取该章 V2 分析（analysis-v2.json 该章条目直连，
// types.ChapterAnalysisResult 透传）。无该章条目 error（前端空态引导先分析）。
func (a *writingState) NovelChapterAnalysisV2(chapterNum int) (types.ChapterAnalysisResult, error)
```

NovelB 门面 +1 委托（字典序插在 NovelChapterAnnotations 后）。绑定面 703→704。

测试：analysis-v2.json fixture（novel_rewrite_handler_test 的 Suggestions
用例同款落盘手法）——命中返回九维+meta；无条目报错；无项目报错。

## 4. 线 C：前端

- **bridge/novel.ts**：AppBindings +`AnalyzeChapter(chapterNum)`（出 Legacy）
  +`NovelChapterAnalysisV2(chapterNum): Promise<ChapterAnalysisV2View>`；
  视图类型 ChapterAnalysisV2View（meta snake 顶层 + Result 九维镜像，全字段
  可缺省防御）。
- **mock/novel.ts**：+2 桩（AnalyzeChapter 短延迟回 V1 形状；V2 回单章样本
  含各维一行）。
- **新组件 `ChapterAnalysisPanel.tsx`**（antd Modal，CreatePage rail 入口）：
  - 打开即拉 V2；缺档 → Alert 引导 +「分析本章」按钮（loading→重拉）
  - 头部：overall 大数 + pacing/engagement/coherence 三小分 + 评分理由；
    chips 行（节奏 pacing/plotStage/对话占比/叙述占比）+ meta（analyzed_at·engine）
  - 分节：钩子/伏笔命中（planted·resolved 徽标）/冲突/情感/角色变化/
    组织变化（稀疏可空）/情节推进/场景/建议/小结——每节计数、空节隐藏
  - 标注区 Segmented「列表 | 高亮视图」：列表=type 徽标行+importance；
    高亮视图=只读 pre-wrap 正文 + `<mark>` 段（Q5 换算/裁剪）+ 点击列表行
    scrollIntoView 对应锚点
- **CreatePage**：rail「章节分析」按钮 + 挂载（content/activeChapterNum 传入）。
- **spaceBindings**：NovelChapterAnalysisV2=play（锁 525→526）；
  bindingNames 再生（704）。
- 测试：ChapterAnalysisPanel.test（拉取渲染九维/空态+分析按钮闭环/高亮段
  换算与交叠裁剪/列表点击滚动锚点/mock 契约）。

## 5. 验收与门禁

- Go：新绑定三例（命中/缺章/无项目）；`go build ./...`+`go vet`；
- 前端：tsc -b 零错、eslint 足迹零告警、面板测试全绿、vitest 全量、
  drift OK@704、计数锁 526；
- 出口：分析本章按钮→V2 落盘→面板九维可见→标注高亮视图可见锚点可达。

## 6. 出口对照（t7 收官判据）

- [ ] 分析 V2 九维有消费面（面板）；
- [ ] 标注层有消费面（列表+只读高亮+锚点）；
- [ ] AnalyzeChapter 出 Legacy 面（AppBindings 可调+mock）；
- [ ] 伏笔面板/重写建议既有消费面零回归（既有测试零改动绿）。

## 7. 观察池（本刀不做）

编辑器 textarea 内 overlay 持久高亮（antd 样式同步脆弱）；标注点击定位到
编辑器光标（需编辑器 ref 联动）；多章对比视图；情感曲线图形化（V1 文本档）；
V1 wire AnalyzeChapter 返回形状的前端消费。
