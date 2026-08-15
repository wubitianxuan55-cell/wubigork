# gaea3-review — 调研档案索引（只读存档）

> 本目录是 2026-08-15 gaea 3.0 架构改造调研的评审报告存档。
> **权威设计文档**：../2026-08-15-gaea3-architecture-design.md（v1.0 定稿）。
> 本目录内容仅供实施阶段查阅证据（文件:行号），不参与日常决策；无需再更新。

## 报告清单（9 份，全部只读调研，带文件:行号证据）

| # | 域 | 文件 | 对应设计章节 |
|---|---|---|---|
| 01 | 前端壳层（MainLayout/appStore/hooks/api/事件订阅） | 01-frontend-shell.md | §5.2 + 附 B |
| 02 | 前端业务页面（8 板块页面组件树/绑定调用面） | 02-frontend-pages.md | §3.1 + §5.2 |
| 03 | 办公工作台前端（bridge 契约/事件→UI/store） | 03-office-frontend.md | §5.1 兼容红线 |
| 04 | 后端壳层 app（装配/门面/主脑/事件出口） | 04-backend-app.md | §5.4 + Step 0 |
| 05 | 办公引擎核心（agent 循环/seam/session） | 05-engine-core.md | §5.1 |
| 06 | 办公引擎能力与存储（工具注册/存储面/沙箱/MCP） | 06-engine-capability.md | §5.2 工具面 + §5.3 + §9 |
| 07 | whisper 轻语域（232 文件结构/记忆管线/语音链） | 07-whisper.md | §3.1 chat 板块归属 |
| 08 | 模型与媒体引擎层（31 处 switch 清单/子步排序） | 08-model-media.md | §5.3 Step 3 |
| 09 | DSH 参考机制（事件日志/seam/作用域/装配/UI slot） | 09-dsh-reference.md | §5.1-5.3 + 附 |

## 关键结论速查（设计文档已吸收，此处仅为来源索引）

- 板块五层清单不一致（导航 9 页/注册表 4/README 7/门面 10/服务域 29）→ 设计 §3.1
- MainLayout 12 个硬编码点 → 设计附 B
- 事件日志挂钩点 core.emit（app.go:214-226）+ 持久化 sink 插 boot.go:114 后 → 设计 §5.1
- 审计 trail（AuditLogger）代码就绪但离线 = Step 1 最小改造面 → 设计 §5.1
- Step 3 子步排序 3a Image → 3b LLM → 3c OCR/ASR/TTS → 3d 分类统一 → 设计 §6
- DSH 五条低成本增量（torn-tail/whole-value/单决策槽/per-agent 工具/预设入日志）→ 设计 §5
- 42 注册工具（22 内置 + 13 boot + 7 桌面 ExtraTools；ocr/semantic_search 定义未注册）→ 设计 §5.2
