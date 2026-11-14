# Gaea 办公板块 CAD/进度计划右栏化 预览包

**包标识：** `gaea-office-cadtab-20261114-c85a9815`

**性质：** 功能提交预览，不是正式版本号发布（版本三源维持对齐不抬号；
`wails.json` / `app_info.go` / `versioninfo.rc` 无漂移）。

## 构建来源

- 提交：`c85a9815`（本专项六刀全量，基线 `9fd5296c`）：
  - `e273577b` `feat(office-ui): CAD 图纸与进度计划回归右栏一级 Tab，办公板块可直接打开 CAD`
  - `369c7b83` `polish(office-ui): CAD 右栏面板单形态收口+文案归一+mock 图纸 fixture`
  - `ecef7bd6` `polish(office-ui): CAD 直达链收口——cadExt 叶模块+快照净化+树内走查实证`
  - `94ea2348` `polish(office-ui): 顶栏「工具」下拉退役——CAD/进度计划已是右栏一级 Tab`
  - `2d863e40` `polish(office-ui): CAD/进度计划视图激活自动加宽——每会话一次性不抢拖拽偏好`
  - `c85a9815` `docs(gaea): progress.md 记 CAD/进度计划右栏专项（行级补丁只含本刀段落）`
- 构建方式：独立干净工作树检出 `c85a9815`（`.worktrees/cadtab-head`），
  pnpm install 17.6s → `npm run build`（Vite 38.19s，dist 落仓根）→ 裸
  `wails build -s -ldflags "-s -w" -trimpath`（54.8s），bindings 未跳过。
- 构建工具：Wails `2.13.0`、UCRT64 GCC（`C:\msys64\ucrt64\bin` 前置）、
  `CGO_ENABLED=1`、系统默认 TMP/TEMP。
- 构建完成：2026-11-14 02:08:17（Asia/Shanghai）。
- exe sha256：`E65681B63B2D0EC027EC952AF8B756AF138DF3C3773B2A6F5F0B5E6A9EAED741`
  （与装机位 `C:\AI\gaea-app\gaea.exe` 逐字节同源——包产物即装机产物，单一
  产物链）。

## 冒烟

- `scripts/gaea_install.ps1 -SourceDir <worktree>\build\bin` → `INSTALL OK`
  （exe + 三 DLL 同装 `C:\AI\gaea-app`，sha256 打印一致）。
- `scripts\smoke.ps1 -ExePath C:\AI\gaea-app\gaea.exe` → `/api/health` **200**
  （`{"status":"ok"}`，冒烟进程 PID 40776 已回收）。
- 二进制标记：`grep -ac GaeaCadOpen` ×3 命中（CAD 绑定面在位）。

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
6. **测试**：cadOpen 5 例 + cadExt 间接 + paneTabs 净化 2 例 + registry 集成
   （产物行 dwg→CAD 路由）+ palette/workspaceTabs/sidebarRegistry 契约改口径；
   mock PickFiles 纯 CAD 过滤回 fixture（浏览器 mock 全链可演示）。

## 已知边界

- **DSH context/fleet 批次在途未入包**：主工作树 51 文件未提交改动（上下文
  交互续刀）不属本专项，按规约七不混入包；安全快照见
  `gaea-uncommitted-20261114.tar.gz`（留盘不入库）。
- **真机实弹挂用户**：真实 dwg（天正导出件）解析质量、问图纸效果需真实引擎
  侧验证；mock fixture 仅证明链路。
- CAD 图纸为可选卡（defaultEnabled:false，master-plan「可选使用」红线）：
  首次经管理开关/＋下拉/文件打开链启用。

## 包内文件

| 文件 | 说明 | 入库 |
|---|---|---|
| `PACKAGE-INFO.md` | 本文件 | ✅ |
| `SHA256SUMS-c85a9815.txt` | exe+三 DLL 校验和 | ✅ |
| `gaea.exe` + 三 DLL | 装机同源产物 | ❌（.gitignore） |
| `gaea-c85a9815-source.tar.gz` | 提交源码包（git archive） | ❌（留盘） |
| `gaea-uncommitted-20261114.tar.gz` | 打包时刻未提交工作树安全快照 | ❌（留盘） |
