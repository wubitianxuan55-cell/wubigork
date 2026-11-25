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
8. 干净树打包/装机收尾时，必须把包版四件（exe + 三 DLL）**回写主树
   `build/bin`**（哈希对账装机位一致），保持 `build/bin` 恒等最近装机版——
   它是测试 DLL wrapper 与 `go test`/`wails build` PATH 前置的引用源、
   `gaea_install.ps1` 的默认 `-SourceDir`。主树压着未提交改动时用包版文件
   回写，**不得**改在主树重建（会把未提交改动混编进去）。

若 `build.bat` 触发 Wails runtime 的 `Access is denied`，按构建指南清理孤儿进程、
恢复系统 `TMP/TEMP` 后执行上述裸 `wails build`，不要跳过 bindings。

## 蒸馏与逆向数据落盘（强制默认，2026-11-20 用户拍板）

**所有蒸馏、逆向、上游取证的数据文件一律放 `C:\AI\re` 下，仓库只放文档与代码。**

1. 一个上游目标一个子目录：`C:\AI\re\<目标>-<版本>`（有版本带版本，如 `codex-0.165.0`；
   无版本用产品名/批次，如 `lingxi`、`agorai`、`aidebate`、`codex-computer-use`）。
   **上游参考克隆沿用上游项目名并保留其 `.git`**（`orca`、`zcode`、`projectlibre`…），
   依赖快照用 `<项目>-<latest|baseline>`。
2. 子目录内分层固定：`docs/`（逆向说明）、`evidence/`（原始证据）、`embedded/`（解出的资产）、
   `scripts/`（可复跑脚本）、`work/`（可重生成输出）、`oss/`（上游源码只读快照）、`data/`（成品证据包）。
   老数据集与克隆树保持原布局，并在 `C:\AI\re\README.md` 登记差异。
3. **禁止**在仓库内新建数据目录；已迁走的 `re-lingxi/`、`clones/` 与 `C:\AI\_distill\` 不得复活
   （2026-11-20 两批迁移已全部并入 `C:\AI\re`，迁移表见其 `README.md` §六）。
4. 仓库文档引用数据时写 **`C:\AI\re\<数据集>\...` 绝对路径**；大件（安装包、源码 tarball、
   二进制副本）只记路径与 SHA256，不复制。
5. 新增/迁移数据集：同步改 `C:\AI\re\README.md` 目录总览，并 `grep -rn` 改掉仓库内所有
   旧路径引用，最后跑 `node scripts/check-docs.mjs` 确认无悬空引用。

## 项目长期记忆

- 需要项目约定、历史决策、当前进度时，读取 `.gaea/AGENTS.md`、
  `.gaea/progress.md` 与 `.gaea/todos.md`。
- 构建方式的完整权威说明始终是 `docs/gaea-build-guide.md`；若实现变化，
  同步更新构建指南以及 `.gaea/AGENTS.md` 的构建入口，禁止让两处漂移。
