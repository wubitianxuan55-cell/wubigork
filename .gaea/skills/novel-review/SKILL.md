---
name: novel-review
description: "网文章节评审（多平台档位）。对已写章节做结构化评审：26 维 PASS/WARN/FAIL + 黄金三问 + S1-S4 问题分级 + 发布门槛（APPROVE/CONCERNS/REJECT），并按目标平台（番茄/起点/知乎盐言）换档。触发方式：/novel-review、/章节评审、「评审这章」「这章能发吗」「按番茄标准看看」。不适用于去 AI 味（用 novel-deslop）或文风指纹（用创作间「文风指纹」）。"
---

# 网文章节评审（gaea 版）

> 蒸馏来源：`oh-story-claudecode`（MIT，https://github.com/zenstory-ai/oh-story-claudecode）的
> `skills/story-review/references/quality-rubric.md` 与 `rubrics/{fanqie,qidian,zhihu}.md`。
> 本技能为**机制重推导 + 知识资产结构化收录**（见同目录 `LICENSE-oh-story.txt`；上游文本未逐字搬运，
> 已改写为 gaea 的「维度 id + 档位 + 证据要求」结构）。

## 何时用

- 用户问「这章能发吗 / 按番茄标准看看 / 评审第 N 章 / 哪里劝退」；
- 章节写完、去味之后，需要**发布前判断**（而不是风格分数）时。

## 与 gaea 既有能力的分工（不要重复造）

| 能力 | 回答的问题 | 数据来源 |
|---|---|---|
| 创作间「文风指纹」/ 章节体检（`internal/novelstyle`） | 「这章像不像 AI 写的 / 离你的文风多远」——**确定性指标** | 句长分布、TTR、口头禅、函数词 z 向量 |
| `novel-deslop` 技能 | 「怎么把 AI 腔改掉」 | 词表 + 替换规则 |
| **本技能** | 「作为作品，这章**能不能发**、哪里劝退、怎么改」 | 19 维 rubric + 平台档位（语义判断，LLM 逐条给证据） |

**铁律**：确定性可测的维度（句长节奏 / 标点节奏 / 格式可读性 / 具体字数表达）以 gaea 体检的实测值为准；
语义维度（卖点 / 冲突 / 动机 / 伏笔）必须**引用原文**作为证据——**没有原文证据的 finding 不输出**，改标「证据不足」。

## 评审协议

1. **读档**：目标章节正文 + 前情摘要（`novelcontext`）+ 大纲该章节点（summary/emotion）+ 伏笔登记表。
2. **逐维打标**：按 `rubrics/generic.json` 的 26 个维度逐条给 `PASS | WARN | FAIL`，FAIL/WARN 必须带：
   `severity`（S1-S4，映射见下）、`dimension`（维度 id）、`location`（章节 + 段落/原句）、
   `evidence`（原文摘录，≤50 字）、`issue`（一句说清）、`fix`（可执行的一步改法）。
3. **黄金三问**（答不出即至少 S2）：
   - 读者为什么翻下一页？
   - 本章改变了什么（情节/关系/信息/情绪至少一项）？
   - 哪个证据支持你的判断？
4. **换平台档位**：按 `rubrics/platforms.json` 覆盖阈值（如番茄：前 3 段无钩子=FAIL、连续 3 章结尾无翻页动力=FAIL）。
5. **出结论**：`APPROVE`（无 S1/S2，S3 可快速处理）/ `CONCERNS`（有 S2 或 S3 量大）/ `REJECT`（有 S1 或卖点·动机·规则崩坏）。

## 分级（S1-S4）

| 级别 | 判据 |
|---|---|
| S1 | 影响主线、角色动机、世界规则或读者信任 |
| S2 | 明显影响留存、节奏、章节效果或人物可信度 |
| S3 | 局部质量、格式、措辞、轻微节奏 |
| S4 | 风格建议或可选增强 |

## 输出格式（固定）

```text
结论: APPROVE | CONCERNS | REJECT（平台档位: generic | fanqie | qidian | zhihu）
S1/S2 问题（逐条）: [severity] 维度 · 位置 · 证据（原文）· 问题 · 改法
S3/S4 问题（逐条）: 同上
黄金三问: 逐问作答（含证据）
体检实测（若已跑）: AI 味分数 / 章长 / 句长分布要点
```

先列 S1/S2，再列 S3/S4；`consistency` 类问题的 fix 只写「事实统一方向」，不写文学创作建议。

## 资产

- `rubrics/generic.json`：26 维通用档位（18 语义/通用 + 8 确定性引擎扩展；id / 维度名 / PASS / WARN / FAIL / 是否可确定性测量）
- `rubrics/platforms.json`：番茄 / 起点 / 知乎盐言三平台档位（覆盖或追加维度）
- `LICENSE-oh-story.txt`：上游 MIT 许可全文 + 出处与改动说明

## 确定性引擎（v4.282.0 起，创作间「平台评审」面板）

`measurable: true` 的维度已落成内核纯函数（包 `internal/novelreview`，零 LLM 零网络）：
创作间 rail「平台评审」→ 选档位（通用 / 番茄 / 起点 / 知乎盐言）→ 逐维出
`PASS | WARN | FAIL | SKIP` + 严重度 + **原文证据（rune 区间 + 段落号）** + 改法，
并给出 `APPROVE / CONCERNS / REJECT` 结论（门槛与 `rubrics/generic.json` 的 verdict 同口径）。

| 引擎维度（15） | 档位差异来源 |
|---|---|
| `length_band` `dialogue_ratio` `format_readability` `punctuation_rhythm` `dash_usage` `format_hygiene` | 各平台阈值（章长带 / 对话占比带 / 长段阈值 / 省略号密度） |
| `opening_freshness` `hook_expectation` `trailer_ending` `emotion_curve` `plot_loop` | 判定词表 + 平台线（番茄情绪节点密、起点爽点/升级密） |
| `perspective_consistency` | 知乎盐言档位开启（第一人称） |
| `protagonist_presence` `lever_mention` | 起点口径（主角出场 / 金手指提及）；主角名取自角色库 `role_type=protagonist` |
| `stated_length_accuracy` | 「这五个字」类表达 vs 引号实体实际字数 |

> `plot_loop` 在引擎里只做「爽点/升级标记密度」的**代理判定**（起点档位），语义层面的
> 「目标→阻碍→行动→反馈」闭环仍由 LLM 评审判定；两者结论不一致时以 LLM 逐条证据为准。

**阈值与词表是数据资产**：`internal/novelreview/rubric.json`（go:embed 默认），
可被工作区覆盖文件 `<cwd>/.gaea/skills/novel-review/rubric.json` **整体替换**
（与 `novel-deslop/words.json` 同款纪律：想增条目把默认表抄进覆盖文件再改）。

**语义维度**（`core_selling_point` / `character_motivation` / `foreshadow_state` 等）引擎不下结论：
由 LLM 评审批次按上文协议逐条给证据；引擎对缺数据的维度显式 `SKIP` 并写明原因（如「角色库未标注主角」），
**不静默给 PASS**。
