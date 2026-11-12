# gaea Day Log · 2026-11-13 · G1.6 并入 main + G1.7 展示策略收口 + G1.8 降级矩阵/真机走查

> 承 11-12 深夜 G1.6（分支 `feat/g1-git-deepening`），凌晨快进并入 main（`205c5947`），
> 随后按交接档 §6 顺序开 G1.7、G1.8。push/装机始终挂用户拍板。

## 1. G1.6 并入 main（本地合并）

- `feat/g1-git-deepening`（7a544d37 feat + 205c5947 docs）自 `de283ace` 快进并入 main；
  main 领先 origin 5 笔（f49ea150 系 2 + de283ace + G1.6 两笔），**未 push**（专项 gate）。

## 2. G1.7 冲突/重命名/二进制展示策略收口（分支 `feat/g1-git-display-strategy`，自 `205c5947`）

§6 G1 余项里最具体的三笔欠账（「大 diff、二进制文件、重命名、删除、冲突、长行的展示策略」，
其中长行已随 F3 收口 2K/20K）：

- **Go**（`internal/app/gaea_git.go`）：
  - `GitFileStatus.Conflict`：六形态判定（任一 U，或 AA/DD/AU/UA/DU/UD）——`isConflictXY`
    与前端 gitStatusInk 同一规则；**冲突文件 `staged=false`**（不可提交，不进提交计数，
    修正原先 UU 落「已暂存」组的误导）。
  - `GitFileStatus.OrigPath`：porcelain -z rename/copy 的第二字段（来源路径）此前被 i++
    直接丢弃，现保留并走与 dest 相同的 `workspaceStatusPath` 归一/过滤口径（工作区外来源
    不溯源）。
  - 测试 3 新：真仓制造 UU（双分支改同一文件 merge）断言 conflict+非 staged；rename 溯源
    断言 origPath==inside.txt；二进制文件 diff 断言 "Binary files" 行原样过管道（clean
    filter 中和/预算层不改语义）。
- **前端**：
  - GitPanel：冲突独立分组（首组 + 提示「未合并冲突不可直接提交……本面板不代解」）；
    冲突行展开给解决指引（`<<<<<<<` 标记 + git add）而非拉 combined diff；重命名行显示
    `← 原名`（title 溯源全路径）；冲突字母 destructive 色。动作按钮按 side 收敛——冲突行
    无 stage/unstage/discard（git add 冲突文件=把冲突标记原样入暂存，属危险动作）。
  - `planDiff.buildGitDiff`：`Binary files`/`GIT_BINARY_PATCH` 前缀 → 诚实占位
    「二进制文件：不提供文本 diff」（原先落「未能解析出 diff 内容」，误导为解析故障）。
  - `gitStatusInk`：冲突判定补齐六形态 + 吃 Go 侧 conflict 标记（原只有 U/AA/DD 四形态）。
  - mock 样例补冲突+rename 两行走查。
- **验证**：Go 新 3 例+定向全绿；`tsc -b`+Vite 24.85s 绿；GitPanel 15 例（3 新）+ink+
  planDiff 15+契约等定向 28/28+15/15 绿；7 文件 ESLint 零问题；无新绑定（746 不变）。

## 3. 坑册

1. `git status --porcelain -z` rename 的 origin 是独立 NUL 字段——只 i++ 不读=静默丢溯源；
2. 冲突文件在 porcelain 里 `staged` 语义为真（X=U 非 ' '），UI 层必须显式排除，否则进
   提交计数；
3. combined diff（git diff 对 UU 文件）非经典 unified 格式，本面板不解析、诚实提示；
4. 前端测试断言文案要与分组级提示错开（冲突组提示与展开行提示同含「git add」，
   findByText 正则会双命中）；
5. `cat >> 新文件` 会静默创建 0 字节文件——写 day-log 前先 `ls` 核存在性。

## 4. G1.7 快进并入 main（00:2x，本地）

- `feat/g1-git-display-strategy`（277dc240 feat + 7b7f314d docs）快进并入 main；
  main 领先 origin 7 笔，**未 push**。

## 5. G1.8 降级矩阵 + G1.6/G1.7 真机走查收官（00:3x）

- **空仓降级矩阵**（`TestGaeaGitUnbornHead`）：status 报未诞生分支无文件、log 诚实报错
  （前端「暂无提交」空态）；**意外实锤——git ≥2.26 把未诞生 HEAD 当空树**：unstage
  （reset）与 staged diff 都成功且语义正确（unstage 清暂存、diff 全显新增），首提交场景
  天然可用，断言按成功语义落。worktree list 空仓正常返回主检出。
- **真机走查**（GAEA_WALKTHROUGH=1 沙箱 + GAEA_WEBVIEW_DEBUG=1 → CDP 9333，实壳实绑定）：
  - 沙箱夹具真仓：UU 冲突（双分支 merge）+ `git mv` rename + 二进制 + `git worktree add`
    linked/detached；
  - 走查断言全绿零缺陷：Git 镜头=冲突组置顶+计数+红字「本面板不代解」+冲突行无快操作、
    rename 行「说明书.md ← 说明.md」、**暂存 1 个文件（冲突正确排除出提交计数）**、
    二进制/未跟踪各行就位；工作树镜头=主检出/当前/detached 徽标+分支色+路径，linked
    chip 在主检出态正确不亮；
  - 构建链：wails build 19.03s（c7f4 @ 7b7f314d+G1.8 测试），正式路径冒烟 200
    （payload releases/gaea-g17-g18-walkthrough-20261113/，exe 2fd2741b，留盘不入库）；
  - 走查壳回收、9333 无监听残留、沙箱 gaea-walkthrough 整体删除。
- **坑册**：
  1. WebView2 CDP 正牌开关=`GAEA_WEBVIEW_DEBUG=1`（固定 9333）——
     `WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS` 传 `--remote-debugging-port` 在 Wails 壳
     不生效（Wails 自管 args）；
  2. 杀壳后 WebView2 子进程可能残留占 9333——树杀后再核属主（PID 已死即清场）；
  3. `git worktree add -b wt/x` 会撞已有 `wt` 分支（ref 目录冲突）——夹具分支名避开
     前缀重叠；
  4. GitPanel 生产消费方在 `sidebarRegistry.ts` 的 `git` pane（非直接 JSX）——右栏
     「展开文件面板」→ Git 页签进入。

## 6. G2 只读提交图（01:0x，分支候选）

> §6 顺序推进：G1 稳定（G1.8 双实证）后开 G2。硬边界=只读：不 checkout/fetch/push。

- **Go**：`GaeaGitGraph(skip,limit,branch)`（分页+parents/装饰+本地分支清单；branch 校验拒
  选项注入/范围/reflog/.lock/空白，中文分支放行）+ `GaeaGitCommitDetail(hash)`（消息体/
  引用/parents + `diff-tree --root --numstat` 二进制 -1 + 全量 diff 4MB 预算截断；hash 只收
  十六进制 7~40 位；合并提交默认空 diff 诚实提示）。
- **前端**：`commitGraph.ts` lane 纯函数——首版算法有真 bug（`lanes[lane]=null` 在布父之后
  执行，把刚放入同槽的首父抹掉，整图塌成单 lane），改「先清位再布父」后 6 拓扑单测钉死
  （线性/外扩/合并/多根复用/截断父 through 到图底/病理钳 MAX_LANES=8）；`CommitGraphView`
  第五镜头：SVG lane 行（dot/穿行竖线/汇入折线，令牌轮换色零裸 hex）+装饰 chip（HEAD->
  main 高亮/🏷tag/远端）+分支筛选下拉+加载更多（--skip 续读 hash 去重）+详情展开（消息体/
  引用/numstat「二进制」标记/ChangesDiff 复用）。
- **验证**：Go 新 2 例（真仓 no-ff 合并 7 提交拓扑/分页/筛选/双注入拒绝/二进制 numstat/
  合并空 diff + refSafe/hashSafe 边界）+app/workbench 全包 exit 0；tsc+Vite 24.01s；vitest
  30/30；11 文件 ESLint 零；drift 748 OK；绑定 746→748 四表+wailsjs 同步。
- **真机走查**：待下一轮打包一并（沙箱夹具真仓拓扑可复用）。
- **坑册**：
  1. lane 算法经典次序坑：清位必须在布父**之前**（首父继承同槽时后清会抹掉刚布的父）；
  2. wailsjs 补账与 CRLF 噪音同批出现——`git diff <file> | grep -c "^[+-][^+-]"` 区分真实
     变更（0 行=纯换行相位），`git checkout -- app/` 整目录还原会误伤真实新增行，逐文件
     处理；
  3. `wails generate module` 可独立再生 wailsjs（无需完整 build）；每跑一次全 M 是 autocrlf
     相位，非真实变更。

## 7. G2 真机走查收官（01:4x）

- 夹具重建：合并拓扑干净落提交（feature 臂改 f.txt/加 g.txt、main 臂加 h.txt）+tag v0.1.0
  +staged rename+双 worktree；wails build 9.98s（G2 版 exe a51323f4）+正式路径冒烟 200。
- 实壳断言全绿零缺陷（GAEA_WALKTHROUGH+GAEA_WEBVIEW_DEBUG→CDP 9333）：提交图镜头 6 行
  拓扑序（b6/b5/b4/b3/b1/b2）、分支选项四项、🏷 v0.1.0 装饰、「只读视图（不 checkout）」
  标注、合并提交展开「默认无 diff」诚实提示、二进制提交「二进制」标记+占位、feature
  筛选重拉精确 3 行（b3/b2/b1）。截图 .tmp/walk-4-graph.png。
- 观察项（非缺陷）：镜头页签增至 5 个后「提交图」在窄栏换行显示——观感可接受，后续若
  拥挤可改图标化。
- 壳回收+沙箱清场+9333 无监听残留。G1+G2 全部镜头真机双实证齐备。

## 8. 仓面卫生批（02:0x）

- check-docs 四查全绿（首杀两笔既有 FAIL）：悬空引用 35 处清零（34 处改指 docs/archive/
  ——2026-09 规划族归档后引用未随迁；AMD_HIP.md 系 DSH 侧笔记从未入库，两处引用改
  「未入库」注记）+ docs/README 补登 17 份（day-log 11-04~13 七份+语音蒸馏审计三份+
  websearch digest+contextview 两份+chat-probe 三份+t0-effect-audit）+ releases 校验和
  计数 530→538（口径补记：F0 起每预览包自带 SHA256SUMS-<commit>.txt，递归实测
  根 36+archive 494+包目录 8）。
- knip 尾款两刀：startWhirlEngine 死包装退役（前端无调用方，WHIRL 启动走受管目录通用
  路径，Go WhirlEngineStart 绑定保留+测试 mock 行随迁）+STREAM_SILENCE/INACTIVITY 重复
  导出合并（统一语义名，ChatPage/测试随迁）。tsc+Vite 25.15s+定向 30/30 绿。余量=
  wailsjs 生成件+voice-worklets 既定假阳性基线（183+26 项，非自研欠账）。
- **race linux 6 例不修（诚实边界）**：无 gh（无法取 CI 日志）+无 WSL（无法本地跑 linux
  测试）——静态猜测修复违「宁不立项不虚构」。开修前提=用户提供 CI 日志或装 WSL。

## 9. AGENTS.md 分流+W1 决策备忘（02:3x）

- **AGENTS.md 顶部速览分流**：速览违「只留最近 3 版」纪律积压至 11-06 时代（最新发布条停在
  v4.529.1，真实最新已 v4.543.0）——v4.455~v4.529 时代条+2026-10-30~11-06 未发布/提交状态
  条**逐字节整段迁** `docs/archive/agents-version-history-2026-09.md`（阈值校准条随迁）；速览
  重写=新 3 版（v4.543/542/539）+当前活跃线（better-sidebar 全清 ahead 16）。**水位
  60449→25904B**，近水位 WARN 解除，check-docs 仍四查全绿。
- **W1 底部工作台/终端决策备忘**（docs/gaea-w1-bottom-workbench-decision-2026-11.md，交接档
  §6 W1 挂链）：回源 v0.32.0 实锤——**上游不拥有 PTY**（v0.21.1 让渡宿主），终端驱动宿主
  会话级终端服务（帧式流+acknowledge 背压+snapshot 重放+进程跨卸载存活+contentId 持久收养
  重载同 shell）；gaea 无宿主服务，**PTY 栈须自建**（Go ConPTY+帧式流+恢复）=成本大头。
  Q2/Q3/Q4 已答（终端=用户交互面与 Agent exec 分立、折叠沿用 pane 模式、任务中心单向 vs
  终端双向）；三选项 A 不做/B 轻量无终端/C 全量含 PTY；**只剩 Q1 待用户拍板**。

## 10. 代码图谱追平 + dshmarket 上游跟踪销号（02:5x）

- **代码图谱再生成**（ff3931f0）：162→164 包（+internal/gaea/workbench、filefilter，F1 增）、
  源码 260k→264.8k 行、测试 208k→210.8k、内部边 467→474；分层不变量三项复算全 ✅；
  设置面板快照资产 `frontend/src/data/codeAtlas.json` 同步（CodeAtlasPanel 7/7 绿）——
  图谱从 5354c161（11-10）追平至 G1/G2+卫生批后。
- **dshmarket 上游跟踪销号**：npm 实测 `dshmarket@1.73.19`（2026-11-12 发布）
  `lib/restart.js` 已含 `windowsHide: true`（detached spawn 行 516+注释 394/511 实锤）——
  本地 `dshmarket@1.73.15.patch` 转为**可退役**；升级动作属 C:\AI\dsh 用户资产，留用户
  升级 dshmarket ≥1.73.19 时摘 patch（集成文档 §176 约定兑现）。

## 11. 全仓验证扫（push 拍板前置，03:0x~03:3x）

- **go test ./...**：仅 voice/native+voice/realtime 4 例既有环境红（ORT API 25 vs DLL 1.17.1，
  在册定性），**全仓零新增红**——自 11-02 后首次整轮全绿（除既有）。
- **golangci 58→0**（a4d727ec）：未推区间累积（CI 只看 origin/main，push 即红的实际阻塞）
  ——gofmt 16 文件+errcheck 35（engine_windows Destroy 全量 `_ =` 化+六处 Close/Fprintln）
  +misspell 2（strat→strategy 日志标签）+staticcheck 2（SA4004 无条件 break 环改 len 守卫
  单调用/SA1024 cutset 重复空格）+unused 3（qwen35Ensure/subagentReadOnlyFromContext/
  engineError 死码退役）；受影响 12 包测试绿。
- **eslint 12 error→0**（6ce83f09）：**mock/core.ts 真 bug**——会话路径单反斜杠被字符串转义
  吃掉（`C:\demo\.gaea\...` 实际成 `C:demo.gaeasessionswork`）改双反斜杠；cards 字符类转义；
  Insights 裸 hex 改 token。
- **version drift OK（4.543.0 三源一致）+全量 vitest 498 文件/4361 用例全绿**（4348→4361，
  含 G1/G2 新测试）。
- 本地快闸口径至此全绿：go build/vet/lint+docs 四查+绑定 drift 748+version drift+全量
  前端测试——**push 拍板的前置验证全部就绪**。

## 12. 日常装机（02:1x，按构建指南标准流程）

- 主树（main @ `75787d27`）构建：frontend Vite 24.04s + wails 58.20s；exe SHA256
  `8CD4660A4182EF83…`；`gaea_install.ps1` 四件套入 `C:\AI\gaea-app` + 桌面快捷方式，
  **INSTALL OK**。
- 三验：装机位哈希与 build/bin 一致（8cd4660a4182…）✓；正式路径冒烟 200 ✓；冒烟进程
  回收、桌面无残留 gaea 进程 ✓。**不抬版本**（日常装机口径，4.543.0）；未 -Launch
  （凌晨不拉用户桌面）——下次拉起即得 G1/G2 全部新面（Git 工作台五镜头含提交图）。
- 边界注记：F0 门「未覆盖正式安装版」指当时预览包（已过时两天，不再适用）；本次为
  标准流程日常装机，与 office/11-10 批自主装机先例同口径。push 23 笔仍挂拍板。

## 13. 发版门欠账立账（CHANGELOG 版本段断档 v4.530~543，02:5x~03:4x）

- **实锤**：CHANGELOG 版本段停在 v4.529.1（2026-10-30）；releases/ 版本说明同样停在
  v4.529.1。此后 **14 个已装机版本（v4.530.0~v4.543.0）零版本段、零发布说明**，内容压在
  17 个「## 未发布」段+**v4.536~541 六版连 CHANGELOG 未发布段都没有**（WHIRL 族/llama 修
  在 CHANGELOG 全缺）。
- **权威考古（wails.json productVersion 抬号史逐笔核）**：v4.530.0=9ccb9333(11-04) /
  4.531.0=608af744(11-04) / 4.532.0=2d3d7125(11-05) / 4.533.0=13b3493f(11-06) /
  4.534.0=f57100ae(11-06 交付合同) / 4.535.0=dd1a943b(11-08 编程板块+3D脑网格+语音星团) /
  4.536.0=d66fd458(WHIRL 接入) / 4.537.0=207c218b(版本检查) / 4.538.0=aee84e61(引擎实升) /
  4.539.0=cee22ac8(35B 单副本) / 4.540.0=5e2960e8(llama 思考通道) / 4.541.0=f60cd0af(plain
  聊天多轮) / 4.542.0=b1c3d98d(上下文界面) / 4.543.0=daaa0065(超大附件)。
- **不代写史（纪律）**：14 版 release notes 正文+CHANGELOG 折段=发版门「人手收尾」工程
  （release.ps1 只追骨架），单方面写史有错配风险（如 OpenMausBot/Living Constellation 是
  11-07 安装=随 v4.534.0，不在 4.535 三线批内）。考古表已入本节，发版时按表折段即可。

## 14. F0 打包工作树回收（03:5x）

- `.worktrees/pkg-f49ea150`（F0 预览包构建树，批已收官包已留盘）按惯例回收：摘
  node_modules junction（主树目标完好）→ 树内 M 文件验明纯 CRLF 幻影（全树 diff 内容行
  =0）→ `git worktree remove --force`。worktree 列表余 main+两外线，.worktrees 空置。
