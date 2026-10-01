package app

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

// TestGetOutlines_FallbackToPMWhenAgentNil 实弹事故回归：项目已开但 outlineAgent
// 未就绪（启动竞态）时，GetOutlines 直读盘兜底——前端大纲库不得空转
// （空库会让下一章恒算 1，反复覆盖第1章）。
// newTestOutlineFile 测试大纲：一条 done 第1章（用户实盘迁移后的形状）。
func newTestOutlineFile() *types.OutlineFile {
	return &types.OutlineFile{Nodes: []types.OutlineNode{
		{ID: "n1", OrderIndex: 1, Title: "第1章", Status: types.OutlineDone},
	}}
}

func TestGetOutlines_FallbackToPMWhenAgentNil(t *testing.T) {
	pm := newStatesTestProject(t)
	if err := pm.WriteOutlines(newTestOutlineFile()); err != nil {
		t.Fatalf("写大纲: %v", err)
	}
	a := &writingState{core: &core{}}
	a.setPM(pm)
	// outlineAgent 刻意保持 nil（模拟启动竞态）

	got := a.GetOutlines()
	raw, _ := json.Marshal(got["nodes"])
	if !strings.Contains(string(raw), "第1章") {
		t.Fatalf("agent 未就绪应回落直读盘读到大纲，got %s", raw)
	}
}
