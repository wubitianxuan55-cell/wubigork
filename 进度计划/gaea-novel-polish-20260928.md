# 小说板块优化批：未保存保护 · 常驻页副作用治理 · 体验收口——规格

> 2026-09-28 立项。来源=用户指令「优化小说板块」（拍板：A+B 合并批＝正确性修复 + 体验收口）。
> 证据来源=三线并行只读审计（小说前端 / Go 后端 / 计划台账）+ 主代理读码复核。
> 目标版本：v4.421.0。绑定面预期零变更（纯前端收口，若某线需 Go 支持则单列并回写本节）。

## 0. 论点（为什么是这些）

小说板块五个子页**全部常驻挂载**（`NovelPage.tsx:212-221` 五 pane 同挂，靠
`.novel-tab-pane{display:none}` 隐藏，`novel-workspace.css:244-253`），而「隐藏」不等于
「卸载」——由此衍生三类真实缺陷，全部由读码确证（非推测）：

1. **窗口级副作用不随页隐藏**：`ChapterPage.tsx:158-177` 与 `NovelSettingPage.tsx:104-113`
   各挂一份无条件 `keydown`，`HomePage.tsx:79-88` 再挂一份。全仓仅此两处 Ctrl+S 处理者
   （grep 实证 `key === 's'` 仅 2 命中），于是**一次 Ctrl+S 同时保存设定与阅读页那章**，
   而创作页正在写的正文反而不保存；F11 在任意子页翻转不可见阅读页的专注模式；
   阅读模式残留时 ←/→ 在书架/设定页翻章。
2. **面板上下文不随项目切换刷新**：`ForeshadowPanel`/`ConsistencyPanel` 的 `load`
   只依赖 `disabled`，而调用点 `<ForeshadowPanel />` 从不传 props（`NovelSettingPage.tsx:321-322`）
   → 切书后仍显示上一本的伏笔表与一致性报告；且 `disabled` 分支（「请先打开小说项目」）
   是死代码，无项目时反而显示红色「伏笔加载失败」。
3. **未保存缓冲无脏保护**：创作页 `content` 是纯受控缓冲 + 显式保存钮（无自动保存），
   但 `selectChapter`（:275-288）、切项目 effect（:273）、`startGeneration`（:335）
   都直接 `setContent(...)` 覆盖。阅读页早已有 `needsCloseConfirm` + 确认弹窗
   （`ChapterPage.tsx:297-311`），创作页是同类缺陷的漏网点——**这是本刀唯一会丢作者正文的一类**。

体验面（B）同源：创作页工具轨 12 个按钮平铺、创作参数每次进页回默认、设定页
下行双面板固定 300px 不可收。

## 1. 刀面（五线，足迹互斥）

### 线1（主代理）创作页 CreatePage 收口
1. **未保存正文保护**：引入 `loadedSnapshot` + `dirty`；切章、生成、删除节点、
   项目切换前经共享原语确认（三选：先保存 / 放弃修改 / 取消）。
2. **覆盖/追加不再绑在 Esc 上**：`handleDirectGenerate` / `handleAddNext` 的
   `Modal.confirm(onOk=覆盖, onCancel=追加)` → 三选项对话框（覆盖 / 作为分支追加 / 取消），
   ✕/Esc 一律＝取消（现行实现里用户想退出却触发了一次 AI 生成）。
3. **AI 反推大纲**：12 分钟轮询加卸载/切项目守卫（`alive` ref），行进中显示已等待时长，
   回落同步路径时同样暴露「取消反推」。
4. 文风指纹打开失败写 `fpMsg`（现写 `stateMsg`，面板里永远看不到）。
5. 设定拉取失败不再被归因成「设定为空，请先在设定页填写」。
6. **创作参数持久化**：目标字数 / 温度 / 写作技能入 localStorage（字号已有先例）。
7. **工具轨治理**：12 个按钮按语义分组（写后质检 / 结构工具 / 模板与文本），
   低频项收进分组，**零功能删除**。

### 线2（子代理 A）常驻页快捷键门控
- `NovelPage` 按 `activeTab` 下发当前页（prop 或事件），`ChapterPage` /
  `NovelSettingPage` / `HomePage` 的 window `keydown` 一律按「本页是否当前页」门控。
- Ctrl+S 单一归属：阅读 tab 保存章节、设定 tab 保存设定；其余子页不响应。
- F11 仅阅读 tab 生效；←/→ 仅阅读 tab + 阅读模式。

### 线3（子代理 B）设定页与两面板上下文
- `ForeshadowPanel` / `ConsistencyPanel`：`load` 依赖 `projectPath`，
  由 `NovelSettingPage` 传 `disabled={needsProject}`（对齐 `WorldviewSectionsEditor` 先例）。
- 设定导入（md/txt/json）在 `dirty` 时经共享原语二次确认。
- 下行「伏笔 + 一致性」区可折叠（默认展开），布局零破坏。

### 线4（子代理 C）组件修复批
- 在线导入进度事件竞态（`BookSearchModal`：起跑前订阅 / 失落兜底轮询）。
- 章际对比选择加 seq 守卫（对齐 `BookSearchModal.pickSeqRef` 先例）。
- 三处 `EventsOff(channel)` → `subscribeWailsEvent`（仓规：禁裸 EventsOff）。
- 提示词工坊切模板在 `dirty` 时确认。
- 新建小说：空标题禁用 OK + `confirmLoading` 防双击。
- 伏笔全表写回串行化（`ForeshadowPanel.persist` 队列 + 在途禁用）。

### 线5（子代理 D，可选）重写弹窗关闭策略 + 换算收敛
- `RewriteModal` / `PartialRewriteModal` 运行中关闭策略统一（不假装可取消；
  运行中禁止误关并提示结果可在「重写历史」查看）。
- `EditorPanel` 复述的 rune↔code-unit 换算收敛到 `annotationMarks`（含单测）。

### 线0（子代理 G）Go 侧正确性修复（后端审计并入，含唯一 P0）
1. **N1（P0）取消生成覆盖已完成章节**：`create_chapter_handler.go` 的取消落盘
   `partial` 无条件覆盖 `chapters/NNN.md`——对已写完的章再生成并中途取消即整章被截断替换，
   无备份、无测试。改为「写前判目标是否已存在且非本次产物」→ 存在则另存
   `NNN.partial-<ts>.md`（或先快照），并补单测。
2. **N2（P1）取消破坏章节互斥**：取消路径先 `delete(chapterGenCancels, key)`
   而生成协程还活着 → 立刻再生成同章可并发出两个写者。改为由 `unregisterChapterGen`
   独占清理。
3. **N3+N4+N5（P1，同一根因）写回路径吞错 + blob/scene 分叉**：
   `novel_llm_rewrite_handler.go` 与 `novel_deslop_handler.go` ①写错误被丢却回报
   `done:true`（用户看到「已改写 N 句」而磁盘未变）②v4 场景改写后未调
   `syncBlobFromScenes`（阅读页/检索/导出仍是旧文）。收敛为单一写回 helper：写入失败
   即返回 error，场景写回后同步 blob。

> 与本刀无关的后端发现挂观察池（不在本刀动）：N6 搜索标题命中抑制正文扫描、
> N7 生成后协程未托管、N8 建章节点忽略 `WriteOutlines` 错误、N9 `ForEachChapter`
> 缺口即停、N10 分支摘要混入、N11 书架全库全章 IO、N12 迁移跳章仍落 v4 标记、
> N13 嵌套大纲节点不可见、N14 零调用者导出绑定。

### 分线表（足迹互斥）

| 线 | 负责人 | 文件足迹 |
|----|--------|----------|
| 线0 Go 正确性 | 子代理 G | `internal/app/create_chapter_handler.go`、`novel_llm_rewrite_handler.go`、`novel_deslop_handler.go`（+测试） |
| 线1 创作页 | 主代理 | `pages/CreatePage.tsx`、`components/novel/RewriteModal.tsx`、`PartialRewriteModal.tsx`、`editor/EditorPanel.tsx`、`create/annotationMarks*` |
| 线2 快捷键门控 | 子代理 A | `pages/NovelPage.tsx`、`pages/ChapterPage.tsx`、`pages/HomePage.tsx` |
| 线3 设定页+两面板 | 子代理 B | `pages/NovelSettingPage.tsx`、`components/novel/ForeshadowPanel.tsx`、`ConsistencyPanel.tsx` |
| 线4 组件修复批 | 子代理 C | `components/novel/BookSearchModal.tsx`、`ChapterAnalysisPanel.tsx`、`create/useChapterStream.ts`、`create/useChapterGateNotice.ts`、`create/NewCharactersModal.tsx`、`PromptWorkshopPanel.tsx`、`CreateNovelModal.tsx` |
| 共享件（已完成，各线只读消费） | 主代理 | `components/novel/unsavedGuard.tsx`（`confirmDiscard` / `chooseUnsavedAction`） |

**跨线契约**：`NovelPage` 向每个 pane 传 `active?: boolean`（默认 `true`，兼容既有
测试直接渲染子页的写法）；子页把窗口级 `keydown` 一律按 `active` 门控。Ctrl+S 归属：
阅读 tab＝保存章节、设定 tab＝保存设定、其余子页不响应。

## 2. 明确不做（本刀）

- 真·取消重写 / 取消导入的**后端**改动（无取消绑定；本刀只做关闭策略与失落兜底）。
- 创作页自动保存（未拍板；本刀只做「不静默丢」）。
- 阅读页视觉改版、新能力（阅读统计/伴读升级）——用户已选 A+B，能力新增另刀。
- 绑定面变更：本刀预期零绑定变化；若某线不得不动 Go，则单列并同步 drift 锁。
- **跨页切书确认**（书架上打开/切换小说时若创作页有未保存正文→确认）：切书由
  `HomePage` 发起、脏状态在 `CreatePage`，需引入跨页共享态（store/事件）；本刀只做
  「切书后如实提示未保留」+ 本页内动作（切章/生成/删除/重写）确认，跨页拦截留观察池。
- **N-15 换算收敛**（`EditorPanel` 内复述的 rune↔code-unit 换算合并进 `annotationMarks`
  + 单测）：属高亮核心偏移数学，改动收益（P3）小于回归风险，留观察池。
- Go 侧其余发现（搜索标题命中抑制正文扫描 / 生成后协程未托管 / 建章节点忽略
  `WriteOutlines` 错误 / `ForEachChapter` 缺口即停 / 分支摘要混入 / 书架全库全章 IO /
  迁移跳章仍落 v4 标记 / 嵌套大纲节点不可见 / 零调用者导出绑定）= 观察池，另刀。残稿
  文件名秒级时间戳（同秒两次取消会覆盖前一个残稿）亦入观察池。

## 3. 验收

- 前端定向：小说域 42 个测试文件 + 新增用例全绿；`tsc -b` 0；`eslint` 0 错。
- 主门禁：`scripts/ci.ps1`（全量档）exit 0；漂移闸 OK@新版本号。
- 反向守卫（防复发）：为「Ctrl+S 仅在当前子页生效」「切书后面板重载」
  「切章破坏性动作有确认」各留一条测试。
- 目检（沙箱）：`GAEA_WALKTHROUGH=1` 起壳走查小说五页快捷键与确认弹窗。

## 4. 落地情况（2026-09-28 收口）

| 线 | 状态 | 关键实测 |
|----|------|----------|
| 线0 Go | 全部落地 | `create_chapter_partial_test.go`（残稿三态 + v4 场景不覆盖）+ `novel_writeback_test.go`（写失败报错 / v4 blob 与场景一致）+ 取消互斥 nil 占位；`internal/app` 定向 `Chapter\|Deslop\|Rewrite\|Cancel` PASS 82 / FAIL 0，场景族 17 例绿；gofmt 空 / vet / build 0 |
| 线1 创作页 | 全部落地（N-15 除外，入观察池） | `unsavedGuard` 原语 7 例；`CreatePage.test` 15 → 22 例（三选确认三分支 / 切章取消不切换 / 先保存再切换 / 并列分支取消不生成 / 载入失败可见化 / 残稿另存不标已保存 / 参数持久化 / 工具轨 12 入口） |
| 线2 快捷键 | 全部落地 | 5 pane `active` 下发 + 三处 keydown 门控 + 目录点章改 commit 后派发；新增 `NovelPage.test.tsx` 3 例；**灵敏度实测**=把门控改 `if (false)` 恰 3 条新守卫变红 |
| 线3 设定页+面板 | 全部落地 | 两面板 `load` 依赖 `projectPath` + 换书清 AI 深检 + 导入确认 + 下行可折叠；三文件 15→21 / 18→22 / 4→7 例 |
| 线4 组件批 | 6 项全部落地（项6 伏笔串行化由主代理补） | 导入进度失落兜底（3s 无事件→按成书清单对账 + 如实提示）/ 章际对比 seq 守卫 / 三处 `EventsOff`→`subscribeWailsEvent` / 工坊脏保护 / 新建防双击 / 伏笔整表写回串行化（23 例绿） |

**跨线契约落地**：`NovelPage` 向五 pane 传 `active?: boolean`（默认 `true`），
`ChapterPage`/`HomePage`/`NovelSettingPage` 消费；Ctrl+S 单一归属成立（阅读 tab = 章节、
设定 tab = 设定、其余不响应）。

**未落地（观察池）**：N-15 换算收敛、跨页切书确认、板块级 keepAlive、Go 余 9 项、
残稿文件名秒级时间戳。**未做**：真机沙箱目检（需窗口；本刀以定向测试 + 全量快闸为证据）。

**踩坑记录（供后续刀复用）**：
1. imperative `Modal.confirm` 的 DOM **不随 `destroy()` 立即卸载**——测试里手动摘
   `.ant-modal-root` 会把 antd 的 holder 引用打断、后续用例渲染不出弹窗；断言一律
   「取最新 `.ant-modal-confirm` + `within()`」，afterEach 只 `Modal.destroyAll()`。
2. antd 弹窗标题父子双匹配（`getByText` 报 multiple），两字按钮会插空格（「取 消」）→
   一律正则 + `getAllByText`。
3. `vi.clearAllMocks()` **不清 `mockImplementation`**：用例里用「永不 resolve」模拟在途时
   必须 `mockImplementationOnce`，否则实现泄漏、后续用例集体超时。
4. 改了共享函数的**返回契约**（如 `refreshSetting` 由 string 变 `{text, ok}`）后必须全局
   搜调用点——本轮漏改一处导致 5 个用例红，值得写成检查项。
5. 反推事件门控不能用「事件到达时的 active」判断：书架 `goto-tab` 与该事件**同 tick 连发**，
   prop 尚未翻转 → 必须「不可见即挂起、切到本页再执行」。
