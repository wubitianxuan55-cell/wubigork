# gaea 仓库代理规约

## 构建、装机与打包（强制默认）

遇到构建、装机、发布包、源码包或产物核验任务时，必须先读取并遵守
`docs/gaea-build-guide.md`。这是默认纪律，不得等待用户再次提醒。

最低要求：

1. 构建前检查并处理 `gaea.exe`、`wails.exe`、`go.exe` 孤儿进程，避免锁住
   `build/bin` 或 Wails runtime。
2. 先在 `frontend` 执行 `npm.cmd run build`；Wails 构建不得跳过 bindings。
3. 使用系统默认 `TMP/TEMP`，设置 `CGO_ENABLED=1`、`CC` 指向
   `C:\msys64\ucrt64\bin\gcc.exe`，并把 `C:\msys64\ucrt64\bin` 放到 `PATH` 最前。
4. 仓根执行 `wails build -s -ldflags "-s -w" -trimpath`。不得使用无 ucrt64
   的构建，否则 cgo 可能静默降级成语音空壳。
5. `gaea.exe` 必须与 `onnxruntime.dll`、`sherpa-onnx-c-api.dll`、
   `sherpa-onnx-cxx-api.dll` 同目录。
6. 在正式路径运行 `scripts\smoke.ps1 -ExePath build\bin\gaea.exe`，
   `/api/health` 必须返回 200。不要从被本机策略拦截的 `.tmp` 路径运行 exe。
7. 打包某次提交时必须从该提交的独立干净工作树构建，不能把主工作树未提交的
   其他业务改动混入包；记录提交号、exe/DLL/ZIP/源码包 SHA256 和冒烟结果。

若 `build.bat` 触发 Wails runtime 的 `Access is denied`，按构建指南清理孤儿进程、
恢复系统 `TMP/TEMP` 后执行上述裸 `wails build`，不要跳过 bindings。

## 项目长期记忆

- 需要项目约定、历史决策、当前进度时，读取 `.gaea/AGENTS.md`、
  `.gaea/progress.md` 与 `.gaea/todos.md`。
- 构建方式的完整权威说明始终是 `docs/gaea-build-guide.md`；若实现变化，
  同步更新构建指南以及 `.gaea/AGENTS.md` 的构建入口，禁止让两处漂移。
