# Gaea 图文二次改造桌面预览包

**包标识：** `gaea-wenxue-20261115-404679be`

**性质：** 功能提交预览包，不是正式版本号发布。`wails.json` 的
`productVersion` 保持 `4.543.0`，本包以提交号标识。

## 构建来源

- 功能提交：`404679be5eba3a655e35befda492cd786fc3873b`
  `fix(wenxue): sync generated git bindings`
- 父提交：`44f8bb6b7972c303ccddaac6dba05e7c5b66b699`
  `feat(wenxue): replace sin board with isolated literary workspace`
- 构建工作树：`C:\AI\wubigrok\.worktrees\pkg-404679be`
  （detached HEAD）
- 构建前 `git status --porcelain` 为空。Wails 会重写一批已跟踪的生成文件；
  构建后 `git diff --quiet` 退出码为 0，跟踪文件内容无差异。
- 前端依赖：junction 复用主工作树 `frontend\node_modules`，未引入主工作树
  的未提交源码。
- 构建链：先在 `frontend` 执行 `npm.cmd run build`，再在仓根执行
  `wails build -s -ldflags "-s -w" -trimpath`；`Skip Bindings=false`。
- 构建环境：Windows amd64、Wails `2.13.0`、Go `1.26.10`、Node `25.2.1`、
  npm `11.6.2`、UCRT64 GCC、`CGO_ENABLED=1`、
  `CC=C:\msys64\ucrt64\bin\gcc.exe`、系统默认 `TMP/TEMP`。
- 构建耗时：前端总门禁 `61.774s`（其中 Vite `22.58s`）；Wails
  `42.425s`（Wails 报告编译 `42.263s`）。

## 本包改动

1. 原“图文/原罪”板块替换为独立的“文学/图文”板块，所有小说工作空间限定在
   `C:\AI\tuwen`。顶部《新建小说》按名称创建独立目录并自动切换。
2. 左栏收敛为单一小说树：小说为父级、会话嵌套；每本小说右缘提供“新建会话”
   和“删除小说”，删除带不可恢复确认，后端仅允许删除 tuwen 根下的直接普通子目录。
3. 图文板块移除知识库、CAD 和进度计划的入口、标签、命令、工具菜单、专用面板、
   文件预览卡及跨板跳转；共享引擎意外返回的相关命令、工具卡和统计也会在图文界面过滤。
4. 办公板块保留原知识库、CAD 和进度计划能力。网页实测确认办公入口不受本次隔离影响。
5. 补齐 `WenxueB` 的 10 个 Git 生成包装器，确保干净提交构建时绑定不再产生源码漂移。

## 验证门禁

- 前端全量回归：**532 个测试文件，4901/4901 通过**，耗时 `310.14s`。
- 图文定向测试：**13/13 通过**，覆盖隔离、左栏、mock 契约。
- 前端生产构建：通过；`tsc -b` 与 Vite 均退出 0。
- ESLint 全量：0 error，61 个既有 warning。
- Go `go test ./internal/app -count=1`：通过，耗时 `129.558s`。
- Go `go vet ./internal/app` 与 `golangci-lint`：通过。
- 绑定漂移检查：`OK@782`。
- 网页实页验证：图文左栏无知识库；右栏仅
  `files/deliverables/browser/git/ui` 五个 Tab；命令面板无知识库/CAD/进度计划；
  办公板块仍保留对应入口。
- 正式包路径冒烟：
  `scripts\smoke.ps1 -ExePath ...\payload\gaea.exe`
  → `/api/health` **200**，响应 `{"status":"ok"}`，PID `36124` 已回收。
- 覆盖装机：`scripts\gaea_install.ps1 -SourceDir ...\payload -Launch`
  → `INSTALL OK`；四件套安装后 SHA256 与本包一致；桌面只保留
  `gaea.lnk`，目标为 `C:\AI\gaea-app\gaea.exe`。新 PID `41880`
  创建于 `2026-11-15 16:12:27`，晚于构建时间 `16:08:10`。

已知非本批阻塞项：

- 全量 `scripts/ci.ps1` 未全绿。失败集中在真实
  `.gaea_config.json.tmp` 写权限、本机 ORT DLL API 版本不匹配、realtime
  pipeline 同环境崩溃及 Bash guard 环境差异；本批受影响的完整
  `internal/app` 已单独通过。
- `npm run knip` 仍报告既有 unused files/exports（含 Wails 生成面、语音 worklet、
  schedule 导出等），未作为本包功能阻断项。
- Vite 构建保留既有 CSS 语法告警、动态/静态混用和 500kB chunk 告警，均未阻断构建。

## 产物与 SHA256

| 文件 | 大小 | SHA256 |
|---|---:|---|
| `payload/gaea.exe` | 65,374,720 B | `08017f551d18391f2b6f8904ae55fd7a4e156aab9878f2535f49c8d5b049c7ec` |
| `payload/onnxruntime.dll` | 19,288,888 B | `68d0d5f502be4509e2a16f932627a95137d748b2fae519027f80125c0b1f550b` |
| `payload/sherpa-onnx-c-api.dll` | 4,294,656 B | `5dd36dbd3572846b72c4a40a4017c5a770160ee8e00da5053bebc8b2632501e8` |
| `payload/sherpa-onnx-cxx-api.dll` | 264,192 B | `4c56327796cdb980dcbf90025c28093d89ca286088fb74c3b107f2f633e67ffe` |
| `gaea-404679be-preview.zip` | 33,336,524 B | `357a21e27d564f5cae46d990ec58605a11cb5cfd4f0488052af96550840fc31f` |
| `gaea-404679be-source.tar.gz` | 44,585,883 B | `d0bbbba1e8ac6610b79ef4afb736a52729756c1093c6f37a0a00130bc47e3d20` |

统一校验见同目录 `SHA256SUMS-404679be.txt`。

## 包内容与边界

- 预览 ZIP 根目录直接包含 `gaea.exe` 和三枚语音 DLL，解压后同目录运行。
- 源码 TAR 由 `git archive 404679be...` 生成，不包含本目录后写的包记录文件。
- exe、三枚 DLL、预览 ZIP 与源码 TAR 留盘但不进入 Git；
  `PACKAGE-INFO.md` 与 SHA256 清单进入 Git。
- 本包已覆盖安装至 `C:\AI\gaea-app`，并按构建指南使用桌面唯一
  `gaea.lnk` 启动校验；安装版 SHA256 与包内 payload 完全一致。
