// Package realtime — Resample16kTo24k 纯函数测试（S2，全离线）。
//
// 用例覆盖设计文档 §2：空帧 / 单样本 / 非整帧（奇数字节）/ 正弦保真 /
// 长度比 3:2 校验 / 已知插值精确值。
package realtime

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

func pcm16Bytes(samples []int16) []byte {
	out := make([]byte, len(samples)*2)
	for i, s := range samples {
		binary.LittleEndian.PutUint16(out[i*2:], uint16(s))
	}
	return out
}

func pcm16Samples(pcm []byte) []int16 {
	n := len(pcm) / 2
	out := make([]int16, n)
	for i := 0; i < n; i++ {
		out[i] = int16(uint16(pcm[i*2]) | uint16(pcm[i*2+1])<<8)
	}
	return out
}

// TestResample_Empty 空输入 → nil（不崩、零分配语义）。
func TestResample_Empty(t *testing.T) {
	if got := Resample16kTo24k(nil); len(got) != 0 {
		t.Errorf("Resample16kTo24k(nil) = %v, want 空", got)
	}
	if got := Resample16kTo24k([]byte{}); len(got) != 0 {
		t.Errorf("Resample16kTo24k([]) = %v, want 空", got)
	}
}

// TestResample_SingleSample 单样本 → 原样返回（位置 0 插值即自身）。
func TestResample_SingleSample(t *testing.T) {
	got := pcm16Samples(Resample16kTo24k(pcm16Bytes([]int16{-12345})))
	if len(got) != 1 || got[0] != -12345 {
		t.Errorf("单样本重采样 = %v, want [-12345]", got)
	}
}

// TestResample_ExactKnownValues 已知精确插值：[0,300] → [0,200,300]
//（中间样本 = 0×1/3 + 300×2/3 = 200；末样本保持）。
func TestResample_ExactKnownValues(t *testing.T) {
	got := pcm16Samples(Resample16kTo24k(pcm16Bytes([]int16{0, 300})))
	want := []int16{0, 200, 300}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d（2 样本 ×3/2）", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("out[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

// TestResample_DCPassthrough 直流信号（恒定值）重采样后逐样本不变。
func TestResample_DCPassthrough(t *testing.T) {
	const v int16 = -777
	got := pcm16Samples(Resample16kTo24k(pcm16Bytes([]int16{v, v, v, v, v, v})))
	if len(got) != 9 {
		t.Fatalf("len = %d, want 9", len(got))
	}
	for i, s := range got {
		if s != v {
			t.Errorf("直流被破坏: out[%d] = %d, want %d", i, s, v)
		}
	}
}

// TestResample_LengthRatio 长度比 3:2：200ms 帧（3200 样本/6400 字节）→
// 4800 样本/9600 字节；奇数字节输入按完整样本数处理（截断尾字节）。
func TestResample_LengthRatio(t *testing.T) {
	chunk := make([]byte, 6400) // voice 管线单帧
	got := Resample16kTo24k(chunk)
	if len(got) != 9600 {
		t.Errorf("200ms 帧重采样 = %d 字节, want 9600（×3/2）", len(got))
	}

	odd := []byte{0x01, 0x02, 0x03, 0x04, 0x05} // 2 完整样本 + 1 尾字节
	if got := Resample16kTo24k(odd); len(got) != 6 {
		t.Errorf("奇数字节输入（2 完整样本）= %d 字节, want 6", len(got))
	}
	if got := Resample16kTo24k([]byte{0x01}); len(got) != 0 {
		t.Errorf("孤字节输入应得空, got %d 字节", len(got))
	}
}

// TestResample_OddSampleCount 非整帧：3 样本 → 4 样本（floor(3×3/2)=4），
// 末位置恰落在最后样本上（精确值，无需外推）。
func TestResample_OddSampleCount(t *testing.T) {
	in := []int16{300, 600, 900}
	got := pcm16Samples(Resample16kTo24k(pcm16Bytes(in)))
	if len(got) != 4 {
		t.Fatalf("len = %d, want 4", len(got))
	}
	// j=3 → pos=3 样本位置 → 精确取 in[2]
	if got[3] != 900 {
		t.Errorf("末样本 = %d, want 900（精确保持）", got[3])
	}
}

// TestResample_SineFidelity 正弦保真：440Hz/振幅 8000 的 16k 正弦重采样后，
// 与直接生成的 24k 同频正弦逐样本比对。内部样本（除末样本外）最大误差
// < 振幅 1%（线性插值对 440Hz@16k 的理论误差 ~0.1%）；末输出样本位置落在
// 最后输入样本之后 2/3 样本处，按末样本保持约定单独校验（不外推）。
func TestResample_SineFidelity(t *testing.T) {
	const (
		freq       = 440.0
		amplitude  = 8000.0
		inSamples  = 1600 // 100ms @16k
		outSamples = 2400 // 100ms @24k
	)
	in := make([]int16, inSamples)
	for i := range in {
		in[i] = int16(math.Round(amplitude * math.Sin(2*math.Pi*freq*float64(i)/resampleInRate)))
	}
	got := pcm16Samples(Resample16kTo24k(pcm16Bytes(in)))
	if len(got) != outSamples {
		t.Fatalf("重采样长度 = %d 样本, want %d", len(got), outSamples)
	}

	tolerance := amplitude * 0.01 // 1% 振幅
	maxErr := 0.0
	for j := 0; j < outSamples-1; j++ { // 末样本为保持边界，单独校验
		want := amplitude * math.Sin(2*math.Pi*freq*float64(j)/resampleOutRate)
		if err := math.Abs(float64(got[j]) - want); err > maxErr {
			maxErr = err
		}
	}
	if maxErr >= tolerance {
		t.Errorf("正弦保真超差: maxErr=%.1f, tolerance=%.1f", maxErr, tolerance)
	}
	t.Logf("interior maxErr = %.2f (amplitude %.0f, %.4f%%)", maxErr, amplitude, maxErr/amplitude*100)

	// 末样本保持：out[末] 应等于 in[末]（不外推约定）
	if got[outSamples-1] != in[inSamples-1] {
		t.Errorf("末样本 = %d, want 末样本保持 %d", got[outSamples-1], in[inSamples-1])
	}
}

// TestResample_InputNotModified 纯函数纪律：入参不被修改。
func TestResample_InputNotModified(t *testing.T) {
	in := pcm16Bytes([]int16{100, -200, 300})
	orig := bytes.Clone(in)
	_ = Resample16kTo24k(in)
	if !bytes.Equal(in, orig) {
		t.Error("入参被修改（违反纯函数纪律）")
	}
}
