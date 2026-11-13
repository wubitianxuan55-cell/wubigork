# gaea-officecom2-20261113-e0461f53

- 提交：e0461f53（波3.5 追加批功能头；main 头为后续 docs 行）
  - 相对 officecom 包（f5576986）新增四笔功能：
    - 19304744 交付登记补 doc 键——office_save 显式 doc 保存进登记表/Dock/
      右侧面板（盲区修复；doc 缺省纯 {} 场景已知盲区记录）
    - e0461f53 Excel 大表 office_read 输出 38KiB 主动截断+结构化指引
      （profile 概况/缩小 range 分块；361KB 硬截断实锤）
    - （前序）562d4eb0 五卡批回归 main（办公对话输出重设计：过程/思考/工具/
      Ask/审批五卡+1266 行样式+设计账+截图）
    - （前序）cacb011c/48cfa091：过程卡下置→撤销、折叠活尾巴、交付卡跨段
      去重、提炼记忆弹窗、office_com 设置段+settle 钩子+停靠条前身的迭代链
- 另含：交互文件固定落点（office_open path 归一→「交互文件」目录+title
  语义命名）、过程卡折叠语义细化（完成=一行摘要/running=最近三卡尾巴）、
  工作态头部删「思考中…」回退文案、过程卡底色透明、Excel 大表输出保护
- 构建：**仓内干净工作树 .worktrees/pkg-e0461f53**（worktree add 定提交+
  node_modules junction；frontend npm build 22.96s + `wails build -s -ldflags
  "-s -w" -trimpath` 53.8s，ucrt64 gcc + CGO_ENABLED=1 + 系统默认 TMP；
  三 DLL 从主树 build/bin 同步入包）
- 冒烟：装机位 C:\AI\gaea-app\gaea.exe /api/health 200
- 装机：gaea_install.ps1 -SourceDir（包工作树产物）→ C:\AI\gaea-app，
  **装机=包版**，exe sha256 4139E78C4D57ABA6…（17:10:01 拉起 PID 46324）
- 三 DLL：68D0D5F5 / 5DD36DBD / 4C563277（与前包一致）
- zip/tar 磁盘保留不入库（P2-3 纪律）
