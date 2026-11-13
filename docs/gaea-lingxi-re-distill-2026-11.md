# 灵犀逆向蒸馏（2026-11-13）

> 目标：本机 WPS 灵犀 `lingxi-desktop 1.3.36`（app-server `5.24.0`，commit `bb60347a`，构建 2026-11-05）。
> 性质：调研批，未动 gaea 业务代码。本文是灵犀侧唯一事实源；配套落地计划见 `docs/gaea-lingxi-optimize-plan-2026-11.md`。
> 证据分级：`[观]`=文件/日志/二进制直接命中，`[推]`=结构推断未运行验证，`[未]`=未取证。
> 原始产物：`re-lingxi/out/REPORT-灵犀逆向分析.md`、`re-lingxi/out/SKILLS-办公能力清单.md`、`re-lingxi/out/文件交付卡片-调研.md`、`re-lingxi/out/GAEA-直接编辑WPS-落地方案.md`、`re-lingxi/lingxi-appserver-re-report.md`、`re-lingxi/out/files.tsv`、`re-lingxi/out/channels.txt`。

---

## 0. 结论卡

**值得蒸馏，但不是抄 UI。** 灵犀的办公板块价值集中在三处，且都不在 Electron 层：

1. **技能包与约束工程**：`skills/`、`builtin_skill/` 共 16 个技能，`License.txt` 为 **MIT（Copyright (c) 2026 灵犀）**，是唯一可整包移植的资产。真正的含金量不是脚本，而是 SKILL.md 里那套「公式优先 / 零公式错误 / 视觉验收 / 入口互斥 / 失败即停」的质量门禁 `[观]`。
2. **交付契约**：文件交付不是卡片数据结构，而是「markdown 链接 + 伪协议（`file:///`、`sandbox:///`）+ 前端协议改写」三级组合；IM 场景单独走 `deliver_result` 工具 `[观]`。
3. **JSAPI 通道形态**：`wpscli` 是 Rust+rmcp 的 **MCP server**，把 WPS 的 JSAPI 抽象成 30 个 `wps_*` 工具（含 `*_script` 逃生舱、`ui` 双模），经 `lingxi-wpscli-proxy` 命名管道 broker 转发给 WPS 引擎 `[观]`。这是「在用户打开的 WPS 里真编辑」的完整参考实现。

**不建议抄的**：5 个 UI 主题配色、云文档 API 体系、Windows 沙箱账户方案、遥测与埋点。它们的收益都在 gaea 既有能力之下或依赖金山服务。

**与本仓库的关系**：gaea 的 COM helper（`internal/office/comhelper/OfficeHelper.cs`，8.6k 行）来自 dsh-office v1.2.0 移植，op 面（word 21 / excel 24 / ppt 19 类）**比灵犀的 20 个语义 action 更宽**。故灵犀不能替代我们的实时编辑主线，只能补三处缺口：逃生舱、选区上下文、交付契约。

---

## 1. 目标指纹

| 项 | 值 |
|---|---|
| 安装目录 | `%LOCALAPPDATA%\Programs\lingxi-desktop`（1.2 GB） |
| Web 包 | `resources/app.asar` 270 MB → 26 622 文件 / 518 MB |
| 前端构建 | `26b61d0a2d85`，2026-11-05T12:13:47Z（Vue3 + Vite，1 019 assets） |
| 后端 | `lingxi.exe` 72 MB，Go，`lingxi-app-server`，块号 `bb60347a` |
| 技能 | `%APPDATA%\WPS 灵犀\app_server_5.24.0\skills`（11）+ `builtin_skill`（5），MIT |
| 原生件 | `wpscli.exe`(Rust MCP, 8 MB)、`kdocs-cli.exe`(v2.7.1, 600+ action)、`lingxi-native-host.exe`、`lingxi-wpscli-proxy.exe`、`lingxi-sandbox-runner.exe`（Rust，源自 Codex CLI）、`lingxi-sandbox-setup.exe` |
| 本机环境 | WPS `12.1.0.29690`（>wpscli 门槛 12.1.0.27880），COM `KWPS/Ket/KWPP.Application` 已注册 |

---

## 2. 架构

```
Electron 主进程 (out/main/index.js 2.0 MB)
  · 532 IPC 通道；python-env 解压(30 306 文件)→沙箱账户→拉起后端
  · IAB 内置浏览器 127.0.0.1:56305；local-file 协议；lingxi:// deeplink
        │ stdio NDJSON JSON-RPC 2.0
        ▼
lingxi.exe (app-server, Go)  ──HTTP──> lingxi.wps.cn / api.wps.cn / cowork.wps.cn
  · session/run/tool.call/skill/connector；MCP client（stdio + streamableHTTP）
  · 启动参数仅 4 个：--stdin --auth-storage=runtime --config <f> --data-dir <d>
        │ spawn（沙箱账户 LingxiOnline）
        ▼
python-env（openpyxl/python-docx/python-pptx/pypdf/pdfplumber/formulas/playwright…）
  skills/{docx,xlsx,pptx,pdf,wps-docs,wps-collab,…}/scripts/*.py
        │      │
        │      └── wpscli(MCP) ──\\.\pipe\lingxi-wps-broker-<pid>-<rand>──> WPS 引擎(JSAPI)
        └── kdocs-cli（云文档 REST 桥）
```

`initialize` 握手（`[观]`，来自 app-server 日志）：
`protocol_version=2026-08-13`、`client_info={wps-lingxi-desktop,1.3.36}`、`capabilities={bash_approval,skill_management,timer_task,auth_refresh}`；
服务端返回 `{lingxi-app-server,144,bb60347a}` + `supported_api_formats=["chat_completions","responses","anthropic"]`。

RPC 方法（`[观]`，index.js 与 lingxi.exe 双向命中）：`session/{start,list,messages,subscribe,unsubscribe,delete,title_updated,pinswitch}`、`run/{start,steer,interrupt,error}`、`tool.call`、`skill/{list_context,confirm}`、`config/{read,write}`、`auth/refresh`、`bash/auth`、`connector/{check,manage,resolve}`、`canvas/edit`、`immemory/compact`、`timer/task`。

agent 工具（`[观]`，`tools/*.tmpl` 编进二进制）：`file`、`bash`、`exec`、`exec_auth`、`pip_install`、`execute_jupyter`、`search`、`web`、`url_fetch`、`imagesearch`、`view_image`、`generate_image`、`share_file`、`render_widget`、`todo_write`、`create_plan`、`switch_mode`、`ask_user_question`、`deliver_result`、`await`、`memory`、`wiki_memory`、`manage_skills`、`group_chat`、`timer_task`、`creative`、`project_insight`、`connector_exec/manage`。
否定项（0 命中）：`mcp__*`、`str_replace`、`spawn_agent`、`web_fetch`。

---

## 3. 资产地图（按可复用度分级）

### S 级：整包可拿（MIT）

| 资产 | 规模 | 价值 |
|---|---|---|
| `skills/*` + `builtin_skill/*` | 271 文件 | SKILL.md 约束 + Python CLI 双层结构；pdf/xlsx/pptx/docx 全套 |
| 技能工程纪律 | 见 §4 | 质量门禁，可直接写进我们的提示词与工具设计 |

技能清单：`docx` `xlsx` `pptx` `pdf` `wps-docs` `wps-collab` `deep-research` `paper-writer` `browser` `image-canvas` `stock-analysis`；内置 `skill-creator` `speech-to-text` `visualizer-design` `web-builder` `wps-cowrite`。
命令面见 `re-lingxi/out/SKILLS-办公能力清单.md`（docx 10 / xlsx 3+3 / pptx 28 / pdf 17 / wps-docs 45+）。

### A 级：契约可抄，实现自研

| 契约 | 原文要点 |
|---|---|
| 交付链接 | 桌面 `[查看文件名](file:///绝对路径)`；Web `[文件名](sandbox:///绝对路径)`，文件夹末尾 `/` |
| 渲染陷阱 | 链接前后**不得加反引号**、不得放代码块（会被解析成代码片段导致整条消息渲染错乱） |
| 协议范围 | 伪协议只用于用户可见 markdown；工具参数仍用真实绝对路径 |
| IM 交付 | 走 `deliver_result(ref=绝对路径)`，**一次一个文件**，多文件多次调用；**禁止** `file:///` |
| 交付时序 | 交付阶段文字必须一次写完**再**调工具（工具调用会中断文字输出） |
| 语义动作 | `texts.replace/style/font_batch/highlight/format_batch`、`styles.data`、`sections.count`、`header_footers.*`（gaea op 面更宽，仅作命名参考） |

### B 级：形态可抄

| 形态 | 说明 |
|---|---|
| `wpscli` = MCP server | Rust + rmcp；30 个 `wps_*` 工具；每个脚本类工具有 `ui` 开关（headless / 带 UI）；`wps_*_script` 是逃生舱 |
| broker 分层 | `app-server → wpscli-proxy(命名管道 + `LINGXI_WPS_BROKER_TOKEN`) → wpscli`；跨沙箱账号转发 |
| 按需读指南 | SKILL.md 明确「禁止任务前通读全部 reference，按步骤读」 |
| 技能根注入 | `SKILL_PATH` / `SKILL_PATH_BASH` 环境变量，脚本内 import 同目录模块 |
| 技能安装 | `business:installSkillFromUrl` / `packSkillZip` / `skills:setEnabled`（zip 契约 + 开关） |

### C 级：仅参考，不采用

云文档 API（v7/v5/v3 三套域名 + kdocs-cli 600+ action）、Windows 沙箱账户（`LingxiOnline` + WFP + 私有桌面）、遥测埋点、5 个办公主题配色、`lingxi-sandbox-runner`（Codex CLI 派生）。

---

## 4. 工程亮点（与 gaea 直接相关）

1. **公式优先 + 零错误门禁**：xlsx 强制用公式而非 Python 算完硬编码；交付前必须 `recalc`（`formulas` 纯 Python 重算 + 注入缓存值）并扫描 `#REF!/#DIV/0!/#VALUE!/#N/A/#NAME?`。
2. **动态数组默认化**：结果尺寸可变时用 `FILTER/UNIQUE/SORT/GROUPBY/XLOOKUP`，溢出区不得放任何内容。
3. **视觉验收闭环**：生成 → 渲染 PNG → 人/模型复核；pptx 还有 `check-visual` 拼 3×3 网格。
4. **入口互斥**：新建 / 模板复刻 / 局部编辑三条路线不得混用，选错就退化。
5. **模板复刻流水线**：`flatten → shell-report → make-skeleton → apply-flat → hybrid-manifest → assemble`。
6. **字体白名单 ∩ 系统字体**，空交集回退微软雅黑（pptx 16:9 硬约束）。
7. **失败即停**：`success:false` 或 exit≠0 必须终止后续写操作；权限错误立即终止不做变通。
8. **目录纪律**：`<task>/scripts/build`、`<task>/scripts/test`，产物在任务目录根部。
9. **逃生舱**：高层工具优先，`*_script` 兜底（模型不必因缺一个 op 就放弃）。
10. **选区上下文**：`<wps_docs_context>` / `<wps_docs_selection>` 注入，划词即改。

---

## 5. 与 gaea 现状对照

| 维度 | 灵犀 | gaea 现状 | 结论 |
|---|---|---|---|
| 实时编辑 Office/WPS | JSAPI（wpscli，需签名授权） | COM（OfficeHelper.cs + `office_*` 工具） | **保留 COM 主线** |
| op 覆盖面 | 20 个语义 action | word 21 / excel 24 / ppt 19 类 | gaea 更宽，不抄 |
| 逃生舱 | `wps_*_script` | 无 | 缺 |
| 选区上下文 | 有 | 无 | 缺 |
| 渲染验收 | wpscli / lbo-cli / 云端 | `office_render` + vision（含 overview 拼图） | gaea 已具备 |
| 交付卡片 | markdown 链接 + 伪协议 | `DeliverableCards.tsx`（启发式 + 产物登记表 + 徽标） | UI 不落后，缺协议优先 |
| PDF 能力 | 17 子命令 | 仅 `docmd` 转换 | 缺 |
| 表格质量门禁 | recalc/audit | COM 直改 | 缺 |
| 技能体系 | 16 个技能 + 市场 | `internal/gaea/skill` 框架 + 2 个技能 | 框架有、内容缺 |
| MCP 暴露 | wpscli 本身即 MCP server | 未见 | 可后置 |

---

## 6. 蒸馏时必须固定的不变量

1. **SKILL.md 契约**：YAML frontmatter 必含 `name` / `description`；正文按「何时读哪份 reference」组织，不把所有内容塞进正文。
2. **技能根注入**：运行时提供根路径变量，脚本内相对 import，不硬编码绝对路径。
3. **CLI 返回契约**：stdout 单 JSON `{success, data, message}`；`success:false` 即失败；非 0 退出即失败。
4. **交付链接契约**：`file:///`（本地）/ `sandbox:///`（会话沙箱），前端识别协议直取路径；启发式只做兜底。
5. **许可头**：移植 skills 必须保留 MIT 版权声明（`Copyright (c) 2026 灵犀`）。

---

## 7. 明确不做

- 不 patch `wpscli` 的 `parent_verify` / `WinVerifyTrust`（产品边界，版本升级即失效）。
- 不引入云文档依赖（kdocs API / transcoder）作为办公主链路。
- 不搬 Windows 沙箱账户方案（gaea 当前无同等隔离需求）。
- 不抄 5 套办公主题配色与前端组件实现。

---

## 8. 风险与未知

| 项 | 级别 | 处置 |
|---|---|---|
| skills 里部分能力依赖金山服务（docx transcoder、pptx 云端排版校验、pdf OCR、lbo-cli 渲染器） | 高 | 移植时逐脚本标注外部依赖，能本地化的本地化（LibreOffice / pypdfium2），不能的标注「不可用」并从技能描述里摘掉 |
| Python 运行时体积（灵犀 python-env 解压 3 万文件） | 中 | 只装实际用到的包；或按需安装 |
| COM 与 JSAPI 能力不等价（JSAPI 能拿 WPS 新特性） | 中 | 需要时再评估「自研 WPS 加载项」通道（见落地方案 P1） |
| 灵犀后续版本变化（模板/工具改名） | 低 | 本文档按 commit 标定，升级后重跑 asar 解析脚本即可复核 |
| `[未]` lingxi.exe 是否自监听回环端口、`auth.json` 加密结构、MCP 运行时命名 | — | 需要动态调试或更高版本样本再补 |

---

## 9. 复现入口

```bash
# asar 头部与清单（自研解析，offset 基准 = 8 + headerSize）
node re-lingxi/asar_head.js "<...>/resources/app.asar"
node re-lingxi/asar_list.js
node re-lingxi/asar_extract.js "<...>/resources/app.asar" out re-lingxi/out/x
# 二进制字符串
python re-lingxi/str.py <exe> <regex> [limit]
python re-lingxi/occ.py <regex> [hits] [pre] [post]
# 技能命令面
grep -rn "add_parser(" "%APPDATA%/WPS 灵犀/app_server_5.24.0/skills"
```
