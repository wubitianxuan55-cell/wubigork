package app

import (
	"strings"
	"testing"
)

// E03：小说功能绑定后，路由不得回退全局陈旧模型名。
func TestRegressionE03NovelBindingBeatsStaleGlobal(t *testing.T) {
	c := newRouterTestCore(t)
	c.cfg.Model = "stale-xai-model"
	if err := c.SetFeatureModel("novel", "herdsman", "qwen3-8b"); err != nil {
		t.Fatal(err)
	}
	eng, model, _ := c.routeModel("novel")
	if eng != "herdsman" || model != "qwen3-8b" {
		t.Fatalf("E03 回归失败: (%q,%q)", eng, model)
	}
}

// E09：方案功能绑定后，office 路由使用绑定模型。
func TestRegressionE09OfficeBindingEffective(t *testing.T) {
	c := newRouterTestCore(t)
	c.cfg.Model = "stale-xai-model"
	if err := c.SetFeatureModel("office", "herdsman", "qwen3-8b"); err != nil {
		t.Fatal(err)
	}
	_, model, source := c.routeModel("office")
	if model != "qwen3-8b" || source != "feature" {
		t.Fatalf("E09 回归失败: (%q,%q)", model, source)
	}
}

// E10：办公 bridge 功能绑定后，gaea 路由使用绑定模型。
func TestRegressionE10GaeaBindingEffective(t *testing.T) {
	c := newRouterTestCore(t)
	c.cfg.Model = "stale-xai-model"
	if err := c.SetFeatureModel("gaea", "herdsman", "qwen3-8b"); err != nil {
		t.Fatal(err)
	}
	_, model, source := c.routeModel("gaea")
	if model != "qwen3-8b" || source != "feature" {
		t.Fatalf("E10 回归失败: (%q,%q)", model, source)
	}
}

// 路由事件可观测：source 字段区分 feature/global/fallback。
func TestRegressionModelRouteSourceLabels(t *testing.T) {
	c := newRouterTestCore(t)
	if err := c.SetFeatureModel("whisper", "herdsman", "qwen3-8b"); err != nil {
		t.Fatal(err)
	}
	_, _, source := c.routeModel("whisper")
	if source != "feature" {
		t.Fatalf("source = %q", source)
	}
	if !strings.Contains("feature global fallback", source) {
		t.Fatalf("非法 source: %q", source)
	}
}
