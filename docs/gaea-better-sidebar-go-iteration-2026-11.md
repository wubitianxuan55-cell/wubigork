# gaea × DSH-better-sidebar：Go 增量复刻长期迭代与优化交接

> 状态：**活跃专项 / 唯一交接入口**（2026-11-12 建立）。
>
> 目标：在现有 gaea 上增量复刻 `DSH-better-sidebar` 中适合 gaea 的能力。
>
> 硬边界：**不重写 gaea**；保留既有会话、任务、Office、记忆、导航和整体 UI 架构；**侧聊服务明确不做**。
>
> 上游基线：`dsh-better-sidebar v0.32.0`，解析后 commit `53e08cec61f1a3cfe32b86e6429876fff1ef0504`（2026-11-12）。
>
> 当前批次：**G2（只读提交图）在分支 `feat/g2-commit-graph` 完成验证，待并入**；G1.6/G1.7/G1.8 已全部并入 main（`d048f3c4`），G1 全验收矩阵（非 Git 仓库/空仓/无提交 HEAD/重命名/二进制/冲突）真仓测试+真机走查双实证。F1+F2+F3+G1.1–G1.5 已提交为 `f49ea150` 并完成打包与冒烟。
> 当前仍**未 push、未覆盖正式安装版**；W1/X1 维持产品门。

本文替代旧的 2026-09 滚动规划，作为后续新会话的恢复入口。旧文档
`docs/gaea-dsh-better-sidebar-long-term-distill-plan-2026.md` 只保留历史调研和早期阶段台账，
不再据此判断当前完成度。

## 0. 新会话先做什么

### 0.1 必读顺序

| 顺序 | 文件 | 用途 |
|---|---|---|
| 1 | `.gaea/AGENTS.md` | 项目级强制约定、构建纪律和当前专项入口 |
| 2 | 本文 | 本专项目标、现状、路线、风险和交接协议 |
| 3 | `.gaea/progress.md` | 最近实际执行记录 |
| 4 | `.gaea/todos.md` | 尚未完成和等待拍板的事项 |
| 5 | `docs/gaea-build-guide.md` | 只有做构建、装机、打包、产物核验时才读，但必须读后执行 |

### 0.2 开工检查

新会话不要直接假设代码已提交，也不要从旧文章推断缺口。先执行：

```powershell
git status --short --branch
git log -5 --oneline --decorate
```

然后确认：

1. 当前功能基线已提交为 `f49ea150`；先确认 HEAD、分支和未提交改动，不 reset、不 checkout、不 stash。
2. 用户是否已指定下一批；未指定时按本文 §6 的顺序推进，不自行重开已拒绝项。
3. 是否只做文档、代码、构建或装机。构建与装机必须另走 `docs/gaea-build-guide.md`。
4. 主工作树若仍有平行测试优化改动（`.gitignore`、`frontend/package.json`、
   `frontend/src/test/setup.ts`、`frontend/vite.config.ts` 等），不得混入本专项提交或发布包。

### 0.3 事实优先级

发生冲突时按以下顺序判断：

1. 用户当前明确指令。
2. 当前源码、测试和 Git 工作树事实。
3. 本文与 `.gaea/progress.md`。
4. 上游源码、CHANGELOG 和设计文档。
5. 上游 README。
6. 旧规划与归档文档。

用户已确认过 README 不是完整能力事实源。回源时必须同时读
`src/`、`CHANGELOG.md`、`docs/plans/`；README 只用于安装说明和概览。

## 1. 目标、边界与产品原则

### 1.1 最终目标

把 better-sidebar 中经过验证的工作台能力重新设计为 gaea 的原生能力：

- Go 是文件系统、Git、上传、监听、并发与安全语义的权威层。
- React/TypeScript 负责树、编辑器、标签页、预览和交互反馈。
- Wails binding 是唯一桌面桥；前端通过现有 bridge、mock、spaceBindings 和生成绑定接线。
- 每批功能都能独立验收、独立回滚并进入长期维护。

### 1.2 明确不做

| 项 | 决策 |
|---|---|
| 从头重写 gaea | **不做** |
| 把 better-sidebar 的 Node/Cordis 宿主搬进 gaea | **不做** |
| 另起一套聊天、会话、任务、Office 或导航系统 | **不做** |
| 侧聊服务 / 侧边对话 | **不做**；除非用户未来明确翻案 |
| 照搬上游无工作区围栏的 fs 路由 | **不做**；gaea 保持更严格围栏 |
| 为了“像素一致”牺牲现有 Office 编辑和证据链 | **不做** |

### 1.3 借鉴方式

- 上游许可为 MIT，但仍按“机制重推导”处理：学习交互形状、状态机、边界与验收方法，不复制整套实现。
- 优先复用 gaea 已有组件、事件、store、diff、预览和任务能力。
- 新前端能力必须同时处理真机、旧后端回退、浏览器 mock 和测试契约。
- 大文件字节不经过无界 JS 桥；能走原生选择器就优先走原生选择器。

## 2. 当前事实基线

### 2.1 Git 状态

截至 2026-11-12 打包收口时：

- 当前分支：`main`，跟踪 `origin/main`。
- 功能提交：`f49ea1500026526b822c80be39cfb08d06026869`
  （`feat-office-dsh-better-sidebar-workbench`），父提交
  `daaa00654d1db7b1e0676bace326c878a1f83cae`。
- 功能提交共 80 个文件，`7,786 insertions / 374 deletions`。
- 本收口文档与包记录作为功能提交之后的独立文档提交，不与源码功能混写。
- 尚未 push；构建时 `main` 相对 `origin/main` ahead 1，包元数据提交后 ahead 2。
- 未覆盖 `C:\AI\gaea-app` 的正式安装版。
- 主工作树仍有不属于本专项的平行测试优化改动和未跟踪测试加速件；它们未进入
  `f49ea150`、独立构建树或发布包。既有 `releases/**` 未跟踪压缩包属于用户产物，不得清理。
- 本包记录位于 `releases/gaea-bettersidebar-20261112-f49ea150/`；ZIP/TAR 仅留磁盘，
  `PACKAGE-INFO.md` 与 `SHA256SUMS-f49ea150.txt` 入库。

### 2.2 上游基线

- 产品：`omdsh-dev/DSH-better-sidebar`
- 版本：`v0.32.0`
- 注解 tag 对象：`a53a98808f9d2e960db98b1df6cec0e358121b6c`
- 解析 commit：`53e08cec61f1a3cfe32b86e6429876fff1ef0504`
- 提交日期：2026-11-12
- 上游宿主基线：DSH `0.2.2-rc.1+`

从旧规划的 v0.18.0 到当前 v0.32.0，文件工作台新增或强化了：

- 批量列目录与只监听展开目录；
- Git 状态着色与目录汇总；
- 多选、Shift 范围选择、批量删除/ZIP 下载；
- 新建文件/文件夹、重命名、删除；
- 上传文件/文件夹、拖放上传与打包进度；
- 排除规则、目录链折叠、文件图标、双来源“打开方式”；
- 编辑器外部修改检测、EOL/编码保持、草稿保护；
- 5MB 单次读取边界、超大文件分页、JSONL 记录卡；
- Agent 会话文件变动、Git staged/unstaged、统一文件变动页和只读提交图。

上述只是本轮回源得到的能力地图，不代表 gaea 已完成。

### 2.3 当前质量与交付快照

功能提交前的 F1 验证记录：

- `frontend npm run build`：通过。
- 前端全量：**483 个测试文件 / 4301 个用例通过**。
- 定向 ESLint：通过。
- `go vet ./internal/gaea/workbench ./internal/app`：通过。
- Go 全量（排除语音环境相关两包）：通过。
- Wails 构建：成功。
- 正式路径冒烟：`/api/health` 200。
- `gofmt -l internal/gaea/workbench internal/app/gaea_workbench.go internal/app/gaea_workbench_test.go`：无输出。
- `git diff --check`：无空白错误，仅有 Windows 行尾转换 warning。

正式可交付构建来自仓内独立干净工作树
`.worktrees/pkg-f49ea150`，检出提交精确为 `f49ea1500026526b822c80be39cfb08d06026869`，
构建前 `git status --porcelain` 为空：

- 前端生产构建：`npm.cmd run build` 通过，Vite `20.38s`。
- Wails：`v2.13.0`，Go `1.26.10 windows/amd64`，ucrt64 GCC `16.1.0`，
  `CGO_ENABLED=1`、系统默认 TMP/TEMP；完整构建 `50.158s`，bindings 未跳过。
- exe：`gaea.exe`，`62,486,528` bytes，SHA256
  `986C89666AB1588D10E0881AAEBA8129B35EC62577034FA88C1A2155F419EE96`。
- 三只 DLL 由 `build/voice-dlls` 派发到 exe 同目录；哈希见包内
  `SHA256SUMS-f49ea150.txt`。
- 冒烟：同一四件套放入
  `releases/gaea-bettersidebar-20261112-f49ea150/payload/` 后，以绝对正式路径执行
  `scripts\smoke.ps1`；`/api/health` 200，`{"status":"ok"}`，PID 30100 已回收。

本次包产物：

| 文件 | 大小（bytes） | SHA256 |
|---|---:|---|
| `gaea-f49ea150-preview.zip` | 32,502,355 | `7038DDBE20F3B5A6821BE9D7140BF69257E9037940F3E6EAEB6C7D1E0E2A172A` |
| `gaea-f49ea150-source.tar.gz` | 39,080,619 | `776686C69D939C4DB8720531705C0CFEFCB6D62E6F5845B7222AE7F411C236FC` |

预览 ZIP 根目录只有 `gaea.exe` 与三只 DLL；源码包由
`git archive --format=tar.gz --prefix=gaea-f49ea150/ ... f49ea150` 生成，内容严格来自提交。
Wails 生成器仍打印既有 `Not found: time.Time`，并在构建后重写生成绑定；这属于已知生成物噪声，
本次 build 与 smoke 均成功。本机直接从 `.worktrees` 路径启动 exe 会被系统策略拦截，
因此包构建必须把同一 payload 放到 releases 正式路径后再冒烟。

F2.1 增量验证（2026-11-12，未再跑全量构建）：

- `gitStatusInk.test.ts` + `FileTree.test.tsx`：**2 文件 / 23 用例通过**。
- 相关 4 个文件的定向 ESLint：通过。
- 覆盖 Git X/Y 分类、冲突优先级、Windows 路径归一、未跟踪目录折叠、文件徽标与目录汇总点。

F2 全项增量验证（2026-11-12，未跑 Vite/Wails 构建）：

- Go：`filefilter`、`workbench` 定向测试通过；`internal/app` 的搜索/资料/工作台定向测试通过；三包 `go vet` 通过。
- 前端：FileTree、Composer、文件树偏好/紧凑/图标/Git 映射与 mock 契约共 **8 文件 / 53 用例通过**。
- `tsc -b --force`：通过。
- 相关前端文件定向 ESLint：通过。
- 绑定生成：`gen_bindings` 与 `wails generate module` 成功，方法数仍为 745。

F3 增量验证（2026-11-12，未跑 Vite/Wails 构建）：

- 前端 12 个相关测试文件 / **99 用例通过**。
- `tsc -b --force`：通过。
- 相关 23 个文件定向 ESLint：通过。
- 覆盖 JSONL 记录卡、Markdown 目录、相对/file/data 图片解析、超长 diff 行、脏稿关闭确认。

G1.1 增量验证（2026-11-12，未跑 Vite/Wails 构建）：

- `gitStatusStore`、GitPanel、FileTree 定向测试：**3 文件 / 29 用例通过**。
- `tsc -b --force`：通过。
- 6 个相关文件定向 ESLint：通过。
- 两组件同时订阅时只共享一个 in-flight `GaeaGitStatus`；Git 面板已增加“Git / 本轮文件”镜头。

G1.2 增量验证（2026-11-12）：

- Go App Git 定向测试：通过。
- `go vet ./internal/app`：通过。
- 覆盖父级仓库根发现、嵌套工作区相对路径归一、仓库外变更过滤、porcelain `-z` rename 目的路径解析、无提交分支名。

G1.3 增量验证（2026-11-12）：

- 统一树纯函数、树组件与 GitPanel 定向测试：**3 文件 / 12 用例通过**。
- `tsc -b --force`：通过。
- 2 个实现文件与 4 个相关测试/组件定向 ESLint：通过。

G1.4 增量验证（2026-11-12）：

- 统一树、统一树组件与 GitPanel 定向测试：**3 文件 / 14 用例通过**。
- `tsc -b --force`：通过。
- 相关实现/测试定向 ESLint：通过。

G1.5 增量验证（2026-11-12）：

- 统一树、统一树组件与 GitPanel 定向测试：**3 文件 / 16 用例通过**。
- `tsc -b --force`：通过。
- 统一树/GitPanel 实现与测试定向 ESLint：通过。

## 3. 当前架构与代码落点

```text
React FileTree / FilePreview / paneTabs
                │
                ▼
gaea/lib/bridge + spaceBindings + generated Wails bindings
                │
                ▼
internal/app/gaea_workbench.go（门面、原生对话框、事件）
                │
                ▼
internal/gaea/workbench（Go 权威）
  fs.go     list / create / rename / delete / export
  text.go   read / page / encoding / EOL / CAS write
  upload.go chunk upload / native import / rollback
  watch.go  expanded-dir fsnotify / debounce
```

### 3.1 后端文件

| 文件 | 职责 |
|---|---|
| `internal/gaea/workbench/fs.go` | 批量目录、符号链接分类、创建、重命名、永久删除、单文件导出、目录/多选 ZIP |
| `internal/gaea/workbench/text.go` | 5MB 可编辑读取、256KB 分页、编码/EOL 检测、mtime 乐观并发、原子写 |
| `internal/gaea/workbench/upload.go` | 3MB 分片上传、2GB 总预算、原生文件/文件夹导入、拒绝静默覆盖、失败回滚 |
| `internal/gaea/workbench/watch.go` | 精确监听展开目录、150ms 去抖、按受影响层级推送 |
| `internal/gaea/workbench/workbench_test.go` | 文件操作、路径围栏、编码/EOL、冲突、上传、导入、监听、ZIP 回归 |
| `internal/gaea/filefilter/exclude.go` | 默认噪音、用户 glob、路径锚定和 ** 通配；树、搜索、watcher 共用 |
| `internal/app/gaea_workbench.go` | Wails 门面、上传管理器/监听生命周期、原生选择器、保存对话框 |
| `internal/app/gaea_workbench_test.go` | 门面级 CRUD、分块上传、文本冲突与编码回归 |

### 3.2 前端文件

| 文件 | 本批变化 |
|---|---|
| `frontend/src/gaea/components/FileTree.tsx` | 批量列目录、展开目录实时刷新、多选/Shift、创建、重命名、永久删除、上传、ZIP、拖放、旧后端回退 |
| `frontend/src/gaea/components/ExplorerView.tsx` | 改名/删除向 pane tab 传播 |
| `frontend/src/gaea/lib/paneTabs.ts` | 文件名或目录子树改名时重定向已开 tab；删除时关闭对应子树 |
| `frontend/src/gaea/components/FilePreview.tsx` | 截断正文先完整读取再编辑、mtime 写回、5MB 保存、超大文本分页追加 |
| `frontend/src/gaea/lib/fileTreePrefs.ts` | 按工作区持久化用户排除规则与紧凑目录偏好 |
| `frontend/src/gaea/lib/fileTreeCompact.ts` | 单子目录链解析、深度封顶和缺失级预取前沿 |
| `frontend/src/gaea/lib/fileIcon.tsx` | 办公/代码/配置/压缩/图片/数据库文件类型图标 |
| `frontend/src/gaea/lib/jsonlRecords.ts` | JSONL 逐行解析、坏行保留、500 条/4000 字预览预算 |
| `frontend/src/gaea/components/JsonlPreview.tsx` | 记录卡/原文切换与无效行可见降级 |
| `frontend/src/gaea/lib/markdownOutline.ts` / `MarkdownToc.tsx` | Markdown 标题提取、浮动目录、滚动高亮与点击定位 |
| `frontend/src/gaea/lib/markdownImage.ts` | 相对路径、Windows drive、file URL 到工作区媒体路径解析 |
| `frontend/src/gaea/lib/editorDrafts.ts` | 多预览 owner 脏稿登记、关闭/导航确认与 beforeunload |
| `frontend/src/gaea/lib/gitStatusStore.ts` | Git 状态单一轮询源、in-flight 去重、focus/30s 刷新与 revision |
| `frontend/src/gaea/lib/unifiedChangesTree.ts` | Git 与 Agent 变更按路径合并、工作区路径归一、目录 rollup |
| `frontend/src/gaea/components/UnifiedChangesTree.tsx` | 目录树、Git 暂存/工作区徽标、Agent 次数徽标与文件打开 |
| `internal/app/gaea_git.go` | 父级仓库根发现、嵌套工作区状态归一/过滤、NUL porcelain 解析与硬化 Git 执行 |
| `frontend/src/gaea/lib/types/files.ts` | 工作台 Level/Entry/Mutation/Upload/Text/Page/Write 类型 |
| `frontend/src/gaea/lib/bridge/*`、`spaceBindings.ts` | 绑定注册、work 空间归属和运行时映射 |
| `frontend/src/gaea/lib/mock/*` | 浏览器 mock、签名与测试回退 |
| `frontend/wailsjs/go/app/OfficeB.*`、`models.ts` | Wails 生成绑定与模型 |

F1 新增 17 个 Wails 方法，绑定面由 728 变为 **745**。

### 3.3 F1 绑定清单

| 方法 | 作用 |
|---|---|
| `GaeaListDirBatch(paths)` | 一次请求批量列多个展开层 |
| `GaeaFileTreeWatch(paths)` | 用当前展开目录集合重建 watcher |
| `GaeaCreateWorkspaceFile(parent, name)` | 新建空文件 |
| `GaeaCreateWorkspaceFolder(parent, name)` | 新建目录 |
| `GaeaRenameWorkspacePath(rel, name)` | 同目录内重命名 |
| `GaeaDeleteWorkspacePaths(paths)` | 永久删除文件或目录 |
| `GaeaWorkspaceUploadBegin(dir, relativePath)` | 开始分片上传 |
| `GaeaWorkspaceUploadAppend(token, chunk)` | 追加一个可独立 base64 解码的分片 |
| `GaeaWorkspaceUploadFinish(token)` | 原子发布上传结果 |
| `GaeaWorkspaceUploadAbort(token)` | 取消并清理上传 |
| `GaeaPickWorkspaceUploadFiles(dir)` | 原生多文件选择并直接复制进工作区 |
| `GaeaPickWorkspaceUploadFolder(dir)` | 原生目录选择并递归复制进工作区 |
| `GaeaReadTextFile(rel)` | 返回完整可编辑正文、mtime、编码、EOL |
| `GaeaReadFilePage(rel, offset)` | 返回行对齐的分页正文 |
| `GaeaWriteWorkspaceFile(rel, content, expectedMtimeMs)` | CAS 保存并保持编码/EOL |
| `GaeaSaveWorkspacePathAs(rel)` | 文件另存；目录导出 ZIP |
| `GaeaSaveWorkspacePathsAs(paths)` | 多选导出 ZIP |

## 4. F1 已交付能力

### 4.1 文件树

- 展开目录层优先通过 `GaeaListDirBatch` 一次取回；旧后端或浏览器 mock 自动回退逐层读取。
- watcher 只监听当前可见/展开目录，不递归监听整棵工作区。
- 文件系统事件 150ms 去抖后只失效受影响层，再批量重列。
- 支持 Ctrl/Cmd 多选与 Shift 范围选择。
- 支持新建文件、新建文件夹、重命名和永久删除。
- 支持原生上传文件、原生上传文件夹和拖放上传。
- 支持单文件另存、目录 ZIP、多选 ZIP。
- 文件或目录改名后同步已打开 pane tab；删除后关闭对应文件 tab。
- 符号链接按目标分类展示；断链明确标记；遍历和写入不跟随符号链接逃出工作区。

### 4.2 文本编辑与预览

- 普通文本和 Markdown 可在预览内编辑。
- 原预览若被截断，进入编辑前会重新读取完整文本，避免把截断内容写回磁盘。
- 保存时携带打开时的 mtime；磁盘被外部改动则明确拒绝覆盖。
- 保持原文件的 UTF-8/BOM、UTF-16、GB18030 与 LF/CRLF/CR 语义。
- 5MB 内可编辑；超过上限进入 256KB 分页只读阅读。
- 保存使用原子写，避免半写文件。

### 4.3 上传导入

- 原生选择器路径由 Go 直接读取源文件，字节不经过 JS bridge。
- 拖放路径使用分片上传；前端每片 3MB，桥上不出现整文件大字符串。
- 上传总预算 2GB，同时最多 16 个会话，防止失控占盘/内存。
- 目标已存在时拒绝静默覆盖。
- 文件夹导入跳过符号链接，避免目录环和围栏外复制。
- 多文件复制中途失败时回滚本批已复制文件。

### 4.4 安全与一致性不变量

以下规则是后续所有批次的红线：

1. 所有输入路径先做工作区词法围栏，再验证父目录真实路径仍在工作区内。
2. 删除符号链接只删链接本身，不删目标。
3. 默认不递归跟随符号链接。
4. 不允许静默覆盖；冲突必须 fail-loud 或先由用户明确确认。
5. 文本保存必须带并发基线，不允许把外部改动无提示覆盖。
6. 大文件编辑有明确预算；超预算只读分页，不伪装可编辑。
7. 上传会话有总量、并发数和空闲 TTL 上限。
8. 监听集合必须随展开目录变化重建，并在退出时关闭。
9. 操作失败必须给用户可见错误，不能呈现假成功或假空目录。

## 5. 上游能力和 gaea 差距

状态说明：`✅` 已落地；`🟡` 有底座但不完整；`⬜` 未做；`⛔` 明确不做。

| 上游能力 | gaea 当前 | 后续批次 |
|---|---|---|
| 懒加载文件树、展开目录监听、批量列目录 | ✅ F1 | 维持 |
| 文件/目录新建、重命名、删除、ZIP | ✅ F1 | 维持 |
| 多选、Shift 范围选择、拖放/原生上传 | ✅ F1 | 维持 |
| 符号链接分类与断链标记 | ✅ F1，比上游更严格 | 维持 |
| Git 状态着色、目录变更汇总 | ✅ F2.1 代码候选 | 真机走查 |
| 排除规则（默认 + 用户 glob） | ✅ F2.2 代码候选 | 真机走查 |
| 单子目录链折叠 | ✅ F2.3 代码候选 | 真机走查 |
| 文件类型图标主题 | ✅ F2.4 代码候选 | 真机走查 |
| 双来源“打开方式” | ✅ F2.5 代码候选（系统默认程序 + 资源管理器） | 真机走查 |
| 编辑器语法高亮、保存、外部修改拦截 | ✅ CodeMirror 6 懒加载，六类语言映射+textarea 双层回落；mtime CAS | 维持 |
| 超大文件分页 | ✅ F1 | 维持 |
| EOL/编码保持 | ✅ F1 | 维持 |
| JSONL 记录卡和坏行降级 | ✅ F3 | 真机走查 |
| Markdown 浮动目录、本地图片 `data:`/`file:` 兼容 | ✅ F3 | 真机走查 |
| 长 diff 单行预算 | ✅ 2K 默认 / 20K 展开 | 维持 |
| 脏稿关闭/切页保护 | ✅ F3 关闭与导航确认 + beforeunload | 真机走查 |
| 会话内文件读写变动树 | 🟡 ChangesPanel 已有三态，尚未与文件页统一 | G1 |
| Git status/stage/unstage/discard/commit/log | 🟡 7 个最小绑定；已收前端单一源并补仓库根/嵌套工作区语义 | worktree/服务深化 |
| 统一 Git / 本轮文件变动页 | ✅ G1.5 统一树 + 来源筛选 + stage/unstage/discard + 行内 diff | worktree/服务深化 |
| Git 只读提交图 | ⬜ | G2，按价值决定 |
| 后台任务、子代理、强制终止 | ✅ gaea 已有并经历过蒸馏 | 只收口 |
| 底部双工作台、自由分栏 | ⬜ | W1，用户拍板 |
| 真实终端 | ⬜ | W1，用户拍板 |
| 插件注册服务和开始页接管 | ⬜ | X1，与 gaea 非插件架构不直接兼容 |
| 侧聊服务 | ⛔ | 不做 |

### 5.1 有意差异

| 项 | 上游 | gaea |
|---|---|---|
| 工作区围栏 | v0.23 起 fs 路由不再做包含性检查 | 保持严格工作区围栏 |
| 上传大小 | 文档口径为无固定上限的流式传输 | F1 采用 2GB 总预算 |
| 单次编辑 | 由编辑器和大文件策略决定 | 5MB 可编辑，超出分页只读 |
| 宿主 | DSH/Cordis 插件环境 | Wails 原生 Go + React |
| 侧聊 | beta 能力 | 用户明确不做 |

这些差异不是“欠蒸馏”，而是 gaea 的产品和安全选择。只有用户明确要求时才改。

## 6. 长期路线与批次验收

### F0：F1 收口与可恢复交付

优先级：最高。

当前状态：**✅ 2026-11-12 收口完成**。功能提交、独立干净工作树构建、正式路径冒烟、
预览包、源码包、哈希记录和长期记忆均已落地；仍待用户决定是否 push 以及是否覆盖正式安装版。

目标：

- 确认 F1 源码、绑定、mock 和测试一致。
- 最终重新执行前端构建与 Wails 构建，让 `models.ts` 由生成器重写，消除手工清理造成的大 diff。
- 以正式路径跑冒烟并记录最终 exe 哈希。
- 更新本文、`docs/README.md`、`.gaea/AGENTS.md`、`.gaea/progress.md`、`.gaea/todos.md`。
- 功能提交已由用户确认执行；push 与装机仍由用户另行决定。

退出标准：

- `gofmt`、`git diff --check` 干净；
- 前端定向测试与全量测试通过；
- Go 相关包与可执行环境下全量测试通过；
- Wails 构建和 `/api/health` 200；
- 绑定数、mock 签名、space 归属一致；
- 文档不再声称未完成能力已交付。

### F2：文件树完整度

优先级：高；风险低到中。

建议顺序：

1. **Git 状态着色（F2.1 ✅ 代码候选，2026-11-12）**：复用现有 `GaeaGitStatus`，文件行显示 X/Y 徽标和语义色，目录/根显示后代变更汇总点；只在挂载、30s、窗口聚焦、目录事件和本树写操作后刷新。纯函数 `lib/gitStatusInk.ts` 负责分类、路径归一、冲突优先级和目录聚合。剩余：桌面真机观感与高变更量性能走查。
2. **排除规则（F2.2 ✅ 代码候选）**：`internal/gaea/filefilter` 单源编译默认噪音与最多 64 条用户 glob；文件树、@ 文件名搜索和 watcher 共用。按工作区持久化，支持名称、相对路径、`*`、`?`、`**`；根行和标题栏齿轮可恢复。
3. **单子目录链折叠（F2.3 ✅ 代码候选）**：Go 增量探测“恰好一个可见非链接子目录”，前端合并为 `a/b/c` 单行，深度封顶 8；展开/收起整链，菜单操作尾目录，tab 仍用真实路径调解。
4. **文件类型图标（F2.4 ✅ 代码候选）**：Word/表格/PPT/PDF/图片/代码/配置/压缩/数据库独立图标，文件夹区分开合；复用 gaea 现有图标库。
5. **打开方式（F2.5 ✅ 代码候选）**：文件菜单“打开方式”子菜单提供系统默认程序与资源管理器定位；失败显示 toast，不再静默。

验收：

- 非 Git 仓库不轮询、不报错、不显示假状态；
- Git 状态变化后树与文件行在明确刷新窗口内更新；
- 排除规则对树、搜索、watcher、ZIP 行为有测试；
- 目录链折叠不破坏 reveal、rename、delete、open tab 调解；
- 对外部程序启动有失败提示且无黑窗。

### F3：预览与编辑器对齐

优先级：高；**代码候选已完成（2026-11-12）**。

范围：

1. CodeMirror 6 已覆盖当前文本/Markdown 编辑入口，Suspense 加载态和 ErrorBoundary 都回落 textarea；额外语言包不在本刀。
2. 脏稿登记按规范化路径支持多 owner；pane tab 关闭/切换与主区预览关闭/导航先确认，页面刷新/关闭有 `beforeunload`。
3. JSONL/NDJSON 逐行记录卡，合法行摘要可展开，坏行原样保留；上限 500 条、单条预览 4000 字符，可切原文。
4. Markdown 目录只提取代码围栏外的 ATX 标题，浮动吸顶、滚动高亮、点击定位；少于两个标题不占位。
5. 本地图片将相对/file URL 解析到当前 Markdown 目录并经 `AttachmentDataURL`（32MB 上限+可读根围栏）加载；`data:image`/远程图直接渲染。
6. 长 diff 单行默认只挂 2K 文本，可显式展开到 20K；超过 20K 的配对行不做字符 LCS，避免主线程卡死。

验收：

- 脏草稿不会因切 tab、关闭预览或页面刷新而无提示丢失；永久删除仍以用户删除确认为最终意图；
- JSONL 坏行不拖垮整个文件；
- 本地图片不突破工作区媒体围栏；
- 2MB 单行和高记录数 JSONL 有性能回归；
- 所有预览入口共享同一安全判定，不出现一份放宽、一份收紧。

### G1：统一文件变动与 Git 工作台

优先级：中高；**G1.1–G1.6 已完成共享状态、仓库根语义、统一树、快操作、行内 diff、worktree 只读展示与子进程预算收口**。

当前 `GaeaGit*` 最小集只能覆盖 status/diff/stage/unstage/discard/commit/log。G1.1 已完成：

- `gitStatusStore` 是全应用唯一 Git 状态轮询者：30s、窗口聚焦、订阅去重、in-flight 合并、非仓库错误快照；
- FileTree 的 Git ink 与 GitPanel 共用同一 status/revision，不再各自轮询；
- GitPanel 增加“Git / 本轮文件”双镜头；“本轮文件”直接复用 ChangesPanel 的写入/编辑/读取三态与证据链，不造第二套模型。
- GitPanel 再增加“统一”镜头：Git Status 与 SessionChange 按文件路径合并，按目录层级展示；叶节点同时显示暂存字母、工作区字母和 Agent 改动次数，点击即可打开文件。
- 统一树支持全部/Git/AI 来源筛选；文件行提供 stage、unstage 和两击 discard，动作直连既有 Git 绑定并在完成后刷新共享 status。
- 已跟踪文件可在统一行内展开 diff；同时有暂存和工作区改动时可切换两个侧面的差异，渲染继续复用 ChangesDiff。

后续仍必须先设计：

- 仓库根识别/嵌套工作区过滤已完成：状态始终从仓库根读取，再归一为当前工作区相对路径；工作区外变更不上树。
- **G1.6 已完成（2026-11-12，`7a544d37`+`205c5947`，已快进并入 main）**：①worktree 只读展示——`GaeaGitWorktrees` 绑定（`git worktree list --porcelain` 解析：主检出/linked/detached/bare/locked，isCurrent 与工作区 toplevel 等值判定），`GitStatus` 增 `isLinkedWorktree`/`mainWorktreePath`（`--absolute-git-dir` ≠ `--git-common-dir`），GitPanel 第四镜头「工作树」+ 头部 linked 标注 chip；②Git 子进程验收欠账收口——全部 git 调用走 `gitRunBudgeted`（读面 20s/8MB、写面 60s/1MB、diff 4MB 展示层截断标记），管道反压早停 + ctx 强杀兜底；③mock/签名/绑定名/legacy 四表同步，绑定 745→746。
- **G1.7 已完成（2026-11-13，分支 `feat/g1-git-display-strategy`）**：冲突/重命名/二进制展示策略收口——①Go `GitFileStatus` 增 `conflict`（六形态 U/AA/DD/AU/UA/DU/UD，与 gitStatusInk 同规则）与 `origPath`（porcelain -z rename 来源路径，走同一工作区过滤口径）；冲突文件 `staged=false`（不可提交，不进提交计数）；②GitPanel 冲突独立分组（首组+诚实提示「本面板不代解」，展开行给解决指引而非拉 combined diff）、重命名行显示 `← 原名`；③`buildGitDiff` 按前缀识别二进制输出为诚实占位「二进制文件：不提供文本 diff」（原先落「未能解析出 diff 内容」误导文案）；④gitStatusInk 冲突判定补齐六形态+吃 Go 侧 conflict 标记。
- **两个设计判定（本批固化）**：①gaea 全应用单一 workspace 锚（`ga.cfg.Workspace`），FileTree/Git/Agent 共用——上游「跟随会话进 linked worktree」会造成多面板基准分裂，**worktree「切换」= 工作区切换器，归 G2/W1 级产品门另立**，不做 add/remove/prune；②**Go Git 服务扩展不重做**——现有 `gaea_git.go` 391 行、硬化与测试齐全，缺的只是超时/预算（已收口）与 worktree 列表（已加），不搬家不重构。
- 后续：无 `git` 的更多能力级降级。
- 工作区变更、暂存区变更、Agent 本轮写入/编辑/读取三视角的数据来源和去重规则。
- 统一行内 diff 已完成；后续不再保留此欠账。
- stage/unstage/discard/commit 的动作边界和危险操作确认。
- 大 diff、二进制文件、重命名、删除、冲突、长行的展示策略。
- 何时需要重做 Git 服务、何时只需在现有 Go 包上扩展。

本批默认不做 push/pull/fetch；是否支持远端操作另立决策门。

验收：

- 工作区/Git/Agent 三个视角各自能解释“为什么这个文件在这里”；
- 任一文件状态只有一个可核对真相，不在多个面板算出不同结果；
- 删除、丢弃、提交均有可回滚或明确不可逆提示；
- Git 子进程有超时、输出预算、错误透传和隐藏窗口；
- 非 Git 仓库、空仓库、无提交 HEAD、重命名和二进制均有测试。

### G2：只读提交图

优先级：中；**✅ 已实现（2026-11-13，分支 `feat/g2-commit-graph`）**——G1 稳定（真仓测试+真机走查双实证）后按 §6 顺序推进。

已落地：

- `GaeaGitGraph(skip, limit, branch)`：分页提交（%P parents/%D 装饰）+ 本地分支清单；lane 拓扑由前端 `commitGraph.ts` 纯函数计算（先清位再布父：首父继承/次父外扩/折线并入/多根复用/截断父 through 到图底，MAX_LANES=8 钳制）。
- 分支筛选：下拉选本地分支按分支取历史，**只读不 checkout/fetch/push**；branch 名严格校验（拒选项注入/范围/reflog/.lock，中文放行）。
- `GaeaGitCommitDetail(hash)`：消息体/引用/parents + `diff-tree --root --numstat`（二进制 -1）+ 全量 unified diff（4MB 预算截断标记）；hash 只收十六进制；合并提交默认无 diff 诚实提示。
- CommitGraphView 第五镜头：SVG lane 行（dot/穿行/汇入，令牌配色零裸 hex）+装饰 chip+加载更多（--skip 续读去重）+详情展开复用 ChangesDiff。

不做（本批边界）：checkout、reflog/stash 视图、逐文件过滤 diff、lane 曲线美化（直线段诚实可读）。

### W1：底部工作台、分栏与终端

优先级：拍板后。**调研备忘已备（2026-11-13）：`docs/gaea-w1-bottom-workbench-decision-2026-11.md`**——Q2/Q3/Q4 已用上游事实与 gaea 现状回答（上游不拥有 PTY、宿主终端服务复杂度、gaea 任务中心单向 vs 终端双向、自建 ConPTY 为成本大头），三选项 A 不做/B 轻量无终端/C 全量含 PTY；**只剩 Q1（是否真的需要多 pane/多终端）待用户拍板**。

上游的底部工作台和真实终端会改变 gaea 主区结构，不能夹在文件工作台里顺手做。先回答：

1. 用户是否真的需要多 pane 和多终端？（→ 见决策备忘 §四/§五）
2. gaea 已有任务实时输出和强制终止，交互终端还缺什么？（→ 备忘 §二：双向输入/持久 shell 态/任意程序/TUI 保真）
3. 窄屏如何折叠？（→ 备忘 §二：上游折叠钮+bottomHeight，gaea 沿用 pane 折叠模式）
4. 终端是用户交互面还是 Agent 工具面？（→ 备忘 §二：上游=会话级用户交互面与 Agent exec 分立；gaea 同口径+workspace 围栏）

未拍板前只维持现有任务与浏览器面板。

### X1：插件服务与开始页接管

优先级：低。

上游的 `ctx.betterSidebar` 和插件开始页依赖 DSH 插件生态。gaea 没有同构插件市场，不应为了“蒸馏完整”引入一套模拟插件系统。只有出现真实第三方扩展需求时再评估。

## 7. 每批工作的固定流程

### 7.1 开工

1. 读用户最新指令和本文。
2. `git status`，确认不是在他人在制改动上硬覆盖。
3. 回源固定 commit，读源码、CHANGELOG、对应设计文档。
4. 写清本批目标、非目标、验收标准和预计文件足迹。

### 7.2 设计

1. 先画数据流：UI → bridge → Go 权威 → 事件/返回值。
2. 先定失败语义和安全边界。
3. 判断能否复用现有类型/组件；尽量不新增第二套平行实现。
4. 有危险操作先定确认、回滚和日志。

### 7.3 实现

1. Go 核心与测试先行，再接 App 门面与绑定。
2. 更新 `spaceBindings`、bridge types、mock 和签名。
3. 用生成器更新 Wails 文件，不手改 `models.ts` 或绑定产物。
4. 前端保留真机、旧后端/mock 回退。
5. 新测试覆盖成功、失败、空态、并发和预算边界。

### 7.4 验收

最低组合：

```powershell
cd frontend
npm.cmd run build

cd ..
go test ./internal/gaea/workbench ./internal/app
go vet ./internal/gaea/workbench ./internal/app
```

此外按影响面补：

- Vitest 定向文件；
- 所需的全量 Vitest；
- Go 相关包全量；
- Wails 构建；
- 正式路径冒烟；
- `scripts/check-bindings-drift.ps1`；
- `node scripts/check-docs.mjs`；
- 真机 `/mock=1` 或桌面走查。

需要构建、打包或装机时，立即转到 `docs/gaea-build-guide.md`，不在本文临场发明命令。

### 7.5 收口

每批结束时更新：

- 本文 §2 当前基线、§5 差距表、§6 状态、§11 台账；
- `.gaea/progress.md`；
- `.gaea/todos.md`；
- `docs/README.md` 的一行状态；
- 若改变构建或通用纪律，再更新 `.gaea/AGENTS.md` 和构建指南。

## 8. 验收矩阵

| 层级 | 必测 |
|---|---|
| Go 核心 | 正常、拒绝、边界、并发、回滚、损坏输入 |
| App 门面 | 参数透传、错误透传、原生对话框取消、生命周期关闭 |
| 绑定 | 名称、参数、返回值、space 归属、mock 签名 |
| React 组件 | 加载/错误/空态、焦点、键盘、危险确认、旧后端回退 |
| 集成 | 文件改名/删除与 tab 联动；watcher 与缓存失效；上传后刷新 |
| 安全 | 路径穿越、符号链接、非法 Windows 名称、覆盖拒绝、HTML/媒体围栏 |
| 性能 | 批量请求、展开层数、长行、大 JSONL、长 diff、watcher 洪水去抖 |
| 产物 | 前端构建、Wails 构建、DLL 同目录、`/api/health` 200、SHA256 |

## 9. 已知问题、红项与风险

### 9.1 当前状态风险

1. **已提交并打包，但未 push、未装机**。新会话不得把本文当作已发布说明。
2. `frontend/wailsjs/go/models.ts` 会由 Wails 构建重新生成；生成器仍会写入空行尾随空白，属于已知生成物噪声，禁止手工清理后再声称是生成结果。
3. 本文记录的 exe 哈希只对应 `f49ea150` 的本次干净构建；后续任何代码或构建环境改动都要重建后再记录。
4. F1/F2/F3/G1 已有自动测试与正式路径冒烟，但仍需要桌面真机走查上传、拖放、ZIP、watcher、Git 快操作和行内 diff。

### 9.2 与 F1 无关的既有红项

- `go test ./...` 整体失败只落在 `internal/voice/native` 和 `internal/voice/realtime`：
  运行时加载 ORT 1.17.1，而 Sherpa 要求 API 25/28。不是 workbench 回归。
- 绑定名称对齐检查通过，但脚本退出 1：
  - 存量死绑定 `EngineUpdateStatus`
  - 存量死绑定 `StrataModelStart`
- Wails 生成时提示 `Not found: time.Time`；当前构建与冒烟成功，但下次生成后应再次确认是否变化。
- `node scripts/check-docs.mjs` 当前存在多份既有未登记文档、悬空引用和 releases 计数漂移；本文必须登记，但不能把整体 checker 误报成绿。
- `.gaea/AGENTS.md` 接近 65536 bytes 预算，新增记忆要短，优先写入本文和 progress。

### 9.3 代码与产品风险

| 风险 | 处置 |
|---|---|
| 多面板各自轮询 Git，导致进程风暴与状态不一致 | G1 前先做单一状态源 |
| watcher 监听整棵大树导致句柄和事件洪水 | 只监听展开集合，保持去抖 |
| 前端仅隐藏排除项但后端仍扫描 | 排除规则必须在 Go 搜索/树/watcher 单源生效 |
| 大文件全量过 Wails bridge 卡死 | 保持分页、原生产物、字节预算 |
| 静默覆盖外部修改 | mtime CAS 不降级 |
| 模糊追问导致重新引入侧聊 | 本文 §1.2 为硬边界 |
| 重做 Git 时顺手改动 push/pull/fetch | 独立决策门，不夹带 |

## 10. 技术决策记录

| ID | 决策 |
|---|---|
| D01 | 在现有 gaea 上增量接线，不重建应用 |
| D02 | 侧聊服务不做 |
| D03 | Go 是文件、上传、监听和并发语义权威 |
| D04 | 保留工作区围栏，不照搬上游无围栏路由 |
| D05 | 上传优先原生选择器；拖放走分片 |
| D06 | 5MB 可编辑、256KB 分页；超过预算只读 |
| D07 | 文本保存使用 mtime CAS 和原子写 |
| D08 | 删除是永久删除，但前端必须明确警告并二次确认 |
| D09 | 底部工作台、终端、提交图分别受用户拍板门控 |

## 11. 迭代台账

每批新增一行，禁改历史结论；结论变化另加日期。

| 批次 | 日期 | 状态 | 内容 | 验证 |
|---|---|---|---|---|
| F1 | 2026-11-12 | 🟡 工作树候选 | Go 文件工作台、17 绑定、文件树 CRUD/多选/上传/ZIP/watch、文本 CAS 编辑与分页、tab 调解、mock/测试 | 前端 483/4301；Go 定向与可运行全量；Wails build；smoke 200；未提交/未装机 |
| F2.1 | 2026-11-12 | 🟡 工作树候选 | Git X/Y 文件徽标、语义色、目录/根汇总点、30s/聚焦/事件刷新；纯映射单测 | 定向 2 文件/23 用例；相关 ESLint；未跑构建与真机走查 |
| F2.2–F2.5 | 2026-11-12 | 🟡 工作树候选 | 默认/用户排除规则、@ 搜索与 watcher 同源；紧凑目录链；完整文件类型图标；打开方式子菜单 | 前端 8 文件/53 用例；Go 定向测试+vet；`tsc -b`；ESLint；绑定 745 重生成；未跑构建与真机走查 |
| F3 | 2026-11-12 | 🟡 工作树候选 | JSONL 记录卡、Markdown 浮动目录、本地图片解析、脏稿关闭/导航保护、2K/20K 长行 diff、CodeMirror 覆盖复核 | 前端 12 文件/99 用例；`tsc -b`；23 文件定向 ESLint；未跑构建与真机走查 |
| G1.1 | 2026-11-12 | 🟡 工作树候选 | Git 状态单一轮询源；FileTree/GitPanel 共用快照；Git / 本轮文件双镜头 | 定向 3 文件/29 用例；`tsc -b`；6 文件 ESLint；未跑构建与真机走查 |
| G1.2 | 2026-11-12 | 🟡 工作树候选 | 父级仓库根发现；嵌套工作区相对路径归一与外部变更过滤；porcelain -z rename/无提交分支解析 | App Git 定向测试；`go vet ./internal/app`；未跑构建与真机走查 |
| G1.3 | 2026-11-12 | 🟡 工作树候选 | 统一变更树：Git+Agent 路径合并、目录 rollup、来源徽标、文件打开 | 定向 3 文件/12 用例；`tsc -b`；定向 ESLint；未跑构建与真机走查 |
| G1.4 | 2026-11-12 | 🟡 工作树候选 | 统一树来源筛选（全部/Git/AI）+ stage/unstage/两击 discard 快操作 | 定向 3 文件/14 用例；`tsc -b`；定向 ESLint；未跑构建与真机走查 |
| G1.5 | 2026-11-12 | 🟡 工作树候选 | 统一树文件行内暂存/工作区 diff，双侧面可切换 | 定向 3 文件/16 用例；`tsc -b`；定向 ESLint；未跑构建与真机走查 |
| G1.6 | 2026-11-12 | ✅ 已并入 main（快进） | worktree 只读展示（`GaeaGitWorktrees` 绑定+GitStatus linked 标注+GitPanel 第四镜头「工作树」+头部 chip）+ Git 子进程超时/输出预算收口（读 20s/8MB、写 60s/1MB、diff 4MB 截断标记）+ 设计判定固化（切换=工作区切换器另立产品门；服务扩展不重做） | Go 定向 10 例+全包绿、vet 绿；`tsc -b`+Vite build 23.42s 绿；GitPanel 12 用例+mock 契约 19/19 绿；绑定 745→746 四表同步+drift OK；9 文件 ESLint 绿；未跑 Wails build 与真机走查 |
| G1.7 | 2026-11-13 | ✅ 已并入 main（快进 `277dc240`+`7b7f314d`） | 冲突/重命名/二进制展示策略收口：Go `GitFileStatus.conflict`（六形态）+`origPath`（-z rename 来源）+冲突 staged=false；GitPanel 冲突独立分组+不代解提示+展开行解决指引、rename `← 原名`；buildGitDiff 二进制诚实占位；gitStatusInk 六形态补齐 | Go 新 3 例（真仓 UU/rename 溯源/二进制 diff）+全包复跑绿；tsc+Vite 24.85s 绿；GitPanel 15 例（3 新）+ink+planDiff+契约 28+15 绿；7 文件 ESLint 绿；无新绑定（746 不变）；**真机走查已过（见 G1.8 行）** |
| G1.8 | 2026-11-13 | ✅ 已并入 main | ①空仓（无提交 HEAD）降级矩阵测试：status 报未诞生分支、log 诚实报错、unstage/diff --cached 按空树语义成功（git ≥2.26 实锤——首提交场景 reset/unstage/diff 天然正确）；②G1.6/G1.7 真机走查收官：沙箱夹具真仓（UU 冲突+rename+二进制+双 worktree）+`GAEA_WALKTHROUGH=1`+`GAEA_WEBVIEW_DEBUG=1` CDP 驱动实壳——冲突组置顶+计数+不代解提示、rename `← 原名`、暂存计数排除冲突、工作树镜头三行徽标/分支、linked chip 主检出态正确不亮，全断言绿零缺陷 | 空仓新 1 例绿；wails build 19.03s（c7f4 @ `7b7f314d`+G1.8 测试）；正式路径冒烟 200（payload releases/gaea-g17-g18-walkthrough-20261113/，exe 2fd2741b）；走查截图 .tmp/walk-2/3；走查壳已回收、沙箱已清场 |
| G2 | 2026-11-13 | ✅ 已并入 main（快进 `05dc2a91`+`c6f6d018`+`d8337d27`，真机走查 d048 后续提交） | 只读提交图：`GaeaGitGraph` 分页（parents/装饰/分支清单，branch 严格校验防注入）+`GaeaGitCommitDetail`（numstat 二进制 -1+4MB diff 预算+合并诚实空 diff+hash 十六进制校验）；前端 `commitGraph.ts` lane 纯函数（先清位再布父/MAX_LANES 钳制）+CommitGraphView 第五镜头（SVG lane/装饰 chip/分支筛选不 checkout/加载更多/详情展开复用 ChangesDiff）；绑定 746→748 | Go 新 2 例+app/workbench 全包 exit 0；tsc+Vite 24.01s；vitest 定向 30/30；11 文件 ESLint 零；drift 748 OK；**真机走查已过**：夹具仓（no-ff 合并+tag+rename+双 worktree）实壳断言全绿——6 行拓扑序/分支选项/tag 装饰/只读标注/合并「默认无 diff」提示/二进制占位/feature 筛选重拉 3 行精确；wails build 9.98s+冒烟 200（payload gaea-g2-walkthrough-20261113 exe a51323f4 留盘） |
| F0 | 2026-11-12 | ✅ 已提交并打包 | 功能提交 `f49ea150`；独立干净工作树构建；预览 ZIP + 提交源码 TAR + 包记录 + 长期记忆更新 | 前端 build/Vite 20.38s；Wails 50.158s；smoke 200；ZIP/TAR/四件套 SHA256 已核验；未 push/未装机 |

## 12. 下一会话建议提示词

需要继续代码时，新会话可直接使用：

```text
先读 .gaea/AGENTS.md、.gaea/progress.md、.gaea/todos.md 和
docs/gaea-better-sidebar-go-iteration-2026-11.md。
这是现有 gaea 的增量复刻，不重写；侧聊服务不做。
先检查当前工作树；功能基线提交为 f49ea150，勿把平行测试优化混入本专项。
本次只做 <批次号/功能>，按文档的验收标准和构建纪律执行；
完成后同步本文台账、progress、todos 和 docs 索引。
```

## 13. 维护纪律

1. 本文只记录当前权威状态，不堆积每日流水；日常细节进 `progress.md` 和 day-log。
2. 差距表每次回源或完成批次后更新，已做项不得继续留在欠账区。
3. 用户拍板变更必须写入 §1、§6 或 §10，不能只留在聊天中。
4. 若本文与旧规划冲突，以本文和当前代码为准。
5. 若新会话没有看到本文，优先修复 `.gaea/AGENTS.md` 的入口，而不是凭旧总结开工。
