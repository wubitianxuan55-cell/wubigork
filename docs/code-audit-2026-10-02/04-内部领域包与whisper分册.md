# 内部领域包与 whisper 分册

> 元信息：覆盖单元 IN1 / IN2 / IN3 / IN4；证据来源 `docs/code-audit-2026-10-02/units/IN1.json`、`IN2.json`、`IN3.json`、`IN4.json`；审计日期 2026-10-02。
> 合并口径：4 个单元的原始 `findings` 合计 79 条（IN1 20 / IN2 20 / IN3 20 / IN4 19），本分册把其中被拆成「同根因两个切面」的少数条目展开为独立 U-ID（如 core/office 平迁在 coupling 与 duplication 各记一次、角色头像本地化两套命名各记一次），最终 **84 条**（U1–U87，缺号 U14/U18/U33 为合并后空位）。因此本板块的重复不是「同一条被两个审计员各报一次」的记账重复，而是**同一语义在不同包/不同文件里各留一份真实实现**，每条都需独立登记、独立消刀。
> 严重度分布（按 84 条 U-ID 逐条对齐单元 JSON 的 `severity` 字段核验）：**P0 = 1、P1 = 49、P2 = 28、P3 = 6**。逐单元 = IN1 23 条（P1 12 / P2 10 / P3 1）、IN2 23 条（P0 1 / P1 14 / P2 8）、IN3 20 条（P1 12 / P2 5 / P3 3）、IN4 18 条（P1 8 / P2 8 / P3 2）——同一源 finding 被拆成两条时两条继承同一严重度，故单元条目数大于该单元 findings 数。
> 编号（U-ID）全篇唯一；`二、屎山 TOP 榜` 展开前 20 名。「延续」表示与历史审计（`docs/gaea-optimization-direction-2026-09.md` / `docs/gaea-whisper-deadcode-survey-20260925.md` / CHANGELOG 记录）重合。
> 行号与证据全部取自单元 JSON，未自行编纂；行号即复核坐标。

## 一、板块总评

| 单元 | 板块 | 评分 | 一句话总评（取自单元 JSON 的 score/verdict） |
| --- | --- | --- | --- |
| IN1 | 内部小说领域包（character/characterlib/characterstate/chapter/outline/narrative/novelcontext/novelgate/novelreview/novelstyle/style/worldview/scene/rewrite/project/bookimport/booksource/export/maturecraft） | 4 | 功能闭环、注释密度极高，但「同一件事有两到四套实现」：project.Manager 1011 行兼 8 类关注点且是各域写盘咽喉；判据口径分裂（novelreview 段落 / novelgate 句子 / novelstyle blocker 三套严重度）；角色域三包各写一遍读-改-写与事件解析，project 侧同样的 RMW 无锁。 |
| IN2 | 内部底座包（ai/modelengine/config/netclient/httpbridge/auth/core/types/util/prompt/promptstore/intent/context/memory/chat/stats/snapshot/analysis/graph/herdsman/skill*/search/routesuggest/assistant） | 5 | 单包内纪律尚可，但「同一件事多套实现」与「该在上层的下沉到底座」两条债集中：底座反向 import CLI 层 gaea/*；图片后端 switch 在 app 抄 5 遍；core 是 office 的逐行平迁副本；ai.Client 1429 行兼 9 类关注点。另有 3 处可核实的数据完整性/安全缺陷（ExtractJSON 取末个 `}`、SSRF 校验全部只拨 ips[0]、两套配置同根）。 |
| IN3 | 内部办公/解析/日程包（office/docmd/schedule/ocr/screen/export） | 4.5 | 不是「脏」而是「八套半个解析器叠在一起」：同一 cd 五闸校验、同一日历语义、三种工程导入口径各写一份；文档域四路解析（自研 PDF 扫描、自研 OOXML 字符串手术、markitdown 子进程、poppler+两个本地 OCR）且每路失败静默换下一路；三处 god-file（mpp.go 924 / docxedit.go 827 / pdf.go 705）。 |
| IN4 | internal/whisper | 4 | 19,510 行是 ackem TypeScript 上游的逐文件 Go 转录（111/123 文件带 ackem 标记）；最痛是「平行实现 + 未接线」叠加：情绪两套标签表、情绪涌现双状态源、桌面助手三条路由中两条 0 生产调用；安全侧 22 种操作一个 95 行 switch、4 层沙箱出口被 downloadAndInstall 绕过、normalizePath 注释称禁 `..` 逃逸而实现无 containment 校验。 |

板块整体：**4.4 / 10**。四个单元共享同一条主线病灶——**同一语义存在 2~5 份实现，且没有任何一份被登记为权威口径**。它在本板块表现为四种形态：① 原子写 / 章节号 / 正则 / 小工具这类「复制粘贴型」重复（IN1、IN2、IN3）；② 判据型重复但结论相反（gate 与 review、两套文风、两套 BM25、两套意图判定）；③ 迁移植入型重复（IN2 的 core/office、IN4 的 ackem 转录）；④ 未接线型重复（IN4 三条路由中两条死路）。第四种最危险：死代码与活字段互锁（U73）、全局状态与实例字段并存（U50），静态分析找不出孤立。

## 二、屎山 TOP 榜

| # | U-ID | 严重度 | 标题 | 位置(file:line) | 类别 | 证据摘要 | 修法 | 工作量 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | U17 | P0 | ExtractJSON 取最后一个右花括号，多对象即解析失败 | internal/util/util.go:82-96 | error-swallow | `start, end := -1, -1`；`if ch == '}' { end = i }`——start 只记首个 `{`，end 每遇 `}` 无条件覆盖 | 改括号配对扫描（depth 0→1 记 start、1→0 即返回），跳过引号内字符与转义；补「两对象拼接 / 尾部多 } / 字符串内含 }」单测 | S |
| 2 | U1 | P1 | project.Manager：8 类关注点挤 1011 行 | internal/project/project.go:154-201 | god-file | `func (m *Manager) ReadWorldviewFile() (*types.WorldviewFile, error)` 同类型兼元数据/世界观迁移/章节扫描/lorebook/记忆/改写版本库/标注 | 拆 meta/fileLayout/worldviewStore/memoryStore 四个类型，Manager 只留 Dir+组合；先下沉纯函数（leadDigits/mainChapterSummaryNum） | L |
| 3 | U2 | P1 | 原子写三套实现，非原子写散落五处 | internal/project/project.go:674-699 | duplication | project.writeFileAtomic（有 fsync 无 MkdirAll）/ narrative.writeFileAtomic(journal.go:203，反之) / fileutil.AtomicWrite(atomic.go:19，两者都无) | 删前两份，统一调 fileutil.AtomicWrite 并在 fileutil 内补 fsync；scene/style/export 写入口改调同一函数 | M |
| 4 | U30 | P1 | 底座反向依赖 CLI 层：建客户端要读 gaea TOML 配置 | internal/ai/client.go:114-127 | coupling | `gcfg, err := gaeacfg.Load()` 出现在每次 NewClient 的代理取值路径上 | ProxySpec 上提到 app 装配层经现成 `proxySpecOverride` 注入；或把 NetworkProxySpec 类型下沉 netclient，config 只做映射 | M |
| 5 | U4 | P1 | 图片后端解析 switch 在 app 层被抄 5 遍 | internal/app/app.go:605-639 | duplication | `case "herdsman": … a.client.SetImageBackend(backend, "herdsman")`，同段在 image_handler.go:840/1020、characterlib_gen_handler.go:629、writing_state.go:60 各一份 | 收敛为 `resolveImageBackend(name)` 走 `ai.NewImageBackend(kind, cfg)` 注册表，5 个调用点各留一行 | M |
| 6 | U31 | P1 | core 是 office 的逐行平迁副本，两处并存 | internal/core/fs.go:1-28 | coupling | 文件头自述「自 internal/office/executor.go 平迁，逐行等价」，office 现仅剩 aliases.go 8 行 | 二选一：删 office 转发层让调用方直连 core，或把 core 合并回 office；禁止长期并存 | S |
| 7 | U22 | P1 | SSE 解析 panic 只写日志，调用方收到空成功 | internal/ai/client.go:686-693 | error-swallow | `if r := recover(); r != nil { slog.Error("SSE 流解析 panic", …) }`，不投递错误帧 | recover 分支内 send 一个带 Error 的 SSEChunk（close 前 select 保护），并标记 usage success=false | S |
| 8 | U25 | P1 | Client 一个类型兼 9 类关注点，1429 行 | internal/ai/client.go:27-69 | god-func | 结构体自述「mu 保护 activeEngineID/imageBackend/token」+ 单飞/信号量/双路重试/SSE/failover/usage | 拆 tokenSource、streamTransport、usageReporter、imageRegistry；先拆 usageReporter 与 streamTransport | L |
| 9 | U3 | P1 | 章节编号解析四套口径，导出/统计/项目各写一遍 | internal/export/export.go:70-81 | consistency | `re := regexp.MustCompile(`^(\\d{3})([a-z]?)\\.md$`)` 与 project.mainChapterSummaryNum(452)/leadDigits(550)/novelcontext.chapterNumFromFile(874) 并存，迁移路径 project.go:783 是第 5 套 | 在 project 暴露唯一 `ParseChapterFileName(name) (num, branch, ok)`，export/stats/novelcontext/bookimport/迁移全改调 | M |
| 10 | U61 | P1 | applyOne 单函数 430 行巨型 switch | internal/schedule/ops.go:106-538 | god-func | `func applyOne(p *Project, op Op) (string, error) { switch op.Type {`——12 种 op 全内联，patch_task 独占 90 行 | 按 op 拆 applyUpsertTask/applyPatchTask/… 到独立文件，用 `map[string]func` 注册表分派 | M |
| 11 | U70 | P1 | PreLLMTurn 325 行上帝函数串 14 个子系统 | internal/whisper/orchestrator.go:129-454 | god-func | `func (o *Orchestrator) PreLLMTurn(userMsg string) PreLLMResult` 内联会话复位/涌现恢复/L0/DnD/成人模式 FSM/psycheBlock 累加 | 按已有注释分段抽 o.stepL0 / o.stepAdultMode / o.stepEmergence / o.assemblePsycheBlock，主函数只留调用序与状态提交 | L |
| 12 | U72 | P1 | 22 种桌面操作巨型 switch，零权限与路径校验 | internal/whisper/desktop_executor.go:40-134 | safety | `func ExecuteDesktopAgentAction(action DesktopAgentAction, path, pathTo, target, query, url, content string, ctx …)` 7 个 string 位置参数 | 拆成 `map[DesktopAgentAction]func(args UseComputerArgs, ctx …)`，args 用结构体；checkActionSettings/evaluatePathPolicy 下沉本层做纵深防御 | M |
| 13 | U74 | P1 | downloadAndInstall 在沙箱外直接执行下载物 | internal/whisper/desktop_executor.go:106-120 | safety | `dl := downloadHTTPS(url, dest)` 后立即 `shellOpen(dest)`，不经过 AlwaysConfirmActions 确认回调 | 删掉分支内自动 shellOpen，改为返回需确认状态；或 executor 入口直接拒绝该 action，只允许携带 AlwaysConfirm 结果调用 | S |
| 14 | U28 | P1 | SSRF 守卫校验全部解析 IP 却只拨 ips[0] | internal/netclient/ssrf.go:68-73 | concurrency | 遍历 `ips` 判 BlockedInternalIP，末行却 `DialContext(… net.JoinHostPort(ips[0].IP.String(), port))` | 只拨已校验通过的 IP（遍历中取第一个非 Blocked 直接建连），或用 `net.Dialer.Control` 在 connect 前复核实际目标 IP | S |
| 15 | U50 | P1 | 情绪涌现双状态源：包级全局平行会话状态 | internal/whisper/emotional_emergence.go:11-15 | concurrency | `var ( recentEventTypes []string; consecutiveMeaningfulCount int` 与 orchestrator.go:59-61 实例字段同语义 | 删包级三变量与 8 个 0 调用函数（Push*/Get*/ResetEmergenceTracking），会话状态一律走 Orchestrator 字段 | S |
| 16 | U78 | P1 | 桌面能力有 3 条路由，2 条 0 生产调用 | internal/whisper/dispatch_router.go:81-167 | dead-code | `RouteDispatch` 全仓仅命中 dispatch_router_test.go 12 处；`ResolveDesktopCapabilityEnhanced` 仅命中 desktop_test.go 8 处 | 确认产品口径后二选一：删 dispatch_router.go(284)+desktop_capability_routing.go(289) 及其测试，或接线进 whisper_handler 并删另一条 | M |
| 17 | U34 | P1 | 三套 BM25 实现，三份分词器 | internal/memory/memory.go:27-47 | duplication | 本包 k1=1.5/b=0.75 自带 tokenize，另与 gaea/memory/search.go(k1=1.2)、gaea/bm25/bm25.go 并存，novelstyle/segment.go:200 是第 4 个 tokenize | 以 gaea/bm25 为唯一实现，memory.Index 改薄封装并删本地 docFreq/tokenize；分词统一走 util 级 Tokenize | M |
| 18 | U6 | P1 | gate 与 review 判据重复且门槛相反 | internal/novelgate/gate.go:26-32 | consistency | `shortSentenceRunes = 5 / shortRatioLimit = 0.40 / minSentences = 20` 与 review 段落版判据并存，同正文可能 gate 报错而 review APPROVE | 把 gate 句级/段级判据登记为 rubric.json 维度，gate 改为「只跑 rubric 硬判据子集」并输出同一 Dimension；splitSentences/splitParagraphs 抽共用 | L |
| 19 | U8 | P1 | 角色域三包各写一遍读-改-写与事件解析 | internal/character/character.go:474-513 | duplication | SaveCharacter/DeleteCharacter/SaveOrganization/…/mergeCharacters 至少 10 处「ReadCharacters→改→WriteCharacters」，同句 slog.Warn 逐字出现 6 次 | 包内收口 `readModifyWrite(func(*types.CharacterFile) error)` 泛型助手，十处改调；characterstate 复用同一索引 | M |
| 20 | U12 | P1 | project 侧读-改-写无线程安全，锁只在 app 层 | internal/project/project.go:901-919 | concurrency | `UpsertAnalysisV2` 与 `syncRewriteIndex`(1014) 整表覆盖；app 层 writingState.mu 只护 getPM/setPM/closePM（writing_state.ts 同层 writing_state.go:20-43） | 照 plans.json 做法把互斥下沉 project.Manager（每文件一把 sync.Mutex），RMW 全在锁内；app 侧 chapterPlanMu 随之删除 | M |

## 三、分类清单

> 每条给出：U-ID / 严重度 / 位置 / 逐字证据（≤3 行）/ 影响 / 修法 / 工作量。**共 84 条**（U1–U87，缺号 U14/U18/U33 为单元内同根因合并后的空位），按类别分组；每条 U-ID 全篇唯一，`二、屎山 TOP 榜` 中的重复出现是指针而非独立条目。

### god-file（上帝文件，8 条）

**U1 [P1] project.Manager：8 类关注点挤 1011 行** — `internal/project/project.go:154-201`（延续）
证据：`func (m *Manager) ReadWorldviewFile() (*types.WorldviewFile, error) {` / `wf, err := loadJSON[types.WorldviewFile](..."worldview.json")`。
同一类型承担项目元数据（Create/Open/Close/WriteMeta 20-129）、世界观结构化迁移（134-206）、章节/分支/摘要命名与扫描（280-564）、lorebook、上下文构建（586-666）、v3→v4 迁移（749-844）、故事记忆（921-979）、改写版本库（981-1084）、标注（1086-1109），且它是 textbook/analysis/export/stats 所有域的写盘入口。历史审计已记 project 侧写盘原子化欠账，本次确认职责未拆。
影响：加一种章节文件后缀要改 4 个扫描/解析函数并复核 6 个调用方；读者无法从类型判断方法归属，测试只能整包驱动。
修法：按存储域拆 4 个文件 + 4 个类型（meta/fileLayout/worldviewStore/memoryStore），Manager 只保留 Dir + 组合；先拆纯函数（leadDigits/mainChapterSummaryNum）零风险。工作量 L。

**U15 [P1] novelcontext 一个文件承担四件事** — `internal/novelcontext/novelcontext.go:144-249`
证据：`reserve := maxRunes * tailReserveRatioNum / tailReserveRatioDen` / `style := clipToRunes(strings.TrimSpace(b.Style), styleBudget, reserve)` / `thread := clipToRunes(strings.TrimSpace(b.Thread), threadBudget, reserve)`。
Render(144-249) 是 105 行顺序敏感的预算配账流水线，同文件还有 collectSceneEntities(421-475)、buildPOVMask/povKnowsFact(650-723)、buildSceneChars(516-564)、synthesizeChapterScene(801-858)、inferLocation(889-904)。
影响：改预算系数要读懂 105 行顺序敏感代码；改 POV 判定要确认与角色块兜底不互相抵消；单测只能整段编译再断言字符串，无法单测「额度不足时跳过哪段」。
修法：拆 budget.go（Render+clipToRunes+sectionCost，输入=已算好的各段文本）、retrieval.go（子图+角色索引）、povmask.go（掩码规则+可观测键表）；CompileSceneBible 只做编排。工作量 M。

**U23 [P1] modelengine/stats.go 兼 9 类职责** — `internal/modelengine/stats.go:308-322`
证据：`type statsRecorder struct {` / `mu       sync.Mutex` / `lastSave time.Time // 上次落盘时间（节流用）` / `pending  int       // 自上次落盘以来累计的调用数（节流用）`。
758 行内同时装：内置定价表（141-197 共 55 条）、normalizeModelID 归一、GLM 目录价查询、用户自填价目优先层、USD/CNY 汇率三态守卫、statsRecorder 懒加载+节流落盘+版本兼容、三维桶聚合与排序、PerFeature 账目行、编码套餐计费口径判定（glmCallBilling）、Manager 上 6 个导出包装。
影响：改一条定价、调节流参数或加一个聚合维度都在同一文件，回归面覆盖计费数字与持久化格式；旧版兼容（`f.Version<3` 重键）与写入口径（`Version: 3` 硬编码）分散在 load/save 两处，升 v4 极易只改一处。
修法：按域拆 pricing.go（定价表+estimatePrice 系列）、store.go（statsRecorder 持久化与节流）、aggregate.go（summary 聚合与排序）、manager_stats.go（导出包装）；纯结构搬移，行为零变化。工作量 M。

**U64 [P1] mpp.go 924 行含 4 套版本字节布局** — `internal/schedule/mpp.go:310-333`（延续）
证据：`switch {` / `case is2013:` / `// 2013+ 变体:行内 var 键整体换 ID 键空间` / `t.uid = t.id`。
单文件承担 CFB 遍历、CompObj 探测、Props/VarMeta/Var2Data/FixedMeta/FixedData 五原语、MPP9/12/14/2013+ 四套偏移与 26 个魔法常量（38-57、153、226、302 等）。CHANGELOG v4.154.0 修过 2013+ 五处漂移，结构未拆。
影响：任何偏移修正要同时判断 4 支；无真机样本时只能靠合成流表单测，格式漂移只能等用户报错（v4.154.0 的 2013+ 五处漂移即如此暴露）。
修法：按版本拆 mpp9.go/mpp12.go/mpp14.go，字段偏移提成 per-version 表，主流程只查表；至少把 2013+ 变体独立成文件。工作量 L。

**U67 [P1] docxedit 827 行三套 XML 扫描叠加** — `internal/office/docxedit/docxedit.go:406-512`
证据：`func parseParagraphs(data []byte) ([]paragraph, error) {` / `dec := xml.NewDecoder(bytes.NewReader(data))`。
同文件内 parseParagraphs（406-512 主扫描）、parseChangeSpans（313-359 第二遍扫描）、maxWID（805 正则第三遍扫描）三套解析叠加，之下还有 zip 读写与 619-758 的逐 run 字节重建。
影响：定位选区、扁平化修订、生成新修订 id 三步基于三份互不共享的解析结果，任何 OOXML 结构变体（嵌套 run/域代码）都要在三处分别理解，且只有端到端测试。
修法：合并为一次 token 遍历产出一份段落模型（含 changeSpan 区间），w:id 分配移到模型上。工作量 L。

**U43 [P1] 图片后端注册表与 5 份手写 switch 并存** — `internal/app/app.go:605-639`
证据：`case "herdsman":` / `eng, ok := a.engineMgr.GetEngine("herdsman")` / `backend := ai.NewOpenAIImageBackend(eng.BaseURL, eng.APIKey)` / `a.client.SetImageBackend(backend, "herdsman")`。
ai 包已提供注册表（image_backend.go:42-67 RegisterImageBackend/NewImageBackend/ImageBackendKinds），app 全部绕过注册表直接 new 具体类型；5 份副本散在 app.go:605-639、image_handler.go:840-890（SetImageBackend）、image_handler.go:1020-1030、characterlib_gen_handler.go:629-660（buildImageClientFor）、writing_state.go:60-80。
影响：新增后端（或改一个后端鉴权方式）要同步改 5 处，漏一处即「设置页能选、生成时静默走 xAI」；注册表 + 两种注入路径（字段注入 SetImageBackend / 直接构造）并存，5 份副本里引擎未启用时的报错文案逐字不同。
修法：把 5 份 switch 收敛成 app 层一个 `resolveImageBackend(name) (ai.ImageBackendConfig, string, error)`，内部走 `ai.NewImageBackend(kind, cfg)`；5 个调用点只保留一行调用。工作量 M。

**U59 [P1] pdf.go 705 行手写解析器，stream 扫描两份** — `internal/docmd/pdf.go:521-574`
证据：`func stripNonTextStreams(s string) string {` / `var b strings.Builder` / `pos := 0`。
decodeFlateStreams(416-466) 与 stripNonTextStreams(521-574) 各实现一遍「向前找 '>' 判定真 stream、向后找独立 endstream、跳过 body 换行」，且成功后的 pos 前进方式不同（一处写 `s[pos:bodyStart]+dec`，一处写 `s[pos:i]`）；另有自研德压、stream 切分、BT/ET 提取，不支持 xref/ObjStm，页数靠 `/Type /Page` 字面计数。
影响：stream 边界规则修正必须改两处，漏改则解压路径与剔除路径对同一 PDF 得出不同 stream 划分，表现为「解压过的 PDF 漏文本或多垃圾」的偶发差异。
修法：抽 `streamSpans(s) []span` 迭代器，两条路径共用；同时在用户可见处诚实标注不支持 xref/ObjStm。工作量 M。

**U29 [P1] config 包配置装配 100+ 手写 if** — `internal/config/config.go:1-539`（热点）
证据（热点自述）：配置装配 = 默认值 + 17 个环境变量 + 100+ 个手写 if 覆盖 + 损坏备份 + 资源目录解析，全部手抄展开；新增一个配置项要改 5 个文件。
影响：配置项增删改的改动面固定为 5 处（defaults、env 读取、文件覆盖、校验、绑定导出），漏一处即「设置里改了不生效」或「环境变量被文件覆盖」。
修法：把配置项改成声明式表 `[]configField{Key, Default, Env, Type}` 由装配器循环应用，defaults/env/覆盖/校验四步全表驱动。工作量 M。

### god-func（上帝函数，5 条）

**U25 [P1] Client 一个类型兼 9 类关注点，1429 行** — `internal/ai/client.go:27-69`（延续）
证据：`type Client struct {` / `// mu 保护以下可变状态：activeEngineID、imageBackend/imageBackendType、token。` / `// 读写必须经锁；持锁期间不得调用可能再取锁的方法（避免死锁）。` / `mu sync.RWMutex`。
一个结构体挂 token 获取/刷新单飞（GetToken/tryRefreshToken/EnsureToken）、并发信号量（sem/acquireSem/releaseSem）、非流式退避重试（chatOnce）、流式退避重试（doStreamRequest）、SSE 解析与工具调用拼装（parseStreamEvents/flushToolCalls）、空闲超时 Body（idleTimeoutBody）、引擎故障转移（failoverTarget/OnFailover）、usage 记账（recordUsage/cacheSplitForUsage）、图片后端切换（SetImageBackend/GetImageBackend）。
影响：改任一条链路（例如给流式加重试次数）都要在 1400 行里定位并重推锁语义；failover 与信号量的交互已出现硬约束（Chat 注释 421-427：不能递归调用 Chat，否则外层 defer releaseSem 未执行会占双槽），说明并发边界已难以本地推理。历史审计的上帝文件清单已列，仍开放。
修法：按域拆独立协作对象：tokenSource（含单飞刷新）、streamTransport（请求+退避+SSE）、usageReporter、imageRegistry；Client 只做组装与委托，先拆 usageReporter 与 streamTransport 收益最大。工作量 L。

**U19 [P2] ReviewChapter 与 fallback 流水线全同** — `internal/chapter/chapter.go:145-204`（延续）
证据：`func (a *Agent) ReviewChapter(ctx context.Context, chapterContent string, outlineNodeTitle string, prevChapterHint string) (*ChapterReviewResult, error) {` / `tmpl := a.eng.Get("chapter-review")` / `if tmpl == nil {`。
ReviewChapter(145-175) 与 reviewChapterFallback(177-204) 除 systemPrompt 来源外逐行同构：同样 ChatSimpleStreamWithOptions(Temperature 0.15/MaxTokens 2048)、同样 util.ExtractJSON + json.Unmarshal、同样返回类型。区别只有模板缺失时内联一段 prompt。
影响：两条路径各自漂移（fallback 已丢 `TimeoutMinutes:5`）；prompts 目录缺文件即走另一套口径，行为差异对作者不可见。历史审计已记该 fallback 并标注 v4.429 补模板，fallback 分支本身仍在。
修法：合并为 `reviewWithPrompt(ctx, sysPrompt, userPrompt)`；模板缺失时把内联 prompt 作为 systemPrompt 常量传入，其余流程唯一。工作量 S。

**U61 [P1] applyOne 单函数 430 行巨型 switch** — `internal/schedule/ops.go:106-538`
证据：`func applyOne(p *Project, op Op) (string, error) {` / `switch op.Type {` / `case "upsert_task":`。
12 种 op 类型（53 行注释枚举）全部内联在一个函数，每个 case 自带校验+变更+回执文案，最长 patch_task 占 90 行；函数内还复制 Validate 与自身 upsert/patch 三份 cd 校验。
影响：新增 op 要改这个函数任意位置；任何 case 都无法单独测试（只能经 ApplyOps 全链路），diff 冲突面 430 行起。
修法：按 op 拆 applyUpsertTask/applyPatchTask/… 到独立函数或文件，用 `map[string]func` 注册表分派。工作量 M。

**U70 [P1] PreLLMTurn 325 行上帝函数串 14 个子系统** — `internal/whisper/orchestrator.go:129-454`（延续）
证据：`func (o *Orchestrator) PreLLMTurn(userMsg string) PreLLMResult {` / `// v4.3a: 重启后首回合回填关系记忆（关联索引/习惯库）并重建关联图（fail-open）。` / `// 必须在读 State 之前执行——State.Associations/Habits 已由 restoreWhisperState 装载。`。
单个函数内联会话复位、涌现恢复、L0 事件解释、DnD 习惯写入、反差人格渐变、L1/L2 递推、重逢 boost、沉默/屏障、成人模式 FSM、情绪涌现、欲望栈、节奏决策、psycheBlock 十几次字符串累加、Tier A/B 记忆组装、运行上下文、追踪落盘；无任何私有方法拆分，局部变量 state/newState/o.State 三者交替写。
影响：每回合必经路径；任何子系统改动都要在这 325 行里定位插入点，函数级测试无法单独覆盖子系统，改动回归面=整条对话链。
修法：按已有注释分段抽私有方法（o.stepL0、o.stepAdultMode、o.stepEmergence、o.assemblePsycheBlock、o.assembleSystemPrompt），PreLLMTurn 只保留调用序与状态提交；先抽纯计算段，字符串累加段合并为一组 struct 后再渲染。工作量 L。

**U71 [P1] 图片后端解析 5 份副本（god-func 视角）** — `internal/app/app.go:605-639`
见 U43：同一根因，本条按「函数级重复」登记——5 份逐行同构的 30 行 switch 函数，任意一份的 case 分支漂移都不会被编译期发现。修法同 U43。工作量 M。

### duplication（复制粘贴与平行实现，16 条）

**U2 [P1] 原子写三套实现，口径不一致且非原子写散落** — `internal/project/project.go:674-699`（延续）
证据：`func writeFileAtomic(path string, data []byte) error {` / `dir := filepath.Dir(path)` / `tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")`。
三份：project.writeFileAtomic（有 fsync 无 MkdirAll）、narrative.writeFileAtomic（journal.go:203，有 MkdirAll 无 fsync）、fileutil.AtomicWrite（atomic.go:19，两者都无）。同域非原子写裸奔五处：scene.Write/writeMeta（scene.go:113/258 直接 os.WriteFile 正文与元数据）、style.SaveProfile（profile.go:143 os.WriteFile）、export 的 TXT/MD 落盘（export.go:136/181）、characterlib 剧照（portrait.go:114）。四个文件（project/plan_store/story_spine_store/analysis）都写着「原子写」注释，实际实现三个版本。
影响：崩溃/断电或第二客户端并发写同一场景或风格档案时可能读到半截 JSON 或半个正文；fsync 语义在三条路径上不同，无法用一句「落盘是原子的」作为安全前提。CHANGELOG.md:6446 记录「重建 fileutil.AtomicWrite，5 处手写重复实现收敛，净删 41 行」与 v4.426.0 sin 四写点收敛，本板块 project/narrative 两处漏网仍开放。
修法：删 project.writeFileAtomic 与 narrative 的同名实现，统一调 fileutil.AtomicWrite（缺 fsync 就在 fileutil 内补 fsync，一处生效三处受益）；scene/style/export 的写入口改调同一函数。工作量 M。

**U4 [P1] 图片后端解析 switch 在 app 层被抄 5 遍** — `internal/app/app.go:605-639`
见 U43（god-file 视角）与 U71（god-func 视角）。本条为 canonical 条目：`case "herdsman":` / `eng, ok := a.engineMgr.GetEngine("herdsman")` / `if ok && eng.Enabled {` / `backend := ai.NewOpenAIImageBackend(eng.BaseURL, eng.APIKey)`；5 个调用点：app.go:605-639、image_handler.go:840-890、image_handler.go:1020-1030、characterlib_gen_handler.go:629-660、writing_state.go:60-80。
影响与修法同 U43。工作量 M。

**U34 [P1] 三套 BM25 实现，三份分词器** — `internal/memory/memory.go:27-47`（延续）
证据：`// Index BM25 索引` / `type Index struct {` / `docFreq    map[string]int // 词 → 包含该词的文档数` / `func tokenize(text string) []string {`。
internal/memory（k1=1.5/b=0.75，unicode 类别 + 中文 2-gram）、internal/gaea/memory/search.go（SearchIndex，k1=1.2/b=0.75，另有 substring fallback）、internal/gaea/bm25/bm25.go（Ranker/Cache，被 gaea/cost 复用）各写一套 BM25 与一套 tokenize；internal/novelstyle/segment.go:200 还有第 4 个 tokenize。
影响：同一句查询在「小说故事记忆检索」（context_handler 用 internal/memory）与「办公记忆检索」（gaea memory_search 工具）下排序不同，用户看到的「相关度」不可比；分词规则改一处（停用词/中文 2-gram）另两处不知道，中文检索质量会分叉；三条路径评分参数还不一致（1.5 vs 1.2），属不可见的行为差异。whisper 死代码调查提过重复记忆实现，仍开放。
修法：以 internal/gaea/bm25 为唯一实现（它是被 cost 复用的那个），把 internal/memory 的 Index 改成对它的薄封装并删掉本地 docFreq/tokenize；分词器统一走一个 util 级 Tokenize。工作量 M。

**U45 [P1] 五套行级重复：db 前缀/原子写/错误解析/JSON 工具/BM25 之外的整包复制面** — `internal/core/fs.go:1-28`
证据：`// Package core — 桌面 agent 文件读写执行器（自 internal/office/executor.go 平迁，逐行等价）。` / `package core`。
见 U31（coupling 视角）：core 的 5 个源文件（fs.go 258 / jobs.go 53 / routing.go 25 / types.go 59 / modes.go 52）全部是 office 的平迁副本，属「整包复制」这一 duplication 最重形态。
影响/修法同 U31。工作量 S。

**U8 [P1] 角色域三包各写一遍读-改-写与事件解析** — `internal/character/character.go:474-513`
证据：`cf, err := a.pm.ReadCharacters()` / `if err != nil {` / `slog.Warn("角色 Chat: 读取角色失败", "error", err)`。
「读 characters.json → 按 ID 找到 → 原地替换/追加 → 整表写回」在 character.go 至少 10 处：SaveCharacter(474)、DeleteCharacter(497)、SaveOrganization(524)、DeleteOrganization(543)、SaveRelationship(562)、DeleteRelationship(581)、ToggleOrgMember(600)、SetCharacterCareer(739)、RemoveCharacterCareer(787)、mergeCharacters(684)；characterstate.ApplyChapterDiff 又用另一套按名字索引的就地改写（characterstate.go:86-93、146-171），characterlib.ProjectCharactersForNovel(store.go:637) 再走 SQL 侧一条。同一句 slog.Warn("角色 Chat: 读取角色失败") 逐字出现 6 次。
影响：任何并发写（角色面板 + 章节差分 + 批次生成）都是读-改-写竞态：后者把前者刚写的字段整表盖掉；新增角色字段要改 10 处的零值处理（`cf==nil` 分支只在少数几处补齐）。
修法：在 character 包内收口 upsert/delete 泛型助手 `readModifyWrite(func(*types.CharacterFile) error)`，十处改调；characterstate 已有的名字索引改为从该助手内暴露，避免两套索引。工作量 M。本条同时是 U12 之外的第二条并发口径，也与 U32 的「唯一仓储」决定绑定。

**U47 [P2] 角色库 29 列 SELECT 清单手抄 5 份** — `internal/characterlib/store.go:289-301`
证据：`row := x.QueryRow(` / `SELECT id, name, kind, gender, age, tags, portrait_url,` / `reference_images, gallery_images, reference_scores,`。
同一段 29 列清单出现在 getOn(288)、FindByName(307)、List(372)、DrawRandom(610)、ProjectCharactersForNovel(641，带 `c.` 前缀变体），对应的 Scan 顺序在 scanCharacter(687-725) 与 ProjectCharactersForNovel(659-667) 再抄两遍。列数/顺序既是 SQL 正确性前提也是 JSON 反序列化前提。
影响：加一列（如新的角色字段）要同步 5 处 SELECT + 2 处 Scan，漏一处即运行时 Scan 参数个数不匹配（报错落在用户路径而非编译期）；变体前缀差异进一步放大手工成本。
修法：抽 `characterColumns(prefix string) string` 与 `scanCharacterRow(scanner) (*Character, error)` 单点，5 处 SELECT 与 2 处 Scan 全部复用。工作量 S。

**U65 [P1] docxedit↔pptxedit 整套函数复制粘贴** — `internal/office/pptxedit/pptxedit.go:641-691`
证据：`func textElement(elem, text, attrs string) string {`。
textElement（pptxedit:641 / docxedit:762）、xmlEscape（656/779）、maxInt（677/815）、minInt（684/822）、locateSpan（438/539）在包内各自独立实现，注释都写「同 docxedit」。
影响：XML 转义或空白折叠规则的修正只落一处，另一份继续按旧口径输出，产生「Word 修对了 PowerPoint 没修」的隐性分叉。
修法：抽 internal/office/ooxml 共用包放这些纯函数，两个 edit 包引用。工作量 M。

**U68 [P1] 四种导出的章节循环四份复制** — `internal/export/export.go:118-134`
证据：`failed := 0` / `for _, ch := range m.listChapters() {` / `content, err := m.pm.ReadChapterBranch(ch.num, ch.branch)`。
ExportTXT(118-134)、ExportMarkdown(163-179)、ExportEPUB(253-273)、ExportDOCX(docx.go:47-68) 各自复制一遍「遍历章节+计数失败+拼 label+slog.Warn+写盘」，ReadWorldview 与 FailedChapters 赋值同样四份。
影响：新增格式或改章节命名/失败策略要改四处，漏一处即「EPUB 有分支章节、DOCX 没有」这类跨格式不一致。
修法：抽 `forEachChapter(fn func(label, content string) error) (failed int, err error)`，四个导出只写各自渲染回调。工作量 M。

**U49 [P1] 两套情绪标签映射表分居两个文件** — `internal/whisper/emotion_fusion.go:37-91`
证据：`func describeInnerFeeling(label string) string {` / `feelings := map[string]string{` / `"SWEET_ATTACHMENT": "想靠近、有强烈的关心冲动、藏不住笑意",`。
emotion.go:32 MapEmotionLabel 产出标签集（ANGRY_ATTACK/FEARFUL_OBEDIENT/TSUNDERE/HURT_GRIEVANCE/SWEET_ATTACHMENT/QUIET_FOND/SHY_HEARTBEAT/COLD_DETACHED/CALM_RATIONAL），emotion_fusion.go 又用同一标签集分别建了 describeInnerFeeling/getEmotionTendency/getEmotionMaxLength/getEmotionProhibitions 四张 map；另有 reactionOpeners(110-120) 与 imperfectionChance(179-189) 两张同域表。九元标签集在 emotion.go 是 if 链阈值，在 emotion_fusion.go 是 4 份字面量表。
影响：标签集扩容（如新增情绪）必然漏改其中若干表：漏改的 map 走默认分支静默返回「正常状态」「平稳、正常」或长度 60，情绪融合提示词悄悄退化而不报错；新增/改名一个标签要同时改 5 处。
修法：以 emotion.go 的标签常量定义一处 `labelMeta` 结构（Label/中文名/内在感受/行为倾向/长度上限/禁清单/json 词池），删掉 5 张散表，各函数改为查表；顺带把反应词池与不完美概率并进同一结构。工作量 M。

**U37 [P2] zhipu 错误体解析跨包刻意复制两份且不等价** — `internal/ai/image_glm.go:139-141`
证据：`// zhipuErrorBodyMessage 解析智谱错误体 {"error":{"code","message"}}（官方形态）。` / `// ai 包与 modelengine 各有一份：跨包复用会引入依赖环，形态极简两处各自锚定。` / `func zhipuErrorBodyMessage(body []byte) string {`。
ai/image_glm.go:141 与 modelengine/engine_models.go:264 各一份；两份并不等价：ai 版 Code 用 `interface{}` 双态兼容，modelengine 版 Code 是 string。
影响：同一家服务商的错误体能被解析，取决于调用发生在哪个包——若官方把 code 回成数字，modelengine 版 json.Unmarshal 会失败并返回空串，错误信息退化为「GLM Key 校验失败：HTTP 400」，用户看不到官方原因（令牌无效/额度不足）。复制带来的不是重复代码量，而是两份行为不同的实现。
修法：把这段解析下沉到叶子包（如 internal/util 或新建 internal/zhipu），两处改为调用同一实现并统一用 `interface{}` + fmt.Sprint 处理 code。工作量 S。

**U10 [P2] 小工具重复：truncateRunes / ExtractJSON / 计数器 / itoa** — `internal/bookimport/parse.go:429-447`
证据：`func truncateRunes(s string, max int) string {` / `func itoa(n int) string {`。
truncateRunes 出现在 bookimport/parse.go:429 与 characterstate/characterstate.go:386（后者注释自陈「本包局部，避免与 analysis 包重名冲突」），而 internal/util/util.go:99 已有 Truncate，全仓 truncateRunes 共 13 份；countNonSpaceRunes 在 novelstyle/fingerprint.go:128 与 novelreview/util.go:38 双实现，countRune 同样双份（fingerprint.go:413 / util.go:148）；ExtractJSON 在 util/util.go:82（返回 string）与 bookimport/reconstruct.go:218（返回 any）双实现；RetryJSON（util.go:19）与 bookimport.CallJSON（reconstruct.go:267）是同一件「JSON 有界重试 + 负反馈」的两个版本。
影响：同一段文本在不同面板被截到不同长度/是否加省略号；JSON 重试策略两套（修正提示文案不同），模型纠偏行为随入口而异。
修法：统一走 util.Truncate/util.ExtractJSON/util.RetryJSON；bookimport 保留的 any 版改为 util.ExtractJSON 的薄包装（或在 util 补 ExtractJSONValue）；删除本地 13 份 truncateRunes 中的重复实现。工作量 M。

**U41 [P2] 章节文件名正则三处复制，注释互指** — `internal/graph/consistency.go:86-87`（延续）
证据：`// chapterFileRe 匹配主线/分支章节文件名 NNN.md / NNNa.md（与 stats.Collect 一致）` / `var chapterFileRe = regexp.MustCompile(`^([0-9]{3})([a-z]?)\\.md$`)`。
同一正则在三处独立声明：graph/consistency.go:87（注释写「与 stats.Collect 一致」）、stats/dashboard.go:70（注释写「与 stats.Collect 一致」）、app/consistency_deep_handler.go:378（deepChapterRe）；memory.go 里还有一套 leadingNumber 解析同一命名。
影响：三处正则今天字面相同，靠注释维系；任何一处按需放宽（如允许两位章节号）都会让「章节枚举」在不同面板给出不同集合（一致性检查扫到 N 章、统计面板漏掉），且没有测试能发现跨包不一致。handoff 已声明「章号一律由文件名派生、不设冗余字段」，本条是实施残留，仍开放。
修法：在 internal/types 或 internal/project 暴露 `ChapterFileRe` + `ChapterFileNum(name)`，三处引用同一来源；删除 memory.leadingNumber 之类的重复派生。工作量 S。

**U60 [P2] 整包无共用 OOXML/excelize 安全工具层** — `internal/office/xlsxpreview/xlsxpreview.go:77-85`（延续）
证据：`func renderGuarded(path string) (result string, err error) {` / `defer func() {` / `if r := recover(); r != nil {` / `result = ""`。
excelize 有已知未修 panic（GO-2026-6452），于是同一段 recover→可读 error 防线被复写：xlsxpreview/xlsxpreview.go:77-85、xlsxpreview:116-122（NeedsRecalc 再抄一遍）、schedule/xlsx.go:431-436，三份文案各自不同；crosslink/hideWindow(174-178) 与 gaea/proc.HideWindow 也是同类工具的两份实现。CHANGELOG v4.370/v4.385 记录两轮 go 扫描发现该缺陷并各加一处防线。
影响：同一 excelize 缺陷的防线有三处漂移风险：某处将来删了 recover，恶意 xlsx 在该入口直接 panic（Wails dispatcher 兜住但表现为整次调用失败），排查成本高。
修法：在 office 内建 `openExcelizeSafe(path) (*excelize.File, error)` 统一 recover 与 Options，所有 excelize 入口（含 schedule/xlsx.go）经它打开；hideWindow 统一走 proc.HideWindow。工作量 S。

**U62 [P2] 角色库头像本地化 3 套实现** — `internal/characterlib/portrait.go:114`
证据（热点与 open_questions 交叉）：characterlib 剧照按 ID+ext 命名直接写盘（portrait.go:114），character.savePortraitToProject（character.go:329）按 charID+ext 命名并解析 data URL/远程 URL，两套互不知情。
影响：同一角色可能在两个目录各存一份头像（单元已列为待盘上取样确认）；「本地化是否成功」出现两套判断标准，任一处的失败都不会让另一处感知。
修法：头像本地化收口为 characterlib 单点（含 data URL/远程 URL 解析），character 侧只调用；删除另一份命名规则并在注释登记权威来源。工作量 M。

**U73 [P2] 头像本地化第三份：字符画/剧照命名与项目侧写入口** — `internal/character/character.go:329`
证据：`savePortraitToProject` 按 charID+ext 命名并解析 data URL/远程 URL，与 `characterlib/portrait.go:114`（按 ID+ext）构成第三套命名口径；两者都直接 os.WriteFile（见 U2 非原子写清单）。
影响/修法同 U62；本条强调它同时是 U2 的非原子写落点之一，收口时须一并走 fileutil.AtomicWrite。工作量 S。

**U76 [P3] OCR seam 之外的第六处注入点分散** — `internal/docmd/ocr.go:98-108`
证据：`// ovisStartWait 是 startOvisServer 等待 llama-server 就绪的时长（默认 60s）；` / `// 包级变量便于单测缩短等待。` / `var ovisStartWait = 60 * time.Second`。
ovisStartWait(100)、ovisHealthy(104)、ovisBuildCmd(108)、ovisOCRPage(153)、tesseractLookPath/tesseractImage(226-228) 六个包级可变函数值，配合 ocrProviderRegistry 的 init panic，构成只为测试存在的接线方式；这些替身与 C10 的三套 OCR 客户端并存，使「OCR 走哪条路」无法从接口层判断。
影响：生产路径被测试替身绑死，改探测逻辑要同时改夹具（ocr_seam_test.go 132 行）；任何 goroutine 与测试并发写这些 var 即数据竞争。
修法：改为显式 config 结构体（Wait/Healthy/Build/OCR 字段）由 New 注入，测试构造自己的 config；与 C10 的 provider 收口合并做。工作量 M。

### consistency（多套并行口径，19 条）

**U20 [P1] cd 工期五闸校验 3 处复制已漂移** — `internal/schedule/ops.go:129-144`（延续）
证据：`switch t.DurationUnit {` / `case "", UnitWd, UnitCd:` / `return "", fmt.Errorf("任务 %s 工期单位非法（wd|cd）：%s", t.ID, t.DurationUnit)`。
同一条 durationUnit 规则在 ops.applyOne/upsert_task(129-144)、ops.applyOne/patch_task(177-205)、project.Validate(104-119) 各写一遍（碎片合计 4 段），错误文案逐字复制。
影响：新增或修改工期单位约束时极易只改 Validate，ops 通道仍接受 → 同一份计划经对话写入与经文件保存得到不同合法性结论，fail-closed 承诺失效。历史审计的双工期口径四刀已落，校验重复未收口。
修法：抽 `validateDurationUnit(t Task) error` 一处实现，Validate 与 ops 两条分支都调用；ops 只负责包上操作序号。工作量 S。

**U3 [P1] 章节编号解析四套口径，导出/统计/项目各写一遍** — `internal/export/export.go:70-81`
证据：`re := regexp.MustCompile(`^(\\d{3})([a-z]?)\\.md$`)` / `if _, err := fmt.Sscanf(match[1], "%d", &num); err != nil {`。
同一件事「从文件名取章节号」有 4 套：export.listChapters 用正则 `^(\d{3})([a-z]?)\.md$`（export.go:69，stats.go:33 逐字重复）、project.mainChapterSummaryNum 只认纯数字并排除分支（project.go:452-467）、project.leadDigits 前导数字容忍任意形状（project.go:550-564）、novelcontext.chapterNumFromFile 又一个前导数字实现（novelcontext.go:874-885）；迁移路径 project.go:783 又用 `fmt.Sscanf(name, "%03d.md", &chapterNum)`（第 5 套）。
影响：口径差即数据差——001a.md 在 project 侧不算主线、在 export 侧算一章；Sscanf 版对 006a.md 解析失败会进 skipped 并中止整次 v4 迁移（project.go:824-827）。新增章节命名形态必须人肉找齐 5 处。
修法：在 project 暴露唯一 `ParseChapterFileName(name) (num int, branch string, ok bool)`，export/stats/novelcontext/bookimport 全部改调；Sscanf 迁移路径同步替换。工作量 M。

**U5 [P1] v4 迁移用 Sscanf 硬格式，分支章节让整次迁移中止** — `internal/project/project.go:783`
证据：`fmt.Sscanf(name, "%03d.md", &chapterNum)`；解析失败进 skipped（824-827），`006a.md` 这类分支章节即触发。
影响：与 U3 是同一根因的另一面——迁移路径不认分支形态，用户带分支章节的项目迁移会中断在中间态，且中止点对用户不可见。
修法：并入 U3 的唯一解析函数；迁移对 skipped 文件改为收集告警并继续（不静默中止），迁移结束输出 skipped 清单。工作量 S（并入 U3 后）。

**U6 [P1] gate 与 review 判据重复且门槛相反** — `internal/novelgate/gate.go:26-32`
证据：`shortSentenceRunes = 5    // 「短句」上界` / `shortRatioLimit    = 0.40 // 短句占比告警线（且句数 ≥ minSentences）` / `minSentences       = 20`。
gate.ChapterQualityIssues（gate.go:66-128）与 novelreview.Review（review.go:98-159）+ dims 是两套独立「确定性体检」：gate 按句子判定 telegraph（>40% 短句且平均句长<10，S2），review.dimParagraphPace 按段落判定（review.go:439 `len(paras)>=12 && avg<=12`）+ 段落匀称度；review 有 15 个维度与 S1/S2/S3>=3 的发布结论，gate 只有 S1-S4 的 Issue 列表。同一条正文可能 gate 报 telegraph_style、review 给 APPROVE。
影响：作者与面板拿到互相矛盾的结论（gate 阻断而 review 放行，或反之）；调门槛要改两处且默认值写在 Go 常量里，而 review 的门槛本应是数据资产 rubric.json——同一层判据两种可配置性。
修法：把 gate 的句级/段级判据登记为 rubric.json 的维度（如 telegraph_sentence），gate 改为「只跑 rubric 的硬判据子集」并输出同一 Dimension 结构；重复的 splitSentences/splitParagraphs 抽到共用工具。工作量 L。

**U7 [P1] 两套文风互不相识：Profile 与 Fingerprint** — `internal/style/profile.go:134-160`
证据：`return os.WriteFile(filepath.Join(dir, "style-profile.json"), data, 0644)` / `// 兼容旧品牌：优先 .gaea/，旧项目回退 .wubigork/ (LoadProfile)`。
style.Profile 存在 `<project>/.gaea/style-profile.json`，由 novelcontext.buildStyle 消费注入 prompt（novelcontext.go:299-309）；novelstyle.Fingerprint 存在 fingerprint.json / style_digest.json（project.go:238-277 提供读写），是另一套统计口径。style 包不依赖 project.Manager，直接 `filepath.Join(projectDir, ".gaea")` 自建路径并自带 .wubigork 兼容分支（第 3 份品牌兼容代码）。
影响：「本书文风」有两个真相源：作者改文风档案不会影响指纹评分与去味，反之亦然；路径兼容逻辑与 project.IsV4 的兼容判断（project.go:738-745）各写一份，新增品牌或迁移目录时漏一处即静默读空。
修法：把 style-profile.json 的路径与读写收进 project.Manager（StyleProfilePath/Read/Write 三方法），去掉 style 包内的品牌分支；在文档里明确「Profile=生成注入口径、Fingerprint=评分口径」的边界并让两者互相引用。工作量 M。

**U13 [P1] AI 味判据三套严重度：blocker/S1/自定义** — `internal/novelstyle/score.go:48-59`
证据：`func severityToWeight(sev string) int {` / `switch sev {` / `case "blocker":`。
严重度有三套语汇：novelstyle 用 low/medium/high/blocker 并映射为权重分（score.go:41-59）；novelgate 用 S1-S4（gate.go:19）；novelreview 用 S1-S4 + pass/warn/fail/skip（review.go:57-69）。三者都被 app 层同时消费（novel_book_health.go:50/55、converge_handler.go:48-52、novel_gate_handler.go:51/106）后拼在一个响应里。
影响：前端要按三种枚举分别着色与排序；「阻塞」在三个体系里语义不同（blocker 只影响 0-100 分、S1 直接 REJECT、gate 的 S1 只出现在空正文），阈值调整必须同时改三处映射。
修法：以 novelreview 的 S1-S4 为唯一严重度枚举，novelstyle 的权重表改为 `severityToWeight(S1..S4)`，gate 保持 S1-S4；前端归一。工作量 M。

**U42 [P1] 图片 backendType 双套口径：注册表常量 vs 字面量** — `internal/ai/image_backend.go:16-22`
证据：`const (` / `// ImageBackendKindOpenAI OpenAI 兼容图片后端（/v1/images/generations）：` / `// 覆盖 xAI / Herdsman / Ollama 等兼容服务。` / `ImageBackendKindOpenAI = "openai"` / `ImageBackendKindComfyUI = "comfyui"`。
ai 包注册表 kind 是 "openai"/"comfyui"/"glm"（image_openai.go:45、image_comfyui.go:57、image_glm.go:50）；但 SetImageBackend 第二个参数是自由字符串，app 层传引擎名 "herdsman"/"ollama"（image_handler.go:863/873、app.go:617/624），另有一处传 "openai"（image_handler_test.go:438/539/609）。同一语义字段至少 4 种取值。
影响：GetImageBackendType 的返回值被当作「当前后端」下发给前端与配置（image_handler.go:734-739、client.go:1283-1292），而 app 内部判断用的是另一种口径（image_handler.go:277/331/337 比 "comfyui"/"herdsman"/"glm"）——两套口径靠人肉对齐，已出现同一个后端在不同面板/配置里叫不同名字（client 内存值 vs a.cfg.ImageBackend）。
修法：把第二个参数改成 `ai.ImageBackendKind*` 常量（herdsman/ollama 都归 ImageBackendKindOpenAI + 一个显式 engineID 字段），或直接删掉这个字符串参数，让后端类型由 backend 实例的类型断言/Name() 提供。工作量 M。

**U40 [P1] ConsistencyIssue/Report 同名两套定义、JSON 字段不兼容** — `internal/graph/consistency.go:19-35`
证据：`type ConsistencyIssue struct {` / `Severity    string `json:"severity"` // error / warning / info` / `Category    string `json:"category"` // attribute / timeline / status / relationship` / `EntityName  string `json:"entity_name"``。
internal/types/types.go:50-61 已经定义了 ConsistencyIssue（severity/section/description/suggestion）与 ConsistencyReport（issues/overall_note）；graph 包又定义了一套同名的（加 category/entity_name/location/evidence/branch，report 用 total_issues/summary）。两套名字相同、用途相近、字段与 tag 完全不兼容。
影响：前端拿到「一致性报告」必须知道它来自哪条链路才能解析字段（section vs category、overall_note vs summary）；消费方按 types.ConsistencyReport 反序列化 graph 的输出会静默丢字段；这类同名异形类型使 grep 式重构（全局改名/统计引用）直接失效。
修法：types 侧保留基础形状，graph 的扩展字段改为嵌入 types.ConsistencyIssue 并显式声明新增字段；或把 graph 版本改名 GraphConsistencyIssue/Report，避免重名。两者取一并在注释里写明权威来源。工作量 S。

**U63 [P1] xlsx 导入用 time.Local，全模块用 UTC** — `internal/schedule/xlsx.go:414-421`
证据：`func tryParse(s string) (time.Time, bool) {` / `for _, layout := range xlsxDateLayouts {` / `if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {`。
calendar.go:49-65 的 parseISO 明确「解析为 UTC 日期」，xlsx 导入却 `ParseInLocation(..., time.Local)`，两套时区口径在同一 `Project.StartDate` 上汇合。
影响：东八区外或夏令时机器上，Excel 里写的开工日可能被解析成前一天（`Format("2006-01-02")` 取到昨日），整份导入计划的 ES/EF 全体错一天，且与手工录入的计划对不上。
修法：xlsx 日期解析后按 parseISO 同口径做 UTC 归一，或日期算术全程不引入 time.Local。工作量 S。

**U56 [P1] 页数三处口径：字面计数/页规格/渲染产物** — `internal/docmd/pdf.go:92-103`
证据：`func countPDFPages(content string) int {` / `n := 0` / `for pos := 0; ; {`。
countPDFPages 在原始字节里字面数 `/Type /Page`；pagespec.go:129-137 pageBounds 按页规格算渲染边界；ocr.go:559-565 又用 `first+i` 推绝对页码。三套都自称「与文本路径一致」，但页数来源互不相同。
影响：PDF 内有非法/重复 `/Type /Page`，或 pdftoppm 跳页时，文本路径与 OCR 路径页码错位（ocr.go:522-523 注释正是为修这个错位而写），按页范围提取会拿到别页内容。
修法：totalPages 判定收进单一函数，OCR 路径以 pdftoppm 实际产物为唯一页码来源，文本路径按同一来源对齐；至少加一致性断言与诚实告警。工作量 M。

**U54 [P1] 三种 OCR 口径各自独立客户端** — `internal/ocr/herdsman.go:23-28`（延续）
证据：`// DefaultOCRModel 是 Herdsman /v1/ocr 的默认模型。` / `DefaultOCRModel = "paddleocr-ppocrv5-server"` / `// DefaultParseModel 是 Herdsman /v1/documents/parse 的默认模型。` / `DefaultParseModel = "minerU"`。
internal/ocr（Herdsman 远端 PaddleOCR/MinerU）与 internal/docmd/ocr.go（本地 OvisOCR2 llama-server + tesseract）是两套并行 OCR 体系，各有自己的模型选择、健康探测、超时与错误文案，再由 app/gaea_ocr.go 第三层串起来。docmd/ocr.go 注释自称 seam 三元组范式，herdsman 路径未接入。
影响：「OCR 引擎」在配置层（activeOCREngine / GAEA_OCR_ENGINE / GAEA_OCR_URL）、模型层、启动层三处各有定义，同一句「设置 OCR 引擎」在不同面板含义不同。
修法：以 docmd 的 OCRProvider seam 为唯一接口，把 herdsman /v1/ocr 也注册成一个 provider kind，删掉 app 中的四路串联。工作量 L。

**U58 [P1] 日历语义 Go 四处 + 前端 TS 各一份镜像** — `internal/schedule/calendar.go:73-91`（延续）
证据：`func IsWorkingDate(d time.Time, cal Calendar) bool {` / `wd := int(d.Weekday()) // time.Weekday 与 JS getDay 同口径：周日=0` / `ok := false`。
calendar.go 自述「前端 calendar.ts 的忠实移植」；同包 mpp.go:507-535 另写一份 mppWeekFromHours 求工作日集合，cost.go:85 与 project.Analyze 再各按自己方式消费 Calendar——工作日判定语义共 4 处。
影响：日历规则（工作制/节假日例外）任何修正必须同时改 Go 与前端 TS，且要记住 mpp 那份独立实现；漏改即出现「甘特显示的工作日与 CPM 算的不同」。历史审计已记 TS↔Go 镜像风险。
修法：工作日判定收敛为 IsWorkingDate 唯一入口，mppWeekFromHours 只产出 Workweek 集合；TS↔Go 加双端对拍 golden（双工期已有 golden 先例可扩）。工作量 M。

**U69 [P1] Markdown 结构识别共 3 份局部实现** — `internal/export/html.go:172-223`
证据：`func markdownToHTML(md string) string {` / `lines := strings.Split(md, "\n")`。
markdownToHTML(html.go:172) 判 "### "/"## "/"# "；markdownHeading(docx.go:93-103) 用 switch 判同样三个前缀；docmd/office.go:160-186 第三种按 word 样式名映射 #/##/###。三份识别规则各自维护。
影响：同一章正文在 HTML 导出、DOCX 导出、预览解析里可能落到不同标题层级，用户看到跨格式标题层级不一致。
修法：export 内抽 `markdownBlockKind(line)` 一处判定供 html/docx 共用；docmd 的样式映射单独成表并加对照测试。工作量 S。

**U39 [P2] core.IsTask 与 internal/intent 两套意图判定** — `internal/core/routing.go:6-24`
证据：`var taskTriggers = []string{` / `"帮我", "整理", "搜索文件", "查找", "打开", "关闭",` / `func IsTask(text string) bool {`。
internal/intent（311 行 + llm.go 兜底）是「指令中枢」的规则引擎（导航/生图/状态/提醒/读屏/发文件/存任务，带强/弱动词与别名表）；core/routing.go 又用 23 个中文子串做「是不是任务」的判定，经 office_handler.go:15 暴露为 OfficeIsTask。两者都对同一句用户输入做第一跳分类，判据与产出粒度完全不同。
影响：同一句话可能被两套规则给出相反结论（如「看一下这段」命中 intent 的弱导航、也命中 core 的「看一下」触发词），调用方必须自己记住该信哪一套；新增一条触发语要改两处，且 intent 有测试与 llm 兜底、core 只有子串匹配，质量不对等。
修法：删除 core.IsTask 与 OfficeIsTask 绑定，或把它改成 intent 的一个薄包装（`intent.Parse != nil`），保证「是不是任务」只有一个判据来源。工作量 S。

**U52 [P2] 中英关键词表成对却多处只查中文** — `internal/whisper/interpreter.go:175-194`
证据：`func hasNegationForPraise(msg string) bool {` / `m := strings.ToLower(msg)` / `for _, w := range praiseWordsZH {`。
L0 解释器为 12 组关键词各维护中英两份表（redlineKeywords/praiseWords/teaseMarkers/hurtfulWords/coldWords/apologyWords/vulnerableWords/vulnerableToPraiseOverride/dndExplicit 各 ZH+EN），但 hasNegationForPraise 只遍历 praiseWordsZH；DetectSoftConcern（473-485）只有中文 softConcernWords；IsDNDMessage 的时长解析只用 `itoa(i)+"小时/分钟"`（430-449）。即「英文消息被识别为 praise 时永不检查否定前缀」，soft concern 对英文完全失明。
影响：同一语义中英行为不一致：英文「you are not amazing」会命中 praise 而非 hurtful/cold（对应中文「你不厉害」因 hasNegationForPraise 返回 true 被拦）；英文疲惫倾诉不触发心理健康软关注。两套表的存在让维护者以为对等，实际是两套口径。
修法：hasNegationForPraise 同时遍历 praiseWordsZH+praiseWordsEN（并用英文否定词 not/no/dont/never 判前后缀）；softConcernWords 补英文子集或统一走一个 (ZH,EN) 词表结构，让新增词必须成对填。工作量 M。

**U51 [P3] 全局偏好反应词状态跨会话共享且 maxSize 死值** — `internal/whisper/emotion_fusion.go:123-172`
证据：`type openerState struct {` / `mu      sync.Mutex` / `recent  []string`。
globalOpenerState 是包级单例（129 行），被 buildReactionOpenerInstruction 在每轮融合块构建时读改写（138-172），prompt 里会写「最近用过：…——本轮必须换一个不同的」；它不按 session 分区，而同一进程内 GUI/微信/语音多会话共用一个 Orchestrator 池；同时 rand.Shuffle 使用全局 math/rand 源且代码中无 Seed 调用；maxSize 字段仅初始化未被任何逻辑读取。
影响：A 会话的开头词会挤掉 B 会话的「未用过」池，导致提示词里「必须换一个」的约束在跨会话时自相矛盾（本轮被要求避开的词其实是别的会话刚用过）；全局可变态使该提示词无法被会话级复现。
修法：把 openerState 作为 Orchestrator 字段（键 sessionID），或改为按 SessionID 分桶的 map 并加清理；删除未被读取的 maxSize 字段。工作量 S。

**U75 [P2] 迁移链 15 级含空迁移与建了又删的表** — `internal/whisper/db/database.go:137-153`（延续）
证据：`var migrations = []string{` / `SchemaV1,` / `SchemaV2,`。
migrations 数组 V1→V15 顺序执行，其中 SchemaV3 是空串（schema_v3.go：`const SchemaV3 = \`\``，注释称「空迁移（仅版本号升级）」），runMigrations 用 `if sql == "" { }` 显式处理这个空洞（177-179）；更重的是 V9（schema_v9.go）建 weixin_account/sync/context/seen 四表、V13（schema_v13.go）立刻 DROP 同一批表并自述「全仓无任何读写代码」——首次安装要执行一次建表再执行一次删表。
影响：迁移链只增不减：每条历史失误（V3 的空壳、V9→V13 的建了又删）永久留在启动路径上；新读者必须读 15 个文件才能确认某表当前是否存在，且「表是否被删过」只能靠 V13 的注释推断。
修法：下一个版本把 V15 之后压缩为一条基线 schema（新装直接建终态），老装保留版本号跳变逻辑；至少删掉 V3 常量与 runMigrations 里的空串分支，并把 V9+V13 合并为纯注释记录在 CHANGELOG 而非迁移链。工作量 M。

**U77 [P2] 人格预设与模板双份字面量，「100% 对齐」不可验证** — `internal/whisper/personality.go:11-84`（延续）
证据：`var PersonalityPresets = []PersonalityPreset{` / `// ─── 平台核心 AI 助手 gaea（默认，首页语音 AI）───` / `{ID: "gaea", Label: "gaea", Gender: "female", Dims: PersonalityDims{T: 85, I: 55, S: 20, O: 80, R: 50},`。
PersonalityPresets 29 条（11-84）与 PersonalityTemplates（136-448）是同一批人格的两次字面量定义：预设的 Dims 在 Presets、示例句/禁止项在 Templates，同一 ID 出现两遍，字段不重叠；reactionOpeners/imperfectionChance/关键词表/情绪标签表同理。每个文件头都声明「100% 对齐 ackem xxx.ts」（111/123 个源文件含 ackem，共 228 处），仓库内无 codegen 脚本引用这些表。
影响：「100% 对齐」是无法验证的声明：上游改一句示例句或加一个人格维度，Go 侧不会报错也不会有人发现；人格 ID 集合在 Presets 与 Templates 间的关系（Templates 缺 gaea 等）只能靠人肉比对，新增人格必须记得同时改两处。whisper 死代码调查已记「预置资产口径之争」，本条量化了双份字面量的规模。
修法：把 Presets 与 Templates 合并为单一结构体（Dims + Templates 字段同处），或加一个测试断言「每个 Preset.ID 都能在 Templates 命中 / 每个 Template.ID 都有 Preset」并放进 CI；文件头把「100% 对齐」改为「蒸馏自 ackem <文件>@<版本>，差异以 Go 侧为准」。工作量 M。

**U38 [P2] dpapi: 前缀字面量三份，注释自称需同步** — `internal/auth/token.go:16-19`
证据：`// securePrefix 是 secure 包加密值的前缀（Windows DPAPI 密文 / 非 Windows 降级值均带此前缀）。` / `// 与 internal/gaea/secure 的 prefix 保持一致，用于识别旧版明文（迁移兼容）。` / `// 注：internal/app/encryptSecretIfLegacy 也直接使用该字面量，改动需同步。` / `const securePrefix = "dpapi:"`。
四处独立声明同一加密信封标记：internal/gaea/secure/secure.go:12 的 const prefix、auth/token.go:19 的 securePrefix、assistant/manager.go:19 的 wxTokenPrefix、app/app.go:791 的 `strings.HasPrefix("dpapi:")`，且注释明确写着「改动需同步」——把跨包不变量写成了人工备忘。
影响：secure 包换前缀（或增加 v2 前缀）时，任一处漏改的路径会把密文当明文处理：auth 会「成功读取」密文串当 token 用（decryptField 的 else 分支标记迁移并原样返回），assistant 会重复加密。没有编译期或测试期护栏。
修法：在 secure 包导出 Prefix 常量（并在 secure 里提供 `IsEncrypted(s string) bool`），auth/assistant/app 一律引用它，删除三份本地字面量。工作量 S。

### coupling（层级倒挂与跨层耦合，5 条）

**U30 [P1] 底座反向依赖 CLI 层：建客户端要读 gaea TOML 配置** — `internal/ai/client.go:114-127`（延续）
证据：`func (c *Client) currentProxySpec() netclient.ProxySpec {` / `if c.proxySpecOverride != nil {` / `return *c.proxySpecOverride` / `gcfg, err := gaeacfg.Load()`。
internal/ai（桌面 AI 主链路）import internal/gaea/config 这个从 CLI agent 蒸馏来的包；同层还有 auth 与 assistant import gaea/secure、core import gaea/fileutil+proc、httpbridge + modelengine 反向引用。层序被倒置成「底层包依赖上层包」，而按包名前缀的显式约定 gaea 层本应是最上层。
影响：① 每次 NewClient（app.go:609、image_handler.go:83/849/1027、writing_state.go:66、characterlib_gen_handler.go:633 等多处构造点）都同步做一次磁盘读 + TOML 解析，请求热路径上附带文件 IO；② 改任一 gaea/config 字段都可能影响桌面聊天代理行为，回归面跨两个域；③ internal/config 与 internal/gaea/config 都以 `UserConfigDir()/gaea` 为根（config_paths.go:75-79 vs config.go:530-532），.gaea_config.json 与 config.toml 在同一目录共存，出问题时无法从「配置」二字判断读的是哪一份。历史审计已提分层/职责边界问题，本条给出具体反向依赖链，仍开放。
修法：把 ProxySpec 的取值上提到 app 装配层，构造 Client 时注入（proxySpecOverride 已是现成注入口）；或把 gaea/config 的 NetworkProxySpec 提取到 netclient 自身的 ProxyConfig 类型，config 包只做映射，ai 不再 import gaea。工作量 M。

**U31 [P1] core 是 office 的逐行平迁副本，两处并存** — `internal/core/fs.go:1-28`
见 U45（duplication 视角）。证据：`// Package core — 桌面 agent 文件读写执行器（自 internal/office/executor.go 平迁，逐行等价）。` / `func Execute(action DesktopAgentAction, path, target, query, url, content string) ExecResult {` / `switch action {`。
internal/core 全部 5 个源文件（fs.go 258 / jobs.go 53 / routing.go 25 / types.go 59 / modes.go 52）都自述「平迁自 internal/office/xxx.go（逐行等价）」，而 internal/office 现在只剩 aliases.go 8 行转发，类型名不同。
影响：core.Execute 的 write_file 会让模型直接覆写任意路径（无二次确认、无路径白名单），delete/move 同理；这类高危动作被复制到第二个包后，任何一次加固都必须在两处同时做，漏一处就是绕过面。此外 core 与 office 的类型不同名，任何按包名判断的逻辑（如 whisper/desktop_confirm_bypass.go 的 AlwaysConfirmActions）会只覆盖其中一个。
修法：删除 office 转发层，全部调用方直连 core；或反向把 core 合并回 office 并删掉平迁副本。两者取一，禁止长期并存。工作量 S。

**U32 [P1] 角色 Agent 与全局角色库职责重叠（19 处读后写回）** — `internal/character/character.go:474-513`
证据（热点自述）：19 处 `ReadCharacters` 后直接写回，无锁无版本；4 处写失败被丢弃。
character（Agent 侧，799 行）与 characterlib（SQL 侧，639 行）对同一批角色各有一套读写路径（store.go:637 ProjectCharactersForNovel 走 SQL），字段子集与排序口径不同；任一侧写入都不会让另一侧缓存失效。
影响：新增角色字段要在两侧各改一遍；同一次会话里 Agent 侧看到的角色与 SQL 侧列表可能不一致（字段零值处理不同）。
修法：明确「characterlib 为角色数据唯一仓储」，character.Agent 只调用其 API，删除本地 JSON 读写（并入 U8 的 readModifyWrite 收口）。工作量 M。

**U11 [P2] 读接口里写盘：ReadWorldviewFile 是迁移器** — `internal/project/project.go:154-201`
证据：`func (m *Manager) ReadWorldviewFile() (*types.WorldviewFile, error) {` / `wf, err := loadJSON[types.WorldviewFile](filepath.Join(m.Dir, "worldview.json"))`。
名字是 Read，实际在第 183/197 行 `writeJSON` 覆盖 worldview.json（把旧 worldview.md/main section 迁移成六维结构），并在 181 行用字符串前缀判断「是否旧默认内容」。novelcontext.buildSetting(276) 与 character.loadWorldviewContext(645) 等只读调用方会顺带触发磁盘写，且都写在各自的读路径里。
影响：「读世界观」在只读场景（体检、导出、并发打开）会产生写副作用与潜在的并发覆盖；迁移形状（>=2 sections、单 main section 特判）藏在读函数的嵌套条件里，无测试可独立验证。
修法：拆出 `MigrateLegacyWorldview() error`（Create/Open 时显式调用一次）与纯读 `ReadWorldviewFile()`；读路径不再写盘，旧格式由迁移步骤一次性转换。工作量 M。

**U35 [P2] 类型重复造：PlanProblem 与 Issue 双定义，靠注释约束一致** — `internal/types/plan_v1.go:20-42`
证据：`// PlanProblem 计划问题单条（确定性判据；与 novelgate.Issue 同形但不跨包依赖，` / `// 避免 types ↔ novelgate 环依赖）。` / `type PlanProblem struct {`。
types.PlanProblem 与 novelgate.Issue 逐字段同形（Code/Severity/Message/Evidence）；PlanMissingCode 与 types.PlanProblemMissingPlan 同为 "plan_missing" 且注释要求「两处必须一致」；outline_* 四个码在 gate.go 里是字符串字面量、在 app 层 planFieldLabels（create_chapter_context.go:267-279）又抄一遍；两类型之间在 create_chapter_context.go:236-238 手工逐字段搬运。
影响：改严重度枚举或加字段（如加 RuleID）要改两处类型 + 搬运点 + 码表；两处字符串常量靠注释约束一致性，无编译期保障。
修法：把 Issue 定义下沉到 types（或让 novelgate 直接返回 types.PlanProblem），删掉重复常量与手工搬运；outline_* 码提为常量并在 planFieldLabels 复用。下沉前先核实 types 是否已 import novelgate（见单元 open_questions），避免真形成环依赖。工作量 M。

### concurrency（并发与锁，2 条）

**U12 [P1] project 侧读-改-写无线程安全，锁只在 app 层** — `internal/project/project.go:901-919`（延续）
证据：`f, err := m.ReadAnalysisV2File()` / `if err != nil {` / `return err`。
UpsertAnalysisV2(901-919) 与 syncRewriteIndex(1014-1045) 都是「读 JSON → 改 → writeJSON」整表覆盖，Manager 自身无任何锁；app 层 writingState.mu 只在 getPM/setPM/closePM 三点使用（writing_state.go:20-43，注释写「保护 pm 的并发读写」），指向对象的文件级 RMW 完全不受保护。唯一被显式加锁的同类模式是 plans.json（app.go:96-99 为此专门加了 chapterPlanMu 并注释说明后写覆盖先写丢更新）。
影响：章节分析落盘与重写版本落盘只要跨客户端（httpbridge 本机面）或跨协程并发，就可能整段丢失对方的条目；受损的是用户数据（分析结论、重写版本索引），且无备份。v4.425 已为 plans.json 单点加锁（app.go:96-99 注释自陈同型风险），未推广到 analysis-v2/rewrites。
修法：照 plans.json 的做法把互斥下沉到 project.Manager（每文件一把 sync.Mutex 或统一 fileMu），RMW 全在锁内；app 侧 chapterPlanMu 随之删除。工作量 M。

**U28 [P1] SSRF 守卫校验全部解析 IP 却只拨 ips[0]** — `internal/netclient/ssrf.go:68-73`
证据：`for _, ip := range ips {` / `if BlockedInternalIP(ip.IP) {` / `return nil, fmt.Errorf("refusing to fetch internal address %s (resolves to %s)", host, ip.IP)`；末行 `return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))`。
守卫遍历所有 A/AAAA 记录，但真正拨号用的是 ips[0]；而且这里是「先 LookupIPAddr、再按 IP 字面量拨号」的两段式，绕开了 net.Dialer 自身的重解析保护——检查集合与使用集合不是同一个（TOCTOU 形状）。
影响：GuardedClient 被抓取任意外部 URL 的通道（booksource、webfetch 内网全禁变体）使用；攻击者可控制 DNS 应答顺序（公网 IP 在前、169.254.169.254/127.0.0.1 在后），或让 ips[0] 在检查后变化，守卫判定为「已检查全部」却把连接打到内网，云元数据/本机服务可达。
修法：只允许拨号到已校验通过的那个 IP：遍历时对第一个非 Blocked 的 IP 直接建连（或用 `net.Dialer.Control` 在 connect 前复核实际目标 IP）；禁止在未校验的 ips[0] 上拨号。工作量 S。

### error-swallow（吞错与静默失败，6 条）

**U17 [P0] ExtractJSON 取最后一个右花括号，多对象即解析失败** — `internal/util/util.go:82-96`
证据：`func ExtractJSON(s string) string {` / `start, end := -1, -1` / `for i, ch := range s {` / `if ch == '{' && start == -1 { start = i }` / `if ch == '}' { end = i }`。
start 只记第一个 `{`，end 每遇 `}` 就无条件覆盖，跨多个 JSON 对象时返回的是「第一个对象开头 + 最后一个对象结尾」的非法串，没有任何配对/嵌套/字符串逃逸处理。
影响：25 处生产调用点（copilot.go:264、chapter.go:57/171/200、outline.go:173/344/670、whisper/memory_ingest.go:118/291、app/novel_plan_handler.go:520、scene_cards_handler.go:553 等）全部把返回值直接 json.Unmarshal。模型回复里跟一句 `{"ok":true}` 或少一个 `}`，就会解析失败并进入重试/报错；RetryJSON 自己也拿它做「是否找到 JSON」的判据（jsonStr==reply 分支），错误结论会被静默当成「没找到 JSON」反复重试。
修法：改成括号配对扫描：只在 depth 由 0→1 记 start、由 1→0 记 end 并立即返回该子串，扫描时跳过引号内字符与反斜杠转义；补一个「两对象拼接」「尾部多 }」「字符串内含 }」的单元测试。工作量 S。

**U22 [P1] SSE 解析 panic 只写日志，调用方收到空成功** — `internal/ai/client.go:686-693`
证据：`go func() {` / `defer func() {` / `if r := recover(); r != nil {` / `slog.Error("SSE 流解析 panic", "engine", reqEngine, "model", reqModel, "recover", r)`。
解析协程吞掉 panic 后不向 chunks 投递任何错误帧；parseStreamEvents 自己的 defer 只做 close(chunks) 与 releaseSem。消费方（ChatSimpleStreamDetailed 1109-1127、ChatStreamChunks 的调用方）看到的是「通道正常关闭、无 Done、无 Error」。
影响：panic 场景（畸形 SSE 触发 slice/map 越界等）表现为「模型答了空字符串且没有报错」：上游把空回复当正常结果落库/回显，错误只存在于日志；叠加 U17 的解析缺陷，用户侧看到的是「AI 突然不说话」，没有任何可操作提示。
修法：recover 分支内 send(SSEChunk{Error: "SSE 解析异常: ..."})（send 对已关闭通道安全，需在 close 前用 select 保护），或把 panic 转成具名错误随通道下发；同步在 usage 记账里标记 success=false。工作量 S。

**U21 [P1] 角色画像写盘失败被丢弃，仍报成功** — `internal/character/character.go:118-131`
证据：`a.mergeCharacters(cf, updates)` / `_ = a.pm.WriteCharacters(cf)` / `}`。
applyUpdates(118-131) 与 GenerateCharacters(458/462) 共 3 处 `_ = a.pm.WriteCharacters(...)`：ChatWithAutoSave/ChatCharacterDetail 已从 AI 回复中抽取了角色更新、合并进内存，落盘失败却既不返回错误也不 emit 事件，调用方与用户看到的是「已更新」，磁盘上什么都没有。同文件 Save/Delete 系列都老实 return error。
影响：用户以为角色修改已保存，重启或切书即全部丢失；且失败原因（磁盘满/AV 占用/JSON 损坏）连日志都没有。
修法：3 处改为返回错误并由调用方决定降级（ChatWithAutoSave 返回 `(reply, error)` 已在签名里），或至少 slog.Warn + 事件；禁止对 pm 写方法使用 `_ =`。工作量 S。

**U53 [P1] OCR 四路降级链吞掉全部真实错误** — `internal/app/gaea_ocr.go:28-39`
证据：`if a.activeOCREngine != "" || a.activeOCRModel != "" {` / `if text, err := a.herdsmanOCRWith(a.activeOCREngine, a.activeOCRModel, imagePath); err == nil && strings.TrimSpace(text) != "" {`。
GaeaOCRText 依次试 activeOCR → herdsmanOCR → herdsmanParseImage → docmd.OCRImageText，四层只判 `err==nil` 从不保留 err，最终只返回最后一层的错误。
影响：用户看到「OvisOCR2 不可用」，真实原因可能是 herdsman 返回 HTTP 401 或 MinerU 超时；排查必须复现四路，用户与开发者都被最外层文案误导。
修法：逐层 errors.Join 收集错误，最终 error 附四层原因（现有 ocrUnavailableError 只保留安装提示）。工作量 S。

**U79 [P1] 安全审计写盘失败 5 处静默丢弃** — `internal/whisper/desktop_router.go:47-53`
证据：`if blockReason := checkActionSettings(action, ctx); blockReason != "" {` / `_ = AppendDesktopAgentAudit(ctx.DataRoot, DesktopAgentAuditEntry{` / `TS: now, Action: string(action),`。
blocked(53)、blocked-by-blacklist(64)、path-policy-blocked(75)、denied(92)、allowed(126) 五处调用全部 `_ = AppendDesktopAgentAudit(...)`，而 AppendDesktopAgentAudit 自身返回 error（desktop_audit_log.go:32）。这不是 lint 覆盖的「err 未处理」表面项：审计是这套桌面执行器唯一的追责面，日志写失败时调用方无从感知。
影响：dataRoot 不可写/被文件占位时，被拦截的敏感操作（含删除、安装）与放行记录一并消失，用户与开发者都看不到「刚才有人尝试删系统目录」；静默失败使审计文件看起来「干净」而实际缺失。
修法：至少对 blocked/denied 三处改为在失败时 slog.Error 并把「审计不可用」并入返回 Summary（fail-loud）；或让 AppendDesktopAgentAudit 失败时降级写 slog 保证有一条痕迹。工作量 S。

**U66 [P2] 办公解析 9 处 err 静默忽略** — `internal/docmd/office.go:33-34`
证据：`rc, _ := f.Open()` / `docXML, _ = io.ReadAll(rc)`。
忽略散布在 33、34、360、362、402、403、426、438、441；xlsxToMarkdown 明确 `if xml.Unmarshal(sheetXML, &sd) != nil { continue }`，该 sheet 静默消失。
影响：zip 项损坏或 worksheet XML 非法时输出「少几张表/几段」的 Markdown 且无告警，format_convert 回执不含丢失信息，用户以为转换完整。
修法：忽略处收集为 warnings 随返回值上报，或至少 slog.Warn + Markdown 末尾附「已跳过 N 个无法解析部件」。工作量 S。

### safety（安全，5 条）

**U72 [P1] 22 种桌面操作巨型 switch，零权限与路径校验** — `internal/whisper/desktop_executor.go:40-134`
证据：`func ExecuteDesktopAgentAction(action DesktopAgentAction, path, pathTo, target, query, url, content string, ctx DesktopExecContext) ExecuteResult {` / `cwd := ctx.CWD` / `if cwd == "" {`。
读/写/移动/删除/列目录/grep/下载/打开应用/关闭进程/下载并安装全部由 action 在这一个 switch 里分派，函数自身不做任何权限检查、路径白名单或身份判断（4 层沙箱全在 desktop_router.go 的调用方）；签名 7 个 string 位置参数无法静态区分语义，测试里已有 `ExecuteDesktopAgentAction(ActionMkdir, sub, "", "", "", "", "", ...)` 全空串调用。
影响：安全边界只存在于调用方：任何绕过 ExecuteUseComputer 直接调用本函数的新代码路径（如 agent_loop_runner.go:254 executeToolBatch）即获得无沙箱的完整 FS/进程/安装能力；参数错位不会被编译器发现（都是 string）。
修法：拆为按能力分文件的小函数注册表 `map[DesktopAgentAction]func(args UseComputerArgs, ctx DesktopExecContext) ExecuteResult`，args 用结构体替代 7 个 string；把 checkActionSettings/evaluatePathPolicy 下沉到本层做纵深防御。工作量 M。

**U74 [P1] downloadAndInstall 在沙箱外直接执行下载物** — `internal/whisper/desktop_executor.go:106-120`
证据：`dl := downloadHTTPS(url, dest)` / `if !dl.OK {` / `return dl`。
该分支下载 url 后立即 `shellOpen(dest)`，而 shellOpen 走 `rundll32 url.dll,FileProtocolHandler`（同文件 304-315）直接执行该文件。确认层 registry AlwaysConfirmActions[ActionDownloadAndInstall]=true（desktop_confirm_bypass.go:9-15）只作用于 ExecuteUseComputer 的第 4 层，本函数的执行路径完全不经过确认回调；downloadHTTPS 只校验 `https://` 前缀（361 行），url 可由 LLM 工具参数直接提供。
影响：一次工具调用即可下载并静默执行远端可执行文件：确认清单里「下载并安装必须单独确认」的意图被实现层绕过，且审计只记录一次 allowed（desktop_router.go:126），没有「已执行下载物」的独立留痕。
修法：删除分支内的自动 `shellOpen(dest)`，改为返回需确认状态交由 RequestConfirm；或强制该 action 在 executor 入口即拒绝，只允许 ExecuteUseComputer 携带 AlwaysConfirm 结果后调用。工作量 S。

**U80 [P1] normalizePath 注释称禁 .. 逃逸，实现无校验** — `internal/whisper/desktop_router.go:270-276`
证据：`// 规范化 + 禁止 .. 逃逸` / `clean := filepath.Clean(p)` / `if !strings.HasPrefix(clean, filepath.VolumeName(clean)+string(os.PathSeparator)) &&`。
函数只做 filepath.Clean（把 .. 消解为绝对路径）后返回，没有任何「结果必须落在 cwd/允许根之内」的比较；`..\..\Windows\System32` 会被 Clean 成 `C:\Windows\System32` 并原样返回。hard block 只有 isHardBlockedWritePath 两条前缀（c:\windows\system32 / syswow64，294-305）且只对写类 action 生效，读类 action 无任何限制；isSensitivePath（279-292）是三条硬编码字面路径（含 c:\program files 一条，无法覆盖 %SystemRoot%/短名/大小写变体），且只产出一句提示文本。
影响：注释与实现不符使评审者误判已有防护：读路径可读任意位置（如 C:\Windows\System32\config 下的大量敏感文件），写路径仅靠 8.3 短名/环境变量/大小写变体即可绕过两条前缀判断。
修法：在 normalizePath 里返回 `(abs, ok)`，ok 仅在 abs 位于配置允许根（或 cwd）之内时为真；evaluatePathPolicy 对读类 action 同样套用该 ok；sensitive/blocked 判断改为对 filepath.VolumeName + 逐段比较，并对 p 先做 os.ExpandEnv 后的 EvalSymlinks 归一。工作量 M。

**U81 [P2] PowerShell 命令拼接仅转义单引号** — `internal/whisper/desktop_executor.go:317-326`
证据：`cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",` / `fmt.Sprintf("Start-Process '%s'", strings.ReplaceAll(target, "'", "''")))` / `proc.HideWindow(cmd) // Windows: 防止弹出 cmd 黑框`。
target 来自工具参数（openAppTarget/closeAppTarget/trashItem 同一模式，330-333/345-348 亦同）：只把 `'` 替换为 `''`，未过滤换行、分号、反引号、`$( )`、`&` 等 PowerShell 元字符；target 含换行即可跳出单引号字符串成为新语句。
影响：路径/应用名含 `'; <命令>; '` 形态即可执行任意 PowerShell，且由于走 exec.Command 无 shell 解析，注入只在 PowerShell 层发生，调用方无任何察觉。
修法：改用参数化调用：把目标作为 PowerShell 参数（-File 传参或 `[Convert]::ToString` 编码后 -EncodedCommand），或只允许 os.StartProcess/ShellExecute 的路径直传；至少加 `\r\n;&|\`$()` 过滤并前置 os.Stat 校验。工作量 S。

**U57 [P2] 截屏丢弃 GDI 返回码，失败帧按成功返回** — `internal/screen/screen_windows.go:158-159`
证据：`_, _, _ = procSelectObject.Call(hdcMem, hbm)` / `_, _, _ = procBitBlt.Call(hdcMem, 0, 0, uintptr(w), uintptr(h), hdcScreen, uintptr(x), uintptr(y), srcCopy)`。
SelectObject/BitBlt 的返回值与 GetLastError 全部丢弃，随后无条件把 DIB 内存当有效像素读成 image.RGBA 并返回 nil error。
影响：BitBlt 失败（DC 丢失、锁定屏、虚拟桌面坐标越界）时返回未初始化或全黑截图且当作成功交给截图工具与视觉链路，下游得到「成功但空白」的图，属典型静默失败。
修法：r==0 时用 syscall.GetLastError() 返回 error，SelectObject 失败同样报错并释放资源。工作量 S。

### legacy-residue（兼容壳与历史残留，5 条）

**U46 [P1] core 会话模式内存态与落盘态两套定义** — `internal/core/sessionmode.go:6-27`
证据：`type SessionModeStore struct {` / `mu    sync.RWMutex` / `modes map[string]bool` / `func (s *SessionModeStore) GetMode(sessionID string) bool {`。
同一包内 modes.go 提供了落盘版会话模式（LoadModes/SaveModes/GetPersistedMode/SetPersistedMode/ClearAllPersistedModes，写到 `<DataRoot>/office/session-modes.json`），sessionmode.go 又提供纯内存版 SessionModeStore；而 app/office_handler.go:9 绑定给前端的 `sm = officecore.NewSessionModeStore()` 用的是内存版，落盘版从绑定面看无调用方。
影响：前端调 OfficeGetMode/OfficeSetMode 设置的「桌面助手模式」只活在内存，进程重启即丢；用户会看到「模式开关自己跳回去」。两套实现同时存在也让人无法判断哪个是权威（落盘版注释里全是 v4.441.1 走查、原子写等维护痕迹，内存版只有 29 行）。
修法：确认权威实现后删一套：要么 app 绑定改走 GetPersistedMode/SetPersistedMode（带 dataRoot），要么删除 modes.go 落盘版并把 DataRoot 依赖一并清理。工作量 S。

**U16 [P2] 品牌兼容壳散落：.wubigork/.gaea 分支各写各的** — `internal/project/project.go:738-745`
证据：`// 兼容旧品牌：新标记 .gaea/v4，旧项目 .wubigork/v4 同样识别` / `if _, err := os.Stat(filepath.Join(m.Dir, ".gaea", "v4")); err == nil {` / `return true`。
旧品牌兼容至少 3 处独立判断：project.IsV4（738-745）、style.LoadProfile（profile.go:148-152 读 .wubigork/style-profile.json）、project.Create 同时落 .gaea/v4 标记（86-92）并保留 v3 迁移分支 MigrateV3ToV4（747-839）。story_thread 字段名、extractCharacterUpdates 的 `---CHARACTER_UPDATE---` 标记协议（character.go:657-658）也是另一类只在单侧实现的历史协议。
影响：兼容面没有单一登记处：新增项目目录布局（如新的标记文件）要在 3 处各判一次；旧品牌分支永远不会被删除，形成只增不减的兼容壳。
修法：集中一个 `legacyPaths()`（或 brand.go）列出 .gaea/.wubigork 两套路径与标记，兼容判断只查这张表；为每个兼容项登记「何时可删」的条件。工作量 S。

**U36 [P2] 手写 itoa 取代 strconv，负数落空串** — `internal/novelgate/gate.go:151-166`
证据：`func itoa(n int) string {` / `if n == 0 {`。
novelgate 与 bookimport 各手写一份 itoa（gate.go:155、parse.go:441）替代 strconv.Itoa；gate 版对 n<0 直接返回空串（循环条件 n>0 一次都不进）。
影响：任何未来引入差值的消息（差值/增量）会静默丢数字变成「标点以句号为主（句号 30 / 逗号 ）」这类半截文案；同时两套非标准整数格式化与全仓 strconv 用法并存。
修法：两处删除，改 strconv.Itoa；若担心 fmt 开销，统一收进 util。工作量 S。

**U55 [P2] docmd 手写 XML 提取不解实体** — `internal/docmd/office.go:133-158`
证据：`texts = append(texts, remaining[contentStart:contentStart+tEnd])` / `remaining = remaining[contentStart+tEnd+6:]` / `text := strings.Join(texts, "")`。
docxToMarkdown 先把 body 反序列化成 InnerXML 字符串，再用 findWt/isSelfClosing/IndexByte 手工切 `<w:t>`，全程不解 `&amp;`/`&lt;`/`&#x…;`（extractCellText 249-274 同款）。同仓库 xlsxpreview 用 excelize、export/docx.go 用 gooxml，都是真解析器。
影响：含 `&` `<` 引号或中文实体的 Word 文档经 format_convert/预览/largefile 转 Markdown 时正文出现字面量 `&amp;`（8 个调用方受影响），而同一文件走 docx 预览面板正常。
修法：用 encoding/xml 的 Token 流重写 docxToMarkdown（或复用 docxedit 的 parseParagraphs 模型），至少对 w:t 文本做实体解码。工作量 M。

**U48 [P2] 序号→id 映射可被重复序号覆盖** — `internal/schedule/xlsx.go:551-554`
证据：`seqID := make(map[int]string, len(raws))` / `for i, r := range raws {` / `id := fmt.Sprintf("t%d", i+1)` / `seqID[r.seq] = id`。
r.seq 缺省用 `len(raws)+1`（496 行），但表里有「序号」列且重复/非递增时同一 key 被后写覆盖；前置引用回链(607) 与分组判定(612-617) 都按这个 map 查。
影响：表里出现重复序号（复制粘贴行常见）时前置引用会指向错误的最后一行，导入出错误的 FS/SS 关系且不报错；612-617 再线性查一遍 `q.seq==no` 只是重复劳动不纠错。
修法：检测到重复 seq 即报错指出行号，或统一用行内 index 作 id、序号列只作显示。工作量 S。

### complexity（复杂度，4 条）

**U26 [P1] 思考预算守护在两处逐字复制** — `internal/ai/client.go:1242-1267`
证据：`req.ChatTemplateKwargs = map[string]any{"enable_thinking": opts.EnableThinking}` / `// 思考与正文共享 max_tokens：显式小预算抬到 4096（同 herdsman 守护）。` / `if req.MaxTokens > 0 && req.MaxTokens < 4096 {` / `req.MaxTokens = 4096`。
modelhub 分支（1248-1254）与 herdsman/ollama 分支（1255-1266）各自写了一遍「enable_thinking + 若开启且 max_tokens<4096 则抬到 4096」，注释里也自认「同 herdsman 守护」。magic number 4096 在两处硬编码，且与 maxTokens 默认值 4096（1226-1228）同值不同源。
影响：调整思考预算下限（模型换代后 4096 不再够）时必须同时改两处，漏改则 modelhub 用户仍拿到「只有推理、无正文」的旧故障；三处 4096 字面量也让「默认预算」这一概念没有单一来源。
修法：抽出 `clampThinkingBudget(req, wantThinking)` 与常量 `minThinkingBudget=4096`，两个分支只调一次；默认 maxTokens 也引用同一常量。工作量 S。

**U44 [P3] 角色索引在单次编译内重建三次** — `internal/novelcontext/novelcontext.go:567-569`
证据：`func charIndex(pm *project.Manager) map[string]types.Character {` / `return loadCharIndex(pm)` / `}`。
charIndex(567) 只是 loadCharIndex(628) 的转发；CompileSceneBible 内 buildSceneChars 用 `charIndex(pm)`（103 行）、buildPOVMask→resolveCharName 再调 loadCharIndex（790 行），每次都要 ReadCharacters 并解析整份 characters.json；同一次编译至少 2-3 次全量读+建索引。
影响：角色表大（数百条）时每次场景编译多付 2 次文件读与两次 map 构建；三个入口（POV/角色块/兜底）对同一份数据的视图可能因读时机不同而不一致。
修法：CompileSceneBible 顶部读一次 characters.json，把索引作为参数传给 buildSceneChars/buildPOVMask；删除 charIndex 转发。工作量 S。

**U27 [P2] 健康巡检串行探测，单轮最坏阻塞 N×8s** — `internal/modelengine/health_probe.go:106-125`
证据：`func (m *Manager) probeRound(ctx context.Context) {` / `for i := range targets {` / `select {` / `case <-ctx.Done():` / `return` / `m.probeOne(ctx, targets[i])`。
每轮把 enabled 且非本地的引擎全部快照出来顺序探测，每个 probeOne 自带 8s 超时（healthProbeTimeout）；探测在锁外执行（好），但整轮是串行的。
影响：8 个云端引擎（xai/deepseek/glm/opencode-go/opencode-zen + 若干 custom）全部不可达时，一轮最长占 64 秒，期间只有 healthStop 生效；下一轮 ticker 已经在排队，巡检密度随引擎数增长而失效。probeOne 每次还会 saveState() 全量落盘（176 行），串行 N 次等于 N 次写盘。
修法：用 errgroup/带缓冲的信号量把 probeOne 并发化（上限 4），并把 saveState 从 probeOne 里提出去，一轮结束只落盘一次。工作量 S。

**U82 [P2] DnD 时长解析用词表 × 枚举 Contains 暴力匹配** — `internal/whisper/interpreter.go:430-449`
证据：`for _, pattern := range []string{"小时", "个钟", "个钟头"} {` / `for i := 1; i <= 24; i++ {` / `if strings.Contains(m, itoa(i)+pattern) {`。
解析「别烦我 N 小时/N 分钟」靠三层嵌套：3 个中文单位 × 24 或 120 次 itoa+Contains，且每轮重建字符串；只支持阿拉伯数字（「两个小时」落到 451 行的「今晚」近似 5 或 459 行的 1 小时），且 `hours = max(1, i/60)` 使 1~59 分钟一律折算为 1 小时（以小时为单位存储的精度损失）。
影响：中文数量词（一/两/半）与小数点全部落到兜底值：用户说「别烦我两小时」实际按 1 小时（一会/一下 分支）或 5 小时（今晚分支）处理，勿扰语义静默失真；同时每回合最多 192 次 Contains 属纯浪费（长消息已被 50 字上限挡住，影响有限）。
修法：改用正则一次性抽取数字与单位（如 `(\d+)\s*(小时|分钟|个钟)`），并补中文数字映射表；DnDResult 的 Hours 改存 time.Duration 或分钟数避免 i/60 折损。工作量 S。

### dead-code（死代码与僵尸资产，5 条）

**U78 [P1] 桌面能力有 3 条路由，2 条 0 生产调用** — `internal/whisper/dispatch_router.go:81-167`（延续）
证据：`// RouteDispatch 核心路由决策` / `func RouteDispatch(input RouteDispatchInput) *DispatchResult {` / `now := input.Now`。
全仓 grep `RouteDispatch(` 仅命中 dispatch_router_test.go 12 处与本文件定义；`ResolveDesktopCapabilityEnhanced(` 同样仅命中 desktop_test.go 8 处与 desktop_capability_routing.go:116 定义；`RunAgentLoop`/`DefaultAgentLoopRunner` 仅命中 agent_loop_runner_test.go。三条入口分别用不同判定：dispatch_router 是 LLM 精判+0.85/0.60 双阈值+人设乘数，desktop_capability_routing 是关键词评分（midConfidenceThreshold 0.4）+正则 fallback，desktop_router 直接由前端 RequestConfirm 驱动。
影响：读代码的人无法判断「桌面能力到底怎么路由」——三条路的语义与阈值互不兼容，其中两条只能被测试触发；任何「路由为什么没触发」的排查都要先证伪两条死路。这是「蒸馏取道不取器」留下的整条未接线决策链，不是散件函数。whisper 死代码调查 B/D 簇（余 196 项，建议不再追刀）——本条提供整文件级可删结论与调用点计数。
修法：确认产品口径后二选一：删除 dispatch_router.go（284 行）+ desktop_capability_routing.go（289 行）及其测试，或将其接线进 whisper_handler 并同时删除另一条；跑一次 `deadcode -test` 基线复扫。工作量 M。

**U50 [P1] 情绪涌现双状态源：包级全局平行会话状态** — `internal/whisper/emotional_emergence.go:11-15`（延续）
证据：`var (` / `recentEventTypes           []string` / `consecutiveMeaningfulCount int`。
orchestrator.go:59-61 为同一语义各存了一份实例字段（o.recentEventTypes/o.consecutiveMeaningfulCount/o.consecutiveVulnerableCount），PreLLMTurn 只写实例字段；而 ResetEmergenceTracking()（orchestrator.go:141 每会话首回合调用）只清这组包级全局，PushEventToHistory/PushMeaningfulTurn/PushVulnerableTurn（emotional_emergence.go:23/36/48）无任何生产调用点，Get* 三个 getter 同样 0 调用（全仓 grep 仅命中定义行）。
影响：两套状态永不互通：全局那份恒为「已 Reset 的零值」；任何后续开发者按函数名把 Push* 接回流程，就会立刻复现注释里声称已修的「包级全局串台」——而现在的 ResetEmergenceTracking 调用点恰好是唯一还连着这组全局的线，删不动也留不得。whisper 死代码调查「余 196 项」之一，本次给出跨会话串台风险证据。
修法：删除包级三变量与 8 个 0 调用函数（Push*/Get*/CountMeaningfulInRecent/ResetEmergenceTracking），会话级状态一律走 Orchestrator 字段；若需保留 API 形态，改为接收 *Orchestrator 或显式 state 参数。工作量 S。

**U83 [P1] agent 自治回路整条 0 生产调用** — `internal/whisper/agent_loop_runner.go:288`
证据（热点自述）：整条 agent 自治回路（LLM 循环 + 工具批量）0 生产调用点，仅测试可达；其 254 行 executeToolBatch 正是 U72 描述的无沙箱调用面。
影响：同 U78 属未接线决策链；但它是 U72 绕过沙箱的具体入口之一，删/留必须与 U72 的决定一起做，否则要么留着一个可达的无沙箱入口，要么删掉后无法复现桌面能力。
修法：与 U78 同一刀处理（确认 ExecuteUseComputer 的当前可达性后决定删或接线）；若保留，必须让它调用带沙箱的入口而不是 desktop_executor 裸函数。工作量 M。

**U84 [P2] 成人模式 FSM 状态字段仅被死函数读取** — `internal/whisper/orchestrator.go:557-570`（延续）
证据：`func (o *Orchestrator) computeIntensityMod(emotion EmotionState, l1 L1State) float64 {` / `aroFactor := clampF(0.8+mathAbs(emotion.Aro)/100*0.7, 0.8, 1.5)`。
computeIntensityMod 全仓无生产调用点（grep 仅 whisper_orchestrator_test.go:105/107/121/122），它是 o.adultStateStr 的唯一「读→影响输出」路径（566 行）；真实强度走 ComputeIntensityModifier（orchestrator.go:338）并按返回值写 PreLLMResult.IntensityMod，与 o.adultStateStr/o.adultBudget 无关。o.adultStateStr 在生产代码里被 runAdultModeFSM 写 9 次、读 5 次，但它仅用于自身状态机内部，系统 prompt 用的是函数返回值（343 行 BuildAdultModeSection(o.Preset.ID, AdultState(adultState), ...)）。
影响：字段与死函数互相「引用」使静态分析看不出孤立：删除 deadcode 只能找到死函数，字段仍有 14 处引用；两套强度口径（FSM 内的 budget/lock 计分 vs ComputeIntensityModifier 的时间/情绪/阶段调制）并存，调参时易改错一处。死代码调查已记「余 196 项=局部方法与散件」，本条给出死函数与活字段的互锁关系。
修法：删除 computeIntensityMod 与 PreLLMResult.IntensityMod 里未被消费的语义；把 adultStateStr/adultBudget 收敛为 runAdultModeFSM 的局部状态或明确标注为「FSM 内部记忆，不参与强度调制」，并在注释里指向 ComputeIntensityModifier。工作量 S。

**U86 [P3] DedupByEmbedding 是占位假实现** — `internal/whisper/memory_fact.go:347-352`
证据：`// DedupByEmbedding 占位方法，Phase 3 实现` / `func (fs *FactStore) DedupByEmbedding(embedding []float64) bool {` / `return false`。
导出方法、接收 embedding 参数、恒定返回 false，注释自述未实现；同域已有真正可用的 FactEmbeddingCache（fact_embedding_cache.go）与 vector_search.go，占位与真实现并存。
影响：调用方（若被接线）会永远得到「无需去重」，静默产生重复事实；同时它使「Embedding 去重」在代码搜索里看起来已实现，掩盖了真实缺口。
修法：删除该方法，或改为显式返回 error/不实现 panic 并在 CHANGELOG 记录 Phase 3 待办；若 Phase 3 已由 vector_search 承担，补注释指向替代实现。工作量 S。

### test-smell（测试结构债，2 条）

**U24 [P2] 测试文件比被测源码还长，单文件多职责** — `internal/modelengine/engine_test.go:1-40`
证据：`internal/modelengine/engine_test.go  1020 行` / `internal/modelengine/stats.go         758 行（同包最大非测试文件）` / `internal/modelengine/glm_catalog_test.go 590 行`。
本单元测试总量约 8.6k 行、51 个测试文件，与非测试代码量同一量级；其中 engine_test.go 1020 行覆盖引擎 CRUD、连接测试、URL 装配、模型列表刷新、预设顺序等 5+ 个域，glm_catalog_test.go 590、stats_test.go 502、catalog_models_test.go 474 同样各自承担多域。
影响：任何一次结构拆分（如按 U23 拆 stats.go）都要在千行测试里同时改，测试本身成为重构阻力；测试文件与被测文件不再同名对应，定位「哪条规则被锁」要靠搜索。与「先测后改」的重构路径直接冲突。
修法：按被测域把 engine_test.go 拆为 engine_crud_test.go / engine_connect_test.go / engine_models_test.go / engine_state_test.go；拆分时只移动测试函数不改断言。工作量 M。

**U85 [P3] 生产包内 6 个可变 var 作测试夹缝** — `internal/docmd/ocr.go:98-108`
证据：`// ovisStartWait 是 startOvisServer 等待 llama-server 就绪的时长（默认 60s）；` / `// 包级变量便于单测缩短等待。` / `var ovisStartWait = 60 * time.Second`。
ovisStartWait(100)、ovisHealthy(104)、ovisBuildCmd(108)、ovisOCRPage(153)、tesseractLookPath/tesseractImage(226-228) 六个包级可变函数值，配合 ocrProviderRegistry 的 init panic，构成一套只为测试存在的接线方式。
影响：任何 goroutine 与测试并发写这些 var 即数据竞争；生产路径被测试替身绑死，改探测逻辑要同时改夹具（ocr_seam_test.go 132 行）。
修法：改为显式 config 结构体（Wait/Healthy/Build/OCR 字段）由 New 注入，测试构造自己的 config。工作量 M。

### doc-drift（注释与实现漂移，2 条）

**U9 [P2] 注释重复与不确定序：同段注释两遍、map 遍历渲染** — `internal/style/profile.go:163-196`
证据：`// ToStyleGuide 生成可注入 prompt 的风格指导文本` / （空行）/ `// ToStyleGuide 生成可注入 prompt 的风格指导文本`。
同文件 163/165 两行逐字重复的 ToStyleGuide 注释；novelcontext.go:510-515 的 buildSceneChars 注释整段重复两遍（内容一致、字段说明更细的那版在后）；buildStyleGuide(173-196) 用 `map[string]string` 的 labels 遍历渲染 Traits，Go 的 map 迭代顺序随机。
影响：同一份风格档案每次生成的风格指导条目顺序不同，prompt 前缀不稳定（影响模型缓存命中和 diff 可读性），也无法对输出做黄金测试；重复注释让读者以为存在两个函数。
修法：删重复注释；把 labels 改成有序切片 `[]struct{key,label string}` 按固定顺序遍历。工作量 S。

**U87 [P3] 重复注释块与全量重试式 schema 执行** — `internal/whisper/personality.go:450-452`
证据：`// BuildPersonalitySection 构建人格提示区块（按亲密度选择示例）` / `// BuildPersonalitySection 构建人格提示区块（按亲密度选择示例）` / `func BuildPersonalitySection(presetID string, stage RelationshipStage) string {`。
同一 doc 注释连写两遍（interpreter.go:469-471 的「心理健康软保护」标题同样重复两次，emotion_fusion.go:175-177 的「自然不完美」亦重复），是批量改写/合并时的残留；另 database.go:158 在 runMigrations 里先无条件 `db.Exec(SchemaV1)` 建立 schema_meta，随后 175 行的 for 循环在全新库（currentVersion=1）时又会执行一次 `migrations[0]=SchemaV1`，同一建表脚本走两遍。
影响：纯洁癖级：重复注释不影响行为；SchemaV1 双执行因全部 `CREATE TABLE IF NOT EXISTS` 而幂等，仅在首启多一次无谓 DDL 与潜在的表已存在告警渠道混淆（错误被包成「首次建表失败」，与实际语义不符）。
修法：删除重复注释行；runMigrations 里把「确保 schema_meta 存在」抽成独立的 `ensureSchemaMeta(db)`，与 migrations[] 的语义分开命名，避免读者以为 V1 会被执行两次是设计。工作量 S。

## 四、拆刀建议

刀序 = 执行顺序；每刀独立可验收、可回滚。「延续」= 与历史审计重合。

| 刀号 | 目标 | 涉及文件 | 风险 | 验收口径 | 延续 |
| --- | --- | --- | --- | --- | --- |
| 刀1 | 修 ExtractJSON 配对扫描并补三例单测（U17，唯一 P0） | internal/util/util.go + util 测试 | 低（纯函数；调用面 25 处只受益） | 单测：两对象拼接 / 尾部多 `}` / 字符串内含 `}` 三例通过；对现有 25 个调用点跑一次集成冒烟（任选 chapter.go:57 与 outline.go:173 两条链路手工走查 JSON 解析成功） | — |
| 刀2 | 拨号只用已校验 IP（U28） | internal/netclient/ssrf.go | 低（不改接口） | 单测：构造多 A 记录（公网在前、内网在后）断言拒绝；HttpBridge/booksource 抓取一个正常外网 URL 手工走查 200 | — |
| 刀3 | SSE panic 转错误帧 + usage 标失败（U22） | internal/ai/client.go:686-693 | 中（流式路径，需防 close 后写通道） | 单测：注入会 panic 的解析输入，断言消费方收到 `Error != ""` 的 chunk 而非空成功；`go test -race ./internal/ai/...` | — |
| 刀4 | 审计与角色写盘 fail-loud（U79、U21） | internal/whisper/desktop_router.go、internal/character/character.go | 低 | 单测：把 dataRoot/项目目录设为不可写，断言 blocked 分支返回带「审计不可用」的 Summary、角色 applyUpdates 返回 error；手工走查一次正常写入仍成功 | — |
| 刀5 | 关掉沙箱外执行下载物（U74），并决定 agent 回路去留（U83、U78） | internal/whisper/desktop_executor.go:106-120、agent_loop_runner.go、dispatch_router.go、desktop_capability_routing.go | 中（若这些入口实际可达，改动作影响用户功能，需先确认可达性） | 手工走查：UseComputer 下载并安装流程必须在确认卡后执行；`deadcode -test` 复扫 3 个文件的调用点计数为 0（或接线后非 0 且走沙箱入口） | 延续（whisper 死代码调查 B/D 簇） |
| 刀6 | 原子写与章节号解析各收一处（U2、U3、U5、U40-U42 相邻） | internal/project/project.go、internal/narrative/journal.go、internal/scene/scene.go、internal/style/profile.go、internal/export/export.go、internal/docmd/pdf.go? 否—仅 project 侧 | 中（落盘路径，回归面=用户数据） | 单测：fileutil.AtomicWrite 覆盖 fsync+MkdirAll 两例；`ParseChapterFileName` 对 001.md/001a.md/006a.md/001-2.md 的用例表；手工走查一次「保存章节→断电模拟（kill 进程）→重开无半截」 | 延续（CHANGELOG.md:6446 收敛残留） |
| 刀7 | project.Manager 读写加锁并把 plans.json 的锁下沉（U12、U11） | internal/project/project.go、internal/app/app.go:96-99 | 中（并发路径，需 -race） | `go test -race ./internal/project/... ./internal/app/...`；单测：并发 8 goroutine 各调 UpsertAnalysisV2/syncRewriteIndex，断言条目数=8 | 延续（v4.425 单点加锁） |
| 刀8 | 图片后端 5 份 switch 收敛到注册表 + backendType 常量（U43、U71、U4、U42） | internal/app/app.go、image_handler.go、characterlib_gen_handler.go、writing_state.go、internal/ai/image_backend.go | 中（设置页与生成链路） | HTTP 断言：对 5 个后端名各调一次生成接口，返回的 GetImageBackendType 均为 `ai.ImageBackendKind*` 常量；单测：未启用引擎的错误文案只有一份 | — |
| 刀9 | 办公/解析侧纯函数级收口：cd 五闸（U20）、time.Local→UTC（U63）、streamSpans（U59）、markdownBlockKind（U69）、重复 seq 报错（U48）、crosslink 裁剪（U51）、页数口径（U56） | internal/schedule/ops.go、xlsx.go、internal/docmd/pdf.go、internal/export/{html,docx}.go、internal/office/crosslink/crosslink.go | 低（均为局部纯逻辑，可逐条独立验证） | 单测：Validate 与 ops 两条通道对同一非法 durationUnit 都给同一错误文案；xlsx 在东八区外时区（TZ 环境变量）下解析结果与 parseISO 一致；C1:D6 区间取到 C、D 两列 | 延续（cd 双工期四刀已落，校验未收口） |
| 刀10 | 情绪域单一口径：标签元数据表（U49）+ 涌现状态单一来源（U50）+ 反应词状态会话化（U51） | internal/whisper/emotion.go、emotion_fusion.go、emotional_emergence.go、orchestrator.go | 中（影响每回合提示词内容，需比对生成文本） | 单测：九元标签逐一断言 labelMeta 命中（无默认分支）；grep 确认 Push*/Get* 与包级三变量已删；手工走查两会话并行各 10 轮，断言「最近用过」列表不互相污染 | 延续（死代码调查余 196 项） |
| 刀11 | 桌面安全纵深：normalizePath containment（U80）+ PowerShell 参数化（U81）+ 22 分支注册表化（U72） | internal/whisper/desktop_router.go、desktop_executor.go | 高（安全语义变更，可能改变现有可用路径） | 单测：`..\..\Windows\System32` 被拒（读与写两类 action 都拒）；`C:\WINDOWS\system32` 变体被拒；target 含换行/分号时断言不产生第二条 PowerShell 语句；手工走查一次正常打开应用仍然成功 | — |
| 刀12 | 结构拆分三刀（收益最大、可后置）：project.Manager 分域（U1）、ai.Client 分域（U25、U26）、applyOne 拆 op（U61） | internal/project/*、internal/ai/client.go、internal/schedule/ops.go | 中高（大范围搬移，需先有刀6/刀7 的测试护栏） | 构建通过 + 全量 `go test ./internal/...`；每刀单独提交，纯搬移不改行为（diff 中无逻辑变更）；先做 U26 的常量提取作为热身 | 延续（上帝文件清单） |
| 刀13 | OCR 三口径收敛到 provider seam（U54、U76、U53） | internal/docmd/ocr.go、internal/ocr/herdsman.go、internal/app/gaea_ocr.go | 中高（涉及本地/远端进程与模型选择） | 单测：把 herdsman 注册为 provider 后，四路串联只剩一路分派；错误信息含各层真实原因（errors.Join 断言四段）；手工走查一次本地 Ovis 与一次远端 MinerU | 延续（docmd/ocr.go 自称 seam 范式） |
| 刀14 | 测试与注释洁癖批（U24、U87、U9、U36、U46、U16、U44） | internal/modelengine/engine_test.go、internal/whisper/personality.go、internal/style/profile.go、internal/novelgate/gate.go、internal/core/{sessionmode,modes}.go、internal/novelcontext/novelcontext.go | 低 | 拆测试只移动函数不改断言（`git diff -M` 显示 rename）；重复注释行删除后 `gofmt`/构建通过；itoa 删除后 grep 0 命中；charIndex 删除后单次编译只读一次 characters.json（加日志计数断言）；会话模式删掉一侧后前端 OfficeGetMode 重启仍保持 | — |

刀序理由：刀1~刀5 是「已实际咬人或直接暴露面」的止血刀（一个 P0 + 一个 SSRF + 一个静默失败 + 一条沙箱外执行路径）；刀6~刀11 是把「多份实现」收敛到单点，收益按调用点数量降序；刀12 才是结构大拆，必须有刀6/刀7 的测试护栏在前；刀13 依赖刀1（OCR 错误聚合要用到 JSON 工具收口后的行为）与 X1/刀5 对桌面链路可达性的结论；刀14 是零风险洁癖批，可随时插入。

## 五、观察项 / 未定论

以下疑点**证据不足以定级**，不进入 TOP 榜与刀序，需后续实测或跨单元核对（全部来自单元 JSON 的 open_questions，去重合并）：

1. **project 侧并发是否已真实咬人**（IN1）：`UpsertAnalysisV2`/`syncRewriteIndex`/`WriteChapterMemories` 是否有真实并发调用路径未证实——app 层只在 getPM/setPM 用了 writingState.mu。需在 httpbridge 本机面双客户端或 chapterGen 协程 + 分析协程组合下用 `-race` 复现，才能把 U12 从结构债升级为「已咬人」。
2. **types ↔ novelgate 是否真会成环**（IN1）：`internal/types/plan_v1.go:20` 自陈「避免环依赖」。若把 Issue 下沉 types，需先 grep 全仓 import 图确认 types 当前是否已 import novelgate，再定 U35 的修法。
3. **scene.Manager 是否存在同章双实例并发写**（IN1）：被 5 处各自构造（project.SceneManager 842、chapterHasBodyContent 535、synthesizeChapterScene 836、MigrateV3ToV4 804 及 app 侧），每次 NewManager 只拼路径、无实例复用。需确认是否有真实路径。
4. **novelstyle/segment.go 的 8 组硬编码词表是否应外置**（IN1）：functionWords/connectives/adjAdvWords/fourCharSet/metaphorMarkers/registerBreakWords/emotionDirectWords 是 Go 变量，而 words.json/patterns.json 已确立「领域词表不写死」纪律（words.go:3-7 明写用户拍板红线）。它们不被覆盖文件替换意味着「去味」行为无法不发版调整。
5. **头像是否真的双存**（IN1）：需在实际项目目录取样（本次只读代码，未读用户数据），确认 U62/U73 的两套命名是否产生重复文件。
6. **bookimport.CallJSON 是刻意独立还是历史复制**（IN1）：reconstruct.go:241-247 注释提到对齐 MuMu 的 expected_type。若要按 U10 收敛到 util，需确认 util.RetryJSON 是否支持「期望类型校验 + 失败原文回注」（当前签名未见 kind 参数）。
7. **bookimport 章节头识别与 project 章节口径是否冲突**（IN1）：导入按文本启发式（isStrongHeading/isWeakHeading/windowSplit），project 按文件名。导入稿是否会产出不符合 NNN.md 约定的文件名而让 ForEachChapter 漏扫，需一次端到端导入验证（本次禁跑测试与构建）。
8. **两套配置系统的排查顺序**（IN2）：internal/config（.gaea_config.json）与 internal/gaea/config（config.toml，UserConfigDir/gaea）同目录共存，字段名/风格/读取时机都不同。除代理配置外是否还有其它跨界读取？桌面端「设置不生效」应先看哪一份？
9. **core 的 write_file/delete_file/move_file 是否有确认卡或路径白名单**（IN2）：本次只看了 core 与 app/office_handler 两层，未追到实际调用链（可能在 whisper 的 desktop agent 回路里）——与 U72/U83 冲突面重叠，需一次联合核对。
10. **三套 BM25 是否有域分工**（IN2）：internal/memory（小说故事记忆）、internal/gaea/memory（办公记忆）、internal/gaea/bm25（通用排序）是明确分工还是历史遗留？确认后才能决定「合并」还是「保留但统一分词与参数」（影响 U34 的刀法）。
11. **httpbridge.Bridge 是否真有多实例场景**（IN2）：桌面内嵌桥 + 网页调试桥是否真实存在？若不存在，globalHub 可安全保留但需注释写明单例前提。
12. **ExtractJSON 的多对象失败是否已咬人**（IN2）：本次只在代码层面证明算法错误，未检索 CHANGELOG 与 `.gaea/AGENTS.md` 的对应条目。这直接决定 U17 是 P0 还是 P1——建议刀1 之前先花 10 分钟检索坑记录。
13. **failover 与健康巡检是否抖动**（IN2）：故障转移 v0（默认关闭）与 modelengine 健康巡检（10 分钟周期）联动时，是否出现「引擎被判失败→转移→巡检又标回 Connected→请求回切」？需追 C 刀 v0 开关是否已在 app 层默认打开。
14. **ObjStm 型 PDF 页数是否系统性失真**（IN3）：需用一个 ObjStm 型 PDF 验证 countPDFPages 与 pdftoppm 页数是否一致（U56 的严重度取决于此）。
15. **mpp 越界 panic 路径**（IN3）：mppI16/mppI32 在部分分支未见长度前置检查（mppParseVarData 只判 `len(metaRaw) < 24` 后即按 entrySize 步进读）。需补畸形样本或 fuzz 确认 U64/L3 的风险等级。
16. **是否存在第二处 HTML 导出入口**（IN3）：internal/export/html.go 的 CompileTemplate 三模板与 internal/app 的 sanitizeFilename 是否与 novel_bookcover 共用同一 CSS 常量？本次未读 app 侧实现。
17. **docx 预览面板走哪条解析链**（IN3）：office.go 的 docxToMarkdown 与 office/docxedit 的 parseParagraphs 是否存在两条调用链（预览面板 vs format_convert 走不同实现）？`.gaea/AGENTS.md:188` 称两者共用 docmd，但 docxedit 自带一套解析（U55/U67 的暴露面取决于此）。
18. **whisper 死代码基线需复跑**（IN4）：deadcode 调查的 590 项已在刀4（commit 12f77401）收敛到 196 项；本次禁构建只能按 grep 调用点核实。整文件级可删的三个（dispatch_router.go 284 / desktop_capability_routing.go 289 / agent_loop_runner.go 288）建议主代理跑一次 `deadcode -test` 复核（U78/U83 的验收依赖它）。
19. **桌面助手链路是否已上线**（IN4）：ExecuteUseComputer 的生产调用点只在 agent_loop_runner.go:254（而 RunAgentLoop 本身 0 生产调用）。若整条 agent 回路未接线，U72/U74/U80 的实际暴露面取决于是否有非 whisper 包（internal/app）直接调用 ExecuteUseComputer——本次 grep 未在 internal/app 命中。
20. **app 层是否另有一套同类实现**（IN4）：本次足迹只读 internal/whisper，未横向比对 internal/app/whisper_handler.go 是否重复实现了 Tier A/B 组装或时间上下文注入（U70/U52 的影响面取决于此）。
21. **九元标签两套映射的取值是否漂移**（IN4）：需一份「随机 10 万组四维取值分别跑两套映射」的对照；本次仅能证明表结构重复，无法证明内容一致（U49 的刀法取决于漂移是否存在）。
22. **涌现实例字段是否随 CloneFullState 携带**（IN4）：state_persistence.go:85 提到「下一轮 PreLLMTurn 的原地修改（UpdateUserProfile/AdvanceOriginStreak/涌现相位）」，但克隆/恢复路径是否携带这几个私有字段无法从只读抽查断定（U50 删全局后是否丢状态取决于此）。

## 六、单元评分明细与证据索引

| 单元 ID | units/<ID>.json | score | 单元 findings 数 | 本分册映射的 U-ID | 本分册条目数 | hotspots 前 3 |
| --- | --- | --- | --- | --- | --- | --- |
| IN1 | `docs/code-audit-2026-10-02/units/IN1.json` | 4 | 20 | U1–U21（缺号 U14/U18）、U32、U44、U47、U62、U73 | 23（P1 12 / P2 10 / P3 1） | project/project.go(1011)、novelcontext/novelcontext.go(913)、character/character.go(799) |
| IN2 | `docs/code-audit-2026-10-02/units/IN2.json` | 5 | 20 | U22–U42 区段（U23、U25–U32、U34–U42 等） | 23（P0 1 / P1 14 / P2 8） | ai/client.go(1429)、gaea/config/config.go(779)、modelengine/stats.go(758) |
| IN3 | `docs/code-audit-2026-10-02/units/IN3.json` | 4.5 | 20 | U43、U48、U51、U53–U61、U63–U69、U76、U85 | 20（P1 12 / P2 5 / P3 3） | schedule/mpp.go(924)、office/docxedit/docxedit.go(827)、docmd/pdf.go(705) |
| IN4 | `docs/code-audit-2026-10-02/units/IN4.json` | 4 | 19 | U49、U50、U52、U70–U75、U77–U84、U86、U87 | 18（P1 8 / P2 8 / P3 2） | orchestrator.go(972)、desktop_executor.go(412)、emotional_emergence.go(462) |
| 合计 | 4 个单元 JSON | 4.4（均值） | **79** | U1–U87（缺号 U14/U18/U33 为同根因合并空位） | **84**（P0 1 / P1 49 / P2 28 / P3 6） | 覆盖 32 个热点文件，全部见各单元 hotspots 数组 |

> 单元 findings 合计 79 与本分册条目 84 的差额 5 条：core/office 平迁（IN2-06）在 coupling 与 duplication 各记 1 条（+1）；头像本地化（IN1 热点/open_questions）按两套命名各记 1 条（+2）；`config.go` 装配（IN2 热点）与 `agent_loop_runner.go`（IN4 热点）各补 1 条（+2）。均为同一单元内不同切面，不引入新证据。

单元 metrics（取自 JSON，供复核工作量口径）：

| 单元 | files_reviewed | loc_reviewed | test_files | test_loc |
| --- | --- | --- | --- | --- |
| IN1 | 32 | 9,825 | 41 | 8,367 |
| IN2 | 22 | 8,600 | 51 | 8,600 |
| IN3 | 36 | 11,510 | 38 | 6,840 |
| IN4 | 123 | 19,510 | 34 | 6,944 |
| 合计 | 213 | 49,445 | 164 | 30,751 |

类别条目数（唯一权威口径 = `三、分类清单` 的小节标题与列出的条目）：

| 类别 | 条目数 | 类别 | 条目数 |
| --- | --- | --- | --- |
| god-file | 8 | complexity | 4 |
| god-func | 5 | dead-code | 5 |
| duplication | 16 | legacy-residue | 5 |
| consistency | 19 | test-smell | 2 |
| coupling | 5 | doc-drift | 2 |
| concurrency | 2 | | |
| error-swallow | 6 | **合计** | **84** |
| safety | 5 | | |

> 上表逐条与 `三、分类清单` 的 U-ID 清单机械核对一致（84 个 U-ID，全篇唯一）。`二、屎山 TOP 榜` 的 20 行中有 3 行为「见 Uxx」指针（U4→U43、U71→U43、U45→U31），指针不重复计数；U43、U31 各为其根因唯一的 canonical 条目。

严重度分布（按 84 条 U-ID 逐条对齐单元 JSON 的 `severity` 字段核验）：

| 严重度 | 条数 | 主要构成 |
| --- | --- | --- |
| P0 | 1 | U17 ExtractJSON 取末个 `}`（25 处生产调用点） |
| P1 | 49 | god-file 8 + god-func 5 + 静默失败/安全/并发/口径主线（详见 `二、屎山 TOP 榜` 与各分类小节） |
| P2 | 28 | 局部重复、口径不一致、可读性、历史残留 |
| P3 | 6 | U44（索引重建）、U51（crosslink 裁剪）、U76（OCR 测试夹缝）、U85（同）、U86（占位实现）、U87（注释与 DDL 洁癖） |
| **合计** | **84** | — |

证据索引：所有 `file:line` 均可在对应单元的 `findings[].file/line` 与 `hotspots[].path` 中直接定位；本分册未新增任何未在单元 JSON 中出现的文件或行号。
