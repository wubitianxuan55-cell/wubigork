package voice

import "encoding/binary"

// wavHeader builds the 44-byte RIFF/WAVE header for PCM16 / 16kHz / mono
// audio, matching the format the frontend captures and Herdsman ASR expects.
func wavHeader(dataLen int) []byte {
	const (
		sampleRate    = SampleRate
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

// wrapPCMAsWAV wraps raw PCM16 / 16kHz / mono bytes into a WAV container.
// The frontend and VAD buffer work with raw PCM; the non-streaming ASR
// endpoint requires a real WAV file, so the header is added before sending.
func wrapPCMAsWAV(pcm []byte) []byte {
	wav := make([]byte, 0, 44+len(pcm))
	wav = append(wav, wavHeader(len(pcm))...)
	wav = append(wav, pcm...)
	return wav
}
