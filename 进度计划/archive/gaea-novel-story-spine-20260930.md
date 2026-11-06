# 长篇刀3：故事层骨架（v4.431.0）

> 用户指令「继续」（承接长篇七刀线程；v4.430 刀2 收官后按台账次刀序开刀3）。
> 规格依据 `docs/gaea-longform-novel-system-2026-09.md` §1.10（整书层没有任何持续
> 状态）/§2.1（M3 弧线水位/M5 节拍/M6 线程 MICE/M8 问题池/M12 主题论证——**M12
> 只出报告不做闸门**）/§2.2 故事层持有物与 §5 刀表刀3 行。

## 0. 范围裁决（刀表 → 本刀落地形态）

刀3 行：主题论证 + 人物弧线节点 + 节拍位 + 支线开关 + 未解问题池；每 10 章结构体检。
落地为**一个结构化整书层契约 + 确定性体检 + 生成注入切片 + 编辑面板**：

### 数据契约（线1）
新 `story_spine.json`（project 层，与 outline.json 平级）：
- `Theme`：控制理念（controlling_idea）+ 反论最强陈述（counter_idea）——M12 报告制。
- `Arcs`：人物弧线——`want/need/misbelief` + 水位点列表（chapter + stage
  〔setup 反证递进 proof 行动推翻〕+ note，M3 的可编辑形态；水位单调由体检提示不做闸）。
- `Beats`：节拍位——hook/first_plot_point/midpoint/second_plot_point/climax + 自定义；
  每项绑章号（0=未定）与 status（planned/hit）。
- `Threads`：支线开关表（M6 简化）——`mice_type`（query/character/event/place/thing）+
  `open_chapter`/`last_advanced_chapter`/`status`（active/parked/closed/abandoned）+
  `close_chapter`。
- `OpenQuestions`：未解问题池（M8 简化）——question/raised_chapter/status/answer_chapter。

### 体检（线2，确定性零 LLM：`NovelStoryHealth`）
输入=spine+磁盘最大章号+章分析 V2 情感强度（有则用，无则该维度如实标「缺数据」）：
1. **支线遗忘**：active/parked 线程 `当前章-最后推进章 > 10` → 推进或显式改状态。
2. **未解问题超期**：open 问题超 15 章未答 → 回答或显式关闭。
3. **节拍滞后**：planned 节拍绑定章号 + 容差 3 < 当前章 → 补标 hit/改绑。
4. **中段塌陷**：分析 V2 强度中段（30%~70% 章区间）均值 < 全书均值×0.75 → 定位区间。
5. **弧线水位覆盖**：arc 有节点但最后一个水位点章号 < 当前进度 1/3 之前 → 提示补。
6. **终局未闭合**：当前章进入最后 15%（按 climax 节拍或最大章估计）仍有 active
   线程/open 问题 → 收束清单。
7. **主题缺失**：controlling_idea 空 → 补主题（报告，不阻断）。
输出=分维度 findings（code/message/severity/chapter 定位）+ 汇总计数；零 spine=空报告
不算错（新书正常态）。

### 生成注入（线3）
`CreateChapter` 与 `NovelChapterScenesGenerate` 注入「故事层切片」区段：
- 本章绑定的节拍位（「本章是 midpoint：价值翻转的枢纽章」）；
- 活跃线程名单（要推进/可收束）；
- open 问题（可释放的悬念素材）；
- 主角弧线最近水位点（misbelief 当前阶段的行动口径）。
无 spine 或全空 → 零注入（零回归）。

### 绑定（线4，+3）
`NovelStorySpineGet` / `NovelStorySpineSave`（整表替换+确定性校验：章号非负、
status 枚举）/ `NovelStoryHealth`；另 `NovelStorySpinePropose`（AI 从 StoryThread+
大纲+前章摘要提炼主题/弧线/节拍初稿，**提案不落盘**确认制）——共 +4。

### 前端（线5）
CreatePage 右栏新面板「故事骨架」：主题两行 + 四张可编辑表（弧线/节拍/线程/问题池）
+「AI 提炼骨架」提案审批 +「结构体检」按钮与报告渲染。

## 1. 验收

- spine 落盘往返+校验拒绝矩阵；体检各维度造数定位（遗忘支线/超期问题/滞后节拍/
  塌陷中段/终局未闭合）；生成注入含切片（HTTP body 断言）且空 spine 零注入；
  提案不落盘；绑定 729→733 四处同步。

## 2. 明确不做（本刀）

- M7 伏笔池（既有伏笔体系在产，不并入 spine）；M8 认知三表完整形态；
  M1/M2 场景层硬闸（刀2 已留观察池）；弧线水位**闸门**（只体检提示）；
  因果 DAG/时间线台账（M9，后续刀）；体检自动触发（按需按钮，与全书体检口径一致）。

## 3. 落地情况

> **已发版 v4.431.0（2026-09-30）**。Go 新 3 文件（types/story_spine.go+
> project/story_spine_store.go+app/story_spine_handler.go）+ 注入两处接线
> （create_chapter_handler/scene_cards_handler）+ 前端 StorySpinePanel 新组件 +
> CreatePage rail 挂载；**绑定面 729→733、spaceBindings 553→557**。测试 Go 4 新例
> （含体检阈值反向验证）+ 前端 5 新例；小说域 36 文件 / 321 例；全量 ci.ps1 CI OK。
