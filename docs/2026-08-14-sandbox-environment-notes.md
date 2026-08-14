# gaea 开发沙箱环境备忘（2026-08-14）

> 目的：记录本 DSH 沙箱（C:\AI\wubigrok 会话）中踩过的全部环境坑与已验证的解法，
> **防止下一次会话犯同样的错误**。本文件是 .gaea/AGENTS.md「沙箱环境备忘」章节的详细版。
> 所有解法均已在本会话实测通过。

## 一、总览：本环境的四个已知限制与对策

| # | 限制 | 对策 | 状态 |
|---|------|------|------|
| 1 | 权限策略可变（workspace-write → danger-full-access） | 开工先 `pwsh` 试写一次 AppData；策略变更通知到达后按新策略行事 | 已解决 |
| 2 | Go 遥测噪音（upload.token Access denied） | `go telemetry off`（持久生效，已执行） | ✅ 已解决 |
| 3 | **wails build 前端编译挂起**（wails 捕获前端输出走管道） | **先独立 `cd frontend && npm run build`，再 `wails build -s`**（跳过前端编译） | ✅ 已解决 |
| 4 | `go test ./...` 单进程树跑大量测试二进制会被 harness 终止/偶发 `fork/exec Access is denied` | 逐包验证 + `scripts/test-all.ps1`（逐包+重试+状态续跑）；exec 拒绝用 `go test -c` 手动编译运行证明代码无恙 | ✅ 已解决 |

## 二、逐项详细记录

### 1. Go 构建缓存与遥测写入

- **症状**：`go test` 报 `open C:\Users\wubi\AppData\Local\go-build\...: Access is denied`；
  stderr 报 `error acquiring upload token: creating token file: ...\go\telemetry\local\upload.token: Access is denied`。
- **成因**：旧策略（workspace-write）禁止写用户 AppData；Go 构建缓存与遥测 token 都在那里。
- **解法（双管齐下，都已验证）**：
  1. 持久：`go telemetry off`（GOTELEMETRY 变为 off，噪音消失；`go env -w GOTELEMETRY=off` 会报
     "cannot be modified"，必须用 `go telemetry off` 命令）；
  2. 临时兜底：`$env:GOCACHE / GOTMPDIR / TMP / TEMP` 指到 `C:\AI\wubigrok\.tmp\...`（目录需先建）。
- **注意**：danger-full-access 策略下默认 GOCACHE 已可写，无需再覆盖——先测默认，失败再覆盖。

### 2. wails build 前端编译挂起（最坑的一个）

- **症状**：`wails build` 日志停在 `Compiling frontend:` 之后不再前进（npm install 已 Done），
  等 15 分钟无进展；独立 `npm run build` 或 `npx vite build` 却 24 秒成功。
- **成因**：wails 自行捕获前端构建子进程的输出（管道），触发沙箱的 named-pipe/EPERM 边界，挂起。
- **解法（验证通过）**：
  ```powershell
  cd frontend; npm run build        # 24s，产出 frontend/dist 并刷新 wailsjs bindings
  cd ..; wails build -s             # 8.9s，跳过前端编译，产出 build/bin/gaea.exe（32MB）
  ```
  `-s` = `--skipfrontend`。注意 `npm run build` 会重新生成 `frontend/wailsjs/`（已跟踪文件，需一并提交）。

### 3. go test 全量与测试二进制 exec 拒绝

- **症状 A（harness 终止）**：`go test ./...` 前台偶发 `Error: spawn EPERM`（命令根本没启动）、
  后台被静默杀掉（日志截断在 4~66 行）、长循环（多包 go test）在 ~2 个包后被终止。
- **症状 B（Windows exec 拒绝）**：个别测试二进制 `fork/exec ...\Temp\go-buildXXXX\bXXX\*.test.exe: Access is denied`
  （chat/herdsman/permission/search/prompt/scene 均出现过）；重跑可能复现也可能通过。
- **判定方法（关键）**：`go test -c -o X.exe <pkg>` 手动编译 → 直接运行 `X.exe` → **PASS 即证明代码无恙**，
  拒绝纯属环境（防病毒扫描锁/沙箱 exec 策略）。
- **对策**：
  1. 变更面逐包验证（单包 `go test <pkg>` 最可靠，本环境几乎必成）；
  2. `scripts/test-all.ps1`：逐包执行 + 每包最多 3 次重试（间隔 800ms 等锁释放）+ 状态文件续跑
     （`powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts\test-all.ps1`，多次调用自动续跑）；
  3. 全量门禁最终由 GitHub Actions CI 兜底。

### 4. PowerShell 脚本编码（.ps1 必须 UTF-8 BOM）

- **症状**：`powershell.exe -File xxx.ps1` 报 `Unexpected token '}'` 等解析错误（UTF-8 无 BOM 被按 GBK 读）；
  中文注释/字符串变乱码。
- **解法**：写 .ps1 后补 BOM：
  ```powershell
  $b = [System.IO.File]::ReadAllBytes($p); [System.IO.File]::WriteAllBytes($p, [byte[]](0xEF,0xBB,0xBF) + $b)
  ```
  仓库内 `scripts/test-all.ps1`、`scripts/smoke.ps1` 已带 BOM。
- **相关**：本会话 pwsh 的 `Get-Content` 对 UTF-8 无 BOM 文件按 GBK 显示乱码（wails log 等），
  读文件用 read 工具或显式 `-Encoding UTF8`。

### 5. 前台命令 600s 硬上限 / 后台长任务被终止

- **症状**：前台命令到 600s 必被杀（`timed out after 600000ms`）；后台任务长于数分钟可能被杀。
- **对策**：长任务（>5 分钟）拆成多段（状态落盘续跑），或验证可省略的部分（单包测试、vitest 等短任务不受影响）。
- 注：vitest 全量（约 3.5 分钟）前台可跑完——`cd frontend; npx vitest run` 243/243。

### 6. 其他小坑

- `npx`/`npm` 直接调用会撞执行策略（`npx.ps1 cannot be loaded`）：用 `& 'C:\Program Files\nodejs\npx.cmd'`。
- npm 缓存写 AppData 也可能被拒：`$env:npm_config_cache = 'C:\AI\wubigrok\.tmp\npmcache'`。
- 版本资源：Wails 用 `build/windows/info.json`（模板）从 wails.json 生成版本信息；
  `fixed` 段需显式写 `product_version`，否则 exe 的 ProductVersion 为 0.0.0.0（FileVersion 正常）。
  **v2.20.0 实测（2026-08-14）**：本机 `windres` 不可用（where.exe 找不到），
  Wails 会静默跳过 VERSIONINFO 资源——无论 `wails build` 还是 `wails build -s` 产出的 exe
  `[System.Diagnostics.FileVersionInfo]::GetVersionInfo(...)` 全部为空（FileVersion/ProductVersion 均空，
  v2.19.0 及更早同样如此，非本版回归）。运行不受影响，仅文件属性无版本信息。要恢复需安装
  MinGW-w64（含 windres）或改用 `github.com/akavel/rsrc` 手动生成 .syso 嵌入。
- 根目录 `versioninfo.rc` 是遗留物，不影响构建（版本来自 info.json 模板），但发布流程中应与
  wails.json 同步更新以免误导。

## 三、本会话环境相关命令速查

```powershell
go telemetry off                                # 关遥测（已持久）
$env:GOCACHE='C:\AI\wubigrok\.tmp\gocache'      # 兜底：缓存指工作区（一般不需要了）
$env:npm_config_cache='C:\AI\wubigrok\.tmp\npmcache'
cd frontend; & 'C:\Program Files\nodejs\npx.cmd' vitest run      # 前端全量 243/243
cd frontend; & 'C:\Program Files\nodejs\npx.cmd' tsc -b          # 类型检查
cd frontend; & 'C:\Program Files\nodejs\npx.cmd' eslint .        # lint（0 errors）
& 'C:\WINDOWS\System32\WindowsPowerShell\v1.0\powershell.exe' -NoProfile -ExecutionPolicy Bypass -File scripts\test-all.ps1   # Go 全量（逐包续跑）
cd frontend; npm run build && cd ..; wails build -s              # 发布构建
& ...powershell.exe -File scripts\smoke.ps1 -ExePath releases\gaea-v2.16.1.exe   # 冒烟
```
