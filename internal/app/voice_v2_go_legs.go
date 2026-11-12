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
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
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
		// 首批单工具语义：一次调用一回复（多 call 余量忽略——频控+话术已兜）。
		{
			call := toolCalls[0]
			ack, preview = l.a.dispatchVoiceV2Tool(call)
			if preview != nil {
				l.a.app.emit(voiceV2ToolPreviewEvent, preview)
			}
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
	res := &resampler24k16k{} // 24k→16k 重采样（63 阶抗混叠低通+跨块相位保真）
	var prev []byte           // 上一 SSE 帧（延一块发布——帧缝修补需要两侧样本）
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
		// 帧缝修补：流式合成帧边界相位不连续=引擎固有（2026-11-07 音频解析
		// 实锤：51s/188 处跳变，节奏=SSE 帧频，近静音区 Δ0.28=可闻咔哒），
		// 检测到跳变即对缝两侧各 2ms 淡出/淡入（社区解法 fades on clicks）。
		// 延一块发布：接缝修补需要前后两侧样本。
		if prev != nil {
			out, newPrev := seamRepair(prev, pcm, sr)
			prev = newPrev
			if sr == 16000 {
				if len(out) > 0 {
					onAudio(out)
				}
			} else if r := res.push(out); len(r) > 0 {
				onAudio(r)
			}
			continue
		}
		prev = pcm
	}
	if prev != nil { // 末帧照常发布
		if sr == 16000 {
			onAudio(prev)
		} else if r := res.push(prev); len(r) > 0 {
			onAudio(r)
		}
	}
	return sc.Err()
}

// seamRepair 帧缝检测+微修补：接缝跳变 > max(0.05, 6×局部RMS) 时，对前帧
// 尾部/后帧头部各 2ms 做淡出/淡入（把硬台阶变平滑斜坡；社区解法 fades on
// clicks）。返回（修补后的前帧整帧, 新的前帧=后帧）。局部 RMS 取前帧尾部
// 400 样本。16k/24k 通用（w 与阈值按 sr 换算）。自然过渡（含真实爆破音）
// 不修——阈值双条件避免把语音该有的瞬态抹平。
func seamRepair(prev, cur []byte, sr int) (out, newPrev []byte) {
	w := sr / 500 // 2ms
	if w < 8 {
		w = 8
	}
	dec := func(b []byte, i int) float64 {
		if 2*i+1 >= len(b) {
			return 0
		}
		return float64(int16(uint16(b[2*i])|uint16(b[2*i+1])<<8)) / 32768.0
	}
	if len(prev) < 2*w || len(cur) < 2*w {
		return prev, cur
	}
	pn := len(prev) / 2
	rmsSum := 0.0
	rmsN := 400
	if pn-rmsN < 0 {
		rmsN = pn
	}
	for i := pn - rmsN; i < pn; i++ {
		v := dec(prev, i)
		rmsSum += v * v
	}
	rms := math.Sqrt(rmsSum / math.Max(1, float64(rmsN)))
	jump := math.Abs(dec(cur, 0) - dec(prev, pn-1))
	if jump <= math.Max(0.05, 6*rms) {
		return prev, cur
	}
	for i := 0; i < w; i++ {
		t := float64(i) / float64(w)
		idx := pn - w + i
		s := int16(dec(prev, idx) * (1 - t) * 32768.0)
		prev[2*idx] = byte(s)
		prev[2*idx+1] = byte(uint16(s) >> 8)
		s2 := int16(dec(cur, i) * t * 32768.0)
		cur[2*i] = byte(s2)
		cur[2*i+1] = byte(uint16(s2) >> 8)
	}
	return prev, cur
}

// resampler24k16k 24k→16k 重采样器（2:3 降采样，带抗混叠低通）。跨 SSE 块
// 保留剩余样本**与分数相位**：全局第 i 个输出读全局位置 1.5×i。
//
// 两级：①63 阶 Hamming-windowed-sinc FIR 低通（fc≈7.2kHz，24k 域）——抗混叠
// 必需：朴素线性插值抽取会把 8~12kHz 内容（汉语齿音所在频段）折叠进 0~8k
// =不和谐金属音调（社区共识与业界解法：polyphase/低通后抽取，2026-11-07
// 用户听感「仿佛多了许多音调」与此逐字吻合）；②分数相位抽取（线性插值）。
type resampler24k16k struct {
	carry []float32 // 跨块剩余样本（已低通，24k 域）
	phase float64   // carry[0] 相对下一输出位置的分数偏移（0..1.5）
	hist  []float32 // FIR 延迟线尾巴（taps-1 样本，跨块滤波连续性）
}

// firTaps24k16k 63 阶低通系数（懒计算一次；fc=7.2kHz@24k，Hamming 窗）。
var (
	firTaps24k16k     []float32
	firTaps24k16kOnce sync.Once
)

func firTaps() []float32 {
	firTaps24k16kOnce.Do(func() {
		const (
			taps = 63
			fc   = 0.3 // 归一化截止=7.2k/24k（新奈奎斯特 8k 留 10% 过渡带）
		)
		tapsF := make([]float32, taps)
		mid := taps / 2
		sum := 0.0
		for i := 0; i < taps; i++ {
			x := float64(i - mid)
			var s float64
			if x == 0 {
				s = 2 * fc
			} else {
				s = math.Sin(2*math.Pi*fc*x) / (math.Pi * x)
			}
			w := 0.54 - 0.46*math.Cos(2*math.Pi*float64(i)/float64(taps-1))
			tapsF[i] = float32(s * w)
			sum += float64(tapsF[i])
		}
		for i := range tapsF {
			tapsF[i] /= float32(sum) // 直流增益归一
		}
		firTaps24k16k = tapsF
	})
	return firTaps24k16k
}

// push 输入一块 24k PCM16（字节序 LE），返回低通+抽取后的 16k PCM16。
func (r *resampler24k16k) push(pcm []byte) []byte {
	n := len(pcm) / 2
	// ① FIR 低通（延迟线跨块：输入 = hist + 新样本，输出 = 滤波后流）
	buf := make([]float32, 0, len(r.hist)+n)
	buf = append(buf, r.hist...)
	for i := 0; i+1 < n*2; i += 2 {
		s := int16(uint16(pcm[i]) | uint16(pcm[i+1])<<8)
		buf = append(buf, float32(s)/32768.0)
	}
	taps := firTaps()
	filtN := len(buf) - len(taps) + 1
	if filtN <= 0 {
		r.hist = buf
		return nil
	}
	filt := make([]float32, filtN)
	for i := 0; i < filtN; i++ {
		acc := float32(0)
		for j, h := range taps {
			acc += buf[i+j] * h
		}
		filt[i] = acc
	}
	r.hist = append([]float32(nil), buf[len(buf)-len(taps)+1:]...)
	// ② 分数相位抽取（线性插值，跨块保相位）
	cur := append(r.carry, filt...)
	total := len(cur)
	if total < 2 {
		r.carry = cur
		return nil
	}
	outN := int((float64(total) - 1 - r.phase) / 1.5)
	if outN <= 0 {
		r.carry = cur
		return nil
	}
	out := make([]byte, 0, outN*2)
	for i := 0; i < outN; i++ {
		pos := r.phase + float64(i)*1.5
		i0 := int(pos)
		if i0+1 >= total {
			break
		}
		frac := float32(pos - float64(i0))
		v := cur[i0]*(1-frac) + cur[i0+1]*frac
		s := int16(v * 32768.0)
		out = append(out, byte(s), byte(uint16(s)>>8))
	}
	consumed := r.phase + float64(outN)*1.5
	keep := int(consumed)
	if keep > total {
		keep = total
	}
	r.carry = append([]float32(nil), cur[keep:]...)
	r.phase = consumed - float64(keep)
	return out
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

// ProbeResampler24k16k 诊断探针用导出包装（WAV 抓取/离线解析；生产路径不用）。
type ProbeResampler24k16k struct{ r resampler24k16k }

// NewProbeResampler24k16k 构造探针重采样器。
func NewProbeResampler24k16k() *ProbeResampler24k16k { return &ProbeResampler24k16k{} }

// Push 输入 24k PCM16 字节流，输出 16k PCM16。
func (p *ProbeResampler24k16k) Push(pcm []byte) []byte { return p.r.push(pcm) }

// ProbeSeamRepair 诊断探针用导出包装（WAV 抓取/离线解析；生产路径不用）。
func ProbeSeamRepair(prev, cur []byte, sr int) (out, newPrev []byte) {
	return seamRepair(prev, cur, sr)
}
