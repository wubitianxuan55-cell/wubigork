# 全仓审计第 47 批 · 留池测试补强三件（compose 补召回 / MPP9 硬断言 / :54 暖启闸）· 2026-10-03

> 接续 [round-48（批次四十六）](round-48-p1-batch46.md)。收批 18/19/20 三条「测试增强」留池：三根新钉 + 一处 var 测试接缝 + 三组反向变异全红→绿。零生产行为变化（唯一生产面改动=semanticCostRecall func→var 接缝，调用点零改动），**不抬版本**。
> 快照：开工时 HEAD = `6ef83631`（批次四十六）；工作树干净。

---

## 一、三件落地

- **compose `<3 补召回` 编排用例**（批 18 留池）：`semanticCostRecall` func→**var 钩子**（同包 `visionExtractPDF` 注入先例，调用点零改动）；`TestCostComposeRecallBelowThreshold` 双向口径——语料 HP300×2/C30×3/塔吊×2，**命中 2<RecallBelow(3) 触发召回恰 1 次且桩条目并入证据链、命中 3≥3 不触发**（阈值下界不抖动）。App 面钩子是方法值（编译器锁引用）不另设缝，builtin 面覆盖 Enhance 编排即覆盖两共用出处。
- **MPP9 门控硬断言**（批 19 留池「弱判据盲区」收口）：`TestMppRealSamples` 从 >0 类弱判据改**逐样本真值表**（三样本 任务/搭接/资源/分配/总工期 全钉：146/176/32/158/344 · 43/37/8/28/83 · 323/138/12/317/30，2026-10-03 解析快照取值），与 2013 硬断言门控同款先例；任务名全空护栏保留。
- **:54 暖启武装位闸用例**（批 20 线 1 登记收口）：`TestWarmComfyUIArmedGate` 三态——未武装恒 `unarmed`（含 atomic 复位保序防污染）、武装+非 comfyui 后端 `backend-not-comfyui`、武装+引擎不在跑 `comfyui-not-running`（空 URL HTTP 探测快速失败，零网络依赖）。

## 二、反向变异三组（主代理亲手，红→还原绿）

1. Enhance 调用摘除 Recall 钩子 → compose 用例红（`应触发补召回恰 1 次, got 0`）。
2. 武装位闸摘除（`if !comfyWarmArmed.Load()`→`if false`）→ 暖启用例红（`未武装应 unarmed`）。
3. MPP 工期换算漂移（`durationTenths*2`）→ **三样本同时红**（344→688/83→166/30→60）——正是 round-21 实证「344→1275769 仍绿」的盲区类漂移，新钉子全部咬住。三处还原后 `git diff` 恰为 4 个意图文件，零残留。

## 三、门禁

- gofmt 0 差 / `go build ./...` 绿 / vet 绿 / builtin 全包 12.2s 绿 + schedule 全包绿 + app 定向族（ImageBackend/WarmComfy/CostCompose）绿。

## 四、余量与下一批

- 留池余：findOutlineNodeByNumAny/ByID、Media 链补重试、MarkdownContent 第四样板、compose 检索路径（本批毕）——多为小件可续收；FE6-05 DOM 统一（产品级）、IN1-04 rubric 接入、45 条 NOT_MOCKED（批 43~46 毕）已关或拍板。
- 观察池在册：WhisperSubgraph 大小写疑似真机 bug（批 46 登记）。
- 下一批候选：留池小件续收或对账地图其他活池；coupling ~20 与零散死码等拍板。
