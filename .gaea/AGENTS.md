# gaea 项目记忆

> 本文件为项目长期记忆（文档记忆层级）。编码规范：**UTF-8 无 BOM**（历史遗留的 GBK/UTF-8 混合编码已清理）。
> 修改后请保持 UTF-8；.ps1 脚本需 UTF-8 带 BOM（见「沙箱环境备忘」）。
>
> **注入预算（2026-11-24 实测）**：gaea 把本文件自动注入本仓内每一次代理会话的提示词，单文件上限
> **32KB**（`internal/gaea/memory/doc.go` 的 `docMaxBytes`），超出部分直接截断、不报错。清理前本文件
> 40KB，「已知注意」「发布流程」等尾部规则一直没被注入。所以这里只留现行规则，按
> 「强制规则 → 现行决策 → 约定 → 纪律 → 流程 → 水位」由重到轻排列；逐条状态与历史写
> `.gaea/progress.md`，不写在这里。

## 构建入口（强制默认，2026-11-09 用户拍板）

任何构建、装机、功能预览包、源码归档或产物核验任务，开始前直接读取并执行
`docs/gaea-build-guide.md`；**不得等待用户提醒，不得凭记忆另造一条构建链**。

- 构建前先清理 `gaea.exe` / `wails.exe` / `go.exe` 孤儿，防止占用
  `build/bin` 与 Wails runtime。
- 先在前端执行 `npm.cmd run build`，再做 Wails 构建；bindings 必须生成，禁止
  `-skipbindings` 绕过。
- 使用系统默认 `TMP/TEMP`；设置 `CGO_ENABLED=1`、`CC=C:\msys64\ucrt64\bin\gcc.exe`，
  并把 `C:\msys64\ucrt64\bin` 前置到 `PATH`。缺 ucrt64 会让 cgo 静默降级成语音空壳。
- 仓根标准命令：`wails build -s -ldflags "-s -w" -trimpath`。若 `build.bat`
  在 Generating bindings 报 `Access is denied`，按指南清理孤儿、恢复系统
  `TMP/TEMP` 后走同一裸 `wails build`，不要跳过 bindings。
- `gaea.exe` 必须与 `onnxruntime.dll`、`sherpa-onnx-c-api.dll`、
  `sherpa-onnx-cxx-api.dll` 同目录；随后执行
  `scripts\smoke.ps1 -ExePath build\bin\gaea.exe`，`/api/health` 必须以 200 通过。
- 本机应用控制策略会拦截 `.tmp` 下的 exe；冒烟必须在 `build\bin` 或正式安装路径执行。
- 编程板块“外部浏览器打开”是跨 gaea 生命周期的持久会话：必须保存 PID/令牌、
  释放 Job 的关闭即杀位，并让 DSH stdout/stderr 直接继承日志文件；下次启动按
  会话状态重接管，停止时可按 PID 回收进程树。详见构建指南「编程板块与 Windows
  子进程」。
- 做“某提交的交付包”时，从该提交的独立干净工作树构建，严禁混入主工作树其他未提交
  业务改动；同时记录提交号、exe/DLL/ZIP/源码包 SHA256 与冒烟结果。
- 装机按 `scripts\gaea_install.ps1`；exe + 三枚 DLL 同放 `C:\AI\gaea-app`，
  桌面只保留 `gaea.lnk`。若本轮只做预览包，不擅自覆盖用户已安装版本。

## 数据根：蒸馏与逆向数据（强制默认，2026-11-20 用户拍板）

**所有蒸馏、逆向、上游取证的数据文件一律落 `C:\AI\re`，仓库只放文档与代码。**

- 一个上游目标一个子目录：`C:\AI\re\<目标>-<版本>`（`codex-0.165.0` / `codex-0.170.0` /
  `codex-computer-use` / `lingxi` / `agorai` / `aidebate`）；**上游参考克隆沿用上游项目名并
  保留 `.git`**（`orca` / `zcode` / `projectlibre` / `unsloth` / `rea` / `deepseek-harness` …），
  依赖快照用 `<项目>-<latest|baseline>`（`dsh-web-latest` / `dsh-web-baseline`）；目录总览、
  布局差异与迁移表以 `C:\AI\re\README.md` 为准。
- 子目录分层：`docs/`（逆向说明）、`evidence/`（原始证据）、`embedded/`（解出的资产）、
  `scripts/`（可复跑）、`work/`（可重生成）、`oss/`（上游源码只读快照）、`data/`（成品证据包）。
- **2026-11-20 两批迁移已执行**：
  ①`C:\AI\_distill\{codex-computer-use,agorai,aidebate}` 与仓库内 `re-lingxi/` → `C:\AI\re`；
  ②仓库内 `clones/` 全部 16 个目录（上游参考克隆 13 + DSH web 依赖快照 2 + mpp2013 样本 1）
  → `C:\AI\re\<同名>`，文件数 16/16 核对一致（`diffs=0`，合计 4.60 GB）。
  `_distill`、`re-lingxi/`、`clones/` 三处旧位置均已删除，**不再复活**。
- 文档引用数据写 `C:\AI\re\<数据集>\...` 绝对路径；大件只记路径与 SHA256/commit。
- **代码耦合**：`internal/schedule/mpp_test.go` 的真实 MPP 样本经 `reSamplePath()` 定位，
  默认根 `C:\AI\re`（`GAEA_RE_ROOT` 可覆盖，旧 `../../clones/` 仅兜底），依赖
  `projectlibre\projectlibre_build\resources\samples\*.mpp` 与 `mpp2013\*.mpp`；
  动数据根或样本目录后必跑 `go test ./internal/schedule -run TestMpp -v`，须 PASS 而非 SKIP。
- 新增/迁移后：改 `C:\AI\re\README.md` 总览 + `grep -rn` 清旧引用 +
  `node scripts/check-docs.mjs` 验悬空。

## 已知注意

- **跑 Go 测试前把 `APPDATA` / `XDG_CONFIG_HOME` 重定向到临时目录**（2026-11-24 实测）：
  `config.MemoryUserDir()` 走 `os.UserConfigDir()`，Windows 上读的是 `%AppData%`——只改
  USERPROFILE/HOME 拦不住对库的访问。`internal/gaea/tool/builtin` 全量有既有用例会打开
  正式 Hephaestus.db（新迁移会因此被直接应用到正式库），`internal/app` 检索族测试会连
  真实引擎清真实向量。测试内隔离写法见 `internal/app/gaea_proposal_test.go` 的
  `proposalBindingEnv`（重定向 + fail-closed 核对 + 关库）。
- 本机默认没有 gcc 时，`go build ./internal/app/` 在 `runtime/cgo` 就失败退出，不会报出
  app 包自身的编译错误（容易误读成通过）：类型检查用 `CGO_ENABLED=0 go build ./...`，
  真构建仍按「构建入口」走 ucrt64。
- 方案库（一级板块，`internal/gaea/proposal` + `ProposalB`）：历史方案按标题 1~4 级存取，
  录入只在办公「方案」运行模式里由模型经 `proposal_save` 完成；检索复用本地 embedding /
  reranker 客户端，不另造检索工具。设计拍板与坑见 `.gaea/progress.md` 2026-11-24 条。
  正式库里的方案「包3技术标 第三版」（3 节短文）是用户 2026-11-24 拍板保留的
  测试夹具，供后续优化做测试，**勿删**。
- 角色库剧照默认跟随绘梦（ImageBackend/ImageModel），可在模型中心单独绑定
- 文生视频依赖本地 ComfyUI 安装 LTX-Video 模型
- 里程碑：2026-08-12 完成通用办公全面打磨（显示/布局/安全三线：成本库/记忆/知识库/技能写入全部硬性确认
  含 yolo、子代理路径；子代理不再继承持久化写入工具）

## 在册决策与边界（2026-11-24 清理）

> 用户 2026-11-24 明示：**现在没有长期规划。** 原「长期规划（权威，2026 定稿）」一节列的下一阶段规划、
> 瘦身总规划、审计衍生优化方向、下一代路线图都是过去式，不再作指导；被移出的原文见
> `docs/archive/agents-cleanup-2026-11-24.md`。提方向、排工作只对下面这些现行决策。

- **性质**：A「终极个人工具」（用户 2026-09-08 拍板）——产品化降为期权不作承诺，不为想象中的用户写代码；
  LICENSE=私有 All Rights Reserved（根目录 LICENSE）。
- **双空间**：工位与乐园并列平级（用户 2026-09-08 拍板）。工作与娱乐分开——记忆分区互不检索、模型策略
  各配各的、上下文不跨空间，跨空间只由用户显式发起。哪个板块属于哪个空间看
  `frontend/src/boards/manifests.ts` 的 `space`（work / play / shared / independent），不凭旧文档里的
  板块归类下判断（例：文学现在归 work）。
- **i18n**：诚实 zh-only——壳层 chrome + 设置外壳三语（zh/en/zh-TW），页面内容层保持 zh 单语，不逐页铺字典。
- **仍在册的排除项**：GoalCard 系（v3.6.0 撤下，使用率低维护面大）勿再提。
- **已被推翻的旧边界**（不要再当约束引用）：「不做算量」（现有 cad-takeoff）、「MCP 不进规划」
  （现有 MCP 服务管理）、「编程板块=DSH 独立窗口不融合」（编程板块已作为一级板块回流）。
- **复用既有工具**（瘦身审计沉淀的编码约定）：新下载走 saveExportBlob、新选取走 pickFile、
  新 slug 用 strutil.TitleSlug、新解码用 b64ToBytes、NAVIGATE 用 manifest id；新域先问「能否是文档」。
- **文档纪律**：`docs/` 的旧调研与已落地计划归档在 `docs/archive/`（见其 README），不把其结论当现状引用。
  新增文档必须登记 `docs/README.md`——守卫 `node scripts/check-docs.mjs`（孤儿登记 / `docs/` 悬空引用 /
  本文件指令预算 / 含非 ASCII 的 .ps1 必须带 UTF-8 BOM），已接入 `scripts/ci.ps1`。

## 项目定位

gaea 是 Windows 桌面端「通用办公」AI 助手（Wails v2：Go 1.26 后端 + React/TypeScript/Vite 前端）。
核心能力：文档撰写、表格处理、格式转换（docx/xlsx/pdf → Markdown）、图表生成、报告拼装、
方案库与记忆中枢、方案编写。品牌定位已从「土壤修复工程办公」全面转为「通用办公」。
一级板块清单以 `frontend/src/boards/manifests.ts` 为准（聊天/文学/绘梦/办公/造价数据库/方案库/编程/
众神殿/记忆中枢/模型中心/角色库/微信助手/进度计划/运行监测）。知识库已于 2026-11-23 整体退役，由方案库接位。

## 技术栈与关键约定

- 桌面框架 Wails v2.13（Go + WebView2）；后端事件总线 + 前端 zustand 桥接（bridge.ts → window.go.app.App）
- **God 类型冻结约定（2026-10-03 批 52，A3③ 裁决）**：`App`/`mediaState`/`writingState` 等历史巨结构体
  维持现状不拆（嵌入地层已稳定，再拆收益近零——round-31 判定）；**新增字段必须挂子结构体**
  （如 cfg/media 子域），不得继续往顶层铺平；相关拍板项（X1-03/AgentRunner 收敛）按
  docs/code-audit-2026-10-02/decision-brief-2026-10-03.md §A3 处置
- **绑定面（v2.17.0 起）**：App 不再直接绑定 Wails；导出方法拆 12 个板块门面
  （internal/app/bindings_*.go：CoreB/OfficeB/MemoryB/CostB/ModelB/VoiceB/ChatB/ImageB/CharlibB/PantheonB/WenxueB/ProposalB，
  纯委托零逻辑改动）。改绑定面方法后用 `go run ./scripts/gen_bindings` 重新生成 +
  `TestBindingsCompleteness` 兜底；摘除或预留绑定走生成器的排除清单；绑定面现值 724（2026-11-24），漂移守卫的 bindingNames 须同批再生；前端调用经 gaea/lib/bridge.ts（按方法名路由门面）或
  api/bridge.ts 的 window.go.app.App 兼容代理；旧 wailsjs 导入走 src/wailsjsCompat 重导出；
  wails build 会重生成 wailsjs/go/app/<门面>.js
- 单模型架构：一个 executor 完成规划与执行，无独立规划器；任务/技能子代理走 `task` / `run_skill`
- 内置工具清单以 `docs/gaea-tool-registry-map-2026-11.md` 为准（`boot.Build` 单一 `tool.Registry`）；新增工具须同批更新契约行、自声明与文档
- 文档能力交给 ModelScope 技能：docx / pdf / xlsx / pptx（安装在 `~/.codex/skills` 与 `.gaea/skills`），
  转换引擎共用 `internal/docmd`（format_convert 工具与预览面板同一实现）
- 内置子代理技能：format-convert / chart-builder / doc-assemble
- 记忆系统：SQLite（`%APPDATA%\gaea\Hephaestus.db` facts 表，按项目 slug 隔离）+ 文档记忆（AGENTS.md 层级）
- 环境依赖：LibreOffice（soffice）、node 全局 docx、Python 3.13（lxml/openpyxl/pypdf/pdfplumber/reportlab/pandas/matplotlib 等）
- 本地 AI 底座：**Herdsman**（localhost:8080/v1；embedding/reranker、视觉、解析/OCR、生图等），
  gaea 的视觉/检索/OCR/解析/生图等依赖它，升级可能破坏契约——用 App.HerdsmanProbe 启动探测。
  gaea 另自管几个外部引擎：llama（35B 对话）、WHIRL（8126）、Strata、Breeze-TTS-2、ComfyUI；
  CosyVoice2 / Ollama / OpenCode Zen / 天璇（tianxuan）/ VoxCPM2 已退役

## 执行纪律：子代理按需使用（2026-09-03 立，2026-11-14 用户撤销强制并发）

用户已明确撤销「默认并发子代理」的强制要求。后续不再为满足并行形式而派活，
由主代理按任务独立性、风险和收口成本自行判断是否使用子代理。

1. **主代理默认直接执行**：单线任务或尚未形成稳定文件契约的任务，由主代理连续完成，
   不强制拆给子代理。
2. **仅在确有收益时拆线**：任务含多条真正互不相交的实现线，且派活能缩短整体交付时，
   才可并行；必须先固定 props/类型/文件足迹，主代理负责集成和最终验证。
3. **失败直接接手**：子代理报错、空转、只回报纪律未落产物或造成集成风险时，不再重试
   形式性并行，主代理立即接管完成剩余工作。
4. **每刀如实回写**：在 `.gaea/progress.md` 记录实际由谁完成、用了哪些验证；不得把
   「已派出」写成「已完成」，也不得为历史一致性继续声称并发是默认要求。

## 交互纪律：不许用「等待」换时间（2026-09-10 用户拍板）

用户原话（对上一轮执行的批评）：「后台运行的东西你为什么要等待」「在浪费我的时间」。
耗时本身不可怕，**干等与重复**才是浪费。后续每一轮工作按下办：

1. **后台任务不阻塞**：起后台任务后立刻去做别的事（改码/读文档/跑定向用例/写文档），
   收到完成通知再取结果。**禁止起完就守着等**——那几分钟是白扔的。
2. **重活先报 ETA 再跑**：实测耗时（2026-11-24 口径）——前端全量 vitest ≈ **6~7 分钟**（517 文件 4734 例）、
   Go 全量 **131 包**（须 ucrt64+cgo、APPDATA 重定向到临时目录；逐包先铺 DLL，所以慢）、
   `scripts/ci.ps1` 全门禁更久。开跑前一句话说清「跑什么、约多久」，用户可否决。
   **「CI 绿」口径（2026-09-27 定）**：写门禁结论必须带定语——「本地快闸」=
   `scripts/ci.ps1`（构建/vet/golangci/Go 全量/漂移/前端/E系/卫生）；
   「Actions」=.github/workflows/ci.yml 三 job（backend+race+frontend，race 以 Actions 的 race job 为准）。禁无定语「CI 绿」。
3. **定向优先，全量收尾**：改动后先跑目标文件（3–10s）确认；全量只在交付前跑一次。
   一轮改动 = 中间定向 + 收尾全量，**同一验证不重复跑**（上一轮全量跑了 4 遍=反例）。
   **工具（2026-09-27 立）**：`scripts/test-related.ps1` = git 改动 → Go 包
   （吃结果缓存，不带 -count=1）+ `vitest related`，几秒~分钟级；`ci.ps1 -Quick`
   = 静态闸（build/vet/lint/卫生守卫）+ 受影响面测试，实测 ~1 分钟（vs 全量
   5-10 分钟）。**全量档（ci.ps1 不带参）保留在 push 前/发版前跑**；
   用户对「几分钟」的抱怨=2026-09-27 拍板用两档制化解，勿再无脑全量。
4. **超时上限按实际需要写**：不要写 900s 这种夸张数字——那是上限不是耗时，
   只会造成「要卡你十几分钟」的观感。
5. **不把「等待」当进度表达**：进度由已完成的具体产出说话，不由轮询次数说话。
   用户催问时先答「在跑什么、还要多久」，再继续。
6. **能并行就并行**：长任务是可并行的（截图取证 / 读码 / 单测互不依赖），
   串行排队本身就是一种浪费。

> 与上一节的关系：并发是**手段**（把时间省下来），本节是**约束**（别把省下的时间又等回去）。

## 工装：CDP 走查与前端调试（2026-09-10 增补）

- `scripts/cdp-walk.mjs` 支持 **`--target <url 片段>`**（多标签时选定目标页，缺省取第一个
  page）与 **`@文件路径` 传入 JS 表达式**（PowerShell 5.1 传原生命令参数会吃掉内层引号，
  长表达式一律写文件再 `@` 传入）。真机走查配方：无头 Edge
  `--headless=new --remote-debugging-port=9333 --window-size=1440,920 --user-data-dir=<tmp>`
  + Vite dev（`?mock=1`）→ 本脚本 eval/截图。
- **真机走查沙箱开关（v4.418，必用）**：`GAEA_WALKTHROUGH=1` 启动壳=workspace 强制指向
  `%TEMP%gaea-walkthroughworkspace`（覆盖用户/项目配置），会话/记忆/交付物全落沙箱，
  走查后删沙箱目录即清场——首条消息自动装配+自动恢复会把回合接进真实会话，此开关是
  唯一根治（2026-09-26 险情防复发）。未设时零行为变化。
- **Vite dev 缓存坑**：同一 CSS/源文件在**同一秒内的两次编辑**，mtime 粒度相同会让
  HMR/转换缓存认不出改动（表现为浏览器仍跑旧样式，reload 也无用）。处置=改完
  `(Get-Item file).LastWriteTime = Get-Date` 再 reload。
- **`vitest` worker 上限**：`maxWorkers` 必须按**物理核**（≈逻辑核/2，夹 [4,8]）而不是
  逻辑核——超配会把单用例墙钟拉到 CPU 时间的数倍，重组件首个 `await import` 直接
  撞 `testTimeout` 假红（详见 `frontend/vite.config.ts` 注释）。

## 发布流程（2026-08-14 修订：补版本资源步骤）

1. 更新 CHANGELOG.md / README.md（版本表）/ wails.json（productVersion）/ releases/README.md（版本表）
2. **同步版本资源**：`build/windows/info.json` 是 Wails 生成版本信息的模板（fixed 段必须含
   `product_version`，否则 exe 的 ProductVersion 为 0.0.0.0）；根目录 `versioninfo.rc` 是遗留物，一并更新以免误导
3. 构建严格按上方「构建入口」与 `docs/gaea-build-guide.md` 执行；以指南当前流程为准，
   不把 `build.bat` 当作唯一路径。产物为 `build/bin/gaea.exe`，发布前必须带三枚语音
   DLL 做真实产物冒烟；不得跳过 bindings 或 `/api/health` 检查
4. 复制 exe 到 `releases/gaea-v<版本>.exe`，生成 `releases/SHA256SUMS-v<版本>.txt`；
   **本地产物只保留最近 5 版**（2026-09-10 用户拍板）——发版后删掉第 6 新的那一版，
   更早版本的身份以 `SHA256SUMS-vX.Y.Z.txt` 为准（旧 exe 不入库）
5. 写 `releases/v<版本>.md` 发布说明（含 SHA256 与冒烟结果），更新 releases/README.md 版本表
6. 冒烟：`scripts/smoke.ps1 -ExePath releases\gaea-v<版本>.exe`（/api/health 200 即通过）
7. 更新 `.gaea/progress.md` 进度记忆与本文件（版本状态）

## 沙箱环境备忘（2026-08-14 整理，详细版见 docs/archive/2026-08-14-sandbox-environment-notes.md）

**防止重蹈覆辙的四条铁律**：
1. `go telemetry off` 已持久生效；构建缓存写入问题随 danger-full-access 策略解除，无需再覆盖 GOCACHE
2. **wails build 前端编译会挂起**（wails 捕获前端输出走管道）——必须 `cd frontend && npm run build`
   再 `wails build -s`（-s = 跳过前端编译，9s 完成）
3. `go test ./...` 单进程树会被 harness 终止、个别测试二进制偶发 `fork/exec Access is denied`——
   逐包验证 + `scripts/test-all.ps1`（逐包/重试/状态续跑）；exec 拒绝用 `go test -c` 手动运行证明代码无恙
4. .ps1 脚本必须 UTF-8 带 BOM（否则 powershell.exe 按 GBK 解析报错）；npx 用 `& 'C:\Program Files\nodejs\npx.cmd'`

## 本地语音引擎（2026-11-24 按代码核对）

- **ASR**：sherpa-onnx，cgo 原生管线（`internal/voice/native`）；VAD 为 silero。DLL 三件套必须与 exe
  同目录（见「构建入口」）。
- **本地 TTS**：Breeze-TTS-2（引擎 id `breeze`，`internal/app/tts_service.go` 按需拉起；流式链独立于
  Base64 管线）。**红线=权重非商业**。云端 TTS 走 grok-tts。
- **已退役**：VoxCPM2（v2.6.9 移除）、CosyVoice2（2026-10 退役）。原「本地 TTS 引擎」段的全部历史与
  坑见 `docs/archive/agents-cleanup-2026-11-24.md`；
  更早的历程见 `docs/archive/2026-08-09-voxcpm2-integration.md` 与
  `docs/archive/2026-08-09-cosyvoice2-llm-gguf-speed-optimization.md`。
- 仍适用的教训：Radeon 8060S 这类核显上 ROCm PyTorch 追不上速度，Vulkan + ggml 才是突破口；
  HuggingFace 直连与 hf-mirror 常不通，ModelScope 直链快。

## 版本状态（当前水位，2026-11-24）

- 产品版本 **4.544.0**（2026-11-17 发版）；之后都是日常装机包，不抬号。版本真源=`wails.json` 的 `productVersion`。
- 2026-11-24 起 main 已全量推送（当时与 origin 一致）；装机位 `C:\AI\gaea-app` =
  包 `releases/gaea-proposal-library-20261124-ddce2d4a/`（知识库退役 + 方案库一级板块 + 文学反馈批）。
- 各线最新进度、包记录与挂池看 `.gaea/progress.md` 顶部和 `.gaea/todos.md`，本节不逐条复述。

## 版本状态（历史存档）

> 逐版历史：`docs/archive/agents-version-history-2026-09.md`（v4.173–v4.146 + v4.49 及更早）；
> 之后的发布记录只在 `CHANGELOG.md` / `releases/`。2026-11-24 清理时从本文件移出的段落
> （11-12~11-14 各线状态速览、长期规划、本地 TTS 引擎等）全文见
> `docs/archive/agents-cleanup-2026-11-24.md`。当前水位见上一节。
