# gaea-officecom-20261113-f5576986

- 提交：f5576986（波3.5 功能提交；main 头 385757f5 为其 todos doc 行）
  - dsh-office COM 实时协同档蒸馏四波全清的收官提交：
    - 波1 abf241ce：八工具 COM 档（open/status/read/edit/render/save/close/help，
      OfficeHelper.cs 逐字节照搬+csc 编译缓存+/codepage:65001）+三处纪律
    - 波2 7b793944：omml LaTeX→OMML 子集直转（WPS 公式 math 腿，WPS 实弹 2
      equations 全建成）+xlsxedit 扩清洗 ops+xlsx_profile 只读工具+PPT 版式引擎
      拍板取消（helper 保真不裁、教学面只发基础 ops）
    - 波3 48cfa091：office_com 设置段五键（OfficePanel「实时编辑」节）+settle
      钩子（TurnDone 槽+活动性守卫+静默文档 Journal+office_file_saved）+壳内
      实时编辑状态条（OfficeLiveBadge 挂 FilePreview 头=迷你卡替代）
    - 波3.5 f5576986：preferLive 引导回（探测下沉 comhelper+boot 提示词分工段，
      SingleModelPrompt 逐字节红线不动）+unseen 交付前总览闭环（save 自动
      总览+Final check）
- 构建：**仓内干净工作树 .worktrees/pkg-f5576986**（git worktree add 定提交；
  node_modules junction；frontend npm build 20.20s + `wails build -s -ldflags
  "-s -w" -trimpath` 50.3s，ucrt64 gcc + CGO_ENABLED=1 + 系统默认 TMP；
  三 DLL 从主树 build/bin 同步入包）
- 冒烟：装机位 C:\AI\gaea-app\gaea.exe /api/health 200
- 装机：gaea_install.ps1 -SourceDir（包工作树产物）→ C:\AI\gaea-app，
  **装机=包版**，exe sha256 21BD0668E1C109E8…（12:52:29 构建）
- 三 DLL：68D0D5F5 / 5DD36DBD / 4C563277（与 contextup4 包一致）
- 本包相对 contextup4（ecef54cd）：dsh-office COM 实时协同档四波全量
  （八工具+omml math+xlsx 扩面+设置段+settle+状态条+引导回+unseen 闭环）+
  workbench ms 粒度 CAS 失明修+publishguard 账本 0 字节修+OfficeComConfig/
  SettingsView office_com 桥
- zip/tar 磁盘保留不入库（P2-3 纪律）
