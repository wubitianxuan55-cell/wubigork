# 轨道 E：Step 3b LLM Seam 完成报告

## 1. 改动文件清单（全部在所有权范围内，共 13 个）

**新增（5）**
- internal/gaea/provider/llm.go — LLM seam 定义（接口/常量/Completion/ChatFromStream/NewLLM/自动适配器）
- internal/gaea/provider/llm_test.go — 注册表互斥/未知 kind fail-closed/按 kind 切换/自动适配/ChatFromStream 聚合（9 个测试）
- internal/gaea/provider/llm_default_test.go — 外部测试包：空 kind 缺省路由到 wubigrok（2 个测试）
- internal/gaea/provider/bridge/llm_seam_test.go — bridge 满足 seam 接口 + Chat 聚合 + 按引擎分支冻结（5 个测试）
- internal/gaea/boot/llm_seam_test.go — NewProvider 缺省/fail-closed/按 kind 切换（3 个测试）

**修改（8）**
- internal/gaea/provider/bridge/bridge.go — Provider 实现 Chat（聚合 Stream）+ 编译期断言 var _ provider.LLMProvider + init 用 provider.LLMKindWubigrok 常量注册
- internal/gaea/boot/boot.go — NewProvider 返回类型改 provider.LLMProvider，内部改经 provider.NewLLM（配置 kind 驱动 + 缺省 wubigrok + fail-closed）
- internal/gaea/agent/agent.go — AgentRunner.prov 字段与 New() 参数由 provider.Provider 改为 provider.LLMProvider（消费者只依赖 seam 接口）
- internal/gaea/agent/task.go — TaskTool.prov/subagentProv 字段、NewTaskTool/SetSubagentProvider/RunSubAgent/RunSubAgentWithSession/runSubAgentInternal 5 处签名改为 LLMProvider
- internal/gaea/agent/helpers_test.go、agent_plan_test.go、evidence_flow_test.go、testutil/mock_provider.go — 测试 mock 补一行 Chat（provider.ChatFromStream 聚合），满足 seam 接口

## 2. 每个 seam 的三元组

| 角色 | 内容 |
|---|---|
| 定义 | provider.LLMProvider{ Provider; Chat(ctx, req) (*Completion, error) }；事件词汇 = Chunk*/ChunkType/Usage/Completion（llm.go）；编译期断言 bridge 与 streamChatAdapter 均满足 |
| 提供者 | bridge 包 init() 以 provider.LLMKindWubigrok="wubigrok" 注册（互斥，重复 panic）；注册表 Register/New/Kinds 为正向先例，未改动其语义 |
| 消费者 | boot.NewProvider（装配，经 NewLLM 只依赖接口）；agent 聊天调用（AgentRunner.prov.Stream/Chat、task 子代理 RunSubAgent、Plan/judge/compact 全部只依赖接口） |

## 3. config 键与默认值

- 复用既有 gaea.toml [[providers]] 段：providers[].kind 驱动 LLM 后端选择（设计文档 §5.3 + 评审 08 §5.6 确认「办公引擎 TOML 配置已含 providers 段，正是换后端只改配置的形态」）；base_url/model/api_key_env 原样透传 provider.Config。
- 新增常量：provider.DefaultLLMKind = "wubigrok"（空 kind 的缺省，与现状一致）；provider.LLMKindWubigrok = "wubigrok"。
- 缺省行为：providers[].kind 为空 → NewLLM 落到 wubigrok（bridge）→ gaea 模型中心，零配置与改造前完全一致（已测试固化）。

## 4. 测试与门禁结果

新增 19 个测试全部通过；go build + go vet + go test（-count=1 全量）三包全绿：
- go build ./internal/gaea/provider/... ./internal/gaea/boot/... ./internal/gaea/agent/... → 0
- go vet 同上 → 0
- go test -count=1 同上 → provider/bridge/boot/agent(+子包) 全部 ok

现有按引擎分支行为冻结（验收硬要求）：bridge Stream 的 herdsman/ollama 思考模式分支原有 3 组测试（Reasoning/CloudNoThinking/MaxTokensGuard），本次补 ollama 同分支（TestBridge_Stream_OllamaEnablesThinking）与空引擎不触发（TestBridge_Stream_EmptyEngineNoThinking），分支行为逐项钉死。

## 5. 遗留缺口（需父代理/其他轨道处理）

1. **app 层 4 处直连 provider.New("wubigrok")**（internal/app，禁止触碰）：gaea_cost_import.go:113、gaea_cost_import_vision.go:518、gaea_knowledge_import.go:91、gaea_summarize.go:37 —— 属「各板块 chat」消费者，尚未切到 seam（provider.NewLLM）。建议父代理或后续 Step 统一改。
2. **新增顶层 [llm] 配置段**（llm.provider/llm.base_url）：任务允许「复用」配置字段，本次复用 providers 段；若需独立顶层 [llm] 段，需改 internal/gaea/config/config.go（跨轨），留待父代理集成（与 Step 3d 配置消费面统一联动）。
3. **bridge.go:41-51 herdsman/ollama 分支**：评审 08 L5 建议将能力标志（supports_thinking）数据化进 EngineConfig/ProviderEntry——涉及 internal/modelengine 与 internal/ai（跨轨），本次只以测试冻结未搬移。
4. **全模块 go build ./... 当前失败**：internal/app/voice_handler.go 引用 voice.Manager.SetASRClient，但 internal/voice/voice_manager.go 已被 ASR/Voice 轨道（并行 WIP）改名删掉该方法——非本轨道引入，属其他轨道中途状态，请父代理协调。
5. **未新增 App 绑定方法**：本 Step 无需新增绑定，无 gen_bindings 需求。