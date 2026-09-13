# docs/ 文档索引

> 2026-09-09 立索引（同日状态大清算）；**2026-09-10 复核重建**——45 份顶层文档 + 2 个子目录**全部登记**（补回 7 份漏登记件），状态行与 git log 逐版核对。**归档件在 `docs/archive/`**；发布物全文在 `releases/`；版本动态速览在 `.gaea/AGENTS.md`（其历史版本段已分流至 `docs/archive/agents-version-history-2026-09.md`）。
>
> **维护规则（防孤儿）**：新增/移动文档必须同步本索引；**未登记文档视同孤儿**，下一轮整理按「未登记 = 待归档或待登记」处理。引用任文档前先读其头部状态行。**守卫**：`node scripts/check-docs.mjs`（孤儿登记 / `docs/` 悬空引用 / `.gaea/AGENTS.md` 指令预算 / **含非 ASCII 的 .ps1 必须带 BOM** 四查），已接入 `scripts/ci.ps1` 全门禁。

## 总纲（权威排序依据）

| 文档 | 一句话 |
|---|---|
| gaea-nextgen-roadmap-2026.md | 全域路线图（权威；§16 调研回填后近期落地项见 v4.88/4.89） |
| gaea-next-stage-plan-2026-09.md | 🔄 **下一阶段规划大纲（活跃指导）**：接棒长期规划——阶段五 记忆 OS+上下文编译（主轴）/ 阶段六 书斋纵深（办公主角：可审计默认化·项目本体·DAG；进度 DCMA·蒙特卡洛；跨域 EVM）/ 板块并行池 / 拍板池 / 维持轨 |
| gaea-slim-masterplan-2026-09.md | 🔄 **瘦身长期总规划（活跃权威）**：七面体检×七轨道×六阶段；**P0–P4 已收官**（v4.167–v4.177）+ 清点刀 v4.179；余 P5 维持与观察池审判 |
| gaea-convergence-plan-2026-09.md | 🔄 **收敛期叠加层**：W1 卫生刀 ✅（v4.164/165）、W2 知识外化三件 ✅、W3-4 office 枢纽解耦 ✅（v4.197）；冲突时以「不加新板块」为先 |
| gaea-competitive-landscape-2026.md | 竞品格局与差异化定位（参考基线 2026-08-29，注意时效） |
| gaea-market-survey-2026-09.md | 🔄 **市场调研增量刷新（v4.240 落地）**：八赛道信号 + 三战略结论（Copilot agentic GA 验证形态/LM Studio Bionic 正面竞品/「本地」已非差异点）+ 候选 4 项交拍板池（已过删除史筛）；候选1 双时间轴已落地（SchemaV21），候选2 记忆可评测为下一刀 |
| gaea-mcp-hub-reversal-review-2026-09.md | **MCP 双向枢纽翻案评估（拍板池条2，建议维持排除）**：入方向已在产（boot/plugins.go 配置化挂载+热插拔）；出方向三条独立反对证据同向=工程形态矛盾（GUI 壳 vs 常驻 server）/与自身入口重复/安全暴露面与人设冲突；留单向导出作远期观察不立项 |
| gaea-backend-perf-survey-2026-09.md | 🔄 **Go 后端性能热点普查（2026-09-12，交拍板池刀路 A-E）**：20 项证据 file:line 已核实（回合边界日志全量重读/remember 全局锁内全库重载/cost_search 每查询重建索引/Hephaestus.db 单连接串行读写等）；go vet 全净；与既有条目零重叠；gaea/ai 热路径零 benchmark 需先立基线 |
| gaea-cost-domain-survey-2026-09.md | 造价域现状基线（file:line；**校正 roadmap §15 过时欠账**）+ 造价刀路池：§1 编码 v4.178 / §2 多方案 v4.204 / §3 五算 v4.194 / §4 询价扫描 v4.195 / §5 索引补齐 v4.196；**六项全清（§6 含量对照 v4.209 收官，本档转现状基线不再当欠账池）** |
| gaea-priceband-datasource-research-2026-09.md | 🔄 **价格带数据源调研（拍板池候选4，v4.275 结案建议）**：现状核实=价格带数据面基础设施已完整在产（四要素字段+统计+导入链+OCR 询价飞轮）；外部自动接入四路评估=B 抓取不建议（脆弱+合规灰）/C 商业 API 不做/D LLM 查价不作基线；真机验证抓实锤当场修=信息价「除税价（元）」列适配（fieldPrice 字典+回归测试，12/12 识别） |
| gaea-memory-graph-51-design-2026-09.md | 🔄 **记忆语义图谱 5.1 设计（阶段五域内基线）**：事件日志投影三向图（v4.210.0）+ 发送前悬空引用定稿闸（v4.211.0）——**5.1 出口判据全满足**；5.2 首刀=前缀稳定证明+装配 dump/diff（v4.212.0）；5.3 首刀=三态生命周期（v4.213.0，判据全满足）；6.2 首刀=项目本体注入（v4.215.0，判据满足）；6.4 首刀=DCMA 14 点体检 TS 纯函数（v4.216.0，判据满足）；6.5 首刀=蒙特卡洛工期带（v4.217.0，判据满足）；6.6 首刀=跨域 EVM（v4.218.0，判据满足） |
| gaea-memory-injection-eval-design-2026-09.md | 🔄 **记忆注入质量评测设计（市场调研候选2，v4.241 落地）**：与检索评测分立——对真实库 work 视图跑装配点同款构建器（晨报预载+项目本体），断言预算/泄漏/悬空结构不变量，门槛=零违规；纯核 memory.EvalInjectionBlocks + 绑定 GaeaMemoryEvalRun + 办公记忆库「注入体检」Drawer |
| gaea-office-dag-63-design-2026-09.md | 🔄 **办公多文件 DAG 6.3 设计（阶段六域内基线）**：dag_plan 规划→run 档→拓扑分波执行器（每节点=TaskTool.RunNew 子代理会话）→7 绑定→任务中心流水线区——首刀 v4.219.0 判据「≥3 节点链全节点可控」满足，**阶段六收官**；余项（波内并行/运行中 steer 直穿/分级审批/模板库）在档续写 |
| gaea-outlook-longterm-plan-2026-09.md | ✅ **长期规划（已收官）**：接通已有→总闸画面→造价开口→闲庭同一人设；阶段一~四出口已于 v4.184~v4.196 全数满足；后续阶段由 gaea-next-stage-plan-2026-09.md 接棒 |

## 域方案与域盘点

| 文档 | 状态 |
|---|---|
| gaea-office-upgrade-plan-2026-09.md | ✅ 主体已落地（分期 v4.23–v4.31）；「远期」行（终端 tab/双工作台/侧边对话）维持拍板门控 |
| gaea-office-mindmap-base-design-2026-09.md | 🔄 M1/B1/B2 看板/M2 画布编辑（v4.108.0）已落地；**B2 后半（字段面板/画廊）待拍板** |
| gaea-pptx-edit-design-2026-09.md | ✅ 刀1 v4.109 / 刀2+刀3 v4.156 已交付；刀4 待反馈；真机走查挂池 |
| gaea-novel-revolution-2026.md | 🔄 刀1–6 已落地（v4.77 批次）+ v4 场景制续刀（场景元数据 v4.199）；GenerationGate 闭环/刀7 续/刀8 未落地 |
| gaea-character-domain-survey-2026-09.md | ✅ 阶段四读档（**结论=一套资产 + 项目工作副本**，非三套人）；出口三小刀已落地（外观锚点 v4.192 / 副本回写 v4.193 / 关联即快照 UI） |
| gaea-dream-studio-nextgen-2026-09.md | 材料汇编+下一代草案，不承诺版本（§0.5 已被图域 longterm-plan 吸收） |
| gaea-image-domain-longterm-plan-2026.md | 长期路线：T0 契约已落地（v4.98.0）；T1+ 未启动 |
| gaea-image-domain-t0-contract-design-2026-09.md | ✅ 已落地随 v4.98.0 |
| gaea-sin-board-design-2026-09.md | ✅ **闲庭·原罪板块（v4.256.0）**：对话式图文混杂故事创作——办公组件复用清单、sin 绑定契约（会话/流式/插图/导出）、cue 编号跨端约定、成人内容边界与验收证据 |

## 蒸馏规划（取道不取器）

| 文档 | 状态 |
|---|---|
| gaea-dsh-univer-office-distill-plan-2026-09.md | ✅ 已蒸馏收官（U 系落地 v4.97~v4.109；Univer 零引入） |
| gaea-dsh-genui-distill-plan-2026-09.md | ✅ 已收官（P0–P5 全部发布，止于 v4.97.0） |
| gaea-dsh-better-sidebar-long-term-distill-plan-2026.md | 🔄 滚动权威：阶段一/二/二.5/三(3a/3b) 已全销账；3c 与阶段四/五维持「拍板后/若做」 |
| gaea-unsloth-modelhub-distill-plan-2026-09.md | ✅ 主体已落地（v4.102~106）；仅真机补验池观察项（CU1 收益/CU3 /free） |

> dsh-better-sidebar-go-port-plan-2026-09.md（整包 Go 化方案）已归档至 docs/archive/——路线未采纳，被滚动蒸馏规划取代。

## 进度计划域（16 项差距已全清收官；观察池=真机项）

| 文档 | 状态 |
|---|---|
| gaea-schedule-gap-vs-project-2026-09.md | **收官总账**：16 项立账→销项版本对照（v4.132~139） |
| gaea-schedule-diff-confirm-design-2026-09.md | ✅ 已收官（四刀 v4.146~149） |
| gaea-schedule-dual-duration-design-2026-09.md | ✅ 已收官（四刀 v4.150~153 + 搭接放开 v4.155） |
| gaea-schedule-projectlibre-distill-2026-09.md | 机制供给参考（蒸馏成果已全部落地） |
| gaea-schedule-standards-digest-2026-09.md | JGJ/T 121-2015 规范权威底稿（缺陷清单已修，v4.129/130） |

> 已实施设计（多工程/资源成本/AOA 手动布局）与 2026-09 市场调研合成版已移入 docs/archive/。

## 审计与基线证据层（读档刀的证据物；结论已进 AGENTS，此处留证）

| 文档 | 状态 |
|---|---|
| gaea-slim-baseline-2026-09.md | ✅ P0 基线证据层：七面四表实测（dist/entry/MemoryHub/exe/依赖/>50KB 源文件）+ 功能等价快照 + IA 走查 |
| gaea-slim-knife2-audit-2026-09.md | ✅ 审计证据：**G-2 已关**（v4.169「基线 N」chip）；G-3 真机走查挂池 |
| gaea-slim-p2-dualspace-2026-09.md | ✅ 审计证据（P2 双空间并列）；权威=masterplan §轨道一·2/3 |
| webview2-shell-audit-2026-09.md | ✅ 审计完成（P0×2/P1×7 + 刀序 A–D）；**刀 A–C 已修**（v4.165/166）；刀 D（printSvg print/拖拽/粘贴）真机取证挂池 |
| gaea-office-hub-decouple-audit-2026-09.md | ✅ 已收刀（v4.197：`office/docmd → internal/docmd`，内核内边清零）；遗留候选（别名门面/archive.go 死码候选/度量口径）挂观察池 |
| distill/ | 🔄 MuMuAINovel 蒸馏规格与实施交接书（01~07 六域 + `09-impl-handoff.md`）；**t1 共享契约已落库（v4.278.0）**，t2~t7 未实施 |
| gaea-sin-booksource-distill-2026-09.md | 🔄 原罪·书源引擎蒸馏规格（so-novel 机制重推导；AGPL 红线=不搬代码与规则 JSON）；并行线在制品，代码落点 `internal/booksource/` |
| gaea-novel-ohstory-distill-2026-09.md | 🔄 小说板块·oh-story-claudecode 蒸馏规格（MIT：机制重推导 + 知识资产带许可收录）；技能×gaea 映射 + 六刀刀路，首刀=T1 评审 rubric 数据化 |

## 上手与工装

| 文档 | 状态 |
|---|---|
| gaea-getting-started-30min-2026-09.md | ✅ 30 分钟上手（2026-09 定稿）：从零到「能安全改代码并发布」的每日动作，权威细节指向 AGENTS + masterplan |
| 2026-08-14-sandbox-environment-notes.md | 沙箱环境备忘（详细版；AGENTS 中为四条铁律摘要） |
| snapshots/ | 目检截图 12 张（侧边栏/浏览器面板证据，2026-09-03；正文引用见 webview2-shell-audit 与 releases/v4.101~105） |

## 已收官历史设计（留原位：代码注释仍引用为设计出处）

gaea-space-shell-design.md（S2.1，v3.9.0）· gaea-space-dimension-design.md（v3.8.0）· gaea-space-assembly-design.md（v3.8.0）· gaea-memory-isolation-design.md（S1.2，v3.8.0）· gaea-page-migration-design.md（P1，v3.9.0；挂账项以 roadmap 为权威）· gaea-edit-tools-design.md（五工具在产，S0.6 起）· gaea-genui-memoryfence-audit-2026-09.md（审计存档；6/7 项 v4.101 收口，resume 槽位口径=唯一开放项）

## 政策 / 约定 / 数据集

ADULT_MODE.md · DREAM_WRITE_POLICY.md · MEMORY_ARCHITECTURE.md · evaluation-set.md · retrieval-eval-set.md（12 条查询集，代码运行时直接解析）· ilink-non-text-protocol.md · 2026-08-15-gaea3-architecture-design.md（历史基准：内核架构事件日志/Manifest/Seam，仍有效）

## 归档区（不在此处展开）

`docs/archive/` 收录历史调研、已落地计划、被后续结论取代的文档（含 8 个 research-2026-09-* 原始稿目录、market-research 系列、superpowers 计划/规格、gaea2/gaea3 时代文档、`agents-version-history-2026-09.md` 版本磁带与 `progress-history-2026-09.md` 进度磁带）。**索引见 `docs/archive/README.md`**——引用归档结论前先确认未被现行权威取代。
