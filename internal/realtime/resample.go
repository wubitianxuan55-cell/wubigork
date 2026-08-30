// resample.go — 16k→24k 线性插值重采样纯函数（S2）。
//
// OpenAI Realtime pcm16 = 24kHz，而 gaea voice 管线全程 16kHz 采集
// （voice.SampleRate）——16k 按 24k 解读 = 花栗鼠音 + 转写崩坏。本函数在
// voice 侧 SendAudio 前调用（放 realtime 包内避免 import 循环：realtime
// 不依赖 voice，方向安全）。
package realtime

import "encoding/binary"

// 重采样比（16k → 24k = 3:2）。
const (
	resampleInRate  = 16000
	resampleOutRate = 24000
)

// Resample16kTo24k 将 PCM16 LE mono 音频从 16kHz 线性插值重采样到 24kHz。
//
// 纯函数：不修改入参、不依赖运行时状态、无副作用。
//   - 输出样本数 = floor(输入样本数 × 3/2)；偶数输入（正常帧）精确 ×1.5；
//   - 单样本输入原样返回（位置 0 插值即自身）；
//   - 落在末尾样本之后的位置按末样本保持（不外推）；
//   - 奇数字节输入按完整样本数处理（截断尾字节）；
//   - 空输入返回 nil。
func Resample16kTo24k(pcm []byte) []byte {
	n := len(pcm) / 2
	if n == 0 {
		return nil
	}
	outN := n * resampleOutRate / resampleInRate // ×3/2
	out := make([]byte, outN*2)
	for j := 0; j < outN; j++ {
		// 源位置 pos = j × inRate/outRate = j × 2/3；rem ∈ {0,1,2} 为
		// 1/3 粒度的插值权重（整数运算，无浮点误差）。
		pos := int64(j) * resampleInRate
		i0 := int(pos / resampleOutRate)
		rem := int(pos % resampleOutRate)

		s0 := pcm16At(pcm, i0)
		s1 := s0
		if i0+1 < n {
			s1 = pcm16At(pcm, i0+1)
		}
		// v = s0×(1-rem/3) + s1×(rem/3)，定点计算（误差 ≤ 1 LSB，截断向零）。
		v := (int32(s0)*int32(resampleOutRate-rem) + int32(s1)*int32(rem)) / resampleOutRate
		binary.LittleEndian.PutUint16(out[j*2:], uint16(v))
	}
	return out
}

// pcm16At 读取第 i 个 PCM16 LE 样本（调用方保证 i 不越界完整样本区）。
func pcm16At(pcm []byte, i int) int16 {
	return int16(uint16(pcm[i*2]) | uint16(pcm[i*2+1])<<8)
}
