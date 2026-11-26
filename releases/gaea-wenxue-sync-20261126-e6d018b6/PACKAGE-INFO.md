# gaea-wenxue-sync-20261126-e6d018b6

- 交付提交：`4f1e8049`（`feat(wenxue): 文学板块对办公优化的同步批——右栏子会话窗/插话绑定/工具面预算守卫/分叉结构缝`）
- 同包提交：`b4934d7a`（`fix(office,xlsx): 重算脚本内嵌二进制 + 独立 LibreOffice 配置目录`——宁丰反馈 #1 补登，已装机批）、
  `e6d018b6`（`docs(gaea): 文学同步批+重算内嵌批记档；清理仓根四张弃用图片`）
- 产品版本：日常装机包，不抬版本（wails.json productVersion 未动）
- 构建工作树：`C:\AI\wubigrok\.worktrees\pkg-e6d018b`，detached HEAD @ e6d018b6
  （主树零未提交业务改动），frontend/node_modules 为主树 junction
- 前端：`npm.cmd run build`（tsc -b + vite build），Vite `30.7s`
- 桌面：系统 `TMP/TEMP`（显式 `C:\Users\wubi\AppData\Local\Temp`）、`CGO_ENABLED=1`、
  ucrt64 gcc；PATH 前置 ucrt64 bin 与主树 `build/bin`；仓根 `wails build -s -ldflags
  "-s -w" -trimpath`，bindings 未跳过，55.8s
- **wailsjs 零漂移**：干净树构建后 `git diff --quiet -- frontend/wailsjs` 过
  （status M 全 CRLF 假象）——`WenxueB.d.ts/.js` 手补的 `WenxueSubagentSteer`
  与 wails 再生件逐字节一致
- 冒烟：正式 payload `/api/health` 返回 200（`{"status":"ok"}`，冒烟进程 PID 60032 已回收）
- 装机：包版四件先回写主树 `build/bin`（哈希对账一致），再 `gaea_install.ps1`
  （默认 `-SourceDir`=主树 build/bin）→ INSTALL OK，装机位 `C:\AI\gaea-app`，
  新壳 PID 34140（09:34:59 启动，晚于构建时刻 09:34:40）
- 回写：包版 exe 已回写主树 `build/bin`；三个 DLL 与上一个包
  （gaea-office-tools2-20261126-113d84ad）逐字节相同
- exe SHA256 `9b8f97111454e6d913cd3b80fc2bbba99bc70dcbfcd6330919d8851278d367de`
  （zip `b02ef89b64c126cb…`，src.zip `1771fe1acd23da31…`，全量见 SHA256SUMS.txt）
- zip `b02ef89b64c126cb…`（33,279,599 B）；src.zip `1771fe1acd23da31…`（56,410,209 B，
  `git archive e6d018b6 -- . ':(exclude)releases'`——排除 releases/ 防历届包嵌套雪球）

## 交付内容

### 文学板块对办公优化的同步批（4f1e8049，两批合一）
办公自文学分叉（44f8bb6b，11-15）后 325 笔的同步审计与施工。引擎/后端本就对齐
（两板块共用 boot.Build）；本包补齐前端 fork 树漏面 + 三类分叉结构缝：

1. **右栏子会话窗 + 子智能体目录树 + 插话**：委派卡点击开右栏（主对话不再被切走）、
   顶栏「N 个子智能体」血缘目录树（行=主区、侧栏钮=右栏）、面包屑谱系；Go 新增
   `WenxueSubagentSteer`（插话终态腿走图文追问管道；独立 Stop 维持批6 决定不换锚）。
2. **子代理体验对齐**：后台完成卡可点、委派显示名「子代理 A/B/C」、子代理会话统一
   投影渲染、queued/aborted 状态词、后台启动回执与子代理答复不在父对话重复成行。
3. **能力面板重设计镜像**（11-19/11-22 两批）：壳/三区取办公现版，工具目录按功能
   分组（wenxue_* 归「图文专属」），样式直引办公 capabilities.css。
4. **上下文视图**：切模型后已打开的上下文 tab 重拉；能力摘要按显示宽度截断。
5. **CSS 同步**：灵犀皮肤文件卡色值接主题 token、代码高亮接 --syntax-* 并修浅色
   选择器（分叉起浅色主题代码块恒用暗色配色）、上下文视图补 5 段 @media 并清
   分叉坏尾（作用域脚本泄漏的 JS 数组字面量）。
6. **分叉结构缝**：i18n 缺键回落办公词典（委派卡裸键名根修）；Toast/useNow 改
   转发（提示曾被静默丢弃）；BoardPortsContext 板块作用域缝（keepAlive 下两板块
   同挂，任务卡注入点/文件打开入口/附件条预览各读各的，办公未提供=行为不变）。
7. **文学工具面预算守卫** `wenxue_toolface_test.go`：量出造价/长文方案两组 6 件
   办公运行模式工具混进文学工具面（约 4400 字符、25%）→ 按组整组拒绝；复测
   **49 件 / 13285 字符**（对照梳理前 55 件 / 17682），预算钉 49/13750。

### 重算脚本内嵌（b4934d7a，宁丰反馈 #1 补登）
xlsx_recalc 脚本 go:embed 进二进制 + 每次调用独立 LibreOffice 配置目录
（外部 LO 实例存活时旧脚本假通过根修）；zip 预检秒退/回读重试/三类错误哨兵；
已随 11-26 上午主树装机（exe 85DD1F23…），本包为其打包版。

## 门禁（打包前全绿）
tsc -b 0；vitest wenxue 30 文件 201 例、办公回归 15 文件 165 例；eslint 54 文件
0 错 0 警告；go vet ./internal/app；go test TestWenxue*+TestBindingsCompleteness+
TestOfficeToolFaceBudget；check-bindings-drift.ps1 OK（665 方法）；node
scripts/check-docs.mjs 卫生守卫 OK。
