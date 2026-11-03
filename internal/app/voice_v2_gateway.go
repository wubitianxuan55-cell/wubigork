// voice_v2_gateway.go — Voice V2 的 Gaea 本机对话网关。
//
// sidecar 仍使用 OpenAI Chat Completions 协议，但不再直连裸 4B：
// sidecar → 127.0.0.1:8792（本服务：人格长期记忆注入）→ 127.0.0.1:8123 4B。
// 网关只读上下文并透明转发 SSE；聊天落库仍由语音台的事务写回完成，避免双写。
package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/httpbridge"
)

const (
	voiceV2GatewayPort = 8792
	voiceV2LLMName     = "qwen3-4b"

	voiceV2ItemIDHeader       = "X-Gaea-Voice-Item-Id"
	voiceV2TurnIDHeader       = "X-Gaea-Voice-Turn-Id"
	voiceV2TurnRevisionHeader = "X-Gaea-Voice-Turn-Revision"
)

// 变量是测试 seam：生产默认连 VoiceV2Ensure 拉起的 4B。
var voiceV2GatewayUpstream = fmt.Sprintf("http://127.0.0.1:%d", voiceV2LLMPort)

var voiceV2GatewayClient = &http.Client{}

func (a *mediaState) startVoiceV2Gateway() {
	a.voiceV2GatewayMu.Lock()
	if a.voiceV2Gateway != nil {
		a.voiceV2GatewayMu.Unlock()
		return
	}
	addr := fmt.Sprintf("127.0.0.1:%d", voiceV2GatewayPort)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/v1/chat/completions", a.handleVoiceV2ChatCompletion)
	srv := &http.Server{
		Addr:              addr,
		Handler:           httpbridge.HostGuard(addr, mux),
		ReadHeaderTimeout: 5 * time.Second,
	}
	a.voiceV2Gateway = srv
	a.voiceV2GatewayMu.Unlock()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("voice v2 gateway panic recovered", "panic", r)
			}
			a.voiceV2GatewayMu.Lock()
			if a.voiceV2Gateway == srv {
				a.voiceV2Gateway = nil
			}
			a.voiceV2GatewayMu.Unlock()
		}()
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Warn("Voice V2 网关启动失败（实时语音 LLM 将不可用）", "error", err, "addr", addr)
		}
	}()
}

func (a *mediaState) stopVoiceV2Gateway() {
	a.voiceV2GatewayMu.Lock()
	srv := a.voiceV2Gateway
	a.voiceV2Gateway = nil
	a.voiceV2GatewayMu.Unlock()
	if srv == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Warn("Voice V2 网关关闭失败", "error", err)
	}
}

func (a *mediaState) handleVoiceV2ChatCompletion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		http.Error(w, "读取语音对话请求失败", http.StatusBadRequest)
		return
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "语音对话请求不是合法 JSON", http.StatusBadRequest)
		return
	}
	// sidecar 的 model_name 只是兼容字段；真实路由固定到 VoiceV2Ensure 拉起的 4B。
	payload["model"] = voiceV2LLMName
	userMsg := lastVoiceV2UserMessage(payload)
	var prepared *voiceV2PreparedTurn
	isStreaming, _ := payload["stream"].(bool)
	if isStreaming && a.app != nil && a.app.whisperState != nil {
		itemID, turnID, revision := voiceV2TurnMetadata(r)
		prepared = a.app.whisperState.prepareVoiceV2Turn(
			a.currentVoicePersonality(), userMsg, itemID, turnID, revision,
		)
	}
	if prepared != nil {
		applyVoiceV2PreparedPrompt(payload, prepared)
	} else if memory := a.voiceV2LongTermContext(userMsg); memory != "" {
		injectVoiceV2SystemContext(payload, memory)
	}
	augmented, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "编码语音对话请求失败", http.StatusInternalServerError)
		return
	}
	url := strings.TrimRight(voiceV2GatewayUpstream, "/") + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, url, bytes.NewReader(augmented))
	if err != nil {
		a.rollbackPrepared(prepared)
		http.Error(w, "构建语音对话上游请求失败", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer local-spike")

	resp, err := voiceV2GatewayClient.Do(req)
	if err != nil {
		a.rollbackPrepared(prepared)
		slog.Warn("Voice V2 网关调用 4B 失败", "error", err)
		http.Error(w, "语音模型暂不可用", http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= http.StatusBadRequest {
		a.rollbackPrepared(prepared)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(resp.StatusCode)
	if err := copyAndFlush(w, resp.Body); err != nil {
		a.rollbackPrepared(prepared)
		slog.Warn("Voice V2 上游 SSE 中断，已回滚未提交回合", "error", err)
	}
}

func (a *mediaState) rollbackPrepared(prepared *voiceV2PreparedTurn) {
	if prepared == nil || a.app == nil || a.app.whisperState == nil {
		return
	}
	a.app.whisperState.rollbackVoiceV2PreparedTurn(prepared)
}

func voiceV2TurnMetadata(r *http.Request) (itemID, turnID string, revision int) {
	if r == nil {
		return "", "", 0
	}
	itemID = strings.TrimSpace(r.Header.Get(voiceV2ItemIDHeader))
	turnID = strings.TrimSpace(r.Header.Get(voiceV2TurnIDHeader))
	if raw := strings.TrimSpace(r.Header.Get(voiceV2TurnRevisionHeader)); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			revision = parsed
		}
	}
	return itemID, turnID, revision
}

func lastVoiceV2UserMessage(payload map[string]interface{}) string {
	messages, _ := payload["messages"].([]interface{})
	for i := len(messages) - 1; i >= 0; i-- {
		m, _ := messages[i].(map[string]interface{})
		if m == nil || m["role"] != "user" {
			continue
		}
		if text, ok := m["content"].(string); ok {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

// injectVoiceV2SystemContext 把只读长期记忆并入既有 system 消息；无 system
// 时前置一条。消息数组原序保持不变，sidecar 的短期对话上下文不受影响。
func injectVoiceV2SystemContext(payload map[string]interface{}, memory string) {
	memory = strings.TrimSpace(memory)
	if memory == "" {
		return
	}
	messages, _ := payload["messages"].([]interface{})
	for _, raw := range messages {
		m, _ := raw.(map[string]interface{})
		if m == nil || m["role"] != "system" {
			continue
		}
		if text, ok := m["content"].(string); ok {
			m["content"] = strings.TrimSpace(text) + "\n\n" + memory
			payload["messages"] = messages
			return
		}
	}
	system := map[string]interface{}{"role": "system", "content": memory}
	payload["messages"] = append([]interface{}{system}, messages...)
}

func applyVoiceV2PreparedPrompt(payload map[string]interface{}, prepared *voiceV2PreparedTurn) {
	if prepared == nil {
		return
	}
	if system := strings.TrimSpace(prepared.pre.SystemPrompt); system != "" {
		injectVoiceV2SystemContext(payload, "【Gaea 人格与关系稳定设定】\n"+system)
	}
	dynamic := strings.TrimSpace(prepared.pre.DynamicContext)
	if dynamic == "" {
		return
	}
	// P1 缓存命中刀（2026-11-04）：动态段改为独立消息插在末条 user 之前，
	// 不再改写 user 消息本体。旧实现把【本轮上下文】拼进 user 消息头部——
	// 下一轮 sidecar 历史回放的是未经增强的原文，公共前缀在第一条 user 就断
	// （4B 实测每轮全量 prefill ~1101 tokens）。独立消息只存在于当前请求，
	// sidecar 历史逐字节稳定，跨轮前缀=[system+全部历史] 全部可复用。
	messages, _ := payload["messages"].([]interface{})
	for i := len(messages) - 1; i >= 0; i-- {
		m, _ := messages[i].(map[string]interface{})
		if m == nil || m["role"] != "user" {
			continue
		}
		ctxMsg := map[string]interface{}{
			"role":    "system",
			"content": "【本轮上下文】\n" + dynamic + "\n【本轮上下文结束】",
		}
		inserted := make([]interface{}, 0, len(messages)+1)
		inserted = append(inserted, messages[:i]...)
		inserted = append(inserted, ctxMsg)
		inserted = append(inserted, messages[i:]...)
		payload["messages"] = inserted
		return
	}
}

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
	// 先放触发词命中，再补高权重/核心记忆；总预算严格压在一段短提示内，
	// 语音首轮不能为了回忆把 4B prefill 重新拖回数秒。
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

func copyAndFlush(w http.ResponseWriter, r io.Reader) error {
	flusher, _ := w.(http.Flusher)
	buf := make([]byte, 32*1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}
