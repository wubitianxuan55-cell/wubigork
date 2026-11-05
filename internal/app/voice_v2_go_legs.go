// voice_v2_go_legs.go — Go 原生管线的 LLM/TTS 腿适配（刀5 引入；刀6 网关退役改进程内）。
//
// LLM 腿（刀6）：8792 网关退役，人格预推进+prompt 装配原位吸收（applyVoiceV2PreparedPrompt
// 语义逐字保留：稳定段 system + 【本轮上下文】独立 user 消息插在末条 user 前，保跨轮
// prefix cache）；路由恒走聊天板块功能绑定（routeModel("chat") 经 ai.Client 流式）——
// voice_chat_llm 三值 radio 随网关一并退役（用户拍板「LLM 腿=35B 挂聊天功能绑定」）。
// TTS 腿：直连 breeze 8012 流式 PCM（16k 优先，24k 回退线性重采样）。
package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/voice/realtime"
)

// voiceNativeLLMLeg 进程内 LLM 腿（原 voiceLLMLeg+8792 网关的组合体）。
type voiceNativeLLMLeg struct {
	a *mediaState
}

// Reply realtime.ResponseLegs 实现。生成前预推进人格状态（prepared 台账），
// 失败/取消回滚，提交由 VoiceV2CommitTurn→takeVoiceV2PreparedTurn 消费——
// 与网关时代的 prepare/rollback/commit 三角语义完全一致。
func (l voiceNativeLLMLeg) Reply(
	ctx context.Context, userText string, history []realtime.ChatMessage,
	turnID string, rev int, itemID string, onDelta func(string),
) (string, int, int, error) {
	userText = strings.TrimSpace(userText)
	// ① 人格预推进（原网关 handleVoiceV2ChatCompletion 同位；plain/无人格=nil）
	var prepared *voiceV2PreparedTurn
	if l.a.app != nil && l.a.app.whisperState != nil {
		prepared = l.a.app.whisperState.prepareVoiceV2Turn(
			l.a.currentVoicePersonality(), userText, itemID, turnID, rev,
		)
	}
	// ② 消息装配（稳定段/动态段/长期记忆注入，applyVoiceV2PreparedPrompt 同位）
	msgs := l.assembleMessages(prepared, userText, history)
	// ③ 聊天板块功能绑定流式（voiceV2ChatLLMStream=网关 chat 腿的可桩点原样骑行）。
	// M5 刀二：注入首批工具 schema（status/save_task），模型按需发起 tool_calls——
	// 截获而非剥除（e211ff9e 反面）；截获处理见 dispatchVoiceV2Tool。
	chunks, err := voiceV2ChatLLMStream(ctx, l.a, msgs, voiceV2ToolSchemas)
	if err != nil {
		l.a.rollbackPrepared(prepared)
		return "", 0, 0, err
	}
	var full strings.Builder
	inT, outT := 0, 0
	var toolCalls []ai.ChatToolCall
	for chunk := range chunks {
		if chunk.Error != "" {
			l.a.rollbackPrepared(prepared)
			return full.String(), inT, outT, fmt.Errorf("voice LLM 流式错误: %s", chunk.Error)
		}
		if chunk.Usage != nil {
			inT, outT = int(chunk.Usage.PromptTokens), int(chunk.Usage.CompletionTokens)
		}
		if len(chunk.ToolCalls) > 0 {
			toolCalls = append(toolCalls, chunk.ToolCalls...)
		}
		if chunk.Done {
			break
		}
		// reasoning delta 不转发——语音不朗读思考（网关 serveVoiceV2ViaChatModel 同位）
		if chunk.Content == "" {
			continue
		}
		full.WriteString(chunk.Content)
		onDelta(chunk.Content)
	}
	// M5 刀二：模型发起工具调用 → 分派（T0 直执行回执/T1 台账+事件出确认卡）。
	// ack 经 onDelta 一次性下发——管线侧 transcript.delta+分句 TTS 同源骑行，
	// 对话落库（CommitTurn）照常。工具轮不取消预推进（ack 就是本轮回复）。
	if len(toolCalls) > 0 && ctx.Err() == nil {
		var preview map[string]interface{}
		ack := ""
		for _, call := range toolCalls {
			ack, preview = l.a.dispatchVoiceV2Tool(call)
			if preview != nil {
				l.a.app.emit(voiceV2ToolPreviewEvent, preview)
			}
			break // 首批单工具语义：一次调用一回复（多 call 余量忽略——频控+话术已兜）
		}
		if ack != "" {
			// 追加而非替换：发起调用前的模型前缀已流经 onDelta（管线半句已播），
			// transcript 必须与已播音频同序一致——前缀+ack 全量落库与朗读。
			full.WriteString(ack)
			onDelta(ack)
		}
	}
	if ctx.Err() != nil {
		// barge-in/取消：预推进无人消费，回滚防幽灵（网关客户端断开回滚同位）
		l.a.rollbackPrepared(prepared)
		return full.String(), inT, outT, ctx.Err()
	}
	return full.String(), inT, outT, nil
}

// assembleMessages 生成 LLM 消息序列：稳定段 system + 历史 + 【本轮上下文】独立
// user 消息 + 本轮 user。动态段独立消息（不改写 user 本体）保证 sidecar 历史
// 逐字节稳定，跨轮公共前缀=[system+全部历史] 全部可复用（P1 缓存命中刀同位）；
// 角色用 user 而非 system：Qwen3.6 模板拒绝会话中 system，user 角色任何模板都收，
// 前缀缓存性质不变。plain 会话（prepared=nil）回退只读长期记忆注入。
func (l voiceNativeLLMLeg) assembleMessages(
	prepared *voiceV2PreparedTurn, userText string, history []realtime.ChatMessage,
) []ai.ChatMessage {
	msgs := make([]ai.ChatMessage, 0, len(history)+3)
	if prepared != nil {
		if system := strings.TrimSpace(prepared.pre.SystemPrompt); system != "" {
			msgs = append(msgs, ai.ChatMessage{Role: "system", Content: "【Gaea 人格与关系稳定设定】\n" + system})
		}
	} else if memory := l.a.voiceV2LongTermContext(userText); memory != "" {
		msgs = append(msgs, ai.ChatMessage{Role: "system", Content: memory})
	}
	for _, h := range history {
		if h.Role == "" || h.Content == "" {
			continue
		}
		msgs = append(msgs, ai.ChatMessage{Role: h.Role, Content: h.Content})
	}
	if prepared != nil {
		if dynamic := strings.TrimSpace(prepared.pre.DynamicContext); dynamic != "" {
			// 管线 history 已剥离末条 user（llmStage：末条 user 由本次请求携带），
			// 插在 history 与本轮 user 之间=网关「末条 user 之前」同位。
			msgs = append(msgs, ai.ChatMessage{Role: "user", Content: "【本轮上下文】\n" + dynamic + "\n【本轮上下文结束】"})
		}
	}
	msgs = append(msgs, ai.ChatMessage{Role: "user", Content: userText})
	return msgs
}

// rollbackPrepared LLM 腿失败路径的预推进回滚（原网关退出路径同位）。
func (a *mediaState) rollbackPrepared(prepared *voiceV2PreparedTurn) {
	if prepared == nil || a.app == nil || a.app.whisperState == nil {
		return
	}
	a.app.whisperState.rollbackVoiceV2PreparedTurn(prepared)
}

// voiceV2LongTermContext plain 会话（无人格预推进）的只读长期记忆注入段：
// 触发词命中+高权重/核心记忆，总预算压在一段短提示内（首轮 prefill 不被回忆拖长）。
func (a *mediaState) voiceV2LongTermContext(userMsg string) string {
	if a.app == nil || a.app.whisperState == nil {
		return ""
	}
	pid := strings.TrimSpace(a.currentVoicePersonality())
	if pid == "" || pid == "plain" {
		return ""
	}
	orch := a.app.whisperState.getOrCreateOrch(pid)
	if orch == nil || orch.FactStore == nil {
		return ""
	}
	candidates := append(orch.FactStore.SearchByTriggers(userMsg), orch.FactStore.SelectForInjection(1200, 0.55, 0, 50)...)
	candidates = append(candidates, orch.FactStore.SelectCoreFacts(4)...)
	seen := map[string]bool{}
	var lines []string
	used := 0
	for _, f := range candidates {
		if f == nil || !f.IsActive() || seen[f.ID] {
			continue
		}
		seen[f.ID] = true
		summary := strings.TrimSpace(f.Summary)
		if summary == "" {
			continue
		}
		line := "- " + summary
		runes := len([]rune(line))
		if used+runes > 900 || len(lines) >= 8 {
			break
		}
		lines = append(lines, line)
		used += runes + 1
	}
	if len(lines) == 0 {
		return ""
	}
	return "【Gaea 长期记忆（只读，可能过时；自然接续，不要逐条复述）】\n" + strings.Join(lines, "\n")
}

// voiceV2ChatLLMStream 语音对话「跟随聊天板块」腿的可桩点（测试替身注入）。
// tools=M5 刀二注入的工具 schema（nil=不带工具面）。
var voiceV2ChatLLMStream = func(ctx context.Context, a *mediaState, msgs []ai.ChatMessage, tools []ai.ChatToolSchema) (<-chan ai.SSEChunk, error) {
	if a.clientRef() == nil {
		return nil, fmt.Errorf("AI 客户端未初始化")
	}
	if a.app == nil {
		return nil, fmt.Errorf("聊天模型路由未初始化")
	}
	eng, model, source, err := a.app.resolveChatModelRoute()
	if err != nil {
		return nil, err
	}
	if model == "" {
		return nil, fmt.Errorf("聊天板块模型不可用")
	}
	return a.clientRef().ChatStream(ctx, &ai.ChatRequest{
		Model: model, EngineID: eng, Feature: "voice_v2", DisableFailover: source == "feature",
		Messages: msgs, Stream: true, Tools: tools,
	})
}

// voiceTTSLeg breeze 流式合成（16k 优先；24k 回退抽点重采样）。
type voiceTTSLeg struct {
	base  string // http://127.0.0.1:8012/v1
	model string
	voice func() string // 人格音色（管线从 session.update 同步）
}

// Speak realtime.ResponseLegs 实现。
func (l voiceTTSLeg) Speak(ctx context.Context, sentence string, onAudio func([]byte)) error {
	payload, _ := json.Marshal(map[string]any{
		"model":           l.model,
		"input":           sentence,
		"voice":           l.voice(),
		"response_format": "pcm",
		"stream_format":   "sse", // 缺省=裸 PCM 流，SSE 解析器将静音（对齐 streamBreezeTTS）
		"stream":          true,
	})
	newReq := func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, "POST", l.base+"/audio/speech", bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	}
	// 共享引擎抢占重试：breeze 同时服务朗读/台词/微信语音，409（generation slot
	// 全被占）是常态竞争而非故障——400ms/800ms/1600ms 退避各重试一次再放弃。
	// 一次 409 就丢整句会让整轮响应零音频（R3 判 cancelled=用户听到半句没有）。
	var resp *http.Response
	var lastErr error
	var err error
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(400*(1<<(attempt-1))) * time.Millisecond):
			}
		}
		req, rerr := newReq()
		if rerr != nil {
			return rerr
		}
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode == http.StatusConflict {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("breeze 409: %s", string(body))
			resp = nil
			continue
		}
		lastErr = nil
		break
	}
	if lastErr != nil {
		return fmt.Errorf("breeze 不可用（重试后仍失败）: %w", lastErr)
	}
	if resp == nil {
		return fmt.Errorf("breeze 不可用（重试耗尽）")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("breeze %d: %s", resp.StatusCode, string(body))
	}
	sr := 24000
	if v := resp.Header.Get("X-Sample-Rate"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			sr = n
		}
	}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 256*1024), 8*1024*1024)
	var carry []float32 // 24k→16k 线性重采样跨块状态
	for sc.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line := sc.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" || data == "" {
			continue
		}
		var ev struct {
			Audio string `json:"audio"`
			Event string `json:"event"`
		}
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			continue
		}
		if ev.Audio == "" {
			continue
		}
		pcm, err := base64.StdEncoding.DecodeString(ev.Audio)
		if err != nil {
			continue
		}
		if sr == 16000 {
			onAudio(pcm)
			continue
		}
		// 24k→16k 线性重采样（2:3；跨块 carry 保相位）
		n := len(pcm) / 2
		cur := make([]float32, 0, len(carry)+n)
		cur = append(cur, carry...)
		for i := 0; i+1 < n*2; i += 2 {
			s := int16(uint16(pcm[i]) | uint16(pcm[i+1])<<8)
			cur = append(cur, float32(s)/32768.0)
		}
		// cur 为样本序列（尾样本落 carry）；以 2/3 步进取样
		total := len(cur)
		outN := (total - 1) * 2 / 3
		out := make([]byte, 0, outN*2)
		for i := 0; i < outN; i++ {
			pos := float64(i) * 3.0 / 2.0
			i0 := int(pos)
			if i0+1 >= total {
				break
			}
			frac := float32(pos - float64(i0))
			v := cur[i0]*(1-frac) + cur[i0+1]*frac
			s := int16(v * 32768.0)
			out = append(out, byte(s), byte(uint16(s)>>8))
		}
		keepFrom := int(float64(outN) * 3.0 / 2.0)
		if keepFrom < total {
			carry = append([]float32(nil), cur[keepFrom:]...)
		} else {
			carry = nil
		}
		if len(out) > 0 {
			onAudio(out)
		}
	}
	return sc.Err()
}

// voiceResponseLegs LLM+TTS 组合腿。
type voiceResponseLegs struct {
	llm voiceNativeLLMLeg
	tts voiceTTSLeg
}

func (l voiceResponseLegs) Reply(ctx context.Context, userText string, history []realtime.ChatMessage, turnID string, rev int, itemID string, onDelta func(string)) (string, int, int, error) {
	return l.llm.Reply(ctx, userText, history, turnID, rev, itemID, onDelta)
}

func (l voiceResponseLegs) Speak(ctx context.Context, sentence string, onAudio func([]byte)) error {
	return l.tts.Speak(ctx, sentence, onAudio)
}

var _ realtime.ResponseLegs = voiceResponseLegs{}
