# 小说板块观察池清账刀（v4.429.0）

> 用户指令「继续优化 gaea，本次完成小说板块」。对齐 v4.427 原罪观察池清账刀方法论：
> 逐项读码定性 → 可修的修、误报的关账、需拍板的留池。上承 `gaea-novel-audit-20260929.md`
> §4.1 观察池 19 项（前端 9 + Go 10）。

## 1. 定性结论总表

### 前端 9 项

| # | 观察项 | 定性 | 处置 |
|---|---|---|---|
| 1 | 局部重写坐标错位（±50 rune 模糊重锚静默接受） | 实锤：`internal/rewrite/partial.go:58-75` 偏移在窗内即静默重锚，作者不知选区已被挪 | **诚实化**：ResolveSelection 返回 reanchored，响应透传，前端提示「选区已按已保存正文重新对齐」；契约级根治（前端正文直传定位）另刀 |
| 2 | 设定 Agent 回填覆盖在途键入 | **辨析后实锤**：ChatPanel 输入区 `disabled={loading}` 已禁（不可达），但设定**编辑器** textarea 不禁——等待期手改正文后 `setContent(result.worldview)` 照样覆盖 | **修**：回填前比对 content 快照，已变则不覆盖，worldview 拼入回复走既有手动应用路径 |
| 3 | BookHealthPanel 情感曲线无 token | 实锤：`buildCurve` 逐章 await，重跑体检/切书后旧循环仍 `setCurve` | **修**：seq 守卫 |
| 4 | 伏笔「排队×失败重读×切书」交错窗口 | **实锤（读码定性，无需延迟注入）**：op1 失败→`loadRef` 重读把 `itemsRef.current` 覆盖成磁盘；op2 链任务随后执行写的是 `itemsRef.current`（=磁盘内容）——op2 改动**静默丢失** | **修**：generation 计数，失败重读后旧代链任务跳过写+如实提示 |
| 5 | 重写历史面板跨书残留 | 实锤：props 只有 chapterNum，切书同章号残留旧书版本列表 | **修**：refresh 依赖加 projectPath |
| 6 | Cmd+K 接受静默不写入 | 实锤：`ChapterEditor.tsx:399-407` 用 `document.activeElement`，CommandBar 抢焦点后非 TEXTAREA 即静默丢 | **修**：打开时记来源 textarea ref，accept 用它，无目标如实提示 |
| 7 | 停止生成在连接建立阶段显示「生成失败」 | **实锤（后端根因找到）**：`create_chapter_handler.go:534-538` ChatStream 建连时 ctx 已取消 → err=ctx.Err() 走 `emit error` 分支 | **修（Go）**：err 分支先判 ctx.Err() → 走 saveCancelledPartial（落盘+cancelled 事件） |
| 8 | 生成中切项目后端继续写旧书 | **关账**：前端 A3 闸门已闭环（切书 effect CancelCreateChapter+requestedPath 事件忽略）；后端在途生成持旧 pm 写旧书在 cancel 到达即停，写盘 G1 已定点合并不写坏 | 落档关账 |
| 9 | httpbridge 跨客户端并发面 | **关账**：127.0.0.1 only（LAN 已收口）；本机面=DSH headless 流水线消费属刻意设计；唯一实证丢更新点（plans.json 无锁）本刀 G-F 加锁 | 落档关账 |

### Go 10 项

| # | 观察项 | 定性 | 处置 |
|---|---|---|---|
| 1 | N5 分支摘要混入 | 实锤：`ReadAllChapterSummaries` 收全部 `NNN{a,b,c}-summary.json`；`outline.go:135`（大纲续写）把分支剧情当主线前情注入。plot_branch（分支浏览器）收分支是**特性**；memory 分支取主章号进事件记忆是语义模糊面 | **修**：加 `ReadMainlineChapterSummaries()`，outline-continue 改用；memory/plot_branch 维持（语义留池） |
| 2 | N9 ForEachChapter 缺口即停 | **定性修正**：ForEachChapter 生产**零调用**（仅测试）；实际影响面=三处内联循环（`countWrittenChapters` / `RunBookHealthCheck` / `novel_import_ai.go:315` 反推） | **修**：四处的上界改磁盘最大章号（`MaxChapterBodyNum`），缺口跳过续扫 |
| 3 | N12 迁移跳章仍落 v4 标记 | 实锤：Sscanf 失败/读失败 continue，`finalizeV4Migration` 照落——未迁移章从此读不到（IsV4 后优先 v4 视图） | **修**：收集未迁移章，非空返回错误列明细且**不落标记** |
| 4 | N13 嵌套大纲节点不可见 | 实锤：`gateOutlineIssues` 两段匹配只遍历顶层（同文件 `gateOutlineTitle` 却递归）；`findOutlineNodeByNum` 不递归——分卷大纲（Children 嵌套）下写前闸/意图注入失明 | **修**：两处递归 |
| 5 | G10 SceneRefs 零写零读 | 确认死契约 | **留池**：刀2 场景卡启用时消费 |
| 6 | G11 chapter-generate.json 无消费点 | 实锤：生产唯一真身=`create-chapter`；chapter-generate 仅 prompt_test 引用；磁盘与 `main.go //go:embed all:prompts` 双入口都在 | **修**：删文件（embed 随编译期消失）+ 测试改引用 create-chapter；工坊清单自动少一行 |
| 7 | NovelChapterPlanSave 无锁整表读改写 | 实锤：`novel_plan_handler.go:94` 单写者，但 httpbridge 本机面可二写者 | **修**：writingState 加 `chapterPlanMu` 持锁贯穿读-改-写 |
| 8 | 零调用者导出绑定口径 11 vs 36 | 口径差=是否含「wailsjs 生成物零 import」宽口径 | **落档**：不删（DSH headless 消费不可静态可见，删绑定需用户拍板） |
| 9 | scene_gen_handler 吞错 | 实锤：`markOutlineDone` `_ = pm.WriteOutlines(of)` | **修**：返回 error 上报调用方 |
| 10 | chapter-review 模板缺失恒走 fallback | 实锤：prompts/ 无该文件，`chapter.go:148` Get 恒 nil | **修**：内置 fallback 提升为 `prompts/chapter-review.json`（工坊可编辑真身） |

## 2. 明确不做（本刀）

- 局部重写契约级根治（前端缓冲正文直传定位）——绑面改造另刀。
- memory 分支摘要语义（分支剧情是否算书内事实）——需产品拍板。
- G10 SceneRefs——刀2 消费。
- 零调用者绑定删除——DSH headless 消费面不可见，需拍板。
- 刀2 场景卡与逐场景生成——长篇七刀下一刀，次刀序不变。

## 3. 验收

- Go：上述 G 项各至少一测；`internal/app`+`internal/project`+`internal/outline`+`internal/rewrite` 定向包绿；gofmt/vet/build 0。
- 前端：F 项各至少一测（小说域定向）；tsc -b 0；改动面 eslint 0。
- 门禁：`scripts/ci.ps1` 全量收口；漂移闸 OK@4.429.0（绑定面 726 零变更——全部为行为内修，无新绑定）。

## 4. 落地情况

> **已发版 v4.429.0（2026-09-29）**。观察池 19 项处置：**修 15**（前端 6 + Go 9）、
> **关账 2**（#8 OpenProject 在途生成=前端闸门已闭环；#9 httpbridge 跨客户端=本机面
> 属 DSH headless 消费刻意设计，plans 锁已缓解唯一实证丢更新点）、**留池 2**（G10
> SceneRefs 随刀2；零调用者绑定删除需用户拍板——DSH headless 消费不可静态可见）。
> memory 分支摘要语义（分支剧情是否算书内事实）作为新观察池项记录。
>
> **实现注记（新观察池项）**：微实验实证 React 受控组件下「DOM 赋值 + dispatch
> input」的 onChange 收到**旧值**（React 恢复受控值）——Cmd+K 因此改走受控数据流；
> **GhostText.tsx:89 的同款手法在生产中的可靠性存疑**（若真机补全接受不生效即此
> 因），挂池待真机验证。

### 代码面
- Go 10 文件：create_chapter_handler（取消错报+findOutlineNodeByNum 递归）、
  novel_gate_handler（gateOutlineIssues 递归）、novel_plan_handler（chapterPlanMu）、
  app.go（锁字段）、novel_foreshadow_lint_handler / novel_book_health / novel_import_ai
  （缺口续扫三处+titleByFile 递归）、scene_gen_handler（markOutlineDone 上报+递归）、
  project.go（ForEachChapter 上界扫描+ReadMainlineChapterSummaries+迁移 fail-closed）、
  outline.go（主线口径）、rewrite/partial.go（reanchored 第四返回值）、
  novel_rewrite_handler（reanchored 透传）。
- prompts：新增 chapter-review.json（内置 fallback 提升为工坊可编辑真身）；删除
  chapter-generate.json（零生产消费死资产，v4.308 台账已在册；embed 随编译期消失，
  prompt_test 引用改 create-chapter/character）。
- 前端 8 文件：NovelSettingPage（回填守卫：等待期正文已变不覆盖、修改稿附回复走
  手动应用）、BookHealthPanel（曲线代际守卫+loading 代际收口）、ForeshadowPanel
  （写链代际短路：失败重读后排队的旧代际任务跳写+如实提示）、RewriteHistoryPanel
  （projectPath 进 effect 依赖：切书同章号重拉）、ChapterEditor（Cmd+K 改受控数据流
  updateScene——DOM 注入+input 派发在受控组件下被 React 恢复受控值，实测值传不上去）、
  PartialRewriteModal（reanchored 提示 Alert）。

### 测试
- Go 新增 8 例：project 3（缺口续扫/主线过滤/迁移 fail-closed）+ app 5（连接建立阶段
  取消走 cancelled、嵌套节点两处递归、计划并发不丢更新、缺口计数）+ rewrite
  reanchored 断言并入既有 13 例 + prompt 测试改写（模板存在性含 chapter-review）。
- 前端新增 9 例：设定回填守卫 2、曲线代际 1、伏笔短路 1、重写历史切书 1、Cmd+K 2、
  reanchored 2。小说域定向 35 文件 / 312 例全绿；tsc -b 0；改动面 eslint 0 错 0 警。
- 反向验证：取消错报（去掉 ctx.Err 分支→CancelledDuringConnect 红）、缺口续扫
  （回 break→SkipsGaps 两处红）、伏笔短路（去 gen 判断→SaveForeshadows 被调 2 次红）、
  曲线守卫（去 seq 判断→svg 出现红）均按「改坏必红」原则抽样确认。

