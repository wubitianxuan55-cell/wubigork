package knowledge

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

func TestSQLiteStoreCRUD(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)

	st, err := OpenSQLite(gdb)
	if err != nil {
		t.Fatal(err)
	}

	e := Entry{Name: "gb36600", Title: "土壤污染风险管控标准", Category: CatStandard, Tags: []string{"土壤", "标准"}, Body: "建设用地土壤污染风险管控标准 GB 36600-2018。"}
	if err := st.Save(e); err != nil {
		t.Fatal(err)
	}
	got, err := st.Get("gb36600")
	if err != nil || got.Title != e.Title || got.Category != CatStandard {
		t.Fatalf("Get = %+v err=%v", got, err)
	}
	if len(st.List()) != 1 {
		t.Errorf("List = %+v", st.List())
	}
	if idx := st.Index(); idx == "" || !containsStr(idx, "gb36600") {
		t.Errorf("Index = %q", idx)
	}
	if err := st.Delete("gb36600"); err != nil {
		t.Fatal(err)
	}
	if len(st.List()) != 0 {
		t.Error("entry not deleted")
	}
}

func TestMigrateLegacyKnowledge(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	defer db.CloseDatabase(dir)

	// 构造旧 Markdown 知识库
	oldDir := filepath.Join(dir, "legacy")
	if err := os.MkdirAll(oldDir, 0o755); err != nil {
		t.Fatal(err)
	}
	entry := "---\nname: soil-guide\ntitle: 土壤修复指南\ncategory: 经验总结\ntags: [土壤, 修复]\n---\n土壤修复方案编制要点。\n"
	if err := os.WriteFile(filepath.Join(oldDir, "soil-guide.md"), []byte(entry), 0o644); err != nil {
		t.Fatal(err)
	}

	n, err := MigrateLegacyKnowledge(gdb, oldDir)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if n != 1 {
		t.Errorf("migrated = %d, want 1", n)
	}
	st, _ := OpenSQLite(gdb)
	got, err := st.Get("soil-guide")
	if err != nil || got.Body == "" {
		t.Errorf("migrated entry missing: %+v err=%v", got, err)
	}

	// 幂等：再次调用跳过
	if n2, _ := MigrateLegacyKnowledge(gdb, oldDir); n2 != 0 {
		t.Errorf("second migrate = %d, want 0", n2)
	}
}

func TestSearchRAGFindsSemantic(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	defer db.CloseDatabase(dir)
	st, _ := OpenSQLite(gdb)

	// 条目正文没有查询词，但语义相关（通过 TF-IDF 召回）
	_ = st.Save(Entry{Name: "risk-assess", Title: "风险评估流程", Category: CatCase, Body: "污染场地健康风险评估包括危害识别、暴露评估、毒性评估与风险表征四个步骤。"})
	_ = st.Save(Entry{Name: "voice-chat", Title: "语音陪伴", Category: CatOther, Body: "语音识别与合成实现对话交互。"})

	// 查询"暴露评估 毒性评估"→ risk-assess 应排第一（RAG 语义召回）
	results := Search(st, "暴露评估 毒性评估", Filter{})
	if len(results) == 0 {
		t.Fatal("no results")
	}
	if results[0].Name != "risk-assess" {
		t.Errorf("top = %s, want risk-assess (got %+v)", results[0].Name, results)
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
