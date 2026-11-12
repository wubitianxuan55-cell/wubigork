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
