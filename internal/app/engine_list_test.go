package app

import (
	"reflect"
	"testing"

	"github.com/gaea/gaea/internal/modelengine"
)

// TestGetEngineList_NilManager 模型中心未初始化时回退默认引擎。
func TestGetEngineList_NilManager(t *testing.T) {
	a := &App{} // engineMgr 为 nil
	got := a.GetEngineList()
	want := []string{"default"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetEngineList() = %v, want %v", got, want)
	}
}

// TestGetEngineList_WithEngines 返回模型中心预置引擎 ID。
func TestGetEngineList_WithEngines(t *testing.T) {
	a := &App{engineMgr: modelengine.NewManager("", "")}
	got := a.GetEngineList()
	// NewManager 预置 xai/ollama 等引擎，必须全部出现在列表中
	for _, id := range []string{"xai", "ollama"} {
		found := false
		for _, g := range got {
			if g == id {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("GetEngineList() = %v 缺少引擎 %q", got, id)
		}
	}
}

// TestGaeaVersion 办公板块版本与产品 V1.0.0 一致。
func TestGaeaVersion(t *testing.T) {
	a := &App{}
	if v := a.GaeaVersion(); v != "1.0.0" {
		t.Fatalf("GaeaVersion() = %q, want 1.0.0", v)
	}
}
