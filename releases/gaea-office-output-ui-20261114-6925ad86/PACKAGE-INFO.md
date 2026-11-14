# Gaea 办公输出 UI 跨端对齐预览包

**包标识：** `gaea-office-output-ui-20261114-6925ad86`

**性质：** 功能提交预览，不是正式版本号发布。版本源 `wails.json` 保持
`4.543.0`，本包以提交号标识，不抬版本。

## 构建来源

- 提交：`6925ad865057f74a311ce1d1c705f239658431c7`
  `feat(office-ui): align composer baseline and inline file links`
- 父提交：`ea479333`
- 构建工作树：`C:\AI\wubigrok\.worktrees\pkg-6925ad86`
  （detached HEAD；构建前 `git status --porcelain` 为空）
- 前端依赖：junction 复用主工作树 `frontend\node_modules`，不引入主工作树源码改动。
- 构建链：先在 `frontend` 执行 `npm.cmd run build`，再在仓根执行
  `wails build -s -ldflags "-s -w" -trimpath`，bindings 未跳过。
- 构建环境：Wails `2.13.0`、UCRT64 GCC、`CGO_ENABLED=1`、
  `CC=C:\msys64\ucrt64\bin\gcc.exe`、系统默认 `TMP/TEMP`。
- 构建耗时：前端 Vite `31.15s`；Wails `1m00.5s`。

## 本包改动

1. 输入卡高度迁移到版本化键 `composerHeightV2`，新 UI 基线固定为 164px，
   覆盖 `112px` 输入区与 `43px` 工具栏。旧 `104px` 高度不再裁掉工具栏；
   用户在新 UI 内主动拖拽后仍会保存。
2. Markdown 行内代码恰好是本地文件路径时，升级为可点击的 `FileChip`。
   代码围栏保持纯代码，普通行内代码不受影响。
3. 补充输入卡旧高度迁移测试与行内文件路径点击测试。

## 验证门禁

- 干净工作树定向测试：`Markdown.test.tsx` +
  `ComposerI18n.test.tsx`，**35/35 passed**。
- 改动面 ESLint：0；`git diff --check`：0。
- 前端生产构建：通过（Vite `31.15s`）。
- Wails 桌面构建：通过（`1m00.5s`），bindings 未跳过。
- 正式包路径冒烟：
  `scripts\smoke.ps1 -ExePath releases\...\payload\gaea.exe`
  → `/api/health` **200**，响应 `{"status":"ok"}`，PID `35232` 已回收。
- 本包未覆盖 `C:\AI\gaea-app` 的当前安装版，保持功能预览包边界。

## 产物与 SHA256

| 文件 | 大小 | SHA256 |
|---|---:|---|
| `payload/gaea.exe` | 63,878,144 B | `baefc5d355a39063741cf2ba37c1112307f0c36d828864881ec7575ac898afd4` |
| `payload/onnxruntime.dll` | 19,288,888 B | `68d0d5f502be4509e2a16f932627a95137d748b2fae519027f80125c0b1f550b` |
| `payload/sherpa-onnx-c-api.dll` | 4,294,656 B | `5dd36dbd3572846b72c4a40a4017c5a770160ee8e00da5053bebc8b2632501e8` |
| `payload/sherpa-onnx-cxx-api.dll` | 264,192 B | `4c56327796cdb980dcbf90025c28093d89ca286088fb74c3b107f2f633e67ffe` |
| `gaea-6925ad86-preview.zip` | 32,942,469 B | `8de5c3f7a1bf389a29e59b32d107c19a1478dc255d3f2fda82c854c806b578a7` |
| `gaea-6925ad86-source.tar.gz` | 42,925,679 B | `8ba1abee7740dee40d26284be65573bfca846fd34532314ffc64b086e29b8e88` |

统一校验见同目录 `SHA256SUMS-6925ad86.txt`。exe、三枚 DLL、预览 ZIP
与源码 TAR 只留盘，不进入 Git；`PACKAGE-INFO.md` 与校验清单入库。

## 边界

- 本包只包含 `6925ad86` 及其历史提交；主工作树中的进度计划、上下文、
  编队、众神殿等其他未提交改动**未进入本包**。
- 预览 ZIP 根目录固定为 `gaea.exe` 与三枚语音 DLL，可直接解压运行。
- 源码 TAR 由 `git archive 6925ad86` 生成，不包含本目录后写的包记录文件。
