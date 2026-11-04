// voice_v2_go_legs.go — Go 原生管线的 LLM/TTS 腿适配（刀5）。
//
// LLM 腿：直连现役 8792 记忆网关（persona 注入/prepared 台账/功能绑定路由
// 逐字复用已验证链路；用户拍板「跟随聊天绑定→35B」= voice_chat_llm=chat 配置兑现，
// 三根 X-Gaea-Voice 头照旧骑行——网关靠它们做轮次关联）。
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

	"github.com/gaea/gaea/internal/voice/realtime"
)

// voiceLLMLeg 网关 SSE 客户端。
type voiceLLMLeg struct {
	base string // http://127.0.0.1:8792/v1
}

// Reply realtime.ResponseLegs 实现。
func (l voiceLLMLeg) Reply(ctx context.Context, userText string, history []realtime.ChatMessage, turnID string, rev int, itemID string, onDelta func(string)) (string, int, int, error) {
	type msg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	msgs := make([]msg, 0, len(history)+1)
	for _, h := range history {
		msgs = append(msgs, msg{Role: h.Role, Content: h.Content})
	}
	msgs = append(msgs, msg{Role: "user", Content: userText})
	payload, _ := json.Marshal(map[string]any{
		"model":    "qwen3-4b", // 网关强制覆写（compat-only，见 voice_v2_gateway）
		"messages": msgs,
		"stream":   true,
	})
	req, err := http.NewRequestWithContext(ctx, "POST", l.base+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", 0, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer local-spike")
	// 轮次关联三头（sidecar monkeypatch 的原生等价物——走 HTTP 骑行）
	req.Header.Set("X-Gaea-Voice-Turn-Id", turnID)
	if rev >= 0 {
		req.Header.Set("X-Gaea-Voice-Turn-Revision", strconv.Itoa(rev))
	}
	if itemID != "" {
		req.Header.Set("X-Gaea-Voice-Item-Id", itemID)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", 0, 0, fmt.Errorf("网关 %d: %s", resp.StatusCode, string(body))
	}
	var full strings.Builder
	inT, outT := 0, 0
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if chunk.Usage != nil {
			inT, outT = chunk.Usage.PromptTokens, chunk.Usage.CompletionTokens
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			full.WriteString(chunk.Choices[0].Delta.Content)
			onDelta(chunk.Choices[0].Delta.Content)
		}
	}
	return full.String(), inT, outT, sc.Err()
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
	req, err := http.NewRequestWithContext(ctx, "POST", l.base+"/audio/speech", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
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
	llm voiceLLMLeg
	tts voiceTTSLeg
}

func (l voiceResponseLegs) Reply(ctx context.Context, userText string, history []realtime.ChatMessage, turnID string, rev int, itemID string, onDelta func(string)) (string, int, int, error) {
	return l.llm.Reply(ctx, userText, history, turnID, rev, itemID, onDelta)
}

func (l voiceResponseLegs) Speak(ctx context.Context, sentence string, onAudio func([]byte)) error {
	return l.tts.Speak(ctx, sentence, onAudio)
}

var _ realtime.ResponseLegs = voiceResponseLegs{}
