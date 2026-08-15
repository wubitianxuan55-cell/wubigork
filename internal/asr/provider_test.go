// Package asr — ASR 提供者 Seam 测试（Step 3c）。
//
// 固化 seam 行为：herdsman 自注册、互斥注册 panic、未知 kind fail-closed、
// 提供者可经注册表构造并实现 ASRProvider 接口。
package asr

import (
	"testing"
)

func TestASRProviderRegistry_Kinds(t *testing.T) {
	kinds := ASRProviderKinds()
	if len(kinds) != 1 || kinds[0] != "herdsman" {
		t.Fatalf("kinds = %v, want [herdsman]（当前唯一实现）", kinds)
	}
}

func TestASRProviderRegistry_Construct(t *testing.T) {
	p, err := NewASRProvider("herdsman", ASRConfig{BaseURL: "http://localhost:8080/v1", Model: "whisper-base"})
	if err != nil {
		t.Fatalf("NewASRProvider(herdsman): %v", err)
	}
	if p.Name() != "herdsman" {
		t.Errorf("Name = %q, want herdsman", p.Name())
	}
	h, ok := p.(*HerdsmanASR)
	if !ok {
		t.Fatalf("类型应为 *HerdsmanASR, got %T", p)
	}
	if h.baseURL != "http://localhost:8080/v1" || h.model != "whisper-base" {
		t.Errorf("配置未生效: baseURL=%q model=%q", h.baseURL, h.model)
	}

	// model 空 → 默认 whisper-base
	p2, err := NewASRProvider("herdsman", ASRConfig{BaseURL: "http://x/v1"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if h2 := p2.(*HerdsmanASR); h2.model != "whisper-base" {
		t.Errorf("空 model 应回退 whisper-base, got %q", h2.model)
	}
}

func TestASRProviderRegistry_UnknownKindFailsClosed(t *testing.T) {
	if _, err := NewASRProvider("no-such-asr", ASRConfig{}); err == nil {
		t.Fatal("未知 kind 应报错（fail-closed）")
	}
}

func TestASRProviderRegistry_DuplicatePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("重复注册应 panic（互斥注册纪律）")
		}
	}()
	RegisterASRProvider("herdsman", func(cfg ASRConfig) (ASRProvider, error) { return nil, nil }) // 已注册 → panic
}

// TestHerdsmanASR_ImplementsInterface 编译期断言：HerdsmanASR 满足 ASRProvider。
func TestHerdsmanASR_ImplementsInterface(t *testing.T) {
	var _ ASRProvider = (*HerdsmanASR)(nil)
}
