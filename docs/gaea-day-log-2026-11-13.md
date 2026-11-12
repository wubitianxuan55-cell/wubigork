# gaea Day Log · 2026-11-13 · G1.6 并入 main + G1.7 冲突/重命名/二进制展示策略收口

> 承 11-12 深夜 G1.6（分支 `feat/g1-git-deepening`），凌晨快进并入 main（`205c5947`），
> 随后按交接档 §6 顺序开 G1.7。push/装机始终挂用户拍板。

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
