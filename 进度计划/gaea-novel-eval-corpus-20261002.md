# 评测回归夹具：3×30 章病历（非版本刀，2026-10-02）

> 用户指令「继续」。v4.435 留池头名落地（纯测试基建，零生产代码改动、零绑定
> 面变更、不发版不抬版本）。

## 落地

`internal/app/eval_corpus_test.go`：三本书 × 30 章的确定性语料生成器
（`buildEvalCorpusBook`）+ 三条回归测试，给评测链（快照/基线/对比）做规模级
回归保护。断言全部走**相对指标**（病历书 vs 基准书的指标差）——对打分器内部
参数变化鲁棒，对链条退化敏感（打分归零/聚合错位/章漏扫即红）：

- **healthy 基准书**：自然行文（对话+情态标记随章号轮换、无 AI 腔模式），
  伏笔 10 埋 8 收（回收率 0.8）——「好」端基线。
- **flavor 病历书**：伏笔表同 healthy（隔离变量），正文换重 AI 腔（重复感叹+
  否定翻转+解释腔）——隔离「AI 味」单一维度；实测均值守恒失效即打分链退化。
- **hollow 病历书**：正文=基准+每章一条 >400 rune 超长段（gate
  `paragraph_too_long` S3 确定命中），伏笔全部埋设（回收率 0）——隔离
  「回收率/质量闸」维度。

三条测试：
1. `TestEvalCorpusSnapshotDeterministic`——三本书各自两次快照逐字节相同
   （§4 确定性验收线在 30 章规模复验）+ 章数恒 30（无漏扫）+ 字数非零。
2. `TestEvalCorpusProfilesSeparate`——flavor.Taste.Mean > healthy；
   hollow 回收率 0 vs healthy 0.8；hollow S3≥30 vs healthy=0。
3. `TestEvalCorpusBaselineCompareScale`——基线后无变化对比零 worse 项；
   单章病变（第 15 章换重 AI 腔）→ AI 味均值 worse 方向。

## 探针纪要（写作用语料时踩的坑）

- novelgate 的 S1 只来自空章——预告尾（trailer_ending）标记是 rubric.json
  给 LLM 评审链用的，**gate 链不实现**；质量闸的确定性触发器是超长段
  （paragraphMaxRunes=400，S3）与电报体（S2）。
- 病理内容必须真正接进对应画像——`buildEvalCorpusBook` 传了素净章节函数而
  断言在等病理指标时，失败信息（S3=0）与病灶（没写病理）之间没有直接线索，
  先用单本探针核对 gate 直调输出再对照快照。

## 留池

- 张力曲线聚合（评测快照加全书情感强度聚合——语料已就位，可做）。
- LLM 盲评 bootstrap / 真 A/B（需模型与人工评）。
- 语料扩维：钩子缺失/对话占比失衡画像（现三本覆盖 基线/AI 味/断链）。
