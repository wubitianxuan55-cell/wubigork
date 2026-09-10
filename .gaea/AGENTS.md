# gaea 项目记忆

> 本文件为项目长期记忆（文档记忆层级）。编码规范：**UTF-8 无 BOM**（历史遗留的 GBK/UTF-8 混合编码已清理）。
> 修改后请保持 UTF-8；.ps1 脚本需 UTF-8 带 BOM（见「沙箱环境备忘」）。

## 版本状态（顶部速览）

> 更早版本见 `docs/archive/agents-version-history-2026-09.md`（覆盖 v4.173–v4.146 与 v4.49 及更早）；**v4.50–v4.145 区间本仓只在 `CHANGELOG.md` / `releases/` 有记录**（速览逐版滚出后未回填）。本段只留最近 14 版：本文件曾达 104 KB 超工作区指令预算（65536 B），尾部纪律/规划段一度对后续会话不可见，2026-09-10 二次分流。

- **品牌资产补齐（2026-09-10，非版本刀）**——2026-09-10 15:20 的「品牌资产刷新」只覆盖 **4 个 SVG**（`build/appicon.svg` / `frontend/public/favicon.svg` / `gaea/assets/logo.svg` / `logo-light.svg`，新视觉=「地核 G」轨道环抱活核），**两个二进制件没跟上**：`build/appicon.png`（1024²）与 `build/windows/icon.ico` 停在 **2026-08-05** 旧视觉（翡翠球体+破土嫩芽+星芒，深蓝底 `#0F172A`）→ **exe 内嵌图标/任务栏/资源管理器/桌面快捷方式全显示旧 logo，只有窗口内 UI 是新 logo（「换了一半」）**。**落地**=① 无头 Edge 渲染 `appicon.svg` →1024×1024 RGBA PNG（**必须传 `--default-background-color=00000000`**，否则圆角外合成不透明白、alpha 静默丢失）；② Pillow 由同一母图出 **7 档 ICO**（16/24/32/48/64/128/256，旧资产缺 24，本次补 Windows 标准全集）；③ `wails build -s -ldflags "-s -w" -trimpath`（**13.5s**）。**验证三条独立证据**=① 几何精度（核心圆 r=50@512 实测直径 **200px**、圆心 (539.5,511.5) 对理论 (540,512)；环 stroke 48 实测线宽 **96px**；圆角 rx=104 对角线首不透明像素 d=**61** 对理论 60.9——零缩放零偏移）；② exe 内嵌图标提取（`ExtractAssociatedIcon`：底板 `#0B1210`/米色核/翡翠环=新「地核 G」）；③ dist bundle 新 logo 独有标记 `gaea-logo-ring`×3、`gaea-logo-light-ring`×3、`0B1210`×2 在册且旧独占色 `#6ee7b7`/`#047857` **零命中**+按旧 SHA256 全仓比对零旧 logo 残留。**门禁**=build exit 0 / 冒烟 200 / **纯资产刀（零前端零 Go 源码改动、绑定面不变）**。**产物**=`build\bin\gaea.exe` **48,572,928 B** SHA256=**5F811126131A21456D7C84B6D568EC2F0B4A748D17F60AD2814879C795FD4264**（桌面副本同哈希；中间态 903F9759…C36EC068=只换 PNG+主图 ICO 的一版，小尺寸优化后重建覆盖；原始 59C2B518… 系 v4.208.0 归档值，releases 档案自洽未动）。**坑**=① 无头 Edge 截图默认白底，CSS 里写 `background:transparent` 无效（截图不继承页面背景），透明基线拿旧资产角像素 alpha=0 对照；② **判新旧前须验该色是否新旧共有**——`#34D399` 同时存在于**新** logo 的 halo 渐变与**旧** logo 主色，拿它判会把新资产误报成旧的，判据只能用**独有**标记（SVG id 名 / 独占色）；③ `pwsh` 不在 PATH，手工跑 `scripts/smoke.ps1` 会 `CommandNotFoundException`（**非 app 故障，易误判为冒烟失败**），用 `powershell`（build.bat 内已有 fallback）；④ 运行中的 exe 可被 `Copy-Item` 直接覆盖（进程持旧映射），桌面副本无需先关 app，但**已加载实例不会换图标，需重启**。**小尺寸可辨识（同日追加）**=原方案对大图统一降采样，致 16×16 环宽仅 1.5px / 核 3.1px、G 字形糊成深色块；新增派生资产 `build/appicon-small.svg`（环 48→64、核 r50→62、去 halo），ICO **按尺寸分流源图：16/24/32 用简化变体、48 及以上用主图**——衔接依据=变体 32px 环宽 **4.0px** ≈ 主图 48px 环宽 **4.5px**（若在 24→32 切会跳）；**坑=Pillow 的 ICO writer 只接受单一源图**，混合源尺寸必须手工组装 ICO 容器（ICONDIR 6B + 逐帧 16B + PNG 帧，**256 档宽/高字段写 0**，写 256 溢出单字节）。ICO=7 档 44384 B SHA256=**775CAED3274EC199DF7F3E188190BFCDB3A319EDD01EF2A350699FC8D6E2BD02**。**未抬版本**——按 README 发版约定「文档整理、令牌对照、单点样式等小改记入 CHANGELOG，不单独抬版本」，本刀属视觉资产补齐，与 `progress.md` 的「工作空间与文档整理（非版本刀）」同类处置。

- **最新发布：v4.211.0（2026-09-11）「5.1 收口：回复发出前剥离悬空 [MEM:] 引用」**——v4.210 余项收口，纯 Go 零绑定前端零改动。闸点=agent stream() 收尾：新 `Options.FinalizeText`/`SetFinalizeText` 定稿钩子在 Message 全文事件发出**前**改写答案文本，改写值同进 session 历史与 TurnResult 摘要——幻觉键到不了持久层；前端收 Message 事件整泡替换（既有语义零新事件形态），流式增量原样透传由重渲染收敛。boot 装配闭包（记忆总闸+库可用性门控；子代理/headless nil 不改写）；`memory.Store.StripDanglingCitations` 纯函数=命中键保留/悬空键含紧邻空白整处剥离/**不 Touch**（触达职责仍在回合收尾，两路分离防双记）；被剥离键当场落 dangling cite 事件（v4.210 悬空可观测性不丢）。空间语义同读端隔离器。测试=Go +5（memory 三例：命中保留+悬空剥离+空间隔离/不触达+零值 Store 跳过；agent 两例：Message/Summary/session 三路改写+nil no-op）；门禁=Go 全量 exit 0/vitest 以 v4.210 全绿基线为准（纯 Go 刀先例）/版本三处 4.211.0。**5.1 出口判据全满足，阶段五下一刀=5.2 上下文编译**（规划大纲 §2）。
- **最新发布：v4.210.0（2026-09-11）「记忆语义图谱：事件日志投影出图」**——**阶段五首刀**（规划大纲 §2 主轴旗，底座#1；设计基线 docs/gaea-memory-graph-51-design-2026-09.md，5.2/5.3 在其上续写）。**缺口**=记忆只有存储没有结构：facts 按名存取、MemoryHub 关联图是演示型（同标签拼边无事件事实）、三脑数据不可推理。**落地三层**=① memory_events 事件日志（**SchemaV18**，追加式 INSERT-only：remember 工具/桌面面板/做梦/蒸馏合并/引用触达五条写路径经 sqliteBackend 落库成功即留痕，Touch 逐条落——5.3 衰减的事实源顺带就位；不存全文只带投影元数据+280B excerpt，内容真相仍在 facts）；② ProjectEvents 纯函数投影（实体/事件/来源三向边 produces/affects/references；实体 id=`mem:<project>/<name>`=[MEM:name] 图寻址；last-write-wins 折算；输出全排序→同日志两次投影逐字段一致）；③ mem_graph_* 物化（embedding 向量占位列 5.2 启用；读前水位对账懒重建→**删库可重建**）。**悬空拒写**=references 边两端实体在场才物化，悬空目标不进图但 Dangling 留痕；cite 事件永不建实体（防幻觉键节点化）；回合收尾 ResolveCitationsDetailed 命中+悬空各落 cite 事件。**绑定 608→609**（GaeaMemorySemanticGraph→SemanticGraph 返回 SemanticGraphView 带事件/悬空统计）；MemoryHub 图谱页「关联图/事件图谱」切换（GraphView source prop，默认 cloud 零变化）+domainColors 三新色。**坑**=①「物化≠投影」DeepEqual 双源口径：Tags/Refs 经 JSON 往返 nil↔[]string{} 漂移，LoadEvents 与 MaterializedGraph 双侧归一 nil 才锁死；②gen_bindings 重生成把 office 门面重排序+改 import 别名（v4.178 同坑），只留 memory 新行其余还原。**测试**=Go +9（投影确定性/三向边/悬空不入图/实体fold/删库重建自愈/BFS 遍历含 [MEM:] 键解析/日志读写/写路径逐 op 挂钩/cite 事件形态）、vitest +1（semantic 数据源切换+标题）、tsc 0、既有 GraphView/MemoryHubPage 8 例零变更。门禁=Go 全量 exit 0 / drift PASS@609 / 版本三处 sync 4.210.0。**5.1 余项**=真·发送前悬空引用剥离（需流式层改写最终文本，独立小刀）；UI 遍历路径/手动重建按钮按需另刀（Go API 已备 GraphNeighbors/RebuildGraph）。
- **最新发布：v4.209.0（2026-09-10）「组价含量对照：拆解含量 vs 同类条目分位带」**——造价刀路池 §6 收官，**survey 六项缺口全清**（docs/gaea-cost-domain-survey-2026-09.md 已逐项回写收口状态，勿再当欠账池引用）。**缺口**=AI 拆解人材机含量只有金额自洽校验（v4.158 R1-R3），含量本身离不离谱没人管。**落地**=`cost.CheckContentBaseline` 纯函数（contentband.go）：拆解组件 vs 相似条目池同键含量分位带，**匹配键=标题归一精确相等（全角→半角/去空白/小写）+单位归一相等（含双方都空），一方缺失一律不比——宁缺勿误与导入链同哲学**；P25/P75 R-7 同口径，q>P75 偏高 warn / q<P25 偏低 warn / 带内静默降噪；P25==P75 同值退化带 ±5% 容差（否则同值样本把微小差异全标异常）；同键样本 <`MinContentSamples`(3) 不比对；文案自带口径「含量 420 高于同类 5 例边界 315（+33.3%）」。**接线**=gaea_cost_compose.go `composeChecks` 汇总金额自洽+含量对照两层，对照池=相似条目本身组件明细（检索相似条目=「同类」口径，与价格带同池，逐条 Get 失败跳过池空静默）；Checks 随视图展示+Apply 留痕自动承载，**零新结构零新绑定前端零改动**。**边界**=外部「行业含量区间」数据源明确不引入——基线源=用户自己的库（私人记忆哲学），样本数即诚实口径。**测试**=Go +5（contentband_test 四例：带外双向+归一匹配+单位守卫三态+退化带+守卫；composeChecks 接线：带外双条/带内双静默/池空剩金额层）。**门禁**=Go 全量 exit 0（116 包 0 FAIL）/ drift PASS@608（零绑定）/ 纯 Go 刀 vitest 以 v4.208 全绿基线为准（v4.197 先例）/ 版本三处 sync 4.209.0。**产物**=build.bat（1m02.7s）→ build\bin\gaea.exe **48,580,096 B** + 冒烟 200 + Desktop\gaea.exe 同哈希；SHA256=**637c6d835d6355b62edfffbed68602bda35fec1e78a7a8d1f7b948607e7f9537**（releases/SHA256SUMS-v4.209.0.txt）；未打 tag（随 v4.200+ 现状）。欠账=组价含量对照端到端待真机（观察池）；造价域新缺口另立调研。
- **最新发布：v4.208.0（2026-09-10）「首页 v7 重设计：书斋「文书台」/ 闲庭「游园画廊」」**——v6 版式廉价感定位=「等大圆角卡片+描边」一种形态重复到底。**v7 结构三形态**（仪表条 hairline 行 / 账页等宽序号行 / 海报墙真大小跨格）+ **四档排印**（display clamp 负字距 / lede / body / label 宽字距，数据等宽）+ **三级色调面**（surface→container→container-high，描边退 hairline）+ 版式记号（序号水印 / 月洞门细环）取代极光斑；信息零删除，契约 testid（ml-space-* / desk-recent-docs / garden-*）全保留。**坑：CSS 重写会静默吃掉「同名 class 的规则」**——`.ml-avatar-ai` / `.ml-avatar-user` 在 HEAD 有规则、v7 新版没有（TSX 仍在用），两个头像退化为同底色；**检查配方=脚本双向比对「TSX className 集合 vs CSS 选择器集合」**，本刀跑出 3 处 TSX 无规则（2 真丢 + 1 空挂 `.p-foot-wide`）后修正。附带**门禁可信化**：CI「无条件重试一次」→「只在册 flaky 隔离复跑」（`known-flaky.txt` 现为空 + 分类器自测）、`maxWorkers` 按物理核夹 [4,8]（原按逻辑核开 ~31 worker→重组件首个 `await import` 撞超时假红）、testTimeout 30s、RTL `asyncUtilTimeout` 5s；**教训=afterEach 清 body 会打挂 antd 模块级 message 容器（实测一次 14 例假红）**，残留改由并发上限从源头消除。门禁=tsc/eslint 0 / e-check OK / drift PASS@608（零绑定）/ Go 全量 / vitest 324 文件 2799 例 / 版本三处 4.208.0 / 产物=主树 wails build -ldflags "-s -w" -trimpath（1m08.5s）→ build\bin\gaea.exe 48,570,880 B（46.32MB）+ 冒烟 /api/health 200 + Desktop/build\bin 双副本；SHA256=59c2b518bdb6165ccbac07bbc565759a2d7571f254e287e27aa00b2bccd97a16（releases/SHA256SUMS-v4.208.0.txt）——构建后工作树仍干净（wails 的 npm install 未改写 lockfile）/ 目检=dev(`?mock=1`)+无头 Edge CDP 两空间 ×4 档共 10 张（无横向溢出）+ **壳内实测**（构建产物 `build\bin\gaea.exe` + `WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS=--remote-debugging-port=9333`：书斋/闲庭**浅色主题+真实数据**渲染正常——内核 6 引擎/真实最近文档 5 条/记忆 1706 条/晨报 9 条、越界元素 0、**真点击切换器原地切换** aria-pressed 与 localStorage 同步、无 reload 绕过）。欠账=海报墙首张（小说）大样中段留白偏多（观感待定）；动效手感（入场分阶 / hover 位移节奏）待上手定论（浅色观感已见实机）；真机池余项=.gsched「基线 N」chips / 最近文档 localStorage 写路径 / 冷启动基线采集。


- **v4.207.0 / v4.206.0 / v4.205.0（2026-09-10）主题令牌收口 + 小说书房工坊接线**——v4.207 深色主题硬编码深灰（RelationGraph 图例/提示、进度里程碑菱形与标签）改走 --color-text / on-surface；v4.206 33 处 `bg-accent text-white`→`text-accent-fg`（= on-primary）+ App 注入 --color-on-primary / --color-surface；v4.205 小说书架画廊接线（画廊头 + 正在编辑横条 + 书脊/角标）+ 阅读页属性检查器由 display:none 改默认可折叠（章节体检入口回来）。三刀均纯前端、绑定 608 零变更、Go 零改动。


- **最新发布：v4.204.0（2026-09-10）「组价多方案对照：P25/中位/P75 当场切」**——造价刀路池 §2 半刀。价格带三档数字早就在，推荐价钉死中位数。本刀不拆逐步盖章：ComposeModal 三档格可点（aria-pressed，应用价随档，默认中位数既有用例不动）；cost_compose 增 mode 参数（未知回落中位数）+输出「三档对照」一行。绑定 608 零变更。门禁=ComposeModal 11/11（+1）/Go TestCostCompose 绿/tsc 0/eslint 0/版本三处 4.204.0。欠账=流式打字机/费率政策对照不做；含量基线等样本；壳内真机走查池不变。


- **最新发布：v4.203.0（2026-09-10）「空间策略其余功能域键：总闸当场切」**——v4.190 欠账收刀：七键写路径早已通，编辑 UI 却只放 gaea 主控。本刀纯前端（绑定 608 零变更）把 chat/whisper/novel/office/characterlib/routine 拉进同一张空间策略卡：六键常显（空键灰字「未配置（维持现状）」），行级编辑闭环与 gaea 同款（带出当前值 → GaeaSpaceProfileSet(space, key, ref) → 视图随返回刷新；空串=清除；后端校验失败原样 message.error）。gaea 主控行与既有 testid 不动。阶段二总闸画面出口补齐「书斋和闲庭可以各绑各的」。门禁=StrategySection 8/8（+2）/tsc 0/eslint 0/版本三处 4.203.0。欠账=功能域×空间交叉矩阵按需另刀；壳内真机走查池不变。


- **最新发布：v4.183.0（2026-09-09）「双空间首页重排版：书斋文书台/闲庭游园画廊 编辑部级排印」**——用户反馈 v4.182 版式「太 low」，定位=同质卡片海（8 等大 Bento 瓦片+右舷五连盒）是模板感根源；重排版信息零删除只动形态，ui-ux-pro-max 双查询定方向（书斋=Premium refined，闲庭=Bold gallery variance8/density3）。**书斋「文书台」**=编辑部排印：竖排空间名书脊 masthead（vertical-rl，≤900 收起）+30px 大字渐变标题（background-clip:text）+命令条（focus-within 辉光）+最近文档卷宗流水（hairline 报表行+等宽序号 hover 点亮）+旗舰横带（accent 竖条+细网格纹）+**目录式索引行**（等宽序号 01-05 两列报表 hover 辉光，替代等大瓦片）+右栏单一仪表纵栏（遥测/写作/会话/记忆/晨报 hairline 分节，去五连盒；≤1180 落底双列）。**闲庭「游园画廊」**=画廊排印：居中大字 clamp42+宽字距+空间名小签两侧渐隐线+会客厅旗舰横幅（**月洞门圆环母题**+84px 圆徽记+CTA 胶囊）+**竖版海报大卡**（62px 圆徽记+底部环形箭头 hover 滑入；设置=行形态）+园底单条信息带（进度环/继续话题 chips/记忆/遥测 hairline 四节，≤1100 折 2×2）。**结构修复（v4.182 遗留缺陷）**=M3 tertiary 角色全仓从未定义（切换器暖色恒走 fallback，闲庭与书斋实际同色）——appStore ThemeTokens 增 tertiaryContainer/onTertiaryContainer+TERTIARY 暖伴侣色表（6 预设×明暗）+getThemeTokens 合并+App.tsx 注入两 CSS 变量，契约测试锁定（12 套 tertiary 对存在且不等）。**坑**：bridge 浏览器 dev mock 优先于 window.go.main.App 桩（隔离目检须打 window.go.app+Gaea* 映射键名才走 realApp 真机路径）；晨报绑定回 JSON 字符串非对象。locale 零新增键；testid/aria 全保留（ModuleLauncher.test 3 例原样绿）。**门禁**=vitest **2746**（+1 tertiary 契约）首跑全绿/tsc 0/eslint 0/e-check OK/locale 0 死键/drift PASS@602/零 Go 改动（版本三处 sync 4.183.0 外）/build strip+冒烟 200/SHA256=a5f54c93af96605c89cec94f22cc31835f684d124552e82e360355d0ead93a09（exe 46.2MB）。目检=esbuild 隔离挂载+Edge 无头截图（书斋满数据/闲庭满数据/书斋 1000px 窄档三张）。欠账=壳内真机走查两首页观感/切换手感（真机池，含 v4.182 欠账）；浅色主题首页观感未目检。


- **v4.182.0（2026-09-09）「双空间首页分版：书斋/闲庭定名 + 切换器迁首页顶栏」**——用户拍板切换按钮迁首页顶栏+两首页各有特色不要长得一样。**定名**=书斋（work，zh-TW 書齋，en Study）/闲庭（play，zh-TW 閒庭，en Lounge），三语覆盖 shell.space.*/shell.search.scope.*+space.ts 兜底 label；组件无运行时硬编码（仅注释）。**切换器迁移**=首页顶栏 SpaceSwitch（SHELL_SPACES 驱动胶囊组 aria-pressed+当前空间点击防抖+闲庭激活 tertiary 暖色令牌+右侧模型徽标，直连 MainLayout.switchSpace）；rail 顶部改竖排空间指示徽标（非交互 div+title，空间感知保留切换动作归一首页），rail 分域导航不变。**书斋「文书台」**=三栏效率台（顶栏+Hero 命令条 AI 直启+最近文档流水主角面板+能力矩阵 Bento+右舷遥测/写作/会话/记忆/晨报；density7/Flat）。**闲庭「游园画廊」**=全幅无右舷无命令条（居中标题+会客厅旗舰横幅全宽渐变大卡+板块大卡两列画廊+园底信息带；density4/Showcase）——信息零删除形态分化。ModuleLauncher 重构=useLauncherData 顶层一次拉取+DeskHome/GardenHome 两变体分发。测试=ModuleLauncher.test 3 例+CommandRail.test 更新+space.test 同步；vitest 2745/tsc 0/eslint 0/drift PASS@602/SHA256=F8B238863236F53803AB07822A1136366EEC92E97403BC5B592C1C04214E1337。欠账=壳内真机走查两首页（v4.183 重排版后合并走查）。


- **最新发布：v4.181.0（2026-09-09）「上下文/轨迹按会话读取 + GLM Coding Plan 429 分诊」**——用户反馈 Agent 网络不按当前会话。**根因**=GaeaTrajectory/GaeaAgentNetwork **无参绑定恒读内核 `gaeaCtrl().SessionPath()`**（ga.ctrl 单例恒指向最近活跃会话），UI 切历史会话后子代理 runs（有参）正确切换而网络树/轨迹仍是内核会话内容——「UI 会话 ≠ 内核会话」是这族 bug 的共同根因（GaeaHistory/ContextView 等其余无参绑定待逐个审计）。**修复**：①两绑定变参 `sessionPath ...string`（显式路径优先/缺省回落内核会话兼容旧调用，GaeaTaskList 先例；resolveGaeaSessionPath 归一+gaeaContextWindow 拆出），门面两行透传，**绑定名 602 零变更 drift PASS**；②bridge 契约 JS 数组形态（Go 变参→JS 数组，TaskList 先例；**proxy 是裸转发勿声明单 string**）+mock 同步签名；③agentNetworkStore 路径声明=poller 增 path，subscribe(cb,{path})/reload(path)，**跨会话切换清空快照置 loading 立即重拉（旧会话树=误导数据宁空勿错）/同会话重试宁旧勿断不变**；④六消费方接线=ContextView（订阅+load reload+依赖补齐）、SubagentsPanel/TasksWorkbench（订阅+prevPath 切换+refresh+retry 全带 path）、TrajectoryView（新 sessionPath prop，App 传 currentSessionPath）、AgentNetworkCard（load 带参）；BrowserPanel 保持内核兜底（无会话上下文的注入式组件，fetchTrajectory 注入点留给消费方）。**附带 GLM 429 分诊**：用户实测 GLM 引擎 429「无可用资源包」——编码套餐资源包挂 coding 端点（/api/coding/paas/v4）计费域，标准端点（/api/paas/v4）查不到即 429，**不是 Key 坏是端点没切**（模型中心 GLM 卡片 std/coding 切换已有）；glmPing 标准端点 429 时错误信息追加分诊提示。**测试坑×2**：纯 Node store 测试无 act（用与原用例一致的 `vi.advanceTimersByTimeAsync(0)` flush，非 fake timers describe 用 `setTimeout(0)` macrotask）；测试间 poller 单例残留连累后续用例（用例内自清理所有订阅）。门禁=Go 全量 0 FAIL（+TestGaeaTrajectoryAndNetworkExplicitSessionPath 磁盘日志+ctrl=nil 下显式路径生效）/vitest 2742（+3 路径声明用例；全量两轮收敛至在册 ContextView/MemoryPanel/TrajectoryView 负载 flaky 族单独复跑 43/43 绿）/tsc 0/eslint 0/e-check OK/drift PASS@602/build strip+冒烟 200/SHA256=D3BD8386862D5F74EC2BC63F8CA5B35394FA812588376DA306ECAC078AA62D55。欠账=TaskCenter 会话归属（任务表 session 维度）挂结构刀池；GaeaHistory/ContextView 等无参绑定「内核会话」语义逐个审计。


- **最新发布：v4.180.0（2026-09-09）「办公任务管理会话关联刀：会话待办区入任务管理页」**——用户反馈任务管理「没与当前会话关联、全是固定内容」。**定位**=任务管理页（TasksWorkbench，GaeaPage→ContextView→AgentRadial 三层下）三区中子代理/本地模型工具本就按 sessionPath 建册，但页面主体是 ② TaskCenter——数据源 `GaeaTaskList` 是**全局后台任务表**（价格抓取/语义索引等系统 cron 周期任务持续在列），天然会话无关=用户看到的固定内容；会话真任务（todo_write 待办）反而无入口。**修复**=TasksWorkbench 首区新增「会话待办」：直订全局 controller store `useStore((s) => s.items)`（当前会话活动流随会话切换，零 props 穿层，ContextModal 弹层实例同源自动正确）+ 复用 `useTodoExtractor` 提取最新 todo_write（倒序取无 parentId 最新一条=与聊天流待办卡同一规则单一来源）；渲染=区块头「会话待办 n/m」+三态条目（in_progress 蓝点脉冲/pending 灰点/completed 绿点划线置灰，level 子步骤缩进），**无待办不占位**。页面四区语义=会话待办/子代理/本地模型工具（均会话）+任务中心（全局 work 空间）。**坑=测试 vi.mock("../lib/bridge") 工厂仅给 onEvent，TasksWorkbench 新 import lib/store（controller.ts import app/onReady）后 vitest 对缺失 named export 宽松处理未炸——新组件引入真 store 时 mock 工厂补齐 named exports 更稳**。门禁=vitest **2739/2739 首跑全绿**（+1：待办区无→出现→清单更新三段用例）/tsc 0/eslint 0/e-check OK/drift PASS@602（零绑定）/零 Go 改动/build strip+冒烟 200/SHA256=69C89001048CB8BFD99BCE7122C69AB92A8C01A85CB4E34D23BAA45A1A71C0DD。欠账=TaskCenter 会话关联需 Go 任务表加 session 维度（Schema+提交链路=结构刀另立版本，当前以区块语义区隔过渡观察反馈）；观察池=「会话后台任务」过滤视图等 session 维度就位。


- **最新发布：v4.179.0（2026-09-09）「瘦身清点刀：knip 死代码清除 + 依赖显式化 + 冷启动基线打点」**——瘦身总规划 P0-P4 收官后清点刀（轨道二尾欠 knip + 轨道五运行面测量）。**knip 首扫三发现三处置**：①死文件 13 全删（App.css/AppBar/MobileSheet/SettingsMobile/SettingsUpdates/自制 Tooltip=各处用的均 antd/PromptShelf/office ParseSummaryCards+SourcePreviewDrawer 互为唯一引用死链对/memoryhub ComingSoon+ModuleCard/typesGenerationCheck/m3-palette——全部零 import 甄别含 scripts 层；连带清 zIndex.ts 过时注释档位值不动）；②幽灵依赖 13 包显式化=20 处 import 靠传递依赖侥幸工作升级即断（jszip docx/pptx 链 8 处、dayjs WeixinPage/GanttToolbar、katex Markdown 直引 CSS、unified+hast-util-sanitize sanitize.ts、@lezer 八子包 diffHighlight.ts——按 lock 现解析版本钉死零漂移）；③npm install 顺带清死传递依赖 @codemirror/search@6.7.2（全仓零引用）；**三存疑依赖验活结案**（v4.166 名单）：gsap=2 hook×6 组件消费/docx-preview=DocxPreview/unist-util-visit=remark 两插件全部在用保留。**冷启动基线打点（轨道五先测量后优化）**：Go 侧 app.New 构造链一条+Startup 七段（migrate/logging/engine/voice+weixin/charlib+stores/tts-kickoff/schedulers）+total 汇总 slog——日常启动自动落 logs/gaea-YYYYMMDD.log 真机基线零成本积累；前端 index.html 解析即 performance.mark('gaea:boot')→React 挂载后第二个 rAF gaea:interactive+measure（CDP getEntriesByName 可读+console.info；**有意不落 gaea.log——现有前端日志通道 GaeaLogFrontendError 是 slog.Error 级，不污染错误信号**）。**初始化链审计结案=懒初始化改造无需立项**：TTS 拉起幂等异步/价格源+文件索引首轮=任务队列异步提交/四引擎模型刷新 goroutine/预载延迟 10s/巡检延迟 1min——同步 Startup 链只剩配置/嵌入 FS/DB 打开/角色库迁移轻量磁盘活，轨道五收敛为真机冷启动/常驻内存采集（与壳内真机池同窗口）。门禁=vitest 2738/2738（5 例 ContextView/MemoryPanel/TrajectoryView 并发负载 flaky 单独复跑全绿在册先例）/tsc 0/eslint 0/drift PASS@602（零绑定）/e-check OK/Go 116 包 0 FAIL/build strip+冒烟 200。SHA256=3FCB187F496AF2B1A89884D860F675D1E56A92B6B50AA932E0655A65521D320F。度量=exe 46.16MB 持平（死文件本就不进 bundle，本刀收益在仓库卫生与依赖正确性）；knip 复扫 Unused files 0/Unlisted dependencies 0。欠账=knip Unused exports 169 项挂下版（甄别三态：测试专用/预留面/真死，可配「新导出须有消费方」门禁）+壳内真机池/审计刀D 不变。


- **最新发布：v4.178.0（2026-09-09）「造价·条目匹配键刀：定额编码贯通存储→导入→匹配→工具面→UI」**——造价域缺口清单 §1 首刀销账（docs/gaea-cost-domain-survey-2026-09.md）。**数据层**=SchemaV17 cost_entries/cost_estimate_items 增 code 列（TEXT DEFAULT ''，归一化存储）+idx_cost_code 索引，迁移测试覆盖新库全链 V1→V17+V16 升级（历史行默认空串）；**cost.NormalizeCode**=全角 ASCII→半角+去空白+大写（ａ１－１２/a1 12/A1-12 同码命中，空串=未录入）；Entry/Summary/Item 三结构 Save/Get/Search/List 贯通，搜索 haystack+BM25 均含 code。**导入链**=表头 fieldCode 映射（清单项目编码/定额编号/清单编码/项目编码/定额编码/定额号/清单号/编码，最长胜出，「编号」仍噪声）；**编码优先匹配**=带码命中→覆盖提示、带码未命中→直接新增（同标题不同编码=不同子目不标题兜底，杜绝跨子目误覆盖）、无码保持原语义；AI 解析 prompt 增 code 提取；vision 路径有意不动（报价单鲜带定额编码）。**匹配消费方**=costref entryIndexes 双索引→三索引、matchEntry 编码精确优先（matchedBy=code|entry_name|title 溯源，带码未命中宁漏勿误配与导入链同语义）；沉淀携带 item.Code+引用条目继承编码。**工具面/UI**=cost_save 增 code 参数（覆盖未传保留原编码）+cost_search 结果表编码列；EntryModal 编码表单项+LibraryView 编码 chip/行前缀+ImportModal 可编辑编码列+ProjectsView 明细编码输入+EntryPicker 引用带入；mock 层同批贯通。**附带收口=e-check 三处陈旧守卫修复**（E25 ChatSend/ChatImportTopic 接受 bridge 形态 app.ChatSend/app.ChatImportTopic=v4.174 双轨退役甩下；E24 MemoryHubPage→GraphView 接受 React.lazy 动态 import=v4.176 甩下；干净 HEAD 实测即红非本刀引入）。门禁=Go 全量 0 FAIL（五包 +9 测试）/vitest 2738/2738（2737+1）/tsc 0/eslint 0/drift PASS@602（零绑定）/locale 0 死/e-check OK/build strip+冒烟 200。SHA256=F83CDB9BED35C7B4F045CC835B59981B95B5B4B67E39EC1D5282EF8F2E199561。欠账=造价缺口 2-6（组价流式分步+多方案/五算快照贯通/询价异常扫描/检索主动维护+清单级测评集/行业含量区间）、编码存量回填等带码样本、壳内真机池/审计刀D 真机取证不变。


- **最新发布：v4.177.0（2026-09-09）「瘦身 P4 懒加载三刀收官（mermaid 动态化 H1 + locales 按需 H7 + SearchModal lazy H8）」**——瘦身执行层第十二版，P4 运行刀收官。**H1 mermaid 动态 import**（Markdown.tsx/mermaidPng.ts 删静态 import→ensureMermaid 异步 await+模块缓存 strict/loose 配置原样；MermaidBlock 迁 useEffect async IIFE 取消/失败语义不变；**全仓 mermaid 静态 import 清零、entry 对 mermaid.core 零引用、566.9KB 独立 async chunk 按需**；测试零改动）。**H7 locales 按需**（zh 静态保留首帧零异步+en/zh-TW 动态 import 显式分支独立 chunk；useT/t/setPref 签名零变动；Provider 先同步渲染回退→chunk 就绪 forceRender+alive）。**主代理关键修复=translate 回退链 DICTS[locale]→DICTS.zh→DICTS.en→key**——**zh 静态恒可用兜底**，非 React 调用方（lib/tools.ts 摘要 currentLocale 默认 en）在 en chunk 未加载时裸键永不外泄（首跑 13 例失败根因）；5 测试补 beforeAll(loadLocale('en'))+ModelSwitcher await。**H8 SearchModal lazy**（React.lazy+Suspense fallback null）。门禁=vitest 2737/2737（首跑 13 例 i18n 时序失败 zh 兜底+适配后全绿；1 例 FilePreviewModal 并发负载 flaky 单独复跑 15/15 绿在册先例）/tsc 0/eslint 0/drift PASS@602（零绑定）/locale 0 死/Go 回归绿/冒烟 200/SHA256=E98E49CD8134C2F0964A1E041F49714465A385AB678E6295A946F94971E7B5D0。**度量=entry 1193→755.48KB（−37%）**；en/zh-TW/SearchModal 独立 chunk；**exe strip 实战验证**（build.bat 固化 `-ldflags "-s -w" -trimpath`——46.2→46.15MB 减量甚微，Go 符号表在 Wails 应用占比小，-trimpath 安全收益为主；下轮起发布构建自动 strip）。**P4 收官**：Go >50KB 0；entry −37%；MemoryHubPage 页壳 15.95KB；mock/office 1.1KB 入口；wailsjsCompat 退役（v4.174）。欠账=壳内真机池/审计刀D 真机取证/观察池刀3（全真机绑定观察项）。


- **最新发布：v4.176.0（2026-09-09）「瘦身 P4 结构刀2 + entry 懒加载（mock/office 拆分 + MemoryHubPage 全 tab lazy + mock 异步 chunk）」**——瘦身执行层第十一版，P4 性能版第二刀。**结构刀2**：mock/office.ts 51.2→1.1KB 入口（6 文件：types/state/**schedule 10.5**/methods_xlsx 4.9/methods_office 32.7/build；**TS2632 教训**=let mockScheduleCurrent 被 Create/Delete 直接重赋值禁跨模块，schedule 状态与方法必须同文件与 weixin.ts 先例一致）。**运行刀 entry 懒加载**（调研修正 P0 两处：**cytoscape/cynefin 是 mermaid 传递依赖非 hub 相关、pageLazy 是 PDF 逐页懒挂载非页面 lazy**——页面级 lazy 早已 main.tsx registerPage 全就位）：**H6 mock 异步 chunk**（proxy.ts 删静态 import→startMockChunk 单例 `import("../mock")`+真机门控零加载+冷路径异步 thunk；events 走 mockEventSharedSync/waitMockReady；**index 1149→967.23KB −182KB/−15.8%**）；**H2-H5 MemoryHubPage 8 组件全页内 React.lazy**（**1402.5→15.95KB 页壳**；GraphView/three.js 1,354.97KB、KnowledgePanel 51.3/Markdown 212.9/katex 261.3/mermaid.core 609.8 全独立 chunk 按需——GraphView lazy 决策：home 默认 tab 首开仍拉一次与现状等价但其余入口不摊 three.js）。**H1 mermaid 动态 import 递下一轮**（mermaidPng.ts footprint 外静态可达 + memoryhub lazy 已拆共享 chunk + Markdown 测试面大）。**连带测试时序修复×2**（mock 异步化使 `window.__mockScheduleFile` 与 SearchModal RouteIntent/UnifiedSearch 演示规则同步断言未就绪——两测试补 `beforeAll(await waitMockReady())`、bridge.ts 入口加 waitMockReady re-export；首跑 7 例失败修复后全绿）。门禁=vitest 2737/2737 全绿/tsc 0/eslint 0/drift PASS@602（零绑定）/locale 0 死/Go 回归绿/冒烟 200/SHA256=402A1B1A937CF6690E435C1F4E61AA6BB6D72487543A6B078E0F9F5F132DBD2F。度量=entry 967.23kB（基线 1193→−19%）。欠账=P4 续（**H1 mermaid 动态 import + mermaidPng.ts、H7 locales 按需 −160~200KB、H8 SearchModal lazy −20~40KB、exe strip wails -ldflags -s -w -trimpath 发布版 dev 保留符号**）、壳内真机池/审计刀D 不变。


- **最新发布：v4.175.0（2026-09-09）「瘦身 P4 结构刀1：三巨文件拆分（config.go/engine.go/store.ts，Go >50KB 2→0）」**——瘦身执行层第十版，P4 性能版结构前置刀。三线并发（足迹互斥）+ 主代理收口，行为零变化。**线A config.go 58.7→16.1KB**（7 文件同包：核心骨架+config_keys/config_types/config_features/config_prefs/config_realtime/config_save；逐段 Contains 断言 8 区逐字节命中）。**线B engine.go 56.5→13.3KB**（8 文件同包：核心+engine_keys/engine_custom/engine_crud/engine_connect/engine_models/engine_modelhub/engine_state；核实无 herdsman 专属函数；45 func 无重复无丢失）。**线C store.ts 54.3→0.4KB**（聚合入口 export *×3+store/controller+store/preview+store/commonts；18 导出面逐一同 63 消费方零改动；963/963 行逐字节对账；controller 相对 import 深度调整 4 行）。主代理收口：足迹 17 项零越界；Go >50KB 源文件 2→0；**mock/office.ts 51.2KB 浮现**（下轮目标）；store/controller.ts 50.2KB 裁定=单 store 自然边界保留（再拆需深拆 reducer 超行为零变化红线——50KB 红线是指导性目标，自然聚合边界可容忍略超）；entry chunk 1190.55→1176.62kB。门禁=Go 128/128 包 0 FAIL/vitest 2737/2737 首跑全绿/tsc 0/eslint 0/drift PASS@602（零绑定）/locale 0 死/冒烟 200/SHA256=729C942B1A21B4FE0DB242565808B8CE86DB6F5B812810C836DDEBE4065F9DB8。欠账=P4 续（mock/office.ts 51.2 拆分、entry 懒加载 MemoryHubPage 1.4MB/GaeaPage 622KB/cynefin 690KB 按 tab 动态 import、exe strip -ldflags -s -w 对靶）、壳内真机池/审计刀D 不变。


- **最新发布：v4.174.0（2026-09-09）「瘦身 P3 版3终局：bridge 双轨退役收官（wailsjsCompat.ts shim 删除，全仓生产代码零引用）」**——瘦身执行层第九版，轨道四「双轨」渐进退役**收官刀**。**里程碑**：全仓生产 wailsjsCompat import 清零（python 实证）；**frontend/src/wailsjsCompat.ts 删除**；LegacySurfaceNames 292→**184**（累计转正 108 方法）；spaceBindings 423。**契约扩展批次三四线共 42 方法**：3a +18（VoiceBindings 9 语音/WhisperClearSession/TTS×2 + ImageBindings 2——**GetTTSSpeakers/GetVoicePipelineConfig 实挂 ImageB 非 VoiceB** + OfficeBindings 6 DataBackup* + Core 1 ListProjects）；3b +38（CharLibBindings 16 角色库族 + NovelBindings 22 章节族/叙事状态族/场景族/项目角色族，**GetCharacters 同名核对=charlib.ts 已有与 Go NovelB 同签名直接可用**，CancelCreateChapter→boolean，v4.7x 小说革命注释段更新 RunChapterGate 保留）；3c +3（QuickBrainstormBranches/CreateChapter 8 参/DeleteOutlineNode）；3d +4（SetActiveASRModel/SetActiveTTSModel/SetChatVoiceModel ImageB + VoiceHealth VoiceB）；spaceBindings 360→378→416→419→423。**业务迁移 8 线并发**（各线定向测试全绿）：组1 语音族（useVoiceChat 8 方法 **VoicePushAudio Array→base64 对齐 Go []byte 契约**、useVoiceState 5、VoiceSettingsPanel 4、ChatPage 5 直调——22/22）；组2 cast 族（**CreatePage/ChapterEditor 删 `App as unknown as {...}` cast 对象**=绑定再生成后 cast 编译期冗余走 NovelB 门面具名——9/9）；组3 角色库（api/characterlib 18 方法 6 处 as unknown as——44/44）；组4 章节/小说角色（ChapterPage 4 + novel/api/character 9——7/7）；组5 收尾（useBindState/DataPanel 9——9/9）；CreatePage 收尾（7 处+3c——9/9）；终局 3d（useVoiceState+VoiceHealth+ChatPanel 4 保留整体 try/catch——47/47）；主代理收尾（ChatPage.test/ChapterPage.test wailsjsCompat 别名改 bindingsBridge 直取、SettingsPage.test 死 vi.mock 改 bridge、**shim 删除**）。门禁=vitest **2737/2737 首跑全绿**（shim 删除后零回归）/tsc 0/eslint 0/drift PASS@602（零绑定）/locale 0 死/Go 回归绿/冒烟 200/SHA256=AD7C7CED1066F7BFF077B2416D41145BEE207619868D411F9A39FDF518737E2C。**收官意义**：S2-3 兼容层退役全部前端统一 gaea/lib/bridge 代理（?mock=1 dev mock 回退+BridgeError 错误归一），masterplan 轨道四「双轨」目标达成；CharacterList 分类遗留裁定=work（v4.48 微信触点先例）有意保留。后续=P4（config.go 58.7/engine.go 55.2/store.ts 54.3 拆分、entry 懒加载、exe strip）+ 壳内真机池/审计刀D 不变。

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
