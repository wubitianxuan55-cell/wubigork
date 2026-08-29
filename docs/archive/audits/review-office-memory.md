# gaea 阶段 6 独立审查报告：办公与记忆板块

审查范围：internal/office/docmd、internal/memory、internal/context、internal/gaea/evidence、internal/search、internal/stats、internal/gaea/semantic、internal/gaea/filewatch、internal/gaea/tasks；重点：CacheReport 聚合 / MergeChild 语义评估。
说明：internal/evidence 目录为空且未被 git 跟踪，真实证据实现位于 internal/gaea/evidence/；TCCA 内核（CacheReport/MergeChild）实际位于 internal/gaea/context/ 与 internal/gaea/cache/，已一并纳入审查。

## 一、TCCA 指标聚合与 MergeChild 语义（v2.25.0 留待项）

### 高危
1. **MergeChild 与 Report() 两套聚合路径语义不一致，子代理指标会丢数**
   证据：internal/gaea/context/metrics.go:56-73（MergeChild 只合并 SavedByFork/SavedByCompact/SavedUSD/SavedLatencyMs/ForkCount）与 :149-158（Report 对存活子节点聚合 SavedByCompact/SavedByFork/ForkCount/SavedUSD/SavedLatencyMs/CompactionCount）。两条路径合并的字段集合不同：MergeChild 不合并 CompactionCount，两者都不合并 CacheHitTokens/CacheMissTokens/BreakCount；且 MergeChild 对 ForkCount 加 +1（:64），Report 聚合子节点时不加（:154）。同一逻辑状态走不同路径得出不同数字。
   影响：子代理（Fork 子会话）的缓存命中/未命中、断点计数在 CacheReport 中永久丢失——这是 V5.30「全会话缓存命中统计」的语义空洞；ForkCount 依聚合方式不同而不同，报告数字不可复现。
   修复方向：收敛为单一语义——MergeChild 与 Report 聚合同一字段集合（把 CacheHitTokens/CacheMissTokens/BreakCount/CompactionCount 补进两条路径），ForkCount 的 +1 只在一个地方生效；补测试锁定「Merge 前后 Report 结果一致」。

2. **MergeChild 读取子节点字段时未加子锁（数据竞争）**
   证据：internal/gaea/context/metrics.go:56-59 只锁 m.mu，随后 :60-64 直接读 child.SavedByFork 等字段，子节点自己的 child.mu 未加锁。子代理在另一个 goroutine 里可能仍在 RecordFork/RecordCompact。
   影响：go test -race 级别的真实竞争；子代理收尾与父合并并发时指标可能读到半更新值。
   修复方向：MergeChild 内先锁 child.mu（或改为 child.Report() 取快照再合并）。

### 中危
3. **MergeChild / MergeChildEdits 在 production 中零调用，属「只测不用」的潜伏代码**
   证据：全仓 grep MergeChild、MergeChildEdits、DetectConflict 仅命中定义与测试（internal/gaea/context/metrics_test.go:76,91；internal/gaea/cache/runtime.go:200），生产链路 internal/gaea/context/manager.go:103 只调 NewChild，无任何回收合并调用。
   影响：子代理指标只能靠 Report() 的「存活子节点」路径聚合（且丢字段，见 #1）；RuntimeLayer 的冲突检测/编辑合并功能形同虚设，子代理编辑冲突永远不会被检测。此即 v2.25.0 留待评估项的答案：语义未接上生产链路。
   修复方向：在子代理回收点（agent 层 Fork 归并处）明确调用 MergeChild/MergeChildEdits，或删除死代码并简化 Report 聚合。

4. **MergeChild 可重复合并同一子节点导致重复计数，且无防重**
   证据：internal/gaea/context/metrics.go:56-73 无 merged 标记；:66-72 从 children 移除后，外部仍持 child 指针可再次调用。
   影响：同一子代理被合并两次 → SavedUSD/令牌数翻倍。
   修复方向：child 加 merged bool，二次合并直接忽略。

5. **RecordCacheTurn/RecordCacheBreak 生产代码从未调用，UI 的命中统计恒为 0**
   证据：全仓 grep RecordCacheTurn|RecordCacheBreak 仅命中 internal/gaea/context/metrics.go:106-119 定义与 metrics_test.go:41-43；Controller.TCCAReport（internal/gaea/control/controller.go:775-780）经 internal/app/gaea_ui_meta.go:204-209 直接序列化给前端。
   影响：前端「缓存命中/未命中/断点」数字永远是 0，V5.30 功能未接线。
   修复方向：在 provider 层拿到每次请求的 cache hit/miss token 后调用 RecordCacheTurn；或在 UI 隐藏未接线的字段。

### 低危
6. **MergeChildEdits 合并不设上限，父列表可突破 20 条**
   证据：internal/gaea/cache/runtime.go:220-222 直接 append；而 TrackEdit 有 20 条裁剪（:168-170）。
   影响：多子代理合并后 RecentEdits 无界增长，污染 turn-tail 注入。
   修复方向：合并后同样裁剪到 20。

### 亮点
- Report()/各个 Record 方法加锁纪律一致，父-子互不持锁递归，无死锁结构（metrics.go:129-161）。
- MergeChildEdits 冲突判定按版本号（pe.Version > ce.Version）而非时间戳，语义清晰（runtime.go:209-219）。
- 指标与 RuntimeLayer 分层干净，测试覆盖了聚合与 ForkCount+1 语义（metrics_test.go:60-96）。

## 二、internal/office/docmd（PDF 提取 / OCR / 转换）

### 中危
1. **数字型 PDF（FlateDecode 压缩文本流）会被静默降级走 OCR，且提取失败路径不可区分**
   证据：pdf.go:408-461 stripNonTextStreams 只保留含 BT+ET+Tj/TJ 的未压缩流，isTextStreamBody（:465-471）对压缩流内二进制直接判非文本 → 压缩文本流被整体剔除；随后 pdf.go:68-79 在 result 为空时无条件进入 OCR 分支。Word/WPS 导出的 PDF 绝大多数是 FlateDecode 压缩文本流。
   影响：数字型 PDF 被当扫描件处理——必须安装 poppler+OCR 引擎，且 OCR 结果比直接提取差（错字、排版丢失）；无引擎时直接报错「需要 OCR」。这是「文档→Markdown」质量的最大短板。
   修复方向：对剔除掉的 stream 做 FlateDecode（Go 标准库 compress/zlib）后再跑 BT/ET 提取；或引入成熟 PDF 库（pdfcpu/pdfium）替代手写解析。

2. **OCR 单页失败即中止整本 PDF，前面所有页成果丢失**
   证据：ocr.go:313-318 tesseract 单页失败直接 return "", err。
   影响：多页扫描件中一页不可读（空白图/损坏页），整次转换失败，已识别页全部作废。
   修复方向：单页失败记日志并跳过，继续后续页；最终若仍有文本则返回部分结果 + 警告，仅当全部页失败才报错。

3. **OvisOCR2 单页输出被 max_tokens=1024 截断，密排页内容静默丢失**
   证据：ocr.go:168 "max_tokens": 1024；A4 中文页 300DPI 渲染后识别文本远超 1024 token。
   影响：长文本页被截尾，无任何提示，进入下游的 Markdown 不完整。
   修复方向：放宽上限（如 4096），或检测 finish_reason 截断后按行重试。

4. **带 pages 规格的 OCR 进度回调脱节（预览路径 pages="" 无感知）**
   证据：ocr.go:296-326 totalOCR 固定为 last-first+1，但 :303-305 跳过不在 pages 规格内的页且 continue 前不累计 done。
   影响：format_convert 走带规格路径时进度最多走到部分百分比。
   修复方向：totalOCR 按实际要识别的页数计算。

5. **markitdown 可用性探测缓存到进程结束，安装后不刷新**
   证据：markitdown.go:22-45 sync.Once 一次性探测。
   影响：用户首次失败后装好 markitdown，本进程内 docx/pptx 仍一直回退内置解析器。
   修复方向：改为带 TTL 的缓存或失败后允许重试探测。

### 低危
6. **startOvisServer 无并发互斥，多路并发转换可能同时拉起两个 llama-server 抢同一端口**
   证据：ocr.go:30-47 ovisServerBase 与 :67-94 startOvisServer 均无 once/锁。
   影响：竞争失败方等待 60s 后自愈杀掉，浪费资源但不出错。
   修复方向：包级 sync.Once 或启动中标记。

7. **xlsx 转换在 sheet 编号断档时静默丢弃后续工作表**
   证据：office.go:396-401 第一个缺失的 sheetN.xml 即 break。
   修复方向：按 workbook 的 sheet 列表驱动而非连续编号探测。

8. **xlsx sharedStrings 富文本（r/t 嵌套）提取为空**
   证据：office.go:358-375 xml:"t" 只匹配直接子节点，带格式的 si 内容取不到 → 单元格输出空串。
   修复方向：结构改为捕获 si 内所有 t 并拼接。

9. **docx 标题样式仅识别英文样式名，中文 Office/WPS 的「标题 1」不识别**
   证据：office.go:161-186 只匹配 Title/Heading1/heading1/1 及 Heading 前缀。
   修复方向：补充「标题」「Heading 1」带空格形式。

10. **pdf.go totalPages==0 时强制置 1，可能掩盖解析失败**
    证据：pdf.go:41-43。
    影响：无 /Type /Page 的畸形 PDF 显示为 1 页且不报错。低风险。

### 亮点
- PDF 流剔除设计扎实：stripNonTextStreams 用「前导 > + 独立 endstream」双重守卫防二进制假命中（pdf.go:408-461）；页码归类锚定 /Type /Page 对象而非 BT 自增（pdf.go:145-184）。
- OCR 编排链路完整：常驻服务探测→静默拉起→健康轮询→超时杀进程树防孤儿（ocr.go:30-94），单测通过包级变量注入假进程（ocr.go:96-106）。
- 页数上限 500 + 显式截断提示（docmd.go:14、gaea_preview.go:212-214），诚实暴露截断而非静默。
- markitdown 有 60s 超时 + 空输出判定，失败回退内置解析器（markitdown.go:50-68）。

## 三、internal/gaea/tasks（任务调度边界）

### 中危
1. **终态任务进度被无条件置 100（SQL 常量恒真）**
   证据：tasks.go:536-539 progress=CASE WHEN ?=100 THEN 100 ELSE progress END 的第 5 个占位符参数是字面常量 100（Exec 实参 nowMillis(), 100, id），100=100 恒真 → 所有 failed/cancelled 任务进度也变 100。
   影响：失败任务显示「进度 100% + 失败」，前后端数据自相矛盾。
   修复方向：按 status 条件（succeeded 才置 100），或直接传任务当前进度值。

2. **用户取消在 handler 返回 nil 时被静默吞掉，任务仍标记成功**
   证据：tasks.go:449-464 switch 第一分支 handlerErr == nil → succeeded 优先于 interrupted && userCancel → cancelled；handler 若不检查 ctx 返回 nil，取消无效且显示成功。
   修复方向：把 interrupted && userCancel 分支提到 handlerErr==nil 之前。

3. **Cancel queued 任务与 worker 出队存在竞态，已取消任务可能照常执行**
   证据：tasks.go:283-307 Cancel 在锁内查 DB status 后调 markTerminal(cancelled)（无 status 守卫的裸 UPDATE），而 popQueued（:470-487）在无 m.mu 的情况下并发做 queued→running 转移；markTerminal 的 UPDATE 也不带 WHERE status（:536-539）。
   影响：取消请求与出队交错时，任务以 running 运行完并最终写回 succeeded/failed，覆盖 cancelled。
   修复方向：出队与取消都走同一把锁或带 status 条件的原子 UPDATE。

4. **handler panic 无人恢复：默认单 worker 下调度器当场瘫痪**
   证据：tasks.go:391-468 runNext 对 h(ctx, t, progress) 无 recover（全文件仅 emitView 有 recover，:602-606）；panic 后 worker goroutine 退出，任务永远停在 running，直至下次重启 resumeInterrupted（:370-389）才续跑；MaxConcurrent=1 时整个调度器本会话内不再消费任务。
   修复方向：runNext 包一层 defer recover：记日志、置任务 failed（或 requeue）、保持 worker 存活。

### 低危
5. **sem 信号量字段创建后从未使用（死代码）**
   证据：tasks.go:118,150 定义并初始化 sem chan struct{}，全文件无 acquire/release。并发上限实际由 worker 数（:179-182）决定。
6. **Submit 对 nil payload 序列化为 "null" 而非 "{}"**
   证据：tasks.go:212-215 json.Marshal(nil) 返回 "null"，pb == nil 恒 false，注释与行为不符。
7. **tasks 表无保留策略，只增不删**
   证据：无任何 DELETE/清理路径；List 仅截显示（:248-270）。

### 亮点
- 持久化状态机 + 重启续跑（tasks.go:370-389）+ 指数退避封顶 60s（:563-580），崩溃不丢任务。
- 中断判定在 cancel() 之前采样 ctx.Err()（:444-447），Close 中断与用户取消正确区分（requeue vs cancelled）。
- 事件推送带 panic 恢复与节流（:502-528, 598-608），进度落库先行、事件节流在后，顺序合理。

## 四、internal/memory + internal/search（检索质量）

### 中危
1. **章节编号断档即停止索引/搜索（首缺即断）**
   证据：memory.go:81-84 ReadChapterSummary(num) 出错即 break——缺一章 summary 则后续所有章节不进索引；search.go:45-52, 74-81 同模式（ReadChapter 出错即 break）；project.go:234-240 与 :469-479 确认文件缺失返回 error 而非 nil。dashboard（dashboard.go:70-74）同样受影响。
   影响：作者删过中间一章（编号不连续）后，后半本书从记忆中整体消失、全文搜索搜不到、仪表盘字数不全——静默数据缺失。
   修复方向：改为扫描 chapters 目录（如 ReadAllChapterSummaries，project.go:295+ 已存在）或对 ErrNotExist 特殊处理继续。

2. **全文搜索无结果上限，高频词命中可返回全库行**
   证据：search.go:133-149 searchInText 逐行匹配不设上限；internal/app/handler_search_export.go:18 直接返回 SearchAll 全部结果。
   影响：搜高频词时返回数千行，前端卡顿。
   修复方向：每源截断（如前 200 条）+ 命中计数提示。

3. **BM25 索引每次请求全量重建 + 每查询全量重分词，复杂度 O(N2)**
   证据：internal/app/context_handler.go:20,51,206 每次调用都 BuildFromProject；memory.go:124-130 Add 每次重算 avgDocLen（O(N)），:153-181 Search 对每篇文档重新 tokenize+建 tf。
   影响：章节多的项目每次语义检索/记忆注入都有明显延迟，且无并发保护。
   修复方向：索引按 project 缓存（内容变更时失效），或增量维护 docFreq/avgDocLen。

4. **记忆注入以「整段当前上下文」作为查询词**
   证据：context_handler.go:63 idx.InjectIntoContext(currentContext, ...)；memory.go:203-226 把整个 currentContext 当 query tokenize。
   影响：查询串过长使 BM25 分数被常见字主导，检索相关性被稀释（检索质量核心问题）。
   修复方向：用最近用户输入/场景关键词做查询，或对查询做长度裁剪。

### 低危
5. **tokenize 的 2-gram 跨越标点生成噪声词元**
   证据：memory.go:66-72 对全串 rune 相邻对做 2-gram，「你好。世界」会生成「好。/。世」。
   影响：docFreq/长度统计带噪，轻微劣化排序；query 与 doc 对称，尚不致命。
6. **stats.go 正则每调用编译一次 + 字数含 Markdown 语法符号**
   证据：stats.go:32,42。

### 亮点
- 零依赖 BM25 实现正确（idf 平滑 memory.go:166-169、长度归一），中文单字+2-gram 双通道分词兼顾检索覆盖。
- Search 按分数排序后取 topK，注入受 token 预算约束（memory.go:216-223）。

## 五、internal/context + 调用层（TCCA / Lorebook 注入）

### 中危
1. **空 Key 词条恒触发：strings.Contains(text, "") 恒为 true**
   证据：engine.go:132-135 对 entry.Key 无空值防护。
   影响：Lorebook 里任何漏填 key 的词条会在每轮请求中无条件注入，挤占 token 预算。
   修复方向：if rule.Entry.Key == "" { continue }，Load 时过滤。

2. **Token 预算只是「记账」，最终 prompt 组装不做总量校验**
   证据：engine.go:207-212 finalSystem 直接拼接 systemPrompt+charInfo+memoryInfo，budget 仅用于展示；FindTriggers 超预算用 break 而非 continue（:136-144），单个大词条会挡掉后续所有可装入的小词条。
   影响：超窗口时模型层截断而非有损优先保留；词条注入顺序可被单条大词条打乱。
   修复方向：组装前按 Remaining() 校验并降级；预算超限改 continue + 按优先级回填。

3. **「上文场景」预算分区从不记账**
   证据：engine.go:42 定义「上文场景」分区，但 BuildFullContext（:177-215）把 PreviousScene 拼进 userPrompt（:209-212）却从不 budget.Track("上文场景", ...)。
   影响：预算面板该分区恒为 0，用户看到的是假账。

### 低危
4. **ModelCapacity 零值陷阱被调用方兜底，但引擎自身无默认**
   证据：engine.go:178 NewBudget(opts.ModelCapacity) 直接使用；context_handler.go:114-116 由调用方默认 128000。引擎独立使用时 0 容量会让所有 limit=0、Lorebook 注入量=0。
5. **Engine.Load 无锁，若未来复用同一 Engine 跨请求会竞争**
   证据：engine.go:98 e.rules = nil 重建；当前调用方每次 NewEngine（context_handler.go:89,118,215）故暂无症状。

### 亮点
- Lorebook 按 Category 映射优先级并排序（engine.go:99-121），注入头带明确标记（:159），预算分区可视化设计清晰。
- 调用方统一兜底容量默认（context_handler.go:114-116）。

## 六、internal/stats（数据真实性）

### 中危
1. **仪表盘「每日趋势/连续天数/今日字数」是虚构数据，成就据此解锁**
   证据：dashboard.go:115-127 注释自认「简化：…实际需文件修改时间」，把最近 N 章字数按倒序硬贴到最近 N 个日期上（第 i 天显示第 N-i 章字数）；:142 StreakDays = min(ChapterCount, 30)（连续天数=章节数）；:132-135 TodayWords = 最后一章字数（今天没写也显示满）；:143-145 CompletedDays = ChapterCount；成就 seven_day_streak（:190-192）在章节数≥7 时解锁，与真实写作习惯无关。
   影响：用户依赖的「日目标进度/连续写作/成就」全部失真，属误导性数据风险。
   修复方向：DailyStats 按 chapter 文件 mtime 聚合到日期；StreakDays/TodayWords 从真实日期统计；在修复前隐藏成就系统。

### 低危
2. **stats.go 与 dashboard.go 两套统计口径并存且数字不一致（一个按目录扫描含分支，一个按连续编号）**
   证据：stats.go:29-48 与 dashboard.go:68-97。

## 七、internal/gaea/semantic + internal/gaea/filewatch（语义索引 / 实时监听）

### 中危
1. **filewatch 运行期错误（含事件溢出）没有消费侧兜底，文档承诺的「回退轮询」未实现**
   证据：filewatch.go:88-92 WatchErr/Healthy API 齐备，但消费端 internal/app/gaea_tasks.go:313-326 fileWatchLoop 只 for ev := range w.Events()，从不检查 Healthy()/WatchErr()，也不设超时轮询；fsnotify 事件溢出（filewatch.go:192-196 setErr）后丢批且无自愈。
   影响：编辑器批量保存/大目录变更时部分文件变更静默丢失，语义索引陈旧且无提示。
   修复方向：fileWatchLoop 周期检查 WatchErr 并触发一次全量重建任务（复用 submitFileIndexTask）。

2. **每次语义搜索都是全表扫描：读全部向量 + JSON 解码 + 余弦**
   证据：semantic.go:151-195 SearchReady 对 kind 全量 SELECT 并解码；Search（:126-131）每查询先 Ensure（vectorDocs 全表读，:197-211）——一次搜索两次全扫。
   影响：file kind 索引随工作区文件数线性膨胀，个人大工作区下检索延迟显著。
   修复方向：小规模可接受；若 file 索引上千条，引入倒排预筛或缓存（优化项，非紧迫）。

### 低危
3. **Ensure 对缺失 id/空文本静默跳过，调用方无感知**
   证据：semantic.go:83-94。
4. **MinCosine=0.1 阈值形同虚设（极低），基本靠 topN 兜底**
   证据：semantic.go:22,186。

### 亮点
- 内容快照比对保证「编辑过的条目自动重嵌」（semantic.go:90-93），Stale 清理删除的文档（:231-253），Remove 供删除事件实时清向量——索引一致性设计完整。
- filewatch 去抖合并 + 事件风暴升级 Full + 输出通道满时升级 Full 防阻塞（filewatch.go:155-181），目录删除/新建正确处理（:215-240），skipDirs 按名字任意深度剪枝（:137-139）。

## 八、internal/gaea/evidence（证据台账）

### 低危
1. **MatchStep 注释宣称「drift-tolerant variant」，实现只有数字+全等匹配**
   证据：evidence.go:518-523 注释 vs matchTodoStep（:440-451）仅 parseStepIndex 与 sameStepText 精确匹配（:459-463）。模型引用带细微差别的步骤文本会匹配失败。
2. **normalizePath 在 Windows 强制小写，绝对/相对路径形态不一致时可能误判**
   证据：evidence.go:486-497；工具层若返回绝对路径而 complete_step 引用相对路径（或反之），HasSuccessfulWrite 判定失败。当前 write_file/edit_file 参数键（path）与提取键匹配（evidence.go:319-330 与 writefile.go:33 一致），风险低。
3. **internal/evidence 空目录残留（未被 git 跟踪），易与 internal/gaea/evidence 混淆**
   证据：git status -- internal/evidence 无输出（目录未跟踪）；真实包在 internal/gaea/evidence/evidence.go。

### 亮点
- Receipt 记录时序正确：complete_step 的 TodoStep 在记录时对「当时最新 todo_write」解析（evidence.go:105-109），跨轮无法伪造。
- 失败 receipt 保留可审计但匹配器只看 Success（:87-91）；路径统一归一化后比对（:465-497）。
- UnverifiedCompletedTodos 的「基线缺失→宽松模式」回退语义清晰（:176-225）。

## 九、建议不做（避免过度工程）
- 语义检索引入 ANN/向量库（faiss/usearch 等）：个人写作规模（数百~数千文档）全表余弦足够，收益低于维护成本。
- 为 BM25 引入中文分词器（jieba）：单字+2-gram 已覆盖写作检索场景。
- tasks 引入多队列/优先级/分布式：单机单用户，队列粒度已够。
- 为 markitdown 可用性做进程常驻服务：cli 探测一次即够。
- docmd 集成完整 PDF 渲染引擎替代手写解析：先补 FlateDecode（标准库即可）观察效果，不要一步到位换引擎。

## 十、下一阶段优先级建议（按投入产出）
1. 修 tasks 终态进度恒 100 与取消竞态（低风险高收益）。
2. 统一 MergeChild/Report 聚合语义或删除死代码，接上子代理回收链路。
3. 修「章节断档即停」：memory/search/dashboard 全部改用目录扫描。
4. 补 PDF 压缩流解压，解决数字型 PDF 被误当扫描件。
5. dashboard 每日统计改真实文件 mtime，或先隐藏成就。
6. filewatch 消费端补 WatchErr 兜底。