package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/retrieval"
	"github.com/gaea/gaea/internal/gaea/semantic"
	"github.com/gaea/gaea/internal/gaea/tasks"
)

// TestSemanticSearchTool_EndToEnd 本地 bge-m3 mock + 临时向量库：
// 云端 agent 通过 semantic_search 工具按语义命中成本/知识条目。
func TestSemanticSearchTool_EndToEnd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Input []string `json:"input"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		data := make([]map[string]any, 0, len(req.Input))
		for i, s := range req.Input {
			vec := []float32{0, 0}
			if strings.Contains(s, "驳接爪") {
				vec[0] = 1
			}
			if strings.Contains(s, "水泥") {
				vec[1] = 1
			}
			data = append(data, map[string]any{"index": i, "embedding": vec})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	defer srv.Close()

	gdb := db.GetDatabase(home)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() {
		db.CloseDatabase(home)
		SetAppSemanticStoreForTest(nil)
		SetAppEmbedderForTest(nil)
	})
	SetAppSemanticStoreForTest(semantic.Open(gdb))
	SetAppEmbedderForTest(retrieval.NewEmbedder(srv.URL, "bge-m3"))

	a := &App{}
	tool := semanticSearchTool{a: a}

	reply, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"不锈钢驳接爪 90 度"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Contains(reply, "没有找到") {
		t.Fatalf("语义检索未命中: %s", reply)
	}
	if !strings.Contains(reply, "相似度") {
		t.Errorf("返回缺少相似度字段: %s", reply)
	}
}

// TestSemanticSearchTool_EmptyQuery 空 query 应返回参数错误。
func TestSemanticSearchTool_EmptyQuery(t *testing.T) {
	a := &App{}
	tool := semanticSearchTool{a: a}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"query":""}`)); err == nil {
		t.Error("空 query 应报错")
	}
}

// TestSemanticSearchTool_FileScope 工作区文件语义检索：scope=file 命中已索引文件。
func TestSemanticSearchTool_FileScope(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	root := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(old) }()

	_ = os.WriteFile(filepath.Join(root, "说明.md"), []byte("振动锤选型要点 桩基施工"), 0o644)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Input []string `json:"input"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		data := make([]map[string]any, 0, len(req.Input))
		for i, s := range req.Input {
			vec := []float32{0, 0}
			if strings.Contains(s, "锤") || strings.Contains(s, "桩") {
				vec[0] = 1
			}
			data = append(data, map[string]any{"index": i, "embedding": vec})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	defer srv.Close()

	gdb := db.GetDatabase(home)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() {
		db.CloseDatabase(home)
		SetAppSemanticStoreForTest(nil)
		SetAppEmbedderForTest(nil)
	})
	SetAppSemanticStoreForTest(semantic.Open(gdb))
	SetAppEmbedderForTest(retrieval.NewEmbedder(srv.URL, "bge-m3"))

	a := &App{officeState: &officeState{core: &core{}}}
	m := tasks.New(gdb, nil, tasks.Options{})
	m.Register(tasks.KindFileIndex, a.fileIndexTaskHandler)
	if _, err := m.Start(); err != nil {
		t.Fatalf("task manager start: %v", err)
	}
	t.Cleanup(m.Close)
	a.officeState.tasks = m
	tk, err := a.GaeaFileIndexRebuild()
	if err != nil {
		t.Fatalf("索引重建提交失败: %v", err)
	}
	done, err := m.Wait(context.Background(), tk.ID, 30*time.Second)
	if err != nil || done.Status != "succeeded" {
		t.Fatalf("索引重建失败: %v status=%s err=%s", err, done.Status, done.Error)
	}
	tool := semanticSearchTool{a: a}
	reply, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"打桩锤","scope":"file"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Contains(reply, "没有找到") {
		t.Fatalf("文件检索未命中: %s", reply)
	}
	if !strings.Contains(reply, "说明.md") {
		t.Errorf("返回缺少文件路径: %s", reply)
	}
}

// TestSpecialistTools_Registered 固化 3.0 Step 3d #6：ocr 等专业工具已注册
// 进 ExtraTools（gaeaSpecialistTools 集中注册），不再是死代码。
func TestSpecialistTools_Registered(t *testing.T) {
	tools := gaeaSpecialistTools(&App{})
	names := make([]string, 0, len(tools))
	for _, tl := range tools {
		names = append(names, tl.Name())
	}
	joined := strings.Join(names, ",")
	if !strings.Contains(joined, "ocr") {
		t.Fatalf("专业工具应包含 ocr, got %v", names)
	}
}

// TestOCRTool_Validation ocr 工具参数校验：空路径与不存在的文件。
func TestOCRTool_Validation(t *testing.T) {
	a := &App{}
	tool := ocrTool{a: a}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"image_path":""}`)); err == nil {
		t.Error("空 image_path 应报错")
	}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"image_path":"Z:\\no-such-file.png"}`)); err == nil {
		t.Error("不存在的图片应报错")
	}
}
