# 当日总账（2026-11-11）：knip 尾款清零+天璇退役漏网收口（代码面卫生小批）

> 承接 2026-11-10 晚自主批（空间契约漂移刀+装机+代码图谱全清）后，用户令
> 「继续优化完善 gaea」。拍板池盘点余项全卡用户输入（沙箱实弹五面/A-B 评测/
> 语音 tap 撤除/V5 拍板等），按可自主验证面推进：九月 v4.179 批挂账的
> **knip Unused exports 169 项三态甄别**欠账 + 顺带收口天璇退役漏网。

## 盘点结论：169 项已被后续批次自然排干

- 全类目 knip 复扫（过滤 wailsjs 生成物）：自研未用导出仅剩 **3 项**——
  `startTianxuanModel`/`startStrataModel`（api/engines.ts）+
  `STREAM_INACTIVITY_TIMEOUT_MS`（ChatPage 再导出）。九月欠的 169 项
  大头（unused exported namespace members 159）随其后各批死文件清理/
  重构自然消化，无需逐项甄别。
- 假阳性甄别在案：`public/voice-worklets/*.js`=AudioWorklet URL 运行时
  挂载（useVoiceChatV2 两处 + worklets.test fs 直读），knip 静态面不可见，
  **保留**；`preview-atlas-main.tsx`=代码图谱预览壳（用户真机目检未收，
  上批留树，**保留**）。

## 刀1 天璇退役漏网收口（-23 行，c7da4558）

- **根**：11-05 天璇退役刀（-1243 行）删净了 Go 侧方法与 wailsjs 绑定，但
  漏了三处下游：engines.ts 的 `startTianxuanModel` wrapper、
  `scripts/gen_bindings/main.go` 参数表五条 Tianxuan 条目
  （Engine/Model 的 Status/Start/Stop）。
- **静默带病机制**：`AppFacade = AppAPI & Record<string, (...args) => Promise<unknown>>`
  （types/wails.d.ts:130）索引签名让任何方法名过 tsc——wrapper 调用已不
  存在的绑定编译期零告警，真调用必炸（`undefined is not a function`）。
  属「零调用方死码」故从未实炸，但留一天多一天漂移。
- **Strata 侧同刀**：`startStrataModel` 前端零调用方，useEngineState.test
  注释明示「刻意未接线」；删 wrapper、注释改指
  StrataEngineStart/Stop 档位路径。**Go 侧 StrataModelStart/Stop 绑定与
  集成测试保留**（tested 预留面，不走前端就是了）。
- ChatPage 删 `STREAM_INACTIVITY_TIMEOUT_MS` 无消费方再导出（真源
  chat/constants.ts，useChatStream 直用；SILENCE/HARD 测试仍从 ChatPage
  导入，保留）。
- gen_bindings 顺手修 WebSearch 三行 gofmt 对齐（既有欠账，stash 对照
  实证非本刀引入）。

## 门禁

- 定向 vitest **83/83**（api/engines 全部 12 个消费测试文件里跑了两批 12
  文件：ChatPage 20+useEngineState 8+mock-contract 四件+ModelPanel/
  RuntimePage/Benchmark/EngineSection/ImageGenPanel）。
- `tsc -b` 0、eslint（三改动文件）0、**knip 自研未用导出复扫清零**、
  gofmt 合式。全量 2738 未跑（范围=删零引用导出，tsc 已证全仓零引用；
  定向面覆盖全部直接消费方）。
- 无运行时行为变化（删的全是零调用方），**本批不装机不发版**。

## 杂项清障

- `.tmp/gaea-voice-588b5ad6` 遗留工作树（11-09 语音打包批）移除：先查
  改动=29 条全是 wails build 再生的 wailsjs 绑定物（无真工作）后
  `git worktree remove --force` + 残留 12K 目录清掉 + prune。codex 会话
  工作树（chat-search-reliability）不动。

## 遗留

- knip「Unused exported namespace members 159」与「duplicate exports
  SILENCE|INACTIVITY 别名」为风格面小账，不值得立刀（下次顺手）。
- CI race linux 6 例平台化刀仍挂（gh 未装+API 不可达，远端无法核对，
  不盲动）。
- 代码图谱预览壳与 5231 dev server 仍留树待用户真机目检。
