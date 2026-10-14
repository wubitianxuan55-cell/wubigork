# gaea 同类产品竞品调研（2026-10）

- 日期：2026-10-14
- 方法：四路并行网络调研（GitHub releases/issues、官方 docs、Reddit/HN/少数派），时间窗 2025H2~2026-10。全部结论挂来源链接于文末。
- 前置拍板约束（本调研提案已遵守）：不做 MCP 接入、不做平台化、不做跨设备同步（含跨设备记忆包）、GoalCard 系勿再提。平台方向另见 `gaea-platform-market-survey-2026-09-15.md`。

## 一、赛道地图

### 1. 通用 AI 桌面/自托管客户端

| 产品 | 热度 | 2026 重心 | 主要抱怨 |
|---|---|---|---|
| Cherry Studio | 52.6k★ | v2.0 Agent 平台（自研 Pi runtime+沙箱）；手机 LAN 配对桌面端；Channels 接企微/钉钉；ComfyUI 工作流接入 | 划词助手呼声最高（372👍）；v2 被批不如 v1；长会话 2 万 token 后 UI 卡 |
| ChatBox | 42k★ | Work Mode（AI 动本机文件）+Skills 库+持久记忆+上下文摘要压缩 | iOS 下架风波；订阅计费信任差 |
| LobeChat→LobeHub | 83.2k★ | 转向 Agent 协作平台（常驻队友/群组共享上下文/teams） | 自托管全家桶依赖；迁移丢数据事故 |
| Open WebUI | 155.6k★ | Realtime 语音通话（闲聊/任务分层转交）；工具逐次审批；Skills 自动发现 | "最臃肿"；改 license 引分裂 |
| Jan | 44.9k★ | Projects；OpenClaw 集成；llama.cpp 0.5+MTP 多 token 预测；消息版本分支 | 模型下载失败最高票；要本地 Whisper |
| AnythingLLM | 67.1k★ | 聊天内直出 PDF/Excel/Word；NPU 原生；会议转写 | RAG 检索/分块质量长期被诟病 |
| Msty | 闭源 | 付费 Studio：Knowledge Stack NG（重排/PII 清洗）；Crew Mode 群聊；工作流 Turnstiles | 免费版弃更（"Msty is dead"）；检索效果被质疑 |
| OpenClaw | ~392k★ | 本机 Gateway+20 余消息渠道+ClawHub 技能市场；turn 跨重启续跑 | 安全噩梦：实例暴露/恶意技能/明文密钥/沙箱默认关 |
| Goose (Block) | 55.3k★ | ACP 状态机循环；桌面实时语音；models.dev 动态元数据 | 热度低于同类；最高票需求=细粒度 shell 权限 |

**军备竞赛五面**：①Agent 运行时自研化（沙箱执行/逐工具审批/turn 可恢复）；②入口反转（手机/IM 经 LAN 或渠道插件接入桌面，gaea 不跟进——涉网络服务面）；③记忆与上下文工程标配化（持久事实记忆+自动 compaction）；④技能资产分发（SKILL.md 目录+市场成为新分发单元）；⑤本地引擎+硬件矩阵（llama.cpp 版本/MTP/MLX/NPU）+语音第二增长点。

**信任教训三则**：Msty 免费版弃更伤口碑；Warp 激进 credit 化遭反噬；Open WebUI 改 license 引分裂。免费可用是此赛道信任基线。

### 2. AI 长篇小说写作

| 产品 | 定价 | 招牌机制 |
|---|---|---|
| Sudowrite | $10/22/44 积分制 | **Saliency Engine**（生成前按相关性初筛实体卡，不全量塞）；**Chapter Continuity**（回看≤25 前文档共 2 万词+当前 2 万词+章节摘要，超限按固定次序裁剪：世界观→角色→前章正文→大纲摘要→文体→风格）；Muse 模型 NSFW 几乎无过滤 | 
| NovelCrafter | $4~20 BYOK | **Codex 四态注入**（Always/检测到名字·别名才注入/检测不注入/Never）+别名表（复数/大小写/排除）；**Progressions**（实体描述随章节演进=状态化实体卡）；Relations 牵引注入 |
| NovelAI | $10/15/25 | **Lorebook**（关键词/正则/&多键、Search Range 默认 1000 字符、Token Budget 按上下文百分比、递归级联、插入位置）；模型转向微调大模型（Xialong=GLM-4.6 355B 微调，28k 滚动窗口）；自研模型默认无审查 |
| SillyTavern | 免费开源 | **World Info 三策略**（constant/关键词/vectorized 向量召回）+扫描深度/递归步数/Sticky·Cooldown·Delay 定时效果/@Depth 插入/按 Context% 预算/互斥组；Data Bank 本地 RAG；预设 JSON 生态 |
| DreamGen | 免费+$8/19 | 无审查+硬红线；steering 控制剧情文风 |
| 蛙蛙写作/笔灵/彩云小梦 | 中文 | 工作流 Agent"10 分钟万字"/爆款拆解+秒出大纲/多走向互动续写；一致性机制披露少 |

**横向结论**：
- **长篇一致性四层谱系并存**：①关键词实体卡检索（ST/NovelAI/NovelCrafter，普遍被吐槽误触/漏触）→②语义显著性筛选（Sudowrite Saliency，唯一官方方案）→③滚动摘要+前文回看（Sudowrite Continuity）→④向量召回（ST vectorized）。知识图谱只有轻量形态，无成熟产品。
- **口碑共识：没有银弹**。20–30 章/约 3 万字后连续性崩坏是通病；更深层痛点是"工作单元与版本管理（哪一版是正典）"。
- **NSFW 2026 现状**：NovelAI/DreamGen/Sudowrite Muse（自研）明确允许；通用大模型 API 收紧+年龄验证。被拒写用户流向 NovelCrafter+OpenRouter、ST 本地、NovelAI/DreamGen。**gaea 的 NSFW 友好+本地推理组合正对迁移风向。**
- **差异化空位：没有一家做"长跑一致性评测/健康度验证"**——四家全部在注入侧发力，缺验证侧机制。gaea 的评测线（病历画像/张力聚合/基线对比）在业界无对应物。
- gaea 已有的"每 10 章阶段+起承转合教义、50 字分支压缩"在上述产品中均无对应物。

### 3. 记忆/知识库 × AI + 本地模型生态

**记忆机制三条路线与 2026 收敛**：
- mem0：从"写时 LLM 裁决（ADD/UPDATE/DELETE）"转向 **ADD-only 单遍累积+读时解冲突**，召回=语义+BM25+实体三路融合（top-200 预算）+时序排序。
- Zep/Graphiti：**双时态知识图谱**——事实冲突不删旧边，打 invalid_at（保历史可回溯）；召回=余弦+BM25+图 BFS 三路+重排。
- Letta(MemGPT)：记忆块常驻上下文，agent 用工具自编辑（"活的 CLAUDE.md"）+sleep-time 后台重组。
- HN 出现「记忆即文件格式」潮（SQLite/Markdown/Git-native）：共识批评=纯向量对时序敏感事实失效、图谱复杂难维护、LLM 抽取成本高（10 万条 $1-3k/月）。
- **收敛方向：本地单文件存储 + ADD-only 写入 + 读时混合裁决 + 时间戳/失效标记 + 可视化审删**（OpenMemory 审删 UI 已成隐私标配）。
- **gaea 的 auto-dream 建议制（确认才入库）与该收敛方向一致，是差异化**；桌面本地优先+建议制在这个区间是空位（Reor 已停更、OpenMemory/Khoj 都是服务器进程形态）。

**本地模型生态（2026-10）**：
- 桌面集成约定：Ollama `11434`（num_ctx 默认仅 4096 要显式调大）、LM Studio `1234` OpenAI 兼容、llama.cpp 是共同底座。
- 讨论度最高：Qwen3 全尺寸（**30B-A3B 是低配甜点**）、GLM-4.5-Air 110B-A12B（消费机质量上限）、gpt-oss-20b（16GB 可跑 agent）；量化被 **Unsloth UD 动态量化**统一（UD-Q4_K_XL=公认默认）。
- **经验红线：凡调工具的模型 ≥UD-Q2_K_XL，工具调用更低比特会崩；长文写作容忍 Q2/Q3，agent/编码要 Q4+**。"harness 比 2 分基准差距更影响体感"。
- 本地 agent 形态：小模型当代理（Qwen3-4B 工具调用评测并列第一）；**"富 RAM 穷 GPU"**（MoE 专家权重卸载内存、注意力留显存）是大 MoE 流行的原因。

### 4. 社区呼声与趋势裁决

高频痛点排序：**上下文/记忆是第一痛点**（3 万字崩坏+正典版本管理）> 订阅费与配额焦虑 > 审查与成人向（用户流向本地）> 隐私与权限过宽（AI 插件读全库送第三方）> 桌面应用臃肿（反 Electron 情绪）。

2026 战场排序（证据裁决后）：**记忆与长期上下文 > 技能包分发 > 隐私本地化混合路由 > 垂直工作台**。订阅聚合被平台钳制（Anthropic/Google 封杀第三方接入订阅）；浏览器/OS 集成、语音是巨头战场，不宜对垒。

gaea 的造价算量、双代号进度**未见社区竞品，属差异化空白**（与既有刀路一致，继续走）。

## 二、候选优化方向（拍板池提案·一轮概览版）

> 2026-10-14 二轮深挖后提案已重排：见「六、拍板提案 v2」，以六为准。本节保留作一轮概览。

按「业界已验证 × gaea 落点现成 × 服务核心垂线」排序。每项含现状假设——执行前需先核实 gaea 当前实现。

### 提案 1｜小说线：实体卡触发注入 + Token 预算（业界标配补课）
- **业界机制**：NovelCrafter Codex 四态（Always/名·别名检测/检测不注入/Never）+别名表；NovelAI 触发词/正则/扫描范围/Token Budget 按上下文百分比/递归级联。
- **gaea 落点**：项目角色+设定实体当前疑为名册/大纲路全量式注入。升级为：默认按「名字·别名检测」注入，关键实体可标 Always 兜底（社区对纯触发的吐槽=误触/漏触）；别名表对中文网文刚需（外号/敬称/乳名）；注入总预算按上下文百分比封顶。
- **刀深**：Go 注入链改造+设置面；中刀。

### 提案 2｜小说线：状态化实体卡（NovelCrafter Progressions，业界独门机制）
- **业界机制**：实体描述随章节演进（第 N 章后自动换状态），注入按当前章号取「截至该章最新状态」。
- **gaea 落点**：角色/设定卡加状态版本列表（章号+描述），与既有「场景五字段版本」同构，机制现成。直击 20-30 章人设漂移通病；黑化/受伤/易容是网文刚需。
- **刀深**：types+SQLite 迁移+注入链；中刀。

### 提案 3｜小说线：前情回看包显式化（Sudowrite Chapter Continuity）
- **业界机制**：回看≤25 前文档共 2 万词+当前文档 2 万词+章节大纲摘要；超限按**固定次序裁剪**（世界观→角色→前章正文→大纲摘要→文体→风格）。
- **gaea 落点**：已有 stage-recap（10 章合账）+分支 50 字压缩（摘要层），疑缺正文级回看与裁剪次序。建议收成单一「上下文组装器」函数：全部阶段摘要+最近 1~2 章正文尾段+本章计划，定义裁剪次序；工坊预览可看完整组装包。
- **刀深**：提示词链收敛重构；中刀。

### 提案 4｜记忆中枢：混合召回（BM25+实体路）+ 时间戳/失效标记
- **业界机制**：mem0 新算法三路融合（语义+BM25+实体，top-200 预算+时序排序）；Zep 双时态（冲突不删旧、打 invalid_at 可回溯）；HN 共识=纯向量对时序敏感事实失效。
- **gaea 落点**：内部向量检索补 SQLite FTS5 全文路（零新依赖）+实体关键词路；记忆条目带 created_at/invalid_at；auto-dream 入库时与相似旧记忆比对，冲突→旧记忆标记失效（不删）并在建议里提示。
- **刀深**：检索链+schema 小迁移；中刀。

### 提案 5｜小说线：长跑健康体检常驻化（差异化空位，业界无对应物）
- **现状**：评测快照/基线/对比/张力已有但被动看；业界四家全在注入侧、无验证侧。
- **建议**：每阶段（10 章）收尾时提示跑一次体检（建议制/手动触发，避免烧 token 重蹈 auto-dream 红线），hollow 画像→横幅预警「本阶段文味流失，建议回看第 X 章风格锚」。链路已在，产品化包装。
- **刀深**：小刀（前端流程+触发点）。

### 提案 6｜本地推理：agent 档位红线 + 甜点预设
- **业界机制**：工具调用需 Q4+（<Q2 崩）；富 RAM 穷 GPU（MoE 专家卸载内存）；Qwen3-30B-A3B/GLM-Air=甜点。
- **gaea 落点**：strata/IQ2_XS 已验证写作档。模型档位元数据加「适用面」标注；agent 场景选低于 Q4 本地模型时警告；若 llama.cpp 上游支持 `--n-cpu-moe` 则暴露旋钮+甜点预设。可与刀池 P2「模型路由」合并。
- **刀深**：小刀。

### 提案 7｜桌面通用面：全局呼出/划词助手（需先核实可行性）
- **证据**：Cherry Studio 最高票需求（372👍）；Witsy 主打此面；社区对"桌面 AI 应用臃肿/权限过宽"敏感，呼出窗须轻+本地。
- **gaea 落点**：Windows 全局热键+快速问答浮窗。**前置核实**：Wails v2 全局快捷键能力与划词取词（UI Automation/剪贴板）桌面权限面；不可行则降级为托盘快速入口。
- **刀深**：待核实后估。

### 观察项（不建议现在做）
- **技能包资产化**：与「规范知识技能化红线」同构，内置技能包目录是低成本跟进方向；但分发/市场面踩平台化排除项，仅记录趋势。
- **记忆导出/携带**：社区有"谁拥有 agent 记忆"治理议题+可携带愿望；与在册排除项「跨设备记忆包」相邻，动前须用户拍板边界。
- **语音/浏览器集成/入口反转（IM 接入）**：巨头战场或涉网络服务面，不对垒。

## 三、风险与教训（外部）

1. 免费可用是信任基线（Msty 弃更/Warp 收费反噬/Open WebUI license 分裂三案例）。
2. 订阅转借是雷区：Anthropic/Google 已封杀第三方 agent 接入订阅账号；gaea BYOK+API 路线安全，勿碰订阅聚合。
3. 若未来做任何技能包，OpenClaw 的安全叙事是反面教材：实例暴露/恶意技能外传数据/密钥明文——边界必须 fail-closed。
4. 臃肿是桌面 AI 通病（Open WebUI"最臃肿"、HN 反 Electron 情绪）：gaea 界面简化线与风向一致，继续守。

## 四、来源（按线）

通用客户端：github.com/CherryHQ/cherry-studio（releases/issues）、github.com/lobehub/lobehub、github.com/open-webui/open-webui、jan.ai/changelog、docs.anythingllm.com、chatboxai.app changelog、msty.ai、github.com/openclaw/openclaw、github.com/block/goose、docs.warp.dev、reddit r/LocalLLaMA（Msty/Warp/Goose 帖）。

长篇写作：sudowrite.com pricing+docs（Saliency/Continuity）、novelcrafter.com（pricing/Codex tracking/aliases）、docs.novelai.net（Lorebook/models）、blog.novelai.net（Xialong）、docs.sillytavern.app（WorldInfo/DataBank）、v2.dreamgen.com、raptorwrite.com、wawawriter.com、ibiling.cn、r/WritingWithAI（一致性/长篇崩坏/Claude+GPT 组合帖）、r/NovelAi、r/SillyTavernAI。

记忆/本地模型：github.com/mem0ai/mem0、arxiv 2501.13956（Zep 双时态）、docs.letta.com、hn.algolia.com（agent memory as file format/mem0 争议）、github.com/reorproject/reor（归档）、github.com/mem0ai/mem0/tree/main/openmemory、docs.ollama.com、lmstudio.ai、r/LocalLLaMA（GLM-Air/Qwen3 甜点/UD 量化）、docker.com blog（本地工具调用评测）。

社区趋势：hn 47633396（封 OpenClaw 订阅）、hn 47115805、hn 47479962（安全噩梦）、hn 46893970（Apple Intelligence 该有的样子）、hn 46709007（Claude 限额）、hn 48613162（桌面 AI 应用为何重）、sspai.com/post/105481·105918·102434、vellum.ai（记忆市场数据）。

---

# 二轮深挖（2026-10-14 续）

## 五、深挖发现

### 5.1 写作工具机制规格书（实现级，官方文档逐页抠出）

**SillyTavern World Info（开源参考实现，最完整）**

| 参数 | 语义 | 默认 |
|---|---|---|
| Scan Depth | 扫描最近 N 条消息；0=仅递归条目+AN | 2 |
| Context % / Budget | 注入上限=上下文百分比或绝对值，预算尽即停 | 25%（社区重度用户 40~75%） |
| Max Recursion Steps | 0=仅受预算限；1=关递归；2=一跳 | — |
| Min Activations | 回扫全日志凑够 N 条激活（与递归互斥） | — |
| 触发% / Inclusion Group | 概率触发；互斥组加权随机 | 100 / 每条 100 点 |
| Insertion Order / Position | Order 大者靠近末端；Position 含 @Depth+角色 | — |
| 副键逻辑 | AND ANY / AND ALL / NOT ANY / NOT ALL；正则键 `/…/flags` | — |
| 三种灯 | 🔵constant 无条件入 / 🟢关键词 / 🔗vectorized 向量召回（仍可关键词触发） | — |
| 定时效果（单位=消息） | Sticky 激活后保持 N 条免概率判定；Cooldown 激活后 N 条禁再触发（Sticky 结束才计）；Delay 聊天不足 N 条不触发 | 0 |
| 递归标志 | Non-recursable / Prevent further recursion / Delay until recursion | — |

生命周期：组装扫描缓冲（近 N 条+说话人名）→ 主键匹配（大小写/全词/正则）→ 副键逻辑核验 → 定时效果闸门 → 概率 roll → 互斥组裁决 → 激活内容纳入递归扫描 → 装填（constant 先占预算，再按 Order 高→低，直接命中优先于递归提及，**装满即停不做逐条裁剪**）→ 按 Position 插入。多 lorebook 合成：Chat→Persona→Character/Global。
**中文坑**：全词匹配默认开但对中日文有害必须关；复数自动变体只处理英语规则形式。

**NovelAI Lorebook**

- Keys 大小写不敏感；`/…/` 正则（flags i/s/m/u）；`A & B` AND。
- Search Range=被搜正文字符数，**max 10000**；Token Budget **0–1 开区间小数=占（maxContext−output）百分比，否则绝对值**；Reserved Tokens 先预留。
- Insertion Order 高者先插；Position 0=顶部、负数自底向上；**Key-Relative Insertion** 相对最后命中 key 插入；Trim Direction 含 **Do Not Trim**（放不下整条就不放）；Prefix/Suffix 在 trim 后附加。
- 级联激活=在非正文上下文条目中搜键；**Advanced Conditions：随机概率 0-100%、数值比较（currentStep/paragraphCount/characterCount）、字符串比较、AND/OR/NOT**——「第 N 章后激活」有原生支持（对状态化实体卡实现有直接参考价值）。
- 三注入位：Memory 顶部 / Author's Note 近底部 / Lorebook 可引用 Memory/AN/最近约 2500 字符正文。

**NovelCrafter Codex**

- 四态：Always / **Include when detected（默认；在所选正文、scene beats 或 chat message 中检测到名字/别名即注入）** / Don't include when detected（仍可经手动场景上下文或 relation 带入）/ Never。
- 别名：自动复数仅英语；大小写默认不敏感；Exclusions 逗号分隔。Track by name/alias 关闭即禁用检测。
- Relations 单向链接、原条目进上下文时联动带入，**无深度/数量参数**（级联风险靠人工节制）。
- **Progressions：锚定到场景的条目增补（默认 addition，可指定 replacement），仅当工作于 ≥锚点场景时对 AI 可见**，回看旧场景不可见。token 预算与装配顺序官方未公开。

**Sudowrite**

- Saliency Engine：生成前自动对「任务+全部 Story Bible」做相关性初筛，不另计费；插件变量 `{{characters}}`/`{{worldbuilding}}` 走筛选、`_raw` 后缀全量透传；卡片/trait 级可手动强制隐藏/显示。内部算法未公开。
- **Chapter Continuity 回看包：前文 ≤25 文档累计 20000 词 + 当前文档从光标向前 ≤20000 词；溢出裁剪次序（先切→后切）：Worldbuilding → Characters → 前章正文 → Linked Outline Summary → Genre → Key Details → Tone → Style → 当前文档前文 → Highlighted Text**，允许部分片段注入。
- Story Bible 依赖链：Braindump→Synopsis→Characters/Worldbuilding→Outline→Scenes→Draft（Synopsis 空则回退 Braindump）；Scene/Prose 生成只看**显式提及**的实体条目。

来源：docs.sillytavern.app/usage/core-concepts/worldinfo、rentry.co/world-info-encyclopedia、docs.novelai.net/en/text/lorebook·storysettings·advancedsettings、novelcrafter.com/help/docs/codex/*（tracking/aliases/relations/progressions-additions）、docs.sudowrite.com（saliency-engine/chapter-continuity/story-bible）。

### 5.2 GitHub 开源长篇生成项目

| 项目 | star | 核心机制 | 最值得抄 |
|---|---|---|---|
| AI_NovelGenerator | 6.2k（活跃） | 新版雪花法「卷→篇章→场景→章节计划→正文」+React Flow 节点画布+实体卡（欲望/恐惧、物品转移、伏笔状态）；**定稿回写三件套：global_summary+character_state+向量库**，AI 产物人工确认才落库 | 定稿回写三件套 |
| goink | 453（极活跃，1100+ commits） | **同为 Wails 桌面应用**。ReAct Agent+31 结构化工具；AI 改稿必须出 Diff 人工批准+内置 Git 历史；角色关系有向图（含演变史）、**伏笔追踪（目标回收章节+超时预警）**、读者认知（已知/期待/误解）；bge-small-zh int8+sqlite-vec+MMR；三层防遗漏=系统提示词+**写后动态注入检查项**+审稿子代理 | 伏笔超时预警；写后动态注入检查项 |
| 天命 tianming | 447 | 每章「读状态→写作→六道门禁→落地→状态回写」：**AI 必须输出 `---CHANGES---` JSON 变更声明**，过门禁（引用/一致性/未知实体/外貌地点矛盾/蓝图出场）后写 15 维事实快照，下章读字段不靠模型记忆，**宣称 3000 章不崩** | 变更声明协议+结构化拒收门禁 |
| chinese-novelist-skill | 3.4k（活跃） | Claude Code skill：偏好记忆 user-preferences.json+断点续写；每章「写前分析→3000-5000 字→去 AI 味润色→字数校验」+**章尾 13 种钩子**+全稿校验不合格自动重写 ≤3 轮 | 偏好记忆+断点续写 |
| RecurrentGPT | 999（停更） | 短期记忆+长期记忆（摘要检索）+**「下一段计划」作为显式一等状态** | 计划即状态 |
| Re3 / DOC / Dramatron / GPT-Author | 学术 | 多候选+判别器重排；detail 控制先松后紧；分层中间稿人改后下钻；大纲层择优 | 多候选择优在大纲层做性价比最高 |
| Long-Novel-GPT | 1.2k | 大纲-正文双窗口同步+多线程扩写 | — |

**横向**：①「递归大纲→逐章展开→审稿循环」已成开源默认架构；②2026 工程线收敛到**外部事实库+结构化变更声明+门禁校验**——记忆带事务语义（不止检索，还校验拒收）；③已证伪：纯滚动上下文硬塞、token 级控制器（FUDGE）、无人工介入的全自动整本生成（口碑普遍差，成功案例都保留人工确认点）。

来源：github.com/YILING0013/AI_NovelGenerator、github.com/sigpanic/goink、github.com/zy-zmc/tianming-novel-ai-writer、github.com/PenglongHuang/chinese-novelist-skill、github.com/aiwaves-cn/RecurrentGPT、github.com/yangkevin2/emnlp22-re3-story-generation、github.com/yangkevin2/doc-story-generation、github.com/google-deepmind/dramatron、github.com/mshumer/gpt-author、github.com/MaoXiaoYuZ/Long-Novel-GPT。

### 5.3 社区实战配方与 NSFW 本地模型谱系

**实战配方（帖子原文参数）**：
- **手动摘要 > 自动总结是压倒性共识**（600+ 楼长跑未翻车的 STMemoryBooks、10M token TTRPG 用户均手动划场景边界）。自动摘要配方（Summaryception 高票）：只提取「相对前文新增/变化」（情节推进/地点/伤势物品/关系信任/已揭动机/承诺/秘密及知情者/未解目标/时间流逝）、禁止复述已知、**禁代词必须全名**、过去时第三人称、每条 3-5 句。
- Lorebook 预算：一般写作 40-50%；重度用户 Scan Depth 3 / Recursion 2 / Context 75%。
- **防重复**：DRY multiplier 0.8 / base 1.75 / allowed length 2 / range 4096（顽固循环提到 1.0-1.5）+ XTC 叠加是主流（KoboldCpp 后端全支持）；prompt 层「四步自检」：先写 3-5 句→自列与近文重复元素→列「还没做过的事」→重写进 `<finalProse>` 标签（GLM4.6 repetition buster，要点=给「要做什么」不是「别做什么」）。
- 采样：极简派 temp 1.0 + Adaptive-p（decay 0.9）+ top n-sigma 1.0；经典 Midnight Miqu 配方 temp 1 / min_p 0.2 / rep_pen 1.05（range 2800）/ smoothing 0.35。
- 风格保持：prompt ≥150 token（可到 2000）内嵌写作样本；「每场景 4-7 轮迭代加深」；哲学层=弃通用大 preset，每个世界定制。
- **版本正典管理反而原始**：多数人 Word+手动——自动正典管理是真空位。

**NSFW 本地写作模型谱系（2026-10 口碑）**：

| 模型系 | 规模/基座 | 口碑 | 量化 |
|---|---|---|---|
| TheDrummer 系（Behemoth 123B/Anubis 105B/Skyfall 31B/Cydonia 24B） | Mistral/Llama 系 | NSFW 特化 go-to；非英语会变笨 | Q4_K_M |
| Magnum v4（anthracite） | 123B=Mistral Large | 文笔 magnificent 但**量化敏感** | ≥Q4，Q2 崩 |
| Midnight Miqu 70B/103B | Llama mix | 「迄今最佳小说写作」，**IQ2_S 仍出佳品**（低量化写作可行证据） | IQ2_S 起可读 |
| Gemma4 31B（2026 当红，非 NSFW 特化） | Gemma4 | 「创意写作第一，小量化大幅领先，frontier 级上下文连贯」；NSFW 需微调/越狱 | 小量化也稳 |
| GLM-5.1 / Kimi K2 蒸馏路线 | 大杯 API 蒸馏 | 智力/长程伏笔最强，文风需配方 | 3bit 可用 |

量化共识：**文风对低量化的容忍度远高于知识（知识低量化非线性崩塌）**；稠密 70B+ Q2 边缘、Q4 甜点；Gemma4 小量化反而惊艳。部署栈事实标准=**KoboldCpp 后端+SillyTavern 前端**（DRY/XTC 全支持）；ollama 在写作社区几乎无人提及。

来源：reddit r/WritingWithAI（1sx1gtw 工作流、1wv9lrx 三部出版复盘、1qqzs8x NSFW 工具评测）、r/SillyTavernAI（1wnydne Summaryception、1q920pl lorebook 设置、1eg2pq5 DRY、1fjwlmr XTC、1opl50q GLM4.6 repetition buster、1u3h0s2 stop using presets）、r/LocalLLaMA（1pwh0q9 2025 年度、1sknx6n 2026 年度、1ci60lf Midnight Miqu、1vef79c 量化伤知识）；huggingface.co（TheDrummer/anthracite-org/sophosympatheia/Sao10K 模型页）。

### 5.4 四垂类 AI 化竞品（一轮未覆盖）

**造价×AI**：广联达 DATA+AI（AI 智能提量=识别 CAD/PDF 构件出量、AI 组价专家=清单匹配定额+历史数据推荐、AI 询比价/清标、**无感建库=项目数据自动沉淀企业定额库**）；鲁班智能土建算量；品茗建筑大模型+HiBIM AI 翻模；国外 Togal.AI（宣称 98% 准确、实测省时 70%）、Kreo。口碑：**「提效大于替代」，复杂/不规范图纸漏项多，必须人工复核**；对广联达怨气集中在商业模式（买断→年费、加收项目）。**真空位=本地桌面+可核查 AI（每个量可溯源、规则透明）；「AI 复核/查漏」比「AI 替代算量」更可信**。

**进度×AI**：MS Project 桌面无 AI；Copilot in Planner（$30/席/月）只做自然语言生成计划；Procore 20 个预置 agent；Alice Technologies optioneering 方案比选。中国双代号生态：**斑马进度（广联达系，免费，MS Project 导入自动生成双代号+纠逻辑错+关键线路预警）**、**孔明 kprogress（表格/横道/双代号三视图同步+AI 排程+本地部署——gaea 最直接对标）**。启示：AI 生成计划是红海标配，**差异点应是「AI 逻辑纠错+双代号自动生成+关键线路推演」这类规范级计算**。

**办公文档×AI**：生成式（Gamma/Kimi Slides/百度文库）痛点恰是「生成后改不动、导出丢格式」；真编辑侧 Microsoft Copilot 2026-04 GA（对象级操作保格式）+**WPS 灵犀（中文市场真编辑事实垄断者）**；把「保格式真编辑」当卖点的供给稀疏（LLeMental/Deckary 等，偏英文初创）。技术流派分明：OOXML 直接操作 vs 重渲染。**gaea「真编辑」定位竞争密度低、技术壁垒可防守，是四垂类中定位最干净的**。

**图像编辑×AI**：Canva Magic Edit/Grab/Expand、PS Firefly 5（精细标杆）；Qwen-Image-Edit 迭代 2509→2511→2.1（7B 单 checkpoint、16 参考图，**⚠️ Qwen Research License 禁商用**）；ComfyUI Desktop 免费但 Cloud 涨价赶客+社区不满密集。**⚠️ 合规红线：gaea 画室若升级 Qwen-Image-Edit 2.1 即踩禁商用；商用须锁定 Apache 2.0 的 2509/2511 或其他许可模型**。

来源：xinhuanet.com/info/20260424（广联达 DATA+AI）、glodon.com、lubansoft.com、togal.ai（peer-reviewed case study）、kprogress.com、zpert.com（斑马）、microsoft.com/en-us/copilot/blog/2026/04/22、wps.cn（灵犀 AI 3.0）、llemental.com、deckary.com、canva.com/canva-ai、huggingface.co/Qwen/Qwen-Image-Edit-2509、comfy.org（qwen-image-2.1/pricing/download）。

### 5.5 上下文管理与压缩机制

**机制对比**：

| | 触发 | 摘要生成 | 压缩后保留 | 恢复 |
|---|---|---|---|---|
| OpenClaw | 预算达 min(90%窗口， 窗口−预留)；overflow 自动压缩重试 | 当前模型（可覆盖），输入 ≤4 万 token 分层采样（最近消息占半逐字）；safeguard 模式摘要质检不合格重试 | keepRecentTokens 默认 2 万逐字；工具调用与结果强制配对；**磁盘全量历史永不删** | 全量 SQLite transcript |
| Claude Code | ~83.5%（可调）；先 **microcompact** 清旧工具输出再整段摘要 | `<analysis>` 时序分析→摘要；须保留事实/文件路径/函数名/报错原文/未决线索 | 重注 CLAUDE.md+memory；lean 变体重读最近 5 文件；**保留已否决方案** | 明文 JSONL+`--resume` |
| Anthropic API | 阈值参数或按需；context editing 默认 10 万 input 触发 | 服务端，可自定义 prompt | 摘要块置顶；keep-recent-turns；保留最近 3 个 tool_use | 客户端自持全史 |
| Letta | 窗口装不下 | 专用 summarizer；**self_compact 摘要请求复用主请求前缀保 KV 缓存** | 滑窗只摘要最旧 30%，近 70% 逐字 | 全量留库可检索 |
| Cherry Studio | 接近窗口上限；**工具输出 >5 万字符卸载分段读** | 跟随当前模型 | 最近 N 条；原文可回看 | 话题原史保留 |

**研究结论（数字）**：Lost in the Middle——性能 U 型，中部显著退化；NoLiMa——去掉字面重叠后 32K 时 11/13 模型跌破短文基线 50%；RULER——宣称 32K 的约一半真正合格；**Chroma Context Rot（18 模型 19.4 万次调用）——纯加长即退化；needle 相似度越低退化越快；1 个 distractor 即拉低；打乱语序的 haystack 全线提升（结构连贯性反而伤性能）；11.3 万 token 全文注入全线差于 300 token 聚焦版**。

**反潮流证据**：Sayou 纯 Markdown+全文检索 **67% vs Mem0 18.5% vs Zep 14.2%**；Manus 原则=压缩必须可恢复（丢正文、留 URL），文件系统是终极上下文；相当多用户手动关 autocompact。

**对照建议**：对话线=双阈值（预警 ~75%/强制 ~88%）+两级化（先清工具输出再整段摘要）+标识符白名单（路径/数字/硬约束逐字保留）+保留已否决方案+摘要后重注最近文件+质检重试+KV 缓存复用。小说线=**「正文永不压缩」原则**（原文全量落盘，stage-recap 降级为指针，回看按需检索回注）；正文回看分层（章→事件卡字段化→10 章合账+人物×章节倒排索引）；关键设定放头尾防 lost-in-middle；伏笔清单在 context 尾部做 recitation；空闲期用便宜模型离线做合账（sleep-time）。

来源：docs.openclaw.ai/concepts/compaction、claudefa.st（CC 阈值分析）、github.com/Piebald-AI/claude-code-system-prompts、platform.claude.com（compaction/context-editing）、docs.letta.com、letta.com/blog/sleep-time-compute、docs.cherryai.com.cn、arxiv 2307.03172/2502.05167/2404.06654、trychroma.com/research/context-rot、manus.im/blog（Context Engineering）、cognition.com/blog/dont-build-multi-agents。

## 六、拍板提案 v2（二轮深挖后重排，替代「二」）

分六组；每刀挂 spec 出处（见五）。A/B 为主线刀包，C/D 小刀，E 合规风险，F 待核实。

### A. 小说线刀包（一条注入链上的七刀，可分批实施）
- **A1 实体卡触发注入+预算**：四态模型（Always/检测注入/检测不注入/Never）+别名表（外号/敬称）+排除词；预算 40-50% 默认、Scan Depth 3、递归 2；**中文必须关全词匹配**。Spec=5.1 ST/NovelCrafter 表。
- **A2 状态化实体卡**：锚点=章节的 addition/replacement，注入按当前章取最新 ≤当前章状态；NovelAI Advanced Conditions 的「第 N 章后激活」原生语义可参照。Spec=5.1。
- **A3 前情回看包**：显式组装器+固定裁剪次序（设定→角色→前章正文→大纲摘要→文体→风格的优先牺牲序）+双上限；结合「正文永不压缩」原则与头尾放置。Spec=5.1 Sudowrite+5.5。
- **A4 定稿回写**：章节定稿时回写「阶段摘要+角色状态+伏笔状态」三件套（gaea 台账/角色面板已有部分，补齐事务性）。Spec=5.2 AI_NovelGenerator。
- **A5 变更声明+章级门禁**：生成章后要求 AI 输出结构化变更声明（人物状态/伏笔/时间线），过校验门禁才落库，不合规拒收重写；与既有评测线互补（门禁=章级闸，评测=阶段病历）。Spec=5.2 天命/goink。
- **A6 伏笔追踪**：伏笔卡带目标回收章节+超时预警；可并入 A4/A5 的状态字段。Spec=5.2 goink。
- **A7 防重复三件套**：DRY（0.8/1.75/2/range4096）+XTC 内置写作档默认（strata 引擎 llama.cpp 上游均已支持，需核实暴露面）+「四步自检」prompt 预设。Spec=5.3。

### B. 记忆/上下文刀包
- **B1 混合召回**：SQLite FTS5 全文路+实体路+时序排序，与向量路融合（一轮提案 4 不变，证据升级：Sayou 全文检索 67% vs 向量方案 14-18%）。
- **B2 记忆时间戳/失效标记**：冲突打 invalid_at 不删（一轮提案 4 后半，不变）。
- **B3 compact 双阈值+两级化**：75% 预警/88% 强制；先清工具输出（>5 万字符分段读）再整段摘要；标识符白名单+保留已否决方案+摘要后重注最近文件+质检重试。Spec=5.5 对比表。

### C. 本地推理刀包
- **C1 模型推荐位三档陈列**：NSFW 特化（TheDrummer 系）/文笔（Magnum/Miqu/Gemma4）/智力（GLM/Kimi 蒸馏），带量化建议与 multilingual 标注；Midnight Miqu IQ2_S 仍佳=低量化写作档可行背书。Spec=5.3。
- **C2 agent 量化红线**：工具调用需 Q4+，低档警告+MoE 甜点预设（一轮提案 6 不变）。

### D. 垂类刀包（与既有刀路池归拍）
- **D1 造价「AI 查漏/复核」**：定位=复核增强而非替代算量，每个结论可溯源（对标 Togal 溯源叙事+广联达 AI 组价）；注意与 docs/gaea-cost-db-master-plan-2026-10.md 四期计划排期关系，须用户拍板插入点。
- **D2 进度「AI 逻辑纠错+关键线路推演」**：对标斑马（免费+双代号自动生成+纠错）与孔明（三视图+本地）；gaea 双代号基本功已收官，此为 AI 层新刀。
- **D3 办公「保格式真编辑」立场强化**：竞争密度低+WPS 之外无中文重度玩家，宣传与产品双双押注「改既有文件不丢格式」（对位生成式竞品「改不动」痛点）。无需新开发，属定位决策。

### E. 合规风险
- **E1 Qwen-Image-Edit 许可证（已核实 2026-10-14）**：画室当前权重=`qwen_image_edit_2511_fp8mixed`（v4.392.0，2026-09-22 接入，Comfy-Org 重打包），**2511 本体=Apache 2.0（HF 页面 license 标签+许可证章节双证），当前商用合规**。红线在未来：Qwen-Image 2.1（7B 单 checkpoint）起改 **Qwen Research License 禁商用**——升级画室基座前必查许可，勿顺手升 2.1。Spec=5.4。

### F. 桌面通用面（待核实）
- **F1 全局热键呼出/划词助手**：同一轮提案 7，Wails 全局快捷键可行性未核实。

**优先级建议**：A 组（小说线主线，A1-A3 先行，A4-A7 随后）> B3（compact 升级）> B1/B2 > C 组小刀 > D 组待拍板插入点 > F。E1 立即核实，不等拍板。
