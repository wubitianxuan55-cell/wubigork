# gaea 项目记忆

> 本文件为项目长期记忆（文档记忆层级）。编码规范：**UTF-8 无 BOM**（历史遗留的 GBK/UTF-8 混合编码已清理）。
> 修改后请保持 UTF-8；.ps1 脚本需 UTF-8 带 BOM（见「沙箱环境备忘」）。

## 版本状态（顶部速览）

> 速览只留最近 3 版（2026-09-15 整段分流；2026-10-21 十八迁分流：v4.454 纠偏条+v4.437~v4.428.1 十一版+审计四批+旧分流记录三条入 `docs/archive/agents-version-history-2026-09.md` 段首，水位 58661→约 29KB）；更早版本全部见 `docs/archive/agents-version-history-2026-09.md`（v4.49 及更早、v4.50–v4.145 仅 CHANGELOG/releases 有记录）。
- **未发布续刀：众神殿最终文档、观点流、Markdown、ZCode 事件墙、当前席重试与跳过（2026-10-30）**——主持人汇总完成后现在自动生成独立 Markdown 最终文档，写入 `DataRoot/pantheon/results/<sessionId>.md` 并把绝对路径存回 `Session.ResultPath`；文档含最终建议、适用条件、保留分歧、风险、证据、参与者和各席观点，历史完成会话首次打开时按需补生成。完成页新增“打开最终文档”，由 `PantheonOpenResult` 安全打开该会话自己的结果文件；手动完整 Markdown/JSON 导出仍保留为审计链。左栏档案可独立删除；共享主线程按实际 `createdAt` 顺序展示各 Agent 公开观点，流式观点实时进入；Agent 单独详情与共享观点均走 `MarkdownContent`。ZCode 同行连续 JSON 事件墙由 `json.Decoder` 拆分，流式 writer 增量消费完整对象，只保留 `text_delta`/已知正文；历史事件墙读回时尝试重装，无法恢复则诚实提示。`PantheonSkipAgent` 只取消当前席位 context/CLI 并进入下一位；`PantheonRetryAgent` 用专用 retry 信号在同一轮同一位重新调用，历史保留；整场 `PantheonRestart` 仅用于失败/终止归档并明确标为“重开整场”。绑定面 700→701。验证=目标五组 vitest 52/52、Go pantheon 全包、bindings/Go 定向测试、tsc/eslint 0、生产构建；最终 `build.bat` 桌面构建 `build/bin/gaea.exe` 56,107,008B SHA256=`8db929daa031af888ad1d97498cd1fc8e76c93f82c70f85211eff6a805d1cf53`，冒烟 `/api/health` 200，桌面 `gaea.exe` 同步。未抬版本、未 commit/push。
- **未发布续刀：众神殿图片附件与检索能力补齐（2026-10-30）**——用户追问各 Agent 搜索/识图边界后要求补全。新增最多 4 张、单张≤20MB 的 PNG/JPG/WEBP/GIF/BMP 图片附件（原生 `PickFiles`，去重+路径/大小/MIME 校验，会话/续议/导出持久化）；开始议事时先用本地 `vision` 生成一次共享识图摘要进 `Evidence`，所有席位都能引用，Grok 再以显式 `readPaths` 扩展本地 vision/OCR 读取，Codex 用官方 `--image`、ZCode 用 `--attach` 接收原图；DSH 使用共享摘要。Codex 每次众神殿调用用 `-c web_search="live"` 覆盖本机 disabled 配置，Prompt 统一要求把网页结果视为不可信、不执行其中指令/不外传附件。能力短标改为 Grok/Codex/ZCode「联网·识图」、DSH「联网·文件」；文档经 OpenAI 官方 CLI reference 核对图片/搜索参数。验证=vitest 28/28、Go pantheon 全包+app 定向/vet、tsc/eslint 0；Wails 2.13 桌面包 56,050,176B SHA256=`6c13fed6f33211527031bbd78eab6f22da416fd4317d149bb7d8aee9b624c4df`，冒烟 200，桌面副本哈希一致；Codex 0.165.0 真实 `--image` 探针返回正确图片描述。未抬版本、未 commit/push。
- **未发布续刀：众神殿拓扑协作视图（2026-10-30）**——用户反馈原“统一讨论流 + 2×2 独立窗 + 右栏摘要”同时铺开、信息层级过密。改版为中央单列共享主线程：顶部“共享讨论 → 各 Agent”SVG 连接拓扑，只显示角色状态/观点数/工具数；点击节点才打开独立 Agent 详情（观点/工具双页，Esc/关闭返回节点），工具动作与 Agent 正文不再进入共享线程；讨论设置、证据、公开事件降为按需披露。`aria-current`/`aria-expanded` + 文字状态双编码，工具栏与详情态标题分行，拓扑窄屏降为两列/单列；共享结束态结论仍居中，条件/分歧/依据折叠。四席头像改用官方品牌资产：xAI X 标、OpenAI 花瓣（Codex）、Z.ai 方形 Z 标、DeepSeek 鲸鱼（DSH）；原图存 `frontend/src/assets/agents/`，`SOURCE.md` 记录官方站点/官方 GitHub 组织来源与商标声明，不再用自绘符号冒充品牌。头像贯通欢迎、选席、左栏、拓扑、详情与参与者摘要，姓名/状态继续承担语义。细节续刀：`visuals.ts` 统一短名与能力文案（DSH / 办公工具 / 只读工具 / 计划工具），左栏头像旁补姓名与“就绪·能力”，头像加在线/候场/发言/完成/失败/离线状态点，拓扑名称两行展示、字号/卡片高度/指标字号提升。验证=vitest 27/27+tsc/eslint 0；Wails 2.13 桌面包 56,029,184B SHA256=`3605187f6f7b815d3272463752f334bbfc637e219249455f4e0e8325ba45ca3b`，`/api/health` 200 冒烟通过，桌面副本哈希一致；IAB 1440 暗/亮、390 窄屏与详情层实拍。未抬版本、未 commit/push。
- **未发布续刀：众神殿四席身份色（2026-10-30）**——四席品牌色从头像/边框扩展到欢迎席位、选席、发言顺序、统一讨论流、工具轨迹、2×2 独立窗与右栏名单；身份色只作线索，正文按主题混色保对比，状态仍走图标+文字。新增带姓名“席位色标”图例；vitest 25/25+tsc/eslint 0+生产构建+CDP 1440 亮暗实拍。未抬版本、未 commit/push。
- **未发布续刀：众神殿当前发言者高亮（2026-10-30）**——运行态新增“当前发言”状态条+消息“正在发言”徽标；当前席用 2px 身份色描边、双层外辉光、底色与轻微上浮，其他席不透明度降至 68%；本轮顺序/右栏/独立窗同步，`aria-current`+文字双编码。`activeSpeakers` 合并 participant=running 与 liveStream=streaming，避免状态落后一拍。vitest 25/25+tsc/eslint 0+生产构建+CDP 1440 亮暗实拍流式态。
- **最新发布：v4.529.1（2026-10-30）「小结标题挤压根修与亮态目检补课」**——v4.529.0 断行修复的补完刀（2f01cc87），纯 CSS 一条规则绑定面零变更。根因=小结头部 flex 行被 tag+印记预留(72px) 挤压，栏展开时标题只剩 256px 绞回四行——v4.529.0 的两行验收恰逢折叠栏态成立（单态验过≠修复成立，v3 轮复拍只拍了无小结面板的失败态漏过）。修复=head flex-wrap:wrap+标题 `flex:1 1 100%` 恒占整行+tag 换行右对齐，字号 27→24px；亮态暖纸×暗态、展开栏×折叠栏四象限实拍两行成立+vitest 19/19+冒烟 200。纪律=flex 收窄类断行目检必须覆盖主列宽度全布局态；沙箱 localStorage 跨壳存活走查前显式重置。产物=`releases/gaea-v4.529.1.exe` 55,948,288B SHA256=`88ea4cdcd8352464019fe30c7f7637671000e1339b4e11215834d10fcbcbcda7`（仅删 v4.526.0.exe，最近五版=v4.529.1~v4.527.0）。文档=`releases/v4.529.1.md`+CHANGELOG/README/releases README+progress/AGENTS。
- **最新发布：v4.529.0（2026-10-30）「众神殿 UI 实拍走查五刀」**——沙箱壳实拍五态定刀的界面打磨批（fd3ea7ee），绑定面零变更纯前端。①失败/中断信息三重去重（session.error 曾在头部红提示+黄小结卡+右栏警示三处复读→头部唯一出口+小结卡改行动指导+警示列表 filter≠error）；②CJK 断行修复（**ch 限宽按拉丁 0 字形宽，CJK 每字 1em 实际减半**——welcome h2 手工两行断行+字号按档案栏展开后最窄列放字〔vw 不感知 rail 宽〕，verdict h3 24ch→min(100%,20em)+line-break:strict）；③探针未回 welcome 全套「检测中」态（原「0 位就绪」红点误导，is-probing 点置灰，探针后恢复红绿语义）；④grok mini 卡不可用显示「内置」非「设置」（死按钮误导根除）+is-fixed cursor+tooltip。⑤核实默认选前 3 可用席位逻辑正确不改。验证=vitest 19/19+tsc 零错+沙箱三轮重拍目检；build.bat 冒烟 200+桌面副本内置同步。产物=`releases/gaea-v4.529.0.exe` 55,948,288B SHA256=`d53a1230161128f9fde9320ad4c30ec0d71d59a9e1426d543edf41a71750e5ba`（仅删 v4.525.0.exe，最近五版=v4.529.0~v4.526.0）。文档=`releases/v4.529.0.md`+CHANGELOG/README/releases README+progress/AGENTS。坑=vitest 必须从 frontend/ 目录跑（repo 根落无 jsdom 环境）。
- **最新发布：v4.528.1（2026-10-30）「众神殿两轮实弹修复与导出双门根修」**——v4.528.0 发布后的实弹修复批，绑定面零变更。①三崩点根修 bed2b286：dsh 席 MISSING_CREDENTIAL（SetDSHAPIKey 注入 DEEPSEEK_API_KEY，空 key 不注入防遮蔽）；起议 null 崩页（Contributions nil→null 三层修复=创建点空切片+store 读回 normalize+前端三处 ?? 兜底）；单席超时 4→8 分钟。②导出双门根修 e078043a：pantheon 导出裸 `<a download>` 壳内不落盘（v4.162 实证）→ saveExportBlob 双门（壳内 GaeaSaveFileAs/浏览器回退）+取消诚实反馈+Request.AgentIDs null 归一化；沙箱壳实弹验收=#32770「保存文件」对话框弹出+取消 toast+零残留（真存落盘腿留用户，写入层生产自用 v4.162）。③浅色白字两根修 60ce7f15（DSH 并行线）：绘梦生成钮实底化+运行监测幽灵令牌接 tertiary。验证=go test pantheon/app 绿+PantheonPage 18/18+mock 83/83+tsc 零错+真机四席圆桌两轮实弹全链过。产物=`releases/gaea-v4.528.1.exe` 55,947,776B SHA256=`ca132527ba5f9487f44c3cf27f73c97751b4e28d9f644e9d25b5e62ea1a00515`（构建冒烟 200；仅删 v4.524.0.exe，最近五版=v4.528.1~v4.525.0）。文档=`releases/v4.528.1.md`+CHANGELOG/README/releases README+progress/AGENTS。
- **最新发布：v4.528.0（2026-10-29）「众神殿：多 Agent 圆桌、至共识讨论、实时工具轨迹与 Grok 办公工具接入」**——用户要求继续完善众神殿：左栏可折叠、讨论次数手动 2–20 或不限至全员一致、各 Agent 工具可用且实时显示轨迹、Grok 接入 Gaea 办公板块工具不再裸跑。落地=共享工作空间圆桌、顺序轮换、各席独立判断链、不限轮次 12 轮安全上限、结果优先小结与续议、档案搜索/筛选、Markdown/JSON 导出、重启中断恢复、长提示首尾保留；Codex 0.165.0/ZCode 0.17.0/DSH 0.2.1-rc.3 真机只读工具探针通过，统一解析 `item.started/completed`、`tool_call/tool.updated`、`tool_call/tool_result` 进入实时轨迹。Grok 由 Gaea 办公 AgentRunner 每轮按议事工作空间新建只读注册表，复用文件/造价/知识/记忆/日程工具，只复制 `ReadOnly()==true`，禁止 `write_file/edit_file/bash/task/install_skill`；脚本化 provider 验证完整工具循环，真实 xAI 工具调用未纳入本次门禁。发布前完整本地快闸全绿（Go 全仓+lint+前端 465 文件/4053 例+生产构建+E/文档守卫）；`internal/pantheon` lint 曾逮 `updatePhase` 无调用并删除。产物=`releases/gaea-v4.528.0.exe` 55,947,264B SHA256=`2215fe0a18a74b962d9588887c6a3e935f990220882b7812291a0e8deae52b8f`（构建冒烟 200；仅删 v4.523.0.exe，最近五版=v4.528.0~v4.524.0）。文档=`releases/v4.528.0.md`+CHANGELOG/README/releases README+`docs/gaea-pantheon-market-research-2026-10-29.md`+progress/AGENTS。race 未跑（本机缺 gcc/CGO），由 Actions 覆盖。
- （2026-10-21 非版本刀·releases 批量整理）用户反馈「前期版本的文档太多了」——releases/ 根曾堆 **1256 项**（686 说明+492 校验和+33 旧线 CHANGELOG txt+37 源码包+5 exe）。V4_RECENT 34 席以外 **653 份发布说明+458 份校验和+33 份 CHANGELOG-v\*.txt+ALIGNMENT_PLAN_v6.md** 平铺迁入 `releases/archive/`（沿用既有两级结构——release.ps1 6b-2 按根/archive 直属计数、check-docs 全递归，**零脚本改动**）；全仓 **1000 处** `releases/vX.md` 引用随迁改写（CHANGELOG 476/progress 101/AGENTS 14/docs 族 300+）；根目录瘦至近 34 版说明+校验和、5 exe、源码包与索引。坑=heredoc 反斜杠又降级（python 一律 Write 脚本文件再跑）。
- **最新发布：v4.457.0（2026-10-07）「造价数据库重塑为工料法成本数据库（工料机资源库 + 五表模版）」**——用户指令「根据成本测算模版重新设计整个造价数据库，采用综合单价带工料机的方式」+「工料机是基本数据，综合单价分析表靠工料机和消耗定额核算出来，就是建立工料法成本数据库」。**绑定面 635→654（+19 全归 CostB）**。【口径纠偏】实测最新产物（旺平矿业/什邡五表）封面自述「综合单价只含人材机，管理费/利润/税金不进综合单价」——早期取费内联进单价已彻底分离（ComposeUnitPrice 只出人材机 / ComposeProjectFees 项目层跑一次，FeePolicy 可切利润基数含不含规费）。【数据层 V25/26/27】gf_resources+调价历史（取用价=现行价优先、为 0 回退基准价）、gf_quotas+gf_quota_items、cost_entries.price_derived（price 保留列名契约不变、语义降级为核算缓存，20+ 消费方零改动）、cost_estimate_items 10 列 + cost_projects 11 列。【三层核算】外委升一等行类（材料 = SUMIF(材料)+SUMIF(外委)）、零含量是合法占位（计 ZeroLines 不告警，只有负值非法）、五表工作簿解析/导出（活公式 VLOOKUP+SUMIF+取费基数链）。【存量已落真实库】1590 条 → 资源 1486（人工 34/材料 1354/机械 64/外委 34），跳过 104（综合单价条目 103 保留在综合单价层），失败 0/无价 0，幂等复跑零新建；命令 `scripts/migrate_workcost`（默认干跑）。老库升级链另验：v7→v27 零损失。【前端】造价数据库新增一级模块**工料机**（资源库/单价分析 + 导入项目表/导出五表）。【修复】①mapGaea 漏 `GaeaWorkcost` 前缀致 19 方法静默落 office 门面（调用仍成功因 proxy 按名遍历搜索，但域错）→补规则归 CostB；②**生成器 CRLF 静默失配**（正则 `$` 在 Go (?m) 不匹配 `\r`，带注释的行因 `.*` 吞 `\r` 仍匹配、只有无注释行失配 → 「新加的最后一行偏偏没被认领」）→ 容许 `\r` + 反向验证；③组成行类别判定短路（normalizeKind("") 返回「材料」被当已标注，挖掘机台班归成材料）；④SaveRecomposed 独立成方法（来源声明不可并入 Save）；⑤工作区行尾统一 LF。【测试】Go vet 0 + workcost 59 例（含 3 真实库探针）+ app 全包绿；前端 tsc/eslint 0 error + vitest 3786 例/439 文件全绿 + 生产构建 ✓；导出经 LibreOffice 重算与独立口算逐分一致；exe 内 13/13 绑定名+6/6 前端文案+3/3 表列名二进制核验。**产物**=releases/gaea-v4.457.0.exe 52,342,784B SHA256=`36e6f36909dcfbea46cc2b6ae0a325f722d4bf13f831470cc24139b35f59786d`（冒烟 200 过；删 v4.452.0.exe）。**文档**=releases/archive/v4.457.0.md+CHANGELOG/README+releases/README。
- **最新发布：v4.456.0（2026-10-04）「微信媒体下载 SSRF 守卫对齐 + 第三轮审计结构收敛」**——用户指令「继续」。**绑定面 635 零变更**。【修复】weixin 媒体下载 dialer 补齐 IN2-04 failover（第三轮审计 §3.3 指认的副本漂移）：批 1 已修进 netclient.GuardedClient 与 webfetch directDialContext 的「校验全量解析 IP 后逐个已验 IP 尝试建连」在 weixin mediaTransport 漏修——固定拨 ips[0]，单一目标连接失败即整体失败（无回退）；抽 `dialVerifiedIPs` 助手+TestDialVerifiedIPs 四态（成功即停/前败后备/全败回最后错误/空集诚实报错），安全语义零变化（全量 blockedMediaIP 复检在前、拨号目标严格取自已验集合）。【结构收敛两批】①第三轮全仓审计收敛批 `0c09977e`：净删 ~1937 行（TS6133 权威清单 38 条/死文件 TrendChart+SubagentsPanel/死本地化键 26×3/死类型/死 CSS ~676 行两路交叉验证/Go 零调用方法 12 个+precheck delete_range 死分支/round4 双抄本→cost.Round4 单源）+**noUnusedLocals/noUnusedParameters 守卫开闸**（反向探针验证 TS6133 生效）+wails.json npm→pnpm；拦下 15 组「看着像死实则活」误报未删；报告归档 docs/code-audit-2026-10-04/。②§3.2 重复收敛批 `da89ca89`+续刀：errText 方言 A 14 份→utils/errText.ts 单源（方言 B 无 fallback 语义不同不并）/EMO_COLORS 9 色全等→再导出/Go strutil.ParseTagsJSON（cost+knowledge 5 调用点）+ContainsHan（hasHan/isCJStr 收敛，isCJStr 名不符实修诚实）/xlsxedit applyOne 253 行 switch→opHandlers 表驱动（summary 契约 11 例先行+unmerge_cells 零覆盖补测+键集冻结+fail-closed 图钉，xlsxedit.go 785→532）/rebuildParagraph 分组+区间计算→ooxml.GroupRuns/AffectedSpans 单源（发射端 w: 修订记号 vs a: 纯替换两方言各自保留；计算面单测钉住）。【审计证伪】novelModelName ×4（函数已不存在）/§3.4 ops.go 零覆盖（12 op 全有 4~47 处测试命中）。【测试】Go vet/golangci 0 issues+全量零 FAIL；xlsxedit 17 例零编辑全绿；前端 vitest 3755/435 全绿。**产物**=releases/gaea-v4.456.0.exe 52,047,360B SHA256=`e197ae508b5721ab31a3f2011d2d89fda6a7d0919dac88971b3c6ba0b22f63f1`（冒烟 200 过；删 v4.451.0.exe；速览迁 1 条：453）。**文档**=releases/archive/v4.456.0.md+CHANGELOG/README+releases/README+AGENTS 速览。
- **最新发布：v4.455.0（2026-10-03）「全仓审计修复收官：512 条审计 50 批清账 + 绑定面 744→635」**——2026-10-02 全仓审计（512 条/P0 25）修复线程收官：46 修复批+2 文档批（round-1~56 台账 docs/code-audit-2026-10-02/），P0 25 全清，四簇/duplication/god-file 族/留池/上报未动池/观察池全清或归拍板；发布前 ci.ps1 全量档首次联合通过。**绑定面 744→635**（A1①：109 条零调用绑定摘除，生成器 excludedBindings 管线）。【要点】底稿串写/会话流生命周期 P0 根修+检索口径单源（语料指纹进 ranker key）+AtomicWrite perm 生效+WhisperSubgraph 真机空图根修（json 标签+键名钉）+god-file 族收官（controller.ts 884→141 hook 分域六文件）+estimateTokens 五处收敛+裸写/CreateTemp 非同构收口+AtomicWriteSync fsync 变体+任务身份三系统对照表（tsk_/dag_/ti- 格式钉）+NOT_MOCKED 45→0（漏 mock 锁 fail-closed）+反向变异三组。【测试】前端 vitest 3760/435 文件全绿；app 全包 135s 绿；全量档 CI OK。**产物**=releases/gaea-v4.455.0.exe 51,919,616B SHA256=`25bb44404f81419153878922d0661aa46708df4dd0b25c99a9e1e272c5048f55`（冒烟 200 过；删 v4.450.0.exe；速览迁 3 条：452）。**文档**=releases/archive/v4.455.0.md+CHANGELOG/README+releases/README+AGENTS 速览+decision-brief 裁决菜单。

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
   **「CI 绿」口径（2026-09-27 定）**：写门禁结论必须带定语——「本地快闸」=
   `scripts/ci.ps1`（构建/vet/golangci v2.14.0/Go 全量/漂移/前端/E系/卫生）；
   「Actions」=.github/workflows/ci.yml 三 job（backend+race+frontend，race 检测
   只在 Actions：本机无 gcc）。禁无定语「CI 绿」。
3. **定向优先，全量收尾**：改动后先跑目标文件（3–10s）确认；全量只在交付前跑一次。
   一轮改动 = 中间定向 + 收尾全量，**同一验证不重复跑**（上一轮全量跑了 4 遍=反例）。
   **工具（2026-09-27 立）**：`scripts/test-related.ps1` = git 改动 → Go 包
   （吃结果缓存，不带 -count=1）+ `vitest related`，几秒~分钟级；`ci.ps1 -Quick`
   = 静态闸（build/vet/lint/卫生守卫）+ 受影响面测试，实测 ~1 分钟（vs 全量
   5-10 分钟）。**全量档（ci.ps1 不带参）保留在 push 前/发版前跑**；
   用户对「几分钟」的抱怨=2026-09-27 拍板用两档制化解，勿再无脑全量。
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
- **真机走查沙箱开关（v4.418，必用）**：`GAEA_WALKTHROUGH=1` 启动壳=workspace 强制指向
  `%TEMP%gaea-walkthroughworkspace`（覆盖用户/项目配置），会话/记忆/交付物全落沙箱，
  走查后删沙箱目录即清场——首条消息自动装配+自动恢复会把回合接进真实会话，此开关是
  唯一根治（2026-09-26 险情防复发）。未设时零行为变化。
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
- **审计衍生优化方向（2026-09-26）= `docs/gaea-optimization-direction-2026-09.md`（候选池；**P0-1 门禁落地 + P0-2 comfyUI 竞态已于 2026-09-27 落地**，见该档 §3 状态注记）**：
  本次全仓审计（三路并行模块审计 + 本机实测复现）——综合 7.0/10 + 分维与 14 模块评分 + P0~P2
  优化项（门禁落地 / internal-app 竞态 / 台账自动化 / 上帝包拆分 / 281 遮蔽绑定 / 平行实现收敛 /
  效果验证闭环 / 甘特库评估 / 仓库体积）+ 本次整理清单 + 未验证项观察池。**排刀前先读它的 §3 与 §5**；
  与既有权威路线冲突时，仍以本段规划为准。
- **用户再拍板（2026-09-08）：工位与乐园并列平级**——办公不是乐园的上位核心，瘦身
  不是「收乐园保办公」；认知面问题=13 板块平铺单一导航面、并列结构未表达。P2 措辞
  已由「乐园折叠」改为「双空间并列落地」，乐园板块在乐园空间内一级可见。
- **唯一权威路线图 = `docs/gaea-nextgen-roadmap-2026.md`**（11 个子代理调研合成）：
  8 板块竞品调研（办公/造价/AI 底座/编程/小说/绘梦/轻语/微信+语音）+ WorkBuddy×灵犀
  深度拆解（§12）+ **版本重定义"双空间"（§10）** + 四层落地（§13 后端/前端/UX/UI）+
  执行计划（§14 阶段 0 地基 → 阶段 1 双空间内核 → 阶段 2 双空间壳 → 阶段 3+ 领域包）。
- **用户拍板：工作与娱乐分开、互不干扰**——工位（办公/造价/资料+工作记忆）与
  乐园（轻语/小说/绘梦/阅读）双空间硬隔离；记忆分区互不检索、模型策略各配各的、
  上下文永不跨界；跨空间仅用户显式发起（如"把乐园封面放进报告"）。
  （编程板块 v4.439 经用户拍板「删除编程板块」整块移除：manifest/页面/Go 绑定
  五方法/暗色 mock 面全部退役，independent 分面归零但机制保留。）
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
  `releases/archive/v4.8.0.md` 欠账清单。
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
- **God 类型冻结约定（2026-10-03 批 52，A3③ 裁决）**：`App`/`mediaState`/`writingState` 等历史巨结构体
  维持现状不拆（嵌入地层已稳定，再拆收益近零——round-31 判定）；**新增字段必须挂子结构体**
  （如 cfg/media 子域），不得继续往顶层铺平；相关拍板项（X1-03/AgentRunner 收敛）按
  docs/code-audit-2026-10-02/decision-brief-2026-10-03.md §A3 处置
- **绑定面（v2.17.0 起；A1① 后 635 个导出方法）**：App 不再直接绑定 Wails；导出方法拆 11 个板块门面
  （internal/app/bindings_*.go：CoreB/OfficeB/MemoryB/CostB/ModelB/VoiceB/ChatB/NovelB/ImageB/CharlibB/SinB，
  纯委托零逻辑改动）。改绑定面方法后用 `go run ./scripts/gen_bindings` 重新生成 +
  `TestBindingsCompleteness` 兜底；摘除绑定走生成器 excludedBindings 清单（A1① 109 名在册）；前端调用经 gaea/lib/bridge.ts（按方法名路由门面）或
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
