// vadcheck — silero ONNX 概率曲线探针（诊断工具）。
//
// 2026-11-05 真麦根修的决定性对照：同一段音频，torch.hub silero（sidecar 时代）
// 概率 mean≈0.72 弱电平也满格，Go ONNX 版却全 discard——本工具复现 ONNX 侧
// 概率曲线，量化模型判性（torch 侧对照=.tmp/torch_vad_probe.py）。
//
// 运行布局（DLL 搜索序：exe 目录优先，防 System32 1.17.1 劫持）：
//
//	go build -o .tmp/rt.test/vadcheck.exe ./scripts/voice_v2/vadcheck
//	.tmp/rt.test/vadcheck.exe -wav X.wav [-model silero_vad_v5_16k.onnx]
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"math"
	"os"

	"github.com/gaea/gaea/internal/voice/native"
)

func main() {
	wavPath := flag.String("wav", "", "16k PCM16 单声道 wav")
	model := flag.String("model", `C:\AI\gaea-app\models\voice\silero_vad_v5_16k.onnx`, "silero onnx")
	flag.Parse()
	if *wavPath == "" {
		fmt.Println("用法: vadcheck -wav X.wav")
		os.Exit(2)
	}
	data, err := os.ReadFile(*wavPath)
	if err != nil {
		fmt.Println("read wav:", err)
		os.Exit(1)
	}
	// 跳 44 字节标准头（探针语料恒定格式）
	pcm := data[44:]
	n := len(pcm) / 2
	x := make([]float32, n)
	for i := 0; i < n; i++ {
		v := int16(binary.LittleEndian.Uint16(pcm[i*2:]))
		x[i] = float32(v) / 32768.0
	}
	rms := 0.0
	for _, v := range x {
		rms += float64(v) * float64(v)
	}
	rms = math.Sqrt(rms / float64(n))

	eng, err := native.New(native.Config{ORTRuntimePath: native.OrtDLLPath()})
	if err != nil {
		fmt.Println("native.New:", err)
		os.Exit(1)
	}
	defer eng.Close()
	vad, err := eng.NewSileroVAD(*model)
	if err != nil {
		fmt.Println("NewSileroVAD:", err)
		os.Exit(1)
	}
	defer vad.Destroy()

	var probs []float32
	for off := 0; off+512 <= len(x); off += 512 {
		p, err := vad.Prob(x[off : off+512])
		if err != nil {
			fmt.Println("Prob:", err)
			os.Exit(1)
		}
		probs = append(probs, p)
	}
	above := func(t float64) float64 {
		c := 0
		for _, p := range probs {
			if float64(p) >= t {
				c++
			}
		}
		return float64(c) * 512 / 16000 * 1000
	}
	max, sum := 0.0, 0.0
	for _, p := range probs {
		if float64(p) > max {
			max = float64(p)
		}
		sum += float64(p)
	}
	fmt.Printf("model=%s\nwav=%s rms=%.4f chunks=%d\n", *model, *wavPath, rms, len(probs))
	for _, t := range []float64{0.35, 0.5, 0.75} {
		ms := above(t)
		gate := "no"
		if ms >= 384 {
			gate = "YES"
		}
		fmt.Printf("prob>=%.2f: %.0fms (过384门=%s)\n", t, ms, gate)
	}
	fmt.Printf("max=%.3f mean=%.3f\n", max, sum/float64(len(probs)))
	fmt.Print("curve: ")
	for i := 0; i < len(probs); i += 20 {
		fmt.Printf("%.2f ", probs[i])
	}
	fmt.Println()
}
