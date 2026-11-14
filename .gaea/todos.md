# 任务进度

> 最后更新: 2026-11-14

| 状态 | 任务 |
|------|------|
| ✅ | **上游 DNA + 命令历史分页收官**（2026-11-14）：批量 `GaeaContextDna`（128 seq/单次折叠/窗口缓存/逐工具 schema）+可见窗口防抖加载；`GaeaContextCommands`（游标分页/合并统计）+“加载更早”；绑定面 751；前端 105 例、Go 定向、tsc/ESLint、亮暗 DNA 走查、Vite 32.85s、Wails 1m22.507s 与 `/api/health` 200 全过；exe `E12BDCF4…8E6`，未装机 |
| ✅ | **上游趋势/事件/附件第二批**（2026-11-14）：增量七分类正负堆叠+零轴、自适应可见窗口、自动总览阈值与手动粘性、最新定位/钉住居中、总览键盘下钻、事件行趋势/节点下钻、按会话滚动恢复、附件按正文顺序混排；相关 88 例、tsc、ESLint 0、Vite build 33.86s、亮暗 mock 走查绿；余=趋势 DNA、命令历史分页、真实会话全链与桌面装机 |
| ✅ | **上下文浏览器 Map + 七分类全量收口**（2026-11-14）：token 面积 treemap、小格降密度但保留 aria/title、鼠标与键盘激活切树展开；修复 skill 漏算上下文总量/百分比；三语补齐；相关 75 例、tsc、改动面 ESLint 0、Vite build 33.21s、暗亮 mock 与键盘走查绿；余=真实会话全链与桌面构建装机 |
| ✅ | **dsh-context 0.84.1 上下文交互续刀**（2026-11-14）：趋势四分位/分类联动/请求预览、Step New-in；浏览器分类搜索/名称序/delta/DNA 搜索与引入步骤/request_header system+tools schema；统计、事件、文件、命令下钻；命令结果展开；亮暗 mock 走查；最终相关 UI+mock 契约 81 例、全量 tsc、Go contextview/app、Vite build 绿；余=历史步骤 surface、工具来源 chip、非图片附件卡 |
| ✅ | **DSH Context / Fleet UX 对齐 + Git 打包**（2026-11-13）：按 dsh-context v0.84.1 源码重排上下文页、命令/文件/事件卡；输入框仅对话显示；Agent 网络迁编队并改卡片拓扑；右栏任务 Tab 与旧持久化链路退役；全部未提交源码/文档/截图提交 `18fb5dcb`，未提交快照 ZIP + 源码 TAR + 文件/哈希清单入库；相关 161 例、tsc、ESLint、前端生产构建与亮暗 mock 走查通过 |
| ✅ | **灵犀输出 UI batch3**（2026-11-13，代码候选）：办公交付物改为 2 列灵犀文件成果卡；ContextPill 并入输入卡底栏；新会话空首页隐藏停靠条并改 GAEA 同行标题；输入框同底悬浮；停靠条/右侧产物面板统一类型卡；定向 53+19+15+50 例、ESLint、构建与浅暗主题走查通过 |
| ✅ | **batch3 桌面构建**（2026-11-13）：完整 Wails 绑定构建成功（1m23.122s），三 DLL 同目录，exe SHA256 `52D605F2…6B965`，正式路径冒烟 200；未装机 |
| ✅ | DSH-better-sidebar Go 增量复刻 F1：文件工作台 Go 权威层、17 个 Wails 绑定、文件树 CRUD/多选/上传/ZIP/watch、文本 CAS/编码保持/分页、tab 调解、mock 与测试；随 `f49ea150` 提交并进入预览包 |
| ✅ | 建立长期迭代交接档 `docs/gaea-better-sidebar-go-iteration-2026-11.md`，同步 AGENTS 入口、docs 索引、progress 与 todos |
| ✅ | F0：`f49ea150` 功能提交 + 独立干净工作树构建（Vite 20.38s / Wails 50.158s）+ 正式路径冒烟 200 + 预览 ZIP/源码 TAR/哈希/包记录/记忆更新 |
| ⏳ | 用户决定是否 push（当前 ahead 25：better-sidebar 全清+卫生批+验证扫+装机；golangci/eslint 已清零，push 即绿） |
| ⏳ | **发版门欠账（2026-11-13 立账）**：v4.530.0~v4.543.0 共 14 已装机版本零 CHANGELOG 版本段/零 releases 发布说明（releases/README V4_RECENT 也停在 v4.529.1），17 个「未发布」段+WHIRL 族六版 CHANGELOG 全缺待折；wails.json 抬号史权威考古表已立 day-log 2026-11-13 §13——发版时按表折段+补 14 版说明（release.ps1 骨架+人手收尾） |
| ✅ | G1.6 worktree 只读展示+Git 子进程预算收口：`GaeaGitWorktrees` 绑定+GitStatus linked 标注+GitPanel 第四镜头；git 调用统一超时（读 20s/写 60s）+输出预算（8MB/1MB/diff 4MB 截断）；设计判定=切换归工作区切换器产品门、服务扩展不重做；分支 `feat/g1-git-deepening`（自 `de283ace`）Go/tsc/vitest/drift 全绿待并入 |
| ✅ | G1.6 已快进并入 main（205c5947，本地未 push） |
| ✅ | G1.7 冲突/重命名/二进制展示策略收口：GitFileStatus.conflict 六形态+origPath 溯源+冲突 staged=false；GitPanel 冲突独立分组（不代解）+rename `← 原名`；buildGitDiff 二进制诚实占位；gitStatusInk 六形态补齐；已快进并入 main（277dc240+7b7f314d） |
| ✅ | G1.8 降级矩阵+真机走查收官：空仓（无提交 HEAD）测试（reset/diff --cached 按空树语义实锤）；沙箱夹具真仓+GAEA_WALKTHROUGH+GAEA_WEBVIEW_DEBUG CDP 实壳走查——冲突分组/rename 溯源/暂存计数排除冲突/工作树镜头全断言绿零缺陷；G1 验收矩阵全部真仓测试+真机走查双实证；wails build 19s+冒烟 200 |
| ✅ | G2 只读提交图：GaeaGitGraph 分页（parents/装饰/分支清单+注入校验）+GaeaGitCommitDetail（numstat/diff 预算/合并诚实空）+commitGraph lane 纯函数+CommitGraphView 第五镜头（SVG lane/装饰/筛选不 checkout/加载更多/详情展开）；绑定 746→748；已并入 main，**真机走查已过**（合并拓扑夹具实壳断言全绿） |
| ✅ | 前端测试成本线收口提交 `de283ace`：vitest 双项目 logic=node/ui=jsdom，全量复验 497 文件/4348 用例绿 393.6s（基线 569s）；余项 --changed 回路+小文件合并仍挂拍板 |
| ✅ | F2.1：文件树 Git 状态着色——X/Y 文件徽标、语义色、目录/根汇总点、30s/聚焦/事件刷新；定向 23 用例与 ESLint 绿（随 `f49ea150` 提交，真机待走查） |
| ✅ | F2 文件树完整度：默认/用户排除规则与树/@搜索/watcher 同源、单目录链折叠、完整文件类型图标、打开方式子菜单；定向 53 用例/tsc/vet/ESLint 绿（随 `f49ea150` 提交，真机待走查） |
| ✅ | F3 预览编辑对齐：JSONL 记录卡、Markdown 浮动目录、本地图片解析、脏稿关闭/导航确认、超长 diff 行预算；定向 99 用例/tsc/ESLint 绿（随 `f49ea150` 提交，真机待走查） |
| ✅ | G1.1 统一 Git 前端：单一 `gitStatusStore`（30s/focus/in-flight 去重）+ FileTree/GitPanel 共用状态 + Git/本轮文件双镜头；定向 29 用例/tsc/ESLint 绿（随 `f49ea150` 提交） |
| ✅ | G1.2 Git 仓库根语义：父级仓库发现、嵌套工作区状态归一/外部过滤、porcelain -z rename 与无提交分支解析；App Git 定向测试+vet 绿（随 `f49ea150` 提交） |
| ✅ | G1.3 统一只读变更树：Git+Agent 路径合并、目录 rollup、暂存/工作区/AI 来源徽标与文件打开；定向 12 用例/tsc/ESLint 绿（随 `f49ea150` 提交） |
| ✅ | G1.4 统一树快操作：全部/Git/AI来源筛选 + stage/unstage/两击 discard 直连既有 Git 绑定；定向 14 用例/tsc/ESLint 绿（随 `f49ea150` 提交） |
| ✅ | G1.5 统一行内 diff：已跟踪文件行展开暂存/工作区 diff，双侧面可切换；定向 16 用例/tsc/ESLint 绿（随 `f49ea150` 提交） |
| ⏳ | G1 深化余项：无 git 的更多能力级降级、危险操作边界再补强（G1.6 worktree 展示+超时/预算、G1.7 冲突/重命名/二进制展示策略均已收口，见上） |
| ⏳ | G2/W1/X1：只读提交图、底部工作台/终端、插件服务与开始页接管，均按用户拍板门控 |
| ⛔ | 侧聊服务 / 侧边对话——用户已明确不做，除非未来重新拍板 |
| ✅ | **五卡实弹反馈三刀**（2026-11-13）：过程卡下置（答后段 tailProcess 标志）+交付卡跨段去重（同轮首现段 ownership）+提炼记忆弹窗（remember/fact_add 行点击 MemoryPeekModal）；exe 35994246 冒烟 200 装机；提交 cacb011c |
| ✅ | **办公对话输出重设计批（五卡）回归 main**（2026-11-13）：office-conversation-redesign-wip 分支 d3342e77 合并 562d4eb0（11-12 清场保全的另一执行体休眠在制；仅 progress.md 一处冲突两侧保留）；五卡 35 例+Transcript+投影共 121 例绿+tsc+构建装机 exe 365a65d4；分支已删；卡片视觉走查挂用户 |
| ✅ | **工作态头部删「思考中…」回退文案**（2026-11-13）：无 phase 运行态只留图标+用时+步数；有 phase 仍显示；exe 8ec56ed2 装机；提交 ded8e63e |
| ✅ | **交互文件固定落点**（2026-11-13）：office_open path 归一（省略→「交互文件」目录+时间戳命名；纯文件名收编；带目录尊重；Execute/WritePaths 同判据+回执告知）；exe 7dc655b6 冒烟 200 装机；提交 e382b9f1 |
| ✅ | **五卡二轮实弹四件**（2026-11-13）：下置撤销（用户改主意恢复 Codex 语义）+折叠态最近三条活尾巴+交付卡滤 .py+提炼记忆卡（notice→结构化卡+弹窗查看/接受/忽略）+溢出归档（dream-pending-overflow.json FIFO 200，修复队列满数据丢失）；exe adc4bc4e 冒烟 200 装机；提交 c5ee0dbd |
| ✅ | **COM 档二轮打包 officecom2**（2026-11-13）：干净树 .worktrees/pkg-e0461f53 定提交构建，装机=包版 exe 4139E78C 冒烟 200；交付物 releases/gaea-officecom2-20261113-e0461f53/（PACKAGE-INFO+SHA256SUMS 入库，zip/tar 留盘）；包相对 officecom 增量=五卡回归+二轮实弹四件+交互文件固定落点/语义命名+折叠细化+底色透明+思考中删除+doc 键+Excel 保护；工作树已回收 |
| ✅ | **dsh-office 四波收官打包**（2026-11-13）：干净树 .worktrees/pkg-f5576986（junction+定提交构建）→ 装机=包版 exe 21BD0668 冒烟 200；交付物 releases/gaea-officecom-20261113-f5576986/（PACKAGE-INFO+SHA256SUMS 入库，zip/tar 留盘）；工作树已回收；包记录 34de50d8 |
| ✅ | **dsh-office 波3.5 收官=全项目收官**（2026-11-13）：preferLive 引导回（探测下沉 comhelper+boot 提示词分工段，SingleModelPrompt 逐字节红线不动）+unseen 交付前总览闭环（edit 置位/render 清除/save 自动总览+Final check）；exe 377175df 冒烟 200 装机；提交 f5576986；余=真机实弹挂用户 |
| ✅ | **dsh-office 波3 收官**（2026-11-13）：office_com 设置段五键（OfficePanel「实时编辑」节）+settle 钩子（TurnDone 槽+活动性守卫+静默文档 Journal+office_file_saved）+壳内实时编辑状态条（OfficeLiveBadge 挂 FilePreview 头=迷你卡替代不启 WinForms）；exe 728e79e6 冒烟 200 装机 PID 36072；提交 48cfa091；余=preferLive 引导回+unseen 总览挂拍板池，真机实弹挂用户 |
| ✅ | **dsh-office 波2 收官**（2026-11-13）：WPS math 腿（internal/office/omml 子集直转+office_edit math 接线；接管崩溃子代理修 stopAmp 死循环+emit 缺口两 bug；suite=wps 实弹 2 equation(s) 全建成）+Excel 扩面（xlsxedit 扩清洗 ops 九类+xlsx_profile 只读工具+三处纪律）+PPT 版式引擎拍板取消（helper 保真不裁+教学面只发基础 ops）；exe bb8befa6 冒烟 200 装机 PID 47776；提交 7b793944；当日账=day-log §16 |
| ✅ | **dsh-office COM 实时协同档波1 落地+装机**（2026-11-13）：上游 v1.2.0（MIT）蒸馏——comhelper 包（OfficeHelper.cs 逐字节照搬+csc 编译缓存+/codepage:65001 偏离+Go 行协议 client）+八工具条件装配（三处纪律全走）+save 落 Journal+工作区边界单点收口；捎带修 publishguard 账本 0 字节静默失写+workbench ms 粒度 CAS 失明两生产 bug；真机冒烟全链过（WPS 附着/中文无 mojibake）；exe dfd1db30 冒烟 200 装机 PID 44252；余=波2/3 挂拍板池（WPS 公式腿/PPT 版式引擎/体验件/引导回）；调研档=docs/gaea-dsh-office-distill-research-2026-11.md，当日账=day-log §15 |
| ✅ | 引擎实升最新版：llama b12110（+49%，80.1 tok/s）+Strata v0.2.13/e0.1.59（42.7 带内无回退）+strata 版本口径修（资产名抓组件版本），v4.538.0 装机 PID 35608（2026-11-11 夜） |
| ✅ | 实弹反馈三连修：联网开关默认关+持久化（根治工具无感注入）、思考流实时上屏、三引擎官方版本检查（EngineUpdateCheck+30min 缓存，v4.537.0 装机 PID 8996，2026-11-11 夜） |
| ✅ | WHIRL 推理引擎接入 gaea（第三受管本地引擎 8126，v4.536.0 装机 PID 34236）：对擂 1.91x→盲测定路线→whirlfix2 张量手术→代码面全链+全测绿+tianxuan 残留清理（2026-11-11 晚） |
| ✅ | knip 未用导出尾款清零+天璇退役漏网收口（169 项自然排干至 3 后清零；c7da4558，2026-11-11） |
| ✅ | 读取桌面施工日志并抽取实际进度 |
| ✅ | 对照既有汇报口径，标出差异 |
| ✅ | 按日志修订施工过程工作汇报 |
| ✅ | 回读验证并交付 |
| ✅ | 众神殿 R6 评测底座：同题聚合、只读指标投影、20 题门禁报告与可复算导出 |
| ✅ | 众神殿 UX/UI：议事厅/协议对照双入口、纪要四视图与各席独立窗口 |
| ✅ | 运行监测 UX/UI：调用统计主工作区、模型筛选排序、范围 KPI 与错误/耗时分析；桌面构建、截图和文档记录完成 |
| ✅ | 编程板块接入隔离 DSH Web `0.2.2-rc.1` 与 REA `6.8.0` MCP/技能，并验证 229 工具 |
| ✅ | 语音交互窗口记忆中枢化：全窗居中虚空大脑、完全透明右侧转写、居中胶囊控制、双主题与中央持续字幕；干净提交构建、冒烟、预览包、源码归档和记忆记录完成 |
| ✅ | 迁移桌面版 9 个外部插件依赖，启用 8 个兼容 bundle；`dsh-ops` 升至适配版本 |
| ✅ | 根修 GPU 遥测 PowerShell 与 dshmarket restart helper 的 cmd 黑窗，并完成构建装机 |
| ✅ | 专项文档、构建指南、隔离说明、progress 与 todos 记忆更新 |
| ✅ | 办公板块工具治理第一批六阶段一天收官（P0 地图/P1a/P1b/P2 两波/P3 谓词 harness/P4 office_batch；九笔推 origin+装机+沙箱 3.5/5 面与 P2 矩阵 4 种子实弹） |
| ✅ | dsh-ops 插件蒸馏三轮+三波落地收官（A1'-D7 候选池全清，装机 317568d4） |
| ✅ | 代码面自主批三刀（2026-11-10 晚）：空间契约漂移对齐+双守卫、REA preflight 改查红队预设、版本三源对齐 4.535.0 |
| ⏳ | 沙箱机制实弹五面（面 1 bash 拦截+office_batch 提交腿需登录态）+九格真实脱敏件采集（需用户授权）——办公治理 P2 余项 |
| ⏳ | A/B 评测实弹（GAEA_AB_EVAL 门控，用户引擎恢复后） |
| ⏳ | 语音 tap 撤除+只播最终 rev（待用户拍板） |
| ⏳ | CAD 识别/网络搜索代理 x509 真机走查（待用户配合） |
| ⏳ | V5 Case OS 立项拍板（docs/gaea-v5-case-os-proposal-2026-10.md） |
| ⏳ | 采集 20–30 个真实议题，补齐单席/独立多答/圆桌/定向质询（可选红队）并做人工质量评审 |
| ⏳ | 安装并配置 Hindsight 后补录「众神殿四框架机制落地 2026-11-03」文档记忆 |
| ⏳ | 按需安装 Ghidra/IDA/JADX/Kuna，补齐 REA 原生反编译 provider |
| ✅ | dshmarket 上游跟踪销号（2026-11-13）：**1.73.19（2026-11-12 发布）lib/restart.js 已含 `windowsHide: true`**（detached spawn 行+注释实锤，npm pack 核验）——本地 `dshmarket@1.73.15.patch` 转为可退役；升级 dshmarket ≥1.73.19 时摘 patch（C:\AI\dsh 用户资产不动，留用户升级时执行） |
| ✅ | 办公对话窗顶栏合并（2026-11-13）：ChatTabs 并入单行顶栏+删模型切换/紧凑钮+右侧面板钮移最右换 PanelRight 图标+捎带修 ApprovalModal aria-modal 回归（d3342e77 引入，守卫复绿） |
| ✅ | **灵犀蒸馏 P0 批接续收官**（2026-11-13，接另一执行体停点）：调研三档入库（re-distill 事实源+advantages 19 条+optimize plan）+re-lingxi 脚本报告+wpsdock 探针；P0-1 交付契约硬化（parseFilePathHref 伪协议+delivery_prompt 交付收尾段）/P0-2 office_macro 受控宏 A 档/P0-3 office_selection 选区上下文/P0-4 四条提示词补齐/P0-5 steer 已有不立项；Go 五包+fileLinks 20 例+tsc -b 绿+wails build+冒烟 200+装机 PID 47776；余=P1 挂拍板（阻塞=Python 运行时决策）+#12 全量参数错误文案挂池 |
| ✅ | **P0-4 余项补完**（2026-11-13）：#12 参数错误三要素高频面（办公 COM 族 8 工具+文件族 10 工具统一「工具名+参数解析失败+json 字段路径」+双守卫测试）+#14 五工具 doc 参数工作区约束显式文案；exe 19:23:26 冒烟 200 装机 PID 33768；余=非高频 ~19 处留池 |
| ✅ | **P1 四项全落**（2026-11-13，拍板=复用系统 Python）：P1-2 xlsx_recalc+xlsx_audit 交付门（LibreOffice 链已在，formulas 弃用）+P1-1 pdf_cli.py 17 子命令（实弹 17/17）+P1-3 pptx-template 五步流水线技能（实弹全链绿）+P1-4 WPS 停靠 dock 档（wpshost+三绑定+WpsDockControl，绑定四件套再生）；装机两次（PID 33768→44752）；同树并跑：编程板块（隔离 DSH；〔更正〕此前误记 codex——codex.exe 是其沙箱辅助进程）继续逆向 B/C/D 批，其文档在途改动留树未卷；余=真机停靠走查+embed 档拍板 |
| ⏳ | **灵犀蒸馏第二批**（2026-11-13 立项，待拍板排刀）：计划档 docs/gaea-lingxi-batch2-plan-2026-11.md——对评估报告逐项对账实码（xlsx 三条/pdf_cli/pptx-template/recalc+audit 四项已落销项）后立项 B1 文案档（交付两条时序禁则+技能刷新契约+xlsx 三条提升决策〔推荐维持〕）/B2 失败语义族 10 类/B3 提示词装配顺序评估（SingleModelPrompt 逐字节红线不动）/B4 记忆三层对照评估（MEMORY-spec 72KB vs dream+事实库+MemoryHub）/B5 技能门禁补缺；拍板池=MCP dispatcher 形态（倾向单 dispatcher）/记忆落地/embed 档/技能市场 |
| ✅ | **batch2 五项收官**（2026-11-13，全批放行）：B1 交付两条时序禁则+技能刷新契约+B2 失败语义族（COM 族 officeComWrapErr 单点收口保 Unwrap 链+batch/vision/ocr/recalc 禁编造指令）+B3 装配顺序对照表（不调序+A/B 设计）+B4 记忆三层对照评估（三条纯文案刀待拍板）+B5 docx/pptx-template 门禁补缺；装机 PID 25684 冒烟 200；余=dispatcher/embed/记忆三刀/pptx 视觉验收实弹挂拍板与用户 |
| ✅ | **P1-4 dock 撤项定案**（2026-11-13 晚）：用户实测翻车（真凶=最小化窗 SetWindowPos 不解除最小化，SW_RESTORE 后实测精确落位）+编程板块新产 REPORT-右侧面板WPS编辑链路.md 定案=灵犀面板是 kdocs 云 iframe（绑云不做）非原生嵌入（符号表零窗口操作）；revert ec3f4f48+绑定回退 748+plan 改判；gaea 本地等价已存在（FilePreview+COM 实时档+OfficeLiveBadge）；教训=蒸馏前先查上游有没有该形态 |
