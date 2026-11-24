# gaea-20261124-58be0f5e

- 交付提交：`58be0f5e`（`docs: archive 41 outdated documents and rebuild the docs index`）
- 相对上一交付包 `gaea-proposal-library-20261124-ddce2d4a` 的增量：四笔提交（`07258861` / `77291f35` /
  `158ed9f7` / `58be0f5e`），全部是记录与文档清理——包记录与夹具拍板、项目长期记忆
  `.gaea/AGENTS.md` 回到现行事实、`docs/` 索引重写与 41 份过期文档归档、17 个源码文件各一行
  注释里的文档路径。**没有功能变化。**
- 产品版本：`4.544.0`（日常装机包，不抬版本）
- 构建工作树：`C:\AI\wubigrok\.worktrees\pkg-58be0f5e`，detached HEAD，
  node_modules 为主树 junction
- 前端：`npm.cmd run build`，Vite `38.48s`
- 桌面：系统默认 `TMP/TEMP`、`CGO_ENABLED=1`、ucrt64 gcc；PATH 前置主树
  `build/bin`（DLL 三件套）；`wails build -s -ldflags "-s -w" -trimpath`，
  bindings 未跳过，Wails `1m11.086s`
- 漂移：Wails 再生 28 个 wailsjs 文件，仅 CRLF/EOL 噪音，`git diff --ignore-cr-at-eol` 内容差 `0`
- 冒烟：正式 payload `/api/health` 返回 **200**（`{"status":"ok"}`，PID 33468 已回收；
  冒烟前后 `gaea.exe` 进程清单均只有用户的装机壳 PID 32336）
- 推送：交付提交 `58be0f5e` 已于 2026-11-24 推送 origin，本地与远端一致

## 可复现性

`payload/gaea.exe` 的 SHA256 与上一交付包 `ddce2d4a` **逐字节相同**（`71205F07…B6309D`）：
增量只有文档、记录与注释，编译产物不变。装机位 `C:\AI\gaea-app\gaea.exe` 就是这份二进制，
**本包未重新装机**——重装只是重启壳，没有内容差异。

## 验证

- 本提交范围：`node scripts/check-docs.mjs` OK（孤儿 0 / 悬空 0）；17 个被改源码文件逐词 diff
  均为注释里的纯路径替换，gofmt 无变化，触及的 Go 包构建通过；本次 Wails 构建含 cgo 成功
- 未跑全量测试：源码改动仅为注释，编译产物与已通过 Go 131 包 / 前端 4734 例验证的上一包相同

## 包布局

- `payload/gaea.exe`
- `payload/onnxruntime.dll`
- `payload/sherpa-onnx-c-api.dll`
- `payload/sherpa-onnx-cxx-api.dll`
- `gaea-20261124-58be0f5e.zip`：上述四件运行文件
- `gaea-20261124-58be0f5e-src.zip`：`git archive` 自交付提交的源码包（6551 文件）

## SHA256

| 文件 | SHA256 |
| --- | --- |
| payload/gaea.exe | `71205F079B9D6796D93756E3A198AE248169F1E0B72B0581D6DF566201B6309D` |
| payload/onnxruntime.dll | `68D0D5F502BE4509E2A16F932627A95137D748B2FAE519027F80125C0B1F550B` |
| payload/sherpa-onnx-c-api.dll | `408F9AB683D1B77CBF7F06D48BD3AF543ED783CC9093C19F520C7C8524619C8F` |
| payload/sherpa-onnx-cxx-api.dll | `DBAC55CECA8383FE28BD7C656FD125BEC60DD35D7DF1A3A28EA36A503EF6C562` |
| gaea-20261124-58be0f5e.zip | `A0BA73587AD2EF4A4D48173FC37D9E9EBF5717B5A3F0F5CE292AEEDA3513346D` |
| gaea-20261124-58be0f5e-src.zip | `41AB572251E8A8D7204F7C27189CB063D5A22F8DAAE81A7CE9AAC30B7D7C29E1` |

## 边界

- 日常装机包，不抬产品版本。
- ZIP / 源码包留盘不入库；`PACKAGE-INFO.md` 与 `SHA256SUMS.txt` 入库。
