//go:build windows && amd64

// logmel_test.go — 金样对拍：Go log-mel vs venv transformers/torch 生产路径特征。
// 金样由 .tmp/voice-native-gen/gen_fixtures.py 生成（breeze 合成 utterance →
// smart_turn._prepare_audio → WhisperFeatureExtractor(chunk_length=8) → [1,80,800]）。
package native

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func loadGoldenFeatures(t *testing.T, name string) [][]float32 {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name+".features.json"))
	if err != nil {
		t.Fatalf("金样缺失（gen_fixtures.py 生成）: %v", err)
	}
	var feats [][]float32
	if err := json.Unmarshal(raw, &feats); err != nil {
		t.Fatalf("金样解析失败: %v", err)
	}
	if len(feats) != whisperNumMels || len(feats[0]) != whisperNumFrames {
		t.Fatalf("金样形状异常: %dx%d", len(feats), len(feats[0]))
	}
	return feats
}

func loadGoldenWav(t *testing.T, name string) []float32 {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name+"_16k.wav"))
	if err != nil {
		t.Fatalf("测试 wav 缺失: %v", err)
	}
	// 44 字节标准 WAV 头 + PCM16
	const headerLen = 44
	if len(raw) <= headerLen || len(raw)%2 != 0 {
		t.Fatalf("wav 形态异常: %d bytes", len(raw))
	}
	pcm := make([]float32, 0, (len(raw)-headerLen)/2)
	for i := headerLen; i+1 < len(raw); i += 2 {
		s := int16(uint16(raw[i]) | uint16(raw[i+1])<<8)
		pcm = append(pcm, float32(s)/32768.0)
	}
	return pcm
}

func TestWhisperLogMelGolden(t *testing.T) {
	for _, name := range []string{"eot_complete", "eot_incomplete"} {
		t.Run(name, func(t *testing.T) {
			golden := loadGoldenFeatures(t, name)
			pcm := loadGoldenWav(t, name)
			feats, err := WhisperLogMel(pcm)
			if err != nil {
				t.Fatalf("WhisperLogMel: %v", err)
			}
			if len(feats) != whisperNumMels*whisperNumFrames {
				t.Fatalf("特征长度 %d != %d", len(feats), whisperNumMels*whisperNumFrames)
			}
			// 归一化后的特征量级 ±2；int8 模型鲁棒，对拍容差 2e-3
			const tol = 2e-3
			maxDiff, maxIdx := 0.0, 0
			for m := 0; m < whisperNumMels; m++ {
				for fr := 0; fr < whisperNumFrames; fr++ {
					d := math.Abs(float64(feats[m*whisperNumFrames+fr] - golden[m][fr]))
					if d > maxDiff {
						maxDiff, maxIdx = d, m*whisperNumFrames+fr
					}
				}
			}
			if maxDiff > tol {
				t.Fatalf("max|diff|=%v (idx=%d, tol=%v) — DSP 移植偏离金样", maxDiff, maxIdx, tol)
			}
			t.Logf("max|diff|=%v (tol=%v)", maxDiff, tol)
		})
	}
}
