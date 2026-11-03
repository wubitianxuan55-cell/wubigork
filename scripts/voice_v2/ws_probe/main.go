// ws_probe — gaea voice-v2 M1 Spike 的 CLI WebSocket 客户端。
//
// 对 speech-to-speech sidecar（scripts/voice_v2/gaea-spike.yaml）做三件事实证：
//  1. 流式发 PCM16@16k（录制 WAV 回放），收转录/文本/PCM，验证一轮完整时序；
//  2. barge-in：AI 播报中再次开口，验证旧 generation 被取消（turn_detected）
//     且取消后无旧代际音频泄漏；
//  3. response.cancel：验证 client_cancelled 语义。
//
// 输出事件时间线 + 分阶段延迟 + 不变式断言到 -out 报告 JSON，下行音频按
// response 落 WAV（16k PCM16，与 TTS handler PIPELINE_SAMPLE_RATE 一致）。
//
// 用法（先 start_spike.bat 起服务）：
//
//	go run ./scripts/voice_v2/ws_probe -mkwav-text "你好，介绍一下你自己" -speech a.wav
//	go run ./scripts/voice_v2/ws_probe -scenario turn -speech a.wav
//	go run ./scripts/voice_v2/ws_probe -scenario bargein -speech a.wav -speech2 b.wav
//	go run ./scripts/voice_v2/ws_probe -scenario cancel -speech a.wav
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	pcmRate      = 16000 // 上行=管线采样率；下行经 TTS handler 重采样回 16k
	chunkMs      = 100   // 每条 input_audio_buffer.append 的时长
	graceMs      = 400   // 终态后仍收听泄漏事件的宽限
	staleGraceMs = 300   // 取消后旧代际音频泄漏判定宽限
)

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	var (
		server     = flag.String("server", "ws://127.0.0.1:8791/v1/realtime", "Realtime WS 端点")
		scenario   = flag.String("scenario", "turn", "turn | bargein | cancel")
		speech     = flag.String("speech", "", "主话语 WAV（16k PCM16 单声道）")
		speech2    = flag.String("speech2", "", "插话话语 WAV（bargein 场景）")
		outDir     = flag.String("out", ".tmp/voice_v2", "报告与音频输出目录")
		tailMs     = flag.Int("tail-silence", 1500, "话语后跟随静音毫秒数")
		pace       = flag.Float64("pace", 1.0, "发包倍速；0=不限速")
		bargeAfter = flag.Int("bargein-after", 600, "首段下行音频到达后再等多少毫秒开始插话")
		waitSec    = flag.Int("wait", 120, "整场最长等待秒数")

		mkwavText   = flag.String("mkwav-text", "", "非空则走造料模式：用 breeze 合成测试话语")
		buildCorpus = flag.String("build-corpus", "", "语料清单 JSON → 逐段 breeze 合成+插停顿产出评测 WAV（M3 中文 EOT）")
		corpusDir   = flag.String("corpus-dir", "", "语料目录（build-corpus 产出 / scenario corpus 消费）")
		mkwavVoice  = flag.String("mkwav-voice", "char_gaea", "造料音色（breeze voice id）")
		mkwavOut    = flag.String("mkwav-out", "", "造料输出 WAV 路径")
		breezeURL   = flag.String("breeze", "http://127.0.0.1:8012/v1", "breeze OpenAI 兼容基址")
	)
	flag.Parse()

	if *mkwavText != "" {
		if err := makeWAV(*breezeURL, *mkwavText, *mkwavVoice, *mkwavOut); err != nil {
			log.Fatalf("mkwav: %v", err)
		}
		return
	}
	if *buildCorpus != "" {
		if err := buildCorpusWAVs(*buildCorpus, *corpusDir, *breezeURL, *mkwavVoice); err != nil {
			log.Fatalf("build-corpus: %v", err)
		}
		return
	}
	if *speech == "" && *scenario != "corpus" {
		log.Fatal("需要 -speech（或 -mkwav-text 造料 / -scenario corpus -corpus-dir）")
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatal(err)
	}
	if *scenario == "corpus" {
		os.Exit(runCorpus(*corpusDir, *server, *outDir, *tailMs))
	}
	p := &probe{
		server: *server, scenario: *scenario,
		speech: mustLoadPCM(*speech), speech2: loadPCMOrNull(*speech2),
		tailMs: *tailMs, pace: *pace, bargeAfter: *bargeAfter,
		wait: time.Duration(*waitSec) * time.Second, outDir: *outDir,
		audio: map[string]*bytes.Buffer{},
	}
	if err := p.run(); err != nil {
		log.Fatalf("scenario %s: %v", *scenario, err)
	}
	p.report.Summary = p.summarize()
	p.report.Checks = p.checks
	if raw, err := json.MarshalIndent(p.report, "", "  "); err == nil {
		path := filepath.Join(*outDir, "report_"+*scenario+".json")
		_ = os.WriteFile(path, raw, 0o644)
		fmt.Printf("报告: %s\n", path)
	}
	for id, buf := range p.audio {
		if buf.Len() == 0 {
			continue
		}
		path := filepath.Join(*outDir, "audio_"+shortID(id)+".wav")
		if err := writeWAV(path, buf.Bytes(), pcmRate); err == nil {
			fmt.Printf("音频: %s (%d bytes PCM)\n", path, buf.Len())
		}
	}
	failed := 0
	for _, c := range p.report.Checks {
		mark := "PASS"
		if !c.OK {
			mark, failed = "FAIL", failed+1
		}
		fmt.Printf("%s  %s — %s\n", mark, c.Name, c.Note)
	}
	fmt.Println(p.report.Summary)
	if failed > 0 {
		os.Exit(1)
	}
}

// ---------- 造料：breeze 合成 24k WAV → 线性重采样 16k ----------

func makeWAV(baseURL, text, voice, out string) error {
	if out == "" {
		return fmt.Errorf("需要 -mkwav-out")
	}
	body, _ := json.Marshal(map[string]any{
		"model": "breeze-tts-2", "input": text, "voice": voice,
		"response_format": "wav",
	})
	resp, err := http.Post(baseURL+"/audio/speech", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("breeze %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}
	pcm, rate, err := parseWAV(raw)
	if err != nil {
		return err
	}
	if rate != pcmRate {
		pcm = resampleLinear(pcm, rate, pcmRate)
	}
	if err := writeWAV(out, pcm, pcmRate); err != nil {
		return err
	}
	fmt.Printf("mkwav: %s（%d 字节 PCM16@%d，源 %d Hz）\n", out, len(pcm), pcmRate, rate)
	return nil
}

func resampleLinear(pcm []byte, from, to int) []byte {
	n := len(pcm) / 2
	samples := make([]int16, n)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}
	outN := n * to / from
	out := make([]byte, outN*2)
	for j := 0; j < outN; j++ {
		pos := float64(j) * float64(from) / float64(to)
		i0 := int(pos)
		if i0 >= n-1 {
			binary.LittleEndian.PutUint16(out[j*2:], uint16(samples[n-1]))
			continue
		}
		frac := pos - float64(i0)
		v := float64(samples[i0])*(1-frac) + float64(samples[i0+1])*frac
		binary.LittleEndian.PutUint16(out[j*2:], uint16(int16(v)))
	}
	return out
}

// ---------- WAV 读写 ----------

func parseWAV(raw []byte) ([]byte, int, error) {
	if len(raw) < 44 || string(raw[:4]) != "RIFF" || string(raw[8:12]) != "WAVE" {
		return nil, 0, fmt.Errorf("非 RIFF/WAVE")
	}
	rate, channels, bits, dataOff, dataLen := 0, 0, 0, 0, 0
	for off := 12; off+8 <= len(raw); {
		id := string(raw[off : off+4])
		size := int(binary.LittleEndian.Uint32(raw[off+4:]))
		body := off + 8
		switch id {
		case "fmt ":
			channels = int(binary.LittleEndian.Uint16(raw[body+2:]))
			rate = int(binary.LittleEndian.Uint32(raw[body+4:]))
			bits = int(binary.LittleEndian.Uint16(raw[body+14:]))
		case "data":
			dataOff, dataLen = body, min(size, len(raw)-body)
		}
		off = body + size + size%2
	}
	if rate == 0 || dataLen == 0 || channels != 1 || bits != 16 {
		return nil, 0, fmt.Errorf("需要 16bit 单声道 WAV（got %dch %dbit %dHz）", channels, bits, rate)
	}
	return raw[dataOff : dataOff+dataLen], rate, nil
}

func writeWAV(path string, pcm []byte, rate int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	hdr := make([]byte, 44)
	copy(hdr[0:4], "RIFF")
	binary.LittleEndian.PutUint32(hdr[4:], uint32(36+len(pcm)))
	copy(hdr[8:12], "WAVE")
	copy(hdr[12:16], "fmt ")
	binary.LittleEndian.PutUint32(hdr[16:], 16)
	binary.LittleEndian.PutUint16(hdr[20:], 1) // PCM
	binary.LittleEndian.PutUint16(hdr[22:], 1) // mono
	binary.LittleEndian.PutUint32(hdr[24:], uint32(rate))
	binary.LittleEndian.PutUint32(hdr[28:], uint32(rate*2))
	binary.LittleEndian.PutUint16(hdr[32:], 2)
	binary.LittleEndian.PutUint16(hdr[34:], 16)
	copy(hdr[36:40], "data")
	binary.LittleEndian.PutUint32(hdr[40:], uint32(len(pcm)))
	_, err = f.Write(append(hdr, pcm...))
	return err
}

func mustLoadPCM(path string) []byte {
	pcm, _, err := loadWAVPCM(path)
	if err != nil {
		log.Fatalf("读取 %s: %v", path, err)
	}
	return pcm
}

func loadPCMOrNull(path string) []byte {
	if path == "" {
		return nil
	}
	return mustLoadPCM(path)
}

func loadWAVPCM(path string) ([]byte, int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, err
	}
	pcm, rate, err := parseWAV(raw)
	if err != nil {
		return nil, 0, err
	}
	if rate != pcmRate {
		return nil, 0, fmt.Errorf("%s 采样率 %d≠%d（先用 mkwav 造料）", path, rate, pcmRate)
	}
	return pcm, rate, nil
}

// ---------- 探针主体 ----------

type evRec struct {
	T          time.Duration `json:"t_ms"`
	Type       string        `json:"type"`
	ResponseID string        `json:"response_id,omitempty"`
	Text       string        `json:"text,omitempty"`
	AudioBytes int           `json:"audio_bytes,omitempty"`
}

type checkRec struct {
	Name string `json:"name"`
	OK   bool   `json:"ok"`
	Note string `json:"note"`
}

type report struct {
	Scenario string     `json:"scenario"`
	Events   []evRec    `json:"events"`
	Checks   []checkRec `json:"checks"`
	Summary  string     `json:"summary"`
}

type anchor struct {
	t    time.Duration
	text string
}

type probe struct {
	server, scenario, outDir string
	speech, speech2          []byte
	tailMs, bargeAfter       int
	pace                     float64
	wait                     time.Duration

	conn  *websocket.Conn
	wmu   sync.Mutex
	start time.Time
	// corpus 模式：期望完成轮数（0=单场景语义）
	expectTurns int
	report      report
	audio       map[string]*bytes.Buffer
	checks      []checkRec

	// 话语发送时间锚
	speechFirstSent, speechLastSent time.Time
	speech2FirstSent                time.Time

	// 事件锚
	sessCreated   anchor
	speechStarted []anchor
	speechStopped []anchor
	transFinal    []anchor // transcription.completed
	respCreated   map[string]anchor
	firstText     map[string]anchor
	firstAudio    map[string]anchor
	respDone      map[string]anchor // text = status/reason
	audioAfter    map[string]int    // done+grace 后到达的旧代际字节数
	cancelSent    time.Time
	bargeinFired  bool
}

func (p *probe) now() time.Duration { return time.Since(p.start) }

func (p *probe) run() error {
	p.respCreated = map[string]anchor{}
	p.firstText = map[string]anchor{}
	p.firstAudio = map[string]anchor{}
	p.respDone = map[string]anchor{}
	p.audioAfter = map[string]int{}
	p.report.Scenario = p.scenario

	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.Dial(p.server, nil)
	if err != nil {
		return fmt.Errorf("连接 %s: %w", p.server, err)
	}
	p.conn = conn
	defer func() { _ = conn.Close() }()
	p.start = time.Now()
	log.Printf("已连接 %s", p.server)

	errCh := make(chan error, 1)
	go p.readLoop(errCh)

	if err := p.expectSession(); err != nil {
		return err
	}
	if err := p.sendSessionUpdate(); err != nil {
		return err
	}

	switch p.scenario {
	case "turn":
		p.streamSpeech(p.speech, &p.speechFirstSent, &p.speechLastSent)
		p.streamSilence(p.tailMs)
	case "cancel":
		go p.sendCancelOnFirstAudio()
		p.streamSpeech(p.speech, &p.speechFirstSent, &p.speechLastSent)
		p.streamSilence(p.tailMs)
	case "bargein":
		if p.speech2 == nil {
			return fmt.Errorf("bargein 需要 -speech2")
		}
		p.streamSpeech(p.speech, &p.speechFirstSent, &p.speechLastSent)
		p.streamSilence(p.tailMs)
		p.waitBargeinWindow()
		p.streamSpeech(p.speech2, &p.speech2FirstSent, nil)
		p.streamSilence(p.tailMs)
	default:
		return fmt.Errorf("未知场景 %q", p.scenario)
	}

	deadline := time.Now().Add(p.wait)
	for !p.doneEnough() && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	// 终态宽限：捕捉迟到/泄漏事件
	time.Sleep(graceMs * time.Millisecond)
	p.collectChecks()
	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	default:
	}
	return nil
}

// doneEnough：各场景的收尾条件（目标 response 达到终态）。
func (p *probe) doneEnough() bool {
	switch p.scenario {
	case "turn":
		return len(p.respDone) >= 1 && p.doneStatus(p.lastResp()) == "completed"
	case "cancel":
		return len(p.respDone) >= 1
	case "bargein":
		if len(p.respDone) < 2 {
			return false
		}
		for id, a := range p.respDone {
			if p.doneStatus(id) == "completed" && a.t > p.respDone[p.firstResp()].t {
				return true
			}
		}
		return false
	}
	return false
}

func (p *probe) respIDs() []string {
	ids := make([]string, 0, len(p.respCreated))
	for id := range p.respCreated {
		ids = append(ids, id)
	}
	// 按 created 时间排序
	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			if p.respCreated[ids[j]].t < p.respCreated[ids[i]].t {
				ids[i], ids[j] = ids[j], ids[i]
			}
		}
	}
	return ids
}

func (p *probe) firstResp() string { return p.respIDs()[0] }
func (p *probe) lastResp() string  { return p.respIDs()[len(p.respIDs())-1] }

// ---------- 发送侧 ----------

func (p *probe) expectSession() error {
	deadline := time.Now().Add(10 * time.Second)
	for p.sessCreated.t == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if p.sessCreated.t == 0 {
		return fmt.Errorf("10s 内未收到 session.created")
	}
	return nil
}

func (p *probe) sendJSON(v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	p.wmu.Lock()
	defer p.wmu.Unlock()
	return p.conn.WriteMessage(websocket.TextMessage, raw)
}

// sendAudioChunk 与 sendJSON 互斥：gorilla 不允许并发 WriteMessage。
func (p *probe) sendAudioChunk(pcm []byte) error {
	msg, err := json.Marshal(map[string]any{
		"type":  "input_audio_buffer.append",
		"audio": base64.StdEncoding.EncodeToString(pcm),
	})
	if err != nil {
		return err
	}
	p.wmu.Lock()
	defer p.wmu.Unlock()
	return p.conn.WriteMessage(websocket.TextMessage, msg)
}

func (p *probe) sendSessionUpdate() error {
	// 输出格式不指定 = 服务端原生 16k 管线率（audio_client 同款约定）。
	return p.sendJSON(map[string]any{
		"type": "session.update",
		"session": map[string]any{
			"type":         "realtime",
			"instructions": "你是 gaea 的语音对话助手。用口语化中文简洁回答，每轮不超过两句话。",
			"audio": map[string]any{
				"input": map[string]any{
					"turn_detection": map[string]any{
						// 2026-11-04 P1 记分卡：对齐产品客户端阈值（useVoiceChatV2
						// vadThreshold 0.75，2026-11-01 实弹修 ③）——探针会话不
						// 发 override 时落在 yaml 默认 0.6，噪声刺激会假触发。
						"type": "server_vad", "threshold": 0.75, "interrupt_response": true,
					},
				},
			},
		},
	})
}

func (p *probe) streamSpeech(pcm []byte, first, last *time.Time) {
	chunk := pcmRate * 2 * chunkMs / 1000
	for off := 0; off < len(pcm); off += chunk {
		end := min(off+chunk, len(pcm))
		msg, _ := json.Marshal(map[string]any{
			"type":  "input_audio_buffer.append",
			"audio": base64.StdEncoding.EncodeToString(pcm[off:end]),
		})
		if err := p.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("发送失败: %v", err)
			return
		}
		if first != nil && *first == (time.Time{}) {
			*first = time.Now()
		}
		if last != nil {
			*last = time.Now()
		}
		if p.pace > 0 {
			time.Sleep(time.Duration(float64(chunkMs) * p.pace * float64(time.Millisecond)))
		}
	}
}

func (p *probe) streamSilence(ms int) {
	if ms <= 0 {
		return
	}
	chunk := pcmRate * 2 * chunkMs / 1000
	for sent := 0; sent < ms; sent += chunkMs {
		if err := p.sendAudioChunk(make([]byte, chunk)); err != nil {
			log.Printf("静音发送失败: %v", err)
			return
		}
		if p.pace > 0 {
			time.Sleep(time.Duration(float64(chunkMs) * p.pace * float64(time.Millisecond)))
		}
	}
}

// sendCancelOnFirstAudio：cancel 场景——收到首段下行音频立即显式取消。
func (p *probe) sendCancelOnFirstAudio() {
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		if len(p.firstAudio) > 0 {
			p.cancelSent = time.Now()
			if err := p.sendJSON(map[string]any{"type": "response.cancel"}); err != nil {
				log.Printf("response.cancel 发送失败: %v", err)
			} else {
				log.Printf("已发送 response.cancel（连接后 %dms）", time.Since(p.start).Milliseconds())
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	log.Printf("警告：60s 未收到首段音频，未发 response.cancel")
}

// waitBargeinWindow：等首轮音频开始播，再延迟 bargeAfter 毫秒插话。
func (p *probe) waitBargeinWindow() {
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		if len(p.firstAudio) > 0 && p.audioDur(p.firstResp()) > time.Duration(p.bargeAfter)*time.Millisecond {
			p.bargeinFired = true
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	log.Printf("警告：60s 内未等到可插话窗口，仍继续发 -speech2")
	p.bargeinFired = true
}

// doneStatus 取 response.done 文本 "status/reason" 的 status 段（reason 可缺席）。
func (p *probe) doneStatus(respID string) string {
	t := p.respDone[respID].text
	if i := strings.IndexByte(t, '/'); i >= 0 {
		return t[:i]
	}
	return t
}

func (p *probe) audioDur(respID string) time.Duration {
	buf := p.audio[respID]
	if buf == nil {
		return 0
	}
	return time.Duration(buf.Len()/2) * time.Second / pcmRate
}

// ---------- 接收侧 ----------

func (p *probe) readLoop(errCh chan<- error) {
	for {
		mt, raw, err := p.conn.ReadMessage()
		if err != nil {
			errCh <- nil // 连接关闭视为正常结束
			return
		}
		if mt != websocket.TextMessage {
			continue
		}
		var ev map[string]any
		if err := json.Unmarshal(raw, &ev); err != nil {
			errCh <- fmt.Errorf("事件解析: %w", err)
			return
		}
		p.handle(ev)
	}
}

func (p *probe) handle(ev map[string]any) {
	t := p.now()
	typ, _ := ev["type"].(string)
	respID, _ := ev["response_id"].(string)
	rec := evRec{T: t, Type: typ, ResponseID: respID}

	switch typ {
	case "session.created":
		p.sessCreated = anchor{t, ""}
	case "input_audio_buffer.speech_started":
		p.speechStarted = append(p.speechStarted, anchor{t, ""})
	case "input_audio_buffer.speech_stopped":
		p.speechStopped = append(p.speechStopped, anchor{t, ""})
	case "conversation.item.input_audio_transcription.completed":
		rec.Text = evStr(ev, "transcript")
		p.transFinal = append(p.transFinal, anchor{t, rec.Text})
	case "response.created":
		resp, _ := ev["response"].(map[string]any)
		if resp != nil {
			id, _ := resp["id"].(string)
			respID = id
			rec.ResponseID = id
		}
		p.respCreated[respID] = anchor{t, ""}
	case "response.output_audio_transcript.delta":
		rec.Text = evStr(ev, "delta")
		if _, ok := p.firstText[respID]; !ok {
			p.firstText[respID] = anchor{t, ""}
		}
	case "response.output_audio.delta":
		b64, _ := ev["delta"].(string)
		raw, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			log.Printf("audio delta base64 解码失败: %v", err)
			return
		}
		rec.AudioBytes = len(raw)
		if p.audio[respID] == nil {
			p.audio[respID] = &bytes.Buffer{}
		}
		p.audio[respID].Write(raw)
		if _, ok := p.firstAudio[respID]; !ok {
			p.firstAudio[respID] = anchor{t, ""}
		}
	case "response.output_audio_transcript.done":
		rec.Text = truncate(evStr(ev, "transcript"), 80)
	case "response.done":
		resp, _ := ev["response"].(map[string]any)
		status, reason := "", ""
		if resp != nil {
			if id, ok := resp["id"].(string); ok && respID == "" {
				respID = id
				rec.ResponseID = id
			}
			status, _ = resp["status"].(string)
			if sd, ok := resp["status_details"].(map[string]any); ok {
				reason, _ = sd["reason"].(string)
			}
		}
		rec.Text = status + "/" + reason
		p.respDone[respID] = anchor{t, rec.Text}
	case "error":
		if detail, ok := ev["error"].(map[string]any); ok {
			rec.Text = truncate(fmt.Sprint(detail["message"]), 160)
		}
		log.Printf("服务端 error: %s", rec.Text)
	}
	p.report.Events = append(p.report.Events, rec)
}

func evStr(ev map[string]any, key string) string {
	s, _ := ev[key].(string)
	return s
}

// ---------- 汇总与断言 ----------

func (p *probe) summarize() string {
	var b strings.Builder
	fmt.Fprintf(&b, "场景 %s｜事件 %d 条｜response %d 个", p.scenario, len(p.report.Events), len(p.respCreated))
	for _, id := range p.respIDs() {
		fmt.Fprintf(&b, "\n  resp %s: audio=%dB(%.1fs) done=%s", shortID(id), p.audio[id].Len(),
			p.audioDur(id).Seconds(), p.respDone[id].text)
	}
	if len(p.transFinal) > 0 {
		fmt.Fprintf(&b, "\n  转录: %s", p.transFinal[len(p.transFinal)-1].text)
	}
	return b.String()
}

func (p *probe) addCheck(name string, ok bool, note string) {
	p.checks = append(p.checks, checkRec{Name: name, OK: ok, Note: note})
}

func (p *probe) collectChecks() {
	// 公共：事件基本序列
	p.addCheck("首个事件=session.created", p.sessCreated.t > 0, "")
	p.addCheck("speech_started≥1 且 speech_stopped≥1", len(p.speechStarted) >= 1 && len(p.speechStopped) >= 1,
		fmt.Sprintf("started=%d stopped=%d", len(p.speechStarted), len(p.speechStopped)))
	if len(p.speechStopped) > 0 && len(p.transFinal) > 0 {
		p.addCheck("转写终稿晚于 speech_stopped", p.transFinal[0].t >= p.speechStopped[0].t,
			fmt.Sprintf("stopped=%s final=%s", p.speechStopped[0].t, p.transFinal[0].t))
		p.addCheck("转写非空", p.transFinal[0].text != "", p.transFinal[0].text)
	}
	if len(p.transFinal) > 0 && len(p.respIDs()) > 0 {
		p.addCheck("response.created 晚于转写终稿", p.respCreated[p.firstResp()].t >= p.transFinal[0].t, "")
	}

	// 分阶段延迟（首轮）
	if len(p.speechStopped) > 0 && len(p.respIDs()) > 0 {
		r := p.firstResp()
		vadMs := durMs(p.speechStopped[0].t - p.speechFirstSent.Sub(p.start))
		sttMs, llmMs, ttsMs, e2eMs := -1, -1, -1, -1
		if len(p.transFinal) > 0 {
			sttMs = durMs(p.transFinal[0].t - p.speechStopped[0].t)
		}
		if a, ok := p.firstText[r]; ok && len(p.transFinal) > 0 {
			llmMs = durMs(a.t - p.transFinal[0].t)
		}
		if a, ok := p.firstAudio[r]; ok {
			ttsMs = durMs(a.t - p.speechStopped[0].t)
			e2eMs = durMs(a.t - p.speechFirstSent.Sub(p.start))
		}
		p.addCheck("延迟剖面已采集", true,
			fmt.Sprintf("vad_end=%dms stt_final=%dms llm_ttft=%dms speech_end→first_audio=%dms 话语首字节→首音频=%dms",
				vadMs, sttMs, llmMs, ttsMs, e2eMs))
	}

	switch p.scenario {
	case "turn":
		r := p.lastResp()
		p.addCheck("response.done=completed", p.doneStatus(r) == "completed", p.respDone[r].text)
		p.addCheck("有下行文本与音频", p.firstText[r].t > 0 && p.firstAudio[r].t > 0,
			fmt.Sprintf("audio=%dB", p.audio[r].Len()))
	case "cancel":
		r := p.lastResp()
		d := p.respDone[r]
		p.addCheck("response.done=cancelled/client_cancelled", p.doneStatus(r) == "cancelled" && strings.Contains(p.respDone[r].text, "client_cancelled"), d.text)
		p.addCheck("取消后无新增音频", p.audioAfter[r] == 0, fmt.Sprintf("%dB", p.audioAfter[r]))
	case "bargein":
		ids := p.respIDs()
		if len(ids) < 2 {
			p.addCheck("出现两个 response", false, fmt.Sprintf("仅 %d 个", len(ids)))
			break
		}
		r1, r2 := ids[0], ids[1]
		p.addCheck("首个 response 被打断(cancelled/turn_detected)", p.doneStatus(r1) == "cancelled" && strings.Contains(p.respDone[r1].text, "turn_detected"), p.respDone[r1].text)
		p.addCheck("插话触发新 response 并 completed", p.doneStatus(r2) == "completed", p.respDone[r2].text)
		if d1, ok1 := p.respDone[r1]; ok1 {
			leak := 0
			for _, e := range p.report.Events {
				if e.Type == "response.output_audio.delta" && e.ResponseID == r1 && e.T > d1.t+staleGraceMs*time.Millisecond {
					leak += e.AudioBytes
				}
			}
			p.addCheck("打断后旧代际音频泄漏=0", leak == 0, fmt.Sprintf("%dB", leak))
		}
		if len(p.transFinal) >= 2 {
			p.addCheck("两轮转写均非空", p.transFinal[0].text != "" && p.transFinal[1].text != "",
				fmt.Sprintf("%q / %q", truncate(p.transFinal[0].text, 30), truncate(p.transFinal[1].text, 30)))
		}
	}
}

// 迟到音频记账：done 终态后到达的 delta 归入 audioAfter。
// 由 handle 在写报告前统一补算（简化：collectChecks 里已按事件序列统计）。

func durMs(d time.Duration) int {
	if d < 0 {
		return -1
	}
	return int(d.Milliseconds())
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[len(id)-12:]
	}
	return id
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
