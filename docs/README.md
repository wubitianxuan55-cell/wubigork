# docs/ 文档索引

> 2026-09-09 立索引（同日状态大清算）；**2026-09-10 复核重建**——**2026-11-03 计数校正：105 份顶层文档（守卫口径，不含本 README）+ 4 个子目录（`archive/` `distill/` `snapshots/` `code-audit-2026-10-02/`）全部登记**（计数以 `node scripts/check-docs.mjs` 输出为准，勿在文中硬写）。状态行与 git log 逐版核对。**归档件在 `docs/archive/`**；发布物全文在 `releases/`；版本动态速览在 `.gaea/AGENTS.md`（其历史版本段已分流至 `docs/archive/agents-version-history-2026-09.md`）。
>
> **维护规则（防孤儿）**：新增/移动文档必须同步本索引；**未登记文档视同孤儿**，下一轮整理按「未登记 = 待归档或待登记」处理。引用任文档前先读其头部状态行。**守卫**：`node scripts/check-docs.mjs`（孤儿登记 / `docs/` 悬空引用 / `.gaea/AGENTS.md` 指令预算 / **含非 ASCII 的 .ps1 必须带 BOM** 四查），已接入 `scripts/ci.ps1` 全门禁。

## 工程与构建

| 文档 | 状态 |
|---|---|
| gaea-build-guide.md | ✅ **日常构建与装机权威**：frontend 预构建、裸 Wails strip 构建、DLL 布局、冒烟与装机命令；含 TMP/TEMP、孤儿构建进程、编程板块 DSH 环境以及 Windows 子进程必须 `CREATE_NO_WINDOW` 的防 cmd 黑窗规则 |

## 总纲（权威排序依据）

| 文档 | 一句话 |
|---|---|
| gaea-nextgen-roadmap-2026.md | 全域路线图（权威；§16 调研回填后近期落地项见 v4.88/4.89） |
| gaea-next-stage-plan-2026-09.md | 🔄 **下一阶段规划大纲（活跃指导）**：接棒长期规划——阶段五 记忆 OS+上下文编译（主轴）/ 阶段六 书斋纵深（办公主角：可审计默认化·项目本体·DAG；进度 DCMA·蒙特卡洛；跨域 EVM）/ 板块并行池 / 拍板池 / 维持轨 |
| gaea-stage7-plan-2026-09.md | 🔄 **阶段七规划（平台收口与复利层，2026-09-15 拍板启动）**：7.1 信任地基（记忆评测+路由学习）→7.2 技能结晶双入口（演示式录制+journal 历史蒸馏）→7.3 任务空间制收窄版（多入口收件箱+板块降视图，纯同步零常驻）→7.4 闲庭清欠并行线；红线含「不需要昼夜运转」新拍板 |
| gaea-longterm-plan-2026-10.md | 🔄 **长期规划 2026-10（阶段八起，v4.479 基线，活跃指导）**：阶段八 一致性即产品（小说 8.1~8.8：实体卡注入/状态化实体卡/组装器/定稿回写/章级门禁/伏笔预警/防重复/评测常驻）→ 九 上下文即预算 → 十 本地智能 → 十一 垂类纵深；在册边界+拍板池 14 天过期 |
| gaea-v5-case-os-proposal-2026-10.md | 🔄 **V5 架构立项建议（2026-10-29，待拍板）**：市场/社区/GitHub/V4 审计四路取证 → 判断 V4 结构触顶（686 绑定/48 工具/60 表/无 case 维）；提出「事项—事务—证据」主轴（案头/Case OS）、四层架构、R0–R3 风险与监督预算、48 能力契约、SQLite 三表持久执行、6 周 P0 + 12 个月路线与 kill gate；含备选方案 A–E 与 8 项拍板 |
| gaea-v5-zcode-debate-2026-10-29.md | 🗣 **V5 立项辩论（进行中）**：Codex 主张只批六周 Case Kernel P0，暂缓 V5 唯一主轴、12 个月路线与 P1 冻结；与 ZCode 按只追加协议逐轮核验事实、指标与真人使用门槛 |
| gaea-pantheon-market-research-2026-10-29.md | 🏛 **众神殿市场调研 + 圆桌讨论 V2（2026-10-29）**：一级板块、Grok/Codex/ZCode/DSH 参与者（Grok 为 Gaea 办公席位）、手动 2–20 轮或不限轮次至全员共识、顺序轮换、工作空间范围、实时工具轨迹与持久化纪要已实现；含竞品/论文证据、公正协作协议与验证记录 |
| gaea-pantheon-v2-roadmap-2026-11-03.md | 🏛 **众神殿 V2 讨论协议整合报告（2026-11-03，活跃指导）**：分歧台账与真相单源化 → 动作/目标贡献/DID 收口 → 独立确认与只读行动清单 → 只读协议评测、20 题门禁与四视图议事界面；整合市场、Orca 与四框架专项调研，明确 144 配置不上 UI、投票不等于事实、执行须另签 |
| gaea-slim-masterplan-2026-09.md | 🔄 **瘦身长期总规划（活跃权威）**：七面体检×七轨道×六阶段；**P0–P4 已收官**（v4.167–v4.177）+ 清点刀 v4.179；余 P5 维持与观察池审判 |
| gaea-convergence-plan-2026-09.md | 🔄 **收敛期叠加层**：W1 卫生刀 ✅（v4.164/165）、W2 知识外化三件 ✅、W3-4 office 枢纽解耦 ✅（v4.197）；冲突时以「不加新板块」为先 |
| gaea-competitive-landscape-2026.md | 竞品格局与差异化定位（参考基线 2026-08-29，注意时效） |
| gaea-market-survey-2026-09.md | 🔄 **市场调研增量刷新（v4.240 落地）**：八赛道信号 + 三战略结论（Copilot agentic GA 验证形态/LM Studio Bionic 正面竞品/「本地」已非差异点）+ 候选 4 项交拍板池（已过删除史筛）；候选1 双时间轴已落地（SchemaV21），候选2 记忆可评测为下一刀 |
| gaea-platform-market-survey-2026-09-15.md | 🔄 **平台级市场调研（阶段七前置，2026-09-15）**：三路并行调研——大厂 agent 任务/技能接口（SKILL.md 成标准·历史蒸馏无人做）+本地生态×记忆×路由（程序性记忆空位·路由不进本地模型）+中文创作/陪伴/图像/监管（网文 AI 检测红线·拟人化新规·成人向刑事执法活跃期）；候选提名表六项交拍板池（①技能结晶②任务收件箱③路由学习④记忆评测⑤自证创作〔挂起〕⑥审计主权口径） |
| gaea-competitor-survey-2026-10.md | 🔄 **2026-10 竞品深挖两轮（阶段八立项证据档）**：Sudowrite/NovelCrafter/天命/goink 四路+五路参数级 spec 落档（§5.1/5.2 注入与验证两侧）+ 办公/模型/垂类三面（§5.3~5.5）；提案 A-F 组=阶段八~十一刀源 |
| gaea-mcp-hub-reversal-review-2026-09.md | **MCP 双向枢纽翻案评估（拍板池条2，建议维持排除）**：入方向已在产（boot/plugins.go 配置化挂载+热插拔）；出方向三条独立反对证据同向=工程形态矛盾（GUI 壳 vs 常驻 server）/与自身入口重复/安全暴露面与人设冲突；留单向导出作远期观察不立项 |
| gaea-whisper-deadcode-survey-20260925.md | ⬜ **whisper 域死代码调研档（2026-09-25，刀4 立项材料入拍板池）**：deadcode -test 基线 590 不可达 func/147 文件/估 6-8k 行，ackem 蒸馏标记 267 处；五簇分解，三方案（a 整域分批 / b 只删 A 簇 / c 定口径预置资产）待拍板；风险=DSH 并发会话+口径之争+体量 |
| gaea-backend-perf-survey-2026-09.md | 🔄 **Go 后端性能热点普查（2026-09-12，交拍板池刀路 A-E）**：20 项证据 file:line 已核实（回合边界日志全量重读/remember 全局锁内全库重载/cost_search 每查询重建索引/Hephaestus.db 单连接串行读写等）；go vet 全净；与既有条目零重叠；gaea/ai 热路径零 benchmark 需先立基线 |
| gaea-cost-db-master-plan-2026-10.md | 🔄 **造价数据库四期规划（2026-10-07，当前刀路基准，取代 09 基线文档的规划职能）**：系统审视（三套价格数据并存/三代叠层两代已死/价格源未实战验证）+ 四期刀路——一期价格单一真相 v4.464 ✓ / 二期退役清理 v4.465~466 ✓ / 三期能力（3.1 按库组卷导出**用户裁决不做**，3.2 AI 组价复活 / 3.3 对标升级 / 3.4 资源价批量维护待做）/ 四期运维（4.3 数据备份已按用户裁决以全库备份卡落地 v4.467）|
| gaea-cost-domain-survey-2026-09.md | 造价域现状基线（file:line；**校正 roadmap §15 过时欠账**）+ 造价刀路池：§1 编码 v4.178 / §2 多方案 v4.204 / §3 五算 v4.194 / §4 询价扫描 v4.195 / §5 索引补齐 v4.196；**六项全清（§6 含量对照 v4.209 收官，本档转现状基线不再当欠账池）** |
| gaea-priceband-datasource-research-2026-09.md | 🔄 **价格带数据源调研（拍板池候选4，v4.275 结案建议）**：现状核实=价格带数据面基础设施已完整在产（四要素字段+统计+导入链+OCR 询价飞轮）；外部自动接入四路评估=B 抓取不建议（脆弱+合规灰）/C 商业 API 不做/D LLM 查价不作基线；真机验证抓实锤当场修=信息价「除税价（元）」列适配（fieldPrice 字典+回归测试，12/12 识别） |
| gaea-memory-graph-51-design-2026-09.md | 🔄 **记忆语义图谱 5.1 设计（阶段五域内基线）**：事件日志投影三向图（v4.210.0）+ 发送前悬空引用定稿闸（v4.211.0）——**5.1 出口判据全满足**；5.2 首刀=前缀稳定证明+装配 dump/diff（v4.212.0）；5.3 首刀=三态生命周期（v4.213.0，判据全满足）；6.2 首刀=项目本体注入（v4.215.0，判据满足）；6.4 首刀=DCMA 14 点体检 TS 纯函数（v4.216.0，判据满足）；6.5 首刀=蒙特卡洛工期带（v4.217.0，判据满足）；6.6 首刀=跨域 EVM（v4.218.0，判据满足） |
| gaea-memory-injection-eval-design-2026-09.md | 🔄 **记忆注入质量评测设计（市场调研候选2，v4.241 落地）**：与检索评测分立——对真实库 work 视图跑装配点同款构建器（晨报预载+项目本体），断言预算/泄漏/悬空结构不变量，门槛=零违规；纯核 memory.EvalInjectionBlocks + 绑定 GaeaMemoryEvalRun + 办公记忆库「注入体检」Drawer |
| gaea-office-dag-63-design-2026-09.md | 🔄 **办公多文件 DAG 6.3 设计（阶段六域内基线）**：dag_plan 规划→run 档→拓扑分波执行器（每节点=TaskTool.RunNew 子代理会话）→7 绑定→任务中心流水线区——首刀 v4.219.0 判据「≥3 节点链全节点可控」满足，**阶段六收官**；余项（波内并行/运行中 steer 直穿/分级审批/模板库）在档续写 |
| gaea-outlook-longterm-plan-2026-09.md | ✅ **长期规划（已收官）**：接通已有→总闸画面→造价开口→闲庭同一人设；阶段一~四出口已于 v4.184~v4.196 全数满足；后续阶段由 gaea-next-stage-plan-2026-09.md 接棒 |

## 下一阶段优化方向（审计衍生 · 候选池）

| 文档 | 一句话 |
|---|---|
| gaea-dsh-office-distill-research-2026-11.md | 🆕 **dsh-office（COM 实时编辑 Office）蒸馏调研论证（2026-11-13，波1 已动刀）**：上游 v1.2.0 MIT 盘点（8 工具+9893 行 C# COM helper 坑册）+我方办公板块对照（零 COM 零 OMML 互补不替代）+本机 WPS/csc 实证+路线 A 分三波+拍板池 6 项 |
| gaea-office-conversation-output-redesign-2026-11.md | 🆕 **办公对话输出 UI 重设计·五卡系统设计账（2026-11-11，已随 d3342e77 并入 main）**：思考卡/工具卡/过程卡/Ask 卡/审批卡五族+共享运行/完成/错误/决策四态；范围=前端呈现层（不动事件契约与会话模型） |
| gaea-lingxi-re-distill-2026-11.md | 🆕 **灵犀（WPS 灵犀 1.3.36）逆向蒸馏事实源（2026-11-13，P0 已落地）**：asar 逆向（技能 16 个 MIT+交付契约三级组合+wpscli=30 工具 MCP 形态）+资产 S/A/B/C 分级+与 gaea 现状对照（COM 主线保留，补逃生舱/选区/交付契约）+不变量五条+明确不做四条 |
| gaea-lingxi-advantages-2026-11.md | 🆕 **灵犀值得抄的 19 个设计（2026-11-13）**：上下文经济学（三层记忆/截断通知/交接单式压缩）+人机协同（plan 模式/steer）+工具设计（参数错误三要素/危险分级/白名单）+内容质量约束（回复风格/交付顺序）；附录=gaea 最缺 5 条 |
| gaea-lingxi-optimize-plan-2026-11.md | 🆕 **灵犀蒸馏落地计划（2026-11-13，P0 已收官装机）**：P0-1 交付契约硬化+P0-2 受控宏逃生舱+P0-3 选区上下文+P0-4 提示词补齐+P0-5 steer 确认（已有）；P1-1~P1-4（pdf 技能/xlsx 门禁/pptx 模板/WPS 面板停靠）挂拍板，阻塞=Python 运行时决策；P2=office MCP 化+技能市场 |
| gaea-lingxi-distill-report-2026-11.md | 🆕 **灵犀蒸馏评估报告（2026-11-13，逆向收官产物）**：值得抄 6（装配顺序/交付双禁/失败语义族/技能门禁/云 CLI 分层/记忆三层）+有条件 4（逃生舱形态/CDP 桥/沙箱字段/MCP 单 dispatcher）+明确不要 6（JSAPI 本体/沙箱账户/主题/云 API/遥测/codex runner）+三档优先级+反例清单；gaea 侧判断全标 [文档] 未读源码 |
| gaea-lingxi-batch2-plan-2026-11.md | 🆕 **灵犀蒸馏第二批计划（2026-11-13，待拍板排刀）**：对评估报告逐项对账实码销项（xlsx 三条/pdf/pptx/recalc 已落）后立项 B1 文案档（交付时序禁则+技能刷新契约）+B2 失败语义族 10 类+B3 装配顺序评估+B4 记忆三层对照+B5 技能门禁补缺；拍板池=MCP dispatcher/记忆落地/embed/技能市场 |
| gaea-lingxi-prompt-order-map-2026-11.md | 🆕 **灵犀↔gaea 提示词装配顺序对照（2026-11-13，B3 产物）**：gaea 14 段实码装配序×灵犀 130 节点四规律逐条对账——三条天然满足、两条（技能目录前置/场景开关前置）经性质分析不调序；含 A/B 实验设计与红线（SingleModelPrompt 逐字节不动） |
| gaea-lingxi-memory-eval-2026-11.md | 🆕 **灵犀记忆三层×gaea 现状对照评估（2026-11-13，B4 产物，不落地）**：core/wiki/raw 三层 vs ProfileBlock+固化+晨报+citations 同构度约七成；可迁移三条纯文案刀（写入门禁清单/core 预算显式化/召回声明头行）待拍板；不搬 wiki 页面树/61 工具面/云端定位 |
| gaea-w1-bottom-workbench-decision-2026-11.md | 🆕 **W1 底部工作台/终端拍板决策备忘（2026-11-13，待拍板）**：上游事实（宿主终端服务/帧式流/xterm 懒 chunk，插件不拥有 PTY）+ gaea 现状对照（任务中心单向 vs 终端双向）+ 自建 ConPTY 成本 + 三选项（A 不做/B 轻量无终端/C 全量含 PTY）；Q1（是否需要多 pane/多终端）拍板权在用户 |
| gaea-optimization-direction-2026-09.md | 🆕 **2026-09-26 全仓审计衍生的优化方向候选池（未排刀、未抬版本）**：本次实测基线（构建/测试/产物/启动）＋ 分维与 14 模块评分 ＋ P0~P2 优化项（门禁落地 / internal-app 竞态 / 台账自动化 / 上帝包拆分 / 281 遮蔽绑定 / 平行实现收敛 / 效果验证闭环 / 甘特库评估 / 仓库体积）＋ 仓库整理清单 ＋ 未验证项观察池。**不改写 `gaea-next-stage-plan-2026-09.md` 与 `gaea-stage7-plan-2026-09.md` 的优先级**，排刀权在用户 |
| code-audit-2026-10-02/（子目录） | 🆕 **2026-10-02 全仓代码审计（27 单元并行 + 6 分册 + 2 附录；入口 `code-audit-2026-10-02/README.md`）**：专挖屎山代码——平均 **4.54/10**、**512 条带 `file:line` 证据**（P0 25 / P1 216 / P2 209 / P3 62）、**104 条可执行刀**；机械证据见附录 A/B（110 个 ≥100 行 Go 函数、743 绑定面中 322 个前端零引用、139 包依赖图）。**只读审计、未改任何源码、未抬版本**；含快照漂移声明（审计窗口内有并发会话改码）与 78 条未定论观察项 |
| code-audit-2026-10-04/（子目录） | 🆕 **2026-10-04 第三轮全仓审计收敛报告（11 子代理分域 + 双证据复核；入口 `code-audit-2026-10-04/README.md`，原始分域报告在 `reports/` 10 份）**：结论=前两轮清零后的漏网项而非屎山重灾；实施净删 ~1937 行（未用符号/死文件/死本地化键/死类型/死 CSS ~676 行/Go 零调用方法 12 个/round4 收敛单源）+ 防复发守卫（`tsconfig.app.json` noUnusedLocals/Parameters 开闸、wails.json npm→pnpm）；**拦下 15 组「看着像死实则活」误报**（GraphNeighbors/SanitizeToolPairing/CleanupStaleRunning/Win32 布局字段等）；遗留建议清单 §3（结构性大件/重复收敛/待拍板/补测） |

## 域方案与域盘点

| 文档 | 状态 |
|---|---|
| gaea-contextview-upgrade-research-2026-11.md | ✅ **办公板块上下文界面升级换代调研（2026-11-11，dsh-context 0.83.1 对账）**：能力对账+1:1 复刻刀路（Wave1-4 六刀），动刀依据档 |
| gaea-contextview-complete-parity-2026-11.md | ✅ **上下文界面 1:1 完整复刻收官批（刀 11–15，2026-11-12）**：收官批验收与合并收口 d20dec1d 依据 |
| gaea-chat-probe-baseline-20261106.md | ✅ **聊天人格实弹基线对照报告（众神殿 action-1 执行，2026-11-06）**：13b65f37 tag 基线取证 |
| gaea-chat-probe-armb-20261106.md | ✅ **臂B（意图/义务门控+Stage 晋级）实弹对照报告（2026-11-06）**：与基线臂对照定案 |
| gaea-chat-probe-delivery-contract-20261106.md | ✅ **交付合同实弹报告（2026-11-06 午后）**：任务交付合同刀实弹 |
| gaea-cad-tool-master-plan-2026-10.md | 🔄 **CAD 图纸识别工具总体规划（2026-10，当前刀路准绳）**：桌面本地图纸信息提取与问答；PoC 合成样本 PASS，一期查看/提取/问图纸代码落地，待真机走查与发版 |
| gaea-cad-poc-report-2026-10.md | ✅ **CAD PoC 门禁报告（2026-10-23）**：go-cad 解析/分幅渲染/中文字形/图框识别与识图问答验证；真实 DWG 样本验证挂池但不阻塞一期 |
| gaea-cad-recognition-market-survey-2026-10.md | 🔄 **CAD 图纸识别市场与开源技术调研（2026-10）**：DWG/DXF/PDF 解析、查看器、识图问答与许可边界；含二轮 GitHub/社区盘点附录 |
| gaea-office-upgrade-plan-2026-09.md | ✅ 主体已落地（分期 v4.23–v4.31）；「远期」行（终端 tab/双工作台/侧边对话）维持拍板门控 |
| gaea-office-mindmap-base-design-2026-09.md | 🔄 M1/B1/B2 看板/M2 画布编辑（v4.108.0）已落地；**B2 后半（字段面板/画廊）待拍板** |
| gaea-pptx-edit-design-2026-09.md | ✅ 刀1 v4.109 / 刀2+刀3 v4.156 已交付；刀4 待反馈；真机走查挂池 |
| gaea-novel-revolution-2026.md | 🔄 刀1–6 已落地（v4.77 批次）+ v4 场景制续刀（场景元数据 v4.199）；GenerationGate 闭环/刀7 续/刀8 未落地 |
| gaea-character-domain-survey-2026-09.md | ✅ 阶段四读档（**结论=一套资产 + 项目工作副本**，非三套人）；出口三小刀已落地（外观锚点 v4.192 / 副本回写 v4.193 / 关联即快照 UI） |
| gaea-dream-studio-nextgen-2026-09.md | 材料汇编+下一代草案，不承诺版本（§0.5 已被图域 longterm-plan 吸收） |
| gaea-image-domain-longterm-plan-2026.md | 长期路线：T0 契约已落地（v4.98.0）；T1+ 未启动 |
| gaea-image-domain-t0-contract-design-2026-09.md | ✅ 已落地随 v4.98.0 |
| gaea-sin-board-design-2026-09.md | ✅ **闲庭·原罪板块（v4.256.0）**：对话式图文混杂故事创作——办公组件复用清单、sin 绑定契约（会话/流式/插图/导出）、cue 编号跨端约定、成人内容边界与验收证据 |
| gaea-ai-workbench-ui-2026-10.md | 🔄 **办公 AI 工作台工具化设计单（2026-10，刀路准绳）**：§7.3 检查器随工具（c90aefb3）/§7.5 首页 12 卡（2b76dbab 批）已落地；余池=亮色暖纸（需拍板） |
| gaea-ui-benchmark-2026-10.md | 🔄 **UI 对标调研（2026-10）**：首刀=监测↔模型中心统计面板合并（e6749ebf 已随 v4.526.0 发版）；后续刀按档推进 |
| gaea-runtime-monitor-redesign-2026-11.md | ✅ **运行监测 UX/UI 重构（2026-11-03）**：调用统计升为首屏主工作区，范围 KPI/趋势/成功率/耗时桶/错误聚合/模型搜索筛选排序分层；记录统计口径、响应式走查、测试与桌面产物身份 |
| gaea-home-redesign-survey-2026-10.md | ✅ **首页重设计调研+实施档（2026-10-27 十三连推送）**：一级导航合一（§8.2）/dock 退役（§8.8）/首页 12 卡；E26 守卫锚已随 0d883cd5 迁 navigateBoard |
| gaea-board-slim-reorg-2026-10.md | ✅ **板块瘦身重组纪要（2026-10-28）**：模型中心导航 13→9/小说质检菜单 8 项三组/CreatePage 40→24 态；真机走查已过 |

## 引擎线（本地推理引擎：调研 / 基座决策 / 批次台账）

| 文档 | 状态 |
|---|---|
| gaea-tianxuan-research-2026-10.md | 🔄 **天璇引擎（tianxuan）立项调研与批次台账（2026-10-18）**：社区/GitHub 调研 + halogen 参照 + 基座定夺（**Gufo 取代 MoE4All**，MIT、同芯片 73/194 tok/s 实测上限）+ 批次 1/2 真机数据 + **批次 4：27B 缓存续写保留 DFlash 接受率、默认 prompt lookup（代码复述 39.3→52.6 tok/s、输出哈希一致，普通 prose 持平）+ 35B-A3B Vulkan MTP 极速档（长答 59–62 tok/s，约 4×，模型中心显式切换）**；35B HIP 线仍终止；**已单独立项到 `C:\AI\tianxuan`**（独立 Git 仓库、`scripts\serve.ps1` 档位入口，gaea 经 `tianxuan_path` 外部引用） |
| gaea-tianxuan-strata-deepdive-2026-10-18.md | 🔄 **tianxuan × strata 深度调研（2026-10-18）**：27B 带宽墙、35B-A3B 冻结证据链、strata 专家分层/预算/KV 流式拆解、tianxuan 多模型路线与移植优先级 |
| gaea-moe4all-engine-assessment-2026-10.md | 🔄 **MoE4All 引擎接入实用性评估（阶段 0 真机全过，2026-10-17）**：v1.0.3 本机 Strix Halo 闸门（工具调用/流式 usage//v1/models/thinking）+ 同模型对擂 Strata + 生命周期面；结论=可接（≈一条引擎种子 + herdsman 式生命周期），但基座决策已选 Gufo，MoE4All 保留为 fallback 与工程纪律参照 |

## 蒸馏规划（取道不取器）

| 文档 | 状态 |
|---|---|
| gaea-voice-port-distill-20261107.md | ✅ **语音 Go 移植蒸馏审计（上游 speech-to-speech v1.1.0 ↔ Go realtime 包）**：逐文件对源+移植漏抄形态清单（跨态清理守卫高发）+探针须 1x 实时节奏+cgo 测试族 CGO_ENABLED=0 静默全跳坑 |
| gaea-voice-upstream-audit-20261107.md | ✅ **语音实时管线全量对源审计（2026-11-07）**：REA 纪律义务清单锚上游 file:line 原文，协议门 turn/bargein/cancel 与真引擎多轮探针取证 |
| gaea-voice-upstream-audit-20261104.md | ✅ **语音管线复刻审计台账（2026-11-04）**：义务须锚上游原文纪律的首份台账，移植面义务分册 |
| gaea-web-search-pantheon-digest-20261105.md | ✅ **众神殿「网络搜索能力弱」论证资料整理（2026-11-05）**：论证取证 digest，网络搜索优化线（三刀收官见基线 150 记录 docs/）输入 |
| gaea-reasonix-v138-distill-2026-09.md | 🔄 Reasonix 真身蒸馏（esengine/DeepSeek-Reasonix v1.15→v1.38）：七刀已落地（v4.374.0：Host 守卫/只读表修正/clean-filter 中和/SSRF 补洞/收尾判定/输出内存上限/stuck 解锁）；留池 9 项（执行序批次/live file observations/采样恢复状态机/压缩救援阶梯/PTY 等） |
| gaea-dsh-016-distill-2026-09.md | 🔄 dsh（deepseek-harness TS monorepo）蒸馏（原误题 Reasonix，已勘误更名）：五刀已落地（v4.373.0：溢出自愈/缓存对齐摘要/shrink 校验/force 多轮/repeat 分级）；留池 12 项按 dsh 口径有效 |
| gaea-dsh-021-core-distill-2026-10.md | 🔄 dsh v0.2.1 核心域蒸馏（deepseek-harness@dsh-v0.2.1-alpha.2）：压缩域等价销号/todo/plan/ask 同构；落地 session-title 自动命名（TurnDone 槽+办公本地路由数据不出本机）；未深扫域诚实声明十三域 |
| gaea-dsh-subagent-distill-2026-10.md | 🔄 dsh 0.2.1 子代理板块蒸馏+刀池：P0 interrupt_agent 模型面中断+白名单 fail-loud（e19543bb）；P1 结算 stopReason+diagnostic 卫生面+subagent_list 枚举（f5aa2892）；P2=session-reference/模型路由/深度记账/send_message 在册 |
| gaea-dsh-022-distill-2026-11.md | 🔄 dsh 0.2.1-rc.1→0.2.2-alpha.2 增量蒸馏（2026-11-02 调研档）：P2 池三项上游演化对账（模型路由上游已给参考设计/depth.ts 零变更）+新刀池 K1 切模型压缩门/K2 checkpoint 持久化门/K3 附件引用完整性/K4 取消错误码词汇，均带刀前核实项；坑=升级指南补写形态/rev-list range 噪声/master 停 rc.2 基线取最高 tag |
| gaea-dsh-subagent-deep-distill-2026-11.md | 🔄 dsh 0.2.2-alpha.2 子代理板块机制级深挖（2026-11-02，src 6360 行全量+同日第二轮 G 系补遗收官）：A~F 六组+G 八条机制纪律+刀池 SA-A structured_output（执行批销号=task output_schema 已等价实现）/SA-B 委派审批语义（随 V5 在册）/SA-C 失败赢过取消（执行批销号=判据缺失）/SA-D+SA-E 已落地（2026-11-02 执行批）；**UA-1 口径修正=活跃耗时是回合区间累计投影非墙钟**；销号=结算四态/list_agents 已对齐/ActivationPool 拒绝语义（gaea 排队维持）/fork 种子平衡前缀已同构；坑=注释是纪律本体优先读/克隆检出≠调研基线 |
| gaea-dsh-subagent-ui-distill-2026-11.md | 🔄 dsh 0.2.2-alpha.2 子代理 UI/UX 蒸馏（2026-11-02 补课档，此前从未系统蒸馏）：ui-subagent 谱系树/只读三态/指标行+ui-chat turn-trigger 源徽标/过程聚合/live 活动词提取；刀池 UA-1 SubagentThread 活跃耗时（meta 就绪）/UA-2 活动词优先级清单；谱系树=嵌套若开 UI 预案留档；坑=上游 UI 蒸馏入口=locales 词汇表先读字典反查组件 |
| gaea-programming-rea-plugin-integration-20261108.md | ✅ **编程板块 DSH Web + REA + 桌面插件迁移（2026-11-08）**：隔离 DSH `0.2.2-rc.1` 内嵌 `dsh.wails.localhost:3091` 解决 Strict Cookie；REA `6.8.0` MCP 实测 229 工具；桌面 9 依赖迁入、8 bundle 启用且 `dsh-wubi` 保持最后；`dsh-ops` 升 `0.5.1` 适配；GPU PowerShell 与 dshmarket restart helper 双修，cmd 黑窗实测 `visible=False`；产物 v4.534.0 SHA256=`9D0178BE67E6...` |
| gaea-dsh-univer-office-distill-plan-2026-09.md | ✅ 已蒸馏收官（U 系落地 v4.97~v4.109；Univer 零引入） |
| gaea-dsh-genui-distill-plan-2026-09.md | ✅ 已收官（P0–P5 全部发布，止于 v4.97.0） |
| gaea-better-sidebar-go-iteration-2026-11.md | ✅ **现行唯一迭代与交接入口（2026-11-12 已提交并打包）**：固定上游 v0.32.0；F1/F2/F3 与 G1.1–G1.5 已提交为 `f49ea150`，独立干净树构建、正式冒烟、预览包、源码包和哈希已记录；未 push/未装机，G1 深化待做。用户已拍板不重写 gaea、不做侧聊 |
| gaea-dsh-better-sidebar-long-term-distill-plan-2026.md | 📚 历史调研与早期阶段台账（v0.18.0 基线）；当前状态已移交 `gaea-better-sidebar-go-iteration-2026-11.md`，勿据此把已完成能力重列欠账 |
| gaea-unsloth-modelhub-distill-plan-2026-09.md | ⚠️ 已退役（2026-10-18 modelhub 引擎整体删除：引擎/Key/MH4 预热随 Unsloth 退役下线；CU1/CU2 等 ComfyUI 侧蒸馏成果保留）。本文保留作历史蒸馏记录 |
| gaea-reasonix-2x-distill-2026-10.md | ✅ **Reasonix 2.x 蒸馏重开（2026-10-24）**：A 档八刀+B 档五刀全清随 v4.524.0 发版（readiness 死闸根修/read_subagent_result/出发快照/steer 收据收口） |
| gaea-zcode-distill-2026-10.md | ✅ **ZCode 蒸馏规划档（Z0~Z18 刀池+十二节跨源合流分析）**：v1 收官随 v4.525.0（运行时 8+产出 5+交汇 2，dfd8e169→faebe184） |
| gaea-attention-transaction-2026-11.md | 🔄 **注意力事务（2026-11-02，众神殿 super agent 讨论产物）**：第一刀=请求/结果成对事件（`approval_resolved`/`ask_resolved`）+ `source` 判别字段贯通实时层/日志/轨迹，前端零改动；阶段 0 基线统计器 `scripts/attention_report`（只读）；开放项=case_id/step_id 事项维、V5 事件账本对齐、旧日志不可补 |
| gaea-orca-pantheon-distill-2026-11.md | 🔄 **Orca 1.4.214 × 众神殿机制蒸馏（2026-11-02）**：判定 Orca 是 Agent 工作台而非议事系统；提炼 Attempt 身份、状态证据、完成权威、单写者租约、工作树边界与外置委派纪律；**P0 O-A/O-B/O-C 已落地**（AttemptRef 贯通、旧 Attempt 封口、禁隐性派生、原生/路径/共享摘要三路图片能力门），O-D 起按门控后置；明确不照搬完整自治与“worktree 即沙箱” |
| gaea-orca-pantheon-distill-round2-2026-11.md | 🔄 **Orca 第二轮蒸馏：投递/观测空间/结案纪律（2026-11-02）**：修正第一轮三处口径（新鲜度分判定 60s 与展示 30min 两层、序号只在同一 authority 内可比、租约防进程复用而 dispatchId 防迟到）；新增 M9–M17（投递状态机、快照不续命、未知态五档、终态图+唯一例外、兄弟在跑不结案、门重阻塞、三句式陈词、释放与归档分离、重试重声明）；刀池重排 O-D/O-E/O-H/O-I，带 Go 形状与逐条验收 |
| gaea-pantheon-deliberation-frameworks-research-2026-11-03.md | 🔄 **众神殿讨论/辩论框架专项调研（2026-11-03，证据附录）**：MALLM 四轴与评测、Captain Claw 动作/主持/确认、ARTEMIS 生成与评审分离、AgentTeams 人工干预与执行账本；逐项写明吸收/不吸收、源码快照与产品边界，优先级由 V2 整合报告 R1–R6 决定 |
| gaea-pantheon-openmausbot-distill-2026-11-07.md | 🔄 **OpenMausBot × 众神殿第一轮蒸馏（2026-11-07，G0 证据冻结）**：钉版 `1bc4749`；A1 主席控制信封 / A2a 环形检测纯函数 / A5 四席共享上下文已落地，A2b 关闭，B1/A3/A4/B2/B3/B4/B5 按门控挂池；三列制（源码直证/本地再设计/已淘汰）+ 统计纪律 + 门槛流水线 |
| gaea-pantheon-openmausbot-distill-round2-2026-11-07.md | ✅ **OpenMausBot × 众神殿第二次蒸馏计划与落地：收口、证据与提问（2026-11-07）**：E1 席位工具证据摘要 / E2 派工回执+A1 透明补完 / E3 主席终局信号 / E4 席位自述提问已落地；E5 同位失败仅观察；含第一轮落地复核（A2c 口径更正/A1 派工未渲染）、不立项红线、G0–G4 门控与逐项验证 |
| gaea-pantheon-agorai-aidebate-distill-2026-11-07.md | 🔄 **Agorai × aidebate 双源蒸馏调研（2026-11-07，只调研不立项）**：钉版 Agorai `1e469be`(v0.8.0/AGPLv3) 与 aidebate `80f3b77`(v0.4.7/Apache-2.0)；判定两源都不足以重开议事协议——Agorai 的产品面（四级可见性/记忆/技能）与本地单用户错位、aidebate 的 tmux 传输层与 headless 席位错位；真增量三小刀=P0 席位「无新增」语义（pass）/P0 开场功能冒烟（canary）/P1 议事健康指标（只展示）；含反例三则（关键词共识/无界自适应超时/多数关轮）、9 条不立项与复活条件、逐条 `path:line` 源码索引；证据层 `docs/research/{agorai,aidebate}-distill/` |
| gaea-pantheon-agorai-aidebate-distill-round2-2026-11-07.md | ✅ **双源蒸馏第二轮增量+两本账五刀落地（2026-11-08）**：增量事实（agorai 吞 NO_RESPONSE/pause 不冻定时器/带内命令权限形同虚设等 8+6 条实锤）与三处分歧修正（M2 从众+耗时两维翻案/过半宽限维持不抄/M6 核对未落地）；落地=轻刀①工具要点契约+轻刀②E1 非空锚与对账行+M1 任务书指纹+B4 墙钟预算阶梯〔token→墙钟改判〕+M2 blind_rebuttal 第三协议〔互盲=并行调度结构性性质〕+收尾审计（续议 seed 预算继承）；§6~§6.7 全程录 |
| gaea-pantheon-seat-hang-cache-2026-11.md | 🏛 **众神殿席位卡死根修与 DSH 前缀缓存（2026-11-06，证据档）**：实弹会话 209 次工具调用/单轮 87 次/单席 147–403s 的取证；根因=自主循环无上限+30s 工具超时+8 分钟上限与自动重试+工具事件持锁整份落盘；落地=DSH 引擎会话续接（缓存命中 25%→96.7% 实测）、工具预算/无进展/开局静默三闸、命令行显式只读+审批、`Contribution.Usage` 可核对账目与前端展示 |
| zcode-office-distill-2026-10.md | ✅ ZCode office 域对源蒸馏分析（A1 锚点语法层/F2 doc_render 等交汇刀的对源证据层） |
| zcode-office-distill-analysis-2026-10.md | ✅ ZCode office 蒸馏子分析报告（机检十二规则/Q1 对源） |
| zcode-office-decision-memo-2026-10.md | ✅ ZCode office 蒸馏决策备忘（双执行体轮次署名权/取舍记录） |
| zcode-dialogue-2026-10-24.md | ✅ 双执行体对话档（ZCode 蒸馏线，单轮存档） |
| zcode-distill-dialogue-2026-10.md | ✅ 双执行体协作对话档七轮全录（协议⑦写前扫描/⑧轮次署名权） |
| zcode-handoff-2026-10-24.md | ✅ DSH 交接档（ZCode 蒸馏线批次交接） |

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
| gaea-t0-effect-audit-2026-11-08.md | ✅ **议事 T0 effect 机械审计·第一遍（2026-11-08）**：众神殿 T0 effect 面逐点机械审计证据物 |
| gaea-slim-baseline-2026-09.md | ✅ P0 基线证据层：七面四表实测（dist/entry/MemoryHub/exe/依赖/>50KB 源文件）+ 功能等价快照 + IA 走查 |
| gaea-slim-knife2-audit-2026-09.md | ✅ 审计证据：**G-2 已关**（v4.169「基线 N」chip）；G-3 真机走查挂池 |
| gaea-slim-p2-dualspace-2026-09.md | ✅ 审计证据（P2 双空间并列）；权威=masterplan §轨道一·2/3 |
| webview2-shell-audit-2026-09.md | ✅ 审计完成（P0×2/P1×7 + 刀序 A–D）；**刀 A–C 已修**（v4.165/166）；刀 D（printSvg print/拖拽/粘贴）真机取证挂池 |
| gaea-office-hub-decouple-audit-2026-09.md | ✅ 已收刀（v4.197：`office/docmd → internal/docmd`，内核内边清零）；遗留候选（别名门面/archive.go 死码候选/度量口径）挂观察池 |
| gaea-session-binding-audit-2026-09.md | ✅ 无参绑定会话语义审计收官（v4.338.0）：**需修 0 项**——「看历史=切内核」构造性自洽，v4.181 旁路看板族无同构残余；复审口径入档；拍板池=GaeaSkillDraftFromSession 参数化 |
| distill/ | 🔄 MuMuAINovel 蒸馏规格与实施交接书（01~07 六域 + `09-impl-handoff.md`）；**t1 共享契约已落库（v4.278.0）**，t2~t7 未实施 |
| gaea-sin-booksource-distill-2026-09.md | 🔄 原罪·书源引擎蒸馏规格（so-novel 机制重推导；AGPL 红线=不搬代码与规则 JSON）；并行线在制品，代码落点 `internal/booksource/` |
| gaea-sin-booksearch-distill-2026-09.md | 🔄 原罪·书源检索蒸馏规格（并行线在制品） |
| gaea-novel-ohstory-distill-2026-09.md | 🔄 小说板块·oh-story-claudecode 蒸馏规格（MIT：机制重推导 + 知识资产带许可收录）；技能×gaea 映射 + 六刀刀路，首刀=T1 评审 rubric 数据化 |
| gaea-novel-skill-distill-survey-2026-10.md | ⬜ **网文写作 Skill 生态调研（2026-10-24）**：十 Skill 溯源到四仓库，核对 MIT/GPL 红线与 gaea 现有覆盖；只调研不立项，候选走拍板池 |
| gaea-novel-booksource-import-2026-09.md | 🔄 小说板块·书源取书→拆书导入接通刀路（用户拍板 2026-09-13：两板块不合并改接通；**2026-10-27 修订：两板块导航级合并，原罪收编为小说第六页签**）；书源引擎（零消费者）作导入取书上游，NovelB +4 绑定 + 书架「在线搜书」；sin 书源线 t2~t5 归并行线不碰 |
| gaea-longform-novel-system-2026-09.md | 🔄 长篇小说创作系统方向（2026-09-28 立项，用户指令「解决如何写一个优秀的长篇小说」）：§1 诊断十四条（带代码证据：意图输入仅一句剧情要求 / StoryThread 无写入点 / ChapterPlan 零消费 / 生成单位整章 blob / 场景卡缺工艺字段 / 分析情报不回写 / 写前契约软提示 / 质量门不收敛 / 上下文堆料 / 风格事后打分 / 无整书状态 / 前文窗口 10 章断档 / 张力字段不写 / 章内续写无衔接）+ §2 机制 + §3 对标 + §4 评测 + §5 七刀刀序 + §7 刀1 契约与验收 |
| gaea-longform-craft-mechanics-2026-09.md | 🔄 长篇创作机制调研交付物（2026-09-28，方法论→可机械化的 14 机制 M1–M14 + 25 条「优秀长篇」机械判据（22 条可确定性自动化）+ 故事层/场景层/文本层三层架构 + 主流工具对应层做法与短板）；要点已并入 gaea-longform-novel-system §2 |

## 当日总账

| 文档 | 状态 |
|---|---|
| gaea-day-log-2026-11-14.md | ✅ 2026-11-14 当日总账（**CAD/进度计划回归右栏一级 Tab + 办公打开 CAD**）：ToolSheet 退役、注册表派生入口、CAD 文件打开总线、pane 视图兼容与干净树构建装机记录 |
| gaea-day-log-2026-11-13.md | ✅ 2026-11-13 当日总账（**better-sidebar G1.6/G1.7/G1.8/G2 全清+真机走查双实证**）：worktree 只读展示+Git 子进程超时/预算、冲突六形态/rename 溯源/二进制占位、空仓降级矩阵（git ≥2.26 空树语义实锤）、只读提交图（lane 拓扑/分支筛选/提交详情）实壳断言全绿；main ahead origin 未 push |
| gaea-day-log-2026-11-12.md | ✅ 2026-11-12 当日总账（**办公板块上下文界面升级换代 dsh-context 0.83.x 全量复刻**，v4.542.0）：调研→拍板→动刀全链一天收官+收官批刀 11-15+合并收口 d20dec1d+真机两连崩根修+超大附件改造 v4.543.0；better-sidebar F1 提交 f49ea150+预览包 |
| gaea-day-log-2026-11-11.md | ✅ 2026-11-11 当日总账（knip 尾款清零+天璇退役漏网收口＋**WHIRL 推理引擎接入** v4.536.0）：对擂 1.91x→盲测→whirlfix2→实弹三连修→引擎实升（llama +49%）→35B 单副本收口 |
| gaea-day-log-2026-11-07.md | ✅ 2026-11-07 当日总账（**语音管线翻车双根修+五路蒸馏审计+三批修刀收官**）：真麦翻车定性（silero v5 帧格式）+回声门 v3+重叠线定罪四刀+协议门 turn/bargein/cancel+UI 四刀 |
| gaea-day-log-2026-11-06.md | ✅ 2026-11-06 当日总账（**聊天板块核心优化第一批+实弹基线+臂B定案**）：四卫生切片+基线对照（13b65f37 tag）+臂B（意图/义务门控+Stage 晋级）+交付合同刀 |
| gaea-day-log-2026-11-05.md | ✅ 2026-11-05 当日总账（**语音 V2 唯一路径化大日**）：卡死根修+网关/V1 双退役+协议四修+交互重设计+bundle v2；桌面入口收敛装机同哈希 |
| gaea-day-log-2026-11-04.md | ✅ 2026-11-04 当日总账（**三引擎退役+Strata flash 档线**）：CosyVoice/Ollama/OpenCode Zen 代码面全删（118 文件）+Strata flash 接入（AMD_HIP 配方实测）+35B 档删除收窄单档 |
| gaea-day-log-2026-11-10.md | ✅ 2026-11-10 当日总账（**办公板块工具治理：众神殿六阶段一天收官**）：P0 注册表地图 82 行三守卫+P1a 结构化写入口（office_generate/四类写升 agent 面）+P1b 发布边界（五类写入拦截实证）+P2 两波（通道 A 引用级深化+abeval 扩+条件视觉首渲覆盖率）+P3 四类谓词 harness+P4 办公批处理进任务中心；六提交（a63339dd…b0166d00）+装机 F9BA8DBEDA41 PID 51408+实弹五面清单；晚批=自主三刀（空间契约漂移/REA preflight/版本三源）+代码图谱建图 |
| gaea-code-atlas-2026-11.md | ✅ **项目代码图谱（2026-11-10 建图）**：十层分层（入口/壳层/内核装配执行/内核共享库/办公/小说/引擎/众神殿/领域库/scripts）+层间依赖矩阵 467 边+分层不变量三项全绿+162 包清单+依赖明细+巨石文件 Top30+前端 src 地图+关键接缝导读；由 scripts/code_atlas_doc.go 单源生成（机械段勿手改） |
| gaea-day-log-2026-11-09.md | ✅ 2026-11-09 当日总账（识图诊断线收官五刀+成本导入兜底腿换正道）：docmd OCR 缓存/stampNoiseText 严格谓词/@引用 30s 预算/失败文案真因/OPCRPDFText 强制扫描件 OCR；38 笔列车与收口见 11-08 线 |
| gaea-dsh-ops-distill-2026-11-10.md | ✅ **dsh-ops 插件蒸馏（2026-11-10，纯调研未动刀）**：有界文件/文档工具插件全貌+八机制蒸馏（revision CAS/动态工具目录/结果预算 ref:// /失败即值/凭据围栏/TOCTOU 双校验/batch plan-resume/基准门禁文化）+同构等价表（六项不立项）+候选池 A1 结果预算/A2 动态工具目录/A3 revision CAS/B1 凭据围栏/B2 失败即值（动刀须点名）+上游诚实文化摘录 |
| gaea-tool-registry-map-2026-11.md | ✅ **P0 工具面注册表地图（2026-11-10）**：唯一注册表全部 reg.Add 行 82+ 契约（来源/空间/副作用/证据档/验收谓词）+P3 四类验收谓词表；由 scripts/registry_map_doc.go 从 tool.RegistryMap 单源生成（勿手改），守卫 tool/boot/app 三处 |
| gaea-voice-memoryhub-ui-20261109.md | ✅ 2026-11-09 语音交互窗口记忆中枢化：全窗居中虚空大脑、完全透明右侧转写、居中胶囊控制、深浅双主题、中央实时与最近轮次字幕；含构建与实机边界记录 |
| gaea-day-log-2026-11-08.md | ✅ 2026-11-08 当日总账（编程板块回流+记忆图谱真 3D 脑网格+语音星团随声互动，v4.535.0）：3+1 提交索引+cmd 黑窗根修口径（proc.HideWindow=HideWindow+CREATE_NO_WINDOW，活探针漏网形态）+验证矩阵+余池（SHA256 前缀 20D0B620，PID 28968） |
| gaea-day-log-2026-10-30.md | ✅ 2026-10-30 当日总账（众神殿三连发）：10 提交索引（v4.528.1 实弹修复+v4.529.0 UI 走查五刀+v4.529.1 亮态补课三发+V5 归档+releases 归位）+验证矩阵+新坑九枚+余池（SHA256 前缀 88ea4cdcd835） |
| gaea-day-log-2026-10-29.md | ✅ 2026-10-29 当日总账（双版本日）：17 提交索引（v4.526.0+v4.527.0 两发+OCR 线全线/暖纸/CI 清欠）+验证矩阵+新坑十二枚+余池（SHA256 前缀 db51be26cfc24e61） |
| gaea-day-log-2026-10-28.md | ✅ 2026-10-28 当日总账：五提交索引+余池清单+exe 哈希（SHA256 前缀 bffba192dacd0817） |

## 上手与工装

| 文档 | 状态 |
|---|---|
| gaea-getting-started-30min-2026-09.md | ✅ 30 分钟上手（2026-09 定稿）：从零到「能安全改代码并发布」的每日动作，权威细节指向 AGENTS + masterplan |
| 2026-08-14-sandbox-environment-notes.md | 沙箱环境备忘（详细版；AGENTS 中为四条铁律摘要） |
| snapshots/ | 目检截图 12 张（侧边栏/浏览器面板证据，2026-09-03；正文引用见 webview2-shell-audit 与 releases/v4.101~105） |

## 已收官历史设计（留原位：代码注释仍引用为设计出处）

gaea-space-shell-design.md（S2.1，v3.9.0）· gaea-space-dimension-design.md（v3.8.0）· gaea-space-assembly-design.md（v3.8.0）· gaea-memory-isolation-design.md（S1.2，v3.8.0）· gaea-page-migration-design.md（P1，v3.9.0；挂账项以 roadmap 为权威）· gaea-edit-tools-design.md（五工具在产，S0.6 起）· gaea-genui-memoryfence-audit-2026-09.md（审计存档；6/7 项 v4.101 收口，resume 槽位口径=唯一开放项）

## 政策 / 约定 / 数据集

ADULT_MODE.md · DREAM_WRITE_POLICY.md · MEMORY_ARCHITECTURE.md · evaluation-set.md · retrieval-eval-set.md（12 条查询集，代码运行时直接解析）· memory-eval-set.md（三脑+经验习得题集，阶段七 7.1-1；**⚠️ 2026-09-26 状态校正：其配套包 `internal/memoryeval` 已于提交 68c56bc1 作为死代码删除，文中「`go test ./internal/memoryeval` 一键跑出四域基线」不再是有效入口，四域基线转历史数据；恢复门禁或改口径见 docs/archive/gaea-optimization-direction-2026-09.md §3 P1-4**）· ilink-non-text-protocol.md · 2026-08-15-gaea3-architecture-design.md（历史基准：内核架构事件日志/Manifest/Seam，仍有效）

## 归档区（不在此处展开）

`docs/archive/` 收录历史调研、已落地计划、被后续结论取代的文档（含 8 个 research-2026-09-* 原始稿目录、market-research 系列、superpowers 计划/规格、gaea2/gaea3 时代文档、`agents-version-history-2026-09.md` 版本磁带与 `progress-history-2026-09.md` 进度磁带）。**索引见 `docs/archive/README.md`**——引用归档结论前先确认未被现行权威取代。
