# 当日总账（2026-11-11）：knip 尾款清零+天璇退役漏网收口（代码面卫生小批）＋ WHIRL 推理引擎接入（v4.536.0）

> 承接 2026-11-10 晚自主批（空间契约漂移刀+装机+代码图谱全清）后，用户令
> 「继续优化完善 gaea」。拍板池盘点余项全卡用户输入（沙箱实弹五面/A-B 评测/
> 语音 tap 撤除/V5 拍板等），按可自主验证面推进：九月 v4.179 批挂账的
> **knip Unused exports 169 项三态甄别**欠账 + 顺带收口天璇退役漏网。
> 晚间续批：**WHIRL 引擎对擂→模型能力盲测→张量手术→gaea 接入全链收官**
>（用户令「搜索 radiance/更适合本机的推理引擎」起线，详见下篇）。

## 盘点结论：169 项已被后续批次自然排干

- 全类目 knip 复扫（过滤 wailsjs 生成物）：自研未用导出仅剩 **3 项**——
  `startTianxuanModel`/`startStrataModel`（api/engines.ts）+
  `STREAM_INACTIVITY_TIMEOUT_MS`（ChatPage 再导出）。九月欠的 169 项
  大头（unused exported namespace members 159）随其后各批死文件清理/
  重构自然消化，无需逐项甄别。
- 假阳性甄别在案：`public/voice-worklets/*.js`=AudioWorklet URL 运行时
  挂载（useVoiceChatV2 两处 + worklets.test fs 直读），knip 静态面不可见，
  **保留**；`preview-atlas-main.tsx`=代码图谱预览壳（用户真机目检未收，
  上批留树，**保留**）。

## 刀1 天璇退役漏网收口（-23 行，c7da4558）

- **根**：11-05 天璇退役刀（-1243 行）删净了 Go 侧方法与 wailsjs 绑定，但
  漏了三处下游：engines.ts 的 `startTianxuanModel` wrapper、
  `scripts/gen_bindings/main.go` 参数表五条 Tianxuan 条目
  （Engine/Model 的 Status/Start/Stop）。
- **静默带病机制**：`AppFacade = AppAPI & Record<string, (...args) => Promise<unknown>>`
  （types/wails.d.ts:130）索引签名让任何方法名过 tsc——wrapper 调用已不
  存在的绑定编译期零告警，真调用必炸（`undefined is not a function`）。
  属「零调用方死码」故从未实炸，但留一天多一天漂移。
- **Strata 侧同刀**：`startStrataModel` 前端零调用方，useEngineState.test
  注释明示「刻意未接线」；删 wrapper、注释改指
  StrataEngineStart/Stop 档位路径。**Go 侧 StrataModelStart/Stop 绑定与
  集成测试保留**（tested 预留面，不走前端就是了）。
- ChatPage 删 `STREAM_INACTIVITY_TIMEOUT_MS` 无消费方再导出（真源
  chat/constants.ts，useChatStream 直用；SILENCE/HARD 测试仍从 ChatPage
  导入，保留）。
- gen_bindings 顺手修 WebSearch 三行 gofmt 对齐（既有欠账，stash 对照
  实证非本刀引入）。

## 门禁

- 定向 vitest **83/83**（api/engines 全部 12 个消费测试文件里跑了两批 12
  文件：ChatPage 20+useEngineState 8+mock-contract 四件+ModelPanel/
  RuntimePage/Benchmark/EngineSection/ImageGenPanel）。
- `tsc -b` 0、eslint（三改动文件）0、**knip 自研未用导出复扫清零**、
  gofmt 合式。全量 2738 未跑（范围=删零引用导出，tsc 已证全仓零引用；
  定向面覆盖全部直接消费方）。
- 无运行时行为变化（删的全是零调用方），**本批不装机不发版**。

## 杂项清障

- `.tmp/gaea-voice-588b5ad6` 遗留工作树（11-09 语音打包批）移除：先查
  改动=29 条全是 wails build 再生的 wailsjs 绑定物（无真工作）后
  `git worktree remove --force` + 残留 12K 目录清掉 + prune。codex 会话
  工作树（chat-search-reliability）不动。

## 遗留

- knip「Unused exported namespace members 159」与「duplicate exports
  SILENCE|INACTIVITY 别名」为风格面小账，不值得立刀（下次顺手）。
- CI race linux 6 例平台化刀仍挂（gh 未装+API 不可达，远端无法核对，
  不盲动）。
- 代码图谱预览壳与 5231 dev server 仍留树待用户真机目检。

---

# 续批（晚间）：WHIRL 推理引擎接入 gaea（v4.536.0）

> 用户线「搜索更适合本机的推理引擎」三段收官：①调研对擂（WHIRL 0.2.5.1
> 同文件 1.82x 碾压 llama Vulkan：Ornith 104.07 vs 57.32 tok/s，纯内核
> plain 85=1.48x）②模型能力盲测（生产 uncensored 6:4 Ornith，**边缘面
> uncensored 独赢**——Ornith 拒答暴力+软化成人，换模型路线被否）③张量
> 手术（753 张量仅 2 个 BF16 router 不兼容，BF16→F32 位级无损+顺序重排
> 终版 whirlfix2，双引擎全验，**你的模型 @ WHIRL 102.57 tok/s=生产
> llama 53.75 的 1.91x**）。本批=④gaea 代码面接入。

## 刀1 Go 侧（第三受管本地引擎，端口 8126）

- **新建 `internal/app/whirl_proc.go`**（照 qwen35_proc.go exe-spawn+
  strata_proc.go 结构）：单档位 `qwen35_whirl`（whirlfix2.gguf+mmproj，
> `--alias` 稳定 ID，`--balance -np 1`，mmproj 缺失降级跳过）；
  whirlEnsure 幂等拉起（端口预占→bin/GGUF stat→detachedCmd+voiceV2Spawn
  →setWithJob→300s 就绪轮询）；WhirlEngineStart/Stop/Status 三绑定面。
- **engine.go 四处注册**：EngineWhirl 常量（端口注释表+8126）/IsLocal/
  BuiltinEngineTypeOf/NewManager order+种子（BaseURL 8126/v1、免鉴权）。
- **散落名单六处**：managedEngineSpec/catalog、idle 收割目标表
  （registered 四道闸）、chat preflight case "whirl"（绑定按需拉起）、
  model_engine_handler managed 分支（顺清 tianxuan 残留两名）、
  gaea_schedule 换模预估、gaea_vision chat-follow（mmproj 在载）。
- **thinking 通道**（WHIRL 服务端对 qwen35moe 模板缺省开思考，对齐 llama
  生产语义缺省关）：漏斗层 `applyWhirlThinkingDefault`（ChatStream/chatOnce
  两漏斗，未表态→顶层 enable_thinking=false，保护手装短预算请求不被默认
  推理烧空——strata「偶尔正文几个字」教训的对称防线）；prepareStreamRequest
  whirl 分支（显式开才开+预算守护）；bridge case "whirl"（DSH 工具循环开
  思考+守护）。通道口径=官方 server.md §4（顶层 enable_thinking 有效，与
  strata 的 ctk-only 相反）。
- **novel_sampler**：whirl 与 strata 同入 DRY/XTC 排除名单（非标键不发）。
- **config 链五处**：`whirl_path` 键（默认 `C:\AI\whirl`，whirl-server.exe
  与 ROCm 运行库 DLL 同驻）照 strata_path 全链（keys/load/types/save/shelf）。

## 刀2 绑定与前端

- gen_bindings：explicitOverrides 三名→model 板块；四连再生（720 方法，
  +3）；bindingNames.ts 手工对账补三条（-names 只打印不写文件）。
- 前端：engines.ts type 联合+三 wrapper；**新收敛常量
  `MANAGED_ENGINE_IDS=['strata','llama','whirl']`**（utils.tsx），判定面
  六处统一引用（EngineCard/LocalEngineLifecycle/LLMSection/BindSection/
  useEngineState×2）——防第四处 `=== 'xxx' ||` 链再造退役漏网。
- **tianxuan 前端残留清理**（上批漏网 8 处）：LLMSection/BindSection 判定、
  RuntimePage×5（snapshot 键/遥测候选/本地卡/文案）、mock RuntimeMonitor。
  遥测夹具载体 tianxuan→strata 迁移（Go 侧 runtime_monitor 早已只发 strata）。
- mock：Whirl 三方法恒「未运行」诚实空态（照 Qwen35 形态）+
  LegacyWhirlEngineMethods 类型面；语言包 offlineDesc 名单×3 加 WHIRL。

## 门禁

- Go：`go build ./...` 绿；定向包全绿（app 126s/modelengine/ai/bridge/
  config），新增 whirl_proc_test.go 七用例（形状/空档位拒绝/未知档位/
  路径解析/启动命令契约/profile 归一化含 llama 档位反误认/目录接线对账）；
  引擎数断言 8→9 四处；离线停用名单两测试加 whirl；whirlProfileFromModels
  剥 .gguf 后缀（WHIRL served id 无后缀，llama 有）。
- 前端：tsc -b 0；**全量 vitest 4262+ 全绿**（exit 0）；RuntimePage 夹具
  迁移+useEngineState 定时器 4→5；check-bindings-drift OK（720 一致）。
- 版本三源 sync-version.ps1 → **4.536.0**；npm build+wails build（ucrt64
  铁律；坑=unset TMP 致 wails 0xc0000135，须设系统默认值非清空）+smoke
  /api/health 200+装机 `C:\AI\gaea-app`（PID 34236）。

## 装机与现场

- WHIRL 引擎本机安装位 `C:\AI\whirl`（whirl-bakeoff 验证目录迁移，版本
  0.2.5.1，Adrenalin 驱动直驱免 ROCm）；Ornith 19.8GB 已按用户令删除
  （C 盘回血）；模型 `whirlfix2.gguf` 与原件并存于统一模型根（llama 8124
  不动，两引擎并存按需拉起）。
- **真机四关挂用户目检**：①模型中心 WHIRL 卡探活/停止 ②chat 绑定
  qwen35_whirl→首请求拉起→实测 ~100 tok/s ③@图片 vision ④idle 5 分钟
  收割+gaea 退出进程不残留（Job Object 兜底）。

## 遗留

- WHIRL /health 遥测（VRAM 快照形态≠/metrics）未接 RuntimePage——与 llama
  同等未接，后续另议。
- flash-next 68GB 档是否可载 WHIRL（IQ4_XS K-quant 理论可载）未测——
  strata flash 档现役无迁移压力。
- 引擎对擂/盲测/手术全记录在本会话记忆档（WHIRL 线）。

---

# 续批 2（夜）：联网开关根治+思考流实时上屏+引擎官方版本检查（v4.537.0）

> 用户实弹三反馈全链闭环：①「理解问题」排查=聊天页联网开关 useState(true)
> 无持久化，切话题重挂载静默回开、工具 schema 无感注入（引擎日志 0→1
> tools 翻真实锤；微信助手腿硬编码联网为独立面）→ **默认关+localStorage
> 持久化+toolMode 决策日志**（08c2e3f9，复装机实弹验证 0 tools）。
> ②「思考卡等正文才出现」=useChatStream reasoning 帧只进缓冲从不渲染的
> 前端盲区（引擎侧 5.3KB 流式正常）→ **streamReasoning 实时通道**（镜像
> delta 的 rAF 节流；流式行展开态「思考中…」卡，done 回落折叠卡，
> af8c3abe）。③本批=引擎管理加官方最新版本检查。

## 刀：引擎官方版本检查（EngineUpdateCheck/EngineUpdateStatus，8126/8124/8091 三卡通用）

- **通道侦察**（先探后写）：WHIRL 本地 0.2.5.1↔tsaipifong/whirl-llm
  releases（已最新）；llama 本地 0.8.0-dev（uocr2 构建）↔ggml-org/llama.cpp
  v0.9.0；Strata 本地 0.1.56↔Niko1221/Strata v0.2.13。三通道 releases/
  latest 免鉴权全通。
- **`engine_update_check.go`**：spec 表（binPath 函数+repo）→ --version
  探针（exec+HideWindow 5s）+GitHub releases/latest（net/http 8s）→
  首个点分版本号抓取（逐行：点分优先，version: 行允许裸 b-number——防
  MSVC 19.44 尾注误抓）+点分逐段比较（补零）→ **30 分钟内存缓存**（GitHub
  免鉴权限流）。返回 local/latest/has_update/release_url/error；官方侧
  失败仍回本地版本。**只告知不自动升级**（引擎在跑+DLL 同驻约束，自动
  升级风险大于收益）。
- 前端：LocalEngineLifecycle 三卡加「检查更新」钮+结果行（绿=已是最新/
  橙=有新版本+发布页链接/红=失败）；engines.ts wrapper；mock 报错态。
- 门禁：whirl_proc 同款 hermetic 测试（注入探针，bin 缺失不触发 exec/
  缓存命中不打接口/官方失败保留本地）；parse/compare 纯函数表；全 app
  套件 130s 绿；722 方法四连+drift OK；tsc 0+modelcenter 119/119；
  三源 4.537.0；装机 PID 8996。
