# gaea-appearance-v4-20261122-7fcd1ff5

- 提交：`7fcd1ff51f370ab2062fe2fab23233248346d1a6`（`feat(gaea): rebuild appearance system around Bloom architecture`）
- 分支：`main`
- 产品版本：`4.544.0`（日常预览包，不抬版本）
- 构建：独立干净工作树 `C:\AI\wubigrok\.worktrees\pkg-7fcd1ff5`，detached HEAD 检出该提交；`frontend/node_modules` 以 junction 复用主树依赖
- 前端：`npm run build`，Vite 生产构建 `27.25s`，6606 modules
- 桌面：ucrt64 + `CGO_ENABLED=1`，`wails build -s -ldflags "-s -w" -trimpath`，bindings 未跳过，Wails `47.13s`
- 漂移：Wails bindings 再生成后仅有 CRLF/EOL 状态噪音；`git diff --ignore-space-at-eol --quiet` 内容漂移为 0
- 冒烟：正式 payload 路径 `/api/health` 返回 `200`，PID `16416` 已回收
- 测试：外观、主题、设置、对比度、图表语义色、上下文卡与 GenUI 回归共 8 文件 `106/106`

## 包内容

- `payload/gaea.exe`
- `payload/onnxruntime.dll`
- `payload/sherpa-onnx-c-api.dll`
- `payload/sherpa-onnx-cxx-api.dll`
- `gaea-appearance-v4-20261122-7fcd1ff5.zip`：payload 四件压缩包，留盘不入库
- `gaea-appearance-v4-20261122-7fcd1ff5-src.zip`：`git archive` 从提交 `7fcd1ff5` 生成的源码包，留盘不入库

## 功能面

- 删除旧 7 套主题，重建四层外观架构：原始色板 → 语义令牌 → 材质与氛围 → 组件映射
- 移植 Bloom 15 套配色，其中 12 套精选；默认 `ripple / 天青`
- 支持亮暗自由切换、跟随系统、三种表面材质、四种背景氛围和一键玻璃态
- 玻璃态使用统一 `--gaea-shell-*` 契约；全局 `v3-rail` / `v3-telemetry` 与路由壳层解耦
- 聊天画布恢复 80px 网格，并以径向 mask 实现中心到四周渐隐
- 清理 UI、图表、语法高亮与预览组件的硬编码颜色，新增四层架构与玻璃态机检

## SHA256

| 文件 | SHA256 |
| --- | --- |
| payload/gaea.exe | `268F8558040DEA6A7B5623C491B42D6080F019AC266BD85F4653D0DD563807E0` |
| payload/onnxruntime.dll | `68D0D5F502BE4509E2A16F932627A95137D748B2FAE519027F80125C0B1F550B` |
| payload/sherpa-onnx-c-api.dll | `5DD36DBD3572846B72C4A40A4017C5A770160EE8E00DA5053BEBC8B2632501E8` |
| payload/sherpa-onnx-cxx-api.dll | `4C56327796CDB980DCBF90025C28093D89CA286088FB74C3B107F2F633E67FFE` |
| gaea-appearance-v4-20261122-7fcd1ff5.zip | `713260DEF14639BFBEAA1139E22D058A0EC10928AFFC8F881FF38CB19D64EDC9` |
| gaea-appearance-v4-20261122-7fcd1ff5-src.zip | `0501E823E4181988A66C0B7B5549FE8F9EDF4A27EE81F5D5F76CAC1B18FD2128` |
