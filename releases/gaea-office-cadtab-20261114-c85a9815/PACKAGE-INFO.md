# Gaea 办公 CAD/进度计划右栏化 + dsh-context 续刀 预览包

**包标识：** `gaea-office-cadtab-20261114-c85a9815`（v2，纳入 dsh-context 批后
构建提交升为 `2eb8d9fd`；目录名保留首发提交号作包系 ID）

**性质：** 功能提交预览，不是正式版本号发布（版本三源维持对齐不抬号；
`wails.json` / `app_info.go` / `versioninfo.rc` 无漂移）。

## 构建来源

- 提交：`2eb8d9fd`（基线 `9fd5296c`），两批全量：
  - **办公 CAD/进度计划右栏化六刀**（`e273577b` → `c85a9815`）：
    - `e273577b` `feat(office-ui): CAD 图纸与进度计划回归右栏一级 Tab，办公板块可直接打开 CAD`
    - `369c7b83` `polish(office-ui): CAD 右栏面板单形态收口+文案归一+mock 图纸 fixture`
    - `ecef7bd6` `polish(office-ui): CAD 直达链收口——cadExt 叶模块+快照净化+树内走查实证`
    - `94ea2348` `polish(office-ui): 顶栏「工具」下拉退役——CAD/进度计划已是右栏一级 Tab`
    - `2d863e40` `polish(office-ui): CAD/进度计划视图激活自动加宽——每会话一次性不抢拖拽偏好`
    - `c85a9815` `docs(gaea): progress.md 记 CAD/进度计划右栏专项（行级补丁只含本刀段落）`
  - **dsh-context 0.84.1 上下文交互续刀**（`2eb8d9fd`，用户拍板代提交，+5042/−1401，
    51 文件）：趋势总览四分位标尺+分类点击；浏览器七分类独立搜索/排序/逐类
    delta；DNA 搜索；system/tools 节点展开系统提示词与逐工具 schema；Context
    Stats 下钻接通（浏览器分类/文件活动四类筛选/事件 kind/命令 program）；命令行
    展开配对工具结果；工具来源插件 chip；Go contextview 新模块 command_page/
    dna/surface + detail/fold/types 扩面 + agent_stream header 通道 + registrymap
    注册面 + OfficeB 绑定面再生；WorkbenchPrefsPanel.test 文案对齐 locale
    （任务视图→编队视图）。
- 构建方式：独立干净工作树检出 `2eb8d9fd`（`.worktrees/pkg-2eb8d9fd`），
  pnpm install 16.8s → `npm run build`（Vite 26.52s）→ 裸
  `wails build -s -ldflags "-s -w" -trimpath`（1m01.4s），bindings 未跳过。
- 构建工具：Wails `2.13.0`、UCRT64 GCC（`C:\msys64\ucrt64\bin` 前置）、
  `CGO_ENABLED=1`、系统默认 TMP/TEMP。
- 构建完成：2026-11-14 08:22:35（Asia/Shanghai）。
- exe sha256：`BFD7CAA2E100A7324420139B7A19E88A4FA009E1CED7EEDF3E414C0B0C02D3F3`
  （与装机位 `C:\AI\gaea-app\gaea.exe` 逐字节同源——包产物即装机产物）。

## 门禁与冒烟

- `go build ./...` + `go vet`（contextview/app）0；`go test ./internal/gaea/contextview/...` ok；
  `tsc -b` 0；mock 契约 / 设置面板 / 上下文页测试绿（WorkbenchPrefsPanel 4 例红
  系 DSH 文案改名未跟上测试，已随 `2eb8d9fd` 修齐，5/5 绿）。
- `scripts/gaea_install.ps1 -SourceDir <worktree>\build\bin` → `INSTALL OK`
  （exe + 三 DLL 同装 `C:\AI\gaea-app`）。
- `scripts\smoke.ps1 -ExePath C:\AI\gaea-app\gaea.exe` → `/api/health` **200**
  （冒烟进程 PID 54044 已回收）。

## 本包增量（自上一包 gaea-officecom-20261113-f5576986 / exe 8ec56ed2 起）

1. **CAD 图纸/进度计划升右栏一级 Tab**（用户拍板）：ToolSheet 临时面板退役；
   入口=欢迎网格卡 / ＋下拉（未启用标注点击即启用）/ Ctrl+K 命令面板 /
   文件打开链 / 输入框旁工具钮。顶栏「工具」下拉随 Tab 化退役。
2. **办公板块可直接打开 CAD**：文件树/交付卡/Git 变更行点 dwg/dxf → 自动
   启用可选卡 → 右栏 CAD 面板装载解析（lib/cadOpen 路由 + 请求总线，
   CadPanel nonce 一次性消费）；pdf 仍走文档预览链。
3. **面板单形态收口**：CadPanel 清退 sheet/inspectorHost/portal 三机制，
   cad-panel.css 重写为右栏唯一纵向布局（净 −156 行）。
4. **激活自动加宽**：切到 CAD/进度计划自动加宽到 560（视口封顶），每会话
   一次性，用户拖窄不回弹。
5. **文案归一**：三语言「CAD 看图→CAD 图纸」（en 补真译），临时面板语义改
   右栏一级标签。
6. **dsh-context 0.84.1 上下文交互续刀**（见构建来源；已知边界：桥仍 live
   surface，历史步骤节点未下发，工具来源插件 chip 待收口）。

## 已知边界

- **真机实弹挂用户**：真实 dwg（天正导出件）解析质量、问图纸效果需真实引擎
  侧验证；mock fixture 仅证明链路。
- CAD 图纸为可选卡（defaultEnabled:false，master-plan「可选使用」红线）：
  首次经管理开关/＋下拉/文件打开链启用。
- v1 首发包（exe E65681B6，仅 CAD 六刀）与本 v2 包同目录演进：v2 为当前
  权威（exe/校验和/源码包/preview zip 均已替换为 2eb8d9fd 构建）。

## 包内文件

| 文件 | 说明 | 入库 |
|---|---|---|
| `PACKAGE-INFO.md` | 本文件 | ✅ |
| `SHA256SUMS-2eb8d9fd.txt` | exe+三 DLL 校验和 | ✅ |
| `gaea.exe` + 三 DLL | 装机同源产物 | ❌（.gitignore） |
| `gaea-2eb8d9fd-source.tar.gz` | 提交源码包（git archive） | ❌（留盘） |
| `gaea-2eb8d9fd-preview.zip` | exe+三 DLL 预览 zip | ❌（留盘） |
