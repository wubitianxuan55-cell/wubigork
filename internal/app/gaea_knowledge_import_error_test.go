package app

// 审计 P0#6（GA3-01）绑定层验收：知识库后端读失败时，导入预览必须**中止并报错**，
// 不得把读失败吞成「空库」后把每一行判成「新增」（静默重复入库）。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/knowledge"
)

// setKnowledgeStoreToClosedDB 注入一个后端 DB 已关闭的 Store：任何查询都失败。
func setKnowledgeStoreToClosedDB(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	st, err := knowledge.OpenSQLite(gdb)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.CloseDatabase(dir); err != nil {
		t.Fatalf("关闭 db: %v", err)
	}
	knowledge.SetStoreForTest(st)
	t.Cleanup(func() { knowledge.SetStoreForTest(nil) })
}

func writeKnowledgeImportDoc(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "土壤修复要点.md")
	if err := os.WriteFile(p, []byte("# 土壤修复要点 土壤修复方案编制要点。"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestGaeaKnowledgeImportPreviewAbortsWhenStoreUnreadable(t *testing.T) {
	setKnowledgeStoreToClosedDB(t)
	a := &App{}
	pv, err := a.GaeaKnowledgeImportPreview(writeKnowledgeImportDoc(t))
	if err == nil {
		note := ""
		if len(pv.Rows) > 0 {
			note = pv.Rows[0].MatchNote
		}
		t.Fatalf("库不可读时必须中止预览并报错，实际 err=nil rows=%d 首行 MatchNote=%q", len(pv.Rows), note)
	}
	if !strings.Contains(err.Error(), "知识库暂不可读") {
		t.Errorf("错误文案应提示「知识库暂不可读…可重试」，实际: %v", err)
	}
	if len(pv.Rows) != 0 {
		t.Errorf("中止时不应返回预览行，实际 %d 行", len(pv.Rows))
	}
}

// 对照组：库可读但为空 → 不是错误，仍应给出「新增」预览（区分「表空」与「读失败」）。
func TestGaeaKnowledgeImportPreviewTreatsEmptyStoreAsNew(t *testing.T) {
	st, err := knowledge.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	knowledge.SetStoreForTest(st)
	t.Cleanup(func() { knowledge.SetStoreForTest(nil) })

	a := &App{}
	pv, err := a.GaeaKnowledgeImportPreview(writeKnowledgeImportDoc(t))
	if err != nil {
		t.Fatalf("空库不是错误: %v", err)
	}
	if len(pv.Rows) != 1 || pv.Rows[0].MatchNote != "新增" {
		t.Fatalf("空库应判「新增」: %+v", pv.Rows)
	}
}
