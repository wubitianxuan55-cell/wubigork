# gaea 小说板块 · oh-story-claudecode 蒸馏规格（2026-09-13）

> 状态：🔄 规格定稿（首刀待落）· 类型：外部项目蒸馏（机制清单 + gaea 落点 + 刀路）
> 上游：https://github.com/zenstory-ai/oh-story-claudecode —— 面向 Claude Code / Codex / OpenCode 等宿主的
> **网文写作工具集插件**（11 个 skill + 7 个子代理 + 自动化 hooks + 流水线脚本）
> 本地克隆：`clones/oh-story-claudecode`（只读参考，已 gitignore）
> 关联：`docs/distill/02-book-import.md`（拆书导入 t2，v4.279~281 已在产）、`.gaea/skills/novel-deslop/`（去 AI 味）、
> `internal/novelstyle`（文风指纹）、`docs/gaea-sin-board-design-2026-09.md`（原罪板块）

## 0. 法务红线（先于一切设计）

上游是 **MIT 许可**（`clones/oh-story-claudecode/LICENSE:1`）——与 gaea 私有（All Rights Reserved）并不冲突，
但必须分两种处置，且沿用屋里「蒸馏=取道不取器」的惯例：

1. **机制**（思路 / 阈值 / 字段语义 / 流程 / 判定条件）→ **重推导实现**，码内注明「重推导自 oh-story（MIT）」；
2. **知识资产**（写作方法论 / 禁词表 / 评审 rubric 等**文本型知识**）→ **可收录**，但必须随附
   `LICENSE` 与出处标注（gaea 既有先例：`.gaea/skills/{pdf,docx,xlsx}/LICENSE.txt`）；
3. **脚本代码不整段移植**——需要时按 gaea 的语言与结构重写（Go 内核纯函数 / TS 组件 / skill 脚本）。

## 1. 上游盘点（技能 × gaea 现状 × 结论）

| 上游 skill | 一句话职责 | gaea 小说板块现状 | 结论 |
|---|---|---|---|
| `story-long-write` | 长篇写作流水线（大纲→正文→回炉） | 创作间流式生成 + 大纲树 + 角色/世界观 | **借鉴**（钩子式校验闸、写作前置契约） |
| `story-short-write` | 短篇写作（情绪拉扯 / 反转） | 无短篇形态 | 暂不采纳（gaea 走长篇/拆书路线，短篇属新形态） |
| `story-deslop` | 去 AI 味（禁词/句式/标点归一 + 门槛） | `.gaea/skills/novel-deslop`（词表外置 v4.225）+ `novelstyle.DeSlopRewrite` | **高价值借鉴**（规则表更细、有门槛分级） |
| `story-review` | 多视角对抗式审查 + 平台 rubric | `novelstyle` 打分 + 章节体检 | **借鉴**（rubric 数据化 + 分平台档位） |
| `story-long-analyze` | 爆款长篇拆文（黄金三章 / 人设 / 爽点 / 节奏） | 拆书导入（结构/大纲反推）+ 文风指纹 | **部分借鉴**（风格档案协议、拆解维度） |
| `story-short-analyze` | 短篇拆解（故事核/反转/共鸣） | 无 | 暂不采纳 |
| `story-long-scan` / `story-short-scan` | 平台榜单抓取（起点/番茄/晋江…，走 CDP） | 书源引擎（并行线在制品，`internal/booksource`） | 归并行线（不重复立项） |
| `story-import` | 逆向导入已有小说（结构映射 + 字数路由 + 追踪事务） | 拆书导入 t2（分章/编码/反推/落库） | **借鉴**（结构映射表、字数路由、事务语义） |
| `story-setup` | 把技能/子代理/钩子部署到各宿主 | gaea 自带技能系统 + 内置子代理 | 不采纳（宿主适配是 Claude Code 生态问题） |
| `story-cover` | 封面生成（GPT-Image / ImageGen） | 绘梦（图像域，含角色参考槽） | 不采纳（gaea 已有更强出图链） |
| `story`（Dashboard） | 本地写作看板 + 作者习惯记忆 | 小说板块 UI + 记忆中枢 | 不采纳（形态不同；作者习惯可归记忆域） |
| `browser-cdp` | CDP 工装 | `scripts/cdp-walk.mjs` 已有 | 不采纳 |

**子代理与钩子**（`skills/story-setup/references/templates/agents/*`、`.../hooks/*`）：
上游把「写作职责」拆成 7 个子代理（架构师/写手/一致性检查/章节抽取/探索/研究/角色设计）并用**钩子**在
「写前/写后/会话起止」做硬拦截。gaea 当前是「单模型 + 任务子代理 + 技能」形态——**机制可借，形态不照搬**（见 §3）。

## 2. 蒸馏结论：gaea 小说板块的四条真增量

1. **写作前/后的确定性校验闸**（上游 hooks：写前守大纲、写后检查散文质量与退化）——gaea 目前只有事后的
   去 AI 味与体检，**没有生成前的结构契约闸**，也没有生成后的自动拦截。
2. **评审 rubric 数据化 + 分平台档位**（上游 `story-review/references/rubrics/{fanqie,qidian,zhihu}.md`）
   ——gaea 的打分是「通用阈值」，没有平台/题材档位，也没有「多视角对抗」的评审形态。
3. **去 AI 味的规则分层**（禁词 / 句式模式 / 标点归一 / 门槛分级，上游 `story-deslop` 四份 references + 三个脚本）
   ——gaea 已把词表外置，但规则仍是单一列表，缺少「模式级」判定与门槛分级。
4. **逆向导入的结构映射契约**（上游 `story-import/references/structure-mapping-*.md` + `length-routing.md`）
   ——gaea 的拆书导入已能分章/反推大纲，但**没有「按篇幅路由 + 结构映射」的显式契约**（长/短篇走不同骨架）。

## 3. 刀路（建议顺序）

| 刀 | 内容 | 落点 | 规模 |
|---|---|---|---|
| **T1** | **质量 rubric 数据化**：分平台/分题材评审档位 + 章节质量评分纯函数（吃现有 novelstyle 指标） | `.gaea/skills/novel-review/rubrics/*.json`（数据资产）+ Go 纯函数 | 小-中 |
| ~~**T2**~~ ✅ | ~~**去 AI 味规则分层**：把「模式级」判定（句式模板/连接词堆叠/四字格密度/标点异常）与门槛分级补进现有词表资产~~ **已落**：`.gaea/skills/novel-deslop/references/ai-patterns.md`（删除优先判断 + 门禁 A-G 要点）+ `gates.json`（7 门禁结构化 + 保护清单 + 执行纪律）+ MIT 许可文件；内核消费 `gates.json` 留后续刀 | `.gaea/skills/novel-deslop/`（数据资产） | 已完成 |
| **T3** | **生成前后校验闸**：写前大纲契约检查（关键字段齐备/章节目标不为空）+ 写后质量闸（不达标给回炉建议，不静默通过） | 内核纯函数 + 创作间提示 | 中 |
| **T4** | **导入结构映射与篇幅路由**：按总字数/章数选骨架（短中长三档），映射到 gaea 大纲字段 | `internal/bookimport`（拆书导入续刀） | 中 |
| T5 | 子代理角色卡资产化（7 角色提示词按 gaea 形态重写为技能资产，接 `run_skill`/任务子代理） | `.gaea/skills/novel-agents/` | 中 |
| T6 | 风格档案协议对齐（上游 style-profile-protocol ↔ gaea 文风指纹参考档） | `internal/novelstyle` | 按需 |

> 不采纳：封面生成（绘梦已在产）、榜单抓取（书源引擎并行线）、宿主部署适配（Claude Code 生态问题）、
> 短篇写作/拆解（gaea 小说板块定位为长篇 + 拆书；若日后要做短篇形态另立调研）。

## 4. 首刀选型

**T1 质量 rubric 数据化** 作为首刀：① 纯数据资产 + 纯函数，零风险可测；② 直接喂到既有「章节体检」
（AI 味分数已有，缺「按平台/题材看及格线」的口径）；③ 不依赖模型在线，壳内可走查。

**T2 去 AI 味规则分层** 紧随其后（同一资产层，可共用一次发版）。

## 5. 机制清单与证据（三路盘点）

> 逐条机制（含 `file:line` 证据、阈值、gaea 落点建议）由三路并行盘点产出，归档于：
> - `.gaea/reports/ohstory-write-core-2026-09-13.md`（写作执行内核）
> - `.gaea/reports/ohstory-analysis-2026-09-13.md`（分析与评审）
> - `.gaea/reports/ohstory-agents-hooks-2026-09-13.md`（子代理 / 钩子 / 状态追踪）
>
> 三份报告随首刀一并入库；本档只保留结论与刀路（细节以报告为准）。
