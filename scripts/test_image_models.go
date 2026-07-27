// +build ignore

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/wubigork/wubigork/internal/ai"
	"github.com/wubigork/wubigork/internal/config"
)

func main() {
	cfg := config.Load()

	fmt.Println("=== Image Model API Test ===")
	fmt.Printf("Backend: %s  Model: %s\n\n", cfg.ImageBackend, cfg.ImageModel)

	client := ai.NewClient(cfg)
	ctx := context.Background()

	type tc struct {
		label   string
		backend ai.ImageBackend
		btype   string
		model   string
	}

	tests := []tc{
		{"xAI grok-imagine", nil, "xai", "grok-imagine-image-quality"},
	}
	if cfg.ComfyUIURL != "" {
		comfy := ai.NewComfyUIBackend(cfg.ComfyUIURL)
		tests = append(tests,
			tc{"ComfyUI Flux", comfy, "comfyui", "flux"},
			tc{"ComfyUI Z-Image", comfy, "comfyui", "z-image-turbo"},
			tc{"ComfyUI Krea2", comfy, "comfyui", "krea2"},
		)
	}

	pass := 0
	fail := 0
	for _, t := range tests {
		fmt.Printf("TEST %-22s ... ", t.label)
		client.SetImageBackend(t.backend, t.btype)

		req := &ai.ImageGenerationRequest{
			Model:  t.model,
			Prompt: "a red apple on white table, studio lighting",
			N:      1,
			Size:   "512x512",
			Seed:   42,
		}

		start := time.Now()
		resp, err := client.GenerateImage(ctx, req)
		elapsed := time.Since(start).Round(time.Millisecond)

		if err != nil {
			fmt.Printf("FAIL %v: %v\n", elapsed, err)
			fail++
			continue
		}
		if len(resp.Data) == 0 {
			fmt.Printf("FAIL %v: empty\n", elapsed)
			fail++
			continue
		}
		size := len(resp.Data[0].B64JSON)
		if resp.Data[0].URL != "" {
			size = len(resp.Data[0].URL)
		}
		fmt.Printf("PASS %v bytes=%d\n", elapsed, size)
		pass++
	}

	client.SetImageBackend(nil, cfg.ImageBackend)
	fmt.Printf("\nRESULT: %d pass, %d fail\n", pass, fail)
}
