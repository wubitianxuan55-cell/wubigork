# gaea-bloom029-starlight-20261122-a37b7548

- 上游基准：`dsh-bloom-theme` `origin/main`
  `bd3e93f9a708d0d951156866a6f2274d0d33c824`（0.29.0）
- 功能提交：`a37b7548f3f9b165128eb2e237f5a312e272ae71`
  （`feat(gaea): add Bloom meteor controls`）
- 星河层级修复：`9a7475ada709b1a1c3bd95afdefe87915a879a87`
  （`fix(gaea): reveal Bloom galaxy starlight above shell`）
- 分支：`main`
- 产品版本：`4.544.0`（日常预览包，不抬版本）
- 构建工作树：`C:\AI\wubigrok\.worktrees\pkg-a37b7548`，detached HEAD
- 前端：`npm run build`，Vite `34.83s`，6611 modules
- 桌面：系统默认 `TMP/TEMP`、`CGO_ENABLED=1`、ucrt64 gcc；
  `wails build -s -ldflags "-s -w" -trimpath`，bindings 未跳过，
  Wails `56.851s`
- 漂移：Wails 再生成仅 CRLF/EOL 噪音，`git diff --ignore-cr-at-eol`
  内容差异文件数 `0`
- 冒烟：正式 payload `/api/health` 返回 `200`，PID `43540` 已回收

## 功能面

- 修复 gaea 壳层 stacking context，Bloom 星尘不再被布局背景吞掉
- `corner / diagonal / galaxy` 三种深色氛围均使用上游 0.29 星尘序列
- `galaxy` 全屏显示星尘、边缘萤火与 60 颗流星候选
- 新增流星三组细节控制：颜色 `colorful / solid`、密度
  `sparse / standard / dense`、速度 `slow / standard / fast`
- 流星设置只在暗色“漫天星辰”下显示，持久化并在根节点实时下发
- 保留亮 15 套配色、暗色模式、三材质、亮色三氛围、自动换色与动效控制

## 验证记录

- `AppearancePanel`、`appStore`、四层架构定向：3 文件 `30/30`
- `tsc -b`：0
- 改动面 ESLint：0
- 本次桌面构建主树与干净树均通过；最终包采用干净树构建
- 未覆盖安装、未 push

## SHA256

| 文件 | SHA256 |
| --- | --- |
| payload/gaea.exe | `19DA7ECD238CE44AAE00EC8BD1B2C0641B203518A6E596ECA18D11B8A5CB7015` |
| payload/onnxruntime.dll | `68D0D5F502BE4509E2A16F932627A95137D748B2FAE519027F80125C0B1F550B` |
| payload/sherpa-onnx-c-api.dll | `408F9AB683D1B77CBF7F06D48BD3AF543ED783CC9093C19F520C7C8524619C8F` |
| payload/sherpa-onnx-cxx-api.dll | `DBAC55CECA8383FE28BD7C656FD125BEC60DD35D7DF1A3A28EA36A503EF6C562` |
| gaea-bloom029-starlight-20261122-a37b7548.zip | `4E3F1D29975559ED906E259A4B1D872B42932E0D17A6DDD4DC318FEE2E3A3DF6` |
| gaea-bloom029-starlight-20261122-a37b7548-src.zip | `E2A9006D17FA5D83137CB15F7A60278BD2E2C4B638FE3BBD6425F84E598E53E7` |
