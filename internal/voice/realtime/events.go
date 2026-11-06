// Package realtime — 语音 V2 Go 原生 Realtime 层（sidecar 退役刀2）。
//
// 8791 /v1/realtime 说标准 OpenAI Realtime GA 协议，契约 =
// frontend/src/voice/protocol/events.ts（未知/扩展事件必须可容忍通过）。
// 行为参照 .tmp/research-hf-sts（上游 6feb527 快照）的 api/openai_realtime。
package realtime

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// ─────────────────────────── client → server ───────────────────────────

// clientEnvelope 线缆信封（type 分派，其余按事件取用）。
type clientEnvelope struct {
	Type         string          `json:"type"`
	Session      json.RawMessage `json:"session"`
	Audio        string          `json:"audio"` // input_audio_buffer.append（base64 PCM16@16k）
	Item         json.RawMessage `json:"item"`
	Response     json.RawMessage `json:"response"`
	Instructions string          `json:"instructions"`
}

// ServerVadConfig server_vad 轮次检测配置。
type ServerVadConfig struct {
	Type              string   `json:"type"`
	Threshold         *float64 `json:"threshold,omitempty"`
	InterruptResponse *bool    `json:"interrupt_response,omitempty"`
	CreateResponse    *bool    `json:"create_response,omitempty"`
	// M14（vad_handler.py silence_duration_ms 运行时更新同位）：静音断句窗 ms。
	SilenceDurationMs *int `json:"silence_duration_ms,omitempty"`
}

// SessionConfig 会话级配置（session.update 深合并目标）。
type SessionConfig struct {
	Instructions string
	Voice        string
	VAD          ServerVadConfig
}

// sessionUpdateWire session.update 的 session 字段形状（宽容：未知字段忽略）。
type sessionUpdateWire struct {
	Type         string `json:"type"`
	Instructions string `json:"instructions"`
	Audio        *struct {
		Input *struct {
			TurnDetection *ServerVadConfig `json:"turn_detection"`
		} `json:"input"`
		Output *struct {
			Voice string `json:"voice"`
		} `json:"output"`
	} `json:"audio"`
}

func parseSessionUpdate(raw json.RawMessage) (SessionConfig, error) {
	var cfg SessionConfig
	if len(raw) == 0 {
		return cfg, nil
	}
	var w sessionUpdateWire
	if err := json.Unmarshal(raw, &w); err != nil {
		return cfg, fmt.Errorf("session.update 解析失败: %w", err)
	}
	cfg.Instructions = w.Instructions
	if w.Audio != nil {
		if w.Audio.Input != nil && w.Audio.Input.TurnDetection != nil {
			cfg.VAD = *w.Audio.Input.TurnDetection
		}
		if w.Audio.Output != nil {
			cfg.Voice = w.Audio.Output.Voice
		}
	}
	return cfg, nil
}

// decodeAppendAudio base64 PCM16 字节解码（*2 字节/样本，管线层再解释）。
func decodeAppendAudio(b64 string) ([]byte, error) {
	if b64 == "" {
		return nil, nil
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("audio base64 解码失败: %w", err)
	}
	return raw, nil
}

// ─────────────────────────── server → client ───────────────────────────

// 事件构造统一走 map→json（形状按 events.ts；字段宽容是前端契约）。
// 具名 helper 防止散落字面量漂移。

func evSessionCreated(id string, cfg SessionConfig) map[string]any {
	return map[string]any{
		"type": "session.created",
		"session": map[string]any{
			"type":         "realtime",
			"id":           id,
			"instructions": cfg.Instructions,
			"audio":        sessionAudioSnapshot(cfg),
		},
	}
}

func evSessionUpdated(cfg SessionConfig) map[string]any {
	return map[string]any{
		"type": "session.updated",
		"session": map[string]any{
			"type":         "realtime",
			"instructions": cfg.Instructions,
			"audio":        sessionAudioSnapshot(cfg),
		},
	}
}

func sessionAudioSnapshot(cfg SessionConfig) map[string]any {
	vad := map[string]any{"type": "server_vad"}
	if cfg.VAD.Threshold != nil {
		vad["threshold"] = *cfg.VAD.Threshold
	}
	if cfg.VAD.InterruptResponse != nil {
		vad["interrupt_response"] = *cfg.VAD.InterruptResponse
	}
	audio := map[string]any{"input": map[string]any{"turn_detection": vad}}
	if cfg.Voice != "" {
		audio["output"] = map[string]any{"voice": cfg.Voice}
	}
	return audio
}

func evSpeechStarted(itemID string, audioStartMs int) map[string]any {
	return map[string]any{
		"type":           "input_audio_buffer.speech_started",
		"item_id":        itemID,
		"audio_start_ms": audioStartMs,
	}
}

func evSpeechStopped(itemID string, audioEndMs int) map[string]any {
	return map[string]any{
		"type":         "input_audio_buffer.speech_stopped",
		"item_id":      itemID,
		"audio_end_ms": audioEndMs,
	}
}

func evBufferCommitted(itemID string) map[string]any {
	return map[string]any{"type": "input_audio_buffer.committed", "item_id": itemID}
}

func evItemCreated(item map[string]any) map[string]any {
	return map[string]any{"type": "conversation.item.created", "item": item}
}

func evTranscriptionCompleted(itemID, transcript string) map[string]any {
	return map[string]any{
		"type":       "conversation.item.input_audio_transcription.completed",
		"item_id":    itemID,
		"transcript": transcript,
	}
}

// evTranscriptionFailed 转写失败终态（M5，conversation.py on_transcription_failed
// 同位帧形：error.type/code/message/param 逐字对齐）。前端 UnknownServerEvent
// 原样放行，本期协议面先对齐、失败 UX 后续接。
func evTranscriptionFailed(itemID, message string) map[string]any {
	return map[string]any{
		"type":    "conversation.item.input_audio_transcription.failed",
		"item_id": itemID,
		"error": map[string]any{
			"type":    "transcription_error",
			"code":    "transcription_failed",
			"message": message,
			"param":   nil,
		},
	}
}

func evResponseCreated(respID string) map[string]any {
	return map[string]any{
		"type": "response.created",
		"response": map[string]any{
			"id":     respID,
			"status": "in_progress",
		},
	}
}

func evOutputItemAdded(respID, itemID string, outputIndex int) map[string]any {
	return map[string]any{
		"type":         "response.output_item.added",
		"response_id":  respID,
		"output_index": outputIndex,
		"item": map[string]any{
			"type":   "message",
			"id":     itemID,
			"role":   "assistant",
			"status": "in_progress",
		},
	}
}

func evOutputItemDone(respID, itemID string, outputIndex int) map[string]any {
	return map[string]any{
		"type":         "response.output_item.done",
		"response_id":  respID,
		"output_index": outputIndex,
		"item": map[string]any{
			"type":   "message",
			"id":     itemID,
			"role":   "assistant",
			"status": "completed",
		},
	}
}

func evContentPartAdded(respID, itemID string) map[string]any {
	return map[string]any{
		"type":        "response.content_part.added",
		"response_id": respID,
		"item_id":     itemID,
	}
}

func evContentPartDone(respID, itemID string) map[string]any {
	return map[string]any{
		"type":        "response.content_part.done",
		"response_id": respID,
		"item_id":     itemID,
	}
}

func evTranscriptDelta(respID, itemID, delta string) map[string]any {
	return map[string]any{
		"type":        "response.output_audio_transcript.delta",
		"response_id": respID,
		"item_id":     itemID,
		"delta":       delta,
	}
}

func evTranscriptDone(respID, itemID, transcript string) map[string]any {
	return map[string]any{
		"type":        "response.output_audio_transcript.done",
		"response_id": respID,
		"item_id":     itemID,
		"transcript":  transcript,
	}
}

func evAudioDelta(respID, itemID string, pcm []byte) map[string]any {
	return map[string]any{
		"type":        "response.output_audio.delta",
		"response_id": respID,
		"item_id":     itemID,
		"delta":       base64.StdEncoding.EncodeToString(pcm),
	}
}

func evAudioDone(respID, itemID string) map[string]any {
	return map[string]any{
		"type":        "response.output_audio.done",
		"response_id": respID,
		"item_id":     itemID,
	}
}

// ResponseDonePayload response.done 的 response 体。
type ResponseDonePayload struct {
	ID            string         `json:"id"`
	Status        string         `json:"status"`
	StatusDetails map[string]any `json:"status_details,omitempty"`
	Usage         map[string]any `json:"usage,omitempty"`
}

func evResponseDone(p ResponseDonePayload) map[string]any {
	body := map[string]any{"id": p.ID, "status": p.Status}
	if p.StatusDetails != nil {
		body["status_details"] = p.StatusDetails
	}
	if p.Usage != nil {
		body["usage"] = p.Usage
	}
	return map[string]any{"type": "response.done", "response": body}
}

func evError(message string) map[string]any {
	return map[string]any{
		"type":  "error",
		"error": map[string]any{"message": message},
	}
}
