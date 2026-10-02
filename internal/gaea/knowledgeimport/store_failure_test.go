package knowledgeimport

// 审计 P0#6（GA3-01）：消费侧不得把知识库读失败当空库——查重落空会把每一行
// 判成「新增」，确认后按同名覆盖写库形成静默重复入库。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/knowledge"
)

func newClosedStore(t *testing.T) *knowledge.Store {
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
	return st
}

func TestMatchRowsReportsStoreReadFailure(t *testing.T) {
	rows, err := MatchRows([]Row{{Title: "土壤修复要点", Category: "工程案例", Body: "正文"}}, newClosedStore(t))
	if err == nil {
		t.Fatalf("库读失败必须返回 error，实际 rows=%+v（每行都会判「新增」）", rows)
	}
	if rows != nil {
		t.Errorf("读失败时不应返回预览行，实际 %+v", rows)
	}
	if !strings.Contains(err.Error(), "知识库暂不可读") {
		t.Errorf("错误文案应提示「知识库暂不可读…可重试」，实际: %v", err)
	}
}

func TestFindSimilarReportsStoreReadFailure(t *testing.T) {
	hits, err := FindSimilar(newClosedStore(t), "土壤修复要点", 0.65)
	if err == nil {
		t.Fatalf("库读失败必须返回 error，实际 hits=%+v（查重恒空且无提示）", hits)
	}
	if hits != nil {
		t.Errorf("读失败时不应返回命中，实际 %+v", hits)
	}
}

func TestParseAbortsWhenStoreUnreadable(t *testing.T) {
	p := filepath.Join(t.TempDir(), "土壤修复要点.md")
	if err := os.WriteFile(p, []byte("土壤修复方案编制要点。"), 0o644); err != nil {
		t.Fatal(err)
	}
	pv, err := Parse(p, newClosedStore(t))
	if err == nil {
		t.Fatalf("库读失败必须中止解析并报错，实际 pv=%+v", pv)
	}
	if pv != nil {
		t.Errorf("中止时不应返回预览，实际 %+v", pv)
	}
}

// 对照组：库可读但为空 → 不是错误，仍判「新增」。
func TestMatchRowsEmptyStoreIsNotError(t *testing.T) {
	st, err := knowledge.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	rows, err := MatchRows([]Row{{Title: "土壤修复要点", Category: "工程案例", Body: "正文"}}, st)
	if err != nil {
		t.Fatalf("空库不是错误: %v", err)
	}
	if len(rows) != 1 || rows[0].MatchNote != "新增" {
		t.Fatalf("空库应判「新增」: %+v", rows)
	}
}
