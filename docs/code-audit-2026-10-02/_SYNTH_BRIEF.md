# gaea 全仓代码审计 · 分册主编作业书（2026-10-02）

你是某分册的**主编**。并行单元审计员已把证据写进 `docs/code-audit-2026-10-02/units/*.json`。派单提示会给你：分册标题、输出文件名、负责的单元 ID 列表。

## 步骤
1. 用 glob/read 读你负责的每个单元 JSON（文件较大，用 read 的 offset/limit 读全，别只看开头）。
2. 去重合并：跨单元同根因（如「同一逻辑在 A 与 B 各写一份」）合成一条并列出所有位置；与历史审计重合的条目标「延续」。
3. 排序：P0 到 P3；同 severity 按「影响面 × 改动收益」。
4. 写分册到 `docs/code-audit-2026-10-02/<派单给的文件名>`（UTF-8 无 BOM，中文 Markdown）。

## 分册结构（严格）
```
# <分册标题>
> 元信息：覆盖单元 / 证据来源 units/*.json / 审计日期 2026-10-02
## 一、板块总评
评分表：单元 | 板块 | 评分 | 一句话总评（取自单元 JSON 的 score/verdict）
## 二、屎山 TOP 榜
表格列：# | 严重度 | 标题 | 位置(file:line) | 类别 | 证据摘要 | 修法 | 工作量
## 三、分类清单
按 category 分小节；每条给 file:line + 证据 + 影响 + 修法。
## 四、拆刀建议
可执行刀序：刀号 | 目标 | 涉及文件 | 风险 | 验收口径（单测 / HTTP 断言 / 构建 / 手工走查）。与历史审计重合的标「延续」。
## 五、观察项 / 未定论
## 六、单元评分明细与证据索引
单元 ID | units/<ID>.json | score | findings 数
```

## 铁律
- 所有结论必须有 file:line 证据（来自单元 JSON；不许自己编，不许无证据升级严重度）。
- 禁泛泛而谈；禁「建议重构」这类无操作性的句子——每条修法要能被直接执行。
- 只写你这一份分册；不修改 units/*.json 与任何其它文件。
- 中文。
- 回报走 schema：title / file / total_findings / p0 / p1 / verdict_short / cutter_count / top10（rank,severity,title,file,line,effort）。
