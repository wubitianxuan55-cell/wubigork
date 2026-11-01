package app

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

// streamVoiceTTS 为语音对话提供首段优先的合成路径。当前实时能力来自
// Breeze SSE；其他引擎或流式不可用时退回原有的阻塞合成链，一次交付整段
// 音频，行为与旧链路一致。
func (a *mediaState) streamVoiceTTS(
	ctx context.Context,
	text string,
	voiceDescription string,
	onAudio func(audio []byte, mimeType string) error,
) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("语音合成文本为空")
	}
	if onAudio == nil {
		return fmt.Errorf("语音音频消费回调未设置")
	}

	engineID, model := "", ""
	switch {
	case a.chatVoiceEngine != "" && a.chatVoiceModel != "":
		engineID, model = a.chatVoiceEngine, a.chatVoiceModel
	case a.activeTTSModel != "":
		engineID, model = a.activeTTSEngine, a.activeTTSModel
	}

	if engineID == "breeze" && model != "" {
		if a.engineMgr != nil {
			if eng, ok := a.engineMgr.GetEngine("breeze"); ok && eng.Enabled {
				a.waitLocalTTSReady("breeze")
				chunks, streamErr := a.streamBreezeVoiceSSE(
					ctx, eng.BaseURL, model, text, voiceDescription, onAudio,
				)
				if streamErr == nil {
					return nil
				}
				if ctx.Err() != nil {
					return ctx.Err()
				}
				if chunks > 0 {
					// 已经播出的分片不能再用整句音频兜底，否则会重复朗读。
					return streamErr
				}
				slog.Warn("语音 Breeze 流播未接管，回退整段合成", "error", streamErr)
			}
		}
	}

	audio, mimeType, err := a.synthesizeVoiceTTS(text, voiceDescription)
	if err != nil {
		return err
	}
	return onAudio(audio, mimeType)
}

// streamBreezeVoiceSSE 把 Breeze 的 PCM delta 立即封装成独立 WAV 块交付。
// 返回值 chunks 用于区分「首块前失败可安全回退」与「已播放部分后失败」。
func (a *mediaState) streamBreezeVoiceSSE(
	ctx context.Context,
	baseURL string,
	model string,
	text string,
	voiceDescription string,
	onAudio func(audio []byte, mimeType string) error,
) (int, error) {
	voice, instructions := a.breezeVoiceAndInstruction(voiceDescription)
	body := map[string]interface{}{
		"model":           model,
		"input":           text,
		"response_format": "pcm",
		"stream_format":   "sse",
	}
	if voice != "" {
		body["voice"] = voice
	}
	if instructions != "" {
		body["instructions"] = instructions
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return 0, err
	}

	base := strings.TrimSuffix(strings.TrimRight(baseURL, "/"), "/v1")
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, base+"/v1/audio/speech", strings.NewReader(string(jsonBody)),
	)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("Breeze 流式合成失败（HTTP %d）", resp.StatusCode)
	}

	sampleRate := 24000
	if sr := resp.Header.Get("X-Sample-Rate"); sr != "" {
		_, _ = fmt.Sscanf(sr, "%d", &sampleRate)
	}

	chunks := 0
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 512*1024), 4<<20)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return chunks, err
		}
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var ev breezeSSEEvent
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &ev); err != nil {
			continue
		}
		switch ev.Type {
		case "speech.audio.delta":
			pcm, err := base64.StdEncoding.DecodeString(ev.Audio)
			if err != nil || len(pcm) == 0 {
				continue
			}
			if err := onAudio(wrapPCMAsWAV(pcm, sampleRate), "audio/wav"); err != nil {
				return chunks, err
			}
			chunks++
		case "speech.audio.done":
			if chunks == 0 {
				return 0, fmt.Errorf("Breeze 流式合成未返回音频")
			}
			return chunks, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return chunks, fmt.Errorf("Breeze 流式读取中断: %w", err)
	}
	if chunks == 0 {
		return 0, fmt.Errorf("Breeze 流式合成未返回音频")
	}
	return chunks, nil
}
