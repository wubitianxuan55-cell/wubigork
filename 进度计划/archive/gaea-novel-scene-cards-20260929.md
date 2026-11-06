# 长篇刀2：场景卡与场景级生成（v4.430.0）

> 用户指令「继续」（承接「继续优化 gaea」线程；v4.429 小说板块观察池收官后按台账
> 次刀序开刀2）。规格依据 `docs/gaea-longform-novel-system-2026-09.md` §1.3/1.4
> （生成单位错了/场景卡缺创作学核心字段）与 §5 刀表刀2 行。

## 0. 现状（读码确认）

- `SceneMeta`（types.go:246）：坐标字段齐（POV/Location/TimeOfDay/Emotion/Tags/Order/
  Status/WordCount），**创作学字段零**——能说「戏在哪谁在看」，说不了「戏要干什么、
  怎么算写成了」。
- 整章生成（CreateChapter）落 blob 后 `rebuildScenesFromBlob` 物化**单场景**——v4
  「场景制」生成侧不成立；场景数量/边界/目标全由模型即兴。
- `GenerateScene` 已有（单场景手动生成）：prompt=标题+章节号+plotReq 自由文本+场景
  圣经——**卡字段无从参与**。
- 局部重写对场景章禁（novel_rewrite_handler.go「选段与场景边界无法对齐」）；
  **整场景**重写不存在。
- `OutlineNode.SceneRefs` 零写零读（G10，本刀启用）。
- 前端 ChapterEditor：场景多框编辑/ⓘ 元数据弹窗（SaveSceneMeta）/单场景 AI 生成/
  重排均在。

## 1. 落地

### 线1 数据契约
- `SceneMeta` 扩 6 字段（全 omitempty，旧档零迁移）：`Goal`（这场戏要什么）/
  `Conflict`（谁·什么阻挡）/`Turn`（价值翻转）/`Outcome`（结果，必须改变状态）/
  `Sequel`（余波 reaction·dilemma·decision）/`ExitHook`（退出钩子）。
- `OutlineNode.SceneRefs` 接线：逐场景生成完成时按章号读-改-写回写场景 ID 列表
  （定点合并纪律=v4.425 G1）。

### 线2 生成链（Go）
- **卡注入**：`GenerateScene` prompt 升级——卡字段非空注入结构化区段（场景目标/
  冲突/转折/结果义务），并注入**前一场衔接**（前场景 Outcome+ExitHook）；
  无卡字段时 prompt 与现状逐字节一致（旧场景零回归）。
- **整章逐场景生成**（新绑定 `NovelChapterScenesGenerate(chapterNum, allowMissingCard)`）：
  按 Order 逐场景调生成内核，每场完成即落盘+emit `scene-gen-stream` 进度；
  **写前闸**=卡缺 Goal 或 Conflict 的场景拒绝并点名（allowMissingCard 显式跳过，
  刀1 覆盖语义复刻；手动单场景 GenerateScene 不上闸=兼容）；全部完成 Stitch→
  WriteChapter blob（读路径兼容）+ SceneRefs 回写；复用 chapterGenMu 同章互斥与
  CancelCreateChapter 取消。
- **场景级重写解禁**（新绑定 `NovelSceneRewrite(chapterNum, sceneID, instruction)`）：
  单场景 whole 重写（快照先行）+版本库留痕（mode=scene）；他场与正稿不动。
  局部重写仍禁场景章（选段→场景映射维持观察池）。
- **场景卡 AI 提案**（新绑定 `NovelSceneCardsPropose(chapterNum)`）：章计划+大纲
  节点+前章摘要 → 2-5 张卡骨架；**提案不落盘**（确认红线同族），前端审批后走既有
  CreateScene+SaveSceneMeta 落卡。

### 线3 绑定
- 新增 3（NovelChapterScenesGenerate / NovelSceneRewrite / NovelSceneCardsPropose），
  绑定面 726→729；gen_bindings+bindingNames+bridge/spaceBindings 四处同步。

### 线4 前端（ChapterEditor 为主）
- ⓘ 弹窗扩 6 卡字段；「按场景卡生成全章」入口（确认弹窗列卡齐备度，缺卡点名，
  可勾跳过）+逐场景进度消费；右键菜单「重写本场景」；「AI 拆场景卡」提案审批弹窗
  （可编辑列表，确认逐张落卡）。

## 2. 验收

- 一章多场景按卡生成（非单场景物化）；卡字段进 prompt（无卡时零回归）；
  缺卡拒绝可覆盖；单场景重写他场字节不动；SceneRefs 回写；同章互斥+取消语义；
  旧档零迁移。测试：Go 每新面至少一锚+前端新入口各一锚；全量 ci 收口。

## 3. 明确不做（本刀）

- 局部重写解禁（选段→场景映射）——维持观察池。
- 电荷翻转/义务绑定硬闸完整形态——随刀3/刀4。
- maxContinues 废止——整章生成路径既有机制不动，逐场景路径天然不需要。

## 4. 落地情况

> **已发版 v4.430.0（2026-09-29）**。三验收点全过（多场景非单场物化/卡字段进
> prompt〔HTTP body 断言〕/单场景重写他场字节不动）。Go 8 文件（scene_cards_handler
> 新增+scene_gen_handler 抽共享内核+chapter_handler 白名单扩+bindings 门面+types 两处）
> + 前端 4 文件 + wailsjs 生成物；**绑定面 726→729、spaceBindings 550→553**
> （四处同步）。测试 Go 5 新例（含关闸反向验证）+ 前端 4 新例；小说域 35 文件 /
> 316 例；全量 ci.ps1 CI OK。坑：bash 内联 node 的模板字符串美元花括号被吞（须
> .mjs 文件化）；CRLF 文件的 python LF 锚不匹配（split/join 或 Edit 工具）；
> antd Modal.confirm 同 title 双节点（title+confirm-title 两层都渲染，断言用
> findAllByText）；SaveSceneMeta 白名单 patch 用指针区分「未传/传空」防旧调用方
> 抹卡；抽共享内核后 GenerateScene 传 nil ctx 撞 SA1012（调用方传
> context.Background，内核 nil 兜底仅作防御）。
