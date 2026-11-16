# DeepSeek chat-demo 蒸馏调研（唯一事实源）

- 日期：2026-11-16
- 对象：deepseek-ai/chat-demo v0.1.1（MIT），官方静态聊天前端，13.5k 行 TS + React 18 + zustand
- 本地克隆：`.tmp/ds-chat-demo/`（shallow，用于后续动刀对照）
- 在线版：https://deepseek-ai.github.io/chat-demo/
- 调研纪律：全量逐文件通读（首轮只读 1/3 被用户打回，本轮补齐 store.ts 正文/SSE/协议层/search UI/sider/session/db/backup/history 实现/语音链/设置弹窗/App 壳）

## 0. 架构总览

```
src/
  services/
    api/        client(SSE 消费+Files API+校验 Key) / sse(手写 WHATWG 解码器) /
                parseEvent(流事件运行时校验) / errors(五分类+重试判定+file_id 失效识别) /
                types(协议) / speech(ASR oneshot + TTS 裸 PCM 流)
    chat/
      store.ts      1094 行，全局状态机（流/缓存/删除/导入导出/标题）
      branches.ts   消息树纯函数（~100 行）
      reply.ts      流事件归并 applyEvent + settleReply + 上下文裁剪 toRequestMessages
      view.ts       buildReplyView：blocks → 工具步骤时间线+正文+来源（纯函数）
      describe.ts   回复非正常态的展示归一（错误分类/拒绝/模板）
      drafts.ts     每会话草稿（文字+附件+上传任务）
      db.ts         IndexedDB 五 store（sessions/summaries/files + 两个 staging）
      backup.ts     导入导出串行队列 + 原子替换入口
      history/      官方 export_format_version 3：手写 zip 读写+CRC+流式解压+schema 校验
      sessionGroups.ts  时间分组（今天/昨天/7天/30天/按月）
      latestReminder.ts 每轮请求的五元组快照（日期/星期/位置/平台/语言）
    markdown/   parse(mdast+llm-math+GFM 无 codeIndented) / highlight(Prism 按需) /
                katex(懒加载+mhchem) / mermaid(串行队列)
    speech/     recorder(16k WAV ScriptProcessor) / readAloud(PCM 流排程播放)
  components/   base(手写浮层定位 anchored.ts+Menu/Modal/Select/Tooltip/Toast) /
                chatInput / messages / search / session / sider / settings / markdown / files
  app/          App 壳 + viewport(移动端 visualViewport 键盘跟随)
  i18n/         轻量自研：TranslationKey = keyof typeof zh，en 缺键即编译错
```

技术栈对齐度：React 18 + zustand + mdast 手写渲染 + CSS Modules；无 antd、无 react-markdown、无路由库（activeId 单状态切换）。

## 1. 一线候选（形态差距大，建议立项）

### 1.1 消息分支版本树（官方核心交互）★最大一块肉

数据模型（`src/services/chat/types.ts`）：

- `MessageBranch = { parentId?: string|null; selectedChildId?: string }` 混入两种消息
- `Session.rootMessageId`；`messages[]` 平铺存**所有分支**的全部消息
- 关键语义：**每条分支记忆自己的选择**（`selectedChildId` 挂在父消息上），切版本不丢各分支后续对话

纯函数层（`src/services/chat/branches.ts`，全 111 行）：

- `normalizeSession`：线性列表 → 单分支规范化（旧数据迁移）
- `getVisibleMessages`：沿 `selectedChildId` 走出当前路径（展示与请求上下文共用）
- `buildMessageVersions(session, previous)`：一次遍历收集兄弟版本；**版本列表未变化时复用旧对象与索引**，流式正文更新不重绘历史消息（`branches.ts:44`）
- `appendBranch`：新消息挂指定父节点并选中新分支，原分支原样保留
- `selectVersion`：只允许切当前路径上节点的兄弟版本，恢复该版本已选后续

交互面：

- 编辑提问 = 同父挂新 `[question, reply]` 分支，**保留原文件**，旧问题全部回答与后续可切回（`store.ts editMessage`）
- 重新生成 = 原问题下新增回答版本，不覆盖旧回答与后续（`store.ts regenerateReply`）
- `BranchSwitcher`（`‹ 2/3 ›`）挂在用户消息与回答消息下方；切换后**锚点恢复滚动位置**（`MessageList.tsx restoreScrollRef` 按 `data-message-version` 查找，避免回答长短变化导致操作区跳出视口）
- 流式中禁止切版本/编辑/重生成（`getStreamController` 互斥）

蒸馏落点：**必须双端**——已验证 `ChatSend(topicID, message, …)` 只带新消息文本，历史上下文由 Go 从 DB 按 topic 线性拼装（`internal/app/bindings_chat.go:20`）。分支语义若只进前端，Go 拼上下文仍会吃进全部分支消息。故：Go 侧 Message 需可查询的 parent/selected 字段（列或 extra JSON 服务端解析）+ 上下文拼装走 `getVisibleMessages` 同款路径算法；前端 ChatPage 管 BranchSwitcher/编辑/重生成。`branches.ts` 纯函数前后端各植一份（或路径裁剪只在 Go 一处）。附带设计点：`ChatMessagesPage(beforeSeq)` 分页游标在树结构下不再等于路径序，须改按「轮」分页或仅对当前路径分页。

### 1.2 续写 continue（小刀高价值）

- `canContinueReply`（`reply.ts`）准入：`prefix=true` 且（status=stopped 或 done+max_tokens）；含工具块/模板/拒绝/客户端工具调用一律不可
- `continueReply`（`store.ts:689`）：旧回答整体作为 assistant prefill 回传（含非空思考+签名），新流内容**追加在旧块之后**，`latestReminder` 沿用原快照不刷新
- 空思考块跳过不占位；prefill 无非空白正文时降级为重发原问题（避免服务端拒绝空 prefill）

### 1.3 消息生命周期状态机（错误/停止/拒绝/模板）

- `ReplyStatus = pending|streaming|done|stopped|error`；`StopReason = end_turn|max_tokens|tool_use|refusal`
- `describe.ts describeReply`：展示归一——upload/network/401（带「去填 Key」动作）/402（带充值链接）/429/5xx/协议错各配文案；**拒绝 refusal 按协议丢弃已下发思考与正文只留模板**（`reply.ts settleReply`）；模板文案 `isTemplate` 标记不进上下文
- `settleInterrupted`（`store.ts:273`）：加载时把上次未完成的 pending/streaming 收敛为 stopped
- **interrupted 宽恕**：已收到 `message_delta.stop_reason` 只是缺 `message_stop` → 按 done 收尾（`store.ts:1056`）
- `retryable` 标记：网络/限流/繁忙可重试，参数/鉴权/格式类重试无用（`errors.ts isRetryableApiError`）；upload 失败的回答在消息列表提供统一重试入口（`MessageList.getUploadRetryId`）

## 2. 二线候选（工程配方）

### 2.1 请求上下文工程（reply.ts 后半，对我们对话裁剪有直接参照价值）

- **成对轮次规则**：只有「提问 + 可用回答」成对进上下文（`getContextRounds`）；失败/拒绝/空回答轮次**整体剔除**（`isUsableReply`：done/stopped + 非 refusal + 非模板 + 有正文）
- 上下文瘦身三招：历史思考不回传；read_link 工具块带签名回传恢复网页正文；`latest_reminder` 只在内容变化时插入
- 空提问剔除：无文字且附件全部无 serverId 时连回答一起省略

### 2.2 流式渲染纪律

- **rAF 批量冲刷**（`store.ts streamReply.flush`）：流增量累积在闭包内存，`requestAnimationFrame` 合并进 zustand；`document.hidden` 时同步 flush；落盘走 `bindChatPersistence`（visibilitychange/pagehide）
- `useThrottleMarkdownRender`（40 行）：按上次渲染耗时×2 自适应节流，上限 2s
- `useReplyView`：sources 数组身份稳定化——来源未变时复用旧数组，防止文本增量绕过 markdown 节流（`useReplyView.ts:24`）
- `updateReply` 更新回答时**保留 parentId/selectedChildId**（分支字段不被流更新冲掉，`store.ts:172`）

### 2.3 并发与持久化纪律（store.ts）

- `historyEpoch` 世代令牌：导入/替换历史时 +1，在途 load/save/stream 在关键点检查 epoch，防旧请求写回新数据
- 删除三件套：`deletingSessions` + `deletionTasks` + `waitForIdle(id)`（先等流收尾落盘再删，防「删除后被收尾保存写回」）
- 会话详情 LRU=5：列表只持 `SessionSummary`（显式字段提取，防消息泄入列表），evict 保护 active/streaming/unsaved/deleting
- 保存失败面：`sessionSaveErrors` → 顶部横幅 + 重试；`pendingFiles` 保存失败保原件
- 手动重命名取消本轮自动标题（`titleRequests` set；它是服务端生成标题 `title:{enabled:true}`，我们自动命名=首问截断，但**手动重命名后不再自动覆盖**这条纪律值得对齐 `useChatTopics.ts:206`）

### 2.4 Markdown 管线（全是对 LLM 输出怪癖的防御，值得逐条对照我们 ChatMarkdown）

- `micromark-extension-llm-math`（官方微/markdown 扩展，专为 LLM 公式定界符容错）
- 禁用 `codeIndented`：前导空格列表不再被误判为代码块（`parse.ts:32`）
- 脚注/引用式链接/定义 → 显示原文（LLM 幻觉脚注不蒸发）；删除线显示原文
- 软换行 → 视觉换行（`MarkdownAstRenderer TextContent` 按 \n 拆行插 br）
- KaTeX `throwOnError` + **流式中保上次成功渲染**，无则原文（`MathFormula.ts`）
- HTML 只放行 `<br>`，其余转纯文本（XSS 安全）；图片渲染为链接
- 表格行补齐列数；GFM 任务列表手工渲染 ☑/□
- Prism 语言包按需加载 + 别名表（`highlight.ts`）；mermaid 懒加载 + 串行渲染队列 + img 标签转义 + `securityLevel: strict` + 流式「只画完整行前缀、失败保旧图」（`MermaidDiagram.tsx`）

### 2.5 搜索引用三件套（正文角标 + 悬停卡 + 右侧来源面板）

- `view.ts buildReplyView`：blocks → `{steps 时间线, answer, sources, searchResults}` 纯函数；`SourceRegistry` 同 URL 只编号一次、后续结果补齐元数据不覆盖
- 正文 `[citation:N]` → mdast 自定义 citation 节点 → `CitationLink` 角标（role=link）→ `HoverCard`（300ms 进/100ms 出延迟）悬停 `SourceCard`（站点图标/名称/日期/标题/摘要）
- 底部汇总条 `SiteThumbnails` +「已搜索 N 个网页」→ 右侧 `SearchPanel` 列表（zustand 单入口 toggle，换会话自动关）
- 时间线步骤聚合规则：普通搜索按种类全局聚合、边想边搜只合并相邻同类；read_link 仅展示进行态；折叠区标题动态轮播（正在思考/正在搜索/已思考（用时 N 秒））

### 2.6 历史 ZIP 导入导出（比 README 更硬核，蒸馏价值上调）

- 官方 `export_format_version: 3`：`manifest.json` 首条目 + 会话 JSON（含消息树 `parent_id` + `current_message_id` + `pinned`）+ 索引 + 图片原件
- **手写 zip 读写**（`zipReader/zipWriter`，fflate Inflate）：CRC 校验、拒绝 ZIP64/分卷/加密、流式分块解压控内存（`INFLATE_CHUNK_BYTES=8K`）
- 导入五步：先单读 manifest 验格式（**失败不打断正在生成的回答**）→ 停流 → 流式写 staging → 逐会话校验（消息树 id/parent 顺序、角色交替、根为提问）→ 单事务 `commitStaging` 原子替换正式数据
- 错误文案 i18n key 化保存（`localMessage`），导入自不同语言版本仍可归一展示；无法识别的片段丢弃、容器不合法整包失败
- 导入/导出/清空三操作共用 `replaceHistory` + 串行队列 `backupQueue`

### 2.7 语音链（与 VoiceConsoleV2 互补的轻形态）

- 输入框 oneshot 语音输入（`useVoiceInput`）：录音→ASR→**插入光标处**；120s 上限自动结束；转写中再点=取消；`operationRef` 世代计数防迟到回调；非安全上下文直接禁用入口
- 移动端录音时输入框文字隐藏、首行实时频谱 `Spectrum`：对数频段分段 + 4dB/倍频程倾斜补偿 + 底噪门槛（-80/-25dB）+ rAF 直改 transform 不进 React
- 录音（`recorder.ts`）：AudioContext 16k 重采样 + ScriptProcessor PCM 采集 + WAV 编码；**analyser 串在链路中间**（WebKit 不处理未连输出的分析节点）
- 朗读（`readAloud.ts`）：TTS 裸 PCM 流（24k/16bit）→ `AudioBufferSourceNode` 时间轴排程边收边播，首帧 0.1s 预缓冲，慢网**不补播过去的时间**；`ReadingBars` SMIL 波形动画两态 + reduced-motion 静态帧；服务端 `filter_markdown` 过滤代码块/表格

### 2.8 移动端与浮层

- `anchored.ts`（~200 行）：手写浮层定位——翻面/对齐/视口收口/`measureKey` 同步重测（防旧位置闪一帧）/scroll capture 全局跟踪——@floating-ui autoUpdate 的零依赖等价
- `viewport.ts`：visualViewport 跟随键盘动画（CSS 变量 `--app-viewport-height`）；`preventPageScroll` 逐层探测放行内部滚动区；双指缩放不干预
- 拖拽上传 `useFileDrop`：dragenter/leave 计数判深度 + 全屏遮罩 portal；粘贴文件直进草稿
- MessageList 滚动：STICKY_THRESHOLD=40 + **仅用户上滚才停止跟随**（`scrollTop < previous` 判方向，布局增高不打断跟随）

## 3. 它没有、我们有的（领先面，勿倒吸）

GenUI 组件渲染（35 种节点）、人格系统、语音房间 VoiceConsoleV2、12 主题、KaTeX 常驻（它懒加载）、附件 OCR、会话分页游标加载、@引用、TTS 情绪。

## 4. 不蒸馏

- 浏览器专属：IndexedDB/localStorage 适配、Files API serverId 补传环（我们 Go 侧管文件）、`file://` 双击、getUserMedia 权限分支
- 位置省市选择（服务端需要）；服务端生成标题（我们引擎无此参数，但「手动重命名后停用自动命名」纪律照学）
- 它的 mdast 手写渲染器整体替换（我们 react-markdown 管线够用，只偷 2.4 的逐条防御配方）
- Rspack/oxlint 工具链、SSE 手写解码器整体替换（我们 Go 侧已有；前端 `useChatStream` 的 EventSource/reader 若有分片 CR 边界坑可回来对照 `sse.ts:38` 的「结尾 CR 等下一段」处理）

## 5. 建议批次（拍板池）→ 2026-11-16 落地状态（第二轮续）

| 批 | 内容 | 状态 |
|---|---|---|
| A 消息树 | 分支版本树+编辑提问/重生成/版本切换 | ✅ 1f419b9d（设计 docs/gaea-chat-branch-tree-design-2026-11.md） |
| B 生命周期 | 错误分类 code+重命名停自动命名+stopped 核对 | ✅ bb8b9e39 |
| C 渲染配方 | 节流+codeIndented+mermaid 双态 | ✅ 0dd5b263 |
| **续写 continue** | 已停止行 prefill 续写拼接原行 | ✅ 9836c883（准入收窄=已停止轮；引擎 prefill 兼容性待实弹） |
| D 搜索引用 | 来源面板两件套（正文角标挂起：协议无 in-text 位置） | ✅ 6550996d |
| E 备份 | 历史 ZIP 导入导出（gaea_chat_history v1，原子替换+pre-import 自动备份） | ✅ db84badb |
| G 杂项 | 时间分组+草稿按会话保留 | ✅ 1f627beb |
| F 语音轻形态 | oneshot 输入框语音+频谱 | 挂起：native whisper 引擎按需拉起接线深+发版门须真麦烟测（repo 红线），待拍板 |
| 二轮装机 | 79ee880C1B5EABEE PID 29240 冒烟 200；FE 全量 4362/4362 | ✅ 2026-11-16 |

顺带修复：ChatTopicCreate 同毫秒连建两话题 UNIQUE 撞键（id 加原子序后缀）——续写测试实测逮到的生产可触 bug。

## 6. 首轮报告勘误

1. 「历史 ZIP 导入导出」当时只读了 README——实现是手写 zip+CRC+流式+原子提交，价值上调
2. 「搜索来源面板」当时没读 SearchPanel——实际是角标/悬停卡/右栏三件套，形态完整
3. 自动标题当时未提——它来自服务端（非本地截断），蒸馏点只在「手动重命名停用自动命名」
4. 「无 pinned」当时指我们——DeepSeek 归档格式确有 `pinned` 字段但其 demo UI 未实现置顶，双方都没有置顶交互
5. SSE 解码器、协议校验、并发纪律（epoch/删除三件套/LRU）首轮全部漏报——已补入 2.2/2.3
