# gaea 项目记忆

> 本文件为项目长期记忆（文档记忆层级）。编码规范：**UTF-8 无 BOM**（历史遗留的 GBK/UTF-8 混合编码已清理）。
> 修改后请保持 UTF-8；.ps1 脚本需 UTF-8 带 BOM（见「沙箱环境备忘」）。

## 版本状态（顶部速览）

> 更早版本见 `docs/archive/agents-version-history-2026-09.md`（覆盖 v4.173–v4.146 与 v4.49 及更早）；**v4.50–v4.145 区间本仓只在 `CHANGELOG.md` / `releases/` 有记录**（速览逐版滚出后未回填）。本段只留最近 14 版：本文件曾达 104 KB 超工作区指令预算（65536 B），尾部纪律/规划段一度对后续会话不可见，2026-09-10 二次分流。

- **品牌资产补齐（2026-09-10，非版本刀）**——2026-09-10 15:20 的「品牌资产刷新」只覆盖 **4 个 SVG**（`build/appicon.svg` / `frontend/public/favicon.svg` / `gaea/assets/logo.svg` / `logo-light.svg`，新视觉=「地核 G」轨道环抱活核），**两个二进制件没跟上**：`build/appicon.png`（1024²）与 `build/windows/icon.ico` 停在 **2026-08-05** 旧视觉（翡翠球体+破土嫩芽+星芒，深蓝底 `#0F172A`）→ **exe 内嵌图标/任务栏/资源管理器/桌面快捷方式全显示旧 logo，只有窗口内 UI 是新 logo（「换了一半」）**。**落地**=① 无头 Edge 渲染 `appicon.svg` →1024×1024 RGBA PNG（**必须传 `--default-background-color=00000000`**，否则圆角外合成不透明白、alpha 静默丢失）；② Pillow 由同一母图出 **7 档 ICO**（16/24/32/48/64/128/256，旧资产缺 24，本次补 Windows 标准全集）；③ `wails build -s -ldflags "-s -w" -trimpath`（**13.5s**）。**验证三条独立证据**=① 几何精度（核心圆 r=50@512 实测直径 **200px**、圆心 (539.5,511.5) 对理论 (540,512)；环 stroke 48 实测线宽 **96px**；圆角 rx=104 对角线首不透明像素 d=**61** 对理论 60.9——零缩放零偏移）；② exe 内嵌图标提取（`ExtractAssociatedIcon`：底板 `#0B1210`/米色核/翡翠环=新「地核 G」）；③ dist bundle 新 logo 独有标记 `gaea-logo-ring`×3、`gaea-logo-light-ring`×3、`0B1210`×2 在册且旧独占色 `#6ee7b7`/`#047857` **零命中**+按旧 SHA256 全仓比对零旧 logo 残留。**门禁**=build exit 0 / 冒烟 200 / **纯资产刀（零前端零 Go 源码改动、绑定面不变）**。**产物**=`build\bin\gaea.exe` **48,572,928 B** SHA256=**5F811126131A21456D7C84B6D568EC2F0B4A748D17F60AD2814879C795FD4264**（桌面副本同哈希；中间态 903F9759…C36EC068=只换 PNG+主图 ICO 的一版，小尺寸优化后重建覆盖；原始 59C2B518… 系 v4.208.0 归档值，releases 档案自洽未动）。**坑**=① 无头 Edge 截图默认白底，CSS 里写 `background:transparent` 无效（截图不继承页面背景），透明基线拿旧资产角像素 alpha=0 对照；② **判新旧前须验该色是否新旧共有**——`#34D399` 同时存在于**新** logo 的 halo 渐变与**旧** logo 主色，拿它判会把新资产误报成旧的，判据只能用**独有**标记（SVG id 名 / 独占色）；③ `pwsh` 不在 PATH，手工跑 `scripts/smoke.ps1` 会 `CommandNotFoundException`（**非 app 故障，易误判为冒烟失败**），用 `powershell`（build.bat 内已有 fallback）；④ 运行中的 exe 可被 `Copy-Item` 直接覆盖（进程持旧映射），桌面副本无需先关 app，但**已加载实例不会换图标，需重启**。**小尺寸可辨识（同日追加）**=原方案对大图统一降采样，致 16×16 环宽仅 1.5px / 核 3.1px、G 字形糊成深色块；新增派生资产 `build/appicon-small.svg`（环 48→64、核 r50→62、去 halo），ICO **按尺寸分流源图：16/24/32 用简化变体、48 及以上用主图**——衔接依据=变体 32px 环宽 **4.0px** ≈ 主图 48px 环宽 **4.5px**（若在 24→32 切会跳）；**坑=Pillow 的 ICO writer 只接受单一源图**，混合源尺寸必须手工组装 ICO 容器（ICONDIR 6B + 逐帧 16B + PNG 帧，**256 档宽/高字段写 0**，写 256 溢出单字节）。ICO=7 档 44384 B SHA256=**775CAED3274EC199DF7F3E188190BFCDB3A319EDD01EF2A350699FC8D6E2BD02**。**未抬版本**——按 README 发版约定「文档整理、令牌对照、单点样式等小改记入 CHANGELOG，不单独抬版本」，本刀属视觉资产补齐，与 `progress.md` 的「工作空间与文档整理（非版本刀）」同类处置。

- **壳内全板块渲染健康巡检（2026-09-12，非版本刀，随 v4.237 线收尾）**——CDP 起壳逐 rail 页点击+错误边界断言：闲庭七页（闲庭首页/首页/聊天/小说/绘梦/模型中心/角色库）+书斋六页（首页/办公/造价数据库/记忆中枢/模型中心/青鸟）**13 页全绿零接管**；配方=walk-pages.mjs（rail 坐标 DOM 定位+逐页停留+错误文本断言+失败自动截图）；巡检结论=无其他「静默坏死」页面。零代码改动不抬版本。

- **DAG 6.3 壳内真机走查（2026-09-12，非版本刀，60c68a34）**——§6 最后一条余项清账（**§6 余项全清，DAG 6.3 线收官**）：CDP 9333 DOM 断言走查（不点危险操作不打模型）——注入体检 Drawer 真跑=GaeaMemoryEvalRun 真实 wire+真实库**端到端打通**（通过：五条结构不变量全部成立；晨报预载 2 条·104/600、本体 2 引用·181/600、门控四开关、toast；截图 .tmp/eval-drawer.png）；办公流水线区 dag-section 在位（空态引导正确）；全程 exceptionThrown=0/页面渲染出错=0。**观察池新增**=办公记忆库面板「不可用—未配置」与同库体检读出 2 条活跃并存——面板 available 旗与体检取数路径（hubOfficeStore 直连）口径不同，既有语义非本线回归，按需另刀。配方=.tmp/walk-v4243.mjs/.tmp/walk-dag.mjs（rail 悬停展开→按 title 前缀点库入口→按钮文本带计数须前缀匹配；**节点列表/状态徽标在展开层，先点 goal 展开**）。零代码改动不抬版本。

- **最新发布：v4.253.0（2026-09-12）「二遍扫描账本收官：#9 修 + #6 实测结案」**——①jobs children 级联链随 pruneTerminalLocked 淘汰联动回收（改前只增不清随后台派生累积）。②#6 filewatch 同步 WalkDir **实测结案**：skipDirs 已覆盖 node_modules/.git 等重目录，开发工作区 925 目录仅 **46ms**——异步化收益趋零而「启动窗口事件丢失+无周期重扫兜底」风险真实，风险大于收益维持同步。二遍扫描账本终态=**修 7、结案 1、观察 4（#1/#7/#11/#12 均已知低危或需独立设计）**。**门禁**=ci.ps1 全绿/版本三处 4.253.0。

- **最新发布：v4.252.0（2026-09-12）「启动链收尾：远程剧照迁移异步化（普查二遍#3）」**——冷启动关键路径再摘一颗：改前 Startup 内同步串行下载全部远程剧照（xAI 临时图，每张 30s 超时，N×30s 全算进冷启动）；改后台协程（recover 守卫，沿用 app 刷新协程先例）。**异步化安全前提核实**=下载期间不占库连接（先收集 todo 再逐条下载+短 Exec），与启动期其他角色库操作在单连接池上安全交错；迁移完成前角色卡显示远程 URL=迁移前既有状态语义零变化；幂等（未迁移的下轮重试）+本体测试覆盖。二遍扫描账本更新=修 6 观 6（#6 filewatch WalkDir 就绪语义维持观察，#1 browser 状态机维持观察）。**测试**=characterlib 迁移本体+app 全量绿。**门禁**=ci.ps1 全绿/版本三处 4.252.0。

- **最新发布：v4.251.0（2026-09-12）「后端第二遍扫描：goroutine/内存/启动链（修5观7）」**——换镜头复查首遍欠覆盖维度。**goroutine 泄漏零**；12 项净修 5：①MCP StartAvailable 并行化+单 server 30s 上限（改前串行无 deadline 挂死 server 无限期卡装配；**AfterFunc 取消仅失败路径生效——成功连接 transport 绑定该 ctx 定时器必须 Stop 否则误杀**；测试=挂死子进程(**坑=select{} 触发死锁检测器自己崩报 EOF,须 sleep 循环保活**)+双挂死好 server 耗时上界 3.5s 守卫并行性）②weixin apiPost 每请求 ctx 超时替代克隆 Transport（长轮询永久每轮新建+TLS 握手）③notifyStart 异步化（Startup 链 10s×N）④tasks 节流 map 随 LRU 回收⑤copyPath 流式拷贝（迁移内存峰值=最大文件→io.Copy）。观察 7 项入普查档二遍扫描节（#1/#3/#6 触及启动语义需单独设计）。**门禁**=ci.ps1 全绿/版本三处 4.251.0。

- **最新发布：v4.250.0（2026-09-12）「走查遗留收账：语义召回搜索路径加界 + 观察项②撤销」**——刀F 处理 v4.249.1 入池两项观察：①语义召回搜索路径加界——tool/app 两侧 semanticCostRecall 的 st.Search 内含 Ensure 增量向量化，ctx 60s 包住「追赶+查询」（1509 条库首次追赶 >120s，真机实测 60.1s×2）；**实态收窄=Ensure 按批持久化增量收敛，60s×2 属追赶窗口非常态**；刀法=ctx 60s→5s 双侧加界，超时回落关键词结果（语义召回只增不改基线，与改前 ctx 超时语义一致快 12×），追赶由既有显式补齐 GaeaSemanticIndexBackfill 完成；**导入后台钩子提案与档头既有拍板「不做后台守护协程」冲突已撤回**。②~~Available() 探测口径 /models 404~~**撤销**——引擎实盘 base_url 本就含 /v1（connected=true、bge-m3 在册），探测路径正确；走查时 curl 打了缺 /v1 的猜测 URL——**证据未复核即入池，同 #8 教训**。零绑定零前端。**门禁**=ci.ps1 全绿/版本三处 4.250.0。**「优化 gaea 的后端」线程全收**。

- **最新发布：v4.249.1（2026-09-12）「走查补刀：app 侧检索客户端单例化 + 五刀真机走查清账」**——后端性能五刀真机走查（CDP 9333 DOM+绑定断言，配方 .tmp/walk-v4249.mjs：**壳内绑定挂 window.go.app 门面（OfficeB/CostB/MemoryB…）非 window.go.main.App，跨门面解析+Wails 逐参显式传**）——绑定全通（看板绑定亚毫秒=条目缓存真机生效；GaeaCostList 真实库 1509 条 62ms）、记忆中枢/造价数据库零接管、exceptionThrown=0。**新发现一并修**（普查#4 的 app 侧孪生）：localSearchEmbedder/localSearchReranker 每调用 new 新客户端改按 (baseURL,model) 键单例。**观察池新增**=①GaeaCostSearch 语义召回在 UI 绑定内同步 Ensure 全库向量化（60s ctx，关键词召回<3 时可阻塞 60s/次，实测 60.1s×2）②Available() 探测口径 /models 404（herdsman 实际部署不实现）→ Available 恒 false——两项按需另刀。**门禁**=ci.ps1 全绿/版本三处 4.249.1。

- **最新发布：v4.249.0（2026-09-12）「后端性能刀E：面板/杂项收官（看板缓存 408×+预览缓存+无界缓存上限）」**——普查档最后一刀，**五刀 A/B/C/D/E 全清收官**。①**会话条目缓存**（普查#5）：看板/节点详情/轨迹/Agent 网络（+resync 共五处）绑定共用 ReadEntriesFor 全量读+逐行解析，前端轮询每回合同份日志反复解析（基线 2MB 日志 9.1ms/6.7MB 每刷）；新 readEntriesForCached——事件日志 **append-only** ⇒ size+mtime 未变即内容未变，缓存解析结果（上限 8 会话；fold/detail/trajectory 层核实只读共享；resync 保持直读）。**看板刷新 9.1ms→22μs（408×）**。②**预览缓存+轻量解码**（普查#7）：列表点换 previewSessionCached（size+mtime 键上限 64）+解码只取 role、content 仅首条用户消息才反序列化。③wssearch textCache 加 64 条上限（普查#10 无界增长）。④graph linker 循环内正则改字符串操作（QuoteMeta 字面量等价，普查#16）。⑤sanitizeFilename 改 strings.Builder（#18）。⑥三处函数内 MustCompile 提包级（#20）。⑦SemanticRecall 死代码删除（#19 零消费方核实；DocText/Cosine 保留）；#17=legacy file backend 观察不修。**测试**=Go+2（两级缓存失效语义）+app bench 1 组。**门禁**=ci.ps1 全绿/版本三处 4.249.0。

- **最新发布：v4.248.0（2026-09-12）「后端性能刀C：SQLite 连接池放宽 + 批量清理事务化」**——普查档刀C（风险最高单独走）。①**连接池 1→4**（普查#6）：Hephaestus.db 改前 MaxOpenConns(1) 把读也串行化（WAL 本可读写并行），且「rows 未关闭时 Exec 等唯一连接」是整类死锁根源（portrait.go/whisper fts.go 两处在册注脚）；放宽后死锁类整体消除（注脚注释更新、收集再写回习惯保留）；写并发由单写者锁+DSN 逐连接 busy_timeout=5000 兜底（modernc 对每新连接应用 DSN 参数，PRAGMA 不丢）。**Benchmark**（2s 时间制）：并行点查 7.6μs→2.5μs（**3.0×**）。②**批量清理事务化**（普查#11）：CleanupArchived 单事务并收紧原子语义（改前删一半失败仍把整批 doomed 报审计=虚报，现在全删或全不删）；semantic.Stale 同款。**测试**=Go+2（并发写 8×25 全落库零 BUSY；写事务进行中另一连接可读+WAL 快照隔离——改前单连接下此测试死锁=新形态守卫）+db bench 2 组。**门禁**=ci.ps1 全绿/版本三处 4.248.0。**普查档余**=刀E 面板杂项，待拍板。

- **最新发布：v4.247.0（2026-09-12）「后端性能刀D：cost 检索固定成本（BM25 缓存+客户端单例+子查询消除）」**——普查档刀D，benchmark 先行。①**BM25 排序缓存**（普查#3，T7-3 原设计接通）：Search 每查询对命中子集从零重建倒排（2000 条库 20.6ms/22MB）；**刀法修正**=bm25.Cache 原设计语料整库与现行为子集排序语义错配，改**包级缓存**（key=db池+数据版本+过滤形态，语料=该形态 SQL 全捞全量 name 序；Store 即建即弃不能挂实例），子集按语料下标取分 tie-break=name 序与改前一致，**失效全链**=Save/Delete/分类+SelfHeal 实际修复推进版本+app 批量导入 cost.InvalidateRankers 显式失效，map 超 16 清空。②**Embedder/Reranker 单例化**（普查#4）：按 retrievalRuntime 配置键缓存实例（探测缓存从此可命中、keep-alive 复用；SetRetrievalRuntime 变更自动重建，构造失败不缓存）。③**子查询消除**：逐行相关 COUNT 改 LEFT JOIN (GROUP BY) 单趟，ComponentCount 语义不变。**Benchmark**（20x）：Search 500 条 2.4×/2000 条 2.1×（分配 22→10MB）。**测试**=Go+1（失效语义三态）+bench 2 组；TestCostSearchBM25Order 原样过=排序口径守卫。**门禁**=ci.ps1 全绿/版本三处 4.247.0。**剩余刀路**=刀C SQLite 读写分离/刀E 面板杂项，待拍板。

- **最新发布：v4.246.0（2026-09-12）「后端性能刀B：记忆写路径（重载出锁+写路径事务化）」**——「优化 gaea 的后端」刀B（普查档 20 项交拍板池，benchmark 先行）。①**全库重载出全局锁**（普查#2）：九个记忆写点改前在 c.mu 内跑 memory.Load（50 条=2.3ms/300 条=7.0ms）阻塞 Send/Cancel 等全部控制器操作；**刀法修正**（非原案后台刷新）=刷新对调用方仍同步（写后立即可见语义与测试依赖不动），Load 改锁外跑——beginMemoryReloadLocked（锁内收口 Options+推进代号 memGen）→解锁→finishMemoryReload（代号守卫：gen≥已换入才换入，慢旧 Load 被拒不回退快照，被跳过的写必然已被更高代 Load 读到）；applySpace 推进代号使在途旧空间快照失效。②**写路径事务化**（普查#12）：五写方法（Save/Archive/Unarchive/ChangeType/touchInSpace，Touch 在引用解析热路径）commitWithEvent 单事务，事件尽力而为失败回退单语句重放（写入本体不丢）；EventLog 抽 execer（事件必须落事务连接——Hephaestus.db 单连接池否则死锁）。**Benchmark**（20x）：Save 0.95→0.49ms/Touch 0.86→0.45ms（均 1.9×）。**测试**=Go+5（事件失败回落保本体/原子提交/未命中零留痕/并发收敛 4×5 全到达）+bench 3 组。**复核撤销普查#8**（BuildSearchIndex 实际分词 body，勘误在档）。**门禁**=ci.ps1 全绿/版本三处 4.246.0。**下一刀候选**=刀C SQLite 读写分离/刀D cost 检索/刀E 面板杂项，待拍板。

- **最新发布：v4.245.0（2026-09-12）「后端性能刀A：回合边界税清偿（事件日志续接+摘要增量化）」**——「优化 gaea 的后端」首批落地（普查档 docs/gaea-backend-perf-survey-2026-09.md 20 项交拍板池，本刀=刀A 两项，先立 benchmark 锁基线再动刀）。①**事件日志续接**（普查#1）：sink turn_done 干净关闭后记「关闭时 seq+刷盘后文件大小」，重开走新 OpenLogResuming——大小未变即跳过 RepairLogFile+countLogLines 全量读+逐行解析直接续 seq；大小不符（外部追加/轮转/截断/删除）自动回落 OpenLog 全量路径，回合间日志可被外部触碰的设计红线由大小核对保住；OpenLog 本体与 save/migrate/subagent_promote 三处一次性调用方零变化。②**摘要增量化**（普查#14）：stream() 每步 DigestMessages 全量重哈希改 DigestCache——相邻请求前缀**指针级身份**核对（字符串不可变⇒数据指针+长度相等即字节相等；ToolCalls 同底层数组），未变消息复用摘要只重哈希新增/改写，身份不符一律重算（前缀稳定证据链只会重算不会错判），Digest 与 DigestMessages 逐元素等价。**Benchmark**（20x）：日志重开 2MB 19.1ms→0.96ms（20×，与大小无关=O(1)）；摘要步进 300 回合 521μs→49μs（10.6×）。**测试**=Go+9（续接/外部追加回落/截断/删除/sink 跨回合/外部写回落；缓存等价六形态/复用计数/重建切片保守重哈希）+bench 3 组（gaea 热路径首批 benchmark）。零绑定零前端。**门禁**=ci.ps1 全绿/版本三处 4.245.0。**下一刀候选**=刀B 记忆写路径/刀D cost 检索，待拍板。

- **最新发布：v4.244.0（2026-09-12）「记忆中枢办公面板口径对齐（观察池清账）」**——v4.243 真机走查观察池项：面板「不可用—未配置」而同库生命周期/体检读出 2 条活跃。**根因**=GaeaMemory() facts/available 等引擎控制器，数据实体却在 SQLite 可直读（hubOfficeStore=与生命周期/体检同路径）。**修复**=facts/available 改管理面直读（available=用户目录可解析），docs/scopes/storeDir 仍引擎域、控制器就绪时补充——「不可用」退化为真不可用。零绑定零前端。测试 Go+1（控制器未构建形态 facts 直读）。**门禁**=ci.ps1 全绿/版本三处 4.244.0/真机复验=面板显示事实列表。

- **最新发布：v4.243.0（2026-09-12）「运行中 steer 直穿 + 危险操作分级审批（DAG §6 末项）」**——roadmap §12.4 后半句清账。①**直穿**=主对话 GaeaSteer 同机制下沉子代理：agent 包在跑登记 subRunners（ref=SessionID→AgentRunner，runSubAgentInternal 起跑登记/defer 注销）+SteerSubagent(ref,text)；GaeaDagNodeSteer 放行 running 节点——凭 node.Ref 注入 steer 队列（不打断工具执行下一回合生效），查无登记如实报错不静默转续跑。②**审批分级**=dag_plan 节点参数 risk（normal 缺省/high=覆盖删除既有文件、批量移动、全局性改动，Validate 拒非法值）；**执行器起跑前置闸**：high&&!Approved→置 StatusHold 待审批+波收尾**链停**（下游保持 pending 不级联跳过）；新绑定 GaeaDagNodeApprove：hold→pending+Approved=true **只放行不自动跑**；已批准重跑不重新挂闸；ApplyEdit 把 Risk 计入形状比较——**风险变更=回 pending+审批归零**；FromRun/Instantiate risk 随模板 approved 剥净。前端：hold 徽标 var(--warn)+「批准」按钮+running 直穿改向框（占位区分）。**绑定 627→628**（再生+手工补行+checkout 还原单参化，diff 恰 1 行）。**测试**=Go+4（风险值/ApplyEdit 归零/模板携带/审批闸端到端+SteerSubagent 寻址三态）+vitest +2（hold 批准/直穿占位）。**门禁**=tsc 0/ci.ps1 全绿（vitest 332 文件 2893 例）/drift OK@628/版本三处 4.243.0。**§6 余项剩一条**：壳内真机走查（等闲置窗口）；候选4 价格带数据源待拍板。

- **最新发布：v4.242.0（2026-09-12）「办公流水线成品直出首刀：一键验收（市场调研候选3）」**——防御性首刀（LM Studio Bionic 竞品信号），DAG 设计档 §6 余项 DeliverableRegistry 用户面清账（进度续写该档不另立）。**口径**=成品=done/accepted 节点 outputs 的 run 级汇总；**验收语义零变更**——仍人拍板（一键=一次拍板覆盖清单，两段式按钮首击武装「确认验收 N 节点」再击执行、重拉自动解除防陈旧误击），不自动验收不定时验收（「人拍板回流记忆」红线不动）。**落地**=①提取单节点验收记忆回写核心 dagAcceptMemoryWrite（单一真源单验收/一键同语义）+新绑定 GaeaDagAcceptAll（锁内一次翻转全部 done+save 一次；锁外逐节点回写单条失败不阻断汇总如实上报=「验收已成立不回滚」同哲学；空验收提示非错误；**绑定 626→627**，gen_bindings 再生+office 门面手工补行 checkout 还原单参化，diff 恰 1 行）②DagPanel run 头「M 件成品」徽标（title=文件清单）+「一键验收 N」两段式按钮（运行中/无 done 隐藏）+五处接线（锁 448→449）。**测试**=Go+2（一键全收/空验收/混合态只收 done 余量+单验收回归重构语义不变）+vitest +2（成品徽标/两段式武装+running 隐藏）。**门禁**=tsc 0/ci.ps1 全绿（vitest 332 文件 2891 例）/drift OK@627/版本三处 4.242.0。**§6 余项剩**：运行中 steer 直穿+分级审批（等审批闸分级面）/壳内真机走查；候选4 价格带数据源待拍板。

- **最新发布：v4.241.0（2026-09-12）「记忆注入质量可评测（市场调研候选2）」**——与检索评测分立：GaeaRetrievalEvalRun 测「查得到」（Recall@10 统计门槛），本刀测「注得对」——对真实库 work 视图跑与装配点**完全相同**的两个真实构建器（晨报预载 v4.16+项目本体 6.2），断言结构不变量（预算合规×2 600 runes/归档与跨空间零泄漏=**精确路径**条目名+引用键逐点核对，块全文子串兜底明确不做——work 正文合法提及归档名不是泄漏子串必误报/引用零悬空）；门槛=零违规（确定性规则与 0.8 分立），固化覆盖是信息面不判死（预算挤兑未全收=设计内），门控三开关透明下发关态体检照跑。**形态**=纯核 memory.EvalInjectionBlocks+绑定 GaeaMemoryEvalRun（**625→626**，gen_bindings 再生）+前端六处（types 手写接口 MemoryArchivedPage 先例避开 wailsjs models 再生依赖/spaceBindings 锁 447→448/mock 全合规样例）+办公记忆库「注入体检」Drawer（渐进披露不加一级房间）。**测试**=Go+7+vitest+3。**坑两实锤=①gen_bindings 再生会回退 v4.237 手工单参化（脚本从 App 签名派生门面，七个变参签名被还原），E27 守卫当场拦下——以后 gen_bindings 后必须 git diff bindings_cost/office 凡含变参行即手工还原；②a.cfg=*appconfig.Config（偏好域）与 ga.cfg=gaeaConfig（引擎域 Memory.Enabled/SpaceModeIsOn）不可混用，裸 &App{} 读 a.cfg 因 core 未初始化直接 panic，测试须 &App{core: &core{cfg:...}}**。**门禁**=tsc 0/ci.ps1 全绿（vitest 332 文件 2889 例）/drift OK@626/E27 PASS/版本三处 4.241.0。**下一刀**=候选3 办公「成品直出」打磨（防御性先出设计）；候选4 价格带数据源待拍板。

- **最新发布：v4.240.0（2026-09-12）「市场调研落地 + 记忆双时间轴（SchemaV21）」**——先调研后动刀：八赛道市场调研落档 `docs/gaea-market-survey-2026-09.md`（微软 Copilot agentic 2026-04 GA=办公 agent 形态被巨头验证；Zep/Graphiti bi-temporal=2026 记忆层共识口径；LM Studio Bionic=本地 agent 工作台正面竞品，「本地+隐私」已是基线不再是差异点；候选 4 项交拍板池，已过删除史筛——平台化/MCP/跨设备记忆包/算量/PM 对照表/GoalCard 均不提名）。**本版落地候选1=记忆双时间轴**：SchemaV21 `memory_events` 加 `recorded_at`（事务时间=日志写入时刻，AppendEvent 服务端盖章调用方不可伪造，At 单一语义=事实时间；旧行 0=读取归一回落 At，V20 同口径）；投影 `eventTimeNote` 把「（发生…·记录…）」标注烘进事件节点 Desc（仅两轴分歧≥1s 才标注——写入路径同毫秒是常态零噪音；回填/导入类路径才透出）。**坑**=双轴曾计划作为 GraphNode 字段，被 TestGraphRebuildFromLog 拦下——mem_graph_* 物化表没有这两列，字段化破坏「物化=日志投影」逐字段可比不变量；改烘 Desc（desc 本就随投影物化）不变量天然保持——**给投影产物加字段前先查物化表有没有这列**。**测试**=Go +6（迁移升级/新库全链/盖章反伪造/旧行归一/cite 路径盖章/desc 标注边界）。零绑定（625 不变）/零前端改动（GraphView 渲染 desc 现成面）。**门禁**=db+memory+app 三包绿/tsc 0/eslint 0/ci.ps1 全绿/版本三处 4.240.0。**下一刀**=候选2 记忆注入质量可评测（LongMemEval 思路，与检索评测 GaeaRetrievalEvalRun 分立：检索面已建制，注入面=晨报预载/brief/衰减剔除口径待立）；候选4 价格带数据源待拍板。

- **最新发布：v4.239.0（2026-09-12）「数学公式渲染对齐：伴侣线接入 KaTeX」**——伴侣/聊天线（ChatMarkdown/MarkdownContent）一直没有数学渲染（$..$、$$..$$、\(..\) 裸显），办公板块却早有 KaTeX——同仓双标。**落地**：① 新 gaea/lib/mathText.ts 跨板块共享（normalizeMath/hasMathContent/ensureKatexCss 自 Markdown.tsx 提取，行为零变化）；② ChatMarkdown plain 管线加 remark-math+rehype-katex，text 先过 normalizeMath，有数学内容才注入 KaTeX CSS；③ MarkdownContent companion/流式同款（GenUI 覆盖件路径不受影响）。零新依赖零新 chunk（katex 经既有共享 chunk 复用）。vitest +2（两线 $..$→.katex 真实渲染）。**门禁**=tsc 0/eslint 0/ci.ps1 全绿/绑定 625/版本三处 4.239.0。**欠账**=真机抽查伴侣线数学回复（等自然使用窗口）。

- **最新发布：v4.238.0（2026-09-12）「观察池收官：回答反馈（点赞/点踩）落记忆事件」**——**设计取舍使「无消费方」顾虑消解**：反馈不做成孤立数据，而是记忆事件（op=feedback）落 memory_events——事件节点随 v4.210 语义图谱投影自然可查，消费者=事件图谱/事件日志现成面。**落地**：① memory/eventlog.go OpFeedback 扩员（closed set）；② sqliteBackend.AppendFeedbackEvent（摘要 eventExcerptLimit 截断+At/Actor 缺省补齐）；fileBackend 诚实报错（Pin 先例——绝不静默假成功）；spaceView 最小透传；Store 值方法分派（与 Pin 同形，无接口级联）；③ **App 绑定 624→625**（gaea_memory_feedback.go：rating 限 up/down 非法拒绝；事件字段 Name=messageID/Title 点赞回答|点踩回答/Space/SourceSession 当前会话/SourceMessage/Actor=panel；bindings_memory 透传一行）；④ 前端 AssistantMessage 操作行 👍/👎（icons +ThumbsUp/Filled/ThumbsDown/Filled，antd Like/Dislike wrap）——一次有效（事件追加式不可撤回，点击后双钮禁用）、失败回退可重试+error toast；Transcript onFeedback 开关全链透传（App→Transcript→TurnBlock→ctx→AssistantMessage 五环）。**测试**=Go +3（落库全字段含摘要 ≤280B 截断/投影只出事件节点不建实体——cite 悬空拒写同边界/文件后端诚实报错）+vitest +3（落库参数+双钮禁用/失败回退+toast/熄灯口径）。**坑**=①批量接线脚本末位断言崩掉整体不写盘，重做只补 props 声明漏了 ctx/调用点→真机按钮不渲染——**批量接线后必须从调用点正向全链 grep 复核（App→Transcript→TurnBlock→ctx→AssistantMessage 五环）**；②Wails 要求绑定参数逐个显式传（facade 可选省略=args:[] received 0 expected 1），调用点显式传空串（Go 侧空串语义核实安全）。**真机复验**=点👍→已反馈态+语义图 ev:10「feedback · 点赞回答」事件节点含摘要——端到端全链打通。**门禁**=Go app+memory 绿/vitest 87 例相关面绿/tsc 0/eslint 0/ci.ps1 全绿/绑定 624→625/版本三处 4.238.0。**观察池全清**；反馈深度消费（按反馈加权重排记忆/注入偏好）属记忆 OS 后续方向按需另刀。

- **最新发布：v4.237.0（2026-09-12）「变参绑定根治：绑定层去变参 + Wails 变参真相定论」**——v4.236 单串化方向对但没修到根：轨迹页走查暴露新错误（unmarshal string→[]string），**Wails v2.13 源码（ParseArgs 严格计数+reflect.Call 按 In(0)=string）+ 真机经验矩阵（七形态×双方法全败：传串 unmarshal 失败/传数组 reflect panic 被 recover 回调永不送达=promise 永久 pending）双重定论=变参绑定在 Wails v2.13 不可用**。修复=bindings_office.go 六处+bindings_cost.go 一处绑定门面 ...string→单 string 透传（核心层保留变参：resolveGaeaSessionPath 跳空回退当前会话/taskListInSpace("")=不过滤语义核实安全）；前端 facade 改必填串+调用点显式传 ""（Wails 要求逐参显式传，省略=count 错真机实锤）。**真机复验**=wire 错误 0+轨迹页红卡消失满载数据+任务管理实时行在位（对比 v4.235 红卡/v4.236 空面板）。**坑固化**=①Wails v2.13 绑定面禁用 ... 参数（核心层可保留）②修 wire 契约后必须重建 exe 再复验③被 .catch 吞掉的接口错误修完报错≠修完功能必须验证数据真到达。**门禁**=Go app 绿/vitest 87 例绿/tsc 0/eslint 0/ci.ps1 全绿/绑定 624 零变更/版本三处 4.237.0。**欠账**=观察池剩点赞点踩（等真实需求）。

- **最新发布：v4.236.0（2026-09-12）「修复变参绑定 wire 契约：reflect 参数错误清零」**——清 v4.235 走查观察池项。**根因（桥接约定与 Wails 真实形态不符）**：Go 五个变参绑定（GaeaAgentNetwork/GaeaTrajectory/GaeaContextView/GaeaContextNodeDetail/GaeaTaskList，均 ...string）要求每个变参元素作为独立顶层参数（args:["path"]），facade 却约定 JS 数组整体作单参（args:[["path"]])→reflect 报错；**推断=v4.174 退役的 wailsjsCompat shim 当年会展开数组，退役后数组调用点静默失去展开**，报错被 .catch 吞掉无人深究，直到壳内走查抓日志。**修复（facade 单可选串化——全部调用方只传 0..1 个路径，Go 变参天然支持）**：bridge/core.ts 五签名 string[]→?:string；七调用点同步简化（AgentNetworkCard/agentNetworkStore/TrajectoryView/ContextView/inspector 数组包裹拆除；TaskCenter/useRunningBadge TaskList([])→TaskList()）；mock/core.ts 四实现对齐；agentNetworkStore 补语义（poller 空串=内核会话哨兵，下发前归一 undefined）；UnifiedSearch/CostImportApply 调用方本就字符串/展开传参核实不动。**回归锁**=新 bridge.variadic.test.ts 4 用例（注入门面 spy 钉死顶层 string/零实参形态）；agentNetworkStore/ContextView 三处旧数组断言更新。**真机复验**=重建壳进办公工作台 reflect 错误 0 条（修复前必现）。**门禁**=tsc 0/eslint 0/ci.ps1 全绿/绑定 624 零变更/版本三处 4.236.0。**欠账**=观察池剩点赞点踩（等真实需求）。

- **最新发布：v4.235.0（2026-09-12）「壳内真机走查：抓到并修复 DAG 模板区 nil slice 崩溃」**——「测试绿≠真机通」再实锤：五刀（高亮×3/regenerate/折叠）壳内 DOM 断言走查（CDP 9333 起壳 + cdp-walk evaluate/--click/--shot），进办公工作台第一步撞上页面级崩溃 `Cannot read properties of null (reading 'length')`——vitest 330 文件全绿没拦住。**根因**=用户机器模板目录不存在 → `DagTemplateStore.List()` IsNotExist 分支 `return nil, nil` → Go nil slice 经 Wails JSON 序列化成 **null** → 前端 `setTpls(null)`（类型标注非空但边界没挡）→ `tpls.length` 炸；测试/mock 给的都是 []——v4.210「nil↔[] 漂移」同族、跨语言面。**修复（双侧归一）**=① Go dag.Store.List/TemplateStore.List 的 IsNotExist 分支返回空集非 nil；② DagPanel 三处绑定边界 `?? []`；③ Go 用例强化空目录 List 断言非 nil（原 len 断言对 nil 也过）。**真机复验**=重建壳重进正常渲染错误 0、办公流水线区在位。**走查其余结论**=data-hl 全链路双主题真机工作（`gaea-display-mode` dark→data-hl dark+body 切深；首查见 light 是翻错键——`gaea-dark` 是被压制的 legacy 键非 bug）；regenerate 按钮真实会话在位。**方法论沉淀**=①抓错误边界底下的堆栈：Runtime.enable 收 `Runtime.exceptionThrown`（边界只给人话）；②minified 崩点按 dist chunk 行:列切上下文反查源码；③`--key` 不带修饰符发不了 ctrl+4（组合键须 Input.dispatchKeyEvent 带 modifiers）。**新观察池**=GaeaTaskList/GaeaAgentNetwork 的 `reflect: cannot use []string as type string`（预置参数形态错，接口报错不崩页面，按需另刀）。**门禁**=Go dag/app 绿/vitest/tsc 0/eslint 0/ci.ps1 全绿/绑定 624 零变更/版本三处 4.235.0。

- **最新发布：v4.234.0（2026-09-12）「AI 输出渲染：超长代码块折叠 + i18n context value 修正」**——观察池第二项清账（对齐 Claude/ChatGPT 长代码处理）。**落地**：① 新 gaea/components/CodeCollapse.tsx 双板块共享——>30 行块级代码默认收起（max-height 380px+底部渐隐+展开/收起胶囊钮），≤30 行零包装直通；**跨 chunk 共享组件不能假定对方加载了 gaea/styles.css 令牌**——渐隐色调用方显式传（gaea 默认 var(--bg)/聊天线传 #0b0e14 同其恒暗面板），按钮用全局 M3 令牌内联样式，文案调用方注入 labels（gaea 走 useT 三语/聊天线硬编码中文一致）。② 新 useTOptional()：无 LocaleProvider 回退 zh 直译不抛错——Markdown 是被裸渲染的共享组件，useT 会让全部裸渲染用例连坐抛错（首跑 27 例挂）。③ 顺带修 LocaleProvider context value 不稳定（真回归，stash 二分+三步实验定位）：value 原先每渲染造新对象，Markdown 经 useTOptional 成为消费者后 en chunk 就绪的 forceRender 把它拖进重渲染级联、ReactMarkdown 整树重解析撕掉 MemCitationChip 弹层——修复=value useMemo 化（deps: locale/pref/setPref/tt），消费者只在语言真变时重渲染。**教训=Provider 的 context value 不 memo 化是埋雷，任何新消费者上线都可能引爆**。vitest +2+MemCitationChip 3 例转绿。**坑四证（固化铁律）**：python heredoc 追加源码第三次转义丢失——追加源码只走 Edit 工具无例外；后台 CI 管道 tail 截丢归因必须整份落盘。**门禁**=tsc 0/eslint 0/ci.ps1 全绿/绑定 624 零变更/版本三处 4.234.0。**观察池剩**=点赞点踩/壳内真机走查。

- **最新发布：v4.233.0（2026-09-12）「AI 输出渲染收尾：MarkdownContent 缺省高亮 + 明暗调色板目检」**——高亮三刀收尾。**① 目检补证**：v4.230/231 此前只有 vitest DOM 断言没有视觉验证——配方=Node 同款 highlight.js 语法包生成 go/sql/python/powershell 样例 hljs HTML → check.html 双 link 真实 gaea/hljs-theme.css（双面板=gaea 随主题 / 聊天线恒暗 .hl-scope-dark）→ 无头 Edge --screenshot 明暗两版人工目检：暗色层次清楚/浅色深阶对比良好/**恒暗面板在浅色主题下令牌仍钉暗色组按设计工作**。**② MarkdownContent 缺省高亮**：NovelSettingPage.tsx:189 直用无覆盖=全仓最后一个无着色渲染面——模块级 defaultComponents（引用稳定不破坏 memo）：未传 components 时块级代码走 ChatCodeBlock（与聊天线同款）、行内交还 .md-content code 默认样式、pre 透传防双层；传了 components（GenUI 缝）完全尊重调用方零变化（消费面盘点：无覆盖调用方仅此一处）。vitest +3。**③ 顺带根治 Go 在册 flaky（两天两度打挂 CI）**：TestCreateChapter_SameChapterConcurrentRejected 家族失败从来不是断言而是 t.TempDir() 清理竞态——CancelCreateChapter 取消路径先删登记表，被取消协程仍有「已生成部分落盘」尾步，waitGensDone 只等表空放行即撞 Windows unlinkat（directory not empty）；根治=writingState 增 chapterGenWG sync.WaitGroup（spawn 前 Add/协程首 defer Done，LIFO 故最后触发=协程真退出）+waitGensDone 两级等待（表空快速路径+WG 5s 超时）——**教训：登记表空≠协程退出，取消路径删登记与协程收尾之间有窗口**，-count=10 全绿（修复前定向复跑即可复现）。**坑三证**：python heredoc 追加含反引号 fence 的测试代码再次转义丢失产出语法错 JS——追加源码只走 Edit 工具无例外。**门禁**=tsc 0/eslint 0/ci.ps1 全绿/绑定 624 零变更/版本三处 4.233.0。**「AI 输出效果对齐同类产品」主线全清**（v4.230 办公高亮/v4.231 聊天线/v4.232 regenerate/v4.233 收尾）；观察池=点赞点踩（个人工具暂无消费方等真实需求）/超长代码块折叠/壳内真机走查等闲置窗口。

- **v4.174–v4.178（2026-09-09）瘦身 P3 版3/P4 三刀与造价条目匹配键刀**——全文迁入 docs/archive/agents-version-history-2026-09.md（2026-09-11 三迁腾预算，check-docs 指令预算守卫）。
## 执行纪律：默认并发子代理（2026-09-03 用户强化习惯）

用户已把「并发子代理」从点名指令固化为**默认习惯**：后续任务默认按此执行、
无需再次点名「并行使用子代理」。

1. **开工先拆线**：任务含 ≥2 条互不相交的线时，先列出「线 × 文件足迹」再动工；
   2-4 条独立线用并发子代理分头执行（运行环境 4 并发槽位），主代理负责定契约、
   跑全量门禁、集成收口——v4.54-v4.59 的「三线并行 + 主代理收口」即标准形。
2. **单线也倾向派活**：一条独立成刀的任务（调研/实现/测试/文档）若体量超过一轮，
   优先派子代理并发执行，而不是排队串行。
3. **足迹互斥铁律**：线间文件足迹不相交；契约/生成类文件（types.ts / bridge.ts /
   mock.ts / 三语字典 / gen_bindings 产物）必须指定单一负责人，生成动作由主代理在
   所有后端子代理完成后统一执行，防止并发写覆盖。
4. **每刀回写**：刀末把本次分线/收口经验（含教训）写回本文件与 `.gaea/progress.md`，
   让习惯持续强化；若某刀必须串行，收口时说明原因。

## 交互纪律：不许用「等待」换时间（2026-09-10 用户拍板）

用户原话（对上一轮执行的批评）：「后台运行的东西你为什么要等待」「在浪费我的时间」。
耗时本身不可怕，**干等与重复**才是浪费。后续每一轮工作按下办：

1. **后台任务不阻塞**：起后台任务后立刻去做别的事（改码/读文档/跑定向用例/写文档），
   收到完成通知再取结果。**禁止起完就守着等**——那几分钟是白扔的。
2. **重活先报 ETA 再跑**：实测耗时——全量 vitest ≈ **140s**（324 文件 2799 例）、
   `scripts/ci.ps1` 全门禁 ≈ **5–7 分钟**（Go 全量 + 前端 lint/build/vitest）、
   `go test ./...` ≈ **2–4 分钟**。开跑前一句话说清「跑什么、约多久」，用户可否决。
3. **定向优先，全量收尾**：改动后先跑目标文件（3–10s）确认；全量只在交付前跑一次。
   一轮改动 = 中间定向 + 收尾全量，**同一验证不重复跑**（上一轮全量跑了 4 遍=反例）。
4. **超时上限按实际需要写**：不要写 900s 这种夸张数字——那是上限不是耗时，
   只会造成「要卡你十几分钟」的观感。
5. **不把「等待」当进度表达**：进度由已完成的具体产出说话，不由轮询次数说话。
   用户催问时先答「在跑什么、还要多久」，再继续。
6. **能并行就并行**：长任务是可并行的（截图取证 / 读码 / 单测互不依赖），
   串行排队本身就是一种浪费。

> 与上一节的关系：并发是**手段**（把时间省下来），本节是**约束**（别把省下的时间又等回去）。

## 工装：CDP 走查与前端调试（2026-09-10 增补）

- `scripts/cdp-walk.mjs` 支持 **`--target <url 片段>`**（多标签时选定目标页，缺省取第一个
  page）与 **`@文件路径` 传入 JS 表达式**（PowerShell 5.1 传原生命令参数会吃掉内层引号，
  长表达式一律写文件再 `@` 传入）。真机走查配方：无头 Edge
  `--headless=new --remote-debugging-port=9333 --window-size=1440,920 --user-data-dir=<tmp>`
  + Vite dev（`?mock=1`）→ 本脚本 eval/截图。
- **Vite dev 缓存坑**：同一 CSS/源文件在**同一秒内的两次编辑**，mtime 粒度相同会让
  HMR/转换缓存认不出改动（表现为浏览器仍跑旧样式，reload 也无用）。处置=改完
  `(Get-Item file).LastWriteTime = Get-Date` 再 reload。
- **`vitest` worker 上限**：`maxWorkers` 必须按**物理核**（≈逻辑核/2，夹 [4,8]）而不是
  逻辑核——超配会把单用例墙钟拉到 CPU 时间的数倍，重组件首个 `await import` 直接
  撞 `testTimeout` 假红（详见 `frontend/vite.config.ts` 注释）。

## 长期规划（权威，2026 定稿）

- **下一阶段规划 = `docs/gaea-next-stage-plan-2026-09.md`（2026-09-10 活跃指导）**：接棒长期规划（阶段一~四已收官）——阶段五 记忆 OS+上下文编译（主轴）/阶段六 书斋纵深——办公主角（可审计默认化·记忆驱动项目本体·多文件 DAG），进度（DCMA 体检·蒙特卡洛工期带）与跨域 EVM 为两翼/板块并行池/拍板池（信息价接入·MCP 翻案·壳外 computer use）/维持轨。提方向前先对其 §0 在册边界（GoalCard v3.6.0 撤下、MCP/平台化不进规划、不做算量、DSH 独立窗口等）。
- **用户拍板（2026-09-08，收敛计划 §0）：性质路线=A「终极个人工具」**——产品化
  降为期权不作承诺，不为想象中的用户写代码；**LICENSE=私有 All Rights Reserved**
  （根目录 LICENSE），未来开源须另行发布开源许可证覆盖对应模块并与私有部分区隔。
- **瘦身长期总规划（2026-09-08 立项）= `docs/gaea-slim-masterplan-2026-09.md`**：七面
  （认知/资产/结构/轮子/运行/产物/数据知识）×六阶段（P0 基线→P5 维持）；治理规则仍在
  收敛计划。防复发规约自 P1 起生效（新下载走 saveExportBlob/新选取走 pickFile/新 diff
  复用 lib/diff/新 slug 用 strutil.TitleSlug/新解码用 b64ToBytes/NAVIGATE 用 manifest id/
  新域先问「能否是文档」）。
- **用户再拍板（2026-09-08）：工位与乐园并列平级**——办公不是乐园的上位核心，瘦身
  不是「收乐园保办公」；认知面问题=13 板块平铺单一导航面、并列结构未表达。P2 措辞
  已由「乐园折叠」改为「双空间并列落地」，乐园板块在乐园空间内一级可见。
- **唯一权威路线图 = `docs/gaea-nextgen-roadmap-2026.md`**（11 个子代理调研合成）：
  8 板块竞品调研（办公/造价/AI 底座/编程/小说/绘梦/轻语/微信+语音）+ WorkBuddy×灵犀
  深度拆解（§12）+ **版本重定义"双空间"（§10）** + 四层落地（§13 后端/前端/UX/UI）+
  执行计划（§14 阶段 0 地基 → 阶段 1 双空间内核 → 阶段 2 双空间壳 → 阶段 3+ 领域包）。
- **用户拍板：工作与娱乐分开、互不干扰**——工位（办公/造价/编程/资料+工作记忆）与
  乐园（轻语/小说/绘梦/阅读）双空间硬隔离；记忆分区互不检索、模型策略各配各的、
  上下文永不跨界；跨空间仅用户显式发起（如"把乐园封面放进报告"）。
  ~~"陪伴×办公融合"（旧 v4.3）已删除~~。
- **本轮关键纠正**：灵犀 = 金山 WPS 独立 AI 办公 Agent（非阿里/通义系）；
  WorkBuddy = 腾讯云 CodeBuddy 全场景 AI 办公工作台（非 Kimi 系；Kimi Work 是月之暗面的）。
- **i18n 决策（2026 追加）**：采用审计 §405「诚实 zh-only」选项——**壳层 chrome +
  设置外壳三语**（已完成，消灭壳层混合语言根因）；**页面内容层保持 zh 单语**，
  不再逐页铺 en/zh-TW 字典（个人中文工具无国际化受众，~5000 字符边际价值≈0；
  未来需国际化时按 S2.3b WireShape 模式整页迁移）。
- **文档纪律**：docs/ 旧调研/已落地计划已归档至 `docs/archive/`（见其 README）；
  后续会话以本文件 + 长期规划 + `.gaea/progress.md` 为权威，勿引用 docs/archive 结论。
  **新文档必须登记 `docs/README.md`**——守卫 `node scripts/check-docs.mjs` 四查（孤儿登记 /
  `docs/` 悬空引用 / **本文件字节预算 ≤ 65536 B**，超了尾部纪律段就对会话不可见 /
  **含非 ASCII 的 .ps1 必须带 UTF-8 BOM**——2026-09-10 实撞：编辑 ci.ps1 丢 BOM，整条门禁语法错静默不跑），已接入 `scripts/ci.ps1`。
- **执行审计（2026-08-30）**：`docs/archive/audit-2026-08-30-v4-execution-review.md` 记录
  v4.x 全量「承诺 vs 代码」对照——裁决=最小版执行（骨架真、纵深欠账）。红线缺口
  三条（记忆注入跨空间未接线 / 任务分账未启用 / 事件过滤仅 1 处）与补课刀序见该文
  §B/§E；后续每刀验收新增「纵深检查」，发布说明必须列欠账清单。
- **执行状态（v4.8.0 后）**：审计欠账大面收账——读屏纵深（多显示器/OCR 本地
  摘要/截图留档）、intent LLM 兜底分类器（默认关，白名单+置信门+硬超时）、
  生图产物 CardPath 接通、iLink 微信通道离线收敛（限频/下载防线/识别管线/
  防御解析/SendFileCard seam）、全局离线模式总开关（EngineType.IsLocal +
  routeModel 云过滤）、成本知识图谱可视化（BuildGraph + CostGraphView 第 8
  模块，绑定面 533）、实时语音 Realtime S0 铺底（internal/realtime seam）。
  剩余欠账（Realtime S1/S2、iLink 真机窗口、离线模式设置 UI、权限升级请求+
  stubGate 竞态、XlsxPreview 虚拟滚动/生命库可写化=观察项）见
  `releases/v4.8.0.md` 欠账清单。
- **欠账收尾小步（2026-08-30，v4.8.3 后）**：VoiceStart realtime 门小修
  （端到端回复走服务端 response 事件，whisperChatFn=nil 也可启动，拼接
  管线双门逐字节保留）；持久化套件统一（desktop_session 原子写 + archive
  JSONL 单次 Write 落整行）；XlsxPreview 大表格行虚拟滚动（观察项收账，
  300 行以上只渲染可见窗口 ±overscan）。Go 全量绿、vitest 809/809、
  tsc/eslint 0、绑定面 535 不变。
- **下一执行**：v4.8.3 已发布（微信图片双向真协议）；剩余——Realtime 真机
  验证轮（真 key 下端到端对话/打断体感/AEC 实效，S2 骨架已就绪待真机数据）；
  手写体识别质量复测（多模态 Qwen 升级后）；iLink 语音/视频等未探明 item
  维持宁漏勿误静默跳过；生命库可写化=观察项。

## 版本状态（历史存档）

> 2026-09-09 整理：本段逐版历史（v2.x~v4.135，约 2100 行）整体迁往 `docs/archive/agents-version-history-2026-09.md`，按需检索；当前动态见顶部速览，发布物全文见 releases/。

## 项目定位

gaea 是 Windows 桌面端「通用办公」AI 助手（Wails v2：Go 1.26 后端 + React/TypeScript/Vite 前端）。
核心能力：文档撰写、表格处理、格式转换（docx/xlsx/pdf → Markdown）、图表生成、报告拼装、
知识库与记忆中枢、方案编写。品牌定位已从「土壤修复工程办公」全面转为「通用办公」。

## 技术栈与关键约定

- 桌面框架 Wails v2.13（Go + WebView2）；后端事件总线 + 前端 zustand 桥接（bridge.ts → window.go.app.App）
- **绑定面（v2.17.0 起）**：App 不再直接绑定 Wails；429 个导出方法拆 10 个板块门面
  （internal/app/bindings_*.go：CoreB/OfficeB/MemoryB/CostB/ModelB/VoiceB/ChatB/NovelB/ImageB/CharlibB，
  纯委托零逻辑改动）。改绑定面方法后用 `go run ./scripts/gen_bindings` 重新生成 +
  `TestBindingsCompleteness` 兜底；前端调用经 gaea/lib/bridge.ts（按方法名路由门面）或
  api/bridge.ts 的 window.go.app.App 兼容代理；旧 wailsjs 导入走 src/wailsjsCompat 重导出；
  wails build 会重生成 wailsjs/go/app/<门面>.js
- 单模型架构：一个 executor 完成规划与执行，无独立规划器；任务/技能子代理走 `task` / `run_skill`
- 内置工具精简为 17 个核心工具（v2.4.3 起）：文件/命令、网络、任务、记忆/知识、技能、format_convert、chart_gen
- 文档能力交给 ModelScope 技能：docx / pdf / xlsx（安装在 `~/.codex/skills` 与 `.gaea/skills`），
  转换引擎共用 `internal/office/docmd`（format_convert 工具与预览面板同一实现）
- 内置子代理技能：format-convert / chart-builder / doc-assemble
- 记忆系统：SQLite（`%APPDATA%\gaea\Hephaestus.db` facts 表，按项目 slug 隔离）+ 文档记忆（AGENTS.md 层级）
- 环境依赖：LibreOffice（soffice）、node 全局 docx、Python 3.13（lxml/openpyxl/pypdf/pdfplumber/reportlab/pandas/matplotlib 等）
- 本地 AI 底座：**Herdsman**（localhost:8080/v1，~110GB 模型：35B 对话 ×2、zimage-turbo、voxcpm2、
  mineru、embedding/reranker、paddleocr、sherpa-onnx 等）；gaea 的聊天/视觉/检索/OCR/解析/ASR/TTS/生图/翻译
  全部依赖它，herdsman 升级可能破坏契约——用 App.HerdsmanProbe 启动探测

## 发布流程（2026-08-14 修订：补版本资源步骤）

1. 更新 CHANGELOG.md / README.md（版本表）/ wails.json（productVersion）/ releases/README.md（版本表）
2. **同步版本资源**：`build/windows/info.json` 是 Wails 生成版本信息的模板（fixed 段必须含
   `product_version`，否则 exe 的 ProductVersion 为 0.0.0.0）；根目录 `versioninfo.rc` 是遗留物，一并更新以免误导
3. 构建（本沙箱：`cd frontend; npm run build` → `wails build -s`；本机：`cmd /c build.bat`），
   产物 build/bin/gaea.exe（同时复制到桌面）；本机 build.bat 已内置真实退出码检查 +
   默认自动冒烟（.tmp 临时副本 → scripts/smoke.ps1，18999 /api/health 200，失败即停；
   `build.bat skip-smoke` 可跳过，发布前不得跳过）
4. 复制 exe 到 `releases/gaea-v<版本>.exe`，生成 `releases/SHA256SUMS-v<版本>.txt`；
   **本地产物只保留最近 5 版**（2026-09-10 用户拍板）——发版后删掉第 6 新的那一版，
   更早版本的身份以 `SHA256SUMS-vX.Y.Z.txt` 为准（旧 exe 不入库）
5. 写 `releases/v<版本>.md` 发布说明（含 SHA256 与冒烟结果），更新 releases/README.md 版本表
6. 冒烟：`scripts/smoke.ps1 -ExePath releases\gaea-v<版本>.exe`（/api/health 200 即通过）
7. 更新 `.gaea/progress.md` 进度记忆与本文件（版本状态）

## 沙箱环境备忘（2026-08-14 整理，详细版见 docs/2026-08-14-sandbox-environment-notes.md）

**防止重蹈覆辙的四条铁律**：
1. `go telemetry off` 已持久生效；构建缓存写入问题随 danger-full-access 策略解除，无需再覆盖 GOCACHE
2. **wails build 前端编译会挂起**（wails 捕获前端输出走管道）——必须 `cd frontend && npm run build`
   再 `wails build -s`（-s = 跳过前端编译，9s 完成）
3. `go test ./...` 单进程树会被 harness 终止、个别测试二进制偶发 `fork/exec Access is denied`——
   逐包验证 + `scripts/test-all.ps1`（逐包/重试/状态续跑）；exec 拒绝用 `go test -c` 手动运行证明代码无恙
4. .ps1 脚本必须 UTF-8 带 BOM（否则 powershell.exe 按 GBK 解析报错）；npx 用 `& 'C:\Program Files\nodejs\npx.cmd'`

## 本地 TTS 引擎（重要记忆，勿遗忘；2026-08-09 整理）

> ⚠️ **VoxCPM2 已于 v2.6.9 移除**：实测不达标（耗时长、音色男女混乱、克隆不稳定）。
> 下方 VoxCPM/Vulkan 相关方法保留为「已废弃教训」，勿重新安装；当前本地 TTS 为 CosyVoice2。
> 注：herdsman 侧实测 voxcpm2 可用（冷启动约 50s，不支持预设音色），qwen3-tts-* 未安装。

本机（Radeon 8060S 核显 / 64GB 统一内存 / Windows）本地 TTS 有两条引擎线，gaea 只连 OpenAI 兼容 8020/8010。

### ~~VoxCPM2~~（已移除 v2.6.9，以下为废弃记录）

- `8030`：主后端 `C:\AI\llama-omni\build\bin\llama-tts-server.exe`（llama.cpp-omni，C++/ggml + Vulkan）
  - 模型：`C:\AI\llama-omni\models\VoxCPM2-BaseLM-Q8_0.gguf`（1.65GB）+ `VoxCPM2-Acoustic-F16.gguf`（1.74GB）
  - 8060S 识别 `KHR_coopmat + bf16`，全量 29 层 offload Vulkan0，加载约 2s
- `8021`：备胎 ROCm PyTorch（`C:\AI\voxcpm\server.py` + `VOXCPM_PORT=8021`）
- `8020`：适配层 `C:\AI\voxcpm\adapter.py`（FastAPI，gaea 入口，契约不变）
- 一键启动：`C:\AI\voxcpm\start_voxcpm_stack.ps1`（8030/8021/8020）

### CosyVoice2（端口 8010）

- `C:\AI\cosyvoice\server.py`：LLM 段 GGUF + Vulkan（`gguf\cosyvoice_f16.gguf`），flow 段 ONNX + DirectML（5 步）
- 启动：`C:\AI\cosyvoice\start_cosyvoice.bat`；约 14s 加载+预热，短句 ~1.5s

### 音色（两引擎统一 4 个，火山引擎 Speech-AI-Forge-spks 录音室样本）

- 中文女 `zh_female.wav`（f0≈221Hz）、中文男 `zh_male.wav`（f0≈133Hz）
- 英文女 `en_female.wav`（f0≈191Hz）、英文男 `en_male.wav`（f0≈109Hz）
- 参考音频 ~7s / 16kHz；转写在 `C:\AI\voxcpm\voices\_meta.json`

### 本次优化方法（AMD 核显提速教训，勿重蹈覆辙）

1. 不要再用纯 ROCm PyTorch 追赶速度：iGPU 共享内存架构下 ROCm 与 CPU 基本相同（RTF ≈1.06~1.12）；
   Vulkan + ggml 的 GEMM/coopmat 才是突破口（克隆 RTF 0.65~0.84）
2. 构建：MSYS2 UCRT64，`cmake -B build -DGGML_VULKAN=ON -DGGML_NATIVE=ON`
3. 坑 1（端口绑不上）：server-voxcpm2 会构造 SSLServer，空证书导致 is_valid_=false 任何端口 bind 失败；
   本地回环不需 TLS，改普通 httplib::Server
4. 坑 2（克隆近静音）：AudioVAE 参数特征必须 frame-major（`ggml_cont(latent)`），
   不能 `cont(transpose(latent))`（dim-major）
5. 坑 3：llama.cpp-omni 的 CLI `-r` 克隆偶发偏静音，HTTP server 路径正常；生产走 server
6. 坑 4：VoxCPM Python 长文本 CFG 2.0 会「静音+整段重试」（RTF 4.8~7.8），CFG 1.5 稳定；
   C++ server 端用 max_steps 限制解码上限
7. 网络：HuggingFace LFS 直连/hf-mirror 都不通，ModelScope 直链快（8.6MB/s）
8. 实测：短句克隆 RTF 0.65~0.84（6 步）、语音设计 0.57~0.60；同 seed 输出确定

### 详细记录

- `docs/archive/2026-08-09-voxcpm2-integration.md`（VoxCPM2 全部历程）
- `docs/archive/2026-08-09-cosyvoice2-llm-gguf-speed-optimization.md`（CosyVoice GGUF 提速）

### 自动启动（当前仅 CosyVoice2）

- gaea 启动时后端 ensure cosyvoice；模型中心 TTS 模型卡片「启动」按钮 → `App.StartLocalTTSService(engineId)`；
  引擎连接测试兜底 ensure（等约 8s）；TTS 合成前兜底 ensure
- 实现：`internal/app/tts_service.go`（core.ensureLocalTTSService 幂等 + 异步轮询，emit `tts-service-status`；
  CosyVoice 直接 python server.py，隐藏窗口 CREATE_NO_WINDOW）
- 端口探测：CosyVoice2 `8010/v1/models`

## 已知注意

- 角色库剧照默认跟随绘梦（ImageBackend/ImageModel），可在模型中心单独绑定
- 文生视频依赖本地 ComfyUI 安装 LTX-Video 模型
- 里程碑：2026-08-12 完成通用办公全面打磨（显示/布局/安全三线：成本库/记忆/知识库/技能写入全部硬性确认
  含 yolo、子代理路径；子代理不再继承持久化写入工具）
