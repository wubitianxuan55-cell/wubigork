package app

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Breeze 真·实时流播（v2）：单请求 SSE（服务端 split_chars 自行分片 + delta
// 推流），PCM delta 包 44 字节迷你 WAV 头后按 tts-stream chunk 事件直发——
// TTSPlayer 队列按到达顺序消费（index 仅驱动进度显示），前端零改动。
// 首音从逐句链的 1~2s 压到 ~300ms，句间无缝。请求建立失败（服务未起/不可达）
// 返回 false 走回退逐句链；首 delta 之后的失败已 emit error，本次朗读由
// breeze 接管不回退（半途换引擎会出现两种音色拼接）。

// wrapPCMAsWAV 把裸 s16le 单声道 PCM 包成规范 44 字节头 WAV（24kHz）。
func wrapPCMAsWAV(pcm []byte, sampleRate int) []byte {
	out := make([]byte, 44+len(pcm))
	copy(out, "RIFF")
	binary.LittleEndian.PutUint32(out[4:], uint32(36+len(pcm)))
	copy(out[8:], "WAVE")
	copy(out[12:], "fmt ")
	binary.LittleEndian.PutUint32(out[16:], 16) // fmt 块长
	binary.LittleEndian.PutUint16(out[20:], 1)  // PCM
	binary.LittleEndian.PutUint16(out[22:], 1)  // 单声道
	binary.LittleEndian.PutUint32(out[24:], uint32(sampleRate))
	binary.LittleEndian.PutUint32(out[28:], uint32(sampleRate*2)) // 字节率 = sr*2ch*2B
	binary.LittleEndian.PutUint16(out[32:], 2)                    // 块对齐
	binary.LittleEndian.PutUint16(out[34:], 16)                   // 位深
	copy(out[36:], "data")
	binary.LittleEndian.PutUint32(out[40:], uint32(len(pcm)))
	copy(out[44:], pcm)
	return out
}

// breezeVoiceAndInstruction breeze 音色/语气解析单一来源（provider 构造与流播
// 共用）：角色预设 voiceId > 面板全局选择；描述=入参人格化描述 > 角色预设
// 描述 > 全局 breeze 描述。
func (a *mediaState) breezeVoiceAndInstruction(voiceDescription string) (voice, instructions string) {
	presetVoice, presetDesc := a.voicePersonaPreset(a.currentVoicePersonality())
	voice = presetVoice
	if voice == "" {
		voice = strings.TrimSpace(a.activeTTSVoice)
	}
	instructions = voiceDescription
	if instructions == "" {
		instructions = presetDesc
	}
	if instructions == "" && a.cfg != nil {
		instructions = strings.TrimSpace(a.cfg.BreezeVoiceDesc)
	}
	return voice, instructions
}

// breezeSSEEvent SSE data 行载荷（speech.audio.delta / speech.audio.done）。
type breezeSSEEvent struct {
	Type  string `json:"type"`
	Audio string `json:"audio"`
}

// 流播就绪路径测试注入点：真装机跑单测绝不能 ensure 拉起真实 GPU 服务
// （breezeCmdFor 会 glob 到 C:\AI\breeze-tts2 真装并加载模型）——Fallback
// 用例双桩隔离，生产语义不变。
var (
	breezeStreamEnsure       = func(a *mediaState) { a.ensureLocalTTSService("breeze") }
	breezeStreamProbe        = func(a *mediaState) bool { return a.ttsReady() }
	breezeStreamPollInterval = 200 * time.Millisecond
	breezeStreamPollAttempts = 60
)

// streamBreezeTTS 尝试接管本次流式朗读。返回 handled=true 表示已接管（成功
// 与否都已 emit，调用方直接返回）；false = 未接管（调用方走回退逐句链）。
func (a *mediaState) streamBreezeTTS(text string) bool {
	if a.engineMgr == nil {
		return false
	}
	eng, ok := a.engineMgr.GetEngine("breeze")
	if !ok || !eng.Enabled {
		return false
	}
	model := strings.TrimSpace(a.activeTTSModel)
	if model == "" {
		model = "breeze-tts-2"
	}
	// 服务须在线（加载完成才开始监听）：未起则按需拉起并短等（~12s）；
	// 起不来回退逐句链（edge/herdsman 兜底，不把朗读挂死在 breeze 上）。
	if !breezeStreamProbe(a) {
		breezeStreamEnsure(a)
		for i := 0; i < breezeStreamPollAttempts && !breezeStreamProbe(a); i++ {
			time.Sleep(breezeStreamPollInterval)
		}
		if !breezeStreamProbe(a) {
			return false
		}
	}

	voice, instructions := a.breezeVoiceAndInstruction("")
	reqBody := map[string]interface{}{
		"input":           text,
		"response_format": "pcm",
		"stream_format":   "sse",
	}
	if voice != "" {
		reqBody["voice"] = voice
	}
	if instructions != "" {
		reqBody["instructions"] = instructions
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return false
	}
	// 端点拼接与 tts.openAIEndpoint 同惯例：baseURL 可能带/不带 /v1，统一 .../v1/audio/speech
	base := strings.TrimSuffix(strings.TrimRight(eng.BaseURL, "/"), "/v1")
	httpReq, err := http.NewRequest(http.MethodPost, base+"/v1/audio/speech",
		bytes.NewReader(jsonBody))
	if err != nil {
		return false
	}
	httpReq.Header.Set("Content-Type", "application/json")
	// 流式读无总超时：连接/响应头由传输层管，读中断靠 body 错误与 chunk 间隔暴露
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return false // 未接管：回退逐句链
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}

	sampleRate := 24000
	if sr := resp.Header.Get("X-Sample-Rate"); sr != "" {
		_, _ = fmt.Sscanf(sr, "%d", &sampleRate)
	}

	a.emit("tts-stream", map[string]interface{}{"type": "progress", "index": 0, "total": 0, "text": ""})
	chunkIdx := 0
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 512*1024), 4<<20) // delta chunk 可达数百 KB
	for scanner.Scan() {
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
			a.emit("tts-stream", map[string]interface{}{
				"type":     "chunk",
				"index":    chunkIdx,
				"total":    0,
				"audio":    base64.StdEncoding.EncodeToString(wrapPCMAsWAV(pcm, sampleRate)),
				"mimeType": "audio/wav",
				"engine":   "breeze",
				"done":     false,
			})
			chunkIdx++
		case "speech.audio.done":
			a.emit("tts-stream", map[string]interface{}{"type": "done", "engine": "breeze"})
			return true
		}
	}
	if err := scanner.Err(); err != nil {
		a.emit("tts-stream", map[string]interface{}{"type": "error", "error": "Breeze 流播中断: " + err.Error()})
		return true
	}
	// 流自然结束但未见 done 事件：按完成收尾（数据已全部转发）
	if chunkIdx > 0 {
		a.emit("tts-stream", map[string]interface{}{"type": "done", "engine": "breeze"})
		return true
	}
	return false // 零 delta：服务端行为异常，回退逐句链
}
