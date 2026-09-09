# Unsloth Model Hub 蒸馏规划（2026-09-05 调研落档）

> 触发：用户实测「unsloth 加载本地模型后速度更快」，要求蒸馏 unslothai/unsloth 给 gaea。
> 方针沿用 Univer 蒸馏先例：**取道不取器**——吸收技术路径与编排思路，不引入 Unsloth 任何形态（含 Studio UI，AGPL）。
> 状态（2026-09-08 补）：**✅ 主体已落地**——v4.102.0 modelhub 引擎落库（`6cd891df`）→
> MH1–MH3 随 v4.104.0、MH4+CU1+CU2 随 v4.105.0；§六「herdsman 不可蒸馏」与 §七 真机
> A/B 实测回写随 v4.106.0（结论：迁模型到 unsloth 不推荐）。剩余=真机补验池观察项
> （CU1 首图收益、CU3 `/free` 支持性——本机 ComfyUI 未配置），非欠账刀。

## 一、调研结论（来源：github.com/unslothai/unsloth README + unsloth.ai/docs/basics/api + 本仓库代码核验）

**Unsloth 现状形态**：桌面应用 + Studio（本地 web UI）+ Core（Apache-2.0）；**Studio UI 为 AGPL-3.0**（README 致谢段落明示 license 划分：core Apache 2.0 / Studio UI AGPL 3.0）。后端 = 调优版 llama.cpp（Vulkan/CUDA/ROCm/CPU 四后端，`UNSLOTH_LLAMA_CPP_BACKEND` 可选），训练侧另持 PyTorch（`UNSLOTH_NO_TORCH=1` 可纯 GGUF 运行）。

**「快」的归因（按可蒸馏性分级）**：
1. **Unsloth Dynamic GGUF（UD-Q4_K_XL 等动态量化）**——关键层保高精度、其余压低，权重文件更小 → 磁盘读入快、显存占用低。官方主推 `repo:variant` 写法即指向 UD 变体。
2. **QAT（量化感知训练）量化**——同为权重侧提速。
3. **MTP（multi-token prediction）投机解码**——官方标称推理 1.4~2.2x。属 llama-server 进程侧 flags，gaea 纯客户端取不到。
4. **按模型自动推荐推理参数**——未显式配置时 Studio 自动选 context/temp/top-k 等；GGUF 内嵌自动调优为内置功能。
5. llama.cpp GGUF mmap 加载与首载后 OS 页缓存——通用机制，官方文档未单列，不立项。

**API 面**：同端口双方言（OpenAI `/v1/chat/completions`、`/v1/models` + Anthropic `/v1/messages`），流式/tool calling/视觉均支持；默认 `localhost:8000` 或 `8888`（`-p` 可改）；鉴权 `sk-unsloth-` Bearer Key（服务端只存 hash，等同密码；`0.0.0.0` 暴露+泄露 key=任意代码执行级风险）。CLI `unsloth run --model repo:variant` 加载后打印 endpoint+key。

**契约风险实锤**：官方文档**未公开** load/unload 的 HTTP 端点（加载只走 UI 或 CLI）。gaea 现用的 `POST /api/inference/load` 与 `GET /api/hub/local` 均为 **Studio 内部 API**，版本升级可能漂移。

## 二、gaea 现状（v4.102.0 已落库，6cd891df）

- `modelhub` 引擎类型：BaseURL 固定 `127.0.0.1:8888/v1`（8888 自动转发当前已加载模型，llama 内部端口每次加载会变）；`sk-unsloth-` Key DPAPI 加密落盘（`SetModelHubKey`/`GetModelHubKeyStatus` 脱敏）。
- 模型清单双源合并：`/v1/models`（已加载）+ `/api/hub/local`（完整本地目录，失败降级只列已加载）。
- `StartModelHubModel` 绑定：`POST /api/inference/load {model_path, load_in_4bit:false, force_reload:false}`；`loading` 状态即返回，靠前端刷新列表看结果。
- 门控路由 offline 白名单已含 modelhub；健康探测已适配。
- **缺口实锤**：`internal/ai/client.go:1147` 对**所有**引擎 temperature≤0 一律兜底 0.7 并显式传（`omitempty` 形同虚设）→ Studio 的按模型自动采样调优被恒定 0.7 顶掉。

## 三、蒸馏项（MH1–MH4）

| # | 项 | 落点 | 量级 |
|---|---|---|---|
| MH1 | **采样参数让位**：modelhub 引擎下用户未显式配置时不传 temperature（去掉 0.7 恒兜底的引擎特例），把自动调优让给 Studio。`ai.Request.Temperature` 已 `omitempty`，只需引擎特判贯通到请求构造链 | `internal/ai/client.go:1147` 附近 | 小刀（Go） |
| MH2 | **加载状态机与一键收敛**：模型卡「已加载/加载中/未加载」徽标 + 一键加载后轮询 `/v1/models` 直到目标出现（现状 loading 即返回靠手刷）；超时诚实失败。`cleanModelHubDisplayName` 已备 | 前端 modelcenter + `useEngineState` | 小刀（纯前端，零绑定） |
| MH3 | **UD 量化档位引导**：目录中 `UD-*` 变体置顶 + 「推荐」标记，说明=动态量化（关键层高精度、体积小、加载快）。variant 名在 `/api/hub/local` 数据自含，纯展示层 | modelcenter 模型卡 | 小刀（纯前端，零绑定） |
| MH4 | **常驻预热**：modelhub 启用且用户钉选默认模型时，gaea 启动后台预热（复用 StartModelHubModel），首聊免等待；失败静默降级不阻塞启动。运行态闸用显式武装位（禁 cfg!=nil 惯性） | `internal/app` 启动编排 + 绑定复用 | 中刀 |

**观察项（不立项）**：MTP/Triton kernels=Studio 进程侧，客户端取不到；Anthropic 方言端点 gaea 无需；TTFT/吞吐遥测已有 dsh-context TimingTotals 对齐线，销账不重做；Cloudflare 隧道 `stream:false` 场景 gaea 暂不涉及。

**拒绝清单**：Studio UI 全形态（AGPL 传染）；unsloth 桌面应用打包/分发；把 `/api/inference/load`、`/api/hub/local` 当公开契约承诺；`0.0.0.0` 局域网暴露引导（key 泄露=任意代码执行，README 明示）。

## 四、风险与守护

1. **内部 API 契约漂移**：load/hub-local 均非文档化端点。现有降级（hub-local 失败→只列已加载）保主链路；MH2 落地时补「404/结构变化」测试锚定，漂移时错误文案引导用户走 Studio UI 加载兜底。
2. **幂等语义未证实**：`force_reload:false` 重复加载同模型是否幂等，官方无文档——MH4 实施前真机核对（重复 load 已加载模型），不证实则预热前先查 `/v1/models` 已含目标再跳过。
3. **Key 安全现状已合规**：DPAPI 落盘+Manager 内存持明文+脱敏展示；预热/收敛日志不得打印 Key。
4. **显存互斥（单卡硬约束）**：预热不是零成本——把 8GB+ 模型拉进显存会挤掉当前活跃引擎的模型（换入换出甚至 OOM）。**预热目标必须跟随「当前活跃引擎」，不得无条件预热 modelhub**；ComfyUI 预热同理且更重（见 §六）。

## 五、刀序建议

MH1 → MH2 → MH3（两把小刀可并行，均零绑定）→ MH4（依赖 MH2 的状态收敛；前置=幂等语义真机核对 + 活跃引擎判定，见 §六显存互斥）。CU1/CU2 与 MH 系列文件零交叠可并行；herdsman 不立项（§六）。

## 六、其它本地引擎适用性对照（2026-09-05 补，用户问：herdsman / ComfyUI 能否参考此方式）

**结论先行：herdsman 不可蒸馏（零加载面 API，gaea 是纯客户端无从干预）；ComfyUI 是「常驻预热」最大受益者，但触发点必须重新设计。**

### 6.1 herdsman（OpenAI 兼容，localhost:8080/v1）——不立项

- **API 面全清单核验**（herdsman-api-docs-2026-08-13，git 历史 f0332aeb^）：`/v1/models`、chat/completions、embeddings、rerank、anthropic/messages、images/*、ocr、documents/parse、audio/*、`/api/benchmarks`——**零 load/unload/keep-alive/加载状态端点**。模型生命周期完全由 herdsman 服务端内部管理，`/v1/models` 是能力清单而非已加载状态，gaea 客户端没有 MH2/MH4 的着力点。
- **MH1 同类让位不适用**：文档未记载「省略采样参数→服务端按模型自动调优」机制，行为未证实前改请求构造是无依据的行为变更。唯一动作=**观察项 HS-obs**：真机一次 `curl` 实验对比「带/不带 temperature」的响应差异，证实服务端有默认采样调优后再议。
- **HS-obs 实测补充（2026-09-06，herdsman 在跑 v0.6.4-0-ga95fb2fb）**：`/props`、`/v1/props` 均 404——llama.cpp 式「采样默认参数外露」端点不存在，服务端采样面**无法从外部观测**；`/api/benchmarks` 可读（自报基准记录）。结论：HS-obs 无判定性实验设计，**坐实不立项**。
- 已有的 `HerdsmanModelCatalog` 只读目录（v4.101 线 B）已覆盖「能力分族展示」，MH3 同类引导无增量。

### 6.2 ComfyUI（internal/ai/image_comfyui.go）——预热最大受益者（CU1–CU3）

- **惰性加载实锤**：模型由 workflow 内 CheckpointLoaderSimple/UNETLoader 节点按名加载（gaea 已用端点=object_info/prompt/history/upload/image），首图要把几 GB checkpoint 读进显存（几十秒）——这正是 unsloth「加载快」体感差距的同类场景，预热把「惰性首次」变「提前完成」。
- **CU1 常驻预热（触发点重设计，不能照抄 MH4）**：预热手段=后台提交一次极小空跑 workflow（1-step、64×64）把默认 checkpoint 拉进显存；但**触发点=绘梦模块首次进入（或首次生图任务前），而不是 gaea 启动**——开机不画图却常驻 8GB 生图模型会挤掉聊天主力（§四-4 显存互斥）。ComfyUI 未启动时静默降级。
- **CU2 量化档位引导**（同 MH3 套路，纯展示层零绑定）：checkpoint 文件名自含量化标记（fp8/scaled 等，如 flux1-schnell-fp8），object_info 模型名列表即可标记「fp8=显存省/加载快」并置顶。
- **CU3 显存释放对偶（观察项偏小刀）**：ComfyUI 官方 `POST /free {"unload_models":true,"free_memory":true}`（gaea 从未用过）——绘梦任务完成后或显存吃紧时释放，与 CU1 成「进/出」一对；实施前先真机确认本机 ComfyUI 版本支持该端点。

### 6.3 对照总表

| 蒸馏点 | modelhub (Unsloth) | herdsman | ComfyUI |
|---|---|---|---|
| MH1 采样让位 | ✅ Studio 自动调优 | ❌ 机制未证实（HS-obs 观察） | N/A（workflow 采样参数为显式必要） |
| MH2 状态收敛 | ✅ | ❌ 无加载状态概念 | 部分：绘梦任务已有 /history 轮询；「已缓存」无公开 API，不立项 |
| MH3 量化档位引导 | ✅ UD 变体 | ⚠️ 收益低，目录分族已覆盖 | ✅ CU2 fp8 标记置顶 |
| MH4 常驻预热 | ✅（活跃引擎跟随） | ❌ 无 load 端点 | ✅✅ CU1，但触发点=绘梦页首入，非启动 |

## 七、实测：herdsman vs unsloth 同模型同文件 A/B（2026-09-05，用户要求「别猜，测」）

**条件**：同一本地 GGUF 文件（Qwen3.6-35B-A3B-Aggressive-Q4_K_P 量化版，23.4GB，存于本机模型目录）分别由 herdsman（8080）与 unsloth Studio（8888，`/api/inference/load` 直加载本地路径成功）加载；AMD Radeon 8060S 统一内存 63.6GB；同 prompt（约 300 字中文说明）、流式、`include_usage` 取服务端真实 completion_tokens、**双方均关思考**（unsloth 需显式 `chat_template_kwargs.enable_thinking=false`）、自然 stop。

| 指标 | herdsman | unsloth | 结论 |
|---|---|---|---|
| 解码速度（首→末 token） | **61.5~62.4 tok/s**（块间隔中位 16ms） | 46.0~49.7 tok/s（中位 20ms） | **herdsman 快 ~25%** |
| 首字延迟 TTFT | 2.8s（**固定开销**：只要 8 token 也 2.81s；一次 1.34s） | **0.57~0.76s** | **unsloth 快 ~4x** |
| ~200 token 回答墙钟 | 5.1~6.1s | 4.75~5.35s | 基本打平；临界≈**400 token**，短答 unsloth 先完成、长答 herdsman 先完成 |
| 默认思考模式 | 关（直出中文正文） | **开**——256 token 全烧在英文 reasoning，正文 0 字、finish=length | gaea 若切 modelhub **必须显式关思考**，否则体感大幅变慢 |
| 加载时间 | 27s（热页缓存，unsloth 先读过该文件） | 冷 50s；热加载在 herdsman 常驻时 >2min（内存挤占，剩 20.9GB） | **口径不齐，不下结论** |

**修正 §六/§三口径**：「modelhub 大概率更快」不成立——**本机解码 herdsman 更快，unsloth 赢在 TTFT**。herdsman 的 2.8s 固定首字开销与算力无关（prompt 46 token 且 42 命中缓存），属 herdsman 网关/调度层问题；若该开销可配置消除，herdsman 全面占优。

**HS-obs-2 数据点（2026-09-06）**：`/api/benchmarks` 显示 herdsman 自报基准 `avg_ttft_ms=119`、`avg_tps≈57.4`（Qwen3.6-35B Q4_K_P-2，4096 ctx standard）——**TTFT 119ms 说明其管线路径本身能做到百毫秒级首字**，§7 实测的 2.8s 固定开销不是模型/算力上限，而是真实聊天路径与 bench 路径的网关/调度差异（herdsman 侧配置问题，gaea 不可改，维持观察）。

**落到 gaea 的动作**：① MH1 补一条——modelhub 引擎请求默认携带 `chat_template_kwargs.enable_thinking=false`（专业秘书人设直答；乐园人格另议）；② herdsman 2.8s 固定开销立为观察项 HS-obs-2（查 herdsman 网关配置/版本，非 gaea 侧可改）；③ 「迁 herdsman 模型到 unsloth」**不推荐**作为提速手段——收益只在短答 TTFT，长答与解码均输。

**测法坑（下次照做）**：流式 chunk 计数≠token 数（unsloth/herdsman 恰好 1 块 1 token，但不可依赖）→ 必须 `stream_options.include_usage`；同分词器下「字数/token」两边差 2.5x = 一边输出了另一语言或 reasoning，先看实际文本再解读数字；unsloth 加载中文路径要用 UTF-8 文件体 `--data-binary`（Git Bash 命令行内联中文 JSON 会被 Studio 拒 "error parsing the body"）；`/api/inference/unload` 需带 `model_path`（空体 422）；`/api/inference/status` 有 `loaded[]/loading[]` 数组可轮询。


