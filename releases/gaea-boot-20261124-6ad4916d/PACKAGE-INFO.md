# gaea-boot-20261124-6ad4916d

- 交付提交：`6ad4916d`（`feat(boot): 开机动画重设计 + 新玻璃 G 标全局替换`）
- 相对上一交付包 `gaea-20261124-58be0f5e` 的增量：一笔提交 `6ad4916d`（43 文件，
  +2606/−535）。开机动画 BootSplash 换 4K 底片（星际日出/人间山谷 AVIF 3840×2160 共
  1.0MB）+ WebGL2 实时动效（纯时间轴 bootScene/着色器/渲染器，无 WebGL2 退回静态交叉
  淡变），保留语音仪式 0/1.596/4.0s、ESC 跳过、声音开关、进度 68→72→100；新玻璃 G 标
  母版 `build/brand/gaea-mark-2048.png`，`scripts/brand_icons.py` 一键生成 favicon×3、
  appicon×3、ICO 七档、UI 标 `gaea-brand-mark.webp`，办公/文学侧栏、指挥轨道、欢迎页、
  关于页五处容器尺寸不动；删旧原画底片与旧品牌 jpg；效果图与生成原件随
  `design/boot-animation-2026-11/` 入库。
- 产品版本：`4.544.0`（日常装机包，不抬版本）
- 构建工作树：`C:\AI\wubigrok\.worktrees\pkg-6ad4916d`，detached HEAD，
  node_modules 为主树 junction
- 前端：`npm.cmd run build`，Vite `45.93s`
- 桌面：系统默认 `TMP/TEMP`、`CGO_ENABLED=1`、ucrt64 gcc；PATH 前置主树
  `build/bin`（DLL 三件套）；`wails build -s -ldflags "-s -w" -trimpath`，
  bindings 未跳过，Wails `1m4.597s`
- 漂移：Wails 再生 wailsjs 28 文件，仅 CRLF/EOL 噪音，`git diff --ignore-cr-at-eol`
  内容差 `0`
- 冒烟：正式 payload `/api/health` 返回 **200**（`{"status":"ok"}`，冒烟进程 PID 39552
  已回收；smoke.ps1 只 `Stop-Process` 自身 PID，不触碰其他实例）

## 可复现性

`payload/gaea.exe` SHA256 `961C6812…A3933`，与上一包 `71205F07…B6309D` **不同**
——本包含真代码增量（开机动画 + 全局品牌图标）。三个 DLL 与上一包逐字节相同。

**INSTALL OK（2026-11-24 15:53）**：装机位 `C:\AI\gaea-app\gaea.exe` = 包版 `961C6812`
（sha256 终验一致），活体 PID 48844（15:53:28 > 构建时刻）。坑：`gaea_install.ps1` 装
主树 `build/bin`——首装误装 15:27 的旧桌面构建 `8FD4BE09`（同源码非包版），包版 exe
复制进 `build/bin` 后重装纠正。装机时用户旧壳已自行退出（非本流程所杀，smoke.ps1 只
回收自身 PID），无「关运行中实例」问题。

## 验证

- 前端全量 518 文件 4746 例绿、`tsc -b` 零错、改动面 ESLint 零错、Vite build 绿（本线
  会话内完成）；生产包无头 Edge 真机走查 1080p/4K/浅色/1366×768/减弱动效
- 挂池疑点（非阻断）：真机里语音晚起时画面时钟可能被语音接管拉回（进度回跳），代码推演
  成立但未实证；若用户反馈开机进度回跳，修法=autoplay 可用时等语音真正起播再起表、语音
  接管时只前 seek 语音画面不回退

## 包布局

- `payload/gaea.exe`
- `payload/onnxruntime.dll`
- `payload/sherpa-onnx-c-api.dll`
- `payload/sherpa-onnx-cxx-api.dll`
- `gaea-boot-20261124-6ad4916d.zip`：上述四件运行文件
- `gaea-boot-20261124-6ad4916d-src.zip`：`git archive` 自交付提交的源码包（6969 文件）

## SHA256

| 文件 | SHA256 |
| --- | --- |
| payload/gaea.exe | `961C6812D6E849FA62D8B2B719FD41373D50BDD93651B04587237CF97A8A3933` |
| payload/onnxruntime.dll | `68D0D5F502BE4509E2A16F932627A95137D748B2FAE519027F80125C0B1F550B` |
| payload/sherpa-onnx-c-api.dll | `408F9AB683D1B77CBF7F06D48BD3AF543ED783CC9093C19F520C7C8524619C8F` |
| payload/sherpa-onnx-cxx-api.dll | `DBAC55CECA8383FE28BD7C656FD125BEC60DD35D7DF1A3A28EA36A503EF6C562` |
| gaea-boot-20261124-6ad4916d.zip | `E1F85F15EB580BFA3482D5945E03A0E7B26A7CF750B98AE98808BA96739E6EF0` |
| gaea-boot-20261124-6ad4916d-src.zip | `6CEC4AE79FD79C6A63939878584DF93810BCBD43774313CA88B491DA1FA2B9C1` |

## 边界

- 日常装机包，不抬产品版本。
- ZIP / 源码包留盘不入库；`PACKAGE-INFO.md` 与 `SHA256SUMS.txt` 入库。
- 开机动画进度百分比仍按时间演示，未接真实启动事件（挂池）。
