# 小说板块优化批 2：未保存保护收口 · 跨书失效治理 · 竞态守卫——规格

> 立项来源=用户指令「继续优化小说板块」（拍板：**审计优化批**，对齐 v4.421/v4.423 方法论）。
> 定刀依据=**四线并行只读审计**（前端创作/阅读流程 10 条 · 前端设定/书架/导入导出 13 条 ·
> Go 后端 1 条线 · 计划台账对账 18 条抽验）+ 主代理读码复核。
> 目标版本：v4.425.0（v4.424.0 已由并行会话「原罪提示词批」占号）。

## 0. 论点（为什么是这些）

v4.421.0 把小说板块的**页内**未保存保护做齐了（unsavedGuard 原语 + 切章/生成/删除/重写），
但审计确认还有三个系统性缺口，全部由读码确证：

1. **未保存保护只覆盖「页内动作」，不覆盖「服务端动作 → 前端重载」这条反向路径**。
   `PartialRewriteModal`（局部重写）与 `RewriteHistoryPanel`（重写历史应用/恢复原文）在
   服务端写盘成功后**无条件** `loadChapter()` 把磁盘正文灌回编辑区，同时把
   `loadedSnapshot` 对齐 → 作者未落盘的手写正文被覆盖**且脏标志被抹掉**（此后再切章/关
   标签都不再提示）。同文件的 `refreshEditorAfterServerRewrite` 对同类场景**有**护栏
   （脏时只提示不刷新）——说明这两处是漏网点，不是有意设计。`CreatePage.tsx:1108`、
   `:1089`、`ChapterPage.tsx:993`。
2. **窗口/面板持有跨书状态，切书后不失效**。v4.421 只修了 `ForeshadowPanel`/
   `ConsistencyPanel` 两处 `load`。审计确认同类漏网至少六处：
   `WorldviewSectionsEditor`（**会把 A 书六维世界观写进 B 书**，P0 级）、`NovelSettingPage`
   的 markdown 缓冲（静默丢、连提示都没有）、`ChapterAnalysisPanel` 主 load、
   `ForeshadowPanel.lintReport`、`BookHealthPanel`、CreatePage 持有的
   `reviewReport`/`fpStatus`。
3. **异步响应缺少「还属于当前上下文吗」的守卫**。创作页 `loadChapter` 有
   token+projectPath 双守卫、阅读页搜索有 `searchSeqRef`，但**生成流事件**
   （`runGeneration` 的 `done/cancelled`）、章节载入在途期的保存、阅读页摘要/问书
   都没有。后果最重的一条是**生成中切项目**：旧书生成完成的事件会把正文挂到新书同号章
   并标「已保存」，点保存即写进新书文件。

体验/正确性面的连带项：生成中 rail 的去味/整章重写未禁用（与流式生成并发写同一章）、
生成中仍能拉剧情向导且点「生成」静默无响应、导出全格式失败仍弹成功、新角色弹窗未按
`active` 门控（会在别的子页抢焦点）。

## 1. 线与足迹（互斥）

| 线 | 负责人 | 文件足迹 | 项 |
|----|--------|----------|-----|
| 线1 共享原语 | 主代理 | `frontend/src/lib/boardActive.ts`(+test)、`components/novel/novelSwitchGuard.ts`(+test)、`layouts/MainLayout.tsx`、`pages/NovelPage.tsx`、`pages/HomePage.tsx`(+test)、`NovelPage.test.tsx` | M1 · M2 · B13 · B9 接线 |
| 线2 创作页 | 子代理 A | `pages/CreatePage.tsx`(+test)、`components/novel/create/NewCharactersModal.tsx`(+test) | A1 · A2a · A3 · A4 · A6 · A7 · A8 · B5a |
| 线3 阅读页 | 子代理 B | `pages/ChapterPage.tsx`(+test)、`pages/chapter/**` | A2b · A9 · A10 |
| 线4 设定页与面板群 | 子代理 C | `pages/NovelSettingPage.tsx`(+test)、`components/novel/{WorldviewSectionsEditor,ChapterAnalysisPanel,ForeshadowPanel,BookHealthPanel,ChapterReviewPanel,StyleFingerprintPanel,RewriteHistoryPanel,NovelInspector,ExportPanel,PromptWorkshopPanel,BookSearchEnginesModal,BookSearchModal}.tsx`(+tests) | B1 · B2 · B3 · B4 · B5b · B6 · B7 · B8 · B9 · B10 · B11 · B12 · M4 |
| 线5 Go 正确性 | 子代理 D | `internal/app/**`、`internal/project/**`（详见 §1.5） | G 系列 |
| 线6 订阅纪律 | 主代理 | `components/TTSPlayer.tsx`、`pages/CharacterLibraryPage.tsx` | M3 |

**跨线契约（不得单方改动）**

1. `lib/boardActive.ts`：`notifyBoardActive(page: BoardId)` / `useBoardActive(page)` /
   `isBoardActive(page)` / `resetBoardActive()`（测试用）。**未收到通知时恒 true**（兼容
   单独渲染页面/既有测试）。壳层 `MainLayout` 在 `page` 变化时调用；
   `NovelPage` 用 `useBoardActive('novel')` 与子页 `active` 取 AND。
2. `components/novel/novelSwitchGuard.ts`：
   - 登记端：`registerNovelDirtyProvider({ id, label(), dirty(), canSave?(), save?() })`
     返回注销函数，在 `useEffect` 里 `return` 它即自动注销。
   - 消费端：`takeDiscardConfirmed(consumerId: string): boolean` —— 书架闸门确认「放弃
     修改」后，各页自己的兜底提示据此避免二次惊吓（**每个 consumerId 每次确认只返回一次
     true**；`consumerId` 与登记 id 同名最省事）。
   - 发起端：`guardNovelSwitch(proceed, targetLabel?)` 放在**真正切书动作之前**，
     返回 `true`＝已拦截（勿再执行 proceed）。
3. `BookSearchModal` 新增可选 `onShelfReconciled?: () => void`：兜底对账命中书架时调用，
   由 `HomePage` 传 `() => { void loadProjects() }`（书架刷新）。

## 2. 项清单（编号 = 审计报告编号）

### 线1（主代理）
- **M1 板块级可见信号**：新增 `lib/boardActive.ts`；`MainLayout` 在 `page` 变化时
  `notifyBoardActive(page)`；`NovelPage` 的 `active` 改为 `activeTab===t.key && boardVisible`。
  修「进过小说板块后切到别的板块，隐藏的阅读页仍吞 Ctrl+S / F11（`preventDefault` 让
  浏览器全屏失效）」。
- **M2 跨页切书确认**：新增 `novelSwitchGuard.ts`；`HomePage` 四处 `openProject`
  调用点（新建/导入/在线导入完成/打开）全部走 `guardNovelSwitch`；CreatePage 与
  ChapterPage 各登记一个脏探针。三选（先保存/放弃修改/取消）仅当**所有**脏持有者都能
  保存；否则两选。阅读页多标签脏且非当前标签时 `canSave=false`（不假装能一次存齐）。
- **B13 删除正在编辑的书 → 关闭项目**：`handleDelete` 成功后若 `card.path === projectPath`
  调 `useAppStore.getState().closeProject()`（后端已 `closePM`，前端 store 仍在挂）。
- **B9 接线**：给 `BookSearchModal` 传 `onShelfReconciled`。

### 线2（创作页）
- **A1（P0）局部重写应用后覆盖未保存正文**：`onApplied` 前过脏闸——脏则**不重载**并如实
  提示「重写已在服务端生效；编辑区有未保存修改，已保留本地版本，未刷新」；同时在
  **打开**局部重写前也过同一道闸（与 `openRewriteModal` 同款）。对齐既有
  `refreshEditorAfterServerRewrite` 口径。
- **A2a（P1）重写历史应用/恢复原文不问脏**：CreatePage 的 `RewriteHistoryPanel.onApplied`
  同 A1；「重写历史」入口按钮同样先过闸。
- **A3（P1）生成中切项目 → 流事件跨书污染**：`runGeneration` 开头记录 `requestedPath`；
  事件回调（`chunk`/`done`/`cancelled`/`error`）与 `useAppStore.getState().projectPath`
  不等则**整体忽略**（并如实提示该次生成已因切书失效）；`projectPath` 变化时对在途生成调
  `CancelCreateChapter` + `finishStream`（不要留下 `generating=true` 悬挂）。
- **A4（P1）载入在途点保存写错章**：`saveActive` 增加在途早退（载入未完成时保存给
  `message.warning` 并返回 false）；或 `loadChapter` await 前清空 `content` 缓冲。二者取
  其一即可，但必须让「第 5 章正文写进第 6 章」不可达。
- **A6（P2）生成中 rail 去味/整章重写未禁用**：rail 的「一键去味/高级去味/整章重写」
  与树里的「重新生成」按 `generating` 禁用（`EditorPanel` 的「局部重写」已是先例）。
- **A7（P2）生成中可拉剧情向导且点生成静默无响应**：`openWizard` 首行按
  `generatingRef` 早退并给 `message.warning`（对齐 `selectChapter` 文案口径）；
  `startGeneration` 的在途早退也要有可见提示，不再静默 return。
- **A8（P2）新角色弹窗未按 active 门控**：`NewCharactersModal` 接收 `active`；
  `active=false` 时事件**挂起**（置 pending），切回创作 tab 再弹（不能直接丢弃）。
- **B5a**：切项目 effect 里一并清 `reviewReport`/`fpStatus`/`fpMsg` 与相关面板开关。

### 线3（阅读页）
- **A2b（P1）重写历史应用/恢复原文不问脏**：`RewriteHistoryPanel.onApplied` 前过脏闸
  （对齐 CreatePage 口径：脏则不重载 + 如实提示）。
- **A9（P2）摘要/问书迟到响应错章**：`runSummary`/`runAsk` 加 seq 守卫（发起时快照
  `readNodeId`，回填前校验仍是同一章）；`summaryCache` 写入也要按发起章号。
- **A10（P3）载入期间输入被覆盖**：载入在途时该 tab 的编辑区只读/loading（或载入完成时
  若已有作者输入则保留输入并提示），不得静默覆盖后还标 `saved=true`。

### 线4（设定页与面板群）
- **B1（P0）维度化编辑器不随 projectPath 重载**：`WorldviewSectionsEditor.load` 依赖
  `projectPath`，无项目走引导态（对齐 `ForeshadowPanel` 先例）。
- **B2（P1）设定页切书静默丢未保存设定**：`loadContent` 命中 projectPath 变化且脏时
  经 `novelSwitchGuard` 登记（id=`settings-worldview`）+ 兜底 `takeDiscardConfirmed`；
  至少给 `message.warning`（对齐 CreatePage 口径）。
- **B3（P2）章节分析面板主 load 无守卫**：加 `loadSeqRef` token + effect 依赖
  `projectPath`（对比章的 `cmpSeqRef` 已是先例）。
- **B4（P2）伏笔体检报告不随切书清除**：`useEffect(() => setLintReport(null), [projectPath])`。
- **B5b**：`BookHealthPanel`/`ChapterReviewPanel`/`StyleFingerprintPanel` 的报告态按
  `projectPath` 失效（切书即清，或关弹窗）。
- **B6（P2）导出全失败仍弹成功**：统计 `values.filter(v => v.startsWith('失败'))`，
  非 0 改 `message.error`（部分失败文案要如实）。
- **B7（P2）提示词工坊关闭不判脏**：`onCancel` 包脏闸（`confirmDiscard`）；重开清
  `detail` 后 `dirty` 恒 false 的漏洞一并堵（保留未保存草稿或明确提示已丢弃）。
- **B8（P2）引擎规则编辑器关闭丢编辑**：`rows` 与首次 `reload` 快照比对得 dirty，
  `onCancel` 走 `confirmDiscard`；`saving` 期间禁止关闭。
- **B9**：兜底对账命中时调 `onShelfReconciled`（契约见 §1.3）。
- **B10（P3）体检失败渲染成空态**：区分 `error` 与「无数据」，给重试。
- **B11（P3）重写历史详情失败永久转圈**：记录 `detailErrors[id]`，渲染错误 + 重试。
- **B12（P3）检查器章节体检挂错章**：回填前校验 `chapterNum` 未变。
- **M4**：`ForeshadowPanel` 补「在途禁用」（v4.421 台账声称已落地，实际只有串行队列无
  禁用；`savingWrite` 需绑到按钮 `disabled`）。

### 线5（Go）——见 §3（待 Go 审计线回报后定稿）

### 线6（主代理）
- **M3 订阅纪律**：`TTSPlayer.tsx:154`、`CharacterLibraryPage.tsx:251` 的
  **整通道裸 `EventsOff`**（v4.62.2 事故纪律禁止形态）改走 `subscribeWailsEvent`
  唯一入口。台账错记「已清零」，本刀补账。

## 3. Go 侧（线5）

待 Go 审计线回报后填入。台账对账线已确证 v4.421 观察池 Go 余 9 项**全部仍开**，其中
与本刀「不静默丢、不静默假成功」同一叙事且成本低的四项优先：

1. 残稿文件名秒级时间戳（同秒两次取消覆盖前一个残稿）→ 唯一化。
2. 生成后协程未托管（`create_chapter_handler.go:707/712` 裸 `go`）。
3. 建章忽略 `WriteOutlines` 错误（`:748/:785/:794`）。
4. `ForEachChapter` 缺口即停（`project.go:625-629`）。

## 4. 明确不做（本刀）

- 刀2 场景卡与逐场景生成（架构级，另刀）。
- 跨章去重第二来源（需带章号的批量摘要读接口，随刀2）。
- Go 其余观察池项（搜索标题命中抑制正文扫描 / 书架全库全章 IO / 迁移跳章仍落 v4 标记 /
  嵌套大纲节点不可见 / 11 个零调用者导出绑定）——影响面小或需产品拍板。
- `EditorPanel` rune↔code-unit 换算收敛（高亮核心偏移数学，回归风险 > 收益）。
- 局部重写「选区坐标取自缓冲、后端按磁盘定位」的错位（需改后端定位契约，另刀；
  本刀只在 UI 侧把它降级为**先保存再重写**的引导）。
- 台账路径类笔误（`frontend/src/wailsjs/...` 实为 `frontend/wailsjs/...`）随本刀文档一并校正。

### 4.1 观察池（本刀新增，均为审计实证但本刀不做）

**前端**
1. 局部重写**坐标错位**未根治：选区来自编辑器缓冲（`EditorPanel` 由 `content` 换算 rune），
   后端按**磁盘**正文定位（`internal/app/novel_rewrite_handler.go:158-163`），偏离 >50 rune
   时要么报「选中的文本与章节内容不匹配」要么被 ±50 模糊重锚静默接受
   （`internal/rewrite/partial.go:58-75`）。本刀只在 UI 侧引导「先保存」。
2. 设定 Agent 回填覆盖在途键入：`NovelSettingPage.handleChatSend` await 后
   `setContent(result.worldview)`；需先核 `ChatPanel` 在等待期是否禁输入。
3. `BookHealthPanel` 情感曲线 `buildCurve` 逐章 await **无 token**，循环中重跑体检或切书
   后旧循环仍 `setCurve(pts)`。
4. 伏笔整表写回的「排队写 × 失败重读 × 切书」交错窗口：op2 可能用「重读后的磁盘内容」
   发整表写而丢掉自身改动；需延迟注入实测才能定性。
5. 重写历史面板不依赖 `projectPath`：仅「切书且章号相同」时残留旧书版本列表
   （点「应用」会因版本 ID 不属于当前书而报错，不会写坏）。
6. `Cmd+K` 补全「接受」用 `document.activeElement` 取插入目标，非 TEXTAREA 时**静默不写入**
   （`ChapterEditor.tsx:399-407`）——需一次真机/浏览器实测定性。
7. 停止生成若发生在「连接建立阶段」，前端显示「生成失败」而非「已停止」——取决于
   `client.ChatStream` 在 ctx 取消时的返回形态（未读到）。
8. 生成中切项目时后端是否继续写旧书：未核 `OpenProject` 是否中断在途章节生成。
9. 跨客户端并发面（httpbridge 是否把小说绑定暴露给第二客户端）未验证。

**Go**
1. `N5` 分支摘要混入（`internal/project/project.go:319-322` 收全部 `*-summary.json`）。
2. `N9` `ForEachChapter` 缺口即停（`project.go:624-638`）；同族新实例
   `novel_foreshadow_lint_handler.go:167-180`、`novel_book_health.go:121-125`、
   `novel_import_ai.go:315-319`（缺口时后续章不计入 totalChapters → 误报 dangling）。
3. `N12` 迁移跳章仍落 v4 标记（`project.go:551-560` continue，`:588` 照 `finalizeV4Migration`）。
4. `N13` 嵌套大纲节点不可见：`novel_gate_handler.go:181-190` gateOutlineIssues 只遍历顶层
   （同文件 `:151-166` 却递归）；`create_chapter_handler.go:1110-1117` `findOutlineNodeByNum`
   不递归。
5. `G10` `OutlineNode.SceneRefs` 零写零读死契约（`internal/types/types.go:181`）。
6. `G11` `prompts/chapter-generate.json` 无生产消费点却在提示词工坊可编辑。
7. `NovelChapterPlanSave`（`novel_plan_handler.go:114-140`）无锁整表读-改-写：当前生产侧
   仅此一处写 plans.json，单客户端不可达；多客户端即为丢更新。
8. 零调用者导出绑定**口径**待对齐：台账记 11，Go 审计按更宽口径实测 36（132 个 NovelB 绑定）。
9. `internal/app/scene_gen_handler.go:229` 同款 `_ = pm.WriteOutlines(of)` 吞错（Go 线足迹外，
   已注明）。
10. `internal/chapter` 的 chapter-review 模板缺失恒走内置 fallback（`chapter.go:148-152`，
     `docs/gaea-optimization-direction-2026-09.md:141` 已在册）。

## 5. 验收

- **前端定向**：小说域 52 个测试文件 + 新增用例全绿；`tsc -b` 0；`eslint` 0 错。
- **反向守卫（防复发）**，每项至少一条测试钉住：
  1. 板块被切走 → 阅读页 Ctrl+S/F11 **不响应且不 preventDefault**（M1 灵敏度实测：把
     `boardVisible` 改 `true` 恰该用例变红）。
  2. 有脏时切书 → 弹窗拦截；「先保存」全成功才放行；保存失败不切（M2）。
  3. 局部重写/重写历史应用后**未保存正文不被覆盖**且脏标志保留（A1/A2）。
  4. 生成中切项目 → 旧书事件不落进新书（A3）。
  5. 载入在途点保存 → 不写出跨章内容（A4）。
  6. 切书后各面板/报告态清空（B1/B3/B4/B5）。
- **Go**：`internal/app` + `internal/project` 定向包绿；`gofmt`/`vet`/`build` 0。
- **门禁**：`scripts/ci.ps1 -Quick` 收尾；全量档在发版前跑一次；漂移闸 OK@新版本号。
- **目检（可选）**：`GAEA_WALKTHROUGH=1` 壳内走查四处确认弹窗与面板切换。

## 6. 落地情况

### 线1 共享原语（主代理，已完成）
| 项 | 状态 | 证据 |
|----|------|------|
| M1 板块级可见信号 | 落地 | 新 `frontend/src/lib/boardActive.ts`（模块级 store + `useSyncExternalStore`，未通知恒 true）；`MainLayout.tsx` 在 `page` 变化时 `notifyBoardActive(page)`；`NovelPage.tsx` 的 pane `active` 改为 `activeTab===t.key && boardVisible` |
| M1 反向守卫 | 落地 | `NovelPage.test.tsx` 新增 1 例（**灵敏度实测**：去掉 `&& boardVisible` 恰该例变红）；`boardActive.test.tsx` 5 例含壳层接线 source-guard |
| M2 跨页切书确认 | 落地 | 新 `components/novel/novelSwitchGuard.ts`（登记 + 单一拦截点，三选仅当全部脏持有者都能保存）；`HomePage.tsx` 四处 `openProject`（新建/导入/在线导入完成/打开）全部过闸 |
| M2 反向守卫 | 落地 | `novelSwitchGuard.test.tsx` 9 例（无脏零打扰 / 三选先保存全成功才放行 / 保存失败中止 / 不能保存降级两选 / ✕＝取消 / 注销即退出）；`HomePage.test.tsx` 新增**调用点接线** 1 例（有脏时点开书不直接切、确认后才切） |
| B13 删除正在编辑的书 | 落地 | `HomePage.handleDelete` 命中当前项目即 `closeProject()`；2 例（删当前关项目 / 删别的书不动上下文） |
| M3 订阅纪律 | 落地 | `TTSPlayer.tsx`、`CharacterLibraryPage.tsx` 的整通道裸 `EventsOff` 改 `subscribeWailsEvent`；顺带收敛 `editor/GhostText.tsx`（带 handler 形态，唯一入口纪律）；新增源码守卫 `frontend/src/events-discipline.test.ts` 2 例防复发 |
| 台账更正 | 落地 | `进度计划/gaea-novel-polish-20260928.md` 补登线5 + 两条台账更正块；`docs/gaea-longform-novel-system-2026-09.md` 更正生成物路径（不是 `frontend/src/wailsjs/`） |

### 线2/3/4/5/6
| 线 | 状态 | 关键实测 |
|----|------|----------|
| 线2 创作页 | 8 项落地（A1/A2a/A3/A4/A6/A7/A8/B5a）+ 1 处授权改动（`create/EditorPanel.tsx` 保存按钮 `disabled`） | `CreatePage.test` 27→37 例、`NewCharactersModal.test` 7 例、`EditorPanel.test` 全绿；**5 处护栏逐个改坏 → 恰好对应用例变红**（变异验证） |
| 线3 阅读页 | 3 项落地（A2b/A9/A10），只动 2 个文件 | `ChapterPage.test` 19→21 例（含主代理补的「恢复上次阅读章」迟到回填守卫）、`pages/chapter/**` 11 文件 110 例全绿；5 处护栏变异验证 |
| 线4 设定页与面板群 | 13 项落地（B1–B12 + M4）；B5b 在 `ChapterReviewPanel`/`StyleFingerprintPanel` 为降级兜底（真相源在 CreatePage）；新建 `ExportPanel.test.tsx`（原无测试） | 12 文件 136 例全绿 |
| 线5 Go | 12 项落地（G1–G9 + N7/N8 + M5）；N9/N12/N13/G10/G11 如实入池 | `internal/project|novelcontext|novelstyle|novelreview|outline|analysis` 全绿；`internal/app` 整包 142.5s 绿；`gofmt`/`vet`/`build` 0；G2/G7/G9 变异验证 |
| 线6 订阅纪律 | 已并入线1（M3） | 源码守卫 `events-discipline.test.ts` 2 例 |

### 收口实测（主代理）
- 小说域定向：**53 文件 / 465 例全绿**（本批开工基线 49 文件 / 401 例 → 净增 4 文件 / 64 例）。
- `tsc -b` exit 0；改动面 **45 个前端文件 eslint 0 错 0 警**。
- Go：`internal/app` 整包绿 + 6 个小说相关包全绿。
- 反向守卫（本批新增，均做过变异验证）：板块被切走 ⇒ 全部 pane 非当前页；跨页切书有脏必拦截且保存失败不切；局部重写/重写历史应用后未保存正文不被覆盖且脏标志保留；生成中切项目旧书事件不落新书；载入在途保存不写出跨章内容；切书后各面板/报告态清空；`boardActive` 壳层接线 source-guard；零整通道 `EventsOff` source-guard。

### 台账更正（本条线）
- `进度计划/gaea-novel-polish-20260928.md`：补登「线5 重写弹窗关闭策略」（原漏登）+ 两条更正块（裸 `EventsOff` 当时未清零；「在途禁用」当时只有串行队列）。
- `docs/gaea-longform-novel-system-2026-09.md`：生成物路径由 `frontend/src/wailsjs/go/app/NovelB.*` 更正为 `frontend/wailsjs/go/app/NovelB.js`（前者目录不存在）。
- 对账结论：v4.421 抽验 10 条 + v4.422 抽验 8 条中**未发现「声明已落地但代码完全找不到」**（15 条完全落地、2 条部分已由本批补齐、1 条路径写错已更正）。
