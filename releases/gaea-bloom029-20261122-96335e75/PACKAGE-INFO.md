# gaea-bloom029-20261122-96335e75

- 提交：`96335e751a6b71b6d8bff84dbf3671ce87706ba6`
  （`feat(gaea): align Bloom appearance tokens with upstream`）
- 分支：`main`
- 产品版本：`4.544.0`（日常预览包，不抬版本）
- 构建工作树：`C:\AI\wubigrok\.worktrees\pkg-96335e75`，detached HEAD
  检出上述提交；`frontend/node_modules` 以 junction 复用主树依赖
- 前端：先执行 `npm run build`，Vite 生产构建 `31.92s`，6611 modules
- 桌面：系统默认 `TMP/TEMP`、`CGO_ENABLED=1`、ucrt64 gcc；
  `wails build -s -ldflags "-s -w" -trimpath`，bindings 未跳过，Wails
  `1m0.058s`
- 漂移：Wails bindings 再生成后仅 CRLF/EOL 状态噪音；
  `git diff --ignore-cr-at-eol` 内容差异文件数为 `0`
- 冒烟：正式 payload 路径 `/api/health` 返回 `200`，PID `61612` 已回收
- 包内容：`payload.gaea.exe` 61,657,088 B；二进制 ZIP 根目录含 exe 与
  `onnxruntime.dll`、`sherpa-onnx-c-api.dll`、`sherpa-onnx-cxx-api.dll`
- 源码包：`git archive` 从提交 `96335e75` 生成，6971 个条目，无目录前缀

## 功能面

- Bloom 0.29 可读强调轨：亮色 `accentL 88% + black 12%`，暗色 `accentD`
- 文案四档、边框四档、状态 tertiary、warning label、danger hover 与侧栏
  active/hover 语义色对齐 upstream
- 修复 `prefers-reduced-transparency` 在已选择材质时被根选择器挡掉的问题
- 顶栏 15 色六列色块网格、单字印记、共用诗句字幕、三模式图标与选中态
- 四层外观架构、15 套 Bloom 配色、三材质、亮暗双套氛围与一键玻璃态

## 验证记录

- 外观/主题/设置定向：4 文件 `110/110`
- 全量前端：518 文件、4706 例；4704 通过，2 例在满并发下失败
  （`ContextView.test.tsx` 水位标、`SchedulingSection.test.tsx` 故障转移开关）；
  两文件隔离复跑 `44/44` 通过
- 仓库快闸 `.tmp` 卫生、版本漂移、Go build/vet 通过；
  `golangci-lint v2.14.0` 被未改动的 Go 基线 98 项问题阻断
- `check-docs` 被两篇既有文档未登记问题阻断：
  `gaea-agent-memory-nebula-distill-2026-11.md`、
  `gaea-day-log-2026-11-22.md`；本次未修改这两处文档索引
- 构建期保留既有 CSS syntax 与 chunk-size 警告，未出现新增编译错误
- 未覆盖安装、未 push

## SHA256

| 文件 | SHA256 |
| --- | --- |
| payload/gaea.exe | `9D619B62FE9AF18BA9DD39DFCC1E1B0345CD0A70641F3911906BCC19FF993803` |
| payload/onnxruntime.dll | `68D0D5F502BE4509E2A16F932627A95137D748B2FAE519027F80125C0B1F550B` |
| payload/sherpa-onnx-c-api.dll | `408F9AB683D1B77CBF7F06D48BD3AF543ED783CC9093C19F520C7C8524619C8F` |
| payload/sherpa-onnx-cxx-api.dll | `DBAC55CECA8383FE28BD7C656FD125BEC60DD35D7DF1A3A28EA36A503EF6C562` |
| gaea-bloom029-20261122-96335e75.zip | `3F831790DE91C875B6FC98996E57C903F8F77E6EC151BDF95E71BB4167E232EF` |
| gaea-bloom029-20261122-96335e75-src.zip | `06F240A7F59A50E5CD8D7232F483550946E864CF8B82B2D5805B3428D1BD1815` |
