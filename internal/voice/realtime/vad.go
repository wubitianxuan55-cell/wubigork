// vad.go — silero 逐块状态机移植（sidecar VAD/vad_iterator.py，规格 §1）。
package realtime

// VadScorer 单块语音概率打分（silero v5 直跑；LSTM 态在实现内）。
type VadScorer interface {
	// Prob 输入恰好 512 样本 float32（/32768 归一），返回语音概率。
	Prob(x []float32) (float32, error)
	// Reset 重置推理态（500ms 空闲后调用；仅模型态，不影响本状态机）。
	Reset()
}

// vadIterator 纯状态机。非并发安全（单音频 goroutine 持有）。
type vadIterator struct {
	scorer         VadScorer
	threshold      float64
	sampleRate     int
	minSilence     int
	speechPad      int
	modelResetIdle int

	triggered        bool
	tempEnd          int
	currentSample    int
	lastSpeechEnd    int
	hasSpeechEnd     bool
	buffer           [][]float32
	prefixBuffer     [][]float32
	preSpeech        [][]float32
	preSpeechSamples int
	activeSpeech     int
	lastUtterActive  int
	modelIdle        int
}

func newVADIterator(scorer VadScorer, threshold float64, sampleRate int, minSilenceMs int, speechPadMs int) *vadIterator {
	return &vadIterator{
		scorer:         scorer,
		threshold:      threshold,
		sampleRate:     sampleRate,
		minSilence:     sampleRate * minSilenceMs / 1000,
		speechPad:      sampleRate * speechPadMs / 1000,
		modelResetIdle: sampleRate * 500 / 1000,
		lastSpeechEnd:  -1,
	}
}

func (v *vadIterator) windowSize() int { return 512 }

// resetStates 全量重置（会话结束）。
func (v *vadIterator) resetStates() {
	if v.scorer != nil {
		v.scorer.Reset()
	}
	v.triggered = false
	v.tempEnd = 0
	v.currentSample = 0
	v.lastSpeechEnd = -1
	v.hasSpeechEnd = false
	v.buffer = nil
	v.prefixBuffer = nil
	v.preSpeech = nil
	v.preSpeechSamples = 0
	v.activeSpeech = 0
	v.lastUtterActive = 0
	v.modelIdle = 0
}

// process 一块 512 样本：nil=无段；非 nil=utterance（prefix+buffer 已拼接拷贝）。
func (v *vadIterator) process(x []float32) [][]float32 {
	if v.scorer == nil {
		return nil // 无打分器（测试桩态）：VAD 不迭代
	}
	prob, err := v.scorer.Prob(x)
	if err != nil {
		// 推理失败按静音处理（sidecar 无此分支：torch 不会失败；Go 侧诚实降级）
		prob = 0
	}
	v.currentSample += len(x)
	w := len(x)

	switch {
	case prob >= float32(v.threshold) && !v.triggered:
		// ONSET
		v.modelIdle = 0
		v.triggered = true
		v.prefixBuffer = v.preSpeech
		v.preSpeech = nil
		v.preSpeechSamples = 0
		v.buffer = [][]float32{cloneF32(x)}
		v.activeSpeech = w
		v.lastUtterActive = 0
		return nil

	case !v.triggered:
		// IDLE：滚窗预语音 pad
		v.rememberPreSpeech(x)
		if float64(prob) < v.threshold-0.15 {
			v.modelIdle += w
			if v.modelIdle >= v.modelResetIdle {
				v.scorer.Reset()
				v.modelIdle = 0
			}
		} else {
			v.modelIdle = 0
		}
		return nil

	default:
		// SEGMENT 内
		v.buffer = append(v.buffer, cloneF32(x))
		if float64(prob) >= v.threshold-0.15 {
			v.activeSpeech += w
			if v.tempEnd != 0 {
				v.tempEnd = 0 // 间隙桥接
			}
			return nil
		}
		if float64(prob) < v.threshold-0.15 {
			if v.tempEnd == 0 {
				v.tempEnd = v.currentSample
			}
			if v.currentSample-v.tempEnd < v.minSilence {
				return nil
			}
			// --- 段结束 ---
			v.lastSpeechEnd = v.tempEnd - w
			v.hasSpeechEnd = true
			v.tempEnd = 0
			v.triggered = false
			spoken := v.speechBuffer()
			v.lastUtterActive = v.activeSpeech
			v.activeSpeech = 0
			v.buffer = nil
			v.prefixBuffer = nil
			v.scorer.Reset()
			v.modelIdle = 0
			return spoken
		}
		return nil
	}
}

func (v *vadIterator) rememberPreSpeech(x []float32) {
	v.preSpeech = append(v.preSpeech, cloneF32(x))
	v.preSpeechSamples += len(x)
	for v.preSpeechSamples > v.speechPad && len(v.preSpeech) > 0 {
		first := v.preSpeech[0]
		excess := v.preSpeechSamples - v.speechPad
		if excess >= len(first) {
			v.preSpeech = v.preSpeech[1:]
			v.preSpeechSamples -= len(first)
			continue
		}
		v.preSpeech[0] = cloneF32(first[excess:])
		v.preSpeechSamples -= excess
	}
}

func (v *vadIterator) speechBuffer() [][]float32 {
	if len(v.prefixBuffer) == 0 {
		return v.buffer
	}
	out := make([][]float32, 0, len(v.prefixBuffer)+len(v.buffer))
	out = append(out, v.prefixBuffer...)
	out = append(out, v.buffer...)
	return out
}

func cloneF32(x []float32) []float32 {
	out := make([]float32, len(x))
	copy(out, x)
	return out
}

// catF32 拼接多块为单段。
func catF32(chunks [][]float32) []float32 {
	n := 0
	for _, c := range chunks {
		n += len(c)
	}
	out := make([]float32, 0, n)
	for _, c := range chunks {
		out = append(out, c...)
	}
	return out
}
