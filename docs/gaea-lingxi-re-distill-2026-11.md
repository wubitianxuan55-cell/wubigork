# 灵犀逆向蒸馏（2026-11-13）

> 目标：本机 WPS 灵犀 `lingxi-desktop 1.3.36`（app-server `5.24.0`，commit `bb60347a`，构建 2026-11-05）。
> 性质：调研批，未动 gaea 业务代码。本文是灵犀侧唯一事实源；配套落地计划见 `docs/gaea-lingxi-optimize-plan-2026-11.md`。
> 证据分级：`[观]`=文件/日志/二进制直接命中，`[推]`=结构推断未运行验证，`[未]`=未取证。
> 原始产物（A 批）：`re-lingxi/out/REPORT-灵犀逆向分析.md`、`re-lingxi/out/SKILLS-办公能力清单.md`、`re-lingxi/out/文件交付卡片-调研.md`、`re-lingxi/out/GAEA-直接编辑WPS-落地方案.md`、`re-lingxi/lingxi-appserver-re-report.md`、`re-lingxi/out/files.tsv`、`re-lingxi/out/channels.txt`。
> 原始产物（B 批，运行时与提示词，2026-11-13 续）：`re-lingxi/out/SKILLS-file-deps.json`（技能逐文件依赖矩阵）、`re-lingxi/out/SKILLS-deps.json`、`re-lingxi/out/WPSCLI-tools.md`（34 工具面）、`re-lingxi/out/prompts/sections/`（131 个提示词段）、`re-lingxi/out/prompts/region-utf8.txt`。抽取脚本：`re-lingxi/wpscli_tools3.py`、`re-lingxi/sections.py`、`re-lingxi/deps3.py`。
> 原始产物（C 批，逆向层，2026-11-13 续）：`re-lingxi/out/GO-symbols-lingxi.md`（30 717 符号 / 600 包）、`re-lingxi/out/GO-symbols-proxy.md`、`re-lingxi/out/PROMPT-tool-contracts.md`（25 工具契约）、`re-lingxi/out/WPSCLI-jsapi-scripts.md`（13 个 JS 外壳）、`re-lingxi/out/WPSCLI-jsapi-bodies.md`（129 条 JS 算子片段）。脚本：`go_symbols.py`、`tool_contracts.py`、`extract_js.py`、`extract_js_body.py`、`probe_wpscli.py`。
> 原始产物（D 批，其余四件，2026-11-13 续）：`re-lingxi/out/NATIVE-HOST.md`（CDP 桥 38 条消息原文）、`re-lingxi/out/SANDBOX.md`（WFP + ACL + 策略字段）、`re-lingxi/out/KDOCS-cli.md`、`re-lingxi/out/GO-symbols-nativehost.md`。脚本：`gen_d_batch.py`、`probe_batch4.py`～`probe_batch4d.py`。
> 原始产物（E 批，工具模板库，2026-11-13 续）：`re-lingxi/out/TOOLS-tmpl.md`（371 个 define 块 / 21 簇 / 128 KB）、`re-lingxi/out/KDOCS-wps-actions.md`（133 个 `wps.*` 动作）。脚本：`extract_tool_tmpl.py`、`probe_batch5.py`～`probe_batch5c.py`。
> 原始产物（F 批，线协议与 IPC，2026-11-13 续）：`re-lingxi/out/WIRE-json-tags.md`（2303 个字段名）、`re-lingxi/out/IPC-channels.md`（510 个通道）。脚本：`extract_wire.py`、`extract_ipc2.py`。
> 原始产物（G 批，全量清单，2026-11-13 续）：`re-lingxi/out/GO-types.md`（15 984 类型 / 1 365 包）、`GO-error-codes.md`（425 个码）、`HTTP-routes.md`（129 条）、`PROMPT-tool-contracts.md`（25 工具 / 102 参数 / 86 条中文原文）、`WPSCLI-tool-js.md`。脚本：`extract_all.py`、`extract_contracts.py`。
> 原始产物（H 批，片段归属尝试，2026-11-13 续）：`WPSCLI-js-statements.md`（213 条真 JS 语句 + 对象模型清点）、`WPSCLI-tool-js-map.md`（邻近法反例）、`WPSCLI-module-map.md`（模块法结果）、`WPSCLI-js-api-inventory.md`。脚本：`rebuild_js.py`、`build_tool_js_map.py`、`build_module_map.py`、`build_js_api.py`。
> 原始产物（I 批，骨架/记忆/占位符/流式，2026-11-13 续）：`PROMPT-skeleton.md`（130 节点装配顺序）、`MEMORY-spec.md`（72 KB）、`WPSCLI-placeholders.md`（66 个）、`STREAM-events.md`。脚本：`extract_i_batch.py`。

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

---

# B 批续批（2026-11-13 · 运行时、技能依赖、提示词库）

> 性质：**纯调研**，未动 gaea 业务代码。本批只读本机落盘态（`%APPDATA%\WPS 灵犀`、安装目录、日志），
> 静态抽取 `lingxi.exe` / `wpscli.exe` 内嵌字符串，外加一次 `wpscli.exe --help` 行为验证（结果见 §13）。
> 证据分级同上：`[观]`=直接命中，`[推]`=结构推断，`[未]`=未取证。
> 本批**解决**了 §8 中标 `[未]` 的三项中的两项半：进程/端口模型、auth 存储、MCP 运行时命名（部分）。

## 10. 一句话结论

上一批把灵犀读成「技能包 + 交付契约 + JSAPI 通道」。续批实测后要补三条更硬的事实：

1. **技能包不是自足的**：`python-env` 只预装 33 个发行包（**没有** python-docx / python-pptx / pypdf / pdfplumber / formulas / requests），
   其余由客户端按 `service\requirements.txt` **首次运行时 pip 联网安装** `[观]`。→ P1 的「Python 运行时」不是「装不装」，而是「谁装、什么时候装、断网怎么办」。
2. **wpscli 的工具面比记载的多**：34 个语义工具（首批记 30），且 `parent_verify` 实测拦截非金山父进程调用 `[观]`。op 面 gaea 不落后，**通道形态**才落后。
3. **提示词模板是明文内嵌的**：`lingxi.exe` 里带 6 个 `templates/local/*.tmpl` + 131 个 `<section>` 段 `[观]`。
   这是本批最高价值资产——交付、错误处理、提权、技能管理、工作区纪律的原文规则可直接对齐。

## 11. 运行时实况（改写 A 批的 Python 假设）

| 项 | 实测 | A 批假设 |
|---|---|---|
| Python | **3.12**（`python312.dll`），29 016 文件 / **349.9 MB** | 「解压 30 306 文件」方向对，包面记错 |
| 预装发行包 | **33 个**：cairocffi cairosvg cffi choreographer contourpy cssselect2 cycler defusedxml et_xmlfile fonttools kaleido kiwisolver logistro lxml matplotlib numpy openpyxl orjson packaging pandas pillow pip platformdirs pycparser pyparsing pypdfium2 python-dateutil simplejson six tinycss2 tzdata webencodings wheel | 记成含 docx/pptx/pypdf/pdfplumber/formulas/playwright |
| 运行时装包 | `requirements.txt` 明示双轨：标 `[预装]` 的随包发，**其余由客户端首次运行 pip 装**：python-docx python-pptx formulas pypdf pdfplumber reportlab requests aiohttp bs4 jinja2 xlrd scikit-learn rdkit pytesseract PyPDF2 httpx websockets playwright nodriver cdp-use send2trash plotly html5lib | 未记载 |
| 端口 | IAB HTTP `127.0.0.1:56305`、本地 WS 探测 `19836`；**app-server 不开 TCP**（日志 `transport=stdio`，`handleConnection connection_id=stdio-default`） | §8 标 `[未]` |
| 沙箱 | 账户 `LingxiOffline` / `LingxiOnline`，口令以 **DPAPI blob** 存 `.sandbox-secrets\sandbox_users.json`；`setup_marker.json` 里 `allow_local_binding=false`、`proxy_ports=[]` | 只记「沙箱账户」 |
| 首次启动开销 | python-env 解压 199 s → portable-node 10 s → PortableGit 37 s → 沙箱 setup 23 s（提权） | 未记 |
| 环境变量面 | `LINGXI_SESSION_ID` `WORKSPACE_DIR` `PYTHONNOUSERSITE=1` `PIP_CACHE_DIR` `REQUESTS_CA_BUNDLE`（自建 `combined-ca-bundle.pem`）`ENABLE_AUDIT=1` `BROWSER_BRIEF` | 部分记 |

**为什么重要**：灵犀的「Office 能力」在断网首启时是**不可用**的——`requests` 都没预装。
gaea 若照抄这套形态，必须显式决定「预置哪些包 + 断网降级语义」，不能默认 `pip install` 一定成功。

## 12. 技能脚本外部依赖矩阵（新增，替换 A 批「逐脚本手工标注」）

产物：`re-lingxi/out/SKILLS-file-deps.json`（ast 精确解析，逐文件）。口径：`cloud_files` = 命中云模块（kdocs/transcoder/wps_http/lbo_cli）、
云域名（`*.kdocs.cn` / `api.wps.cn` / `lingxi.wps.cn` / `ksyuncs.com`）或云凭据 env（`WPS_SID`/`WPS_API_BASE`/`KDOCS_CLI_PATH`）的文件数。

| 技能 | py | 云文件 | 缺包（python-env 未预装） | 归类 |
|---|---|---|---|---|
| docx | 12 | **3**（`docx_transcoder_client` `docx_cli` `docx_lint`） | docx, requests | 混合，主体本地 |
| xlsx | 9 | **2**（`lbo_cli` `test_lbo_cli`） | formulas | 混合，主体本地 |
| pdf | 2 | **1**（`pdf_to_md`） | pypdf, pdfplumber, requests | 混合，`pdf_cli` 17 子命令全本地 |
| pptx | 36 | **4**（`transcoder_api` `export_screenshots` `pptx_cli` `web_style_extractor`） | pptx, bs4, requests, Xlib, api, browser, kdocs | 混合，OOXML 层全本地 |
| wps-docs | 19 | **10** | jinja2, requests | 纯云（kdocs REST + WPS_SID） |
| wps-collab | 1 | 1 | bs4, requests | 纯云 |
| image-canvas | 3 | 2 | — | 纯云（AI 出图网关） |
| speech-to-text | 3 | 2 | requests, mutagen | 纯云（ASR） |
| web-builder | 2 | 2 | — | 纯云（部署到 ks3/EdgeOne） |
| wps-cowrite | 5 | 3 | — | 纯云 |
| browser | 12 | 0 | cdp_use, httpx | 本地（CDP） |
| paper-writer | 2 | 1 | aiohttp, rdkit | 本地 + 公共学术 API |
| deep-research / stock-analysis / visualizer-design | 1/0/0 | 0 | — | 纯文档（无脚本或仅 md） |
| skill-creator | 4 | 0 | yaml | 本地 |

**可直接移植的本地子集**（不碰金山服务）：
- `docx`：`create / edit / inspect / pack / restore / source-patch / source-status / source-undo / unpack`（`format=md` 导出与 lint 才走 transcoder）
- `xlsx`：`audit / build / recalc`（`recalc` 需 `formulas`；`render` 走 `lbo_cli` → ks3，属云）
- `pdf`：`pdf_cli` 全部 17 子命令（需 `pypdf` + `pdfplumber`；`pdf_to_md` 是云，别移植）
- `pptx`：`ooxml_flatten / make_skeleton / apply_flat / hybrid_manifest / assemble / pptx_merge / theme_* / font_* / pack_pptx`（`check-visual`、`export-screenshots` 走云）

**结论**：A 批担心的「逐脚本标注外部依赖」现在有了机器可复算的基线（`deps3.py`），移植时按这张表切，不要再手工判断。

## 13. wpscli 工具面：30 → 34，且 parent_verify 实测拦截

产物：`re-lingxi/out/WPSCLI-tools.md`。

- 行为验证 `[观]`：`wpscli.exe --help` → `错误：不支持非金山应用调用本程序。`，exit=1（`src\parent_verify.rs`）。
  → **动态 `tools/list` 拿不到**，schema 只能静态还原；A 批 §7「不 patch parent_verify/WinVerifyTrust」的结论继续成立，本批也未尝试绕过。
- 静态还原 `[观]`：`[描述][工具名]` 成对切段，得 **34 个语义工具 + 3 个 legacy 逃生舱**：

| 组 | 工具 |
|---|---|
| Writer/docx（8） | `wps_doc_check` `wps_doc_compare` `wps_doc_create` `wps_doc_edit` `wps_doc_export` `wps_doc_fill` `wps_doc_read` `wps_doc_script` |
| PDF（8） | `wps_pdf_annotate` `wps_pdf_edit` `wps_pdf_export` `wps_pdf_extract` `wps_pdf_form` `wps_pdf_read` `wps_pdf_script` `wps_pdf_search` |
| Presentation（6） | `wps_ppt_create` `wps_ppt_edit` `wps_ppt_export` `wps_ppt_read` `wps_ppt_script` `wps_ppt_watermark` |
| Spreadsheet（7） | `wps_sheet_chart` `wps_sheet_clean` `wps_sheet_create` `wps_sheet_edit` `wps_sheet_export` `wps_sheet_read` `wps_sheet_script` |
| 通用（5） | `wps_status` `wps_search` `wps_file_info` `wps_image_extract`（OCR） `wps_browser`（CDP） |
| legacy 逃生舱（3） | `wps_writer` `wps_spreadsheet` `wps_presentation`（任意 JSAPI 脚本） |

- 参数词汇（工具间复用）：`file` / `output` / `script` / `script_file` / `ui`（默认 headless，`ui:true` 起真 WPS 界面）/ `format` /
  `what`（full·page·metadata·bookmarks·fields·tables·outline）/ `target`（find·paragraph·range·slide·shape·placeholder）/
  `data`·`from_text`·`from_json`·`from_csv` / `range`·`chart_type`·`operations`。
- **对照结论**：灵犀多出来的只有 `wps_doc_check`（校对）、`wps_doc_compare`（文档比对）、`wps_file_info`/`wps_search`、`wps_image_extract`（OCR）、
  `wps_browser`。gaea 的 `office_*` op 面（word 21 / excel 24 / ppt 19 类）依旧更宽——**不缺 op，缺逃生舱**。

## 14. 提示词模板库（本批最高价值资产）

`lingxi.exe` 内嵌 Go `text/template`：`templates/local/{agents_main_v2, agents_persona, agents_project, memory_migrate, model_specific, skills_main_v2}.tmpl`
+ `templates/tools/{connector_exec, deliver_result, generate_image}.tmpl`；`{{ template }}` 引用还出现 `Prolog / Epilog / StatusTable / bootstrap / connectors / memory / skills_list / skills_local / skills_main_v2_selection / skills_web` `[观]`。

按 `<section>…</section>` 自定界抽出 **131 个段、443 KB** 明文 → `re-lingxi/out/prompts/sections/<tag>.md`（同名多版本按长度降序保留）。
高频段：`<system_reminder>`(33 版) `<instructions>`(12) `<tool_use>`(8) `<im_scene>`(6) `<sharing_files>`(6) `<delivery_format>`(5)
`<skills_instructions>`(5) `<memory_system>`(4) `<error_handling>`(4) `<response_ending>`(4)。

对 gaea 可直接对齐的规则原文（摘）：

| 段 | 原文要点 |
|---|---|
| `delivery_format` | 协议按客户端分流：web=`[文件名](sandbox:///绝对路径)`、本机=`[查看文件名](file:///绝对路径)`；文件夹路径末尾加 `/`；**「链接和图片语法必须直接输出，前后不要加反引号，也不要放进代码块——反引号会被解析成代码片段，导致整条消息渲染错乱」**；「伪协议仅用于用户可见的 Markdown 链接；调用 file、bash、view_image 等工具时，参数仍使用真实绝对路径」 |
| `sharing_files` | IM 场景**必须** `share_file`（交付物是云文档链接，不要在正文重复粘贴，失败不要重试、不要编造）；**唯一例外**是 `.png/.jpg/.jpeg/.webp` 预览图可直接 `![说明](sandbox:///绝对路径)`；`.pptx/.xlsx/.docx/.pdf` 等最终文件禁止 `file:///`；同一文件只能交付一次 |
| `response_ending` | 有价值才给一句后续建议；**禁止**「如果您需要更多帮助，请随时告诉我」「希望这对你有帮助」等套话 |
| `skills_instructions` | 「**强制规则：在执行任务前，必须先读取相关的 SKILL.md**」；触发条件按文档类型映射技能；无匹配技能则跳过 |
| `skill_management_rules` | 「**禁止使用 uninstall + install 组合来『刷新』**」；official 技能不可改；清单截断时用 `skill(action="list")` 复核 |
| `error_handling` | 同一操作最多 3 次；**登录墙/验证码/Cloudflare 这类认证拦截不属于「换个方法」的范围**，必须交人处理（`ask_user_question` 或直接说明），不得改用搜索绕开 |
| `escalation_rules` | 提权用 `sandbox_permissions="require_escalated"` + **面向用户的 `justification`（「是否允许下载项目依赖以运行测试？」）**；不要先问一句再调；提权重试仍失败则停 |
| `workspace_and_files` | 访问文件前先看路径链上的 **`AGENTS.md` / `CLAUDE.md` / `RULES.md`**；深层规则不覆盖上层；受限档默认在工作目录产出副本、不主动申请回写原件 |

→ gaea 侧可直接落地的五条：反引号禁令、`sandbox:///` 与工具参数分离、认证墙交人、提权 justification 措辞、`AGENTS.md` 链式读取纪律。

## 15. 鉴权与配置存储（§8 的 `[未]` 已解）

| 位置 | 内容 | 加密 |
|---|---|---|
| `app-server-data\auth.json` | `{"auth_mode":"wps_sid","wps_sid":{"value":""}}` —— **明文**，登录后写入 WPS SID | 无 |
| 环境变量 | `LINGXI_WPS_SID` 可覆盖（日志：`not logged in (use 'login' command or set LINGXI_WPS_SID)`） | — |
| `%APPDATA%\WPS 灵犀\config.json` | 整文件二进制（首字节 `f6 fa f5 19 …`），**非** `enc:v1:` 文本 | 整文件加密 |
| `app-server-data\config.json` | 明文：`sandbox.*`（bash/python/node/kdocs/wpscli 路径、`browser_in_app_port:56305`）、`compact.cache_expire_snip`、`tracing.endpoint=https://cowork.wps.cn/api/aioffice/v1/trace/report` | 无 |
| `auth-binding-key.v1` | 100 字符 base64，配合 scopedStore 做设备绑定 | — |
| scopedStore（`index.js`） | 值级加密：`enc:v1:<nonce b64>:<tag b64>:<ct b64>`，`aes-256-gcm`（nonce 12 / tag 16 / key 32）；主密钥 = `sha256("lingxi-desktop:scoped-store:v1\0" + machineId)`；有 `safeStorage` 时主密钥再被 DPAPI 包一层 | AES-256-GCM + DPAPI |

**建议（给 gaea）**：要抄就抄**值级 `enc:v1` + 机器绑定派生密钥**这套（可迁移、可缺省降级），不要抄整文件二进制加密（不可读、排障困难）。

## 16. MCP 连接器体系（§8 的「MCP 运行时命名」部分解）

- 运行时：`github.com/modelcontextprotocol/go-sdk@v1.6.1`（`mcp/server.go` `mcp/streamable.go` `jsonrpc/`）`[观]`。
- 传输：**stdio（`command`）+ http（`streamable_http`）**；错误模板原文：「mode=http 时必须指定 endpoint 参数（MCP Server 的 URL）。http 模式的传输类型: 支持streamable_http, 不支持sse」`[观]`。
- 工具命名：**不扁平化**。二进制里 `mcp__` 命中 0，工具统一走单一入口 `connector_exec`（`connector_exec: missing id`），工具名以 `server/tool` 形态传给连接器（`%s/%s`）`[观+推]`。
- 连接器清单（`connectors\catalog.json`，20 个）：qq_mail / netease_mail / custom_mail / ifind / wind / amap / tencent_map / tdx / edgeone_makers / baidu_map / zhihuiya / hundsun_juyuan / zsxq / yingmi / tencent_wj / qixin_huiyan / **dingtalk / feishu / wecom / tencent_meeting**（后四个是 `auth_type: cli`——走 `npm install -g <cli>` + device 登录）。
- **对 gaea P2-1 的修正**：若把 `office_*` 暴露成 MCP，选「单 dispatcher + `server/tool` 参数」而不是「每个工具一个 `mcp__server__tool` 名字」——后者会把工具清单炸开，且与灵犀的 `connector_exec` 形态不一致。

## 17. 顺带观察：安全分类标签

`lingxi.exe` 里有一组注入/越权分类标签（各命中 1 次，`[观]`）：
`系统提示泄露`、`恶意输出处理`、`mcp_tool_poisoning`、`agent_snooping`、`remote_script_exec`、`write_block_device`、`full_access_worker`、`prompt_injection`。
配合 `<instruction_boundaries>`(12 处) 与 `BASH_FUNC_wpscli` 环境清洗，可见灵犀把「工具投毒 / 代理窥探 / 设备写入」当成独立分类在管。
gaea 侧可借鉴的是**分类名 → 处置动作**的映射表形态，而不是具体词表。

## 18. 对 A 批的修订

| A 批结论 | B 批实测 |
|---|---|
| wpscli 30 个 `wps_*` 工具 | **34 个语义 + 3 legacy**；且 `--help` 被 parent_verify 拦 |
| python-env 含 docx/pptx/pypdf/pdfplumber/formulas/playwright | **只预装 33 包**，上述全部运行期 pip；断网首启不可用 |
| 「逐脚本标注外部依赖」（手工） | 已机器化：`SKILLS-file-deps.json` + `deps3.py` 可复算 |
| `[未]` 端口 / auth / MCP 命名 | 端口与 auth 已解；MCP 命名解到「单 dispatcher + server/tool」 |
| 提示词纪律只有零星摘录 | 131 段全文落盘，交付/错误/提权/技能管理四类规则可直接对齐 |

## 19. 剩余未知

- `lingxi.exe` 在云电脑 / 桌宠 / IAB 等特定模式下是否另开监听 `[未]`（本机为纯本地未登录态，未跑到那些分支）。
- `kdocs-cli` 的 600+ action 全清单 `[未]`（属 C 级，暂不投工）。
- 沙箱 `wpscli-proxy` 的命名管道 token 派生与 WFP 规则细节 `[未]`。
- 本机 `safeStorage.getSelectedStorageBackend()` 实况（`dpapi` 还是 `basic_text`）`[未]`；若是 `basic_text`，scopedStore 会降级为机器绑定密钥——需实测确认。

---

# C 批续批（2026-11-13 · 逆向层：二进制结构、符号、内嵌脚本）

> 性质：**纯调研**。本轮目标是"结构级"证据，不是行为级复现：没有运行 lingxi.exe / wpscli，
> 没有改任何二进制，也没绕过 `parent_verify`。
> 环境限制：Ghidra 未配置（`binary_session` → `ghidra: not_configured`，缺 `GHIDRA_INSTALL_DIR`），
> 所以**没有伪代码级证据**。本轮用四条可用证据线：Go 构建元数据、Rust 元数据+串、PE 导入表、pclntab 符号与 .rodata 串。
> 产物：`re-lingxi/out/GO-symbols-lingxi.md`、`GO-symbols-proxy.md`、`PROMPT-tool-contracts.md`、
> `WPSCLI-jsapi-scripts.md`、`WPSCLI-jsapi-bodies.md`。脚本：`go_symbols.py`、`tool_contracts.py`、`extract_js.py`、`extract_js_body.py`、`probe_wpscli.py`。

## 20.1 构建身份（REA Evidence 级）

| 件 | sha256 | 语言/工具链 | 构建坐标 |
|---|---|---|---|
| `lingxi.exe` | `0b789ef5b259b770fd59eddbfb876a843ac8bc4ce52bf8405d89ac5f0fed0620` | **Go 1.26.2** | module `ksogit.kingsoft.net/lingxi-ai/lingxi-app-server/cmd/lingxi`，main `v0.0.0-20261105113224-bb60347a9f48+dirty`，**162 依赖 / 12 build settings** |
| `wpscli.exe` | `796966c196bac24a74eb691608c96564f71985a18bf2bcbdd48ec0259adf65e6` | Rust（vendored 35 crate） | **无** cargo-auditable 段；rustc commit 仅能推断 |
| `lingxi-wpscli-proxy.exe` | `fc5878e3f64c817a1766f4c959cb53dcd6cff2d1da416ee17602d4fe7f22be42` | **Go**（A 批误记 Rust） | 子命令 `cmd/wpscli-broker-poc`，包 `internal/wpsclibroker` |

## 20.2 App-server 依赖面（162 个，按用途分组）

| 用途 | 依赖（版本） |
|---|---|
| MCP / schema | `modelcontextprotocol/go-sdk v1.6.1`、`google/jsonschema-go v0.4.3`、`yosida95/uritemplate v3.0.2` |
| 存储 | `modernc.org/sqlite v1.59.0`（纯 Go SQLite）、`go-redis/redis/v8 v8.11.5`、`go.etcd.io/etcd/client/v3 v3.6.6` |
| 对象存储（四云齐备） | `aws/aws-sdk-go v1.55.8`、`aliyun/aliyun-oss-go-sdk v3.0.2`、`tencentyun/cos-go-sdk-v5 v0.7.71`、`huaweicloud/…-obs v3.25.9`、`minio/minio-go/v6 v6.0.57` |
| 邮件连接器 | `emersion/go-imap/v2 v2.0.0-beta.8`、`emersion/go-message v0.18.2`、`emersion/go-sasl`、`go-mail/mail v2.3.1` |
| Git / SSH | `go-git/go-git/v5 v5.19.2`、`xanzy/ssh-agent v0.3.3`、`skeema/knownhosts v1.3.1` |
| Shell 门控 | `mvdan.cc/sh/v3 v3.11.0`（bash 命令解析/审批） |
| 文档/渲染 | `yuin/goldmark v1.7.17`（markdown）、`disintegration/imaging`、`golang.org/x/image` |
| 令牌/加密 | `golang-jwt/jwt/v4 v4.5.2`、`lestrrat-go/jwx/v2 v2.0.9`、`ProtonMail/go-crypto v1.1.6`、`cloudflare/circl v1.6.3` |
| 可观测 | `go.opentelemetry.io/otel v1.44.0`、`prometheus/client_golang v1.23.2`、`go.uber.org/zap v1.27.0` |
| 金山内部 | `lingxi-agents v0.0.195-release`、`lingxi-env v0.0.84-release`、`lingxi-sandbox v0.0.6-release`、`open-std/kso v0.0.39`、`yunsdk/vassdk-go/v2 v2.2.0` |

**读法**：交付/连接器/沙箱/记忆/存储全在主进程内用 Go 实现（含 SQLite、IMAP/SMTP、四云对象存储、go-git），
Python 只是 Office 脚本执行器。这解释了为什么 `requirements.txt` 那么薄——非 Office 能力压根不走 Python。

## 20.3 包结构（pclntab 符号级）

`lingxi.exe` 命中 **30 717 个符号 / 600 个包**，其中本仓相关：
`…/core`(563)、`…/internal`(207)、`…/core/connectors`(101)、`…/core/store`(66)、`…/core/connectors/mail`(55)、`…/core/sandbox`(13)；
沙箱 SDK 三包合计 837 个符号（`lingxi-sandbox/sdk` 325、`…/windows_sandbox` 405、`…/sandboxing` 107）。

`lingxi-wpscli-proxy.exe` 命中 75 个符号 / 11 包，`internal/wpsclibroker` 下只有 8 个：
`RunProxy`、`RunProxy.deferwrap1`、`brokerPipeName`、`forwardedEnvironment`、`readFrame`、`writeFrame`、`writeAll`、`writeProxyUserError`。

## 20.4 工具契约层：`prompt.*` 416 个导出 → 25 个注册工具

`lingxi-agents` 把「工具描述 / 参数描述 / 解析失败响应」做成**导出的 Go 符号**，等于把工具 schema 摊开在符号表里。
`PROMPT-tool-contracts.md` 里逐工具列了参数名。抽样：

| 工具 | 参数 |
|---|---|
| `File` | Action, Path, Content, Edits, EditFind, EditReplace, EditAll, StartLine, EndLine, Brief |
| `TimerTask` | Action, TaskID, Title, Prompt, Schedule, Enabled |
| `CreatePlan` | Action, Name, Overview, Plan, Todos, TodoID, TodoContent, Edits, EditFind, EditReplace, EditAll |
| `ConnectorManage` | Action, ID, Mode, Name, Command, Endpoint, Headers, Description, Brief |
| `Bash` | Command, RunInBackground, **SandboxPermissions**, Justification, Brief（超时走 `Await` 的 Timeout） |
| `ShareFile` | Paths, Message ／ `ShareFileCloudDoc` 为独立第二契约 |

**双轨发现**：`deliver_result` / `exec_auth` **没有** `*ToolDesc` 符号 → 它们的描述走 `templates/tools/*.tmpl`；
而 `share_file` 反而有完整 `prompt.*` 契约（含 `ShareFileCloudDocUnavailableResp`）。
即灵犀的工具描述是 **prompt 包 + tmpl 文件** 两条路并行，不是单一来源。

## 20.5 wpscli 内部结构

- vendored crate 35 个：`rmcp`、`tokio`、`tokio-util`、`schemars`、`clap_builder`、`tracing-subscriber`、`regex`、`serde(_core/json)`、`chrono`、`parking_lot`、`aho-corasick`、`uuid` …
- 自有模块 56 个 `.rs`：`main` `server` `service` `dispatcher` `schema` `params` `auth` `doctor` `bridge` `parent_verify` `pdf_scripts` `encoding` `path` …
- **13 个内嵌 JS 外壳 + 129 条 JS 算子片段 + 66 个占位符**（`__RANGE__` `__SHEET_ACCESS__` `__PH_INDEX__` `__PAGES_PER_FILE__` …）。
- JS 外壳的语义（原文，`WPSCLI-jsapi-scripts.md` 第 7 段）：

```javascript
var want = filePath.replace(/\//g, "\\").toLowerCase();
for (var i = 1; i <= Application.Documents.Count; i++) {
  var d = Application.Documents.Item(i);
  if ((d.FullName || "").toLowerCase() === want) { doc = d; break; }
}
if (!doc) doc = Application.Documents.Open(filePath, false, false, false);
var __result = (function(doc) { __BODY__ })(doc);
__SAVE_AND_CLOSE__
```

**这就是「在用户已打开的文档上真编辑」的实现要点**：先按 `FullName` 命中已打开实例（拿到用户当前视图/选区），
命中不了才 `Open`；改完立刻 `Save / SaveAs` + `Close`，并把结果 `JSON.stringify` 回传。

## 20.6 broker 协议（A 批的「命名管道」到这里落地）

- `internal/wpsclibroker`：`RunProxy` → `brokerPipeName` → `readFrame`/`writeFrame`（+`writeAll`）、
  `forwardedEnvironment`（转发环境到子进程）、`writeProxyUserError`（失败写回用户可读错误）。
- 类型：`brokerRequest` / `brokerResponse`（带 `json:"error,omitempty"`）→ **JSON 帧**。
- 帧长守卫原文：`invalid broker frame size: %d`、`broker frame is too large: %d bytes` → **长度前缀帧**。
- 传输：`github.com/Microsoft/go-winio`（命名管道），凭据走环境 `LINGXI_WPS_BROKER_TOKEN` / `LINGXI_WPS_BROKER_PIPE`。
- PE 导入表只有 kernel32 基础集（48 个），无 `CreateNamedPipeW`——管道能力由 go-winio 动态解析，这点与符号层一致。

## 20.7 对 A/B 批的修订

| 原结论 | 本批 |
|---|---|
| `lingxi-wpscli-proxy.exe` 是 Rust（源自 Codex CLI） | **Go**，`cmd/wpscli-broker-poc`，go-winio + JSON 帧 |
| 工具 30 个（tmpl 命中） | tmpl 轨 30；`prompt.*ToolDesc` 轨 **25**；两轨不同源 |
| wpscli 只是「30 个 wps_* 工具」 | 另有 **13 个 JS 外壳 + 129 条 JS 算子 + 66 占位符** 的模板机制 |
| 沙箱是外部 runner | `lingxi-sandbox` 以 837 个符号**编进主进程**；`lingxi-sandbox-runner.exe` 只是伴生件 |

## 20.8 边界与剩余未知

- **无伪代码**：Ghidra 未配置，本轮全部是元数据/符号/串级 `[观]`，没有函数体级 `[推]`。
  要出伪代码需先设 `GHIDRA_INSTALL_DIR` 并重启 MCP 客户端（REA 只在启动时读该变量）。
- `rizin` / `objdump` 通道本轮**未跑**（可用，但元数据+符号已够回答本轮问题）。
- 未做：`prompt.*` 函数体与 .rodata 文案的逐条配对（文案已在 §14 段库，配对留待需要时）。
- 未做：`kdocs-cli.exe` 的 action 表、`lingxi-native-host.exe` 的 CDP 桥、沙箱 WFP 规则细节。
  → **已在 §21（D 批）补上**：native-host 协议、沙箱策略字段与 WFP/ACL、kdocs-cli 形态。

---

# D 批续批（2026-11-13 · 其余四件：native-host / 沙箱两件 / kdocs-cli）

> 性质：纯静态盘点，未运行任何服务端件。工具链已辨认：`lingxi-native-host.exe` 与 `lingxi-wpscli-proxy.exe` 是 **Go**，
> `lingxi-sandbox-{setup,runner}.exe` 是 **Rust（Codex 派生）**，`kdocs-cli.exe` 是 **Go 1.23.3 单模块**。
> 产物：`re-lingxi/out/NATIVE-HOST.md`、`SANDBOX.md`、`KDOCS-cli.md`、`GO-symbols-nativehost.md`；生成脚本 `gen_d_batch.py`。

## 21.1 五件工具链一览（修正 A 批）

| 件 | 大小 | 语言 | 构建坐标 | 角色 |
|---|---|---|---|---|
| `lingxi.exe` | 68.9 MB | Go 1.26.2 | `…/cmd/lingxi` | 主进程（stdio JSON-RPC） |
| `lingxi-native-host.exe` | 6.7 MB | **Go** | `…/cmd/native-host`，包 `internal/nativehost` | 浏览器/CDP 桥 |
| `lingxi-wpscli-proxy.exe` | 2.6 MB | **Go** | `…/cmd/wpscli-broker-poc`，包 `internal/wpsclibroker` | 命名管道 broker |
| `lingxi-sandbox-setup.exe` | 14.6 MB | **Rust**（Codex） | `codex_execpolicy` / `codex_otel` / `codex_home` | 沙箱账户/WFP/ACL 置备 |
| `lingxi-sandbox-runner.exe` | 8.2 MB | **Rust**（Codex） | `codex_protocol::permissions` | 沙箱内执行器 |
| `kdocs-cli.exe` | 7.3 MB | **Go 1.23.3** | module `kdocs-cli` `(devel)`，**仅 3 依赖** | 云文档 REST 客户端 |

> A 批把 `native-host` 记为「CDP 桥」但没定语言，把 `sandbox-runner` 记为「Rust，源自 Codex CLI」——后者**成立**（`codex_protocol` 等符号实锤）。

## 21.2 native-host：本地 HTTP + 插件长连 + CDP 转发

包 `internal/nativehost` 的 31 个导出符号把生命周期讲清了：

`ListenWithCleanup` / `StartServer` / `EmbeddedServer` / `PortFilePath` / `writePortFile` /
`KillStaleNativeHost` / `queryPortPids` / `queryProcessExePath` / `killProcess` /
`NewStdinIdleMonitor` / `StdinIdleMonitor` /
`marshalNM` / `UnmarshalNM` / `ReadNM` / `WriteNM` / `withBodyLimit` /
类型：`CDPRequest` `CDPResponse` `NMInbound` `NMOutbound` `StatusResponse` `PortCleanupResult` `CDPError`。

消息面（原文，`NATIVE-HOST.md` 有全 38 条）：

| 类 | 消息 |
|---|---|
| 会话 | `[NH-MSG] session ready: %s`、`[NH-MSG] session ready (legacy, no session_id)`、`[NH-CDP] session_end: session=%s group=%s reason=%s`、`[NH-MSG] session_end from plugin: session=%s reason=%s` |
| CDP 转发 | `[NH-CDP] sending cdp.command: id=%d method=%s session=%s brief=%s`、`[NH-MSG] cdp.event: method=%s session=%s`、`[NH-MSG] cdp.response: id=%v session=%s`、`[NH-CDP] waiting for cdp.response: id=%d method=%s`、`[NH-CDP] got cdp.response: id=%d method=%s elapsed=%v` |
| 插件 | `[NH-MSG] Plugin connected (status: accepted)`、`[NH-MSG] Plugin status: state=%s`、`[NH-MSG] Plugin error: code=%s message=%s session=%s fatal=%v` |
| 心跳 | `[NH-HB] ping send failed: %v`、`[NH-HB] plugin unresponsive (last pong %v ago), marking disconnected`、`[NH-HB] plugin reconnected (pong received after disconnect)` |
| 拒绝/超时 | `[NH-CDP] rejected: plugin not connected`、`[NH-CDP] rejected: session ended by plugin: %s reason=%s`、`[NH-CDP] rejected unhealthy session: session=%s reason=%q`、`[NH-CDP] TIMEOUT waiting for cdp.response: id=%d method=%s elapsed=%v`、`[NH-CDP] TIMEOUT waiting for session_ready: …` |
| 清理 | `native-host: write port file %s: %w`、`[native-host] port %d written to %s`、`[native-host] WARN: failed to send session_start: %v`、`stdin idle for %v, exiting to prevent zombie pr…`、`[port-cleanup] failed to kill pid %d: %v`、`netstat -ano | findstr :%d` |

**结构读法**：起回环 HTTP 服务 → 写端口文件 → 插件（浏览器侧）连上 → NH 消息承载 `session_start/session_end/cdp.event/cdp.response` →
`cdp.command` 下发 + 按 id 等 `cdp.response` + 心跳判活；上一轮残留靠 `netstat` 找 pid 再 kill。
**对 gaea 的价值**：这是「浏览器自动化跨进程」的最省事形态——不需要 embed 浏览器，也不需要 CDP 直连端口暴露给沙箱。

## 21.3 沙箱两件：策略字段 + WFP + ACL

- **provisioning payload 字段**（原文）：`version / offline_username / online_username / codex_home / command_cwd / read_roots / write_roots / deny_read_paths / deny_write_paths / proxy_ports / allow_local_binding / otel / real_user / mode / runtime / refresh_only`；
  模式枚举 `full / interactive-provision / provision-only / read-acls-only`。
- **网络边界走 WFP**（不是应用层代理）：`FwpmEngineOpen` / `FwpmFilterAdd` / `FwpmTransactionCommit`，
  provider 名 `Lingxi Windows Sandbox WFP`，描述 `Persistent WFP provider for Lingxi Windows sandbox filters`；`CapSids` 4 成员 + `LingxiSandboxUsers` 组。
- **文件边界走 ACL**：setup marker 用保护性 SDDL `D:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;GA;;;<sid>)`；
  deny-read 状态落 `deny_read_acl_state.json`；有守卫 `refusing to apply a deny-read ACE to filesystem root`。
- **runner 契约**：`command / cwd / env / permission_profile / workspace_roots / codex_home / real_codex_home / cap_sids / network_proxy_restricting_sid / timeout_ms / tty / stdin_open / use_private_de…`；
  路径约定 `.agents.codex/tmp`、`.codex.sandbox`、`project-roots://codex-project-ro…`。

**对 gaea 的映射**：这套 `permission_profile + workspace_roots + deny_*` 与我们 P0-2 的 `sandbox_permissions` 几乎同构；
要补的不是新机制，而是把「deny 路径 + 网络 SID」这两个我们今天没有的维度写进工具 schema（若未来做本地沙箱）。

## 21.4 kdocs-cli：云能力独立成 CLI

- 形态（Usage 原文）：`kdocs-cli auth set-token <token>`、`kdocs-cli <service> <action> [json_params]`、
  `kdocs-cli call <tool_name> [json_params]`、`kdocs-cli feedback [--reason …] [--skill-name …]`，另有 `upgrade -y` 与 checksum 校验。
- 认证交换端点：`/office/v5/ai/skill_hub/wps_auth/exchange`；凭据 `WPS_SID`（`KDOCS_CLI_APP_ID` 在 app-server 侧注入）。
- 动作/参数词表：snake_case 候选 **1293** 个（动作名与入参名同表，静态无法完全切分；按 `call <tool>` 形态看是「工具名即动作」）。
- **对 gaea 的价值在分层不在代码**：灵犀把「云」做成独立 CLI，技能脚本只拼 JSON；这比把云 SDK 编进主进程轻，也便于断网降级。我们若不接金山云，这层整体不移植。

## 21.5 仍缺的三块（下一批候选）

1. `lingxi.exe` 的 `templates/tools/*.tmpl` **正文**没在 .rodata 里按名相邻——`c_deliver_result` 只是内容块类型枚举；
   工具描述正文可能被压进 embed FS 的拼接串，需要按 `deliver_result` 的**参数提示文案**反查（下一批可用 rizin 交叉定位）。
2. `prompt.*` 416 个符号 ↔ .rodata 文案的**逐条配对**（符号给结构，文案在 §14 段库，尚未对齐 id）。
3. Ghidra 仍未配置 → 依旧没有伪代码；`wpsclibroker.RunProxy` 等热点只能到符号级。

---

# E 批续批（2026-11-13 · 工具模板库 / IAB 路由 / kdocs-cli 的 JSAPI 动作面）

> 纯静态。本批把 §21.5 的第 1 项（`tools/*.tmpl` 正文）**结掉**，并补上主进程 IAB 的 HTTP 面与 kdocs-cli 的第三层能力。
> 产物：`re-lingxi/out/TOOLS-tmpl.md`（371 个 define 块 / 21 簇，128 KB）、`re-lingxi/out/KDOCS-wps-actions.md`（133 个动作）。脚本：`extract_tool_tmpl.py`、`probe_batch5*.py`。

## 22.1 `tools/*.tmpl` 正文找到了

方法：按 `{{ define "x" }}…{{ end }}` 抓块（479 命中 / 去重 371），再按物理邻近（间隔 > 2500 字节）聚簇 → **21 簇 ≈ 21 个模板文件**。

簇清单（offset / define 数 / 归属）：

| 簇 | offset | define 数 | 归属（按 define 名判定） |
|---|---|---|---|
| 1 | 40 019 674 | 1 | `model_specific` |
| 2 | 40 033 789 | 2 | `members`（群成员） |
| 3 | 40 038 587 | 9 | `url_fetch`（含 `no_urls_provided` `status_failed` `error_prefix`） |
| 4 | 40 043 219 | 2 | `connectors` 段 |
| 5 | 40 048 259 | 10 | `image_search`（`no_queries` `no_results` `success`） |
| 6 | 40 055 255 | 19 | **`todo_write` + `deliver_result` 混簇**（`name_arg` `type_arg` `ref_arg` `missing_arg` `todos_arg`） |
| 7 | 40 066 293 | 4 | `skills_list` / `skills_local` / `skills_web` / `skills_main_v2_selection` |
| 8 | 40 069 837 | 13 | `connector_exec`（`id_arg` `arguments_arg` `call_missing_tool`） |
| 9 | 40 076 313 | 111 | 主模板混编（含 `Prolog` `Epilog` `StatusTable` —— 那是 `net/http/pprof` 的调试模板，非灵犀自有） |
| 10 | 40 107 675 | 57 | **`share_file` / `render_widget` / `view_image` 混簇**（`cloud_doc_description` `cloud_doc_success` `paths_arg` `message_arg`） |
| 11 | 40 121 467 | 21 | `connector_manage`（`endpoint_arg` `headers_arg` `add_missing_mode`） |
| 12 | 40 131 935 | 2 | `share_file` 云文档分支补丁 |
| 13 | 40 139 138 | 1 | `skills_local`（第二形态） |
| 14 | 40 142 186 | 1 | `memory` 段 |
| 15 | 40 150 024 | 23 | `search` / `await` / `academic` 混簇（`cancelled` `completed` `exec_error`） |
| 16 | 40 164 292 | 12 | `execute_jupyter`（`code_arg` `error_kernel` `kernel_failed_user`） |
| 17 | 40 190 517 | 23 | `timer_task`（`schedule` `add_success` `list_success`） |
| 18 | 40 222 843 | 1 | `im_compact_prompt` |
| 19 | 40 231 797 | 1 | `description`（孤立） |
| 20 | 40 235 367 | 57 | `bash`（`command_arg` `justification_arg` `idle_timeout` `run_in_background_arg`） |
| 21 | 40 253 369 | 1 | `memory` 段（第二形态） |

### `deliver_result` 契约原文（簇 6，逐字）

`description` 是一整块 `<instructions>`，首句 `向用户交付成果文件或可访问链接，在模型总结的结尾处调用。`，正文八条：

```
- 生成文件或需要展示可访问链接时，使用此工具交付
- name：文件名
- type：交付物类型：
 - local：本地文件，ref 填文件绝对路径（如 C:\Users\...\report.docx 或 /home/lingxi/workspace/report.docx）
 - cloud：WPS 云文档，ref 填 *.kdocs.cn/l/xxxx 格式的云文档链接（如 https://365.kdocs.cn/l/abcXYZ）
 - web：除云文档外的 http/https 网页链接
- ref：具体路径或链接。local 填绝对路径，cloud 填 *.kdocs.cn 云文档链接，web 填完整 URL
- 文档、图片、表格、网页等所有文件类型均通过此工具交付
- 单次调用只交付一个文件或链接，多文件多次调用
- 勿传入 HTML 源码或文件夹路径
- 调用前不要加markdown格式的有序列表或无序列表
```

- `name_arg`：`原始文件名`
- `type_arg`：`交付物类型：local（本地文件路径）、cloud（*.kdocs.cn 云文档）、web（除云文档以外的http/https 网页）`
- `ref_arg`：`具体路径或链接。local 填文件绝对路径（从根目录开始的完整路径），cloud 填 *.kdocs.cn/l/xxxx 云文档链接，web 填非云文档的完整 URL`
- `success`：`文件：{{ .Name }} 已成功交付给用户，后续不再使用任何markdown格式输出交付物路径`
- `missing_arg`：`参数错误：name、type、ref 均为必填项且不能为空`；`invalid_arg`：`参数错误：{{ .Error }}`

**注意 `success` 那句**：交付成功后**禁止再用 markdown 输出交付物路径**；`description` 又写「调用前不要加 markdown 格式的有序列表」——
两条合起来就是「交付动作前不加列表、后不再贴路径」的**双禁**，与 `delivery_format`（协议链接直出、禁反引号和代码块）同属防渲染错乱。gaea 可直接对齐。

### `share_file` 契约原文（簇 10）

- `description`：`在 IM 频道中向用户发送文件或图片。` + `<instructions>` 九条（仅 IM 可用、绝对路径、支持多文件、可带说明、调用时机在全部生成完成后……）
- `paths_arg`：`要发送的文件路径数组……绝对路径……支持图片、文档、代码等常见文件类型`
- 失败族：`missing_paths` / `parse_args_failed` / `partial_success` / `all_failed` / `operation_failed`
- 云文档分支：`cloud_doc_description` `cloud_doc_unavailable` `cloud_doc_success` `cloud_doc_partial` `cloud_doc_all_failed`
  其中 `cloud_doc_all_failed` 明写 **「禁止重试本工具，也不要编造云文档链接」**。

### `render_widget` 契约原文（簇 10，节选）

> `code` 为 HTML 片段（不含 `<html><head><body>`）；容器 ECharts 元素必须显式 `width:100%` + `height`，否则画布尺寸为 0 静默失败；
> 每次调用是独立渲染环境、不共享 JS 状态；`code` 只放图表不放大段文字；不要内嵌 mermaid（mermaid 直接在回复文本里输出代码块）。

→ 这是 gaea `render_widget` / GenUI 围栏那一套可以直接抄的**失败模式清单**。

## 22.2 IAB（内置浏览器）的 HTTP 面

主进程 `index.js` 里 IAB 是标准 `node:http` 服务，**只挂两个路由**（原文）：

```js
handleRequest(e,n){
  if(e.method==="GET" && e.url==="/status"){ this.handleStatus(n); return }
  if(e.method==="POST" && e.url==="/cdp"){ this.handleCdp(e,n); return }
  this.sendJson(n,404,{error:{code:-32600,message:"not_found"}})
}
```

- `GET /status` → `{iabConnected:true, version:"1", ready:true}`
- `POST /cdp` → CDP 方法转发（JSON-RPC 风格，错误码沿用 `-32600`）
- 端口：`server.listen(0,"127.0.0.1")` **临时端口**，启动后打日志 `IAB HTTP listening on 127.0.0.1:${port}`（本机实测 56305）。
- 与 native-host 的关系：IAB 是**应用内**浏览器通道；native-host 是**插件/外部浏览器**通道（§21.2），两者共用「本地 HTTP + CDP」形态但不共进程。

## 22.3 kdocs-cli 的第三层：`jsapi` 动作代理

§21.4 只看到 REST 那层。本轮新增：kdocs-cli 还有 `jsapi` 子命令命名空间（错误串 `Error: unknown jsapi subcommand %q`），
其下是 **`wps.<namespace>.<op>` 形态的语义动作注册表**：命中 **133 个动作 / 26 个命名空间**（全量见 `KDOCS-wps-actions.md`）：

| 命名空间 | 动作数 | 例 |
|---|---|---|
| `wps.tables` | 18 | `wps.tables.*`（表格增删改/边框/合并） |
| `wps.shapes` | 17 | 形状增删、环绕方式、叠放次序 |
| `wps.texts` | 15 | 文本替换/样式/高亮/批量格式 |
| `wps.sections` | 11 | 节、页眉页脚距离 |
| `wps.images` | 10 | 图片插入/水印 |
| `wps.content_controls` | 7 | 内容控件 |
| `wps.footnote_endnotes` | 7 | 脚注/尾注 |
| `wps.header_footers` | 7 | 页眉页脚对齐、页码 |
| `wps.bookmarks/comments/revisions` | 5/5/5 | 书签、批注、修订 |
| `wps.fields/lists/tocs/captions/hyperlinks/ole/styles/breaks/headings/protection/references` | 1–4 | 其余 |

**对照 gaea**：这套 `wps.*` 动作命名与我们的 `office_*` op 面是同一层语义（word 21 类）。
差别还是通道（这里走 kdocs-cli → WPS JSAPI；我们走 COM）。**但命名值得借鉴**：`<对象>.<操作>_batch` 的批量语义（`heading_text_batch`、`style_font_batch`）比我们的「一次一个 op」更适合模型批量改稿。

## 22.4 修订与剩余

| 项 | 状态 |
|---|---|
| §21.5-1 `tools/*.tmpl` 正文未找到 | **已结**：371 个 define / 21 簇，`deliver_result`/`share_file`/`render_widget` 全文可取 |
| §21.4 kdocs-cli「纯 REST 客户端」 | 修正：还有 `jsapi` 子命令（133 个 `wps.*` 动作）+ `auth` / `upgrade` / `feedback` |
| 主进程 IAB 只有「56305」的记载 | 修正：临时端口 + 仅 `/status` `/cdp` 两个路由 |

剩余（下一批候选）：
1. `prompt.*` 416 符号 ↔ `TOOLS-tmpl.md` 文案的**逐条配对**（现在两边都有了，缺按工具 id 对齐）。
2. `wpscli` 的 34 个工具 ↔ `wps.*` 133 动作 ↔ gaea `office_*` op 的**三方映射表**（做移植时最省事的中间件）。
3. Ghidra 伪代码仍未做（`GHIDRA_INSTALL_DIR` 未配置）。

---

# F 批续批（2026-11-13 · 线协议词汇 + IPC 通道全表）

> 纯静态、只读灵犀产物（`lingxi.exe` 与主进程 bundle），未进入任何业务仓代码。
> 产物：`re-lingxi/out/WIRE-json-tags.md`（2303 个字段名）、`re-lingxi/out/IPC-channels.md`（510 个通道）。脚本：`extract_wire.py`、`extract_ipc2.py`。

## 23.1 线协议词汇表：2303 个 json tag

`json:"…"` 命中 **3061** 次、去重 **2303** 个字段名。这条事实顺带解释了 A 批的一个悬案：

> **为什么二进制里搜不到 `inputSchema` 的 JSON？**
> 因为工具 schema 是运行时由 `google/jsonschema-go v0.4.3` **按 Go struct tag 现算**的，不是内嵌 JSON 文本。
> 所以「字段名权威」在 tag，不在任何 JSON 文件里 `[观]`。

分组抽样（括号内为该串在二进制中的出现次数）：

| 组 | 字段 |
|---|---|
| 会话/运行 | `session_id`(2) `run_id`(2) `agent_id`(2) `group_id`(2) `message_id`(2) `request_id`(5) `title`(5) `status`(4) `state`(2) `mode`(2) |
| 工具调用 | `tool_name`(2) `tool_call_id`(1) `arguments`(2) `input`(2) `output`(2) `result`(3) `error`(3) `error_code`(1) `brief`(3) `justification`(1) `sandbox_permissions`(1) |
| 鉴权 | `auth_mode`(1) `wps_sid`(2) `token`(2) `refresh_token`(1) `device_id`(2) `user_id`(2) |
| 沙箱 | `read_roots`(1) `write_roots`(1) `deny_read_paths`(1) `deny_write_paths`(1) `proxy_ports`(2) `allow_local_binding`(2) `permission_profile`(1) `cap_sids`(1) |
| 技能 | `skills`(2) `skill_name`(1) `enabled`(4) `location`(4) `installed`(1) |
| 连接器/MCP | `connector`(2) `connector_id`(2) `server`(2) `endpoint`(3) `headers`(4) `command`(2) `transport`(1) |
| 内容块 | `type`(13) `text`(2) `content`(4) `delta`(2) `finish_reason`(2) `usage`(2) `input_tokens`(3) `output_tokens`(3) |

**读法**：沙箱那一组字段与 §21.3 的 setup payload 字段**完全对齐**（`read_roots/deny_read_paths/proxy_ports/allow_local_binding`），
说明这条 JSON 既走 setup 二进制、也走 app-server 的 RPC——同一份契约两处复用。

## 23.2 IPC 通道全表：510 个

bundle 里通道是**常量表**形态 `t.AgentSteerRun="agent:steerRun"`（不是散落字符串），因此可以完整枚举：

- 命中 510 个（去重后仍 510），命名空间 **18+**。
- 头部命名空间：

| 命名空间 | 通道数 | 代表 |
|---|---|---|
| `file` | 57 | 文件读写/预览/缩略图 |
| `agent` | 52 | `agent:steerRun`、`agent:getSessions`、`agent:testModel`、`agent:queue:*` |
| `fileBrowser` | 41 | 文件浏览器 |
| `iab` | 35 | 内置浏览器（与 §22.2 的 HTTP 面配套） |
| `base` | 33 | 基础能力 |
| `pet` | 27 | 桌宠 |
| `window` | 23 | 窗口控制 |
| `business` | 22 | `business:installSkillFromUrl`、`packSkillZip`、`getSkills` |
| `project` | 21 | 项目 |
| `auth` | 19 | 登录/登出/刷新 |
| `debugCenter` | 18 | 调试 |
| `im` | 13 | IM 频道 |
| `localScheduledTask` | 12 | 本地定时任务 |
| `upload` / `remoteControl` / `logAnalysis` / `cowork` / `sandbox` | 12/10/9/8/8 | 其余 |

**与 A 批的差额**：A 批记「532 IPC 通道」是裸扫 `ns:method` 字面量（含事件名与误命中）；本批按常量表口径得 **510**，
差额来自：事件通道（`*:changed` 一类）与 `window:*` 之外的非 `ns:method` 形态。

## 23.3 本批边界

- 仍**没有**字段 ↔ 结构体的绑定关系（tag 与 struct 的归属需要反编译），所以「某个 RPC 的完整参数集」还拼不出来。
- `prompt.*` 416 符号 ↔ `TOOLS-tmpl.md` 文案的配对仍未做（两边都齐，缺 join 规则）。
- `wpscli` 34 工具 ↔ `wps.*` 133 动作的映射仍未做。
- Ghidra 仍未配置，无伪代码。

---

# G 批续批（2026-11-13 · 五份全量清单 + 一处抽取修正）

> 纯静态、只读灵犀产物。本批一次出五份清单，并**修正上一批的一个抽取缺陷**（模板 trim 标记）。
> 产物：`GO-types.md`、`GO-error-codes.md`、`HTTP-routes.md`、`PROMPT-tool-contracts.md`、`WPSCLI-tool-js.md`；脚本：`extract_all.py`、`extract_contracts.py`。

## 24.1 先修正：模板 define 漏了 trim 标记

上一批（§22.1）用的正则是 `{{ define "x" }}…{{ end }}`，**只匹配不带 trim 标记的写法**。
实际灵犀大量模板写成 `{{ define "x" -}} … {{- end }}`，因此上一批的 **371 个 define / 21 簇是漏计**。

修正后重抽：

| 口径 | 上一批（漏计） | 本批（修正） |
|---|---|---|
| `{{ define }}` 命中 | 479 | **585** |
| 去重 define | 371 | **446** |
| 簇 | 21 | **23** |
| 被漏掉的典型 define | — | `start_line_arg` `end_line_arg` `step_limit_reminder` `exec_err` … |

影响：`File` 工具的 `start_line` / `end_line` / `content` / `action` 等参数文案上一批全部缺，修正后可完整取到（见 §24.4）。

## 24.2 Go 类型清单：15 984 个（1 365 个包）

`GO-types.md`：去重 **(包, 类型)** 15 984 对，包 1 365 个。业务包类型全量列出（`core` 174 / `mcp` 118 / `protocol` 119 / `agent` 59 / `store` 39 / `config` 29 / `dw` 22 / `session` 18 / `auth` 17 / `server` 8 / `rpc` 7 / `skill` 4 / `connector` 4）。
读法：类型名能看出分层（`core` 最厚），但**字段仍需反编译**才能拿到。

## 24.3 错误码清单：240 CamelCase + 185 snake

`GO-error-codes.md`。CamelCase 侧按命名特征（Unauthorized/Denied/Conflict/NotFound/Exceeded/…）筛，例：
`UserUnauthorized` `ResourceConflict` `PermissionDenied` `SessionIsRunning` `ResponseTooLarge` `InvalidModelConfig`；
snake 侧例：`admission_denied` `missing_identity` `missing_group_id` `lifecycle_logout` `publish_canceled`。
**用途**：核对 RPC 失败语义时的权威码表（比猜字符串可靠）。

## 24.4 工具契约：25 工具 / 102 参数 / 86 条有中文原文

join 方法（两级）：
1. 先按签名 define 把工具锁定到**簇**（`file`→含 `start_line_arg` 的簇、`bash`→含 `justification_arg`…）；
2. 簇内找「参数 camel → snake → `<arg>_arg`」；簇内没有时，仅在「非通用名 + 全局唯一」时回退，否则标 `歧义`。

命中 **86** / 未命中 **16**。样例（逐字）：

| 工具 | 参数 | 原文 |
|---|---|---|
| `File` | `action` | `操作类型。read: 读取文件; write: 覆盖写入; append: 追加写入; edit: 精确替换文本` |
| `File` | `content` | `write/append 时写入的内容（空字符串会被校验拒绝；需要清空文件时传一个空格）` |
| `File` | `edits` | `edit 时的替换列表，每项包含 find（要查找的精确文本）和 replace（替换后的文本，空字符串表示删除）；按序执行，单项未匹配时跳过，不影响其他项` |
| `Bash` | `sandbox_permissions` | `可选的单条命令沙箱覆盖选项。省略或设为 use_default 时沿用当前执行策略；设为 require_escalated 时需提供 justification，用于向用户发起非沙箱执行的授权请求。` |
| `Await` | `terminate` | `仅在用户明确要求终止/停止/杀掉该后台进程时才传 true…其余所有情况都必须传 false 或省略。` |
| `AskUserQuestion` | `options` | `选项列表（2-4 个）。顺序即展示顺序，先排好序再填入。` |

**未命中的 16 个参数**（`brief` `action` `prompt` `queries` `type` `reason` `path` `new_content` `old_text` `query` …）
不是缺失，而是**跨工具共用的同一段文案**（如 `brief` 在 search/web/bash 里是同一句话），按工具切分天然歧义。
所以「工具 × 参数」的唯一权威仍是 §23.1 的 json tag；文案是**共享词条**。

## 24.5 云 API 路由：129 条 / 68 组

`HTTP-routes.md`（口径收紧：只留 `/api/` `/office/` `/kdocs/` `/kdg/` `/open/`，排除 grpc/genproto/包路径噪声）。
A 批的「950 条去重」是宽口径（含 `/internal/`、protobuf 包路径）；本批 129 条是**真正的业务端点**。

## 24.6 wpscli JS 片段分组：129 条

`WPSCLI-tool-js.md`：按文档类型分组 ppt=19 / doc=13 / sheet=8 / pdf=3 / image=3 / other=83。
`other` 占多数是因为多数片段是**算子级碎片**（如 `ws.Range("__RANGE__").Value2 = "__VALUE__"`），不含 `Application.` 前缀，
说明这些是拼进外壳 `__BODY__` 的语句片段——与 §20.5 的「外壳 + 算子」两层结构一致。

## 24.7 边界

- 类型名有了、**字段归属没有**（需 Ghidra）。
- 错误码有了、**触发条件没有**（需 Ghidra 或动态）。
- 契约文案 86/102 命中；剩余 16 个是共享词条，按工具切分无意义。
- `wpscli` 34 工具 ↔ `wps.*` 133 动作的映射仍未做（需要算子片段与工具名的邻近关系，属下一批）。

---

# H 批续批（2026-11-13 · 逐工具归属尝试：两条路都失败，改交付可靠层）

> 目标是上次留的 KPI：`wpscli` 34 工具 ↔ JS 片段 的映射。本批**两条路都试了，都不成立**，所以改为交付可靠的那一层。
> 产物：`WPSCLI-tool-js-map.md`（邻近法结果，作反例保留）、`WPSCLI-module-map.md`（模块法结果）、`WPSCLI-js-statements.md`（收紧后的真 JS 语句 + 对象模型清点）。脚本：`build_tool_js_map.py`、`build_module_map.py`、`rebuild_js.py`。

## 25.1 路一：物理邻近 → 失败（会给出**错误**结论）

做法：34 个工具名各自取全部出现位置，片段按最近工具名归属。

结果：**122/129 全部被吸到 `wps_presentation`**。原因很清楚——34 个工具名全部聚在描述区（约 5.65 MB 一处连续串），
片段散布在 5.15–5.86 MB，最近距离没有区分度。这条路的输出保留在 `WPSCLI-tool-js-map.md` 里**当反例**，不要当结论用。

## 25.2 路二：Rust 模块路径划区 → 只对 PDF 有意义

做法：抽 `src/*.rs` / `jsapi/*.rs` 标记做区域边界，片段归最近模块。

结果：标记 169 个但只有 **56 种**，且绝大多数是依赖 crate 的 panic 路径（`slice` `str` `string` `de` `addressing` `parking_lot` `shard` `bytes_mut`…），
本仓自己的模块只剩：

| 模块 | 片段数 | 判读 |
|---|---|---|
| `jsapi/pdf_scripts.rs` | 22 | **真信号**：PDF 的 JS 是独立脚本模块 |
| `doctor.rs` | 4 | 体检/诊断，不起文档 |
| `auth.rs` `bridge.rs` `server.rs` `service.rs` `dispatcher.rs` `params.rs` | 3/5/5/27/4/2 | 与文档无关 |

→ 逐工具归属**在静态条件下不可得**：要么反编译命令模块（需 Ghidra），要么动态抓一次 `tools/call`。

## 25.3 改交付：收紧后的「真 JS 语句」集

上一批的 129 条里混进了描述文本（有一条 4000 字符的描述块）。本批加判定（含 `var/return/function` + 括号 + 分号，中文占比 < 12%）：

- 候选 300 → 去重 **213** 条真语句。
- 按文档类型：`doc` 13 / `ppt` 11 / `sheet` 7 / `pdf` 7 / `image` 2 / 其余 173（模板语句）。

**PDF 那 7 条最有价值**，直接给出 JSAPI 真身：

```javascript
var pdfDoc = Application.Documents.Open(filePath, false, false, "", "", false);
for (var p = 0; p < pdfDoc.PageCount; p++) { var text = pdfDoc.PageText(p); }   // PageText 0-based
var ok = pdfDoc.Exports.ExportToMarkdown(outputPath, startPage, endPage);        // 1-based 页面范围
pdfDoc.Close(0); pdfDoc.Save(); pdfDoc.Close(0);
```

其余大量片段是**模板语句**，印证 §20.5 的三件套：

```javascript
var __result = (function(doc) { __BODY__ })(doc);
return typeof __result === "string" ? __result : JSON.stringify(__result);
return JSON.stringify({ok: true, format: 'pdf', output: "__OUTPUT_PATH__"});
```

## 25.4 工具侧分类（不靠归属也能定）

按工具描述与是否出现脚本壳，34 个工具可分两类：

| 类 | 工具 | 判读 |
|---|---|---|
| **JS-backed**（起 WPS 进程，走 JSAPI） | `wps_doc_*`(8) `wps_pdf_*`(8) `wps_ppt_*`(6) `wps_sheet_*`(7) + legacy 3 | 每族都带 `*_script` 逃生舱；PDF 有独立脚本模块 |
| **Rust-native**（不起 WPS） | `wps_status`（doctor） `wps_search`（文件检索） `wps_file_info` `wps_browser`（CDP） | 描述里没有文档对象，也没有脚本入参 |

这条分类是可靠的：JS-backed 判定依据是「描述里出现 script/script_file 或 JSAPI」，Rust-native 判定依据是「描述里完全没有文档对象、且不涉及 WPS 引擎」。

## 25.5 结论与本批边界

- **逐工具 JS 归属：静态不可得**（已用两条路证伪，反例已存档）。
- 可靠层：文档类型 × 对象模型调用（`WPSCLI-js-statements.md`）+ 工具是否起 WPS（上表）。
- 若必须要到工具粒度：① Ghidra 反编译 `commands/doc.rs` 等模块；② 动态起一次 WPS 抓 `tools/call`（本机 WPS 12.1.0.29690 满足 wpscli 门槛，但需要父进程签名，`parent_verify` 会拦，见 §13）。

---

# I 批续批（2026-11-13 · 提示词骨架 / 记忆规格 / 占位符契约 / 流式事件）

> 纯静态四合一。产物：`PROMPT-skeleton.md`、`MEMORY-spec.md`（72 KB）、`WPSCLI-placeholders.md`、`STREAM-events.md`。脚本：`extract_i_batch.py`。

## 26.1 主提示词装配顺序（130 个节点）

在 `40 380 000`–`41 070 000` 区间按偏移取已知段名，压缩连续重复后得 **130 个节点**。开头顺序（节选）：

```
<group_chat> → <skills> → <runtime> → <language> → <format> → <im_scene>
→ <agent_loop> → <tool_call_style> → <agent_loop> → <memory_system>
→ <memory_index> → <memory_recall_rules> → <tool_use> → <system_reminder> …
```

读法（对提示词工程有直接参考价值）：

1. **场景开关前置**：`group_chat` / `im_scene` 在最前，说明「场景最高优先级」不是靠文末一句强调，而是靠**位置**。
2. **格式与工具风格先立规矩**：`language` → `format` → `tool_call_style` 在 `agent_loop` 之前，先把输出形态钉死，再讲循环。
3. **记忆插在循环之后**：`memory_system` 系列紧跟 `agent_loop`，即「先告诉你怎么干活，再告诉你能记住什么」。
4. `<system_reminder>` 出现多次（33 次是段库统计），说明它是**动态插入位**，不是一次性段落。

## 26.2 记忆子系统规格（72 KB 合订）

`MEMORY-spec.md` = 14 个段的原文 + 61 条记忆工具文案（`wiki_memory_*` / `write_memory_*` / `edit_memory_*`）。

覆盖：`memory_system`（三层：core / index / recall）、`memory_write_rules`（什么该写、什么不写）、
`memory_management`、`memory_reference_rules`、`knowledge_base`、`import_rules`（哪些目录可读不可写）、
`fragments_format`、`rollout_summary`、`user_profile`/`existing_user_profile`。

这是灵犀**长期记忆**的完整设计说明：写入有门禁（用户偏好/协作规则/长期上下文才写，其余不干涉）、
召回有分层（core memory 常驻 + index 检索）、wiki 页面有 front-matter 与来源锚点。

## 26.3 占位符契约（66 个，带出现次数）

`WPSCLI-placeholders.md`。次数最高的几个说明模板复用密度：

| 占位符 | 次数 | 用途 |
|---|---|---|
| `__BODY__` | 24 | 算子主体插槽（所有外壳共用） |
| `__BODY____TEMPLATE_PATH____OUTPUT_PATH__` | 4 | 链接器把相邻占位符合并后的形态（实为 `__BODY__`+`__TEMPLATE_PATH__`+`__OUTPUT_PATH__`） |
| `__FILE_PATH__` / `__OUTPUT_PATH__` / `__RANGE__` / `__TEXT__` | 多 | 基础入参 |
| `__CHART_TYPE__` `__DATA_RANGE__` `__OPERATIONS_JSON__` | 2 | 表格图表/清洗 |

**组合占位符是个坑**：`__BODY____OUTPUT_PATH__` 这种不是设计出来的，是**相邻两个占位符在字符串表里连写**导致的假名字。用它对模板时要注意拆开。

## 26.4 流式事件：静态拿不全（负结果）

只数带引号的字面量：`"char_delta"` 1 次、`"line_delta"` 1 次、`"error"` 8 次、`"usage"`/`"ping"` 各 1 次，
`content_block_*` 各 1 次。⇒ 事件名在二进制里多为**裸 token 或运行期拼接**，静态无法枚举完整流式协议。
A 批记的 `content_block_delta` / `char_delta` / `line_delta` / `patch_delta` 属于「存在但不成表」。

要拿到完整事件表只有两条路：反编译 app-server 的 SSE 写出点，或抓一次真实会话（需登录态）。

## 26.5 I 批的净增量

| 项 | 之前 | 现在 |
|---|---|---|
| 提示词装配顺序 | 只有 131 个段各自正文 | **130 节点顺序 + 四条装配规律** |
| 记忆子系统 | 段库里零散 5 段 | **72 KB 合订 + 61 条工具文案** |
| 占位符 | 只有 66 个名字 | **次数 + 类型 + 上下文 + 组合坑** |
| 流式事件 | A 批列了几个名字 | **确认静态枚举不可行**（负结果） |
