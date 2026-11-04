//go:build windows && amd64 && cgo

// engine_windows.go — sherpa-onnx(paraformer STT / silero VAD 冒烟) + onnxruntime_go(smart-turn EOT) 实装。
package native

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	sherpa_onnx "github.com/k2-fsa/sherpa-onnx-go-windows/v2"
	ort "github.com/yalue/onnxruntime_go"
)

// Engine 进程内语音引擎。上层管线串行使用（单会话池），各腿自带互斥兜底。
type Engine struct {
	cfg    Config
	rec    *sherpa_onnx.OfflineRecognizer
	vad    *sherpa_onnx.VoiceActivityDetector
	eot    *ort.AdvancedSession
	eotIn  *ort.Tensor[float32]
	eotOut *ort.Tensor[float32]
	eotMu  sync.Mutex
	sttMu  sync.Mutex
}

// New 按需装配引擎：给出的模型路径为空则跳过该引擎。
func New(cfg Config) (*Engine, error) {
	if cfg.NumThreads <= 0 {
		cfg.NumThreads = 2
	}
	if err := initORT(cfg.ORTRuntimePath); err != nil {
		return nil, fmt.Errorf("onnxruntime 初始化失败: %w", err)
	}
	e := &Engine{cfg: cfg}
	if cfg.SmartTurnModelPath != "" {
		if err := e.initEOT(); err != nil {
			e.Close()
			return nil, err
		}
	}
	if cfg.ParaformerModelPath != "" {
		if err := e.initSTT(); err != nil {
			e.Close()
			return nil, err
		}
	}
	if cfg.SileroModelPath != "" {
		if err := e.initVAD(); err != nil {
			e.Close()
			return nil, err
		}
	}
	return e, nil
}

// Close 释放全部原生资源。
func (e *Engine) Close() {
	if e == nil {
		return
	}
	if e.eot != nil {
		e.eot.Destroy()
		e.eot = nil
	}
	if e.eotIn != nil {
		e.eotIn.Destroy()
		e.eotIn = nil
	}
	if e.eotOut != nil {
		e.eotOut.Destroy()
		e.eotOut = nil
	}
	if e.rec != nil {
		sherpa_onnx.DeleteOfflineRecognizer(e.rec)
		e.rec = nil
	}
	if e.vad != nil {
		sherpa_onnx.DeleteVoiceActivityDetector(e.vad)
		e.vad = nil
	}
}

// OrtDLLPath 解析随 exe 派发的 onnxruntime.dll：exe 同目录优先，缺省走系统搜索序。
func OrtDLLPath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	candidate := filepath.Join(filepath.Dir(exe), "onnxruntime.dll")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}

var (
	ortInitOnce sync.Once
	ortInitErr  error
)

func initORT(path string) error {
	ortInitOnce.Do(func() {
		if path == "" {
			path = OrtDLLPath()
		}
		if path != "" {
			ort.SetSharedLibraryPath(path)
		}
		ortInitErr = ort.InitializeEnvironment()
	})
	return ortInitErr
}

// --- EOT（smart-turn v3.2）---

func (e *Engine) initEOT() error {
	in, err := ort.NewTensor(ort.NewShape(1, whisperNumMels, whisperNumFrames),
		make([]float32, whisperNumMels*whisperNumFrames))
	if err != nil {
		return fmt.Errorf("EOT 输入张量创建失败: %w", err)
	}
	out, err := ort.NewTensor(ort.NewShape(1, 1), make([]float32, 1))
	if err != nil {
		in.Destroy()
		return fmt.Errorf("EOT 输出张量创建失败: %w", err)
	}
	opts, err := ort.NewSessionOptions()
	if err != nil {
		in.Destroy()
		out.Destroy()
		return fmt.Errorf("EOT 会话选项创建失败: %w", err)
	}
	defer opts.Destroy()
	sess, err := ort.NewAdvancedSession(e.cfg.SmartTurnModelPath,
		[]string{"input_features"}, []string{"logits"},
		[]ort.Value{in}, []ort.Value{out}, opts)
	if err != nil {
		in.Destroy()
		out.Destroy()
		return fmt.Errorf("EOT 会话创建失败: %w", err)
	}
	e.eot, e.eotIn, e.eotOut = sess, in, out
	// warmup：1 秒静音过一遍图（对齐 sidecar SmartTurnAnalyzer warmup）
	if _, err := e.EndOfTurnProbability(make([]float32, whisperSampleRate), whisperSampleRate); err != nil {
		return fmt.Errorf("EOT warmup 失败: %w", err)
	}
	return nil
}

// EndOfTurnProbability 返回 utterance 的 smart-turn 完结概率（p>0.5 视为说完）。
// 输入必须 16 kHz；内部含 last-8s 取窗/补零与 log-mel 全前端。
func (e *Engine) EndOfTurnProbability(pcm []float32, sampleRate int) (float32, error) {
	if e.eot == nil {
		return 0, fmt.Errorf("EOT 引擎未装配")
	}
	if sampleRate != whisperSampleRate {
		return 0, fmt.Errorf("EOT 需要 %d Hz 输入，得到 %d Hz", whisperSampleRate, sampleRate)
	}
	feats, err := WhisperLogMel(pcm)
	if err != nil {
		return 0, err
	}
	e.eotMu.Lock()
	defer e.eotMu.Unlock()
	copy(e.eotIn.GetData(), feats)
	if err := e.eot.Run(); err != nil {
		return 0, fmt.Errorf("EOT 推理失败: %w", err)
	}
	return e.eotOut.GetData()[0], nil
}

// EndOfTurnTiming 带推理耗时的 EOT（供延迟记分卡）。
func (e *Engine) EndOfTurnTiming(pcm []float32, sampleRate int) (float32, time.Duration, error) {
	start := time.Now()
	p, err := e.EndOfTurnProbability(pcm, sampleRate)
	return p, time.Since(start), err
}

// --- STT（paraformer int8 全句转写）---

func (e *Engine) initSTT() error {
	cfg := &sherpa_onnx.OfflineRecognizerConfig{
		DecodingMethod: "greedy_search",
		ModelConfig: sherpa_onnx.OfflineModelConfig{
			Paraformer: sherpa_onnx.OfflineParaformerModelConfig{Model: e.cfg.ParaformerModelPath},
			Tokens:     e.cfg.ParaformerTokensPath,
			NumThreads: e.cfg.NumThreads,
			Provider:   "cpu",
			Debug:      0,
		},
	}
	rec := sherpa_onnx.NewOfflineRecognizer(cfg)
	if rec == nil {
		return fmt.Errorf("paraformer 加载失败（model=%s tokens=%s）",
			e.cfg.ParaformerModelPath, e.cfg.ParaformerTokensPath)
	}
	e.rec = rec
	// warmup：512 样本静音（对齐 sidecar ParaformerSTTHandler warmup）
	if _, err := e.Transcribe(make([]float32, 512), whisperSampleRate); err != nil {
		return fmt.Errorf("STT warmup 失败: %w", err)
	}
	return nil
}

// Transcribe 全句转写（非流式，整段送入；与 sidecar paraformer-zh 行为等价）。
// 非 16k 输入由 sherpa 内部重采样。
func (e *Engine) Transcribe(pcm []float32, sampleRate int) (string, error) {
	if e.rec == nil {
		return "", fmt.Errorf("STT 引擎未装配")
	}
	if len(pcm) == 0 {
		return "", nil
	}
	e.sttMu.Lock()
	defer e.sttMu.Unlock()
	stream := sherpa_onnx.NewOfflineStream(e.rec)
	defer sherpa_onnx.DeleteOfflineStream(stream)
	stream.AcceptWaveform(sampleRate, pcm)
	e.rec.Decode(stream)
	// 与 sidecar 一致：中文结果去空格
	return strings.ReplaceAll(strings.TrimSpace(stream.GetResult().Text), " ", ""), nil
}

// --- VAD（sherpa silero；仅冒烟/调试，管线轮次机用 ORT 原始概率自建迟滞）---
func (e *Engine) initVAD() error {
	cfg := &sherpa_onnx.VadModelConfig{
		SileroVad: sherpa_onnx.SileroVadModelConfig{
			Model:              e.cfg.SileroModelPath,
			Threshold:          0.5,
			MinSilenceDuration: 0.5,
			MinSpeechDuration:  0.25,
			WindowSize:         512, // silero v5 @16k
			MaxSpeechDuration:  20,
		},
		SampleRate: whisperSampleRate,
		NumThreads: 1,
		Provider:   "cpu",
	}
	vad := sherpa_onnx.NewVoiceActivityDetector(cfg, 30)
	if vad == nil {
		return fmt.Errorf("silero VAD 加载失败（model=%s）", e.cfg.SileroModelPath)
	}
	e.vad = vad
	return nil
}

// VADSegments 冒烟辅助：整段 PCM 切出语音段（每段返回原始样本）。
func (e *Engine) VADSegments(pcm []float32) [][]float32 {
	if e.vad == nil {
		return nil
	}
	const chunk = 512
	for len(pcm) > 0 {
		n := chunk
		if n > len(pcm) {
			n = len(pcm)
		}
		e.vad.AcceptWaveform(pcm[:n])
		pcm = pcm[n:]
	}
	e.vad.Flush()
	var out [][]float32
	for !e.vad.IsEmpty() {
		seg := e.vad.Front()
		if seg != nil {
			out = append(out, seg.Samples)
		}
		e.vad.Pop()
	}
	return out
}

// --- SileroVAD（silero v5 直跑，LSTM 态内持；管线轮次机用） ---

// SileroVAD silero v5 ONNX 会话（512 样本@16k；state [2,1,128]）。
type SileroVAD struct {
	sess   *ort.AdvancedSession
	in     *ort.Tensor[float32]
	state  *ort.Tensor[float32]
	out    *ort.Tensor[float32]
	stateN *ort.Tensor[float32]
	mu     sync.Mutex
}

// NewSileroVAD 装配 silero v5。
func (e *Engine) NewSileroVAD(modelPath string) (*SileroVAD, error) {
	if err := initORT(e.cfg.ORTRuntimePath); err != nil {
		return nil, fmt.Errorf("onnxruntime 初始化失败: %w", err)
	}
	in, err := ort.NewTensor(ort.NewShape(1, 512), make([]float32, 512))
	if err != nil {
		return nil, fmt.Errorf("VAD 输入张量创建失败: %w", err)
	}
	state, err := ort.NewTensor(ort.NewShape(2, 1, 128), make([]float32, 256))
	if err != nil {
		in.Destroy()
		return nil, fmt.Errorf("VAD 状态张量创建失败: %w", err)
	}
	out, err := ort.NewTensor(ort.NewShape(1, 1), make([]float32, 1))
	if err != nil {
		in.Destroy()
		state.Destroy()
		return nil, fmt.Errorf("VAD 输出张量创建失败: %w", err)
	}
	stateN, err := ort.NewTensor(ort.NewShape(2, 1, 128), make([]float32, 256))
	if err != nil {
		in.Destroy()
		state.Destroy()
		out.Destroy()
		return nil, fmt.Errorf("VAD 状态输出张量创建失败: %w", err)
	}
	opts, err := ort.NewSessionOptions()
	if err != nil {
		in.Destroy()
		state.Destroy()
		out.Destroy()
		stateN.Destroy()
		return nil, fmt.Errorf("VAD 会话选项创建失败: %w", err)
	}
	defer opts.Destroy()
	// 模型须为 sr 常量化版（silero_vad_v5_16k.onnx——yalue 不支持 0 维标量输入，
	// sr=16000 已烘进图，.tmp/voice-native-gen/fix_silero_sr.py 生成并等价验证）。
	sess, err := ort.NewAdvancedSession(modelPath,
		[]string{"input", "state"}, []string{"output", "stateN"},
		[]ort.Value{in, state}, []ort.Value{out, stateN}, opts)
	if err != nil {
		in.Destroy()
		state.Destroy()
		out.Destroy()
		stateN.Destroy()
		return nil, fmt.Errorf("VAD 会话创建失败: %w", err)
	}
	return &SileroVAD{sess: sess, in: in, state: state, out: out, stateN: stateN}, nil
}

// Prob 一块 512 样本的语音概率（状态推进）。
func (s *SileroVAD) Prob(x []float32) (float32, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	copy(s.in.GetData(), x)
	if err := s.sess.Run(); err != nil {
		return 0, fmt.Errorf("VAD 推理失败: %w", err)
	}
	// LSTM 态推进：stateN → state（下一块的输入态）
	copy(s.state.GetData(), s.stateN.GetData())
	return s.out.GetData()[0], nil
}

// Reset LSTM 态清零。
func (s *SileroVAD) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := s.state.GetData()
	for i := range d {
		d[i] = 0
	}
}

// Destroy 释放。
func (s *SileroVAD) Destroy() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sess != nil {
		s.sess.Destroy()
		s.sess = nil
	}
	s.in.Destroy()
	s.state.Destroy()
	s.out.Destroy()
	s.stateN.Destroy()
}
