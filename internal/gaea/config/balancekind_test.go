package config

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

// TestBalanceKindRoundTrip 3.0 Wave 4：ProviderEntry.BalanceKind 经
// UpsertProvider + SaveTo（RenderTOML）写盘、再 TOML 解码回来值不变。
func TestBalanceKindRoundTrip(t *testing.T) {
	c := Default()
	if err := c.UpsertProvider(ProviderEntry{
		Name:        "local",
		Kind:        "openai",
		BaseURL:     "http://localhost:1234/v1",
		Model:       "llama",
		BalanceURL:  "http://localhost:1234/balance",
		BalanceKind: "custom-backend",
	}); err != nil {
		t.Fatalf("UpsertProvider: %v", err)
	}

	path := filepath.Join(t.TempDir(), "nested", "gaea.toml")
	if err := c.SaveTo(path); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}

	var got Config
	if _, err := toml.DecodeFile(path, &got); err != nil {
		t.Fatalf("saved file does not parse: %v", err)
	}
	p, ok := got.Provider("local")
	if !ok {
		t.Fatal("provider 'local' missing after round-trip")
	}
	if p.BalanceKind != "custom-backend" {
		t.Errorf("balance_kind = %q, want custom-backend", p.BalanceKind)
	}
	if p.BalanceURL != "http://localhost:1234/balance" {
		t.Errorf("balance_url not preserved: %q", p.BalanceURL)
	}
}

// TestBalanceKindRenderOnlyWhenSet balance_kind 只在非空时渲染：
// 空 kind 不输出该行（保持既有 gaea.toml 输出风格），渲染结果仍可解析。
func TestBalanceKindRenderOnlyWhenSet(t *testing.T) {
	c := Default()
	rendered := RenderTOML(c)
	if strings.Contains(rendered, "balance_kind") {
		t.Errorf("空 BalanceKind 不应渲染 balance_kind 行:\n%s", rendered)
	}

	// 设置后渲染并回读。
	mm, _ := c.Provider("mimo-pro")
	mm.BalanceKind = "custom-backend"
	rendered = RenderTOML(c)
	if !strings.Contains(rendered, `balance_kind = "custom-backend"`) {
		t.Fatalf("渲染缺少 balance_kind 行:\n%s", rendered)
	}
	var got Config
	if _, err := toml.Decode(rendered, &got); err != nil {
		t.Fatalf("rendered TOML does not parse: %v\n---\n%s", err, rendered)
	}
	if g, _ := got.Provider("mimo-pro"); g == nil || g.BalanceKind != "custom-backend" {
		t.Errorf("mimo-pro balance_kind not preserved: %+v", g)
	}
}
