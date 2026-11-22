package bridge

// cachehit_e2e_test.go — 缓存命中率机制实测（2026-11-21，用户要求实测证据，
// 方法学=上游 Reasonix cachehit_e2e_test：按规范序字节前缀模拟 DeepSeek 自动
// 前缀缓存，跑的是 gaea 真实请求组装路径——bridge 转换 + ai.ChatRequest 序列
// 化 + 真实工具 schema + AgentRunner 多轮驱动）。
//
// 缓存键模型：DeepSeek/OpenAI 兼容端按内部规范序 token 流（system→tools→
// 逐条消息）做前缀缓存。JSON 字段序里 messages 在 tools 前，直接测 JSON 字节
// 会让「追加消息」假红（首版实测踩过）——canonicalBytes 按规范序展开才是忠实
// 模型。
//
// 断言四层：
//  ① append-only：无压缩时每个请求的规范序缓存键是前一个的字节扩展
//    ——会话内 token 流前缀逐字节稳定=缓存全程命中。
//  ② system/tools 字节恒定（线上 JSON 侧断言）。
//  ③ 命中率：64-token 块口径下第 2 次请求起 ≥60%。
//  ④ 重写恢复：会话重写（压缩形态）后前缀断、system+tools 存活（pin）、
//    下一轮稳定性恢复——「重写=缓存部分冷」钉成可量化形态。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// canonicalBytes 服务端缓存键的规范序展开（system→tools→逐条消息）。
func canonicalBytes(req *ai.ChatRequest) []byte {
	var b strings.Builder
	for _, m := range req.Messages {
		if m.Role == "system" {
			b.WriteString(m.Content)
			b.WriteByte(0)
		}
	}
	tj, _ := json.Marshal(req.Tools)
	b.Write(tj)
	b.WriteByte(0)
	for _, m := range req.Messages {
		if m.Role == "system" {
			continue
		}
		b.WriteString(m.Role)
		b.WriteByte(0)
		b.WriteString(m.Content)
		b.WriteByte(0)
		for _, tc := range m.ToolCalls {
			b.WriteString(tc.ID)
			b.WriteByte(0)
			b.WriteString(tc.Function.Name)
			b.WriteByte(0)
			b.WriteString(tc.Function.Arguments)
			b.WriteByte(0)
		}
		b.WriteString(m.ToolCallID)
		b.WriteByte(0)
	}
	return []byte(b.String())
}

// cacheProbeClient 收真实 ai.ChatRequest（bridge 转换产物），json.Marshal 出
// 线上字节（与真实 client 同一 struct 同一 encoding），按规范序测缓存。
type cacheProbeClient struct {
	mu       sync.Mutex
	reqs     [][]byte  // 线上 JSON 字节
	canons   [][]byte  // 规范序缓存键字节
	hitRates []float64 // 每次 Stream 的缓存命中率（首写=0）
	stable   []int     // 与上一次的规范序公共前缀字节数
	prev     []byte
}

func (c *cacheProbeClient) ChatStream(ctx context.Context, req *ai.ChatRequest) (<-chan ai.SSEChunk, error) {
	c.mu.Lock()
	b, _ := json.Marshal(req)
	canon := canonicalBytes(req)
	call := len(c.reqs) + 1
	common := 0
	if c.prev != nil {
		m := min(len(canon), len(c.prev))
		for common < m && canon[common] == c.prev[common] {
			common++
		}
	}
	tokens := func(n int) int { return n/3 + 1 } // 近似：办公中文负载 ~3B/token
	total := tokens(len(canon))
	var hit int
	if c.prev != nil {
		hit = (tokens(common) / 64) * 64 // DeepSeek 64-token 块粒度
	}
	c.reqs = append(c.reqs, b)
	c.canons = append(c.canons, canon)
	c.stable = append(c.stable, common)
	c.hitRates = append(c.hitRates, float64(hit)/float64(total))
	c.prev = canon
	c.mu.Unlock()

	usage := &ai.ChatUsage{
		PromptTokens:          int64(total),
		PromptCacheHitTokens:  int64(hit),
		PromptCacheMissTokens: int64(total - hit),
	}
	var chunks []ai.SSEChunk
	if call == 1 { // 首轮：工具调用（驱动多轮形态）
		chunks = []ai.SSEChunk{{
			Done: true,
			ToolCalls: []ai.ChatToolCall{{
				ID: "c1", Type: "function",
				Function: ai.ChatToolFunction{Name: "echo", Arguments: `{"q":"报表口径"}`},
			}},
			Usage: usage,
		}}
	} else {
		chunks = []ai.SSEChunk{
			{Content: fmt.Sprintf("第 %d 轮答复。", call)},
			{Done: true, Usage: usage},
		}
	}
	ch := make(chan ai.SSEChunk, len(chunks))
	for _, s := range chunks {
		ch <- s
	}
	close(ch)
	return ch, nil
}

// LLMClient 其余方法桩（本测试只走 ChatStream）。
func (c *cacheProbeClient) ChatSimpleStream(ctx context.Context, model, systemPrompt, userMsg string) (string, error) {
	return "", fmt.Errorf("not implemented")
}
func (c *cacheProbeClient) ChatSimpleStreamWithOptions(ctx context.Context, model, systemPrompt, userMsg string, opts ai.ChatSimpleOptions) (string, error) {
	return "", fmt.Errorf("not implemented")
}
func (c *cacheProbeClient) GenerateImage(ctx context.Context, req *ai.ImageGenerationRequest) (*ai.ImageGenerationResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// echoTool 无副作用工具（Registry 全接口形态）。
type echoTool struct{}

func (echoTool) Name() string        { return "echo" }
func (echoTool) Description() string { return "回显工具（缓存 e2e）" }
func (echoTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"q":{"type":"string"}}}`)
}
func (echoTool) ReadOnly() bool { return true }
func (echoTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	return `{"ok":true}`, nil
}

func commonPrefixLen(a, b []byte) int {
	m := min(len(a), len(b))
	n := 0
	for n < m && a[n] == b[n] {
		n++
	}
	return n
}

// sysToolsBytes 线上 JSON 里 system 内容与 tools 的字节（恒定断言用）。
func sysToolsBytes(b []byte) (string, string) {
	var m map[string]json.RawMessage
	if json.Unmarshal(b, &m) != nil {
		return "", ""
	}
	var req ai.ChatRequest
	_ = json.Unmarshal(b, &req)
	sys := ""
	if len(req.Messages) > 0 {
		sys = req.Messages[0].Content
	}
	return sys, string(m["tools"])
}

// TestCacheHitPrefixStable 多轮会话：缓存键 append-only、system/tools 恒定、
// 命中率爬升——「会话内缓存全程命中」的机制级证明。
func TestCacheHitPrefixStable(t *testing.T) {
	pc := &cacheProbeClient{}
	prov := &Provider{name: "probe", model: "deepseek-v4-pro", client: pc}
	sys := strings.Repeat("你是办公助手，遵循工作区规范。", 200) // ~5.4KB 真实量级
	reg := tool.NewRegistry()
	reg.Add(echoTool{})
	sess := agent.NewSession(sys)
	a := agent.New(prov, reg, sess, agent.Options{MaxSteps: 4}, event.FuncSink(func(event.Event) {}))
	ctx := context.Background()
	for _, q := range []string{"第一问：查一下报表口径。", "第二问：汇总。", "第三问：出结论。"} {
		if _, err := a.Run(ctx, q); err != nil {
			t.Fatalf("Run: %v", err)
		}
	}
	if len(pc.canons) < 4 {
		t.Fatalf("应产生 ≥4 次请求（工具轮+收尾×3），实得 %d", len(pc.canons))
	}
	// ① append-only：规范序缓存键逐请求扩展。
	for i := 1; i < len(pc.canons); i++ {
		if !strings.HasPrefix(string(pc.canons[i]), string(pc.canons[i-1])) {
			t.Fatalf("请求 %d 的缓存键不是请求 %d 的前缀扩展（断在 %dB）——token 流前缀不稳定=缓存全冷",
				i+1, i, commonPrefixLen(pc.canons[i], pc.canons[i-1]))
		}
	}
	// ② system/tools 线上字节恒定。
	sys0, tools0 := sysToolsBytes(pc.reqs[0])
	for i := 1; i < len(pc.reqs); i++ {
		s, tl := sysToolsBytes(pc.reqs[i])
		if s != sys0 {
			t.Fatalf("请求 %d 的 system 内容漂移", i+1)
		}
		if tl != tools0 {
			t.Fatalf("请求 %d 的 tools 字节漂移（顺序/内容变化=缓存全冷）", i+1)
		}
	}
	// ③ 命中率：第 2 次起 ≥60%。
	for i := 1; i < len(pc.hitRates); i++ {
		if pc.hitRates[i] < 0.60 {
			t.Fatalf("请求 %d 命中率 %.1f%% < 60%%（64-token 块口径）", i+1, pc.hitRates[i]*100)
		}
	}
	for i, hr := range pc.hitRates {
		t.Logf("请求 %d：canonical_stable=%dB hit=%.1f%%", i+1, pc.stable[i], hr*100)
	}
}

// TestCacheHitRewriteBreaksPrefixThenRecovers 会话重写（压缩形态）：前缀断、
// system+tools 存活（pin）、下一轮稳定性恢复。
func TestCacheHitRewriteBreaksPrefixThenRecovers(t *testing.T) {
	pc := &cacheProbeClient{}
	prov := &Provider{name: "probe", model: "deepseek-v4-pro", client: pc}
	sys := strings.Repeat("你是办公助手，遵循工作区规范。", 200)
	reg := tool.NewRegistry()
	reg.Add(echoTool{})
	sess := agent.NewSession(sys)
	a := agent.New(prov, reg, sess, agent.Options{MaxSteps: 4}, event.FuncSink(func(event.Event) {}))
	ctx := context.Background()
	_, _ = a.Run(ctx, "第一问：查一下报表口径。") // 工具轮+收尾
	_, _ = a.Run(ctx, "第二问：汇总。")
	before := len(pc.canons) // 轮数不假设（runner 步数语义），只要求 ≥2
	if before < 2 {
		t.Fatalf("重写前应 ≥2 次请求，实得 %d", before)
	}

	// 模拟压缩重写：折叠中段为摘要（与 compact.Replace 同机制）。
	msgs := sess.Messages
	compacted := []provider.Message{
		msgs[0], // system 原样（pin 对象）
		{Role: provider.RoleUser, Content: "[会话摘要] 早期对话已折叠，结论：口径已核对。"},
		msgs[len(msgs)-1],
	}
	sess.Replace(compacted)
	_, _ = a.Run(ctx, "第三问：出结论。") // 重写后首请求
	after := len(pc.canons)
	if after != before+1 {
		t.Fatalf("重写后应恰 +1 请求，实得 %d→%d", before, after)
	}
	brk := commonPrefixLen(pc.canons[after-1], pc.canons[before-1])
	if strings.HasPrefix(string(pc.canons[after-1]), string(pc.canons[before-1])) {
		t.Fatal("重写后缓存键应断（Replace 改写了历史）")
	}
	// pin：重写后公共前缀 ≥ system+tools 部分（键头存活）。tools 字节从
	// 实际请求提取（手拼序列化会有偏差——首测 9153 vs 9163 的 10B 差即此）。
	_, tools0 := sysToolsBytes(pc.reqs[0])
	pinFloor := len(sys) + 1 + len(tools0) + 1
	if brk < pinFloor {
		t.Fatalf("重写后公共前缀 %dB < system+tools %dB——键头未被 pin", brk, pinFloor)
	}
	// 恢复：下一轮重新 append-only。
	_, _ = a.Run(ctx, "第四问：补充。")
	if !strings.HasPrefix(string(pc.canons[len(pc.canons)-1]), string(pc.canons[len(pc.canons)-2])) {
		t.Fatal("重写后第 2 轮应恢复 append-only（新前缀已稳定）")
	}
	for i, hr := range pc.hitRates {
		t.Logf("请求 %d：canonical_stable=%dB hit=%.1f%%", i+1, pc.stable[i], hr*100)
	}
}
