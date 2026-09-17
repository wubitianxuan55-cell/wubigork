# gaea 阶段七出口对账 + 7.3-1 会话级回源收口规格书

> 2026-09-17 立项。来源：用户指令「继续」。阶段七三门刀全部落地（7.1 v4.312/316
> / 7.2 v4.313/317/319 / 7.3 v4.318/332）——本刀=出口判据逐条复核对账（诚实
> 注记挂账项）+ 收掉 7.3-1 唯一结构欠账「会话级回源」（v4.318 出口对账遗留：
> Session 路径已落库，但 resume seam 活在 gaea App 树内，收件箱在首页拿不到）。
> 目标版本：v4.333.0。

## 1. 会话级回源设计

- **通道**：新模块 `gaea/lib/pendingSessionResume.ts`——模块级单例 pending +
  `requestSessionResume(path)`（set pending + 派发 window CustomEvent
  `gaea:resume-session`）。**gaea 板块 keepAlive=true**：已挂载态走事件直达；
  冷启态（首次挂载）事件已丢、由 App 挂载时 consume pending 兜底——两态皆达。
- **消费**：gaea App.tsx 挂载 effect——consume pending → resumeRecentSession
  （内部按 projectGroups 解析跨项目并切工作区后恢复）；同时 addEventListener
  事件 → 同路恢复 + 清 pending；卸载移除监听。
- **收件箱侧**：TaskInboxBoard +可选 `onResumeSession?: (path: string) => void`；
  源会话按钮——有回调且 task.session 非空走精确回源；否则回退 V1 板块粒度
  onNavigate('gaea')（既有行为零变化）。ModuleLauncher/TasksFirstHome 接线：
  `requestSessionResume(path); onNavigate('gaea')`。
- Session 字段落库已是会话路径（palette 存 currentSessionPath 实证）。
- 零新绑定；零后端改动。

## 2. 出口对账（docs/gaea-stage7-plan §5 及各节判据注记）

| 门 | 状态 | 注记要点 |
|---|---|---|
| 7.1 | 已落 | 评测基线落档+CI+releases ✅；路由账目/归因/建议制 ✅；**挂账=本地引擎建议「真实被采纳样本」待真机** |
| 7.2 | 已落 | 双入口在产、调用计数可见（v4.319）；**挂账=录制复跑 ≥80% 抽样（真机）+ 结晶技能 ≥5 次真实调用（≈09-30 清池核数）** |
| 7.3 | 已落 | 收件箱四判据 ✅（②会话级回源**本刀收口**）；7.3-2 三判据 ✅（v4.332）；**缺省 classic→tasks 翻转候用户试用拍板** |
| 7.4 | 已清 | t6（323/324）/t7（325）/Gate（326）/画室一（327-329）全清 ✅ |
| 总判据⑤ | 欠账 | **每刀真机走查 ≥1 条**——v4.319~v4.332 各刀走查挂池未走（须用户闲置窗口），列下一班清池首项 |

## 3. 测试

- pendingSessionResume 单测（set/consume 清零；requestSessionResume 派发事件）。
- TaskInboxPanel 增例：有 onResumeSession → 按钮带 task.session 精确回调；
  无回调 → V1 板块粒度回退（既有用例零改动）。
- gaea App 消费挂接：以模块+面板契约为准（App 树重测试不建）。

## 4. 门禁

tsc -b / eslint / vitest 全量 / ci.ps1 / drift OK@707（零新绑定）/ 版本三处 4.333.0。

## 5. 观察池

回源会话已删除/归档时的诚实降级（handleResumeSession 既有吞错口径）；收件箱
行内会话标题预览（现只显按钮）；多工作区同名的消歧提示。
