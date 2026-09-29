# 长篇刀4：质量收敛闭环（v4.432.0）

> 用户指令「继续」（长篇七刀线程，v4.431 刀3 后按次刀序开刀4）。规格依据
> `docs/gaea-longform-novel-system-2026-09.md` §1.7（质量门不收敛：全是报告，
> 从报告到修复是人工动作，没有「体检不过→定向修补→复检→判收敛」的自动闭环，
> 也没有收敛判据避免无限重写）与 §5 刀表刀4 行。

## 0. 落地形态

### 收敛判据（线1，纯函数零 LLM）
`convergeCheck(text, targetTaste) → {tasteScore, s1s2 []Issue, s3 []Issue, converged}`：
- **收敛** = AI 味分 ≤ targetTaste（默认 40）且确定性 S1/S2 质量信号归零；
- S3 仅提示不阻断（与门禁 severity 口径一致）。

### 收敛闭环（线2，流式 `NovelChapterConverge`）
每轮（默认上限 2 轮）：
1. **快照先行**：本轮前整章存版本库（`RewriteMode` 新增 `converge` 值——轨迹即
   版本历史，可逐轮回滚）；
2. **定向修补**：把 S1/S2 清单 + AI 味命中要点拼成重写指令，走既有 LLM 重写链
   （`rewriteUnit`）逐单元修补；LLM 后仍高于目标再走**确定性去味**（`DeSlopRewrite`）
   兜底；
3. **复检**（同一纯函数重跑）；
4. **判停**（三态，避免无限重写与越改越糟）：
   - `converged`：达判据即停；
   - `stopped(no-improve)`：本轮 AI 味未降（after ≥ before）→ **回滚本轮**（版本库
     Restore 语义：磁盘回到本轮前）并停；
   - `stopped(max-rounds)`：轮次耗尽，剩余项如实报告。
流事件 `converge-stream`：`round-start{round,before}` / `round-done{round,before,
after,applied}` / `converged{rounds,taste,remainingS3}` / `stopped{reason,taste,
remaining}` / `error`。

### 预检（线3，dry-run `NovelChapterConvergePreview`）
返回当前判据状态（AI 味分/S1S2 清单/是否已收敛/计划轮次）——前端确认弹窗数据源，
不发起修补。

### 绑定（线4，+2）
`NovelChapterConverge(chapterNum, maxRounds, targetTaste)` /
`NovelChapterConvergePreview(chapterNum, targetTaste)`；733→735。

### 前端（线5）
CreatePage rail「收敛修补」→ 预览弹窗（当前分/清单/计划）→ 执行中逐轮轨迹行
（before→after + applied）→ 终态如实（达标/无改善已回滚/轮次上限+剩余项）。

## 1. 验收

- 判据纯函数矩阵（收敛/未收敛/S3 不阻断）；
- 闭环三态：达标即停（不烧多余轮）；无改善回滚（磁盘字节回本轮前）；上限停+剩余报告；
- 每轮版本库留痕 mode=converge；
- 空/零问题章诚实拒绝（已收敛直接 converged 零轮）；
- 绑定 733→735 四处同步。

## 2. 明确不做（本刀）

- 平台评审（novelreview 15 维）并入判据——外部标准报告制，另刀；
- 场景卡字段级修补（刀2 观察池维持）；
- 自动默认开启（作者按需触发，与体检按钮口径一致）。

## 3. 落地情况

> **已发版 v4.432.0（2026-09-30）**。Go 新 converge_handler.go + RewriteModeConverge
> 常量 + 前端 ConvergeModal 新组件 + CreatePage rail 挂载；**绑定面 733→735、
> spaceBindings 557→559**。测试 Go 6 新例（含 no-improve 回滚集成 + 回滚函数直测
> 〔语义改坏复验必红〕）+ 前端 4 新例；小说域 37 文件 / 325 例；全量 ci.ps1 CI OK。
> 坑：no-improve 集成场景里 rewriteUnit 安全闸常使回滚成 no-op——回滚灵敏度靠
> 直测钉（写脏→回滚→字节还原）；v4 无场景的混合态章回落 blob 单元（与
> ReadChapterAsStitch 口径一致）。
