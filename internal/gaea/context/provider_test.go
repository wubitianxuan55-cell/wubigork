package context

import "testing"

func TestLookupProviderDeepSeek(t *testing.T) {
	h := LookupProvider("deepseek")
	if !h.HasPrefixCache {
		t.Error("deepseek must have prefix cache")
	}
	if h.CompactTrigger != 0.80 {
		t.Errorf("deepseek CompactTrigger = %v, want 0.80", h.CompactTrigger)
	}
	if h.TailTokens != 16384 {
		t.Errorf("deepseek TailTokens = %d, want 16384", h.TailTokens)
	}
}

func TestLookupProviderOpenAI(t *testing.T) {
	h := LookupProvider("openai")
	if !h.HasPrefixCache {
		t.Error("openai must have prefix cache")
	}
	if h.CompactTrigger != 0.70 {
		t.Errorf("openai CompactTrigger = %v, want 0.70", h.CompactTrigger)
	}
}

func TestLookupProviderAnthropic(t *testing.T) {
	h := LookupProvider("anthropic")
	if h.HasPrefixCache {
		t.Error("anthropic must NOT have prefix cache")
	}
	if h.CompactTrigger != 0.60 {
		t.Errorf("anthropic CompactTrigger = %v, want 0.60", h.CompactTrigger)
	}
	if h.TailTokens != 8192 {
		t.Errorf("anthropic TailTokens = %d, want 8192", h.TailTokens)
	}
}

func TestLookupProviderUnknown(t *testing.T) {
	h := LookupProvider("some-unknown-provider")
	if h.Name != "unknown" {
		t.Errorf("unknown provider name = %q, want unknown", h.Name)
	}
	if h.HasPrefixCache {
		t.Error("unknown provider must use conservative no-prefix-cache default")
	}
	if h.CompactTrigger != 0.65 {
		t.Errorf("unknown CompactTrigger = %v, want 0.65", h.CompactTrigger)
	}
	if h.TailTokens != 8192 {
		t.Errorf("unknown TailTokens = %d, want 8192", h.TailTokens)
	}
}

func TestLookupProviderCaseInsensitive(t *testing.T) {
	h := LookupProvider("DeepSeek-Chat")
	if !h.HasPrefixCache {
		t.Error("case-insensitive deepseek lookup must find prefix cache")
	}
}

func TestProviderHintApplyToPolicy(t *testing.T) {
	p := CompactPolicy{}
	h := LookupProvider("deepseek")
	h.ApplyToPolicy(&p)
	if p.Ratio != 0.80 {
		t.Errorf("Ratio after ApplyToPolicy = %v, want 0.80", p.Ratio)
	}
	if p.TailTokens != 16384 {
		t.Errorf("TailTokens after ApplyToPolicy = %d, want 16384", p.TailTokens)
	}

	// pre-set values are preserved
	p2 := CompactPolicy{Ratio: 0.5, TailTokens: 1000}
	h.ApplyToPolicy(&p2)
	if p2.Ratio != 0.5 || p2.TailTokens != 1000 {
		t.Errorf("ApplyToPolicy must not override pre-set values, got %+v", p2)
	}
}
