# gaea 项目记忆

> 本文件为项目长期记忆（文档记忆层级）。编码规范：**UTF-8 无 BOM**（历史遗留的 GBK/UTF-8 混合编码已清理）。
> 修改后请保持 UTF-8；.ps1 脚本需 UTF-8 带 BOM（见「沙箱环境备忘」）。

## 版本状态（顶部速览）

> 更早版本见 `docs/archive/agents-version-history-2026-09.md`（覆盖 v4.173–v4.146 与 v4.49 及更早）；**v4.50–v4.145 区间本仓只在 `CHANGELOG.md` / `releases/` 有记录**（速览逐版滚出后未回填）。本段只留最近 14 版：本文件曾达 104 KB 超工作区指令预算（65536 B），尾部纪律/规划段一度对后续会话不可见，2026-09-10 二次分流。

- **品牌资产补齐（2026-09-10，非版本刀）**——2026-09-10 15:20 的「品牌资产刷新」只覆盖 **4 个 SVG**（`build/appicon.svg` / `frontend/public/favicon.svg` / `gaea/assets/logo.svg` / `logo-light.svg`，新视觉=「地核 G」轨道环抱活核），**两个二进制件没跟上**：`build/appicon.png`（1024²）与 `build/windows/icon.ico` 停在 **2026-08-05** 旧视觉（翡翠球体+破土嫩芽+星芒，深蓝底 `#0F172A`）→ **exe 内嵌图标/任务栏/资源管理器/桌面快捷方式全显示旧 logo，只有窗口内 UI 是新 logo（「换了一半」）**。**落地**=① 无头 Edge 渲染 `appicon.svg` →1024×1024 RGBA PNG（**必须传 `--default-background-color=00000000`**，否则圆角外合成不透明白、alpha 静默丢失）；② Pillow 由同一母图出 **7 档 ICO**（16/24/32/48/64/128/256，旧资产缺 24，本次补 Windows 标准全集）；③ `wails build -s -ldflags "-s -w" -trimpath`（**13.5s**）。**验证三条独立证据**=① 几何精度（核心圆 r=50@512 实测直径 **200px**、圆心 (539.5,511.5) 对理论 (540,512)；环 stroke 48 实测线宽 **96px**；圆角 rx=104 对角线首不透明像素 d=**61** 对理论 60.9——零缩放零偏移）；② exe 内嵌图标提取（`ExtractAssociatedIcon`：底板 `#0B1210`/米色核/翡翠环=新「地核 G」）；③ dist bundle 新 logo 独有标记 `gaea-logo-ring`×3、`gaea-logo-light-ring`×3、`0B1210`×2 在册且旧独占色 `#6ee7b7`/`#047857` **零命中**+按旧 SHA256 全仓比对零旧 logo 残留。**门禁**=build exit 0 / 冒烟 200 / **纯资产刀（零前端零 Go 源码改动、绑定面不变）**。**产物**=`build\bin\gaea.exe` **48,572,928 B** SHA256=**5F811126131A21456D7C84B6D568EC2F0B4A748D17F60AD2814879C795FD4264**（桌面副本同哈希；中间态 903F9759…C36EC068=只换 PNG+主图 ICO 的一版，小尺寸优化后重建覆盖；原始 59C2B518… 系 v4.208.0 归档值，releases 档案自洽未动）。**坑**=① 无头 Edge 截图默认白底，CSS 里写 `background:transparent` 无效（截图不继承页面背景），透明基线拿旧资产角像素 alpha=0 对照；② **判新旧前须验该色是否新旧共有**——`#34D399` 同时存在于**新** logo 的 halo 渐变与**旧** logo 主色，拿它判会把新资产误报成旧的，判据只能用**独有**标记（SVG id 名 / 独占色）；③ `pwsh` 不在 PATH，手工跑 `scripts/smoke.ps1` 会 `CommandNotFoundException`（**非 app 故障，易误判为冒烟失败**），用 `powershell`（build.bat 内已有 fallback）；④ 运行中的 exe 可被 `Copy-Item` 直接覆盖（进程持旧映射），桌面副本无需先关 app，但**已加载实例不会换图标，需重启**。**小尺寸可辨识（同日追加）**=原方案对大图统一降采样，致 16×16 环宽仅 1.5px / 核 3.1px、G 字形糊成深色块；新增派生资产 `build/appicon-small.svg`（环 48→64、核 r50→62、去 halo），ICO **按尺寸分流源图：16/24/32 用简化变体、48 及以上用主图**——衔接依据=变体 32px 环宽 **4.0px** ≈ 主图 48px 环宽 **4.5px**（若在 24→32 切会跳）；**坑=Pillow 的 ICO writer 只接受单一源图**，混合源尺寸必须手工组装 ICO 容器（ICONDIR 6B + 逐帧 16B + PNG 帧，**256 档宽/高字段写 0**，写 256 溢出单字节）。ICO=7 档 44384 B SHA256=**775CAED3274EC199DF7F3E188190BFCDB3A319EDD01EF2A350699FC8D6E2BD02**。**未抬版本**——按 README 发版约定「文档整理、令牌对照、单点样式等小改记入 CHANGELOG，不单独抬版本」，本刀属视觉资产补齐，与 `progress.md` 的「工作空间与文档整理（非版本刀）」同类处置。

- **最新发布：v4.225.0（2026-09-11）「规范知识出内核第二刀：小说 AI 味词表外置」**——红线（v4.224 拍板）审计后首落地（小说域）。**引擎=机制留码，词表=知识出码**：internal/novelstyle/words.json go:embed 内置默认（genui 先例，开箱即用）+ LoadWordsFile 整表覆盖 API（不合并可预测；惯例路径 .gaea/skills/novel-deslop/words.json，改词表不发版）+ 引擎三处接线（rewrite 替换表/segment 分词词表 vocab **原子快照可重建** tokenize 无锁读旧快照/score 规则 9 黑名单）+ app ensureNovelStyleWords 每进程懒加载（两入口；缺失/解析失败静默保默认=增强面不挡主功能）+ SKILL.md 机制说明与改表指引。测试 Go +2（漂移守卫 27/19/覆盖全链/缺失静默/cleanup 恢复内置）。**坑**=①词表快照持有者须包级变量初始化先于一切 init（segment 的 vocab 构建在 init 里读它）；②双 config 坑再证：app 包 config 名被 internal/config 占用，gaea/config 需别名；③ConventionDirs 变量 .gaea 首位即优先序，无 ConventionDirsAt。**门禁**=Go 全量 65 包 0 FAIL/绑定 624 零变更/版本三处 4.225.0。**审计余项入拍板池 §5.4/5.5**：DCMA 14 点形态（阈值数据化 vs 整体转技能）、造价 R1-R3 技能化（内嵌 Apply 留痕链路）——涉产品面待拍板。

- **最新发布：v4.224.0（2026-09-11）「架构修正：规范知识出内核——排版细则改技能按需加载」**——**用户拍板的架构红线，永久有效：领域规范/行业模板/检测规则不应植入代码，应以 skill 技能按需加载防臃肿；内核只留通用机制；新需求先问「能否做成技能」**。动作=①撤销 v4.223 Go 侧 FormatChecker（format.go+test 删、registry/gaea_lint/docxedit.ReadDocumentXML 全回退——无消费者的取数口也不养）；②技能包 .gaea/skills/gongwen-9704（SKILL.md=红头要素表七项+排版细则表六项+换算备查+判定纪律「容差内即符合/不适用≠缺失/实测值进建议/宁缺勿误」；scripts/docx_layout.py=stdlib 解析 docx 排版事实，最小 docx 实测通过）；③既有红头要素/造价表式 Go checker 同属「规范在码」列观察池候选，迁移涉 GaeaDocumentLint 行为变更待拍板。**门禁**=Go 全量 0 FAIL/绑定 624 零变更/版本三处 4.224.0。

- **最新发布：v4.223.0（2026-09-11）「办公·中文文书规范包第二刀：GB/T 9704 排版细则 lint」**——板块并行池开工（roadmap 办公#5；非阶段旗）。纯 Go 零绑定零前端（LintReport 走既有 GaeaDocumentLint/前端规范包分组）。**缺口**=红头要素（v4.1c）只查文本层，GB/T 9704-2012 排版层无人管。**落地**=standard 包 FormatChecker：ParseDocxLayout 纯函数（document.xml 容错子集解析：pgMar/spacing/ind/jc/rFonts/sz/t，run 级 rPr 优先段级缺省，表格内段落计入）+ 六项检查（页边距 37/35/28/26mm±2 容差聚一条/标题二号 22pt 小标宋只建议不判定/正文三号仿宋多数票/行距 28~30 磅固定值多数符合即符合/首行缩进 2 字符/层级标题一、黑体（一）楷体字体名缺失不判宁缺勿误；数据不足=「不适用」不误报缺失，违规带实测值建议）。取数口=docxedit.ReadDocumentXML 新导出；LintDocumentWithLayout 聚合（layout=nil 行为不变）；gaea_lint.go docx 分支解析失败 fail-open。**测试**=Go +4（XML 片段解析事实/合规全过/违规六项实测/不适用+无边距）。**门禁**=Go 全量 0 FAIL（纯 Go 刀 vitest 以 v4.222 基线为准）/绑定 624 零变更/版本三处 4.223.0。**坑**=①range 头里组合字面量方法调用须括号 `(FormatChecker{}).CheckLayout(l)`（语法错 setup failed）；②attr helper 收 xml.StartElement 值非指针。**注**：本刀植入式 checker 已被 v4.224 按用户架构红线撤销并转为技能包 gongwen-9704，规则知识保留在技能资产中。

- **最新发布：v4.222.0（2026-09-11）「6.3 余项：dag_plan 增量改图——run_id 编辑模式」**——纯 Go 零绑定零前端改动（设计档 §6 又清一条，**DAG 线自然收尾**：§6 仅剩等外部设计的「运行中 steer 直穿+分级审批」与「够用」的 DeliverableRegistry）。**缺口**=改单节点指令也要整链重规划：run id 换、既有状态与验收全丢。**落地**=dag_plan 增 `run_id` 可选参=编辑模式（省略=新建原行为不变），核心 internal/gaea/dag `ApplyEdit` 纯函数原地调和：节点按 id 对账，title/prompt/dependsOn 全等→原样保留（状态/产物/ref/运行痕迹延续）；有变/新增→回 pending；运行中拒绝；被移除节点仍被依赖=悬空依赖 fail-closed 不静默断链；**改已验收节点=撤销验收**（产物是旧指令的产出），RevokedAcc 计数透出人读文案。**测试**=Go +2（dag 调和四态+撤销验收+三守卫；app 编辑路径全流程+三拒绝）。**门禁**=Go 全量 0 FAIL（纯 Go 刀 vitest 以 v4.221 基线为准，v4.209/4.212 先例）/drift@624 零变更/版本三处 4.222.0。**坑**=Edit 工具old_string 带函数签名行而 new_string 忘带会静默删函数行——本刀在 dag.go Derived 与 template.go atomic 两处触雷均当轮自愈（go build 立验）。

- **最新发布：v4.221.0（2026-09-11）「6.3 余项：子代理证据落账+按会话归因+波内并行」**——三合一刀（设计档 §6 首条清掉）。**起因=真机归因链核查发现结构性缺口：子代理写盘从不落 Journal**（子代理 Options 无 JournalDir/SessionID→flushJournal 跳过；v4.219「窗口归因」真机恒空集，fake-runner 测试靠手工塞卡绿，壳内真机走查恰挂观察池未跑——**测试绿≠真机通，fake runner 模拟什么归因就是什么**）。**① 子代理证据落账**=TaskTool 增 journalDir（SetSubagentJournalDir，boot 注入与主执行器同目录）+ Options.SessionID=run.Ref 经 runSubSession 下发——task 工具/RunNew/RunFollowUp 三路径全覆盖（汇于 runSubSession 是全覆盖关键）；ephemeral 无 ref 不启用宁缺勿错；顺带收口「task 子代理编辑对版本时间线/回滚不可见」审计缺口（6.1 同向，子代理开始建回滚基线）。**② 按会话归因**=节点 outputs=SessionID==ref 的证据卡 Target（精确归因，主对话同期写盘不再并入；ref 空回退窗口增量）；UI 口径注三语同步。**③ 波内并行**=dagExecute 波内 goroutine 并发（归因已精确不再依赖窗口不重叠；失败级联/终止级联/互斥落盘不变；波内并发由测试互等信号锁死）。**测试**=Go +3（agent 落账往返+独立会话文件/app 并行互等+会话隔离/生命周期 fake-runner 改真实落账形态）。**门禁**=Go 全量 0 FAIL/drift PASS@624/tsc -b 0/eslint 0/vitest 329 文件 2845 例全绿/版本三处 4.221.0。

- **最新发布：v4.220.0（2026-09-11）「6.3 余项：流水线模板库——存模板一键重建」**——阶段六收官后续刀（设计档 docs/gaea-office-dag-63-design-2026-09.md §6 余项之二清掉）。**模板只取图形状**：internal/gaea/dag/template.go 纯函数包（FromRun=剥状态/产物/ref/运行痕迹+空名回退 goal 截 24 字+限长 40 rune 显式报错；Instantiate=全新草稿 run 不自动起跑——起跑仍人拍板与整链首跑同闸；TemplateStore 落 `<cwd>/.gaea/work/dag/templates/`，**run 档同域子目录**，Store.List 只读顶层 *.json 互不混；Save 前全量 Validate=成环/悬空依赖/坏形状拒入库）。**绑定 620→624**（GaeaDagTemplateSave/List/New/Delete，Office 门面手工补行）；重建=模板当前形状快照，改模板不影响已重建 run；删除只删模板档（已重建 run 不受影响）。前端 DagPanel：模板折叠条（默认收起、有模板才显形）逐条「新建/删」+ run 卡「存模板」内联输入（默认带出 goal 截断，Escape/取消收起）；模板拉取失败静默不挡主列表，dag-retry 顺带重拉模板；?mock=1 预置「月度经营报告」模板。**测试**=Go +6（template_test：剥痕迹/名规则/实例化全新/存储往返+坏形状拒+穿越拒绝/列表倒序+与 run 隔离；app TestDagTemplateFlow=存→列→重建→重建链 fake-runner 跑通全 done→删→再删报错）、vitest +4（显隐折叠/新建重拉/存模板提交/删除+失败静默）。**门禁**=Go 全量 0 FAIL/drift PASS@624/tsc -b 0/eslint 0/vitest 329 文件 2845 例全绿/版本三处 4.220.0。**坑再证**=gofmt -w 会把 bindings_office.go 整文件重排（import 排序+全文件换行重写 492 行 diff）——门面只手工补行，格式化器/生成器禁过门面文件；bindingNames.ts 由 `gen_bindings -names` **只打印不写盘**，按输出手工补行。**§6 余项剩**：波内并行（等按 ref 归因定案）/运行中 GaeaSteer 直穿+危险操作分级审批（等审批闸分级面）/dag_plan 增量改图/产物登记 DeliverableRegistry。

- **最新发布：v4.219.0（2026-09-11）「阶段六 6.3 首刀：办公多文件 DAG——委托式文件流水线」**——阶段六收官刀（办公#3「多文件任务图（DAG）编排」；roadmap §12.4 任务中心可插话 DAG）。设计基线 docs/gaea-office-dag-63-design-2026-09.md（余项续写该档：波内并行/运行中 GaeaSteer 直穿/危险操作分级审批/DAG 模板库）。**架构五层**=dag_plan 规划工具（internal/app/gaea_dag.go，work 空间，PersistWrite()=子代理注册表 FilterRegistry 自动剔除防嵌套派生；ExtraTools 注入与 imageGenTool 同路）→ 存储 internal/gaea/dag/dag.go 纯函数包（Validate 拒环/悬空依赖/空链 fail-closed、Order 拓扑分波波内按 id 排序确定性、Derived 派生态五判序=running>failed>draft>ready>accepted、Store JSON 落 <cwd>/.gaea/work/dag/ 原子写+safeID 拒穿越+SaveNew 防同秒覆盖）→ 执行器（App dagExecute 波次推进波内顺序=产物归因窗口单调不重叠；每节点=TaskTool.RunNew 全新子代理会话——新导出方法+boot OnNodeRunnerReady 钩子镜像 OnFollowUpReady：headless 闸+FilterRegistry+transcripts 落盘自动进既有子代理树/tab，文本增量走 gaea-subagent-text 与追问同路；cancel 登记在受理方同步完成=「登记存在」与「goroutine 在跑」无窗口）→ 控制 7 绑定（GaeaDagList/Get/Run/NodeRun/NodeSteer/NodeAccept/Cancel，613→620，Office 门面手工补行防 gen_bindings 重排噪音）→ 前端 TasksWorkbench「办公流水线」区（DagPanel 新组件：run 卡+节点卡状态点/产物 chips 走 openFilePreview/steer 输入框/重跑/验收；derived=running 每 2.5s 轮询自校正；locale 三语 dag.*；?mock=1 示例 4 节点链可走查）。**关键口径**=节点产物=运行窗口前后 Journal 证据卡 ID 集合差 Target 过滤工作区（主对话同期写盘会并入，UI 口径注诚实不造精确）；验收=人拍板→hubOfficeStore().Save（save 路径自动落 memory_events 成 5.1 图谱实体，6.2 项目本体自然吃到；同节点 id 覆盖=最新产物语义）；重跑把 accepted 退回 done=验收失效诚实降级；终止级联（Cancel→ctx 贯穿子代理→未起跑 skipped）；重启中断懒清扫（List/Get 见 running 无在途→failed「应用重启中断」）；steer=RunFollowUp 续跑管道（followUpClaims 每 ref 单飞）。**测试**=Go +6（校验四态/菱形分波确定性+子集/派生态判序/中断清扫/存储往返+穿越拒绝；fake-runner 3 节点链全生命周期=顺序+产物归因+上游产物进下游 prompt+验收回流记忆+重跑降验收/上游失败下游 skipped/Cancel 级联/未接线 fail-closed/懒清扫/steer 三守卫/dag_plan 拒绝）、vitest +9（DagPanel 空态/重试/显隐矩阵/验收/起跑/steer/预览/轮询起停）。**门禁**=Go 全量 0 FAIL/drift PASS@620/tsc -b 0/eslint 0/vitest 329 文件 2840 全绿/版本三处 4.219.0。**顺带归正既有漂移**=spaceBindings.test 锁 433 实为 435（v4.215 +2 漏更）改锁 442。**坑**=①「go dagExecute 尚未注册 dagCancels」竞态→waitDagDone 假绿，修法=cancel 登记移到起跑方同步完成；②run id 秒级时间戳跨测试撞名+上个测试 goroutine 残留→全局 map 挡住懒清扫，NewID 加进程内原子序号；③Derived 判序 draft 必须排在 failed 之后（fail-closed/清扫场景 RunCount=0 但有 failed）；④测试断言级联终态必须等执行器收尾（dagCancels 条目消失）不能只看派生态。**判据「一条 ≥3 节点链全节点可控」满足——阶段六 6.1~6.6 出口判据全满足，阶段六收官**。

- **最新发布：v4.218.0（2026-09-11）「阶段六 6.6 首刀：跨域 EVM——进度×造价挣值三数两指数」**——施工债级优先可穿插；gaea 独有跨域（进度+造价同在本机一个库），无市对位。TS 纯函数 computeEvm（frontend/src/schedule/evm.ts，零绑定零 Go 改动，mirror dcma/monte 先例）：**PV**=任务预算成本按基线起止线性分摊到数据日期（里程碑基线完成点整额兑现，无基线行的任务不计入）；**EV**=任务预算×完成率（progress/100）；**AC**=手工录入（来源口径注直出「本机无自动实际成本面」，缺失→CPI/CV null 不伪造）；**SPI**=EV/PV、**CPI**=EV/AC、SV/CV 偏差同出；BAC=任务预算合计（固定+资源分配，复用 computeCosts v4.122 成本 rollup，CPM 未过 fail-closed 同口径）；evTop=挣值来源 Top5（transparency）。进度页新增「挣值分析」视图七档（三数大字+两指数+SV/CV+**口径注列表每数一条**+数据日期/AC 录入面；菜单/Segmented/侧栏/ExportDialog 守卫同步）。**测试**=vitest +6（PV 分摊与 BAC/EV-SPI-SV/落后场景 SPI=0 SV=-500/CPI 两侧含 CV=+100/里程碑 dur=0 完成点整额兑现/无基线与循环依赖 fail-closed）。**门禁**=tsc -b 0/绑定 613 零变更/版本三处 4.218.0。**判据「三数+两指数出且口径注全」满足——阶段六 6.1/6.2/6.4/6.5/6.6 判据全满足，余 6.3 DAG 一刀（需先立设计）**。
- **最新发布：v4.217.0（2026-09-11）「阶段六 6.5 首刀：进度·蒙特卡洛工期带」**——nPlan 对位，复用造价价格带哲学「工期也给你三档」。TS 纯函数 monteCarlo（frontend/src/schedule/monte.ts，零绑定零 Go 改动，mirror dcma 先例）：**种子化 RNG**（mulberry32，默认 seed=42，同输入同结果跨会话可比=确定性可测）+ **三角分布采样**（[d·(1−u/2), d, d·(1+u)]，默认不确定度 30%=工期偏乐观不对称先验，UI 滑杆 0~60% 可调，u=0 退化为确定值）+ **逐次重算 CPM**（默认 200 次，planFinish/日历与页面同语义，采样后映射去 O(n²)）；输出=P25/中位/P75 三档（取整）+min/max/均值+10 桶直方图+**关键路径稳定度**（逐任务关键出现频率降序+当前关键任务平均频率=稳定度指标；换线风险可见：计划态非关键但频率>0=可顶掉，当前关键但频率低=不稳）。SchedulePage 新增「工期模拟」视图（三档大字+直方图+频率条+不确定度滑杆；菜单/Segmented/侧栏六档；ExportDialog defaultView 守卫扩展）。**测试**=vitest +8（同种子逐字段一致/三档单调 min≤P25≤中位≤P75≤max+均值≥计划值的不对称性/零不确定度退化+稳定度=1/线性链关键率全 1/并行链计划态非关键任务频率>0 且≤1/直方图频数守恒=有效次数/CPM 断裂 ok=false 不伪造三档/RNG 序列与值域）。**门禁**=tsc -b 0/绑定 613 零变更/版本三处 4.217.0。**判据「三档+稳定度本地可跑（无云依赖）」满足**。
- **最新发布：v4.216.0（2026-09-11）「阶段六 6.4 首刀：进度·DCMA 14 点计划质量体检」**——对标 Acumen Fuse；TS 纯函数 dcmaAudit（frontend/src/schedule/dcma.ts）吃 SchedulePage 前端自算 CPM+基线（与 deadline/drift 同镜像口径，零绑定零 Go 改动）。14 点逐项：1 缺逻辑（首末豁免收紧=孤立任务不得因 es 并列豁免）/2 负搭接/3 正搭接/4 FS 占比/5 硬约束（模型无约束字段恒过+note 诚实标注）/6 高浮时/7 负浮时/8 长工期/9 无效日期（工作日序号口径 ES<0 或 EF<ES）/10 资源缺失（未启用资源维度=诚实 n/a）/11 错过基线（EF 晚于基线 EF 且未完成）/12 关键路径连续性（critical 子图连通分量计数，手动任务分段如实呈现）/13 CPLI/14 BEI（后两者需基线+数据日期；BEI 完成口径=progress≥100 的诚实近似）；超标样本点名上限 5、每点阈值+口径 note；**CPM 断裂时通过率评分置 null（整体分不可信，逐点结论仍在）**。进度页新增「质量体检」视图（DcmaView 纯展示+菜单/Segmented/侧栏五档）。**测试**=vitest +11（十四点逐形态全覆盖）+SchedulePage 菜单用例五档同步。**门禁**=tsc -b 0/绑定 613 零变更/版本三处 4.216.0。**判据「十四点纯函数全测」满足**。
- **最新发布：v4.215.0（2026-09-11）「阶段六 6.2 首刀：记忆驱动项目本体——项目本体注入」**——6.2 出口判据「Plan 注入带决策引用且可关闭」落地（依赖 5.1 图谱底座 v4.210）。**BuildProjectBrief 纯函数**（memory/projectbrief.go）=记忆→项目事实表：固化全收+project/feedback 型按衰减评分降序，每行带 [MEM:name] 稳定引用键（模型采纳句末引用→v4.210 触达/徽标/定稿闸全链同源）+决策来源归因（SourceSession/SourceMessage→「依据 session-x 会话 · turn N」，存量行无来源不造数）；600 rune 预算截行不截半句；无候选返回空串零注入前缀不变。**注入点=work 空间会话装配**（buildSystemPrompt，晨报预载同位）——缓存稳定前缀跨会话项目连续性；BootOptions.ProjectBrief 穿管（mirror MorningPreload，CLI/TUI 维持原行为）。**可关闭**=config project_brief 键（默认开）+GaeaMemoryBrief/GaeaSetMemoryBrief 绑定（appconfig 写+gaeaRebuildLocked 即时生效）+MemoryPanel「项目本体 开/关」胶囊。绑定 611→613（GaeaSetMemoryBrief 按 GaeaSetMemoryEnabled 先例落 Office 门面）。**测试**=Go +2（固化优先+引用键 citations 可解析+归因文案/极小预算零注入+无候选空+确定性）、vitest +1（开关读取+切换持久化）。**门禁**=Go 全量 exit 0/drift PASS@613/tsc -b 0/版本三处 4.215.0。**坑再证**=internal/config（app 偏好 ~/.gaea_config.json）与 internal/gaea/config（引擎 TOML）是两个包，晨报/项目本体开关在前者、经 BootOptions 穿管进装配。
- **最新发布：v4.214.0（2026-09-11）「阶段六 6.1 首刀：可审计默认化——文件预览默认版本条」**——阶段旗交阶段六（书斋纵深·办公主角）。纯前端零新绑定零 Go 改动。**缺口**=VersionTimeline（v4.28 B1 逐版本预览/恢复）只藏在交付面板 vN 徽标里，文件工作台没有一级入口。**落地**=FileVersionStrip 新组件默认可见挂进文件预览标题栏下（三件套通用，数据源与交付面板同源 GaeaJournalList）：版本数+最近 AI 改动时间+复核状态+「可恢复」标注，无版本时显示「AI 改动会自动成为版本点」说明条；展开=该文件逐版本时间线（groupVersionsByPath 按路径过滤，预览基线经 openFilePreview 新开预览/一键恢复走 RollbackRecord+toast+重拉时间线+父级静默重读预览；恢复动作自身生成新证据卡=恢复也是版本）。pptx 页码归因来自证据卡摘要（appendPptxEvidence 的 p3 格式 Before/AfterSummary）。「AI 改动自动成点」=证据链既有能力（v4.1 起）。**测试**=vitest +4（默认可见+展开/恢复贯通 RollbackRecord+onRestored/空态说明条/证据链不可用静默降级）；FilePreview 27 例既有全绿。**门禁**=tsc -b 0/绑定 611 零变更/版本三处 4.214.0。**6.1 判据满足**（默认可见回滚入口+AI 改动自动成点）。
- **最新发布：v4.213.0（2026-09-11）「5.3 首刀：记忆生命周期三态（固化/衰减/归档）」**——「90 天一刀切」替代落地：**固化**（SchemaV19 facts.pinned，绑定 GaeaMemoryPin 609→611）=免疫衰减归档+**豁免保留期硬删**（CleanupArchived `AND pinned=0`，替代核心）+晨报/预载排序加权（morningRecencySort 固化组优先）；与 archived 正交（手动归档可作用固化条、pinned 保留）、Save 覆盖不丢（ON CONFLICT 不触 pinned 列）；已归档不可 pin（固化只作用活跃集合，明确报错）；文件后端不支持=诚实报错（对齐 space_id 先例）。**衰减**=纯函数非存储列：DecayScore（半衰期 30 天指数、下限 0.01、固化恒 1、无时间戳=无衰减证据不造数）+ LifecycleOf（阈值 60 天）；评分输入（save/touch）自 v4.210 全在 memory_events、状态动作 pin/unpin 同日志落事件=「衰减有留痕」。**三态可查**=GaeaMemoryLifecycle 总览（固化/衰减列表带评分+闲置天数+归档计数+保留期+阈值下发）；前端 OfficeMemoryLibrary 统计行（固化/衰减/活跃/归档/保留天数）+ FactCard 固化锁（aria-pressed）+「固化」徽标。「蒸馏 no-op 转真实合并」（DistillMerge）与「预取可关闭」（晨报预载开关）此前已落地不重建——**5.3 出口判据全满足**。测试=Go +5（pin/unpin 闭环+事件留痕+归档不可 pin/固化豁免清理/评分五形态/三态分类/晨报加权）、vitest +1（统计行+徽标+解除）。门禁=Go 全量 exit 0/drift PASS@611/版本三处 4.213.0。
- **最新发布：v4.212.0（2026-09-11）「5.2 首刀：上下文编译 dump/diff + 前缀稳定证明」**——阶段五 5.2 首刀，纯 Go 零绑定前端零改动。意图分类/预算调度底座已有（TCCA L3 SkillLayer.Route=意图分类；doc 预算/compaction=预算调度），本刀补证据链：**消息级编译摘要**（agent compile.go：canonical role+长度前缀+content+toolcalls 逐消息 SHA256 链，长度前缀杜绝「A|BC==AB|C」拼接歧义；toolcalls 参数参与摘要）+ **DiffMessages 相邻请求判定**（Stable/PrefixBytes/AppendCount/RewriteCount；CacheShape V5.10 只看 system+tools 头部，本刀补消息历史半边）。**判定随 RequestHeader 落会话日志**（event.RequestHeaderInfo 追加 compileHash/prevCompileHash/prefixStable*/prefixBytes/appendCount/rewriteCount，追加列旧日志零值=未计算）——日志即装配 dump、相邻记录即 diff、ReadEntriesFor 即回放；contextview RequestRecord.Prefix 随趋势柱下发与 CacheHitTokens 配对=出口判据「相邻两轮前缀字节级稳定（缓存命中可证）」完整证据链，前端渲染按需另刀。测试=Go +4（纯函数五形态/agent 同会话 4 请求跨 2 回合全稳定+摘要链接续/外部改写历史→诚实判漂移+RewriteCount 归因/contextview 折叠贯通+旧日志 nil）。门禁=Go 全量 exit 0/vitest 以 v4.210 基线为准（纯 Go 刀）/版本三处 4.212.0。坑=contextview 折叠里连续 request_header 无 usage 会覆盖 pending（真实日志每请求后必有 usage，测试须补 usage/turn_done 才落账）。
- **最新发布：v4.211.0（2026-09-11）「5.1 收口：回复发出前剥离悬空 [MEM:] 引用」**——v4.210 余项收口，纯 Go 零绑定前端零改动。闸点=agent stream() 收尾：新 `Options.FinalizeText`/`SetFinalizeText` 定稿钩子在 Message 全文事件发出**前**改写答案文本，改写值同进 session 历史与 TurnResult 摘要——幻觉键到不了持久层；前端收 Message 事件整泡替换（既有语义零新事件形态），流式增量原样透传由重渲染收敛。boot 装配闭包（记忆总闸+库可用性门控；子代理/headless nil 不改写）；`memory.Store.StripDanglingCitations` 纯函数=命中键保留/悬空键含紧邻空白整处剥离/**不 Touch**（触达职责仍在回合收尾，两路分离防双记）；被剥离键当场落 dangling cite 事件（v4.210 悬空可观测性不丢）。空间语义同读端隔离器。测试=Go +5（memory 三例：命中保留+悬空剥离+空间隔离/不触达+零值 Store 跳过；agent 两例：Message/Summary/session 三路改写+nil no-op）；门禁=Go 全量 exit 0/vitest 以 v4.210 全绿基线为准（纯 Go 刀先例）/版本三处 4.211.0。**5.1 出口判据全满足，阶段五下一刀=5.2 上下文编译**（规划大纲 §2）。
- **最新发布：v4.210.0（2026-09-11）「记忆语义图谱：事件日志投影出图」**——**阶段五首刀**（规划大纲 §2 主轴旗，底座#1；设计基线 docs/gaea-memory-graph-51-design-2026-09.md，5.2/5.3 在其上续写）。**缺口**=记忆只有存储没有结构：facts 按名存取、MemoryHub 关联图是演示型（同标签拼边无事件事实）、三脑数据不可推理。**落地三层**=① memory_events 事件日志（**SchemaV18**，追加式 INSERT-only：remember 工具/桌面面板/做梦/蒸馏合并/引用触达五条写路径经 sqliteBackend 落库成功即留痕，Touch 逐条落——5.3 衰减的事实源顺带就位；不存全文只带投影元数据+280B excerpt，内容真相仍在 facts）；② ProjectEvents 纯函数投影（实体/事件/来源三向边 produces/affects/references；实体 id=`mem:<project>/<name>`=[MEM:name] 图寻址；last-write-wins 折算；输出全排序→同日志两次投影逐字段一致）；③ mem_graph_* 物化（embedding 向量占位列 5.2 启用；读前水位对账懒重建→**删库可重建**）。**悬空拒写**=references 边两端实体在场才物化，悬空目标不进图但 Dangling 留痕；cite 事件永不建实体（防幻觉键节点化）；回合收尾 ResolveCitationsDetailed 命中+悬空各落 cite 事件。**绑定 608→609**（GaeaMemorySemanticGraph→SemanticGraph 返回 SemanticGraphView 带事件/悬空统计）；MemoryHub 图谱页「关联图/事件图谱」切换（GraphView source prop，默认 cloud 零变化）+domainColors 三新色。**坑**=①「物化≠投影」DeepEqual 双源口径：Tags/Refs 经 JSON 往返 nil↔[]string{} 漂移，LoadEvents 与 MaterializedGraph 双侧归一 nil 才锁死；②gen_bindings 重生成把 office 门面重排序+改 import 别名（v4.178 同坑），只留 memory 新行其余还原。**测试**=Go +9（投影确定性/三向边/悬空不入图/实体fold/删库重建自愈/BFS 遍历含 [MEM:] 键解析/日志读写/写路径逐 op 挂钩/cite 事件形态）、vitest +1（semantic 数据源切换+标题）、tsc 0、既有 GraphView/MemoryHubPage 8 例零变更。门禁=Go 全量 exit 0 / drift PASS@609 / 版本三处 sync 4.210.0。**5.1 余项**=真·发送前悬空引用剥离（需流式层改写最终文本，独立小刀）；UI 遍历路径/手动重建按钮按需另刀（Go API 已备 GraphNeighbors/RebuildGraph）。
- **最新发布：v4.209.0（2026-09-10）「组价含量对照：拆解含量 vs 同类条目分位带」**——造价刀路池 §6 收官，**survey 六项缺口全清**（docs/gaea-cost-domain-survey-2026-09.md 已逐项回写收口状态，勿再当欠账池引用）。**缺口**=AI 拆解人材机含量只有金额自洽校验（v4.158 R1-R3），含量本身离不离谱没人管。**落地**=`cost.CheckContentBaseline` 纯函数（contentband.go）：拆解组件 vs 相似条目池同键含量分位带，**匹配键=标题归一精确相等（全角→半角/去空白/小写）+单位归一相等（含双方都空），一方缺失一律不比——宁缺勿误与导入链同哲学**；P25/P75 R-7 同口径，q>P75 偏高 warn / q<P25 偏低 warn / 带内静默降噪；P25==P75 同值退化带 ±5% 容差（否则同值样本把微小差异全标异常）；同键样本 <`MinContentSamples`(3) 不比对；文案自带口径「含量 420 高于同类 5 例边界 315（+33.3%）」。**接线**=gaea_cost_compose.go `composeChecks` 汇总金额自洽+含量对照两层，对照池=相似条目本身组件明细（检索相似条目=「同类」口径，与价格带同池，逐条 Get 失败跳过池空静默）；Checks 随视图展示+Apply 留痕自动承载，**零新结构零新绑定前端零改动**。**边界**=外部「行业含量区间」数据源明确不引入——基线源=用户自己的库（私人记忆哲学），样本数即诚实口径。**测试**=Go +5（contentband_test 四例：带外双向+归一匹配+单位守卫三态+退化带+守卫；composeChecks 接线：带外双条/带内双静默/池空剩金额层）。**门禁**=Go 全量 exit 0（116 包 0 FAIL）/ drift PASS@608（零绑定）/ 纯 Go 刀 vitest 以 v4.208 全绿基线为准（v4.197 先例）/ 版本三处 sync 4.209.0。**产物**=build.bat（1m02.7s）→ build\bin\gaea.exe **48,580,096 B** + 冒烟 200 + Desktop\gaea.exe 同哈希；SHA256=**637c6d835d6355b62edfffbed68602bda35fec1e78a7a8d1f7b948607e7f9537**（releases/SHA256SUMS-v4.209.0.txt）；未打 tag（随 v4.200+ 现状）。欠账=组价含量对照端到端待真机（观察池）；造价域新缺口另立调研。
- **最新发布：v4.208.0（2026-09-10）「首页 v7 重设计：书斋「文书台」/ 闲庭「游园画廊」」**——v6 版式廉价感定位=「等大圆角卡片+描边」一种形态重复到底。**v7 结构三形态**（仪表条 hairline 行 / 账页等宽序号行 / 海报墙真大小跨格）+ **四档排印**（display clamp 负字距 / lede / body / label 宽字距，数据等宽）+ **三级色调面**（surface→container→container-high，描边退 hairline）+ 版式记号（序号水印 / 月洞门细环）取代极光斑；信息零删除，契约 testid（ml-space-* / desk-recent-docs / garden-*）全保留。**坑：CSS 重写会静默吃掉「同名 class 的规则」**——`.ml-avatar-ai` / `.ml-avatar-user` 在 HEAD 有规则、v7 新版没有（TSX 仍在用），两个头像退化为同底色；**检查配方=脚本双向比对「TSX className 集合 vs CSS 选择器集合」**，本刀跑出 3 处 TSX 无规则（2 真丢 + 1 空挂 `.p-foot-wide`）后修正。附带**门禁可信化**：CI「无条件重试一次」→「只在册 flaky 隔离复跑」（`known-flaky.txt` 现为空 + 分类器自测）、`maxWorkers` 按物理核夹 [4,8]（原按逻辑核开 ~31 worker→重组件首个 `await import` 撞超时假红）、testTimeout 30s、RTL `asyncUtilTimeout` 5s；**教训=afterEach 清 body 会打挂 antd 模块级 message 容器（实测一次 14 例假红）**，残留改由并发上限从源头消除。门禁=tsc/eslint 0 / e-check OK / drift PASS@608（零绑定）/ Go 全量 / vitest 324 文件 2799 例 / 版本三处 4.208.0 / 产物=主树 wails build -ldflags "-s -w" -trimpath（1m08.5s）→ build\bin\gaea.exe 48,570,880 B（46.32MB）+ 冒烟 /api/health 200 + Desktop/build\bin 双副本；SHA256=59c2b518bdb6165ccbac07bbc565759a2d7571f254e287e27aa00b2bccd97a16（releases/SHA256SUMS-v4.208.0.txt）——构建后工作树仍干净（wails 的 npm install 未改写 lockfile）/ 目检=dev(`?mock=1`)+无头 Edge CDP 两空间 ×4 档共 10 张（无横向溢出）+ **壳内实测**（构建产物 `build\bin\gaea.exe` + `WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS=--remote-debugging-port=9333`：书斋/闲庭**浅色主题+真实数据**渲染正常——内核 6 引擎/真实最近文档 5 条/记忆 1706 条/晨报 9 条、越界元素 0、**真点击切换器原地切换** aria-pressed 与 localStorage 同步、无 reload 绕过）。欠账=海报墙首张（小说）大样中段留白偏多（观感待定）；动效手感（入场分阶 / hover 位移节奏）待上手定论（浅色观感已见实机）；真机池余项=.gsched「基线 N」chips / 最近文档 localStorage 写路径 / 冷启动基线采集。


- **v4.207.0 / v4.206.0 / v4.205.0（2026-09-10）主题令牌收口 + 小说书房工坊接线**——v4.207 深色主题硬编码深灰（RelationGraph 图例/提示、进度里程碑菱形与标签）改走 --color-text / on-surface；v4.206 33 处 `bg-accent text-white`→`text-accent-fg`（= on-primary）+ App 注入 --color-on-primary / --color-surface；v4.205 小说书架画廊接线（画廊头 + 正在编辑横条 + 书脊/角标）+ 阅读页属性检查器由 display:none 改默认可折叠（章节体检入口回来）。三刀均纯前端、绑定 608 零变更、Go 零改动。


- **最新发布：v4.204.0（2026-09-10）「组价多方案对照：P25/中位/P75 当场切」**——造价刀路池 §2 半刀。价格带三档数字早就在，推荐价钉死中位数。本刀不拆逐步盖章：ComposeModal 三档格可点（aria-pressed，应用价随档，默认中位数既有用例不动）；cost_compose 增 mode 参数（未知回落中位数）+输出「三档对照」一行。绑定 608 零变更。门禁=ComposeModal 11/11（+1）/Go TestCostCompose 绿/tsc 0/eslint 0/版本三处 4.204.0。欠账=流式打字机/费率政策对照不做；含量基线等样本；壳内真机走查池不变。


- **最新发布：v4.203.0（2026-09-10）「空间策略其余功能域键：总闸当场切」**——v4.190 欠账收刀：七键写路径早已通，编辑 UI 却只放 gaea 主控。本刀纯前端（绑定 608 零变更）把 chat/whisper/novel/office/characterlib/routine 拉进同一张空间策略卡：六键常显（空键灰字「未配置（维持现状）」），行级编辑闭环与 gaea 同款（带出当前值 → GaeaSpaceProfileSet(space, key, ref) → 视图随返回刷新；空串=清除；后端校验失败原样 message.error）。gaea 主控行与既有 testid 不动。阶段二总闸画面出口补齐「书斋和闲庭可以各绑各的」。门禁=StrategySection 8/8（+2）/tsc 0/eslint 0/版本三处 4.203.0。欠账=功能域×空间交叉矩阵按需另刀；壳内真机走查池不变。


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
