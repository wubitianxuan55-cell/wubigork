//go:build windows && amd64 && cgo

// probprobe_test.go — silero 逐块概率对拍（诊断用，恒跑）。
package native

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSileroProbDump(t *testing.T) {
	p := filepath.Join(`C:\AI\models\voice`, "silero_vad_v5_16k.onnx")
	if _, err := os.Stat(p); err != nil {
		t.Skip("model missing")
	}
	eng, err := New(Config{})
	if err != nil {
		t.Fatalf("eng: %v", err)
	}
	defer eng.Close()
	vad, err := eng.NewSileroVAD(p)
	if err != nil {
		t.Fatalf("vad: %v", err)
	}
	defer vad.Destroy()
	raw, err := os.ReadFile(filepath.Join("testdata", "eot_incomplete_16k.wav"))
	if err != nil {
		t.Fatal(err)
	}
	pcm := raw[44:]
	_ = pcm
	// 与 python 相同：整段 float32 按 512 块直喂
	samples := make([]float32, 0, (len(pcm))/2)
	for i := 0; i+1 < len(pcm); i += 2 {
		s := int16(uint16(pcm[i]) | uint16(pcm[i+1])<<8)
		samples = append(samples, float32(s)/32768.0)
	}
	var probs []string
	for off := 0; off+512 <= len(samples); off += 512 {
		pr, err := vad.Prob(samples[off : off+512])
		if err != nil {
			t.Fatalf("prob: %v", err)
		}
		probs = append(probs, strconvFormatFloat(pr))
	}
	t.Logf("go probs: %v", probs)
}

func strconvFormatFloat(v float32) string {
	return filepathJoin(strconvFmt(v))
}

func filepathJoin(s string) string { return s }

func strconvFmt(v float32) string {
	b := []byte{}
	u := uint32(v * 1000)
	_ = u
	// 简单三位小数
	ip := int(v * 1000)
	d := ip % 1000
	if d < 0 {
		d = -d
	}
	ip /= 1000
	s := ""
	neg := ip < 0
	if ip == 0 && v < 0 {
		neg = true
	}
	ip2 := ip
	if ip2 < 0 {
		ip2 = -ip2
	}
	// 整数部分
	if ip2 == 0 {
		s += "0"
	} else {
		digits := []byte{}
		for ip2 > 0 {
			digits = append(digits, byte('0'+ip2%10))
			ip2 /= 10
		}
		for i := len(digits) - 1; i >= 0; i-- {
			s += string(digits[i])
		}
	}
	if neg {
		s = "-" + s
	}
	s += "."
	if d < 10 {
		s += "00"
	} else if d < 100 {
		s += "0"
	}
	ds := []byte{}
	for d > 0 {
		ds = append(ds, byte('0'+d%10))
		d /= 10
	}
	for i := len(ds) - 1; i >= 0; i-- {
		s += string(ds[i])
	}
	b = append(b, s...)
	return string(b)
}
