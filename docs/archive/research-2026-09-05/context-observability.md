# 上下文与 Token 可观测性开源项目调研（2026-09-05）

调研方式：GitHub API 元数据 + README/官方文档逐个核查。星数为 2026-09-04/05 API 快照。gaea 参照现状：已有上下文页（Go 折叠权威 + 前端渲染）、逐类 signed delta、文件 ±行增量、工具结果懒加载；缺图片缩略卡、图片 token 估算、成本费率 hover。

## 1. langfuse/langfuse

- 链接：https://github.com/langfuse/langfuse ；星数 ~34.2k
- License：混合。LICENSE 主体版权 ClickHouse（2023-2026），`ee/` 目录单独商业许可；主体开源条款未逐行核实（GitHub API 显示 NOASSERTION）
- 活跃度：pushed 2026-09-04，非常活跃
- 亮点：
  - Trace = 嵌套 observations 树（span/event/generation），UI 按时间轴嵌套展示输入/输出/元数据/延迟；每个 generation 记录精确 prompt、响应、token usage（https://langfuse.com/docs/tracing）
  - Token/成本口径：usage 与 cost 均按 bucket（input/output/cached/reasoning 等）拆为**互斥桶**，total 一律由求和得出而非独立存储；模型定价用 regex match_pattern 匹配 + 分档价（如超 200K 输入价、service_tier）；无 ingest 时用 tokenizer 兜底估 token，无法定价则不显示成本（https://langfuse.com/docs/model-usage-and-cost）
  - 按 user/tag/model 聚合 dashboard + Metrics API
- gaea 可借鉴点：互斥 bucket + "total=求和"的记账口径与「regex 模型匹配 + 分档费率」定价表，可直接作为 gaea 成本 hover 的数据模型蓝本。

## 2. Arize-ai/phoenix

- 链接：https://github.com/Arize-ai/phoenix ；星数 ~11.3k
- License：Elastic License 2.0（ELv2）；活跃度：pushed 2026-09-04，main 约 9.9k commits
- 亮点：
  - 基于 OpenTelemetry + OpenInference 语义约定：LLM span 挂 `llm.token_count.prompt/completion/total` 属性，成本 = token 计数 × 定价，在 span/trace 两级 rollup（https://arize.com/docs/phoenix/tracing/how-to-tracing/cost-tracking 、https://arize-ai.github.io/openinference/spec/semantic_conventions.html）
  - 已知坑：root agent span 携带累计 token 与子 LLM span 各自计数会**重复统计**（openinference#3164），聚合去重是该体系公认难点
  - tracing/评测/playground/prompt 管理一体，自动插桩覆盖主流框架与厂商
- gaea 可借鉴点：把 token 计数固化为 span 级属性约定、trace 级只做汇总展示；gaea 的逐请求权威 delta 天然规避父子重复计数，是相对优势。

## 3. AgentOps-AI/AgentOps

- 链接：https://github.com/AgentOps-AI/AgentOps ；星数 ~5.8k
- License：MIT；活跃度：pushed 2026-06-25，近两月放缓
- 亮点：
  - Session drill-down：session replay + step-by-step agent 执行图，装饰器（@session/@agent/@operation/@workflow）生成嵌套 span 层级
  - 花费跟踪：LLM provider spend、workflow 执行定价、API 账单跟踪；多会话/跨会话指标与延迟分析
- gaea 可借鉴点：「会话回放 + 每步成本并排」的 drill-down 叙事，适合 gaea 把上下文页与单次请求 delta 联动成回放视图。

## 4. ccusage/ccusage（原 ryoppippi/ccusage）

- 链接：https://github.com/ccusage/ccusage ；星数 ~18.4k（已迁至独立 org）
- License：GitHub API 显示 NOASSERTION（具体条款未核实）；活跃度：pushed 2026-09-04，非常活跃
- 亮点：
  - 纯本地离线统计：读取 coding agent CLI 本地用量文件（支持 Claude Code 等 18 种工具），产出 daily/weekly/monthly/session 报表 + Claude Code 5 小时 billing blocks（`blocks`）
  - 成本口径：内嵌 **LiteLLM 定价文件快照**（Nix 锁定 + 定时工作流刷新），`--offline` 全离线可用；cache creation 与 cache read **单列**；可在 ccusage.json 按原始模型名覆盖单价
  - `--breakdown` 按模型拆分成本、`--json` 结构化导出、statusline hook 实时显示
- gaea 可借鉴点：本地 JSONL + 内嵌定价快照 + 缓存 token 单列，就是 gaea「日/会话聚合 + 成本费率 hover」可直接对齐的成熟口径。

## 5. 图片 token 估算公式（重点核实）

- **Anthropic 现行官方口径（已核实官方文档，2026-09）**：每 28×28 像素 patch = 1 visual token，成本 = ⌈width/28⌉ × ⌈height/28⌉，且**先按档位缩放再计**：高分辨率档（Claude 4.7+，长边 ≤2576px，上限 4784 tokens）、标准档（其余模型，≤1568px / 1568 tokens）。官方例：1000×1000 → 1296 tokens。来源：https://platform.claude.com/docs/en/build-with-claude/vision
- **旧近似式（社区长期通用）**：tokens ≈ (width × height) / 750；pxpipe 旧实现为 `ceil(w*h/750*1.10)`（ANTHROPIC_PIXELS_PER_TOKEN=750）。pxpipe#26 明确指出 /750 是 28²=784 px² 网格的 ~4-5% 连续近似且忽略档位缩放，**系统性高估大图成本**。来源：https://github.com/teamchong/pxpipe/issues/26 、https://github.com/BerriAI/litellm/issues/20367
- **OpenAI 侧开源实现**：jamesmcroft/openai-image-token-calculator（MIT，https://github.com/jamesmcroft/openai-image-token-calculator），按所选模型规格计算 token/成本，但 README 未列具体常数；流传的 GPT-4o 口径（短边缩至 768px、512px tile、85 base + 170/tile）本次未从 OpenAI 官方页面逐字核实——**未核实**
- 附带发现：teamchong/pxpipe（7.3k，MIT，pushed 2026-09-04，https://github.com/teamchong/pxpipe）——Claude Code 减量工具，实测密集文本约 3.1 字符/图片 token
- gaea 可借鉴点：缩略卡上应同时显示「缩放后尺寸 → 实际计费 token（⌈w/28⌉×⌈h/28⌉）」，避免用裸 /750 误报。

## 6. traceloop/OpenLLMetry（简查）

- https://github.com/traceloop/OpenLLMetry ；~7.4k；Apache-2.0；pushed 2026-08-10
- OpenTelemetry 版 GenAI 插桩 SDK 集，定位在采集端而非 UI；对 gaea 前端呈现借鉴有限，不再展开。

## 未覆盖 / 未核实项

- langfuse 主体 LICENSE 的具体开源条款、ccusage 许可文本：未核实
- OpenAI 官方 vision token 常数（85/170 等）：未核实
- AgentOps 托管 dashboard 的具体 UI 截图细节：仅据 README，未逐页核实

## 综合观察：上下文可观测 UX 趋势

1. **计量与定价解耦**是通用架构：token 计数在采集/权威层，费率表独立维护（langfuse regex+分档价、phoenix span 属性 rollup、ccusage 内嵌 LiteLLM 快照）；gaea 宜内置一份可刷新的费率快照并允许用户按模型覆盖。
2. **缓存与推理 token 必须拆桶**：cached read/creation、reasoning 等作为互斥桶分列、total 恒等于求和，已是 langfuse/ccusage/litellm 共识；成本 hover 逐桶报价是最佳实践。
3. **多模态 token 从面积近似转向 patch 精确式**：Anthropic 已用 28×28 patch（⌈w/28⌉×⌈h/28⌉）取代 /750，且先档位缩放再计；图片可视化的正确形态是「缩略卡 + 缩放后尺寸 + 实际计费 token」三者成对。
4. **聚合要防父子重复计数**：phoenix#3164 证明 root span 累计值与子 span 混算是通病；gaea 以权威逐请求 delta 为唯一口径、rollup 复用同口径即可规避。
5. **drill-down 统一为「聚合 → 逐步 → 单调用」三级联动**：AgentOps 回放、langfuse trace 树、ccusage session 报表均支持从汇总一键下钻；gaea 的上下文页 ↔ 请求 trace 时间线应补锚点互跳，形成同类体验。
