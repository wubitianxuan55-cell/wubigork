package voice

import "encoding/binary"

// RealtimeSampleRate OpenAI Realtime pcm16 音频采样率（24kHz）：输入侧 16k
// 采集经 realtime.Resample16kTo24k 重采样，输出侧 24k 原样打包 WAV。
const RealtimeSampleRate = 24000

// wavHeaderAt builds the 44-byte RIFF/WAVE header for PCM16 / sampleRate / mono
// audio, matching the format the frontend captures and Herdsman ASR expects.
func wavHeaderAt(dataLen, sampleRate int) []byte {
	const (
		channels      = 1
		bitsPerSample = 16
	)
	byteRate := sampleRate * channels * bitsPerSample / 8
	blockAlign := channels * bitsPerSample / 8

	h := make([]byte, 44)
	copy(h[0:4], "RIFF")
	binary.LittleEndian.PutUint32(h[4:8], uint32(36+dataLen))
	copy(h[8:12], "WAVE")
	copy(h[12:16], "fmt ")
	binary.LittleEndian.PutUint32(h[16:20], 16) // fmt chunk size
	binary.LittleEndian.PutUint16(h[20:22], 1)  // PCM
	binary.LittleEndian.PutUint16(h[22:24], channels)
	binary.LittleEndian.PutUint32(h[24:28], uint32(sampleRate))
	binary.LittleEndian.PutUint32(h[28:32], uint32(byteRate))
	binary.LittleEndian.PutUint16(h[32:34], uint16(blockAlign))
	binary.LittleEndian.PutUint16(h[34:36], bitsPerSample)
	copy(h[36:40], "data")
	binary.LittleEndian.PutUint32(h[40:44], uint32(dataLen))
	return h
}

// wavHeader builds the 44-byte RIFF/WAVE header for PCM16 / 16kHz / mono
// audio, matching the format the frontend captures and Herdsman ASR expects.
func wavHeader(dataLen int) []byte {
	return wavHeaderAt(dataLen, SampleRate)
}

// wrapPCMAsWAV wraps raw PCM16 / 16kHz / mono bytes into a WAV container.
// The frontend and VAD buffer work with raw PCM; the non-streaming ASR
// endpoint requires a real WAV file, so the header is added before sending.
func wrapPCMAsWAV(pcm []byte) []byte {
	return wrapPCMAsWAVAt(pcm, SampleRate)
}

// wrapPCMAsWAVAt wraps raw PCM16 / sampleRate / mono bytes into a WAV container.
func wrapPCMAsWAVAt(pcm []byte, sampleRate int) []byte {
	wav := make([]byte, 0, 44+len(pcm))
	wav = append(wav, wavHeaderAt(len(pcm), sampleRate)...)
	wav = append(wav, pcm...)
	return wav
}

// wrapPCMAsWAV24k wraps raw PCM16 / 24kHz / mono bytes into a WAV container
// （S2 realtime 输出音频冲洗用：response.done 聚合器冲出的 24k PCM 原样
// 加 24kHz 头，前端现播放环 decodeAudioData 直接吃）。
func wrapPCMAsWAV24k(pcm []byte) []byte {
	return wrapPCMAsWAVAt(pcm, RealtimeSampleRate)
}
