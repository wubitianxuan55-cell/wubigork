# gaea-office-tools2-20261126-113d84ad

- 交付提交：`113d84ad`（`chore(bindings): 同步 wailsjs 生成件——补齐首批撤下 4 个绑定后的漂移`）
- 内容提交：`72625cb9`（`fix(office,tools): 办公工具梳理二批：工具面再收 2 件/约 1000 字符 + 预算守卫 + 造价口径规矩迁技能`）、
  `b16bc58e`（`feat(model): GLM 引擎目录加 GLM-5.4（官方 2026-11-25 口径）`）
- 基础提交：`68b8f9c7`（`docs(gaea): 补登三档工具面改后实测`）
- 产品版本：日常装机包，不抬版本（wails.json productVersion 未动，4.544.0）
- 构建工作树：`C:\AI\wubigrok\.worktrees\pkg-72625cb9`（后切到 113d84ad），detached HEAD，
  frontend/node_modules 为主树 junction
- 前端：`npm.cmd run build`，Vite `38s`（tsc -b 在前，合计 1m27s）
- 桌面：系统默认 `TMP/TEMP`、`CGO_ENABLED=1`、ucrt64 gcc 16.1.0；PATH 前置
  ucrt64 bin 与主树 `build/bin`；`wails build -s -ldflags "-s -w" -trimpath`，
  bindings 未跳过，16s（Go 缓存热；同树首次构建 1m14s）
- 冒烟：正式 payload `/api/health` 返回 200（`{"status":"ok"}`，冒烟进程 PID 23156 已回收）
- 装机：`gaea_install.ps1 -SourceDir <payload> -Launch` → INSTALL OK，装机位
  `C:\AI\gaea-app`，新壳 PID 14692（00:14:56 启动，晚于构建时刻 00:12:54），
  诊断口 18123 `/healthz` 回 `ok`；装机前查旧壳（PID 50204）空闲：6s 窗口 0.14 CPU 秒，
  子进程仅 WebView2 与常驻 office-helper
- 回写：包版四件已回写主树 `build/bin`（与装机位、payload 三处 SHA256 一致）；
  构建工作树已回收（node_modules junction 先摘，主树 500 条目不变）
- exe SHA256 `726CA97309F7524E4334D74A676E51BC9AD3E7C3BB22B16D26AF69D01C72300E`
  （zip `B662308D…`，src.zip `25484433…`，全量见 SHA256SUMS.txt）；三个 DLL 与
  上一个包逐字节相同
- 源码包：`git archive 113d84ad -- . ':(exclude)releases'`（5493 文件；排除 releases/
  防历届发布包嵌套雪球——上一包 src.zip 834MB 即未排除所致）

## 交付内容

### 办公工具梳理二批（承接 799e9ba8 首批）
排刀依据 = 69 个真实办公会话（9~11 月）的调用频次。同口径 TestOfficeToolFaceDump：

| 档 | 首批后 | 本批后 | 自梳理前累计 |
|---|---|---|---|
| 标准 | 60 件 / 17395 字符 | 58 件 / 16404 | 73 件 / 21779 → −25% |
| 造价 | 68 件 / 28867 | 66 件 / 27824 | 82 件 / 38774 → −28% |
| 方案 | 72 件 / 21917 | 70 件 / 20863 | 79 件 / 24945 → −16% |

COM 实时协同档 10 → 7 件（按本机条件装配，不计入上表）。

- **A2 `request_permission`**：描述 1188 → 559 字符。69 会话仅 4 次调用，却每轮占
  三档最大单项；只留调用前必须知道的规矩，各决策的后续动作由回执给出。
- **B4 `xlsx_recalc` 回执自带审计**：重算后直接附审计结果（错误值/自引用/未重算）
  与交付门结论（重算 25 次、审计 22 次几乎成对）；`xlsx_audit` 保留为只读复核面。
  **`docx_accept` 并入 `docx_apply`**（`accept:true`；Journal 记录名仍是
  docx_accept，通道 A 复核不受影响）。
- **B2 `slash_command` 并入 `run_skill`**：项目命令按名展开提示词、技能优先、
  `name="list"` 取清单；命令展开不计技能用量。**`read_skill` 改判保留**——它是
  只读席位（众神殿白名单、只读闸）的唯一读技能通道，`run_skill` 是写类工具。
  顺带消除 `slash_command` 对技能名空指针（技能条目不带 Render，已实测复现）。
- **B3 COM 档**：`office_status` / `office_help` 并入 `office_read`
  （`status:true` / `help:true`，仍是只读面）；`office_macro` 退役（两个宏各等价
  一条 replace op，操作清单里已有同款写法，5 次调用）。
- **守卫 `TestOfficeToolFaceBudget`**：三档件数/字符预算定数（58/66/70 件，
  16750/28250/21250 字符），已下线工具名不得回潮。
- **C2（技能侧）**：造价口径两条规矩迁入造价 playbook「口径纪律」与
  cost-estimate 技能「口径」节。用户级 AGENTS.md 里的两条迁出属用户数据，待确认。

### 其他
- GLM-5.4 进内嵌模型目录（上下文 512000、输出 128K；按量付费未开放不填单价；
  积分系数 In 4.6 / Cached 1.1 / Out 16）。装机后已删除热更新文件
  `%APPDATA%\gaea\whisper_data\glm_catalog_remote.json`（备份在
  `%TEMP%\gaea-tool-audit\glm_catalog_remote.json.bak-20261126`）。
- wailsjs 生成件同步：首批撤下的 4 个绑定（GaeaMorningPreload / GaeaSetMorningPreload /
  GaeaMemoryBrief / GaeaSetMemoryBrief）与 MemoryEvalReport 字段此前没同步生成件，
  干净树再生成时暴露，113d84ad 补齐。**从 72625cb9 打的首个包已作废**。

## 未做 / 待定
- B5 后台任务三件（wait / bash_output / kill_shell）、B6 子代理管理三件并入 task：
  未动。wait 60 次、bash_output 39 次是高频协议面，并入收益小、风险大，建议不做。
- 流空闲超时（60s→10min）造价档长思考真机实弹仍未做。
- 用户级 AGENTS.md 两条造价规矩迁出：等用户确认（改前先备份）。

## 验证

- Go：app 包 1423 用例四分片并线 78s，0 失败；skill / command / boot / tool /
  builtin / evidence / skillstats / modelengine 包通过；`go build ./...`、`go vet`。
- 前端：主树与干净树 `tsc -b` 各 0 错；受影响 vitest 6 文件 89 例通过；干净树
  spaceBindings + mock contract 15 例通过。
- 注册表地图重生成，`registry_map_doc -check` OK。
- 存量红（未改动的 HEAD 同样红）：agent `TestStaleGuardBlocksMoveFileWithoutRead`。
