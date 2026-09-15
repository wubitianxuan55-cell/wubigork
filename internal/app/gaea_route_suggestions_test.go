package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/routesuggest"
)

// TestRouteSuggestionRecordRoundTrip 状态文件读写往返：保存→读取→覆盖→再读。
func TestRouteSuggestionRecordRoundTrip(t *testing.T) {
	dir := t.TempDir()
	id := routesuggest.BuildSuggestionID("chat", "xai", "grok-4", "deepseek", "deepseek-chat")

	if err := saveRouteSuggestionRecord(dir, id, "ignored"); err != nil {
		t.Fatalf("首次保存失败: %v", err)
	}
	recs := loadRouteSuggestionRecords(dir)
	if recs[id].Status != "ignored" || recs[id].DecidedAt == "" {
		t.Fatalf("读取不符: %+v", recs[id])
	}

	if err := saveRouteSuggestionRecord(dir, id, "applied"); err != nil {
		t.Fatalf("覆盖保存失败: %v", err)
	}
	recs = loadRouteSuggestionRecords(dir)
	if recs[id].Status != "applied" {
		t.Fatalf("覆盖后状态应为 applied: %+v", recs[id])
	}

	// 文件落盘形状：版本 1 + records 映射。
	b, err := os.ReadFile(routeSuggestionsPath(dir))
	if err != nil {
		t.Fatalf("读取状态文件失败: %v", err)
	}
	var probe struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(b, &probe); err != nil || probe.Version != 1 {
		t.Fatalf("状态文件形状不符（version=1）: %s", b)
	}
}

// TestLoadRouteSuggestionRecordsTolerant 文件缺失/损坏/空 records 均回空表（建议可重算，状态丢得起）。
func TestLoadRouteSuggestionRecordsTolerant(t *testing.T) {
	dir := t.TempDir()
	if recs := loadRouteSuggestionRecords(dir); recs == nil || len(recs) != 0 {
		t.Fatalf("文件缺失应回空表: %+v", recs)
	}
	if err := os.WriteFile(routeSuggestionsPath(dir), []byte("{not-json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if recs := loadRouteSuggestionRecords(dir); len(recs) != 0 {
		t.Fatalf("损坏文件应回空表: %+v", recs)
	}
	if err := os.WriteFile(routeSuggestionsPath(dir), []byte(`{"version":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if recs := loadRouteSuggestionRecords(dir); len(recs) != 0 {
		t.Fatalf("records 缺省应回空表: %+v", recs)
	}
}

// TestRouteSuggestionIgnoreApplyValidateIDs 空壳 App（engineMgr/cfg 就绪即可）验证
// ID 解析防御：非法 ID 直接报错，不触盘。
func TestRouteSuggestionIgnoreValidateIDs(t *testing.T) {
	dir := t.TempDir()
	for _, bad := range []string{"", "chat", "chat|xai>grok-4"} {
		err := saveRouteSuggestionRecord(dir, bad, "ignored")
		if err == nil {
			t.Fatalf("非法 ID 不应落盘: %q", bad)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "route_suggestions.json")); !os.IsNotExist(err) {
		t.Fatal("非法 ID 不应产生状态文件")
	}
}
