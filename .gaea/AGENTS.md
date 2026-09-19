# gaea 项目记忆

> 本文件为项目长期记忆（文档记忆层级）。编码规范：**UTF-8 无 BOM**（历史遗留的 GBK/UTF-8 混合编码已清理）。
> 修改后请保持 UTF-8；.ps1 脚本需 UTF-8 带 BOM（见「沙箱环境备忘」）。

## 版本状态（顶部速览）

> 速览只留最近 3 版（2026-09-15 整段分流：水位根治，此前 14 版口径废止——10 条迁 archive 段首）；更早版本全部见 `docs/archive/agents-version-history-2026-09.md`（v4.49 及更早、v4.50–v4.145 仅 CHANGELOG/releases 有记录）。
- **非版本刀（2026-09-19）todos 对账清账 + booksource 进度回调乱序根修**——起因=「TaskCenter 会话维度」行标 ⬜ 实际 v4.229.0 已落（SchemaV20+SubmitSpaceSession+过滤面 chip 熄灯）险些重做；子代理全表对照 git log/CHANGELOG 核查：**关 3 行**（TaskCenter/审计刀A v4.165.0/knip v4.268.0 清账）+**改写 2 子句**（t4-C4 已随 t7 收官〔面板只读高亮+锚点 v4.325.0〕编辑器 overlay=有意裁剪入观察池；造价 §6=v4.209.0 已收官〔基线源=自有库〕）+删 7.3-1 旧欠账注记（v4.333.0 已收口）+补记 7.3-2 缺省翻转候拍板；**过时源曾传染 v4.338 未做段——排刀前先对账，勿按旧未做段排刀**。**根修**=booksource fetchChapters OnProgress 锁外发射乱序（全量 ci 实测 [0/3 2/3 1/3]，负载 flaky 家族+1）且对消费方并发调用——发射互斥下现读计数：序列非降/末次=成功总数/消费方免同步，契约注释补「轻量回调」纪律；-count=10 绿，本机无 gcc -race 不可跑（race 门=Actions）。**教训**=进度类回调的发射序是契约一部分——计数加锁≠发射有序；todos「已落地未关」会传导进 release notes。
- **非版本刀（2026-09-19）真机走查班：v4.337~v4.343 新面清池（用户在场背书，只读纪律）**——CDP 9333 附着打包壳（v4.343.0 exe）：**双空间 13 页巡检全绿**（书斋 6+闲庭 7，错误边界 0/空白 0/console error 0/exception 0）。**夹具实数据深验**（.tmp/makefixture-v343 造 12 章+大纲+前 3 章 analysis-v2+第 1 章两标注，走查后删净双复核）：①章际对比差值表逐字精确（7.8 vs 6.0 Δ-1.8，四维+情感强度）②**定位镜像高亮真机完美**——「夜风从窗缝里钻进来」青色 mark 精确罩住正文对应字串，真实字体/滚动条下零偏移（v4.343 最险面过关）③情感曲线 3 点+tooltip 逐字精确④体检/分析空态诚实。**坑**=①书架工程清单在 C:\AI\xiaoshuo（novelsDir 设置），夹具造在仓库 novels/ 不上墙②工程列表启动时读，壳须重启重扫③CreatePage activeChapterNum 有 lastMainChapter 兜底——分析面板章号≠树选中章，夹具驱动须先点树行（内层 div [title*=摘要] 才是点击目标，外层容器无 handler）④嵌套模板字面量里 \d 会被外层吞掉弄坏 eval 正则——CDP 探针用 [0-9] 或 DOM 直读。**清场**=杀壳→删 C:\AI\xiaoshuo\对话流走查工程→ls 双复核（用户真实工程零触碰）。**剩余挂池**=pptx 刀2/mspdi 样本类深走查（需数据）、持久全量 overlay（真机镜像对齐已过，可议）。
- **最新发布：v4.355.0（2026-09-19）「真机走查班：UnifiedSearch 参数序错位根修」**——前端优化六连发后按惯例开走查班（CDP 9333 附着壳 walk-v4354 配方）：双空间 13 页巡检 0 崩 0 空白 0 exception；**拽出月级 bug**=UnifiedSearch 三调用方（记忆中枢三脑检索/工作区搜索面板/Ctrl+K）参数序错位——Go GaeaUnifiedSearch(query, topN int, scope ...string) 而前端传 (query, scope, topN)，scope 字符串落 int 形参每次 unmarshal 失败，自 v2.23 真壳上一直坏；捂因=mock 按前端期望写契约全绿+bindings 测试只收方法名+前端吞错静默（v4.350 失败可见化「检索失败，请重试」一次走查拽出）。修复=bridge 签名 (query, topN?, scope?)+三调用方换序+mock 同序+三测试断言换序（零 Go 改动 706 不动）；新壳复验确证=错误变 embedding 请求失败（本机 8080 语义服务未部署属环境挂观察池），三态如实呈现。其余发现=file:// 头像 WebView2 拒载 27 条（既有挂池）/longtask 69~165ms 误报挂池。门禁=tsc 0+eslint 0/0+全量 ci 绿 3295 例+drift OK@706+版本三处 4.355.0。坑=①mock 契约必须从 Go 绑定复制签名含参数序（按前端写=错位时 mock 全绿真壳全坏）②静默吞错捂 bug 时长与可见性成反比，失败可见化是最便宜的测试③探针按钮定位优先类名 testid。产物=exe 50877440B SHA256=b012287b…（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略删 v4.349。文档=releases/v4.355.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（七十迁：v4.352 入 archive）+progress/todos。
- **最新发布：v4.354.0（2026-09-19）「前端优化第六轮：P0 数据形状崩溃×2 / 异步竞态 11 处 / 资源生命周期」**——用户口径「继续」，三路并行只读审计（异步竞态/数据形状防御/资源生命周期第二层）按证据定刀，纯前端 17 源码+1 测试，零绑定 706。**P0×2**（后端超集形状+mock 不可复现同型）：①伏笔面板 STATUS_META 3 态 vs Go 6 态并集，「项目重置」置 pending 重拉即整页崩（一键自毁式）→类型 6 态+STATUS_META 补全+statusMetaOf 防御+流转 6 态文案，测试+2；②MemoryPanel skills=null（Go 规则类记忆<2 条 return nil 无 omitempty，mock 恒 []）轻量用户点扫描即崩→setSuggestions 入口统一归一。**竞态 11 处**（统一 seq/loadToken 已有范式）：PromptWorkshop 模板切换（A 正文覆盖 B 模板保存即数据损坏）/BookSearchModal（错章节范围导入）/createTopic+清空对话清错对象（新增 invalidateLoads）/useSinStory（切故事串台）/ChapterPage.handleSave（保存期间打字被强制 saved=true 静默丢稿→快照比对）/SearchModal（Enter 双发+旧意图卡可执行）/MemoryHub runSearch/切后端/outlineStore（错书章节读写）/Weixin toggle（连点两条相反 Save）/TTSPlayer 卸载泄漏外放。辨伪 18+。门禁=tsc 0 --force+eslint 0/0+全量 ci 绿 3295 例+drift OK@706+版本三处 4.354.0。坑=①Go nil slice→JSON null mock 不复现，收窄层归一优于逐点防御②saved 语义=缓冲==磁盘，异步完成侧快照比对③Input.Search 回车已触发 onSearch 双挂=双发④竞态修复走已有范式。产物=exe 50877440B SHA256=758eba83…（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略删 v4.348。文档=releases/v4.354.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（六十九迁：v4.351 入 archive）+progress/todos。
- **最新发布：v4.353.0（2026-09-19）「前端优化第五轮：ResizableDrawer 弹层可访问性收口 / 可点 div 键盘化余量」**——用户口径「继续」（v4.352 池延续），纯前端 9 源码+1 测试，零绑定 706。①ResizableDrawer 三缺补齐（ApprovalModal 先例）：role=dialog+aria-modal+aria-label（新 prop，两调用方传 i18n 标题）/Esc 关闭（可编辑目标不劫持）/焦点管理（开记录+聚焦 tabIndex=-1，卸载还原 try 兜底）；测试新建 4 例。②键盘化余量：ChapterTreePanel 节点/OutlinePanel 大纲行/StatsPanel 两折叠→role=button+tabIndex+Enter/Space；ChapterEditor+ChapterTreePanel 五 icon Button 补 aria-label；FiveCalcPanel 遮罩补 Esc；辨伪 RailItem/FactCard 已是 button（清单是快照非现状）。门禁=tsc 0 --force+eslint 0/0+全量 ci 绿 3293 例+drift OK@706+版本三处 4.353.0。坑=①effect 放依赖声明后（依赖数组渲染期即时求值，插前面 TDZ ReferenceError）②React 合成 KeyboardEvent 无 isContentEditable③审计清单逐点核对再动手。产物=exe 50875904B SHA256=2f463362…（releases+SUMS 仅本地；桌面副本同哈希；冒烟 200 过）；保留策略删 v4.347。文档=releases/v4.353.0.md+CHANGELOG/README+AGENTS 迁 1 插 1（六十八迁：v4.350 入 archive）+progress/todos。
 **非版本刀（2026-09-13）oh-story 蒸馏 T3：生成门补确定性两路（写前大纲契约 + 写后质量体检）**——续 T1（评审 rubric 资产）与 T2（去 AI 味门禁资产），本刀把上游 hooks 的「写前守契约、写后查质量」补进内核（规格 §2 真增量第 1 条：gaea 此前只有事后去味与体检，**无生成前结构契约闸、无生成后确定性质量闸**）。**落地**=①新包 `internal/novelgate`（零 LLM 零网络纯函数）：`OutlineContractIssues`（标题/本章计划/关键要点/情感基调四项齐备性，缺席 S2~S4）+ `ChapterQualityIssues`（空正文 S1 / 超长段落 S3 带行号 / **电报体** S2〔短句占比 >40% 且平均句长 <10 字——把逗号长句拆碎同属 AI 味〕/ 平均句长偏短 S3 / 连续堆叠问号感叹号 S3〔RE2 无反向引用，用「同类标点 ≥3 连」表达〕/ 省略号滥用 S3 / 通篇句号化 S3）；`Issue{Code,Severity,Message,Evidence}` 与评审 rubric 的 S1-S4 同轴、必带证据；阈值常量集中、口径一律 rune。②接线=`RunChapterGate`（章节生成门）新增确定性两路 `outlineContract` + `deterministic`，与 AI 四路（analysis/review/consistency/aiTaste）并列：必出、零成本、失败不阻断（找不到大纲节点返回空表）。**测试**=Go +7（契约齐备与空项分级/空正文 S1/正常文本零误报/电报体/超长段落/标点堆砌与省略号/句号化/证据随行）。**门禁**=本刀自身证据（go build/vet exit 0 + internal/novelgate 7 例 + 生成门定向用例绿）；全量 ci.ps1 未取全绿——**并行线在制品** internal/app/novel_review_handler_test.go:117 为红（与本刀无关）；drift OK@648；**未抬版本**（随下次发版入 exe）。**未做（下刀）**=T4 导入结构映射与篇幅路由；写前契约硬闸（需先有「一键补大纲」）；书级白名单 .deslop-whitelist 与 T2 gates.json 的内核消费。
- **AGENTS.md 八迁分流（2026-09-13，非版本刀，纯文档）**——CI 卫生守卫 WARN（59926B 逼近预算）触发：v4.261.0~v4.255.0 六条入 archive（八迁记录更新），主文件恢复 14 版，水位 59926→47518B；迁移完整性三项断言全过。坑=CRLF 行尾下 node indexOf 锚点不带行尾+写入前验 includes。
- **对比度普查收账（2026-09-13，非版本刀，零代码）**：13 页×明暗两态 WCAG 程序化扫描（.tmp/walk-v4274-contrast.mjs）——dark 15/light 63 告警逐类甄别后**大头为脚本误报**（渐变/图片背景无法合成，目检实际清晰）；真实低对比仅亮态 accent 弱化文本（~10 处 1.5~1.8）+schedule 行号 1.3+weixin purple tag 3.39——全为装饰性文本非正文，**无 AA 硬伤不动**。观察池新增=亮态 accent 弱化文本打磨候选（修则需动 lightFn 令牌，视觉拍板项）。**坑=对比度自动扫描对 background-image/渐变必然误报，告警须逐类目检甄别后才能定刀**。
- **v4.270 留池补验收官（2026-09-13，非版本刀，零代码）**：sin_illustrate live 端到端**全通**——模型真调工具/图片真实落盘（sin/art「雨夜回眸」.png 1.8MB）/轨迹 artifacts 在位/`extra.illustrations['tool0']` 回写画廊可见/过程卡「思考过程·319 字·生成插图」元数据在位。**历史两次失败归因翻案**=上游 grok-4.6 一次性空返回（仅 reasoning 无正文无工具），后端兜底如实报错（sin_handler.go 空正文不落库），重试即成——**非 gaea 缺陷**。清场=故事删+产物图删（壳句柄锁删挂起，杀壳后 PowerShell 删成）。**坑**=①node rmSync 对被占用文件不抛错但删不动（Windows delete-pending），删后必须 existsSync 复核 ②走查脚本清场要放 finally（v4.270 脚本超时 throw 路径跳过清场）。**观察池新增**=sin/art 存 3 张疑似历史走查孤儿图（00:36/05:22/09:34「雨夜站台」走查 prompt 产物），待人工确认删除。

- **品牌资产补齐（2026-09-10，非版本刀）**——2026-09-10 15:20 的「品牌资产刷新」只覆盖 **4 个 SVG**（`build/appicon.svg` / `frontend/public/favicon.svg` / `gaea/assets/logo.svg` / `logo-light.svg`，新视觉=「地核 G」轨道环抱活核），**两个二进制件没跟上**：`build/appicon.png`（1024²）与 `build/windows/icon.ico` 停在 **2026-08-05** 旧视觉（翡翠球体+破土嫩芽+星芒，深蓝底 `#0F172A`）→ **exe 内嵌图标/任务栏/资源管理器/桌面快捷方式全显示旧 logo，只有窗口内 UI 是新 logo（「换了一半」）**。**落地**=① 无头 Edge 渲染 `appicon.svg` →1024×1024 RGBA PNG（**必须传 `--default-background-color=00000000`**，否则圆角外合成不透明白、alpha 静默丢失）；② Pillow 由同一母图出 **7 档 ICO**（16/24/32/48/64/128/256，旧资产缺 24，本次补 Windows 标准全集）；③ `wails build -s -ldflags "-s -w" -trimpath`（**13.5s**）。**验证三条独立证据**=① 几何精度（核心圆 r=50@512 实测直径 **200px**、圆心 (539.5,511.5) 对理论 (540,512)；环 stroke 48 实测线宽 **96px**；圆角 rx=104 对角线首不透明像素 d=**61** 对理论 60.9——零缩放零偏移）；② exe 内嵌图标提取（`ExtractAssociatedIcon`：底板 `#0B1210`/米色核/翡翠环=新「地核 G」）；③ dist bundle 新 logo 独有标记 `gaea-logo-ring`×3、`gaea-logo-light-ring`×3、`0B1210`×2 在册且旧独占色 `#6ee7b7`/`#047857` **零命中**+按旧 SHA256 全仓比对零旧 logo 残留。**门禁**=build exit 0 / 冒烟 200 / **纯资产刀（零前端零 Go 源码改动、绑定面不变）**。**产物**=`build\bin\gaea.exe` **48,572,928 B** SHA256=**5F811126131A21456D7C84B6D568EC2F0B4A748D17F60AD2814879C795FD4264**（桌面副本同哈希；中间态 903F9759…C36EC068=只换 PNG+主图 ICO 的一版，小尺寸优化后重建覆盖；原始 59C2B518… 系 v4.208.0 归档值，releases 档案自洽未动）。**坑**=① 无头 Edge 截图默认白底，CSS 里写 `background:transparent` 无效（截图不继承页面背景），透明基线拿旧资产角像素 alpha=0 对照；② **判新旧前须验该色是否新旧共有**——`#34D399` 同时存在于**新** logo 的 halo 渐变与**旧** logo 主色，拿它判会把新资产误报成旧的，判据只能用**独有**标记（SVG id 名 / 独占色）；③ `pwsh` 不在 PATH，手工跑 `scripts/smoke.ps1` 会 `CommandNotFoundException`（**非 app 故障，易误判为冒烟失败**），用 `powershell`（build.bat 内已有 fallback）；④ 运行中的 exe 可被 `Copy-Item` 直接覆盖（进程持旧映射），桌面副本无需先关 app，但**已加载实例不会换图标，需重启**。**小尺寸可辨识（同日追加）**=原方案对大图统一降采样，致 16×16 环宽仅 1.5px / 核 3.1px、G 字形糊成深色块；新增派生资产 `build/appicon-small.svg`（环 48→64、核 r50→62、去 halo），ICO **按尺寸分流源图：16/24/32 用简化变体、48 及以上用主图**——衔接依据=变体 32px 环宽 **4.0px** ≈ 主图 48px 环宽 **4.5px**（若在 24→32 切会跳）；**坑=Pillow 的 ICO writer 只接受单一源图**，混合源尺寸必须手工组装 ICO 容器（ICONDIR 6B + 逐帧 16B + PNG 帧，**256 档宽/高字段写 0**，写 256 溢出单字节）。ICO=7 档 44384 B SHA256=**775CAED3274EC199DF7F3E188190BFCDB3A319EDD01EF2A350699FC8D6E2BD02**。**未抬版本**——按 README 发版约定「文档整理、令牌对照、单点样式等小改记入 CHANGELOG，不单独抬版本」，本刀属视觉资产补齐，与 `progress.md` 的「工作空间与文档整理（非版本刀）」同类处置。

- **壳内全板块渲染健康巡检（2026-09-12，非版本刀，随 v4.237 线收尾）**——CDP 起壳逐 rail 页点击+错误边界断言：闲庭七页（闲庭首页/首页/聊天/小说/绘梦/模型中心/角色库）+书斋六页（首页/办公/造价数据库/记忆中枢/模型中心/青鸟）**13 页全绿零接管**；配方=walk-pages.mjs（rail 坐标 DOM 定位+逐页停留+错误文本断言+失败自动截图）；巡检结论=无其他「静默坏死」页面。零代码改动不抬版本。

- **DAG 6.3 壳内真机走查（2026-09-12，非版本刀，60c68a34）**——§6 最后一条余项清账（**§6 余项全清，DAG 6.3 线收官**）：CDP 9333 DOM 断言走查（不点危险操作不打模型）——注入体检 Drawer 真跑=GaeaMemoryEvalRun 真实 wire+真实库**端到端打通**（通过：五条结构不变量全部成立；晨报预载 2 条·104/600、本体 2 引用·181/600、门控四开关、toast；截图 .tmp/eval-drawer.png）；办公流水线区 dag-section 在位（空态引导正确）；全程 exceptionThrown=0/页面渲染出错=0。**观察池新增**=办公记忆库面板「不可用—未配置」与同库体检读出 2 条活跃并存——面板 available 旗与体检取数路径（hubOfficeStore 直连）口径不同，既有语义非本线回归，按需另刀。配方=.tmp/walk-v4243.mjs/.tmp/walk-dag.mjs（rail 悬停展开→按 title 前缀点库入口→按钮文本带计数须前缀匹配；**节点列表/状态徽标在展开层，先点 goal 展开**）。零代码改动不抬版本。

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
