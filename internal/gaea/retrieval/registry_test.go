package retrieval

import (
	"strings"
	"testing"
)

// TestEmbedderRegistry_AllKinds "openai" kind 经注册表构建为 *Embedder。
func TestEmbedderRegistry_AllKinds(t *testing.T) {
	kinds := EmbedderKinds()
	if len(kinds) != 1 || kinds[0] != EmbedderKindOpenAI {
		t.Fatalf("EmbedderKinds = %v, want [openai]", kinds)
	}
	e, err := NewEmbedderByKind(EmbedderKindOpenAI, EmbedderConfig{BaseURL: "http://localhost:8080", Model: "bge-m3"})
	if err != nil {
		t.Fatalf("NewEmbedderByKind(openai): %v", err)
	}
	if _, ok := e.(*Embedder); !ok {
		t.Fatalf("kind=openai 应返回 *Embedder, got %T", e)
	}
}

// TestEmbedderRegistry_ConfigRouting 同形配置 + 不同 kind 得到不同实现：
// 切换后端只改 kind，消费方（EmbeddingProvider 接口）零改动。
func TestEmbedderRegistry_ConfigRouting(t *testing.T) {
	var consumer func(kind string) (EmbeddingProvider, error)
	consumer = func(kind string) (EmbeddingProvider, error) {
		return NewEmbedderByKind(kind, EmbedderConfig{BaseURL: "http://127.0.0.1:8080"})
	}
	p, err := consumer(EmbedderKindOpenAI)
	if err != nil {
		t.Fatalf("consumer(openai): %v", err)
	}
	if _, ok := p.(*Embedder); !ok {
		t.Errorf("consumer(openai) 应返回 *Embedder, got %T", p)
	}
}

// TestEmbedderRegistry_UnknownKindError 未知 kind fail-closed（附已注册列表）。
func TestEmbedderRegistry_UnknownKindError(t *testing.T) {
	_, err := NewEmbedderByKind("no-such-embedder", EmbedderConfig{BaseURL: "http://x"})
	if err == nil {
		t.Fatal("未知 kind 应报错")
	}
	if !strings.Contains(err.Error(), EmbedderKindOpenAI) {
		t.Errorf("错误应附已注册 kind 列表: %v", err)
	}
}

// TestEmbedderRegistry_DuplicateKindPanics 互斥注册：重复即 panic。
func TestEmbedderRegistry_DuplicateKindPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("重复注册应 panic")
		}
	}()
	RegisterEmbedder(EmbedderKindOpenAI, func(cfg EmbedderConfig) (EmbeddingProvider, error) { return nil, nil })
}

// TestEmbedderRegistry_EmptyKindPanics 空 kind 注册直接 panic。
func TestEmbedderRegistry_EmptyKindPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("空 kind 应 panic")
		}
	}()
	RegisterEmbedder("", func(cfg EmbedderConfig) (EmbeddingProvider, error) { return nil, nil })
}

// TestRerankerRegistry_AllKinds "openai" kind 经注册表构建为 *Reranker。
func TestRerankerRegistry_AllKinds(t *testing.T) {
	kinds := RerankerKinds()
	if len(kinds) != 1 || kinds[0] != RerankerKindOpenAI {
		t.Fatalf("RerankerKinds = %v, want [openai]", kinds)
	}
	r, err := NewRerankerByKind(RerankerKindOpenAI, RerankerConfig{BaseURL: "http://localhost:8080", Model: "bge-reranker-v2-m3"})
	if err != nil {
		t.Fatalf("NewRerankerByKind(openai): %v", err)
	}
	if _, ok := r.(*Reranker); !ok {
		t.Fatalf("kind=openai 应返回 *Reranker, got %T", r)
	}
}

// TestRerankerRegistry_ConfigRouting 同形配置 + 不同 kind 得到不同实现。
func TestRerankerRegistry_ConfigRouting(t *testing.T) {
	var consumer func(kind string) (RerankProvider, error)
	consumer = func(kind string) (RerankProvider, error) {
		return NewRerankerByKind(kind, RerankerConfig{BaseURL: "http://127.0.0.1:8080"})
	}
	p, err := consumer(RerankerKindOpenAI)
	if err != nil {
		t.Fatalf("consumer(openai): %v", err)
	}
	if _, ok := p.(*Reranker); !ok {
		t.Errorf("consumer(openai) 应返回 *Reranker, got %T", p)
	}
}

// TestRerankerRegistry_UnknownKindError 未知 kind fail-closed（附已注册列表）。
func TestRerankerRegistry_UnknownKindError(t *testing.T) {
	_, err := NewRerankerByKind("no-such-reranker", RerankerConfig{BaseURL: "http://x"})
	if err == nil {
		t.Fatal("未知 kind 应报错")
	}
	if !strings.Contains(err.Error(), RerankerKindOpenAI) {
		t.Errorf("错误应附已注册 kind 列表: %v", err)
	}
}

// TestRerankerRegistry_DuplicateKindPanics 互斥注册：重复即 panic。
func TestRerankerRegistry_DuplicateKindPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("重复注册应 panic")
		}
	}()
	RegisterReranker(RerankerKindOpenAI, func(cfg RerankerConfig) (RerankProvider, error) { return nil, nil })
}

// TestRerankerRegistry_EmptyKindPanics 空 kind 注册直接 panic。
func TestRerankerRegistry_EmptyKindPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("空 kind 应 panic")
		}
	}()
	RegisterReranker("", func(cfg RerankerConfig) (RerankProvider, error) { return nil, nil })
}
