# gaea 自动门可见化 + CI race 扩面（刀8 race 项收口）规格书

> 2026-09-17 立项。来源：用户指令「继续」。7.3-2 仍锁窗（≈09-23）。
> 两件收口：①v4.326 观察池第一项——chapter-gate 事件零前端消费，生成后
> 自动门对用户不可见；②revolution 刀8 第三项——CI race 门禁已在（S0.5）
> 但只盖办公核心包，小说 revolution 期新生包全在门外。
> 目标版本：v4.331.0。

## 1. 论点

- 自动门（v4.326）异步跑完只落 slog+事件，用户在创作页毫无感知——「生成
  →体检→分析同步」这条主轴的最后一环（知会作者）缺失。
- race 门禁现盖 gaea/agent·tool·control + novelstyle·novelcontext·narrative；
  chapter/analysis/promptstore/novelgate/rewrite/characterstate 六个新生包
  无并发防线（自动门/覆盖缓存/状态机都在这些包的并发路径上）。

## 2. 裁决

| # | 裁决 |
|---|---|
| Q1 | chapter-gate 消费=创作页**轻通知**（antd message.info key 防叠，6s）：「第 N 章生成后自动体检：契约 X 项 · 质量 Y 项 · AI 味 Z · 分析已同步」——零弹窗打扰，空体检显示「完成」 |
| Q2 | 新钩子 `useChapterGateNotice`（useChapterStream 同款 runtime.EventsOn/EventsOff 通道模式）；分支章通知带分支标记；非 report 类型忽略 |
| Q3 | race 扩面六包：chapter/analysis/promptstore/novelgate/rewrite/characterstate——**internal/app 不进**（race 下已知 TempDir 清理竞争 flaky 先例，扩面候单独评估），注释写明 |
| Q4 | revolution 刀8 行不整行勾选（跨模块验收+回退演练未做）——行内补注 race 项已扩面收口 |
| Q5 | 零新绑定（707 不动） |

## 3. 落地

- 前端：`useChapterGateNotice.ts`（含载荷窄化类型）+ CreatePage 挂载 +
  测试（通知文案拼装/空体检/非 report 忽略/卸载退订）。
- CI：`.github/workflows/ci.yml` race job `go test -race` 追加六包 + 注释。
- revolution 文档刀8 行补注。
- 门禁：go build/vet、tsc -b、eslint、vitest 全量、ci.ps1、版本三处 4.331.0。

## 4. 观察池

自动门报告落地页（点击通知跳 t7 章节分析面板）；internal/app race 扩面；
跨模块验收+回退演练（刀8 余项，候排）。

## 5. 门禁

同上；race 六包本地跑普通模式全绿（CI 侧跑 -race）。
