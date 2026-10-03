# 全仓审计第 43 批 · NOT_MOCKED 首刀（novel 长篇族 11 件补 mock，白名单 45→34）· 2026-10-03

> 接续 [round-44（批次四十二）](round-44-p1-batch42.md)。取留池「45 条 NOT_MOCKED 补 mock」首刀：**无 AI 依赖、形状可从 Go 直读的 novel 长篇族 11 件**补真 mock，防腐锁同步删条目。dev mock 是行为新增（申报），**不抬版本**。
> 快照：开工时 HEAD = `0ec0ea93`（批次四十二）；工作树干净。

---

## 一、切面与形状来源

- **选取口径**：评测基线 5 件（Snapshot/SnapshotsList/SnapshotDelete/BaselineSet/Compare）+ 故事脊椎 3 件（SpineGet/SpineSave/StoryHealth）+ 风格摘要读/清 2 件 + 上下文盘点 1 件——全部零 LLM 或纯状态面，形状逐一对着 Go 实现读（evalSnapshotBody 全字段/对比指标行集与 lowerIsBetter 方向/快照索引投影行键/盘点 {name,runes,note,preview} 四键/spine 缺档 `{version:1}`/digest 缺档 `{exists:false}`/体检空报告口径），**不是拍脑袋的空对象**。
- **评测族做内存闭环**（快照/基线两槽 + 零值快照构造器）：设基线→快照→对比→删除全链在浏览器可走查；对比口径同 Go（base 空=基线无则如实拒绝「尚无基线」、两名相同拒绝、指标行集与方向逐行同源、恒非 stale）。快照名 UTC `20060102T150405Z` 同形。
- **演示口径如实标注**：零值快照=「mock 工程无真实分析链产物」注释在册；盘点恒列固定面四行（setting 预算口径/章节计划/大纲要点/成人向档位回显，对齐 Go 恒列行）数据行全空。
- AI 依赖件（ChapterConverge/OutlineReconstruct×4/SceneCardsPropose/SceneRewrite/StorySpinePropose/StyleDigestBuild/ChapterAnnotations/ChapterScenesGenerate 等 12 件 novel 项 + 非 novel 项）留白名单待后续刀。

## 二、落地

- `mock/novel.ts`：Pick 清单 +11 名；模块级内存态（MockEvalBody 类型 + mockEvalBody/mockEvalSnapshotName + snapshots/baseline/spine 三槽）；11 方法按既有批次注释风格（消费方面板 + 演示口径说明）。
- `mock/contract.test.ts`：NOT_MOCKED 白名单删对应 11 条（防腐锁「谁补谁删」口径，45→34）。

## 三、门禁

- tsc -b 绿（frontend）/ contract+store 金样 60 例绿 / **全量 vitest 435 文件 3762 例全绿** / 改动两文件 eslint 0 错 0 警（prefer-const 一处当场修）。
- **坑复训**：中途一次在仓库根跑 `npx tsc -b` 命中假 tsc 包 exit 0 假绿、vitest 错配置 24 假红——**门禁必须 cd frontend**（在册教训再实证，vitest 同罪）。

## 四、余量与下一批

- NOT_MOCKED 余 34 条（novel AI 族 12 + 角色/对话/子代理/DAG/办公/语音/绘图族 22）；下一刀候选=DAG 验收两件（methods_dag 有真实 mock 状态机可真验收）或子代理/事件族空态件。
- 下一批候选：NOT_MOCKED 续刀或对账地图其他活池；coupling ~20 与零散死码等拍板。
