# Gaea 图文板块工具梳理·功能打包

**包标识：** `gaea-wenxue-tools-20261115-0eb7a2fc`

**性质：** 功能提交打包（非正式版本号发布，`productVersion` 不变，以提交号标识）。
覆盖 2026-11-15 图文板块工具梳理专项全部批次（调研/计划 + 七笔功能/修复提交 +
三笔记档），构建自独立干净工作树，主工作树零未提交内容（`git status --porcelain`
为空，全部工作均已入库）。

## 构建来源

- 打包提交：`0eb7a2fc`（HEAD）
  `docs(wenxue): 定稿/阅读刀+网页版实测修复记档`
- 功能提交区间：`83d4cad6`（刀0 死面清理）→ `63d33adf`（刀0b 预览收缩）→
  `a7997cb9`（刀1 案A 独立引擎）→ `79532dbb`（刀2 技能子集）→ `2514d3eb`
  （刀3 专属工具四族）→ `bac474cd`（刀4 模板小说化）→ `5a17f731`（locale
  死键清扫）→ `722df07b`（模型中心文学功能绑定）→ `2408e382`（角色标签页）
  → `d282025b`（角色/阅读 tab 白名单修复）
- 构建工作树：`C:\AI\wubigrok\.worktrees\pkg-0eb7a2fc`（detached HEAD，
  `git status --porcelain` 为空）
- 前端依赖：junction 复用主工作树 `frontend\node_modules`
- 构建链：`frontend` `npm.cmd run build`（26.8s）→ 仓根
  `wails build -s -ldflags "-s -w" -trimpath`（50.6s，Skip Bindings=false）
- 构建环境：Windows amd64、UCRT64 GCC、`CGO_ENABLED=1`、
  `CC=C:\msys64\ucrt64\bin\gcc.exe`、系统默认 `TMP/TEMP`
- 构建后 `git diff --quiet -- frontend/wailsjs` 通过（29 项 status 为 CRLF
  显示假象，内容零漂移）
- DLL：预构建第三方件，取自主工作树 `build/bin`（与历次装机同源同哈希）

## 本包内容（相对上一包 404679be）

1. **图文独立引擎（案A）**：`wx` 控制器（Cwd=当前小说目录、独立 wire seq、
   事件三分线 `gaea-wenxue-event/-subagent-text/-ready`）+ boot
   `ToolFilter/SkillFilter/SkillExtras` 三缝（双 nil=办公逐字节现状）+
   办公域工具 deny 名单 + 43 个会话态 `Wenxue*` 绑定 + 聊天流映射全面重指。
2. **图文专属工具十二件**：章节三件/角色三件/时间线两件/统计/定稿三件
   （`wenxue_final_*` 仅定稿目录，`wenxue_chapter_write` 域隔离拒写定稿）。
3. **技能面子集**：allowlist 五面同源收口 + 内嵌三件
   （story-long-write/story-deslop/wenxue-workbench）+ read_skill 双引擎
   串仓缝实例化根修。
4. **模型中心文学专属功能绑定**：`wenxue` 功能键端到端（配置五件套/卡片
   腾位/双引擎 Build 位点注入纪律）。
5. **前端**：办公死面清理（memoryhub 全目录等）+ 预览链收缩（docx 编辑保留）
   + 主区「角色」「阅读」两个新 tab（角色卡管理与定稿阅读，均与对应工具
   同源）+ 任务模板小说化 + 模型中心绑定卡 + 视图 tab 白名单单源化修复。
6. 办公板块零影响：boot 三缝双 nil 逐字节现状、`gaeaExtraTools` 不含
   `wenxue_*`（有断言钉死）、办公文件仅 2 处加性跟随重建钩子。

## 验证门禁

- 前端全量 vitest：532 文件 / 4901 例全绿（331.98s）
- Go：`internal/app` + `internal/gaea/...` 全量绿；config/app 定向功能绑定
  测试绿；绑定漂移 OK@817；locale 死键清零；tsc -b 0；eslint 0 错误
- **网页版实测**：vite dev `:5218/?mock=1` + 浏览器驱动——文学板块
  「角色」「阅读」tab 跳转/渲染/往返 + 截图留证（修复
  `handleChatTabSelect` 视图白名单漏新 id 的跳转失效）
- 冒烟：`releases/.../payload/gaea.exe` `/api/health` 200

## 产物（SHA256 见 SHA256SUMS-0eb7a2fc.txt）

- `payload/gaea.exe` — `bac42ee4de7e9849974cf844cff66af1fccc21a37286496f909477e83f9a932d`
- `payload/onnxruntime.dll` — `68d0d5f5…`
- `payload/sherpa-onnx-c-api.dll` — `5dd36dbd…`
- `payload/sherpa-onnx-cxx-api.dll` — `4c563277…`
- `gaea-0eb7a2fc-preview.zip` / `gaea-0eb7a2fc-source.tar.gz`（留盘不入库）

## 装机记录

- 主工作树同源构建装机两次（中间批 E073EE0C… / 最终批与本包同码）：
  最终装机即本包代码（PID 40640 / 冒烟 200）；本包 exe 为独立干净树重建，
  哈希 `bac42ee4…`（与主树非 clean-tree 构建的 `A0717380…`/`F5AF2C6B…`
  不同属预期——构建路径差异，代码同源）。
