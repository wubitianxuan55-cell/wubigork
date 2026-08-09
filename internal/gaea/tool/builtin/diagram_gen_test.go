package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDiagramGenFramework(t *testing.T) {
	out := filepath.Join(t.TempDir(), "framework.png")
	args := map[string]interface{}{
		"title": "gaea 三层架构框架图",
		"kind":  "framework",
		"nodes": []map[string]interface{}{
			{"id": "app", "label": "应用层：项目管理 / 办公助手 / 知识库", "level": 0, "group": "应用层"},
			{"id": "svc", "label": "服务层：模型调度 / 文档引擎 / 记忆中枢", "level": 1, "group": "服务层"},
			{"id": "data", "label": "数据层：本地模型 / SQLite / 文件存储", "level": 2, "group": "数据层"},
		},
		"output": out,
	}
	raw, _ := json.Marshal(args)
	res, err := (diagramGen{}).Execute(context.Background(), raw)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var r struct {
		OK   bool   `json:"ok"`
		Path string `json:"output"`
	}
	if err := json.Unmarshal([]byte(res), &r); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if !r.OK {
		t.Fatalf("result not ok: %s", res)
	}
	info, err := os.Stat(r.Path)
	if err != nil {
		t.Fatalf("output missing: %v", err)
	}
	if info.Size() < 5000 {
		t.Fatalf("output suspiciously small: %d bytes", info.Size())
	}
}

func TestDiagramGenFlow(t *testing.T) {
	out := filepath.Join(t.TempDir(), "flow.png")
	args := map[string]interface{}{
		"title": "项目周报流程",
		"kind":  "flow",
		"nodes": []map[string]interface{}{
			{"id": "a", "label": "收集本周数据"},
			{"id": "b", "label": "汇总完成事项"},
			{"id": "c", "label": "输出周报文档"},
		},
		"edges": []map[string]interface{}{
			{"from": "a", "to": "b", "label": "整理"},
			{"from": "b", "to": "c", "label": "生成"},
		},
		"output": out,
	}
	raw, _ := json.Marshal(args)
	res, err := (diagramGen{}).Execute(context.Background(), raw)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var r struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal([]byte(res), &r); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if !r.OK {
		t.Fatalf("result not ok: %s", res)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("output missing: %v", err)
	}
}

func TestDiagramGenValidation(t *testing.T) {
	if _, err := (diagramGen{}).Execute(context.Background(), json.RawMessage(`{"nodes":[]}`)); err == nil {
		t.Fatal("expected empty-nodes error")
	}
	if _, err := (diagramGen{}).Execute(context.Background(), json.RawMessage(`{"nodes":[{"id":"a","label":"x"}],"kind":"bad"}`)); err == nil {
		t.Fatal("expected bad-kind error")
	}
}
