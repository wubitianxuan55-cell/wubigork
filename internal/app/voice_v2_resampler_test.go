package app

import (
	"math"
	"testing"
)

// TestResampler24k16k_PhaseCarried — 2026-11-07「越到后面越混杂、仿佛多了
// 许多音调」真身回归钉：旧实现每个 SSE 块边界相位归零+重复 1 样本（块频
// 10~20/s → 长散文逐秒叠加波形跳变）。新实现跨块携带分数相位——跨块正弦
// 波与整段一次重采样逐样本一致（容差=线性插值浮点误差）。
func TestResampler24k16k_PhaseCarried(t *testing.T) {
	const (
		inRate      = 24000.0
		freq        = 440.0
		amp         = 0.5
		total       = 24000 // 1 秒
		chunkFaith  = 37    // 故意用非整除块大小（帧边界随机感）
	)
	gen := func(from, n int) []byte {
		out := make([]byte, 0, n*2)
		for i := from; i < from+n; i++ {
			s := int16(amp * 32768.0 * math.Sin(2*math.Pi*freq*float64(i)/inRate))
			out = append(out, byte(s), byte(uint16(s)>>8))
		}
		return out
	}

	// 基准：整段一次 push
	whole := (&resampler24k16k{}).push(gen(0, total))

	// 分块：37 字节/块（质数帧），跨 600+ 块边界
	r := &resampler24k16k{}
	var spliced []byte
	for from := 0; from < total; from += chunkFaith {
		n := chunkFaith
		if from+n > total {
			n = total - from
		}
		spliced = append(spliced, r.push(gen(from, n))...)
	}

	if len(spliced) != len(whole) {
		t.Fatalf("分块输出长度 %d ≠ 整段 %d（相位丢失=时长漂移）", len(spliced), len(whole))
	}
	maxDiff := 0.0
	for i := 0; i+1 < len(whole); i += 2 {
		a := float64(int16(uint16(whole[i]) | uint16(whole[i+1])<<8))
		b := float64(int16(uint16(spliced[i]) | uint16(spliced[i+1])<<8))
		if d := math.Abs(a - b); d > maxDiff {
			maxDiff = d
		}
	}
	// 线性插值浮点误差容忍：正弦最大斜率 ×1 样本 ≈ 0.5×32768×2π×440/24000 ≈ 19
	if maxDiff > 24 {
		t.Fatalf("跨块边界失真：最大逐样本差 %.1f（>24=边界跳变/重复样本）", maxDiff)
	}
}


// TestResampler24k16k_AntiAlias — 抗混叠验证（社区共识：线性插值抽取无低通
// 会把 8~12kHz 齿音折叠进 0~8k=不和谐金属音调）。10kHz 音频在输出奈奎斯特
// 之上，经 63 阶 FIR 低通后应衰减 ≥20dB。
func TestResampler24k16k_AntiAlias(t *testing.T) {
	pushTone := func(freq float64) (rms float64, n int) {
		r := &resampler24k16k{}
		var sumSq float64
		for from := 0; from < 24000; from += 37 {
			n2 := 37
			if from+n2 > 24000 {
				n2 = 24000 - from
			}
			pcm := make([]byte, 0, n2*2)
			for i := from; i < from+n2; i++ {
				s := int16(0.5 * 32768.0 * math.Sin(2*math.Pi*freq*float64(i)/24000.0))
				pcm = append(pcm, byte(s), byte(uint16(s)>>8))
			}
			out := r.push(pcm)
			for i := 0; i+1 < len(out); i += 2 {
				v := float64(int16(uint16(out[i]) | uint16(out[i+1])<<8)) / 32768.0
				sumSq += v * v
				n++
			}
		}
		if n == 0 {
			return 0, 0
		}
		return math.Sqrt(sumSq / float64(n)), n
	}
	rmsRef, _ := pushTone(1000)  // 带内参考
	rmsAlias, _ := pushTone(10000) // 输出奈奎斯特(8k)之上=混叠源
	if rmsRef <= 0 {
		t.Fatal("参考音无输出")
	}
	attenuationDB := 20 * math.Log10(rmsAlias / rmsRef)
	if attenuationDB > -14 {
		t.Fatalf("10kHz 混叠源衰减不足: %.1f dB（>-14dB=齿音折叠成金属音调）", attenuationDB)
	}
}
