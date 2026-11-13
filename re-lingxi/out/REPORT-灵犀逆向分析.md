# WPS 灵犀 1.3.36 逆向分析报告（办公板块）

> 目标：本机 `C:\Users\wubi\AppData\Local\Programs\lingxi-desktop`（WPS 灵犀 / lingxi-desktop 1.3.36，Kingsoft）
> 目的：为「我们的办公板块」做能力蒸馏，先建立事实基线。
> 证据等级：`观察`=直接读到文件/日志/字符串；`推断`=结构推导未运行验证；`未知`=未取证。

---

## 0. 结论摘要

1. 灵犀是 **Electron 外壳 + 独立 Go agent 后端（app-server）+ Python 技能运行时** 的三段式架构，办公能力几乎全部落在 app-server 与 skills 目录，不在 Electron 主进程里。
2. 办公板块在 UI 层只有 **5 个主题**：`partner` / `word-agent` / `excel-agent` / `ppt-agent` / `professional-mode`；每个主题对应一套推荐卡片（prompt 模板预置）。
3. 办公能力由 **11 个官方技能 + 5 个内置技能** 承载，全部是「SKILL.md（给模型的说明书）+ Python CLI（给工具的手）」双层结构，`skills/License.txt` 为 **MIT**，是本项目最直接可复用的资产。
4. 文档生成的技术选型很关键：**DOCX 走服务端 transcoder（写 JS 代码 → 服务端生成 DOCX）**，XLSX 走 openpyxl + 纯 Python 公式重算引擎，PPTX 走 python-pptx + HTML 幻灯片双路线，PDF 走 pypdf/pdfplumber/pypdfium2。
5. **渲染/视觉验收**统一走本机 `wpscli` 代理（`doc/sheet/ppt/pdf export --format png`），xlsx 另有自研 `lbo-cli` 原生渲染器，pptx 有云端截图对比（check-visual 拼网格）。
6. 云文档走 kdocs 原生 API（v7/v5/v3 三套域名并存），并有 `kdocs-cli.exe` 作为 600+ action 的命令桥。
7. 沙箱：Windows 侧建独立本地账户（`LingxiOnline`）跑工具，`wpscli` broker 只认该账户 → 这也是 `wpscli --help` 报「不支持非金山应用调用本程序」的根因（`观察`+`推断`）。

---

## 1. 目标识别与产物清单（观察）

| 项 | 值 |
|---|---|
| 产品名 / 版本 | WPS 灵犀 / `lingxi-desktop` 1.3.36 |
| 安装目录 | `%LOCALAPPDATA%\Programs\lingxi-desktop`（1.2 GB） |
| 卸载注册表 | `HKCU\...\Uninstall\685ecaa5-...`，DisplayName=灵犀 |
| 主可执行 | `WPS 灵犀.exe`（214 MB，Electron） |
| Web 包 | `resources/app.asar` 270 MB → 解包 26 622 文件 / 518 MB |
| 前端构建 | `webBuildVersion 26b61d0a2d85`，builtAt 2026-11-05T12:13:47Z |
| 后端 | `app_server_5.24.0`，Go，`lingxi.exe` 72 MB，commit `bb60347a`，build 2026-11-05 |
| 技能包 | `app_server_5.24.0/skills`（MIT）+ `builtin_skill` |
| 用户数据 | `%APPDATA%\WPS 灵犀`（local-db、scoped-store、logs、python-env、sandbox-runtime） |
| 其它原生件 | `wpscli.exe`(8 MB)、`kdocs-cli.exe`(7.6 MB)、`lingxi-native-host.exe`、`lingxi-wpscli-proxy.exe`、`lingxi-sandbox-runner.exe`、`lingxi-sandbox-setup.exe` |

asar 解包口径（自研解析，非 `asar` CLI）：文件头 `headerSize=6938964`，数据区起点 `8+headerSize`，条目 offset 相对数据区。

---

## 2. 总体架构

```
┌──────────────────────────────────────────────────────────────┐
│ Electron 主进程 (out/main/index.js 2.0 MB, minified)          │
│  · 532 个 IPC 通道（preload 常量表）                          │
│  · 生命周期：python-env 解压 → 沙箱账户 → 拉起 app-server      │
│  · IAB 内置浏览器 HTTP 127.0.0.1:56305                        │
│  · local-file 协议 / lingxi:// deeplink / scoped-store 加密   │
└───────────────┬──────────────────────────────┬───────────────┘
     stdio JSON-RPC (protocol 2026-08-13)      │ HTTP
                │                              │
┌───────────────▼────────────────┐   ┌─────────▼────────────────┐
│ app-server: lingxi.exe (Go)    │   │ 云侧 lingxi.wps.cn       │
│  · agent run / session / queue │   │  /api/aioffice/v1/*      │
│  · MCP client (stdio+http)     │   │  wss://lingxi.wps.cn     │
│  · skills 加载 / 工具执行       │   │  cowork.wps.cn trace     │
│  · Windows 沙箱账户隔离         │   │  api.wps.cn / kdocs.cn   │
└───────────────┬────────────────┘   └──────────────────────────┘
                │ spawn (沙箱账户 LingxiOnline)
┌───────────────▼──────────────────────────────────────────────┐
│ Python 运行时 %APPDATA%\WPS 灵犀\python-env (30 306 文件)      │
│  skills/{docx,xlsx,pptx,pdf,wps-docs,wps-collab,...}/scripts  │
│  外部二进制：wpscli / kdocs-cli / lbo-cli / ripgrep / git      │
└──────────────────────────────────────────────────────────────┘
```

渲染层：`desktop-renderer-dist`（Vue3 + Vite，1 019 个 assets，Vue Router 非标准用法）注册为 `/_desktop/lingxi/` HTTPS handler，`apiOrigin=https://lingxi.wps.cn`。

---

## 3. 通信协议（观察，来自 app-server 日志）

`initialize` 请求（客户端 → 服务端）：

```json
{"protocol_version":"2026-08-13",
 "client_info":{"name":"wps-lingxi-desktop","version":"1.3.36"},
 "capabilities":{"bash_approval":false,"skill_management":false,
                 "timer_task":true,"auth_refresh":true},
 "device_id":"<sha256>","device_name":"EVO-X3"}
```

服务端 `server_info`：`{"name":"lingxi-app-server","version":"144","commit":"bb60347a"}`，
capabilities：`bash_approval`、`skill_management`、`timer_task`、`session_persistence`、`connectors`、`session_subscribe`、`run_steer`，以及
`supported_api_formats: ["chat_completions","responses","anthropic"]` ← 说明后端能对接三类模型 API 形态。

传输：`transport=stdio`（默认 `stdio-default` 连接）。
内部 Go 模块路径（日志泄漏，可直接用于定位行为）：
`ksogit.kingsoft.net/lingxi-ai/lingxi-app-server`、`lingxi-env@v0.0.84-release`、`lingxi-sandbox@v0.0.6-release`。

IPC 通道（preload 常量，532 条）按前缀分布：`file 57 / iab 52 / agent 52 / fileBrowser 41 / base 33 / pet 27 / window 23 / business 22 / project 21 / auth 20 / sandbox 8 / cowork 8 / skills 2 / kdocs 1` 等。
关键家：`agent:*`（session/queue/ws/steer/stop）、`business:getSkills|installSkillFromUrl|packSkillZip`、`skills:getContent|setEnabled`、`sandbox:*`、`cowork:*`、`kdocs:getUserInfo`。

**后端 RPC 方法（`观察`，index.js 与 lingxi.exe 双向命中，详见 `lingxi-appserver-re-report.md`）**

`session/{start,list,messages,subscribe,unsubscribe,delete,title_updated,pinswitch}` · `run/{start,steer,interrupt,error}` · `tool.call` · `skill/{list_context,confirm}` · `config/{read,write}` · `auth/refresh` · `bash/auth` · `connector/{check,manage,resolve}` · `canvas/edit` · `immemory/compact` · `timer/task`。
传输为 **NDJSON JSON-RPC 2.0 over stdio**，启动参数仅 4 个：`--stdin --auth-storage=runtime --config <f> --data-dir <d>`（无 `--port`）。

**agent 工具清单（`观察`，`tools/*.tmpl` 编进二进制）**

`ask_user_question`、`render_widget`、`todo_write`、`create_plan`、`switch_mode`、`bash`、`exec`、`exec_auth`、`pip_install`、`execute_jupyter`、`file`、`search`、`web`、`url_fetch`、`imagesearch`、`view_image`、`generate_image`、`share_file`、`connector_exec`、`connector_manage`、`memory`、`wiki_memory`、`manage_skills`、`group_chat`、`timer_task`、`deliver_result`、`await`、`creative`、`project_insight`；代码固有标识 `fs_read_file`/`fs_write_file`/`fs_edit_file`/`fs_list_dir`/`fs_download_file`/`bash_exec_sandbox`/`bash_auth`/`await_process`。否定项：`mcp__*`、`str_replace`、`spawn_agent`、`web_fetch` 均 0 命中（该版本不用这些命名）。

---

## 4. 办公板块（本报告主体）

### 4.1 UI 层：5 个办公主题

`OfficeBaseView` 暴露 theme 枚举（`ExcelCase.vue_*` 内）：

| theme | 语义 | 背景色板（渲染参数） |
|---|---|---|
| `partner` | 通用问答/助手 | #55C2FF / #CC88FF / #FF9543，opacity .2 |
| `word-agent` | 文字/文档 | #678DFF / #7479F6 / #A0FFE6 |
| `excel-agent` | 表格 | #FDFF84 / #D7FFC5 / #BDFFEB，opacity .4 |
| `ppt-agent` | 演示 | #FFDE65 / #FFBB7C / #FF8585 |
| `professional-mode` | 专业模式 | 同 partner |

首页问候语硬编码：`AI办公` + `用灵犀`。推荐卡片（`Z5` 数组）预置 3 个 case，其中两条是办公场景，含完整 prompt 与示例文件（如「按需完善文档细节：帮我用红色字填上正确答案」+ `人教版小学四年级数学（下册）期末测试卷.docx`，「审阅论文并给出修改建议」+ KANO 模型论文 .docx）——说明 **demo 走 file_id 直连云文档**。

办公相关页面组件（`desktop-renderer-dist/assets`，命名即职责）：

- 文档容器：`PAiofficePage`(210 KB)、`PWordPage`、`PExcelPage`、`PPPTPage`/`PPPTPageV2`/`PPPTPageLegacy`、`POfficePageLayout`、`OfficeBaseView`
- 画布/预览：`PCanvasContainerWeboffice`、`PCanvasContainerDraft`、`PCanvasHeader`、`PHtmlpptContainer`、`PImagepptContainer`、`PFileHtmlPreview`、`PArtifactFilePreview`
- 办公对话：`PAiofficeSharePopover`、`PCoworkAgentChat`、`PCoworkWorkbenchPanel`、`PDocSpace*`
- PPT 模板市场：`PPTAgentTemplateListV2`、`PPTTemplateListV2`、`PAIPPTTemplateCard`
- 技能市场/管理：`PSkillHub`、`PSkillMarket`、`PSkillDetailShell`、`POpenSourceSkills`、`PSkillRiskConfirmModal`

### 4.2 技能层：11 个办公/生产力技能

| 技能 | 定位 | 入口 |
|---|---|---|
| `docx` | 建/改/渲染 Word | `docx.sh` / `docx_cli.py` |
| `xlsx` | 建/改/审计/渲染 Excel | `xlsx_cli.py`(build/recalc/audit) + `lbo_cli.py`(render) |
| `pptx` | 建/改/模板复刻/讲稿/渲染 | `pptx_cli.py`（28 子命令）+ `pptx.sh` |
| `pdf` | 17 项 PDF 操作 | `pdf.sh` / `pdf_cli.py` |
| `wps-docs` | 金山云文档（drive/ET/DbSheet/AP） | `wps_docs_cli.py` + `et/scripts/et.py` |
| `wps-collab` | 金山协作 IM/邮件/日程/会议 | `wps_collab.py`（Python SDK 形态） |
| `paper-writer` | 中英文论文写作（含 docx-api） | `academic_search.py`、`scientific_visualization.py` |
| `deep-research` | 深度研究（证据库/知识图谱/大纲模板） | assets 模板 + references |
| `browser` | Playwright/CDP 浏览器自动化 | `scripts/browser.py`、`cdp_http.py` |
| `image-canvas` | 图像编辑/抠图 | `cli.py`、`image_edit_client.py` |
| `stock-analysis` | 股票分析 | SKILL.md |

内置技能（`builtin_skill`）：`skill-creator`（生成新技能）、`speech-to-text`（ASR）、`visualizer-design`（ECharts/HTML-SVG/图片叠图三种可视化模式）、`web-builder`（Vite/React 静态化）、`wps-cowrite`（与 WPS 客户端协同写、含 `canvas_edit_hook`/`doc_save_hook`）。

### 4.3 各格式技术栈与关键设计

**DOCX（最值得抄的设计）**
- 生成不是本地拼 XML，而是「**模型写 JS → 服务端 transcoder 生成 DOCX**」：`POST https://lingxi.wps.cn/api/transcoder/docx/create`，`docx-helper` 仅远程 create 可用；JS 骨架 `h.sections([{role:'cover'},{role:'front'},{role:'body'}]).build()`，支持 `h.toc()`、页眉页脚、分节页码（封面无/目录罗马/正文阿拉伯）。
- 本地只做：`lint`（JS 静态校验）、`inspect`（refs/images/headings）、`edit --replace OLD=NEW [--regex --parts]`、`unpack/pack`（OOXML 部件）、`source-status/source-patch/source-undo`（JS 源 revision + sha256 CAS 原子补丁）。
- 明确不支持：批注、修订、复杂样式重构、复杂目录、域代码。

**XLSX**
- `build`（跑任务脚本）+ `recalc`（`formulas` 库纯 Python 重算，注入缓存值，扫描 `#REF!/#DIV/0!/#VALUE!/#N/A/#NAME?`）+ `audit`（公式结构审计）。
- 硬规则：公式优先、零公式错误、动态数组（FILTER/UNIQUE/SORT/GROUPBY/XLOOKUP）为默认、溢出区不得放任何内容、散点图必须走 `chart_labels.add_scatter_xy_series` + `finalize_scatter_chart`、结构化引用聚合必须写在超级表外（防循环引用）。
- 自写 `pivot_ooxml`（生成 pivotCache/pivotTable 部件）、`dynamic_array`（encode/materialize/preserve 三策略）。

**PPTX**
- 双路线：python-pptx 直接产 OOXML；HTML 幻灯片 → 云端 `html-to-pptx` / `check-layout`(static|browser|full) / `screenshot` / `check-visual`（截图拼 3×3 网格做视觉验收）。
- 模板复刻路线：`flatten`(拆页) → `shell-report`(共享元素) → `make-skeleton` → `apply-flat` → `assemble`/`hybrid-manifest`，`ooxml_flatten` 负责 EMU→pt、组合坐标重算、theme schemeClr 落地、媒体外置。
- 字体：`font_whitelist.json` ∩ 系统字体，空交集回退微软雅黑；仅支持 16:9。

**PDF**
- pypdf（结构）+ pdfplumber（文本/表格）+ pypdfium2（渲染）+ `to-md`（OCR 走 `api.wps.cn/v7/longtask/exporter`）。

### 4.4 渲染与视觉验收（三条通道）

| 通道 | 用法 | 适用范围 |
|---|---|---|
| `wpscli`（本机代理，`lingxi-wpscli-proxy.exe`） | `wpscli doc\|sheet\|ppt\|pdf export --format png [-o DIR] [--page/--sheet/--range]` | 本地客户端首选，一期仅 PNG，Broker 校验工作区范围 |
| `lbo-cli`（自研原生渲染器） | `lbo_cli.py render --outdir preview --page-mode single\|chunk` | xlsx 专用，必须先 recalc |
| 云端截图 | pptx `screenshot`/`check-visual`、`export-screenshots`、transcoder `/api/transcoder/screenshot/async` | 云端或 wpscli 不可用时 |

`wpscli --help` 实测：`错误：不支持非金山应用调用本程序。` — 调用方受限（日志显示 broker 需解析沙箱账户 `LingxiOnline` 的 SID）。

**wpscli 的真身（`观察`，二进制字符串）**：Rust 编写、`rmcp` 框架的 **MCP server**（`wpscli 0.1.0 — WPS Office document processing tools for AI Agents`，构建路径 `H:\master`），对外暴露约 30 个 `wps_*` 工具（见附录 D），并有 `src\parent_verify.rs` + `WinVerifyTrust` 做调用方 PE 签名校验 → 这就是「非金山应用」判定的实现。它还内建 Browser(CDP)、Image(OCR) 工具族。

对我们最有价值的两点：
1. 它把 WPS 的 **JSAPI** 抽象成高层工具（`wps_doc_script` / `wps_sheet_script` / `wps_ppt_script` / `wps_pdf_script` 可跑任意 JSAPI 脚本），说明「AI 操作 Office」的关键是 JSAPI 沙箱而非 OOXML 手搓。
2. MCP 协议化 → 我们的 agent（gaea）可以把它当 MCP server 接（需解决 parent_verify，或自研等价 MCP server 走 COM/JSAPI）。

### 4.5 云文档层

- 鉴权：cookie `wps_sid`（+ `TMP_LX_UUID` 优先），csrf 同值，域限 `.kdocs.cn`/`.wps.cn`，公共头 `Origin/Referer=https://365.kdocs.cn`；出站做 SSRF + 逐跳重定向校验。
- 域名分层：`api.wps.cn`(v7)、`365.kdocs.cn/3rd/drive`(v5)、`www.kdocs.cn/api/v3/office`、`lingxi.wps.cn`（transcoder 与 lingxi 业务）。
- 文件类型路由：`.xlsx/.ksheet`→ET，`.dbt`→DbSheet，`.otl`→AP（智能文档），链接形态 `https://365.kdocs.cn/l/{file_id}`。
- `kdocs-cli.exe v2.7.1` 是命令桥：`drive 52 / sheet 49 / otl 6 / dbsheet 69 / form 4 / wpp 126 / aippt 1 / wps 287 / pdf 11 / kwiki 9` actions，支持 `auth set-token`（系统密钥链）、`--token/--endpoint/--output text|markdown|json|raw`。
- 默认落盘位置：`我的云文档/应用/WPS灵犀/<会话标题>`。

### 4.6 沙箱与原生桥

- Windows：创建本地账户 `LingxiOnline`，`lingxi-sandbox-setup.exe` 安装/重置密码，`lingxi-sandbox-runner.exe` 以该账户执行工具；`state/sandbox-runtime` 存运行时状态；`password_check=ok sandbox_account_exists=true → state=ready`。
- `lingxi-native-host.exe`：native messaging host（浏览器扩展 ↔ 客户端）。
- 沙箱首次安装耗时 23.2 s（`duration_ms=23211`）。

---

## 5. 对「我们的办公板块」的蒸馏建议

### 5.1 可直接复用（低风险）
1. **skills 全量**：`skills/` 与 `builtin_skill/` 均为 MIT（`Copyright (c) 2026 灵犀`），保留版权声明即可整体移植；关键价值在 SKILL.md 的**约束工程**（公式优先、零错误、视觉验收、白名单字体、失败即终止）。
2. **CLI 契约**：docx/pdf/pptx/xlsx 的子命令与参数设计已成体系，可直接作为我们工具层接口规约（见 §4.3 与附录命令表）。
3. **文档质量门禁**：`lint → build → recalc/audit → render → 视觉验收` 这条流水线，是「AI 出文档不翻车」的核心，值得原样搬。

### 5.2 需要替换（依赖金山服务）
| 依赖 | 说明 | 替代方案 |
|---|---|---|
| `lingxi.wps.cn/api/transcoder/docx/create` | DOCX 靠服务端 JS→DOCX | 自建 LibreOffice/UNO 或 python-docx 深度封装；或本地 `docx` 模板 + OOXML 拼装 |
| `wpscli` | 本机 WPS 渲染，带调用方校验 | LibreOffice headless（`soffice --convert-to png`）或 OnlyOffice |
| `lbo-cli` | 自研 xlsx 渲染 | LibreOffice Calc 导出 PDF→PNG |
| `kdocs` v7/v5/v3 + `kdocs-cli` | 云文档 CRUD | 自建对象存储 + 文档服务，或对接其它云文档 |
| `cowork.wps.cn` trace、`api.wps.cn` OCR | 遥测/OCR | 本地 ASR/OCR（PaddleOCR 等） |

### 5.3 最小可行落地顺序
1. 建 skill 运行时骨架：`<skill>/SKILL.md + scripts/ + $SKILL_PATH 注入`（灵犀用 `SKILL_PATH`/`SKILL_PATH_BASH` 环境变量注入技能根路径，脚本内 `import` 同目录模块）。
2. 接一个 agent 后端（任意 MCP/OpenAI 兼容），把 skills 注册为工具。
3. 打通 xlsx 闭环（本地库齐全：openpyxl + formulas + Pillow），先做表格（无外部服务依赖）。
4. 打通 pdf 闭环（pypdf/pdfplumber/pypdfium2 全本地）。
5. docx/pptx 先做「本地生成 + LibreOffice 渲染验收」，后期再评估要不要引入服务端 transcoder。

---

## 6. 未解问题 / 下一步

- `lingxi.exe` 的完整 RPC 方法表与工具 schema（子代理进行中，字符串级取证）。
- app-server 如何选择模型（hero-badge 显示 `DeepSeek V4`，`supported_api_formats` 三形态），需要抓实际 `run` 请求。
- 前端路由未用 `createRouter`（疑自研视图状态机），办公板块的页面切换契约待确认。
- `config.json`/`auth.json` 为 safeStorage 加密，需在应用运行时读取方可解密（本次未做动态调试）。

---

## 附录 A：sar 解包与工具链（可复现）

```bash
# 头部解析 + 清单
node asar_head.js <app.asar>      # → out/asar-header.json
node asar_list.js                 # → out/files.tsv（26622 行）
# 选择解包（offset 基准 = 8 + headerSize）
node asar_extract.js <app.asar> out out/x
node asar_extract.js <app.asar> desktop-renderer-dist out/drd
```

## 附录 B：关键取证文件

- `%APPDATA%\WPS 灵犀\logs\app\app.2026-11-13.1.log`（启动、python-env、沙箱、initialize 原文）
- `%APPDATA%\WPS 灵犀\logs\app-server\app-server_2026-11-13.log`（Go 侧启动、模块路径、沙箱）
- `%APPDATA%\WPS 灵犀\app_server_5.24.0\skills\*`（技能与 CLI）
- `re-lingxi/out/channels.txt`（532 IPC 通道）
- `re-lingxi/out/files.tsv`（asar 全量清单）

---

## 附录 C：与 gaea 办公板块的差距（本仓库现状对照）

gaea 现有（`观察`，`internal/`）：
- `internal/app/office_com_tools.go`：`office_open / office_read / office_edit / office_render / office_save / office_close`，COM 驱动本机 Word/Excel/PPT，`render` 输出 PNG（单页 / `overview` 6 页拼图 / `chart` 单图，宽度 600–2000），并明确「用 vision 工具读取 PNG」闭环。
- `internal/office/{docxedit,pptxedit,xlsxedit,xlsxpreview,comhelper}`：Go 原生 OOXML 编辑 + C# COM helper + xlsx 预览。
- `internal/docmd`（含 `soffice.go`）：文档转 Markdown / LibreOffice 转换。
- `internal/gaea/skill`：技能框架（builtins/gongwen/cad_takeoff/pipeline/parallel），`skills/` 目录目前仅 `story-deslop`、`story-long-write` 两个。

| 能力 | 灵犀 | gaea | 蒸馏动作 |
|---|---|---|---|
| docx 编辑 | Python + 服务端 transcoder | COM + 原生 OOXML（不丢格式） | 保留 gaea 路线；抄「生成→渲染→vision 复核」门禁到 docx 技能 |
| xlsx | openpyxl + `formulas` 重算 + audit | COM + xlsxedit + xlsxpreview | **新增 recalc/audit 门禁**（公式错误扫描）成本最低、收益最高 |
| pptx | python-pptx + HTML 幻灯片 + 模板复刻流水线 | COM + pptxedit | 抄 `flatten→make-skeleton→assemble` 模板复刻思路；字体白名单 ∩ 系统字体 |
| pdf 全操作 | 17 子命令（合并/拆分/水印/OCR/表格） | 无（仅 docmd 转换） | **整包移植** `skills/pdf`（MIT，纯本地库） |
| 云文档 | kdocs v7/v5/v3 + kdocs-cli 600+ action | 无 | 视需要：自建云盘接口，或先不做 |
| 协作 IM/邮件 | wps-collab | 无（仅微信 beta） | 按需 |
| 深度研究/论文 | deep-research + paper-writer | 无 | 直接移植 SKILL.md 与模板（纯文本资产） |
| 浏览器自动化 | Playwright/CDP skill | 无 | 移植脚本层 |
| 技能市场 | business:installSkillFromUrl / packSkillZip / skills:setEnabled | 有 skill 框架但无市场 | 移植 zip 打包/安装契约 |
| 沙箱 | Windows 独立账户 LingxiOnline | 未见同等隔离 | 按需评估 |

**最小蒸馏包（建议顺序）**：`pdf` → `xlsx`(recalc/audit/chart_labels) → `pptx`(flatten/模板) → `deep-research`/`paper-writer`（纯文档资产）→ `browser`。

---

## 附录 D：wpscli MCP 工具面（Rust / rmcp，观察自二进制字符串）

**Word**：`wps_doc_create`、`wps_doc_read`（what: full/outline/metadata/tables/paragraph，target.find 搜索并返回字符区间）、`wps_doc_edit`、`wps_doc_check`（拼写/格式/完整性）、`wps_doc_compare`（两文档差异）、`wps_doc_fill`（`{{placeholder}}` 模板填充）、`wps_doc_export`（txt/html/md/png 逐页）、`wps_doc_script`（自定义 JSAPI）
**Sheet**：`wps_sheet_create`（from_json/from_csv）、`wps_sheet_read`（range/used_range/charts/all_sheets/formula）、`wps_sheet_edit`（value 或 formula）、`wps_sheet_chart`（xlColumnClustered/xlLine/xlPie）、`wps_sheet_clean`（trim/dedup/remove_empty）、`wps_sheet_export`（csv/png/md）、`wps_sheet_script`、`wps_spreadsheet`（legacy script）
**PPT**：`wps_ppt_create`、`wps_ppt_read`（slide/shapes/notes）、`wps_ppt_edit`（slide+shape+text，或 slide+image+pos(cm)）、`wps_ppt_export`（pdf/html/png/md）、`wps_ppt_watermark`、`wps_ppt_script`、`wps_presentation`（legacy）
**PDF**：`wps_pdf_read`（full/page/metadata/bookmarks/fields）、`wps_pdf_edit`（merge/split/watermark/bookmark）、`wps_pdf_form`（列/填表单域）、`wps_pdf_search`（页内字符偏移）、`wps_pdf_extract`（"1-3,5"）、`wps_pdf_annotate`（高亮/FreeText）、`wps_pdf_export`、`wps_pdf_script`
**其它**：`wps_image_extract`（OCR：png/jpg/bmp）、Browser(CDP) 工具族、`check`（WPS 安装与连接检查）

设计要点：高层工具优先，`*_script` 作为逃生舱；PDF 坐标原点在左下；sheet 范围导出 PNG 用于视觉验收；`wps_doc_read` 返回字符区间以支撑「读→定位→精确插入」闭环。

---

## 附录 E：「直接编辑 WPS」的两条链路（重点）

### E1 本地 WPS 文档 —— wpscli → JSAPI 引擎

- 入口：`resources/wps-cli/wpscli.exe`（Rust + `rmcp`），由 PATH 注入供技能调用（`pdf/render` 等），实际调用方是 app-server 的沙箱进程。
- 机制：**把 JSAPI 脚本送进 WPS 引擎执行**，不是外部拼 OOXML。证据：
  - 二进制内含业务 JS 片段：`...1Application.ActiveDocument.Content.Find.Execute('`、`rng.Find.Execute("__FIND__", false, false, __USE_REGEX__, ..., "__REPLACE__", 2)`、`while (rng.Find.Execute() && guard < 10000)`；
  - 字符串 `kwpsjsrunner`（WPS 的 JS runner）、`src\jsapi\pdf_scripts.rs`（Rust 侧内置 PDF JS 脚本）；
  - `wps_status` 工具自述：`Check WPS Office installation, engine connectivity, file version (> 12.1.0.27880), and API authorization`；
  - 内部 API 名：`kwpsapiex`（advanced JSAPI 预载）、`DocumentEx.ConvertToMarkdown`、`WorkbookEx.ConvertToMarkdown`、`Presentation.ConvertToMarkdown`；`PageText(index)` 为 0-based；PDF 坐标原点左下。
- 双模：每个脚本类工具都有 `ui` 参数（`Run with WPS UI visible (headed mode)`），默认 headless → **同一套工具既能后台批量，也能把 WPS 窗口推到用户面前逐字编辑**。
- 调用方校验：`src\parent_verify.rs` + `WinVerifyTrust`（PE 签名链），因此 `wpscli --help` 直接返回「不支持非金山应用调用本程序」。另有 `lingxi-wpscli-proxy.exe` 作 broker，日志显示它需要解析沙箱账户 `LingxiOnline` 的 SID。

**broker 链路（`观察`，见子报告 `lingxi-appserver-re-report.md` §5）**

```
Electron ──stdio JSON-RPC──> lingxi.exe (app-server)
                                  │ spawn + LINGXI_WPS_BROKER_TOKEN
                                  ▼
                        lingxi-wpscli-proxy.exe  (Go, internal/wpsclibroker)
                                  │ 命名管道 \\.\pipe\lingxi-wps-broker-<pid>-<str>
                                  │ go-winio，帧上限检查（broker frame is too large）
                                  ▼
                            wpscli.exe (Rust, MCP server)
                                  │ JSAPI
                                  ▼
                            WPS 引擎 (wps.exe 12.1.0.29690)
```

要点：**脚本工具链不是每次直连 WPS，而是走一个带 token 的本地 broker**；broker 同时承担「跨沙箱账号转发」——这正是它必须解析 `LingxiOnline` SID 的原因。我们的等价实现可以直接照抄这个分层：`gaea` 里一个 `wps-broker`（命名管道 + token）→ 一个 MCP/JSON-RPC 工具服务 → COM/JSAPI。

沙箱执行器另有独立管道 `\\.\pipe\lingxi-runner-<hex>`，由 `lingxi-sandbox-runner.exe`（Rust，源自 Codex CLI 的 Windows 沙箱）实现，帧协议含 `command/cwd/env/permission_profile/workspace_roots/timeout_ms/tty/stdin_open`。
- 本机环境满足门槛：WPS `12.1.0.29690`（> 27880），COM 已注册 `KWPS.Application` / `Ket.Application` / `KWPP.Application`。

### E2 云文档协同 —— wps-cowrite

- 触发：对话轮次携带 `<wps_docs_context>` / `<wps_docs_selection>`（用户在 WPS 里划词/打开文档）。
- 执行：`wps_docx.py` 用**语义动作词表**写入 —— `texts.content/count/range/search/replace/style/structure/font/font_batch/highlight/format/format_batch`、`styles.data`、`sections.count`、`header_footers.header_content|footer_content|*_link_previous`。
- 落盘：`doc_save_hook.save_document()` 强制引擎持久化（`https://api.wps.cn`，可用 `LINGXI_CANVAS_SAVE_API_BASE` 覆盖），`canvas_edit_hook.begin_canvas_edit` 标记编辑会话。
- 定位：ET/PPTX 当前不支持协同，回落到 `xlsx` / `pptx` 技能。

### E3 对我们的意义

真正的能力差不在「能不能改 WPS」，而在三件事：
1. **语义动作层**：把「改第 3 段开头、把金额列标红、整份换个字体」这类意图映射成稳定 API，而不是让模型猜底层参数。
2. **逃生舱**：`wps_*_script` 让模型在高层工具覆盖不到时直接写引擎脚本（等价于 gaea 里加 `office_script`）。
3. **选区上下文**：用户划词即改，靠 `<wps_docs_selection>` 这种结构化上下文注入完成，不需要模型自己找位置。
