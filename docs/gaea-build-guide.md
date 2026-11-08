# gaea 桌面端构建指南

> 日常构建 + 装机的标准流程与坑清单。发版流程另见 `scripts/release.ps1` 与 CHANGELOG.md 发版约定。

## 标准流程（日常装机）

```bash
# 1. ucrt64 gcc 前置（必须）
export PATH="/c/msys64/ucrt64/bin:$PATH"

# 2. 仓根构建
wails build                  # 产物 build/bin/gaea.exe（约 60MB）

# 3. 装机（exe + 3 DLL → C:\AI\gaea-app，桌面刷新 gaea.lnk 并清散件）
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/gaea_install.ps1
```

`gaea_install.ps1` 可选参数：`-Launch`（装完拉起）、`-KeepDesktopFiles`（保留桌面散件，仅取证用）。

首次构建前前端依赖：`cd frontend && npm install`。开发热加载用 `wails dev`。

## 交付布局（2026-11-05 拍板）

- **exe + 三个 DLL 同装 `C:\AI\gaea-app`**：`gaea.exe` + `onnxruntime.dll` + `sherpa-onnx-c-api.dll` + `sherpa-onnx-cxx-api.dll`
- **桌面只留 `gaea.lnk` 一个快捷方式**，散件默认清掉
- **单文件 exe 不可行**：语音 DLL 是 load-time 链接（Windows loader 要求与 exe 同目录；go:embed 盖不住 load-time import；sherpa-onnx 上游只有 DLL 无静态库，k2-fsa 同理）
- 必须走 ucrt64 gcc：**缺 gcc 时 `wails build` 不报错、cgo 静默降级 stub**，产物能起但语音链路是空壳

## 坑清单

| 坑 | 处置 |
|---|---|
| PATH 无 ucrt64 gcc → cgo 静默降级 stub | 构建前 `export PATH="/c/msys64/ucrt64/bin:$PATH"`；gcc 在 `C:\msys64\ucrt64\bin` |
| 旧壳占锁 build/bin | 构建前杀 gaea.exe；`gaea_install.ps1` 自带杀壳+杀残留校验（KILL FAIL 即退出） |
| `build.bat` 在 **Generating bindings** 报 `frontend/wailsjs/runtime/package.json: Access is denied` | 已复现为 `.tmp` 的 `TMP/TEMP` 重定向触发 Wails 清理/解压 runtime 冲突。先完成 `frontend\npm.cmd run build`，再在**使用系统默认 TMP/TEMP 的新终端**按同一 `CGO_ENABLED=1`/`CC`/`PATH` 运行裸 `wails build -s -ldflags "-s -w" -trimpath`。不要再跳过 bindings。 |
| System32 ORT 劫持 DLL | ucrt64 bin 必须前置在 PATH，防止劫持 DLL 随 exe |
| 版本号真源 = `wails.json` 的 `productVersion` | `versioninfo.rc` 是死文件，改它无效 |
| 测试/运行 exe 找不到 DLL | exe 与三 DLL 必须同目录 |
| 桌面散件旧 exe（2026-10-31 事故） | 用户跑的一直是桌面旧壳；装机脚本默认清桌面散件 |
| 拉起验证 | 壳起来后验 PID 启动时间 > 构建时刻（CIM `CreationDate`），防旧壳假象 |
| 挂死 wails.exe/go.exe 孤儿连环锁（2026-11-07） | 构建 Cancel/失败会遗留 wails.exe+go.exe 孤儿持句柄，后续绑定生成子进程重抽 `frontend/wailsjs/runtime/` 报「package.json: Access is denied」连环假象（文件 ACL 实为全 Allow，`cat`/改名/重建全通）；处置=CIM 找 wails.exe/go.exe/gaea.exe 孤儿 taskkill /T /F 后按本指南 `wails build` 流程走即通 |

## 编程板块与 Windows 子进程

- 编程板块使用隔离 DSH：`C:\AI\dsh`、`DSH_HOME=C:\AI\dsh\home`、
  `DSH_AGENTS_HOME=C:\AI\dsh\home\agents`。
- 浏览器侧 authority 固定为 `dsh.wails.localhost:3091`，否则 DSH 的
  `SameSite=Strict` Cookie 会被跨站 iframe 拒绝。
- 启动任何可能带控制台的 Windows 子进程时必须使用
  `internal/gaea/proc.HideWindow`。裸 `syscall.SysProcAttr{HideWindow:true}`
  会先显示再隐藏，表现为周期性 cmd 黑窗。
- 当前 GPU 遥测的 PowerShell、`nvidia-smi`、`netstat`、ComfyUI Python
  启动点均已走该封装；新增路径照此执行。
- Web profile 的 `dshmarket` 使用本地 pnpm patch 修复 detached helper
  未设置 `windowsHide` 的问题。升级插件后检查 patch 是否仍在。
- 完整环境、REA 与插件版本记录见
  `docs/gaea-programming-rea-plugin-integration-20261108.md`。

## 构建后验收

1. `build/bin/gaea.exe` 修改时间 = 构建时刻
2. 装机脚本输出 `INSTALL OK`，四个文件 sha256 打印且与 build/bin 一致
3. 桌面散件已清（只余 gaea.lnk）
4. 需要实弹时：从 `C:\AI\gaea-app\gaea.exe` 或桌面 gaea.lnk 拉起，验 PID 启动时间 > 构建时刻

## 版本号

- 日常装机不抬版本；产品号只在用户可感批次落地时动（见 README 发版约定）
- 版本源：`wails.json` → `productVersion`；`scripts/sync-version.ps1` 同步三处
