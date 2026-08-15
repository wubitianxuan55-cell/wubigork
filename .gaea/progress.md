# 任务进度

> 最后更新: 2026-08-15（gaea 3.0 执行 · v2.37.0 发布完成）

## 阶段 7 收官 + 3.0 Step 0/1 完成（5 子代理并行，全部提交）

| 状态 | 任务 | 提交 |
|------|------|------|
| ✅ | Step 0 修债（office 注册 GaeaSend + MainBrainChat 测试 8/8 + 版本同步脚本） | 4dbba0c |
| ✅ | Step 1 会话事件日志（append-only 日志 + 投影 + checkpoint + 迁移 + 派生 + GaeaHistory 黄金测试逐字节） | 72fae6c |
| ✅ | T7-2 可见性收口（吞错清零 + 成本凭据 + 批量事务，41 测试） | 0a9fb6f |
| ✅ | T7-3 名实相符（PDF FlateDecode + OCR 容错 + 检索索引 + dashboard 真实，约 33 测试） | d7934eb |
| ✅ | T7-4 前端性能收尾（写路径清零 + 三态错误 + 渲染收敛，41 用例） | 5364281 |

## 发布

- **v2.37.0**（2026-08-15，统一构建发布，含三刀 + Step 0/1）：gaea-v2.37.0.exe（34.5MB）
  - SHA256=37A56F54DF653E3D9E8A5751EA282CEB34BF5BBCA2672D26439BF7BAEBA7A62B
  - 冒烟通过（/api/health 200）；详见 releases/v2.37.0.md

## 门禁快照（父代理实测）

- go build ./... 干净；go vet 干净
- 逐包测试全绿：session(67)/event/boot/config/search/memory/stats/bm25/auth/characterlib/channels/weixin
- internal/app：仅 1 个既有 flaky（TestDrainAndPersistAll_FinalRoundLands，单独 3/3 通过）；docmd 4 个 GBK 环境失败为基线
- 前端 tsc 0 / eslint 0 / vite build 通过；vitest 新增 41 全过（27 个 jsdom localStorage 环境失败为基线）

## 遗留（下一会话）

- Step 1 app 层接线（运行时「日志即真相」：Resume→Restore / Save→日志 / 压缩→checkpoint / flush）+ gen_bindings
- Step 2 板块 Manifest（board 包 + 9 板块 + MainLayout 清单化 + PageRegistry）
- Step 3 Provider Seam（Image/LLM/OCR/TTS seam 化）
- 回退保障硬要求不变（每 Step 独立提交 / 旧数据只读兼容 / 二进制保留 5 版 / 运行时开关 / 回退演练）
