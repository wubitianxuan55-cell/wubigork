# 全仓审计第 46 批 · NOT_MOCKED 收尾刀（白名单清空，漏 mock 锁全量接管）· 2026-10-03

> 接续 [round-47（批次四十五）](round-47-p1-batch45.md)。NOT_MOCKED 池收官：**全部 25 件补齐**（Go 忠实退化 1 / Go 口径空态 5 / 诚实拒绝 19），白名单清空——绑定面完整性从此 fail-closed。dev mock 行为新增（申报），**不抬版本**。
> 快照：开工时 HEAD = `0ac73967`（批次四十五）；工作树干净。

---

## 一、25 件三态处置（口径逐件对着 Go 定，不拍脑袋）

- **Go 忠实退化**（1）：ChatStreamCancel 恒 `false`——Go 对未知 runID 返回 false 不算错，mock 永无在跑流（ChatStreamPlain 即返回、无事件流），false 是每个可能输入的忠实答案；**前端零消费**（grep 证，仅绑定面清单引用），无须把 ChatStreamPlain 重建成脚本流。
- **Go 口径空态/零值**（5）：DocumentLint 空报告（issues:[]/passed:true，确定性 lint 诚实退化）；TTSVoiceParams 全零值（0/空串=不指定，Go 引擎未接同语义）；WhisperGraphSubgraph 空子图（Go「entity 不存在返回空子图非 nil」）；WhisperProactiveNow 恒 `{shouldSend:false}`（门控未过=Go 合法返回）；WarmComfyUI `{started:false, reason:"mock-no-comfyui"}`（Go 口径 {started,reason}）。
- **诚实拒绝**（19）：全部 AI 生成/评分/反推/聚敛/书封/抠图/组价/PPT 大纲件——`dev mock：XX 需真实 LLM 内核（或绘图引擎）`，同 ImportNovelBookEx 诚实拒先例；按钮触发面下真实后端失败同样以错误呈现，**严格优于原 TypeError**；两个链路前置件（ComposeApply/OutlineReconstructApply/TaskGet）文案注明「需先有前步结果」。

## 二、命名面教训（编译器三连抓）

- AppBindings 键名=**映射后短名**（OfficeB/VoiceB/NovelB/CostB 门面剥 Gaea 前缀；MemoryB 部分保留）——首轮按 wailsjs 门面方法名写 Pick，tsc 三轮 TS2344 当场抓齐（Did you mean '"CostCompose"'…）；终审是编译器再实证。
- **新观察池登记（疑似真机 bug）**：bridge 契约面 `WhisperSubgraph` 声明小写 `nodes/edges`，但 Go `whisper.Subgraph` **无 json 标签**（原始序列化为大写 `Nodes/Edges`）——真机图谱子图面板若直读契约字段将得 undefined；是否中间层归一待真机验证，mock 遵从契约面不编造第三种口径。

## 三、落地与门禁

- 8 个 mock 文件（charlib/cost/imagegen/voice/chat/memory/novel/office methods+types）：Pick 清单 +25 名、方法 25 件；contract.test.ts NOT_MOCKED **清空**（45→0，历四刀），注释改写为 fail-closed 语义：任何新绑定不补 mock 直接红，例外须重新登记并写明理由。
- tsc -b 绿 / contract+store 金样 60 例绿 / **全量 vitest 435 文件 3762 例全绿** / mock 目录 eslint 0 错 0 警。

## 四、余量与下一批

- **NOT_MOCKED 池收官**（批 17 留池「45 条补 mock」关账）：绑定面 744（现行 635+legacy 109 拍板待裁）与非 legacy 面 mock 完整性 100%。
- 观察池 +1：WhisperSubgraph 大小写疑似真机 bug（本批注释在册）。
- 下一批候选：对账地图其他活池或留池测试补强件（compose 检索路径/MPP9 门控/:54 暖启判据）；coupling ~20 与零散死码等拍板。
