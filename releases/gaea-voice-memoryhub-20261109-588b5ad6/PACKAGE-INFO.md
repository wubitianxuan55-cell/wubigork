# Gaea 语音交互窗口预览包

**包标识：** `gaea-voice-memoryhub-20261109-588b5ad6`

**性质：** 功能提交预览，不是正式版本号发布。

## 构建来源

- 提交：`588b5ad60a23a12c6e9a7490225a12f13f97a83e`
- 提交说明：`feat(voice): mirror memory hub in interaction workspace`
- 构建方式：在独立干净工作树中检出上述提交，先执行前端生产构建，再执行
  `wails build -s -ldflags "-s -w" -trimpath`。
- 构建工具：Wails `2.13.0`、Go `1.26.10 windows/amd64`、Node.js `25.2.1`、
  npm `11.6.2`、UCRT64 GCC `16.1.0`。
- 构建时间：2026-11-09 20:09:46（Asia/Shanghai）。
- 构建未包含主工作树未提交的模型引擎改动。

## 本次界面

- 虚空大脑作为全窗主视觉，居中显示并随语音状态反馈。
- 右侧转写区完全透明：无背景、无边框、无阴影、无模糊。
- 底部操作拆为独立胶囊卡片并水平居中。
- 深色与浅色主题均可用。
- 中央字幕保留实时片段和最近一轮问答。

详细设计和验证记录见 `gaea-voice-memoryhub-ui-20261109.md`。

## 使用方式

将整个目录解压到同一路径后，直接运行 `gaea.exe`。`gaea.exe` 必须与下列三枚
DLL 保持同级；缺少任一 DLL 时语音链路不可用。

## 文件校验

| 文件 | 字节数 | SHA256 |
|---|---:|---|
| `gaea.exe` | 61,605,376 | `0E739EA589559072DD6E83FA8404D844AFC420DC6BBE9381076C3BEC3EE987E4` |
| `onnxruntime.dll` | 19,288,888 | `68D0D5F502BE4509E2A16F932627A95137D748B2FAE519027F80125C0B1F550B` |
| `sherpa-onnx-c-api.dll` | 4,294,656 | `5DD36DBD3572846B72C4A40A4017C5A770160EE8E00DA5053BEBC8B2632501E8` |
| `sherpa-onnx-cxx-api.dll` | 264,192 | `4C56327796CDB980DCBF90025C28093D89CA286088FB74C3B107F2F633E67FFE` |
| `gaea-voice-memoryhub-ui-20261109.md` | 4,133 | `41557E3BB56E0A9A2F6E37A09A8F404A1187533940B21DF2CEA42EB1833CDDB3` |

外层 ZIP、源码归档及本包的校验值见
`../SHA256SUMS-voice-memoryhub-20261109-588b5ad6.txt`。

## 验证

- 干净提交的前端生产构建通过。
- Wails 桌面构建通过，bindings 未跳过。
- exe 与三枚语音 DLL 放入正式 `build\bin` 后执行 `scripts\smoke.ps1`。
- `/api/health` 返回 `200`，响应为 `{"status":"ok"}`。

## 版本边界

工作区存在既有版本源漂移：

- `internal/app/app_info.go = 4.531.0`
- `wails.json = 4.535.0`
- `versioninfo.rc = 4.533.0`

本包不修改或冒充其中任一正式版本号，仅以功能提交号标识。
