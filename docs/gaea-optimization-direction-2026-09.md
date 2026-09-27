# gaea 下一阶段优化方向（2026-09-26 全仓审计衍生 · 候选池）

> **来源**：2026-09-26 全仓审计——三条并行模块审计（①创作与陪伴域 ②生产/工作域 ③平台与壳层域）+ 本机实测复现（构建/测试/产物/启动/本机底座探活）+ 外部对标调研。
> **类型**：优化方向**候选池**——未排刀、未抬版本、不改动任何权威路线的优先级。排刀权在用户。
> **与权威路线的关系**：不改写 `docs/gaea-next-stage-plan-2026-09.md`（活跃指导）与 `docs/gaea-stage7-plan-2026-09.md`（阶段七，4/5 出口判据已过）；本文只做两件事：把审计发现登记成可执行优化项、把已经漂移的旧文件对齐事实。冲突时以既有权威路线为先。
> **评分口径**：作为**个人自用工具**（2026-09-08 拍板的路线 A）的「好用 / 可信 / 可维护」，10 分制。行数用 `(Get-Content $f).Count` 累加（`Measure-Object -Line` 会少计），已排除 `clones/`、`node_modules/`、`dist/`、`build/`、`backups/`。
> **综合评分 7.0 / 10**；14 个模块区间 5.5 ~ 8.0。

---

## §0 一页读

1. **工程纪律是真的，且可核查**：版本三处互锁、产物 SHA256 与 `SHA256SUMS-*.txt` 逐位对得上（本次复算三方一致）、720 绑定面有生成器 + CI diff + 类型级双向断言三层防漂移；4,764 个 Go 测试函数与 3,388 个前端用例本次全量实跑全绿。这是全仓最可信的部分（8.5/10）。
2. **两笔最大的账是「门禁落地」与「规模收敛」**：本地 `scripts/ci.ps1` 不含 lint / 绑定漂移 / `-race` / 冒烟；唯一 workflow 三个 job 有两个结构性跑不通（frontend 跑 `npm ci` 但仓库只有 `pnpm-lock.yaml`；race job 在 ubuntu 因 Windows-only syscall 编译失败）。同时 `internal/app` 已是 5.0 万行 / 1,791 个导出方法 / 依赖 108 个内部包，且存在实测到的无锁并发访问。
3. **唯一无法靠写代码补的缺口是「效果验证」**：技能结晶复用率、路由建议采纳、录制复跑成功率、逐刀真机走查全部挂账；零外部依赖的记忆质量门禁（`internal/memoryeval`）已作为死代码删除，而文档仍在宣称「CI 一键跑出四基线」。

三类优化（按投入产出排序）：**① 门禁从纸面到落地 → ② 规模与台账减负 → ③ 效果验证闭环**。

---

## §1 实测基线（本次复现，可核对）

| 项 | 结果 |
|---|---|
| `go build ./...` | exit 0，11.0s |
| `go vet ./...` | exit 0，3.8s（零输出） |
| `go test ./internal/... .` | exit 0，2.3 分钟，全包 ok |
| 前端 `vitest run` | **399 文件 / 3,388 例全绿**，234.2s |
| 绑定面 | 11 个门面 **720** 方法；生成器从 6 个接收者收集 **1,001** 个导出方法，去重后 720（**281 个同名遮蔽**，App 版本胜出） |
| Go 规模 | 源码 852 文件 / 197,233 行；测试 817 文件 / 144,801 行；`internal/` 141 个包 0 导入环 |
| 前端规模 | `src` 源码 706 文件 / 142,900 行；测试 399 文件 / 54,625 行 |
| 产物身份 | `releases/gaea-v4.420.0.exe` SHA256 = `a4c9aa43…674e` = `SHA256SUMS-v4.420.0.txt` 记录 = 桌面副本（**三方一致**） |
| 启动冒烟 | 启动到 `/api/health` 200 约 **0.5s**；常驻 230 MB（另有 WebView2 子进程 166 MB） |
| 本机底座 | Herdsman `:8080` 提供 **13 个本地模型**（35B/27B 对话、bge-m3、reranker、OCR、ASR、TTS、zimage-turbo）；CosyVoice `:8010` 在线；ComfyUI `:8188` 按需拉起 |

复现命令见 §6。**已知与本机差异**：本机无 gcc，`-race` 未实际执行（race job 的编译失败是用 `GOOS=linux go build` 交叉编译复现的）。

---

## §2 评分

### 2.1 分维（10 分制）

| 维度 | 分 | 关键依据（含保留意见） |
|---|---|---|
| 工程纪律与可验证性 | 8.5 | 三链对账成立；扣分＝文档数字陈述无守卫、单版提交 75% 是发版台账 |
| 测试体量与真实度 | 8.0 | 4,764 / 3,388 实跑全绿，回归测试写根因；扣分＝8 个包零测试、无覆盖率门槛 |
| 门禁真实覆盖 | 4.5 | 见 §3 P0-1：lint/race/drift/workflow 四件未落地 |
| 架构与可维护性 | 6.0 | 141 包 0 环、app 单一组装根；扣分＝上帝包、平行实现分叉、281 遮蔽绑定 |
| 性能与资源占用 | 6.0 | 启动与常驻可接受；扣分＝自审热区未修（`docs/gaea-backend-perf-survey-2026-09.md` 20 项）、exe 51.4 MB 未达 ≤40 MB 目标 |
| 产品纵深与差异化 | 8.0 | 办公真编辑/小说场景制/造价个人库/三脑记忆真闭环；扣分＝效果全部挂账、每域撞专才 |
| 外部依赖与韧性 | 4.5 | 本地底座真实；扣分＝7 域横切 × 上游高速漂移、bus factor=1 |
| 文档与知识资产 | 7.5 | 260 余份文档、逐版根因/证伪/事故实录罕见；扣分＝体积膨胀与数字漂移 |

### 2.2 逐模块

| 模块 | 定性 | 规模（后端自有 / 测试；前端） | 分 |
|---|---|---|---|
| 通用办公 gaea | 闭环但欠账 | gaea 68.4k / 1,836 + office 3.5k/47；前端 99.1k | 8.0 |
| 进度计划 schedule | 完整闭环 | 4.1k / 120（测试≈实现 90%）；前端 18.6k（含测试） | 8.0 |
| 原罪 sin | 完整闭环 | app/sin_* 4.0k / 83；前端 6.0k / 99 例 | 8.0 |
| 绘梦 imagegen | 闭环但欠账 | image 家族 2.1k / 测试比 1.18；前端 9.7k / 141 例 | 8.0 |
| 模型中心 modelcenter | 完整闭环 | 10.5k / 401；前端 8.0k / 121 例 | 8.0 |
| 壳层与导航 shell | 闭环但欠账（效果未验） | 545 / 21 + app 1.1k / 23；契约面 1.7k | 8.0 |
| 小说 novel | 闭环但欠账 | 15 包 11.7k / 203 + app 5.2k / 123；前端 23.5k | 7.5 |
| 对话 chat | 闭环但欠账 | 自有 4.4k / 100（背后共享引擎 13.6k） | 7.5 |
| 轻语 whisper | 闭环但欠账（约 10% 死码） | 20.3k / 396（测试比 0.35 最低） | 7.0 |
| 造价数据库 cost | 完整闭环 | 7 包 6.5k / 115；前端约 5.7k | 7.0 |
| 记忆中枢 memoryhub | 机制建成·效果未验证 | 9 包 8.8k / 230；前端 10.9k | 7.0 |
| 角色库 characterlib | 闭环但欠账 | 2.7k / 43 + app 2.3k / 38；前端 5.1k | 6.5 |
| 微信青鸟 weixin | beta | 2.3k / 62 + app 1.4k / 31；前端 1.7k | 6.0 |
| 编程 code | 闭环但欠账（壳外本体） | gaea 侧仅 267 行 / 17 | 5.5 |

> 模块定性的共同点：**机制建成、接线欠账**，且活性常由测试而非用户维持（whisper 的挂死字段与 test-only 文件、小说缺模板恒 fallback、绘梦 T2/T4 空位、原罪自曝「两个代理各以为对方会补」）。因此 `func Test` 计数不能当能力面证据——须单列「引用只来自 `*_test.go`」一档（见 P1-6）。

---

## §3 优化方向

### P0-1 门禁从纸面到落地（成本：0.5 天，收益：让「CI 绿」这句话重新可信）

- **论点**：「全量 ci 绿」在 CHANGELOG 出现 70+ 次，指的是本地 `scripts/ci.ps1` 绿；而该脚本不含 golangci-lint、不含 `scripts/check-bindings-drift.ps1`、不含 `-race`、不含冒烟、无覆盖率。`.github/workflows/ci.yml` 三个 job 里两个结构性跑不通。
- **证据**：① frontend job 跑 `npm ci`，但仓库自 v4.371 起只有 `frontend/pnpm-lock.yaml`，无 `package-lock.json`（实测目录内只有 pnpm-lock.yaml）；② race job 在 ubuntu 上对 12 个包跑 `-race`，而 `internal/gaea/tool/builtin/hide_window_other.go:15`、`bash.go:174/238`、`bash.go:513`、`bgjobs.go:154` 使用 `syscall.Handle` / `SysProcAttr.HideWindow` —— `_other.go` 后缀不是有效 GOOS 约束，Linux 下实测 `[build failed]`；③ `.golangci.yml` 是 v1 配置结构而 action 用 `version: latest`，且 CHANGELOG 全文 `golangci` 零命中（无生效证据）。
- **动作**：三选一并写进文档口径，不留下「声称有门禁」的模糊地带。① 把 drift + 钉版本的 golangci + 可跑的 race 并进 `ci.ps1`；② 修 workflow（补 `pnpm install --frozen-lockfile`、给 Windows-only 文件加 `//go:build windows`、钉 lint 版本、Go job 去掉「整批重试一次」改成与前端同款的在册 flaky 分类器）；③ 保留现状但把脚本改名「本地快闸」并在 `README.md` 与 `AGENTS.md` 明示 ≠ GitHub 门禁。
- **验收**：`scripts/ci.ps1` 与 workflow 至少有一条路径能真正拦住「绑定面漂移 / 未使用符号 / 竞态」三类回归；文档不再出现无定语境的「CI 绿」。

### P0-2 修实测竞态，并把 `internal/app` 纳入 race（成本：0.5 天）

- **论点**：最大且最有状态的包被显式排除在 `-race` 之外（ci.yml 注释：race 下 TempDir 清理竞争 flaky 先例），而该包内存在确凿的无锁并发访问。
- **证据**：`internal/app/app.go:141-142` 声明 `comfyUICancel` / `comfyUICmd`；写侧五处无锁——`internal/app/image_handler.go:1036`、`:1070`、`:1093`、`image_handler.go:1108-1109`（`cmd.Wait()` goroutine）、`image_handler.go:1295-1297`（UI 线程调用的 `GetComfyUIStatus`）；`:1161-1167` 读并清引用。
- **动作**：把这两个字段收进一个带 `sync.Mutex` 的小结构体（或复用既有 `imageGenMu`）；修好后把 `internal/app` 分批纳入 race job（先只放不带 TempDir 的用例子集）。
- **验收**：新增用例在 `-race` 下稳定通过；该字段不再有裸写。

### P0-3 发版台账自动化（成本：1 天，收益：每版固定开销从 9 个文件降到 1）

- **论点**：近 200 次提交的 churn 前 10 名全是台账文件，第一个真代码文件排第 11；单版提交 12 个文件里 9 个是台账（75%）。
- **证据**：v4.420.0（`f4028968`）代码只改 3 个前端文件，提交含 12 文件 9 台账；版本号散在 `internal/app/app_info.go`、`wails.json`、`versioninfo.rc` 三处手改，靠 `scripts/check-version-drift.ps1` 互锁（该闸本身是「v4.324~v4.333 十版未跑 sync-version」事故的产物）；`SHA256SUMS-*.txt` 历史上有四种格式，`scripts/` 下零引用它（无自动核验）。
- **动作**：一条 `scripts/release.ps1`：读版本 → 同步三处 + `releases/README.md` 版本表 → 追加 CHANGELOG 骨架条目 → 构建产物并生成固定格式 SUMS（`<sha256>  <文件名>`，可被 `sha256sum -c` 直接校验）→ 按保留策略删第 6 新版 exe。人手只写发布说明正文。
- **验收**：一次发版的人工文件数 ≤ 3；`sha256sum -c releases/SHA256SUMS-v<X>.txt` 可用。

### P1-1 `internal/app` 拆分（成本：2~3 天，分三步）

- **论点**：221 个源文件 / 50,136 行 / 1,791 个导出方法 / 依赖 108 个内部包，同时承担绑定层、handler 层与组装根三种角色。
- **证据**：最大件 `internal/app/image_handler.go` 1,596 行（ComfyUI 进程生命周期 + 生成 + 进度取消 + 落盘 + 后端配置 + Windows 遥测六组职责，几乎无共享状态）；`gaea_ui_extra.go` 1,188、`whisper_handler.go` 1,019；31 个文件 >800 行。
- **动作**：① 先拆 `image_handler.go` → `image_lifecycle.go` / `image_generate.go` / `image_progress.go` / `platform_stats.go`；② 再按板块把 `gaea_*` handler 归入子包（保持绑定方法签名不变，门面委托层不动）；③ 每次拆分后跑 `go test ./internal/app/...` 与漂移闸。
- **验收**：`internal/app` 单文件不超过 800 行；绑定面 720 不变；漂移闸 OK。

### P1-2 清理 281 个同名遮蔽绑定（成本：1 天）

- **论点**：生成器从 6 个接收者收集 1,001 个导出方法，按名字去重后只暴露 720——被遮蔽的 281 份仍编译、仍可被包内代码直接调用，从而绕过 App 版本里的修复。
- **证据**：`scripts/gen_bindings/main.go:385-403` 显式去重（App 版本胜出）；接收者分布 App 615 / OfficeB 209 / writingState 162 / NovelB 126 / mediaState 88 / whisperState 74 / CoreB 70 / core 54（合计 1,001）。
- **动作**：写一个诊断脚本列出 281 个重名对与其调用点；对「内嵌状态类型上也定义同名方法」的写法逐个改名或删除（保留语义唯一的一份）；在 `gen_bindings` 里把「重名遮蔽」从静默改为**显式告警/可选 fail**。
- **验收**：重名数归零或全部有注释说明；生成器对新增重名会报错。

### P1-3 收敛已分叉的平行实现（成本：2~4 天）

- **论点**：同一语义多份实现已经开始行为分叉，比单纯重复更危险。
- **证据**：`truncateRunes` 在 8 个包各有一份、**三种行为**（补 `...` / 补 `…` / 取 n-1）；`featureModel` 在 5~6 个包逐字相同；检索/记忆入口 **11 套**（`gaea/memory` fan-in 52 是事实主干，另有 knowledge / semantic / retrieval / bm25 / wssearch / internal/search / internal/memory / whisper 向量与 FTS 等）；对话流 **3 套**（play 聊天 / work 办公 / settings ChatPanel）；前端 `gaea/lib/mock` 是第三份手写契约（23 文件 6,791 行）。
- **动作**：① 先收敛零风险的纯函数重复（`truncateRunes` → `internal/gaea/strutil`；`featureModel` → 一处）；② 给检索/记忆指定真相源（建议 `gaea/memory`），其余显式标注为「降级路径」并在文档登记；③ mock 从「手写同源」改为「生成或契约测试兜底」（现在靠 10 个 `mock-contract-*.test.ts` 共 879 行兜）。
- **验收**：`truncateRunes` 只剩一份且有行为测试；检索入口在文档里有唯一真相源；mock 漂移能被 CI 捕获。

### P1-4 效果验证闭环（成本：1 天立规则 + 30 天窗口）

- **论点**：这是唯一不能靠写代码补的缺口，也是阶段七唯一未过的出口判据。
- **证据**：`docs/gaea-stage7-plan-2026-09.md` 记「7.2 结晶技能 ≥5 次真实调用 ≈09-30 核数」「录制复跑 ≥80% 待真机」「7.3-2 缺省翻转候拍板」「v4.319~v4.332 各刀走查挂池」；记忆侧，零外部依赖的四域基线包 `internal/memoryeval` 已于提交 `68c56bc1`（2026-09-25）作为死代码删除，而 `docs/memory-eval-set.md` 与 `docs/README.md` 仍在宣称「CI 一键跑出四域基线 recall@10=1.000」（本次实测确认该包不存在）。
- **动作**：① 记忆质量门禁二选一——恢复一个零外部依赖的评测入口，或把 `docs/memory-eval-set.md` 明确改标为「历史基线，已无 CI 入口」（本次已做后者，见 §4）；② 给技能结晶 / 路由建议 / 记忆检索各定 30 天窗口与最小样本量，到期**要么通过、要么冻结线程**，不再继续挂池；③ 7.3-2 缺省翻转明确拍板一次（翻或不翻都要落档）。
- **验收**：三处挂账全部收敛为「通过 / 冻结」二态；文档不再出现无终点的「待真机」。

### P1-5 模块级必修四件（成本：各 0.5~1 天）

| 模块 | 问题 | 证据 | 动作 |
|---|---|---|---|
| 角色库 | 自陈缺口未收口：副本→库无回写通道、外观锚点分裂 | `docs/gaea-character-domain-survey-2026-09.md`「这正是『角色在章节和出图里是同一个人』要防的事」；一致性评分是单次视觉 LLM 主观分，无金标/无校准 | 落「外观锚点单一来源 + 显式回写按钮 + 关联即快照」三条出口判据；评分加可信度标注 |
| 对话 | 无「停止生成」入口 | `frontend/src/hooks/useChatStream.ts:76` sending 期间直接 return，长回合只能等 30s 超时或切话题 | 补停止按钮（复用既有取消链路）；顺带把 GenUI 预算常量合成单一来源 |
| 小说 | 审查链恒走 fallback；「场景驱动」名不副实 | `internal/chapter/chapter.go:148-152` 取不到 `chapter-review` 模板即走内置 fallback，而 `prompts/` 无该文件、`internal/chapter` 是唯一零测试包；`create_chapter_handler.go:488` 是整章 blob 事后物化单场景 | 二选一：补 `prompts/chapter-review.json` 让它可被提示词工作坊接管，或承认 fallback 为正式路径并补测试；规划档把「场景驱动」从已完成改回进行中 |
| 绘梦 | 前端契约无归一化层，缺字段即炸整页；前端用例密度 0.14 | CHANGELOG v4.415.0「GetSystemStats 契约是 Partial 而 api 层零归一化直透……整页崩」 | 把 Partial→default 归一化固化为 api 层纪律；给全部 `Get*Stats` 类绑定补「字段缺失」用例 |

### P1-6 whisper 死码口径拍板 + 「仅测试引用」单列一档（成本：0.5 天定口径）

- **论点**：审计档的结论是「余 196 项散件、建议不再追刀」，但实测残余更重，且死码工具看不见「挂死字段」这一层。
- **证据**：`docs/gaea-whisper-deadcode-survey-20260925.md` 记 590 不可达 func / 147 文件，刀 4 主体净删 88 文件 9,920 行后基线 549→196；本次复核：全仓零引用 7 文件 550 行 + 仅测试引用若干 + `orchestrator.go:120-121` 每会话真实构造 `ConfirmSvc`/`DeliveryCoord` 但**零读取**（构造函数被调用即算可达，故 deadcode 不报）；`agent_loop_runner_test.go` 覆盖的是生产不可达回路。
- **动作**：① 拍板「蒸馏 = 预置资产池」还是「落地未接线即死码」；② 无论哪种，把「引用只来自 `*_test.go`」列为独立统计档写进脚本（现有 `deadcode -test` 会把这类判为可达）；③ 按口径清一次或落档不再重查。
- **验收**：whisper 域有一份带口径的最终清单；后续审计不再重复争论。

### P1-7 文档数字守卫与过期引用清理（成本：0.5 天）

- **论点**：`check-docs.mjs` 只管孤儿登记 / 悬空引用 / AGENTS 预算 / ps1 BOM，**不校验任何数字**，因此计数类陈述会持续漂移。
- **证据**：`releases/README.md` 自述「221 份校验和」而实存 390 份；`docs/README.md` 头部「45 份顶层文档 + 2 个子目录」与实际不符（本次已改）；`frontend/package.json` 版本停在 4.128.0 而产品版本 4.420.0；`.gaea/AGENTS.md` 顶部「最新发布」停在 v4.419.0。
- **动作**：① 给 `check-docs.mjs` 加一条轻量计数校验（docs 顶层数量、SUMS 数量、releases 说明数量）；② 计数不要在文中硬写，改为「见脚本输出」或加自动生成标记；③ 本次已修的部分见 §4。

### P2-1 甘特 / 网络图的库评估 spike（成本：1 天，含结论落档）

- **论点**：进度计划是单作者最重的前端面（非测试 11,305 行 + 1,111 行 CSS），但采用社区库**只对交互甘特成立**：AOA 双代号时标网络图无对口库，上报件导出不应外包。
- **证据**：`frontend/src/schedule` 目录的第三方 import 仅 `antd` / `@ant-design/icons` / `dayjs` / `react` / `zustand`，`frontend/package.json` 无任何甘特或时间线库；`ganttExport.ts` / `networkExport.ts`（866 行）只依赖 `types` / `calendar` / `aoa*` / `layout` / `store` 的纯函数、**不 import 任何 React 视图组件**（换视图不动导出）；交互侧 `GanttView.tsx` 引用 19 个本地模块、8 处消费 `useScheduleStore`，三视图共享 745 行 `store.ts`。
- **动作**：spike 对表六条硬需求——① 双工期 wd/cd 双条渲染 ② 前锋线 ③ 基线叠加对比 ④ 驱动链高亮 ⑤ WBS 表与条体联动（分组/排序/筛选）⑥ 右键菜单 + 撤销重做。≥2 条落不了即保留自研并落档理由。若换：只换 PDM 交互甘特、保留 `GanttView` 作适配层、导出层零改动、必须动态 import（entry chunk 已从 755 KB 回涨到约 970 KB）。
- **验收**：六个候选（frappe-gantt / wx-react-gantt / dhtmlx-gantt / bryntum / vis-timeline / 自研）一张对照表 + 明确结论。许可证需现场核对（GPL 系与「未来按模块开源」的计划存在张力）。

### P2-2 GenUI 预算常量单一来源（成本：0.5 天）

- **证据**：Go 侧与 TS 侧各有一套预算常量（Go 10 个 / TS 19 个），命名已分叉（`maxChartPoint` vs `maxChartPoints`），`scripts/` 与 workflow 对 genui 零引用＝无守卫。
- **动作**：以 Go 侧 `internal/gaea/genui/validate.go` 为唯一真相源，用生成器产出 TS 常量 + 一个双向断言测试。

### P2-3 仓库体积治理（成本：半天，除历史重写外）

- **证据**：工作区 3,537 MB；`.git` 256 MB；git 跟踪 11 个源码包 tarball 共 166.5 MB（占跟踪体积 83%）；`.tmp` 730 MB；`clones/` 623 MB；`backups/` 120 MB。
- **动作**：① `.tmp` 守卫 `scripts/clean-tmp.ps1` 的瞬态模式补 `uiwalk*` / `home-shots` / `*-shots`（当前只覆盖 `ui-sweep*`，而实测 577 MB 全在 `uiwalk/edgeprofile`，守卫根本扫不到——这是本次唯一的守卫缺口）；② 源码包 tarball 出 VCS 并加 `.gitignore`（历史仍可 `git checkout` 找回；**真要缩 `.git` 的 256 MB 需 filter-repo 重写历史，涉及已推送的 remote，需单独拍板**）；③ `releases/*.exe` 保留策略已有，建议同样脚本化。
- **验收**：`.tmp` 反复堆积能被守卫自动清掉；仓库跟踪体积回到 50 MB 以内（不含历史）。

### P2-4 硬编码收敛策略（低优先，仅登记）

- **证据**：前端 398 个文件含 7,988 处中文字面量；三语字典各约 1,549 键但只有 99/703（14%）文件调用 i18n；样式上 tailwind v4 与 27 个手写 CSS（19,287 行）并存，`design-system/` 只有 12 个 .md（零代码，是纸上规范）。
- **动作**：i18n 已拍板「诚实 zh-only」（内容层单语），故**不做全量铺译**，只收敛：新代码禁止新增硬编码中文（可在 lint 加白名单外提示）；样式侧把 `design-system/` 要么落成组件库、要么降级为参考文档，避免「名义规范 + 实际手写」双层。

---

## §4 本次已执行的整理（2026-09-26，非版本刀，纯文档 + 卫生）

| 动作 | 对象 | 说明 |
|---|---|---|
| 新增 | `docs/gaea-optimization-direction-2026-09.md` | 本文档；已登记 `docs/README.md` |
| 事实对齐 | `docs/README.md` | 头部计数改为脚本输出的真实值；`memory-eval-set.md` 条目改注「包已删、基线转历史」 |
| 事实对齐 | `docs/memory-eval-set.md` | 头部加状态校正：`internal/memoryeval` 已于 `68c56bc1` 删除，`go test ./internal/memoryeval` 不再是有效入口 |
| 事实对齐 | `releases/README.md` | 校验和份数与发布说明份数改为实数；补源码包 tarball 的「出 VCS 候选」注记 |
| 卫生清理 | `.tmp/uiwalk/edgeprofile*` | Edge 无头走查 profile（577 MB，可再生）；**同级 `.js` / `.mjs` 探针脚本与 PNG 截图全部保留** |
| 卫生清理 | `.tmp` 瞬态 | `smoke-gaea.exe`（49 MB）、`node-compile-cache`、`Test*` 测试残留、`gaea-fts-test`、`gaea-memoryeval` 残留 |
| 卫生清理 | `dist/`、根目录过期 exe | 根 `dist/`（10.6 MB，CI 会重建）；根 `gaea.exe`（内嵌 4.412.0 的旧副本）、根 `gaea-v4.420.0.exe`（与 `releases/` 副本同哈希的重复件） |

**明确保留（不动）**：`releases/*.tar.gz`（11 个里程碑源码包，用户拍板保留）、`backups/`（用户备份）、`clones/`（含 `.mpp` 真实样本，删了 `mpp_test.go` 会永远 Skip）、`whisper_data/`（运行时库）、`build/bin/`（当前构建产物）、各版 `releases/*.exe`（保留策略内）。

---

## §5 观察池（本审计未验证项，勿当结论引用）

1. **真机交互未走查**：只做了启动冒烟与健康检查，模块的「手感 / 生成质量 / 多模型实际表现」未经人工验证。
2. **`-race` 未实际执行**：本机无 gcc（`CGO_ENABLED=0`），race job 的编译失败是用交叉编译复现的；P0-2 的竞态是读码确证 + 无锁事实，未由 race 检测器报出。
3. **外部市场事实**：竞品时间线来自 2026-09-26 检索（`web_fetch` 被本机 DNS 策略拦截，改用直连抓原文）；「ChatGPT Work 发布时间」「QClaw 最新形态」等若干条未能核实。
4. **模块级未核实项**：`OutlineNode.SceneRefs` 是否死字段；ohstory T6 书级 `style.md` 注入是否生效；原罪数据是否有加密设施（当前只能确认目录级隔离）；DAG「壳内真机走查（真实三件套样例链）」在 HEAD 上的完成状态；`.mpp` MPP12/14 在干净 checkout 无法证实也无法证伪（真实样本在 gitignored 的 `clones/`）。
5. **评分漂移**：模块分数在 ±0.5 内会因「是否把共享引擎算进本模块」的口径不同而变。

---

## §6 复现命令

```powershell
# 构建与静态检查
go build ./...; go vet ./...

# Go 测试（全量约 2~4 分钟）
go test ./internal/... . -count=1

# 前端测试（约 4 分钟；maxWorkers 按物理核，见 vite.config.ts 注释）
cd frontend; npx vitest run --reporter=dot

# 绑定面（720）与漂移
go run ./scripts/gen_bindings -names | Measure-Object   # 期望 720
powershell -NoProfile -File scripts\check-bindings-drift.ps1

# 产物身份与冒烟
(Get-FileHash releases\gaea-v4.420.0.exe -Algorithm SHA256).Hash
powershell -NoProfile -File scripts\smoke.ps1 -ExePath releases\gaea-v4.420.0.exe -Port 18999

# 仓库卫生守卫（孤儿/悬空/预算/BOM）
node scripts/check-docs.mjs

# 规模统计（禁用 Measure-Object -Line）
Get-ChildItem internal -Recurse -Filter *.go |
  Where-Object { $_.Name -notlike '*_test.go' } |
  ForEach-Object { (Get-Content $_.FullName).Count } | Measure-Object -Sum
```
