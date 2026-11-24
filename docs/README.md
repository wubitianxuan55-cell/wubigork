# docs/ 文档索引

> **2026-11-24 清理**（用户令「许多记载的文档已经过期可以清理」）：顶层只留现行文档。已收官 / 未立项的规划、已退役的功能与引擎、
> 已闭环线的调研与过程档共 41 份移入 `docs/archive/`；此前只登记在本索引、实际早已归档的文件改在 `docs/archive/README.md` 登记。
> 顶层现为 74 份文档（计数以 `node scripts/check-docs.mjs` 输出为准）。**当前没有长期规划**（用户 2026-11-24 明示）。
>
> **维护规则（防孤儿）**：新增 / 移动文档必须同步本索引；**未登记文档视同孤儿**。文档过期（功能退役、计划收官或作废、结论被取代）
> 就移入 `docs/archive/` 并在那里登记一行，不要留在顶层只改标签。引用任文档前先读其头部状态行。
> **守卫**：`node scripts/check-docs.mjs`（孤儿登记 / `docs/` 悬空引用 / `.gaea/AGENTS.md` 指令预算 / 含非 ASCII 的 .ps1 必须带 UTF-8 BOM），已接入 `scripts/ci.ps1`。
>
> **数据根（2026-11-20 用户拍板）**：所有蒸馏 / 逆向 / 上游取证的数据文件都在仓库外 `C:\AI\re`（总览见 `C:\AI\re\README.md`）。
> `docs\` 只放文档；文档引用数据一律写 `C:\AI\re\<数据集>\...` 绝对路径，不写仓库内相对路径。
>
> 发布物全文在 `releases/`；现行规则与决策在 `.gaea/AGENTS.md`；各线进度在 `.gaea/progress.md`。

## 工程与构建（权威与单一事实源）

| 文档 | 内容 |
|---|---|
| gaea-build-guide.md | 日常构建与装机权威 |
| gaea-tool-registry-map-2026-11.md | P0 工具面注册表地图（2026-11-10） |
| gaea-code-atlas-2026-11.md | 项目代码图谱（2026-11-10 建图） |
| third-party-notices.md | 第三方代码与设计来源许可记录 |
| gaea-programming-rea-plugin-integration-20261108.md | 编程板块 DSH Web + REA + 桌面插件迁移（环境记录；插件版本以 2026-11-22 当日总账为准） |

## 政策 / 约定 / 数据集

| 文档 | 内容 |
|---|---|
| MEMORY_ARCHITECTURE.md | 记忆体系三脑架构（2026-08 落地） |
| ADULT_MODE.md | 成人模式决策记录（2026-08-14 成文，2026-11-16 仍在修订） |
| DREAM_WRITE_POLICY.md | 记忆「自动做梦」写入审批决策（2026-09-21 建议制修订） |
| evaluation-set.md | 生产化评估集（1.x 真实修复点样本，能力验收基线） |
| retrieval-eval-set.md | 检索质量测评查询集；`GaeaRetrievalEvalRun` 运行时按本路径解析文内 JSON 块，**不可移动** |
| memory-eval-set.md | 记忆质量测评题集；配套 `internal/memoryeval` 已删，四域基线为历史数据（见文首校正） |
| ilink-non-text-protocol.md | 微信 iLink 非文本消息协议备忘（图片 / 文件真机定稿） |
| gaea-cloud-model-limits-20261116.md | gaea 云端模型底线实测 |

## 已落地功能的设计档（留原位：代码注释引用为设计出处）

| 文档 | 内容 |
|---|---|
| gaea-space-dimension-design.md | 双空间 S1.1 空间维度落地设计（v3.8.0 收官） |
| gaea-memory-isolation-design.md | 双空间 S1.2 记忆空间隔离器设计（v3.8.0 收官） |
| gaea-space-assembly-design.md | 双空间 S1.3~S1.5 按空间装配设计（v3.8.0 收官） |
| gaea-space-shell-design.md | 双空间壳 S2.1 设计（v3.9.0 收官） |
| gaea-page-migration-design.md | 页面迁入设计 P1（v3.9.0 收官） |
| gaea-edit-tools-design.md | edit_file 工具层设计（grep / edit_file / multi_edit / edit_lines / move_file 五工具，实现权威） |
| gaea-chat-branch-tree-design-2026-11.md | 聊天消息分支版本树设计（A 批） |
| gaea-office-conversation-output-redesign-2026-11.md | 办公对话输出 UI 重设计·五卡系统设计账（2026-11-11，已随 d3342e77 并入 main） |

## 在途与待拍板

| 文档 | 内容 |
|---|---|
| gaea-cost-db-master-plan-2026-10.md | 造价数据库系统审视与分期规划（2026-10-07；后续刀路以本文为准，逐期关账） |
| gaea-cad-tool-master-plan-2026-10.md | CAD 图纸识别工具总体规划（2026-10，当前刀路准绳） |
| gaea-cad-recognition-market-survey-2026-10.md | CAD 图纸识别市场与开源技术调研（2026-10） |
| gaea-cad-poc-report-2026-10.md | CAD PoC 门禁报告（2026-10-23） |
| gaea-lingxi-batch2-plan-2026-11.md | 灵犀蒸馏第二批计划（2026-11-13，待拍板排刀） |
| gaea-lingxi-memory-eval-2026-11.md | 灵犀记忆三层×gaea 现状对照评估（2026-11-13，B4 产物，不落地） |
| gaea-w1-bottom-workbench-decision-2026-11.md | W1 底部工作台/终端拍板决策备忘（2026-11-13，待拍板） |
| gaea-agent-memory-nebula-distill-2026-11.md | Nebula Memory → gaea 记忆中枢对齐蒸馏报告（2026-11-22） |
| gaea-obscura-egolite-browser-distill-research-2026-11.md | Obscura × ego(lite) 浏览器蒸馏可研（2026-11-14，P1+P2 已动刀） |
| gaea-deepseek-chatdemo-distill-2026-11.md | DeepSeek chat-demo 蒸馏调研（唯一事实源） |
| gaea-better-sidebar-go-iteration-2026-11.md | 现行唯一迭代与交接入口（2026-11-12 已提交并打包） |
| gaea-ui-benchmark-2026-10.md | UI 对标调研（2026-10） |
| gaea-attention-transaction-2026-11.md | 注意力事务（2026-11-02，众神殿 super agent 讨论产物） |

## 近期落地线的调研与台账（线已收官，留作出处）

| 文档 | 内容 |
|---|---|
| gaea-dsh-office-distill-research-2026-11.md | dsh-office（COM 实时编辑 Office）蒸馏调研论证（2026-11-13，波1 已动刀） |
| gaea-dsh-ops-distill-2026-11-10.md | dsh-ops 插件蒸馏调研（2026-11-10；同日三波落地已收官） |
| gaea-wenxue-tools-survey-2026-11.md | 图文（wenxue）板块工具/技能梳理调研（2026-11-15） |
| gaea-home-redesign-survey-2026-10.md | 首页重设计调研+实施档（2026-10-27 十三连推送） |
| gaea-runtime-monitor-redesign-2026-11.md | 运行监测 UX/UI 重构（2026-11-03） |
| gaea-voice-memoryhub-ui-20261109.md | 2026-11-09 语音交互窗口记忆中枢化 |
| gaea-voice-port-distill-20261107.md | 语音 Go 移植蒸馏审计（上游 speech-to-speech v1.1.0 ↔ Go realtime 包） |
| gaea-voice-upstream-audit-20261107.md | 语音实时管线全量对源审计（2026-11-07） |
| gaea-voice-upstream-audit-20261104.md | 语音管线复刻审计台账（2026-11-04） |
| gaea-lingxi-re-distill-2026-11.md | 灵犀（WPS 灵犀 1.3.36）逆向蒸馏事实源（2026-11-13，P0 已落地） |
| gaea-lingxi-advantages-2026-11.md | 灵犀值得抄的 19 个设计（2026-11-13） |
| gaea-lingxi-distill-report-2026-11.md | 灵犀蒸馏评估报告（2026-11-13，逆向收官产物） |
| gaea-lingxi-optimize-plan-2026-11.md | 灵犀蒸馏落地计划（2026-11-13，P0 已收官装机） |
| gaea-lingxi-prompt-order-map-2026-11.md | 灵犀↔gaea 提示词装配顺序对照（2026-11-13，B3 产物） |
| gaea-orca-pantheon-distill-2026-11.md | Orca 1.4.214 × 众神殿机制蒸馏（2026-11-02） |
| gaea-orca-pantheon-distill-round2-2026-11.md | Orca 第二轮蒸馏 |
| gaea-pantheon-deliberation-frameworks-research-2026-11-03.md | 众神殿讨论/辩论框架专项调研（2026-11-03，证据附录） |
| gaea-pantheon-openmausbot-distill-2026-11-07.md | OpenMausBot × 众神殿第一轮蒸馏（2026-11-07，G0 证据冻结） |
| gaea-pantheon-openmausbot-distill-round2-2026-11-07.md | OpenMausBot × 众神殿第二次蒸馏计划与落地 |
| gaea-pantheon-agorai-aidebate-distill-2026-11-07.md | Agorai × aidebate 双源蒸馏调研（2026-11-07，只调研不立项） |
| gaea-pantheon-agorai-aidebate-distill-round2-2026-11-07.md | 双源蒸馏第二轮增量+两本账五刀落地（2026-11-08） |
| gaea-pantheon-seat-hang-cache-2026-11.md | 众神殿席位卡死根修与 DSH 前缀缓存（2026-11-06，证据档） |

## 当日总账（流水记录，按日留存）

| 文档 | 内容 |
|---|---|
| gaea-day-log-2026-11-22.md | 2026-11-22 当日总账（编程板块·DSH 隔离栈全量升级） |
| gaea-day-log-2026-11-20.md | 2026-11-20 当日总账（设置中枢全线收官+闭环后增量五批） |
| gaea-day-log-2026-11-19.md | 2026-11-19 当日总账（众神殿四笔+办公 R2 恢复契约包） |
| gaea-day-log-2026-11-18.md | 2026-11-18 当日总账（codex 逆向二轮 C 级+版本对齐 0.170+差距档案四刀） |
| gaea-day-log-2026-11-14.md | 2026-11-14 当日总账（CAD/进度计划回归右栏一级 Tab + 办公打开 CAD） |
| gaea-day-log-2026-11-13.md | 2026-11-13 当日总账（better-sidebar G1.6/G1.7/G1.8/G2 全清+真机走查双实证） |
| gaea-day-log-2026-11-12.md | 2026-11-12 当日总账（办公板块上下文界面升级换代 dsh-context 0.83.x 全量复刻，v4.542.0） |
| gaea-day-log-2026-11-11.md | 2026-11-11 当日总账（knip 尾款清零+天璇退役漏网收口＋WHIRL 推理引擎接入 v4.536.0） |
| gaea-day-log-2026-11-10.md | 2026-11-10 当日总账（办公板块工具治理：众神殿六阶段一天收官） |
| gaea-day-log-2026-11-09.md | 2026-11-09 当日总账（识图诊断线收官五刀+成本导入兜底腿换正道） |
| gaea-day-log-2026-11-08.md | 2026-11-08 当日总账（编程板块回流+记忆图谱真 3D 脑网格+语音星团随声互动，v4.535.0） |
| gaea-day-log-2026-11-07.md | 2026-11-07 当日总账（语音管线翻车双根修+五路蒸馏审计+三批修刀收官） |
| gaea-day-log-2026-11-06.md | 2026-11-06 当日总账（聊天板块核心优化第一批+实弹基线+臂B定案） |
| gaea-day-log-2026-11-05.md | 2026-11-05 当日总账（语音 V2 唯一路径化大日） |
| gaea-day-log-2026-11-04.md | 2026-11-04 当日总账（三引擎退役+Strata flash 档线） |
| gaea-day-log-2026-10-30.md | 2026-10-30 当日总账（众神殿三连发） |
| gaea-day-log-2026-10-29.md | 2026-10-29 当日总账（双版本日） |
| gaea-day-log-2026-10-28.md | 2026-10-28 当日总账 |

## 子目录

| 目录 | 内容 |
|---|---|
| codex-distill/ | codex 0.165.0→gaea 蒸馏的计划与施工档（入口 codex-distill/README.md；资产卡 63 篇） |
| code-audit-2026-10-02/ | 2026-10-02 全仓代码审计（27 单元 + 6 分册 + 2 附录；入口 code-audit-2026-10-02/README.md）；`.gaea/AGENTS.md` 的 God 类型冻结约定引用其 decision-brief |
| code-audit-2026-10-04/ | 2026-10-04 第三轮全仓审计收敛报告（入口 code-audit-2026-10-04/README.md） |
| distill/ | MuMuAINovel 蒸馏规格与实施交接书——**小说板块已于 2026-11-15 退役**，t2~t7 未实施，仅存档 |
| research/ | agorai-distill / aidebate-distill / zcode-distill 原始调研稿目录 |
| snapshots/ | 目检截图 12 张（2026-09-03，侧边栏 / 浏览器面板证据） |
| archive/ | 归档目录；逐份登记见 `docs/archive/README.md` |

## 归档区（不在此处展开）

`docs/archive/` 收录历史调研、已落地计划、被后续结论取代的文档，以及版本与进度磁带。**逐份登记见 `docs/archive/README.md`**——
引用归档结论前先确认未被现行规则取代。历史文档里写作 `docs/<文件名>` 而在顶层找不到的，多半已在该目录。
