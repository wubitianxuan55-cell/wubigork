package cache

import (
	"sync"
)

// SpawnTemplate 定义子代理的 L4 固定模板——同类子代理共享相同的前缀字节，
// 使 DeepSeek 服务端前缀缓存可以命中该段。
type SpawnTemplate struct {
	Kind        TaskKind
	Prefix      string
	Description string
}

// spawnTemplates 是全局子代理模板注册表
var spawnTemplates struct {
	mu    sync.Mutex
	items map[TaskKind]SpawnTemplate
}

// RegisterSpawnTemplate 注册子代理模板。已注册模板的 Prefix 会被注入为子代理的
// 独立 System 消息（在 L1+L2 之后、User 任务描述之前），使得 DeepSeek 前缀缓存
// 可覆盖 [L1+L2+template] 这段字节。同类子代理共享模板→缓存命中。
func RegisterSpawnTemplate(st SpawnTemplate) {
	if st.Kind == "" || st.Prefix == "" {
		return
	}
	spawnTemplates.mu.Lock()
	defer spawnTemplates.mu.Unlock()
	if spawnTemplates.items == nil {
		spawnTemplates.items = make(map[TaskKind]SpawnTemplate)
	}
	spawnTemplates.items[st.Kind] = st
}

// LookupSpawnTemplate 查找已注册的子代理模板。
func LookupSpawnTemplate(kind TaskKind) (SpawnTemplate, bool) {
	spawnTemplates.mu.Lock()
	defer spawnTemplates.mu.Unlock()
	if spawnTemplates.items == nil {
		return SpawnTemplate{}, false
	}
	t, ok := spawnTemplates.items[kind]
	return t, ok
}

// ─── 内置子代理模板前缀 ─────────────────────────────────────
// 每个模板是固定文本 block，所有同类子代理使用完全相同文本→缓存命中。
// 每个精炼到 ≤200 tok，覆盖核心行为指令但不含具体任务内容。

const subagentExplorePrefix = `你是 Hephaestus 的代码探索子代理。你的任务是在代码库中进行深入调查。

核心规则：
- 读取相关文件，理解代码结构和逻辑
- 只做只读操作，不修改任何文件
- 将调查结果以结构化摘要返回：关键发现、代码路径、调用链
- 返回应自包含——父代理只看到你的最终答案`

const subagentResearchPrefix = `你是 Hephaestus 的外部调研子代理。你的任务是从外部资源收集信息。

核心规则：
- 使用 web_fetch 和 web_search 获取信息
- 引用来源，区分事实与推断
- 只做只读操作，不修改任何文件
- 返回结构化调研报告：结论、论据、来源`

const subagentReviewPrefix = `你是 Hephaestus 的代码审查子代理。你的任务是审查代码变更。

核心规则：
- 读取变更文件，检查正确性、安全性、测试覆盖
- 只做只读操作，不修改任何文件
- 按严重程度分类问题：blocker / major / minor / nit
- 返回审查报告：问题列表 + 改进建议`

const subagentSecurityPrefix = `你是 Hephaestus 的安全审计子代理。你的任务是审计代码的安全性。

核心规则：
- 检查：注入攻击、认证绕过、敏感信息泄露、权限提升
- 只做只读操作，不修改任何文件
- 按风险等级分类：critical / high / medium / low
- 返回审计报告：风险描述 + 攻击路径 + 修复建议`

// receiptFooter 实测反馈④（2026-11-20）：回执末两行对账口径，与
// DefaultTaskSystemPrompt 同源要求。前缀统一拼接，文本恒固定——同类
// 子代理仍共享同一前缀文本，缓存命中不受影响。
// （2026-11-23：subagent_writing 网文创作模板随 novel-agents 技能退役删除，
// 五前缀收敛为四。）
const receiptFooter = `

末尾固定附两行（供父代理横向对账多份回执）：[findings] 关键发现至多 3 条（无则写 none）；[risks] 风险项至多 2 条（无则写 none）。字符/汉字计量由系统自动附加（[metrics] 行），不要自报字数。`

// BuiltinSpawnTemplates 返回内置子代理模板列表，供 boot 阶段注册。
func BuiltinSpawnTemplates() []SpawnTemplate {
	return []SpawnTemplate{
		{Kind: TaskKind("subagent_explore"), Prefix: subagentExplorePrefix + receiptFooter, Description: "代码探索"},
		{Kind: TaskKind("subagent_research"), Prefix: subagentResearchPrefix + receiptFooter, Description: "外部调研"},
		{Kind: TaskKind("subagent_review"), Prefix: subagentReviewPrefix + receiptFooter, Description: "代码审查"},
		{Kind: TaskKind("subagent_security"), Prefix: subagentSecurityPrefix + receiptFooter, Description: "安全审计"},
	}
}
