package config

// config_features_test.go — 功能级模型绑定域测试（2026-11-15 图文专属绑定）。
// 钉死 wenxue 功能键四向（Get/Set/Enabled 双向）、saveSetters 落键正确、
// 与 applyConfigFile 回读链（临时家目录，hermetic 不碰真实 ~/.gaea_config.json）。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWenxueFeatureModelRoundtrip(t *testing.T) {
	cfg := loadDefaults(t.TempDir())
	if v := cfg.GetFeatureModelEnabled("wenxue"); !v {
		t.Fatal("wenxue 功能默认应启用（loadDefaults true）")
	}
	if eng, mdl := cfg.GetFeatureModel("wenxue"); eng != "" || mdl != "" {
		t.Fatalf("默认绑定应为空，读到 %q/%q", eng, mdl)
	}
	cfg.SetFeatureModel("wenxue", "deepseek", "deepseek-chat")
	cfg.SetFeatureModelEnabled("wenxue", false)
	if eng, mdl := cfg.GetFeatureModel("wenxue"); eng != "deepseek" || mdl != "deepseek-chat" {
		t.Fatalf("Set 后读回不符: %q/%q", eng, mdl)
	}
	if cfg.GetFeatureModelEnabled("wenxue") {
		t.Fatal("SetFeatureModelEnabled(false) 未生效")
	}
	// 与 gaea 键域隔离（各表各的）
	if eng, _ := cfg.GetFeatureModel("gaea"); eng == "deepseek" {
		t.Fatal("wenxue 绑定不应串进 gaea 键域")
	}
}

func TestWenxueFeatureSaveSettersAndLoad(t *testing.T) {
	// saveSetters 三键写入 configFile → JSON 落盘字段名正确
	cf := configFile{}
	for _, tc := range []struct {
		key, val string
		wantJSON string
	}{
		{KeyFuncWenxueEngine, "herdsman", `"func_wenxue_engine":"herdsman"`},
		{KeyFuncWenxueModel, "qwen3.5-uncensored", `"func_wenxue_model":"qwen3.5-uncensored"`},
		{KeyFuncWenxueEnabled, "false", `"func_wenxue_enabled":false`},
	} {
		setter, ok := saveSetters[tc.key]
		if !ok {
			t.Fatalf("saveSetters 缺键 %s", tc.key)
		}
		if err := setter(&cf, tc.val); err != nil {
			t.Fatalf("setter %s: %v", tc.key, err)
		}
		b, _ := json.Marshal(cf)
		if !containsStr(string(b), tc.wantJSON) {
			t.Fatalf("落盘 JSON 缺 %s：%s", tc.wantJSON, b)
		}
	}

	// applyConfigFile 回读链：临时家目录 .gaea_config.json → Config
	home := t.TempDir()
	doc := `{"func_wenxue_engine":"herdsman","func_wenxue_model":"qwen3.5-uncensored","func_wenxue_enabled":false}`
	if err := os.WriteFile(filepath.Join(home, ".gaea_config.json"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := loadDefaults(home)
	applyConfigFile(cfg, home)
	if eng, mdl := cfg.GetFeatureModel("wenxue"); eng != "herdsman" || mdl != "qwen3.5-uncensored" {
		t.Fatalf("回读绑定不符: %q/%q", eng, mdl)
	}
	if cfg.GetFeatureModelEnabled("wenxue") {
		t.Fatal("回读启停不符（应 false）")
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
