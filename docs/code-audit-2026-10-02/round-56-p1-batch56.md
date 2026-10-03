# 全仓审计第 56 批 · 发布就绪终验（全量档 CI 首次联合通过）· 2026-10-03

> 接续 [round-55（批次五十五）](round-55-p1-batch55.md)。审计修复线 55 批的收尾验证：**`scripts/ci.ps1` 全量档（非 -Quick）完整通过**——53+ 批修复以来首次全仓联合验证。本批零代码改动（顺带 `wails generate module` 清净了 wailsjs 陈旧构建噪音，与 HEAD 逐字节一致，无提交物）。**不抬版本。**
> 快照：开工时 HEAD = `99b3988a`（批次五十五）；验证后工作树仍干净。

---

## 一、终验内容（ci.ps1 全量档 = push 前/发版前必跑口径）

- Go：构建 + `go test -count=1 ./...` + golangci-lint（v2.14.0）。
- 前端：lint + build + 全量 vitest。
- 守卫族全绿：E 系列回归守卫（E24 图谱/E25 聊天合并/E26 空间切换/E27 变参禁令等）+ **绑定漂移闸（bindingNames 635 一致 / legacy 75 / 死绑定 0）** + 仓库卫生守卫（AGENTS.md 56552 B 预算内 / .ps1 编码合规 / 孤儿 0 悬空 0）。

## 二、顺带清理

- `wails generate module`：wailsjs 工作树曾有历史会话遗留的陈旧构建噪音（含 A1① 已摘名字），重生成后与 HEAD 逐字节一致（HEAD 的 wailsjs 门面本就不含这些名）——零 diff，仅工作树更干净。

## 三、发布就绪声明

- 审计修复线全部收官：**P0 25 清 / 四簇+留池+上报未动池+观察池全清或归拍板 / 拍板池 16 项中 14 项已执行或零动作确认**。
- 发版待办（等用户示意）：版本号+CHANGELOG+构建 exe+SHA256SUMS+源码包归档（releases/README 三处同步）+tag+push（44+ 批本地提交随发版推 origin）。
- 遗留观察池：fsync 全量基准（需真机写路径基准）、WhisperGraphPanel 真机复走（等闲置窗口，随真实数据验证出图）。
- 拍板池仅余 B1/B2 两项新功能立项（桌面 agent 接线、安装确认 UI），等排期示意。
