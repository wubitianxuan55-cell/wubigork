# 灵犀记忆三层 × gaea 记忆现状对照评估（batch2 B4，2026-11-13）

> 输入：`re-lingxi/out/MEMORY-spec.md`（72KB：14 段原文+61 条记忆工具文案）、
> gaea 实码（`internal/gaea/memory/` 14 文件 + `internal/app/gaea_dream*` +
> fact 库 + MemoryHub 面板）。
> 定位：**评估报告，不落地**。落地须拍板（拍板池②），且办公记忆写入确认红线
> （未经确认不写持久化数据）优先于任何灵犀形态。

## 0. 结论卡

**灵犀三层（core 常驻/wiki 检索/raw 溯源）与 gaea 现状同构度约七成**：gaea 已有
「画像常驻+检索+固化+溯源引用」的等价物，缺的是**三个组织性概念**——统一目录
树（wiki 页面树）、写入门禁的成文规则、召回规则的显式化。建议：**不搬结构，
搬三条规则文案进记忆面板/提炼提示词**；wiki 目录树列为远期观察。

## 1. 逐层对照

| 灵犀层 | 灵犀语义 | gaea 等价物 | 缺口 |
|---|---|---|---|
| **core/核心记忆**（每轮全文注入，≤6000 字，称呼/风格/确认习惯/项目稳定规则） | core_memory 常驻 | ProfileBlock（user 画像 600 rune 预算）+ PinnedFactsBlock（固化正文）+ 晨报预载（work） | gaea 分散在四个块，无「单页常驻规则」概念；**预算口径不同**（灵犀 6000 字 vs gaea 画像 600 rune+固化预算） |
| **wiki/知识层**（目录树索引+关键词搜索，主题知识页，front-matter+sources 溯源） | memory_index+搜索 | 记忆检索（memory.Search/SetSearchIndex）+ 记忆图谱（graphstore）+ MemoryHub 面板 | **无统一目录树形态**；gaea 的记忆是平铺条目+图谱，不是页面树；citations 已做来源归因（[MEM:] 键）≈ sources 字段 |
| **raw/源文件索引**（只记溯源凭证：file_id/path/url） | wiki 页 frontmatter sources | ProjectBrief 的 [MEM:] 来源会话归因 + citations | 基本等价 ✓ |
| **memory_write_rules**（只写用户偏好/协作规则/长期上下文；其余不干涉不记录不询问） | 61 条工具文案里成文 | dream 提炼=建议制（notice→结构化卡→用户接受才落）+ 办公记忆确认红线 | **机制更强**（gaea 是硬确认门，灵犀是提示词软约束）；缺的是「什么值得写」的**成文清单**给提炼提示词 |
| **memory_recall_rules**（召回规则显式化） | 独立段 | activation（pinned=正文/relevant=检索可达）+ decay + injection_eval | 机制在但未成文给模型看（内部逻辑 vs 提示词规则）；**不建议成文**——gaea 的召回是代码行为，写出来反而让模型越权干预 |
| **wiki 页 front-matter/格式** | fragments_format 段 | 记忆条目结构化字段（Name/Description/Body/Type） | 等价 ✓ |

## 2. 可迁移的三条（建议采纳，纯文案）

1. **写入门禁清单进提炼提示词**（dream/fact_add 的指引）：只写「用户偏好、协作
   规则、长期上下文」三类；项目细节、实体详情、长文全文、一次性任务状态**不写**。
   灵犀原文可蒸馏为三行。落点：dream 提炼提示词+fact_add description。
2. **core 常驻预算显式化**：给 ProfileBlock/固化正文一条成文上限说明（现预算在
   代码里），让「画像越写越长挤爆前缀」有据可依的清理路径——MemoryHub 面板
   提示一句即可。
3. **召回规则一句话进记忆块头**：「以下为系统自动召回，模型不得据此声称拥有
   全部记忆，需要更多用检索工具」——防「看见了晨报就以为看过全部记忆」的越权
   推断。落点：memory.Compose 各块的头行。

## 3. 不搬的（附理由）

- **wiki 页面树目录**：gaea 记忆是条目+图谱形态，引入页面树=重构存储+面板；
  收益（组织性）不抵成本（迁移+双形态过渡期）。列远期观察池。
- **记忆工具面 61 条**（wiki_memory_*/write_memory_*/edit_memory_*）：gaea 已有
  fact_add/检索/MemoryHub 全覆盖，工具面不缺。
- **云端存储定位**（「均存储于云端」）：gaea 记忆本地优先是既有立场，不随。

## 4. 若拍板落地的最小刀（预估 0.5 人日）

第 2 节三条文案刀：dream 提炼提示词+fact_add description+Compose 块头各一处，
纯文案+既有测试锚定不变式；无结构/迁移面。拍板池②的答案就在这三刀。
