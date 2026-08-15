# gaea 前端整体（frontend/src）独立审查报告

审查范围：frontend/src 全部 423 个文件（pages / components / gaea / lib / stores / hooks / utils / api / types）。
审查方法：只读；所有结论均经 read/grep 逐行核实，证据格式为 文件:行号。重点关注：性能（虚拟化/渲染）、状态一致性、错误可见性、残留 any、文件大小、mock 契约、可访问性。阶段 6（v2.33.0）宣称完成「巨型文件拆分 + any 清零 + 漂移检查恢复」，本报告一并核验这三项是否完整落地。

总体结论：三项收敛措施中，any 清零与漂移检查是完整、可验证的落地（见亮点 1-2）；巨型文件拆分只完成了一半——T6-10.1 把 ChatPage/Composer/Transcript 拆了，但 CostLibraryView(961 行)/Sidebar(849)/XlsxPreview(776)/CharacterPage(732) 仍是单体。下一阶段最值得做的不是新功能，而是三件事：流式渲染收敛（修 memo 击穿）、错误可见性收口（静默 catch 清零）、加载错误态补齐。

## 高危

### H1. 发送/审批等用户关键操作失败仍被静默吞掉，UI 状态与后端事实脱节
- 证据：gaea/lib/store.ts:454 — (display !== submit ? app.SubmitDisplay(...) : app.Submit(submit)).catch(() => {})；:459/:461 — Cancel 两次 .catch(() => {})；:464 — Approve app.Approve(id, allow, session).catch(() => {})；:465 — AnswerQuestion 同样；:467 — NewSession .catch(() => {}) 后无条件 dispatch({ type: "reset" })。
- 影响：store.ts:336-338 的注释明确写了阶段 6 原则「T6-1.2 去静默 catch：错误必须可见」，但主发送路径恰恰是静默的。Submit 失败时：user 消息已入 items（:452 先 dispatch 再调用），running 已置 true（:242），后端却没收到任何东西；唯一兜底是看门狗（:434-443）30 秒后 localCancel——但 localCancel（:246-253）只复位状态、不产生任何错误提示。用户看到的是「消息发出去了、一直转圈、然后没了」，最坏情况是用户以为失败重发，造成重复提交。Approve 更严重：:464 先 dispatch({type:"clearApproval"}) 把审批弹窗清掉，再调用后端；若调用失败，用户永远不知道自己的「允许/拒绝」没有送达，后端的工具调用会一直挂着等到超时——这是权限门的静默失守。
- 修复方向：给 send/approve/answerQuestion/newSession 等用户触发操作统一接 toast.show(..., "warn") 或 notice 通道（store 里已有 logBridgeError 可复用模式）；Approve 改为「先调用成功再 clearApproval」，失败时保留弹窗并提示重试。

### H2. 流式 chunk 下「全量重渲染」链路的 memo 被内联回调击穿（长会话/长回复必现卡顿）
- 证据：Transcript.tsx:430-431 — merged/segments 两个 useMemo 都依赖 items，每 chunk 重建；:464-474 — subcallsByParent Map 每 chunk 重建（子数组全部新分配）；:499-578 — renderSegment 的 useCallback 依赖 [userTurn, openTurn, onRewind, scheduleMeasure, subcallsByParent, dismissedErrors]，其中 userTurn（:489-496）与 subcallsByParent 每 chunk 都变 → renderSegment 每次都是新函数；:524-528 — UserMessage 的 onToggle 是内联箭头函数（每次新引用）；:539-543 — AssistantMessage 的 onCapture 是内联箭头（每次新引用）。
- 影响：gaea/components/Message.tsx:81/:158 明明给 UserMessage/AssistantMessage 加了 memo，但传入的 onToggle/onCapture 每次 chunk 都是新引用，memo 浅比较必然失败，全部历史消息在流式期间每 chunk 重渲染一次；ToolCard（ToolCard.tsx:36 memo）同样被子调用数组重建击穿。叠加 store.ts:136-145 每 chunk [...s.items] 全量复制数组，长会话（几十轮、上百条消息）流式时每 token 的渲染成本 ≈ 全部历史消息数 × 每消息 markdown 解析，这是主办公对话页最核心的性能瓶颈。
- 修复方向：① 把 onToggle/onCapture/onRewind 用 useCallback 稳定化（或让 UserMessage/AssistantMessage 改为接收 item 引用并自行订阅）；② renderSegment 改为「按 item id 缓存已生成的 ReactNode」或把每个轮次拆成独立 memo 组件，让未变化的轮次直接短路；③ store 的 text/reasoning 追加保持 [...s.items]（数组不可变是 zustand 要求，保留现状，靠 ② 解决渲染侧）。

### H3. reconcileFinalAnswer 的「前缀包含」启发式可能再次吞掉最终回答（正是它想修的那个 bug）
- 证据：gaea/lib/store.ts:370-389 — turn_done 时拉 History，取最后一条 assistant 正文的 slice(0, 120) 作 probe（:381），然后 if (!rendered.includes(probe))（:382）决定是否补发。rendered 是把所有已渲染 assistant 正文 join 起来的字符串（:377-380）。
- 影响：这是「末端事件丢失 → 最终回答不可见」的兜底修复，但判定标准是「120 字符前缀是否出现在已渲染文本里」。两个方向都会错：① 假阴性——同一轮里两次相似提问、或助手回答以相同开场白开头（如「好的，我来分析一下……」），probe 命中旧文本 → 最后一条真实回答被跳过，该吞还是吞；② 假阳性——新回答前 120 字符与旧回答相同但后续内容不同，同样被跳过。probe 前缀命中概率随会话变长而升高。
- 修复方向：用消息 id/时间戳/长度差做判定（History 若带稳定 id 则按 id 去重），或把「最后一条 assistant 是否已渲染」改为比较最后消息的完整文本与渲染文本的末尾而非前缀包含；至少把 probe 加长并把匹配方向改为「渲染文本 endsWith 该正文」的宽松版。

## 中危

### M1. 主办公对话区（Transcript）无 role="log"/aria-live，且与 ChatPage 的可访问性水平不一致
- 证据：gaea/components/Transcript.tsx:587 — 外层容器只有 <div className="transcript" ref={scrollRef} onScroll={onScroll}>，无任何 ARIA 角色；对比 pages/ChatPage.tsx:328 — 对话区带 role="log" aria-live="polite" aria-label="对话消息"。
- 影响：屏幕阅读器用户在 gaea 主对话页无法感知新消息/流式输出；两套对话渲染栈（Transcript vs MessageList）可访问性标准不一致。
- 修复方向：Transcript 容器补 role="log" aria-live="polite" aria-label（注意 aria-live 与流式高频更新可能造成朗读风暴，可考虑流式时用 polite、完成时 announce）。

### M2. MarkdownContent 未 memo，persona 模式流式时每 chunk 全量 markdown 重解析
- 证据：components/MarkdownContent.tsx:11 — 无 memo（直接导出普通函数组件）；components/chat/MessageList.tsx:81 — 流式消息每 chunk <MarkdownContent source={display} .../> 重新解析累计文本；:86 — persona 模式（非 plain）所有消息走 MarkdownContent；对比 :85 的 plain 模式走 ChatMarkdown（components/ChatMarkdown.tsx:47 已 memo，text 未变的旧消息可短路）。
- 影响：hooks/useChatStream.ts:127 — 每 delta setStreamText(prev => prev + (p.content || ""))，累计字符串逐 chunk 增长；persona 模式下每条旧消息每 chunk 都要 react-markdown 重新解析一遍，长回复（数千 token）时解析成本 O(n^2) 级，界面明显掉帧。
- 修复方向：给 MarkdownContent 加 memo（一行改动，收益最直接）；MessageList 内旧消息的 source 引用本就不变，memo 后即可短路。

### M3. Transcript 无虚拟化 + 运行结束时全部过程卡整体重挂载
- 证据：gaea/components/Transcript.tsx:593-617 — segments.map 渲染全部轮次；:598-601 — const segKeyFinal = running ? segKey : "done-" + segKey，running 翻转为 false 时所有过程卡的 key 同时变化 → 全部重挂载。
- 影响：① 长会话（几十轮）下整棵对话 DOM 常驻，翻历史 + 流式并存时首帧/滚动卡顿；② turn 结束那一瞬所有 ProcessCard 重挂（useGSAPCollapse 重新跑、useMemo body 重新计算、useEffect 里 setOpen(!small) 重新执行），可能造成视觉跳动与瞬时主线程峰值；③ 与 Sidebar 已做的 react-window 虚拟滚动（Sidebar.tsx:660-669）形成明显反差。
- 修复方向：不要上完整虚拟化（见「建议不做」）；优先做「完成轮默认折叠 + 每轮独立 memo 组件 + 轮级内容不随流式重建」；segKeyFinal 的 done- 前缀改为仅在「运行中↔完成」切换时对最后一轮生效，历史卡 key 保持稳定。

### M4. 面板数据加载无错误态：失败 = 无限 loading / 静默空列表
- 证据：gaea/components/KnowledgePanel.tsx:51-61 — loadList 只 setLoading(true)，KnowledgeList 无 .catch，reject 后 loading 永远为 true，且 setLoading(false) 在 then 里永不执行；stores/outlineStore.ts:36-42 — loadOutlines 无 try/catch，fetchOutlines reject 时 loading 卡 true、出现未处理拒绝；gaea/components/FileTree.tsx:78 — catch { setEntries([]) } 把「加载失败」伪装成「空目录」。
- 影响：后端偶发失败（启动竞态、索引未就绪）时，知识库面板转圈到天荒地老、大纲不出现、文件树显示虚假的「空目录」——都是用户可感知的假状态，且没有任何错误提示或重试入口。
- 修复方向：统一加 { data, loading, error } 三态；错误时渲染 ErrorCard/EmptyState 带重试按钮；FileTree 把空数组与错误区分开。

### M5. gaea/App.tsx 整树订阅 + Sidebar 的 ui 对象每次渲染重建
- 证据：gaea/App.tsx:95 — useController() 返回 store(useShallow(s => s))（store.ts:350），items 每 chunk 换新数组 → App 每 chunk 全树重渲染；gaea/components/Sidebar.tsx:509 — const ui: SessionRowUI = {...} 每次渲染新建对象，传给 react-window 的 rowProps={{ rows, ui }}（:665），可见行每 chunk 全量重渲染。
- 影响：流式期间应用外壳（Sidebar、Composer、TodoPanel、右栏）每 chunk 重渲染；Sidebar 虽有虚拟化缓解 DOM 成本，但 hooks 重算（rows useMemo 依赖 filteredGroups 引用未变会短路，ui 对象不会）仍白跑。属于结构性浪费，中等会话规模即有感。
- 修复方向：App 改为只订阅需要的字段（或把 items 订阅下沉）；Sidebar 的 ui 用 useMemo 包一层（依赖展开为各字段）。

### M6. HistoryPanel 全量渲染所有会话（无虚拟化/分页）
- 证据：gaea/components/HistoryPanel.tsx:112-203 — groups.map → g.items.map 渲染全部会话，无上限；对比 Sidebar.tsx:472 — 项目内会话有 PER_PROJECT_PAGE = 8（:55）折叠 + react-window。
- 影响：个人长期使用积累数百上千会话后，打开历史抽屉一次渲染全部条目；搜索过滤（:27-34）虽在内存里做，但渲染量不降。
- 修复方向：给 HistoryPanel 复用 Sidebar 的「显示更多」分页或 react-window（行高可变度低，适合固定行高虚拟化）。

### M7. 图标按钮大面积只有 title 无 aria-label / 关键折叠控件缺 aria-expanded
- 证据：gaea/components/HistoryPanel.tsx:184-194 — 重命名/删除按钮仅 title；gaea/components/FileTree.tsx:93-106 — 目录行按钮仅 title 无 aria-expanded；Transcript.tsx:656 — CompactionCard 的折叠按钮无 aria-expanded（对比 :322/:203 的 ProcessCard/InlineReasoning 有）；components/chat/MessageList.tsx:43-45 — 复制按钮仅 Tooltip title。
- 影响：title 属性对主流屏幕阅读器不生效（不读 title 是常见行为），这些纯图标操作对键盘/读屏用户完全不可达；折叠控件无 aria-expanded 时展开状态不可感知。
- 修复方向：重点路径（历史面板操作、文件树、折叠卡）补 aria-label/aria-expanded；不需要机械全量扫（见「建议不做」）。

### M8. Toast 无 role="status"/aria-live，错误提示读屏不可达
- 证据：gaea/components/Toast.tsx:36-48 — 容器 <div className="fixed top-3 ..."> 无任何 ARIA；且 Toast 只有 info/warn 两档（:3），没有 error 档，Sidebar.tsx:779 传的 "warn" 是最高级。
- 影响：应用里大量「操作失败」仅以 toast 呈现（如导出失败 Sidebar.tsx:779、会话操作失败 gaea/App.tsx:100），读屏用户完全收不到；且错误与普通信息视觉上只差左边框颜色，弱视用户难区分。
- 修复方向：Toast 容器加 role="status" aria-live="polite"（或 role="alert" 用于错误），增加 error 档位。

## 低危

### L1. 巨型文件拆分未收尾（文件大小）
- 证据：gaea/components/CostLibraryView.tsx 961 行、gaea/components/Sidebar.tsx 849 行、gaea/components/XlsxPreview.tsx 776 行、pages/CharacterPage.tsx 732 行、gaea/components/KnowledgePanel.tsx 567 行（行数统计于本机）。其中 CostLibraryView 含 12 处 .map、XlsxPreview 5 处，业务逻辑与渲染混编。
- 影响：可维护性/后续改动成本；与 T6-10.1「巨型组件拆分」的既定方向不一致（ChatPage/Composer/Transcript 已拆）。
- 修复方向：按 CostLibraryView/XlsxPreview 内已见到的模式（CostRow/ListView/TableView 已 memo 化，说明拆分基础好），把数据获取/加工抽到 lib/ 或 hooks/。

### L2. Sidebar 过渡断言与无意义按钮
- 证据：gaea/components/Sidebar.tsx:64-65 — isInterruptedSession 的 as SessionMeta & { interrupted?: boolean } 断言，注释说「契约层补齐后可移除」，但字段并未进 types；:611-617 — 折叠态当前会话指示按钮没有 onClick 处理，是个不可交互的 button。
- 影响：断言若与 Go 契约漂移无编译期保护（interrupted 是运行时字段）；无 onClick 的 button 对键盘用户是「假焦点」，且语义应为展示元素。
- 修复方向：interrupted 补进 types/SessionMeta；折叠态指示改用 div/span。

### L3. 两套对话渲染栈并存，行为细节分叉
- 证据：gaea/components/Transcript.tsx（主办公对话）与 components/chat/MessageList.tsx（whisper 聊天）是两套独立实现：一个已 memo 但被击穿、一个部分 memo；滚动策略各写一套（Transcript.tsx:367-423 vs ChatPage.tsx:78-89）；错误展示一个用 ErrorCard（Transcript.tsx:560）一个用行内错误块（MessageList.tsx:64-76）。
- 影响：流式渲染优化要双份落地；两处可访问性/错误呈现标准不一致（见 M1/M7）。
- 修复方向：不合并（成本高），但把「渲染性能基元」（markdown 渲染组件、流式消息组件、滚动 hook）抽成共享件。

### L4. mock 契约测试覆盖不均衡
- 证据：gaea/lib/mock-contract-t63.test.ts 与 mock-contract-e5.test.ts 只锁定 chat 与 E5 域（成本/检索）；mock/core.ts、mock/memory.ts、mock/office.ts、mock/model.ts、mock/settings.ts 等域没有对应的契约锚点测试；makeMockApp（mock.ts:40-52）靠 TS 返回类型约束形状。
- 影响：Go 侧契约变更时，未覆盖域的 mock 可能悄悄漂移（历史上 mock 泄漏进真实应用的问题，bridge.ts:429-432 注释有记载）。
- 修复方向：给其余域补「结构 + 关键锚点值」冒烟测试（照 E5 测试的写法，成本低）。

### L5. 其他零散项
- 证据：stores/appStore.ts:209 — novelsDir 硬编码默认路径；:312/:319 — as ProjectInfo / as StatsData 无运行时校验；gaea/hooks/useBridgeWatch.ts:6-44 — 每 5 秒一次 Meta() 轮询常驻（含断连竞速逻辑，但空转成本可合并到已有看门狗）；gaea/lib/bridge.ts:375-376 — HerdsmanDigitalLife/HerdsmanOperations 返回 Promise<unknown>，调用点需自行断言。
- 影响：均为低风险整洁性问题。

## 亮点（做得好、不要动）

1. any 清零完整落地（核验通过）：全库 \bany\b 仅命中 13 处，全部是注释（如 api/settings.ts:3「消除 (window as any)」）、测试工具（ChatPage.test.tsx:168 的 expect.any）或 HTML 属性（CostImportModal.tsx:263 的 step="any"），无一处真实 any 类型用法。替代方案（types/wails.d.ts 统一声明 + api/ 封装层）质量高。
2. 编译期绑定面漂移双向断言（核验通过）：gaea/lib/bridge.ts:880-918 — AppBindingTarget / MockOnlyNames / LegacySurfaceNames + AssertNever 类型级校验；bindingNames.ts（467 行）由 scripts/gen_bindings 生成；scripts/check-bindings-drift.ps1 存在；frontend/package.json:8 — build = tsc -b && vite build，断言随 typecheck 生效。这是「漂移检查恢复」的真实落地，且历史上 6 处映射错误（KeepWarm*/PreloadPlan*/AgentMode 等）已被注释记录并修正。
3. 错误归一化层设计好：BridgeError（bridge.ts:689-699）保证 message 可枚举 + code 机器可读；invoke 统一入口（:720-730）把每次失败写 gaea.log 再抛；LogFrontendError 自身不套 invoke 防递归（:753）。
4. Sidebar 是虚拟化样板：react-window 行模型拍平（Sidebar.tsx:67-75、:465-489）、ResizeObserver 测高（:494-507）、jsdom 兜底高度（:60）、overscan（:668）、aria 分隔条（:870-883）都做得很完整——其他长列表应向它看齐。
5. useNow 共享单间隔时钟（gaea/lib/useNow.ts:6-23）：替代 N 个独立 setInterval，无人订阅时自动停表。
6. 面板懒加载：gaea/App.tsx:23-26 — Memory/History/Capabilities/Knowledge 四个面板全部 lazy。
7. useItems 独立订阅（gaea/lib/store.ts:525-532）：注释明确说明设计意图，把流式高频 items 更新与 App 其他字段隔离——这是 M5 的减害措施，保留。
8. useChatStream 终态四路径 + 无帧超时（hooks/useChatStream.ts:104-158）：先订阅后收帧、30s 无帧超时、启动失败/事件错误/超时/卸载四路径全部收尾、finally 必执行——阶段 6 SSE 加固在前端的最完整体现。
9. 看门狗校准（gaea/lib/store.ts:434-443）：30s 用 GaeaRunning 校准一次，防止 turn_done 丢失导致永久「执行中」，是 H1 的配套兜底（保留，但要给用户可见提示）。
10. prefers-reduced-motion 双处尊重：gaea/lib/useEntranceAnimation.ts:55 与 hooks/useChatStream.ts:167。
11. mock 契约测试（mock-contract-t63/e5）：把 mock 钉在 Go 侧契约上（如 RetrievalEvalRun 锚点 total=12/threshold=0.8，mock-contract-e5.test.ts:48-76），防止「mock 与真实不符」再次发生。
12. 大规模 memo 纪律：31 处 memo(（Message/ToolCard/ToolGroup/DeliverableCards/CostRow 等），说明团队有性能意识——问题只在「被内联回调击穿」的执行层（H2）。
13. ChatPage 滚动与朗读资源管理：role="log"（:328）、stickToBottom ref 模式（:77-89）、speakUrl 的 revokeObjectURL 生命周期（:136-143）都是正确做法。
14. 根级 ErrorBoundary（main.tsx:160-164 + components/ErrorBoundary.tsx:26-39）：捕获后写 gaea.log 并可重试。

## 建议不做（避免过度工程）

1. 不要给 Transcript 上完整虚拟化。过程卡高度动态（GSAP 折叠动画）、流式逐 chunk 插入、轮次间有分段交替，固定行高虚拟化收益低、滚动锚定复杂度高。更优解是 H2/M3 的「memo 修复 + 完成轮折叠 + 轮级稳定 key」，成本一个数量级更低。
2. 不要引入新的全局状态方案。zustand + 局部 store（usePreviewStore/useUpdatedFilesStore/useComposerInsertStore 等小 store 模式）已经覆盖需求；App 订阅粒度问题（M5）用「细分订阅」解决，不是重构。
3. 不要做 rAF 批量合并流式渲染。store.ts:395-404 用 queueMicrotask 保证每 chunk 即时渲染是刻意设计（注释写明「不被 React 18 批处理合并」），改成批量会牺牲流式即时性，且收益不确定——先做 H2/M2 的 memo 层修复，重测再决定。
4. 不要机械全量补 aria-label。重点覆盖主路径（对话区、Toast、历史操作、折叠卡）即可；对 500+ 文件做无障碍大扫除会淹没真正的问题（M1/M7/M8）。
5. 不要推翻 bridge 的 Proxy 路由与漂移检查。realApp() 按调用时解析 + gaeaToGaea 映射 + 类型级断言这套组合工作良好（亮点 2/3），任何「更直观」的重写都会丢掉已锁定的契约。
6. 不要扩大 mock 覆盖到全后端语义。浏览器 mock 的定位是布局/联调（?mock=fresh/running/demo 场景系统），继续加深 mock 会持续增加维护成本；要补的是 L4 的契约冒烟测试，不是行为。

## 下一阶段前端最值得做的 3-5 件事

1. 流式渲染收敛（优先级最高）：修复 H2 的 memo 击穿（稳定 onToggle/onCapture 引用）→ 给 MarkdownContent 加 memo（M2，一行改动）→ 让 Transcript 的轮级 key 稳定（M3）。预期长会话流式帧率与 CPU 占用明显改善，收益可量化（DevTools Performance 对比）。
2. 错误可见性收口：把 store.ts 中所有用户触发的静默 catch（H1 列出的 454/459/461/464-467 等）统一接入 toast/notice；Approve 改「成功后再清弹窗」。这是阶段 6「T6-1.2 去静默 catch」的未竟部分，应在本阶段收尾。
3. 加载错误态补齐：KnowledgePanel.loadList、outlineStore.loadOutlines、FileTree 等 Promise 路径补 error 态与重试（M4），杜绝「无限 loading / 假空目录」。
4. a11y 主路径：Transcript 对话区 aria-live（M1）、Toast role=status + error 档（M8）、历史/文件树图标按钮 aria-label（M7）。
5. 巨型文件二次拆分收尾：CostLibraryView(961)/XlsxPreview(776)/Sidebar(849)/CharacterPage(732) 按既定方向抽 lib/hooks（L1），顺带把 Sidebar 的 ui 对象 memo 化（M5 的一半）。

（报告完；证据文件索引见正文各条目。）
