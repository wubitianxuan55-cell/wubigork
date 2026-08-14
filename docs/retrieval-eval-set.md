# 检索质量受控测评查询集（Retrieval Eval Set）

> 用途：阶段 5 T5-6 检索质量回归门槛 **Recall@10 ≥ 0.8** 的受控测评查询集。
> 引擎：`GaeaRetrievalEvalRun` 解析本文件底部的 ```json 代码块，逐条调用跨库统一语义检索
> （GaeaSemanticSearch：cost/knowledge/office/file 四库，本地 Herdsman bge-m3），
> 取前 10 条命中计算单条召回率，汇总 recall@10，与门槛 0.8 比较得出 passed。
> 来源：造价/工程办公域真实业务查询，从 docs/ 现有文档与成本库 mock 数据取样。

## 匹配规则

- 每条查询标注 `expected`：`[{kind, name}]`，kind 为 cost/knowledge/office/file，
  name 为目标条目名（知识/成本/办公记忆的 name 字段）或文件相对路径。
- 命中判定：expected 与 topHits 均序列化为 `kind:name`，同 kind 且 name 精确相等
  或**互为子串**即记命中（如预期 `P.O 42.5 水泥` 命中 `P.O 42.5 水泥（散装）`）。
- 单条 recall = 命中预期数 / 预期总数；recall@10 = 各条 recall 的平均值。
- 查询集随各库实际数据增长而维护：新增库条目时按需补 expected，保证每条查询
  至少 1 个预期命中，且四类 kind 尽量都有覆盖。

## 查询集

| # | 查询 | 预期命中（kind:name） |
|---|------|----------------------|
| 1 | 打桩设备 台班价 | cost:HP300 高频液压振动锤；cost:桩机台班费 |
| 2 | P.O 42.5 水泥 价格 | cost:P.O 42.5 水泥；file:docs/材料价格信息价.md |
| 3 | 振动锤选型要点 | knowledge:振动锤选型要点；office:桩基施工-振动锤选型 |
| 4 | 投标文件格式要求 | knowledge:投标文件格式要求；file:docs/投标文件模板.md |
| 5 | 挖掘机 220 台班 租赁价 | cost:挖掘机 220 台班 |
| 6 | 三轴搅拌桩 水泥掺量 20% | knowledge:三轴水泥搅拌桩工艺要点；cost:桩机台班费 |
| 7 | 地下室抗浮水位 设计要求 | office:项目-地下室抗浮水位；knowledge:抗浮设计要点 |
| 8 | 桩基检测 低应变 规范 | knowledge:桩基检测规范要点 |
| 9 | 螺纹钢 HRB400 价格 | cost:螺纹钢 HRB400 |
| 10 | C30 泵送混凝土 单价 | cost:泵送商品混凝土 C30 |
| 11 | 清单计价 综合单价 组成 | knowledge:综合单价组成 |
| 12 | 土方开挖 放坡 安全要求 | knowledge:土方开挖放坡要求；office:项目-土方开挖方案 |

```json
[
  {"query": "打桩设备 台班价", "expected": [{"kind": "cost", "name": "HP300 高频液压振动锤"}, {"kind": "cost", "name": "桩机台班费"}]},
  {"query": "P.O 42.5 水泥 价格", "expected": [{"kind": "cost", "name": "P.O 42.5 水泥"}, {"kind": "file", "name": "docs/材料价格信息价.md"}]},
  {"query": "振动锤选型要点", "expected": [{"kind": "knowledge", "name": "振动锤选型要点"}, {"kind": "office", "name": "桩基施工-振动锤选型"}]},
  {"query": "投标文件格式要求", "expected": [{"kind": "knowledge", "name": "投标文件格式要求"}, {"kind": "file", "name": "docs/投标文件模板.md"}]},
  {"query": "挖掘机 220 台班 租赁价", "expected": [{"kind": "cost", "name": "挖掘机 220 台班"}]},
  {"query": "三轴搅拌桩 水泥掺量 20%", "expected": [{"kind": "knowledge", "name": "三轴水泥搅拌桩工艺要点"}, {"kind": "cost", "name": "桩机台班费"}]},
  {"query": "地下室抗浮水位 设计要求", "expected": [{"kind": "office", "name": "项目-地下室抗浮水位"}, {"kind": "knowledge", "name": "抗浮设计要点"}]},
  {"query": "桩基检测 低应变 规范", "expected": [{"kind": "knowledge", "name": "桩基检测规范要点"}]},
  {"query": "螺纹钢 HRB400 价格", "expected": [{"kind": "cost", "name": "螺纹钢 HRB400"}]},
  {"query": "C30 泵送混凝土 单价", "expected": [{"kind": "cost", "name": "泵送商品混凝土 C30"}]},
  {"query": "清单计价 综合单价 组成", "expected": [{"kind": "knowledge", "name": "综合单价组成"}]},
  {"query": "土方开挖 放坡 安全要求", "expected": [{"kind": "knowledge", "name": "土方开挖放坡要求"}, {"kind": "office", "name": "项目-土方开挖方案"}]}
]
```
