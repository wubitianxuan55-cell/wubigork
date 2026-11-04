// logmel.go — Whisper log-mel 特征提取的 Go 移植（smart-turn v3.2 的输入前端）。
//
// 数学逐字对齐生产路径（venv speech-to-speech sidecar 的实跑分支）：
//
//	transformers 5.x WhisperFeatureExtractor(chunk_length=8) + do_normalize=True + torch STFT 分支
//	（models/whisper/feature_extraction_whisper.py::_torch_extract_fbank_features），
//	外层再加 speech_to_speech/VAD/smart_turn.py::_prepare_audio 的 last-8s/左侧补零。
//
// 参考实现快照：.tmp/research-hf-sts（上游 6feb5278，未改动）；金样 fixture 由
// .tmp/voice-native-gen/gen_fixtures.py 在同一 venv 生成（testdata/*.features.json）。
package native

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"sync"

	"gonum.org/v1/gonum/dsp/fourier"
)

const (
	whisperSampleRate = 16000
	// smart_turn.py：MAX_AUDIO_SECONDS=8 → 128000 samples；不足左侧补零、超长取尾窗。
	whisperWindowSamples = 8 * whisperSampleRate
	// WhisperFeatureExtractor 默认：n_fft=400 / hop=160。
	whisperNFFT      = 400
	whisperHopLength = 160
	whisperNumFrames = 800 // 128000/160 帧，torch.stft 出 801 帧、生产路径丢弃末帧后为 800
	whisperNumMels   = 80
	whisperNumBins   = whisperNFFT/2 + 1 // 201
	whisperCenterPad = whisperNFFT / 2   // torch.stft center=True 反射补 200/200
	logMelFloor      = 1e-10             // torch.clamp(mel_spec, min=1e-10)
	logMelMaxRange   = 8.0               // log_spec.max() - 8.0
	logMelShift      = 4.0               // (log_spec + 4.0) / 4.0
	zNormVarEpsilon  = 1e-7              // (x-mean)/sqrt(var+1e-7)，wav2vec2 式整段归一
)

//go:embed mel_filters.json
var melFiltersJSON []byte

var (
	melOnce    sync.Once
	melFilters [][]float32 // [80][201]
	melErr     error
)

func loadMelFilters() ([][]float32, error) {
	melOnce.Do(func() {
		var parsed struct {
			Shape []int       `json:"shape"`
			Data  [][]float64 `json:"data"`
		}
		if err := json.Unmarshal(melFiltersJSON, &parsed); err != nil {
			melErr = fmt.Errorf("mel_filters.json 解析失败: %w", err)
			return
		}
		if len(parsed.Shape) != 2 || parsed.Shape[0] != whisperNumMels || parsed.Shape[1] != whisperNumBins {
			melErr = fmt.Errorf("mel_filters.json 形状异常: %v", parsed.Shape)
			return
		}
		melFilters = make([][]float32, whisperNumMels)
		for m := range melFilters {
			row := make([]float32, whisperNumBins)
			for b := 0; b < whisperNumBins; b++ {
				row[b] = float32(parsed.Data[m][b])
			}
			melFilters[m] = row
		}
	})
	return melFilters, melErr
}

// WhisperLogMel 把 16 kHz 单声道 PCM 转成 smart-turn 的 [80][800] 输入特征
// （按 mel 主序铺平：特征[m*800+t] = 第 m 个 mel 桶第 t 帧，即 ONNX [1,80,800] 布局）。
func WhisperLogMel(pcm []float32) ([]float32, error) {
	filters, err := loadMelFilters()
	if err != nil {
		return nil, err
	}

	// ① last-8s / 左侧补零（smart_turn._prepare_audio）
	audio := make([]float32, whisperWindowSamples)
	if len(pcm) >= whisperWindowSamples {
		copy(audio, pcm[len(pcm)-whisperWindowSamples:])
	} else {
		copy(audio[whisperWindowSamples-len(pcm):], pcm)
	}

	// ② 整段零均值单位方差归一（do_normalize=True；音频已定长 → attention mask 全 1）
	var sum, sumSq float64
	for _, v := range audio {
		f := float64(v)
		sum += f
		sumSq += f * f
	}
	mean := sum / float64(whisperWindowSamples)
	variance := sumSq/float64(whisperWindowSamples) - mean*mean
	norm := 1.0 / math.Sqrt(variance+zNormVarEpsilon)
	windowed := make([]float64, whisperWindowSamples)
	for i, v := range audio {
		windowed[i] = (float64(v) - mean) * norm
	}

	// ③ center=True 反射补 200/200（torch.stft pad_mode=reflect）
	padded := make([]float64, whisperWindowSamples+2*whisperCenterPad)
	for t := 0; t < whisperWindowSamples; t++ {
		padded[whisperCenterPad+t] = windowed[t]
	}
	for j := 0; j < whisperCenterPad; j++ {
		padded[j] = windowed[whisperCenterPad-j]
		padded[whisperCenterPad+whisperWindowSamples+j] = windowed[whisperWindowSamples-2-j]
	}

	// ④ 周期 Hann 窗（torch.hann_window(400) = np.hanning(401)[:-1]）
	hann := make([]float64, whisperNFFT)
	for n := 0; n < whisperNFFT; n++ {
		hann[n] = 0.5 * (1 - math.Cos(2*math.Pi*float64(n)/float64(whisperNFFT)))
	}

	// ⑤ STFT：800 帧 × 400 点实 FFT → 功率谱 → mel 滤波
	fft := fourier.NewFFT(whisperNFFT)
	frame := make([]float64, whisperNFFT)
	spectrum := make([]complex128, whisperNumBins)
	power := make([]float64, whisperNumBins)
	feats := make([]float32, whisperNumMels*whisperNumFrames)
	for t := 0; t < whisperNumFrames; t++ {
		base := t * whisperHopLength
		for n := 0; n < whisperNFFT; n++ {
			frame[n] = padded[base+n] * hann[n]
		}
		coeffs := fft.Coefficients(spectrum, frame) // 201 bins
		for b, c := range coeffs {
			power[b] = real(c)*real(c) + imag(c)*imag(c)
		}
		for m := 0; m < whisperNumMels; m++ {
			row := filters[m]
			acc := 0.0
			for b := 0; b < whisperNumBins; b++ {
				acc += float64(row[b]) * power[b]
			}
			if acc < logMelFloor {
				acc = logMelFloor
			}
			feats[m*whisperNumFrames+t] = float32(math.Log10(acc))
		}
	}

	// ⑥ log10 后全局 max-8 钳制 + (x+4)/4（对 log10 值直接做，与 torch 路径 clamp→log10→max→shift 同序）
	maxVal := feats[0]
	for _, v := range feats[1:] {
		if v > maxVal {
			maxVal = v
		}
	}
	floor := maxVal - logMelMaxRange
	for i, v := range feats {
		if v < floor {
			v = floor
		}
		feats[i] = float32((float64(v) + logMelShift) / logMelShift)
	}
	return feats, nil
}
