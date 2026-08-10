package knowledgeimport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/gaea/gaea/internal/gaea/knowledge"
)

func newTestStore(t *testing.T) *knowledge.Store {
	t.Helper()
	st, err := knowledge.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func TestParseMarkdown(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "桩基施工要点.md")
	content := "# 桩基施工要点\n\n振动锤选型需匹配地质条件…（案例经验）"
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	pv, err := Parse(p, nil)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(pv.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(pv.Rows))
	}
	r := pv.Rows[0]
	if r.Title != "桩基施工要点" || !strings.Contains(r.Body, "振动锤选型") || r.MatchNote != "新增" {
		t.Errorf("row wrong: %+v", r)
	}
}

func TestParseMarkdownOverwriteMatch(t *testing.T) {
	store := newTestStore(t)
	_ = store.Save(knowledge.Entry{Name: "existing-1", Title: "桩基施工要点", Category: knowledge.CatCase, Body: "旧正文", Status: "现行"})
	dir := t.TempDir()
	p := filepath.Join(dir, "桩基施工要点.md")
	if err := os.WriteFile(p, []byte("新正文内容"), 0o644); err != nil {
		t.Fatal(err)
	}
	pv, err := Parse(p, store)
	if err != nil {
		t.Fatal(err)
	}
	if pv.Rows[0].MatchNote != "将覆盖更新" || pv.Rows[0].ExistingName != "existing-1" {
		t.Errorf("expected overwrite match: %+v", pv.Rows[0])
	}
}

func TestParseXLSX(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "规范条文表.xlsx")
	f := excelize.NewFile()
	defer f.Close()
	sheet := f.GetSheetName(0)
	_ = f.SetCellValue(sheet, "A1", "标题")
	_ = f.SetCellValue(sheet, "B1", "分类")
	_ = f.SetCellValue(sheet, "C1", "正文")
	_ = f.SetCellValue(sheet, "A2", "GB 36600 风险管控")
	_ = f.SetCellValue(sheet, "B2", "规范标准")
	_ = f.SetCellValue(sheet, "C2", "建设用地土壤污染风险管控标准要点")
	if err := f.SaveAs(p); err != nil {
		t.Fatal(err)
	}
	pv, err := Parse(p, nil)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(pv.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(pv.Rows))
	}
	r := pv.Rows[0]
	if r.Title != "GB 36600 风险管控" || r.Category != "规范标准" || !strings.Contains(r.Body, "土壤污染") {
		t.Errorf("xlsx row wrong: %+v", r)
	}
}

func TestSlugName(t *testing.T) {
	a := slugName("P.O 42.5 水泥")
	if a != "p-o-42-5-水泥" {
		t.Errorf("slug = %q", a)
	}
}

func TestSimilarity(t *testing.T) {
	if s := Similarity("桩基施工要点", "桩基施工 要点"); s < 0.7 {
		t.Errorf("near-duplicate similarity = %v, want >=0.7", s)
	}
	if s := Similarity("桩基施工要点", "水泥材料"); s >= 0.3 {
		t.Errorf("unrelated similarity = %v, want <0.3", s)
	}
}

func TestFindSimilar(t *testing.T) {
	store := newTestStore(t)
	_ = store.Save(knowledge.Entry{Name: "pile", Title: "桩基施工要点", Category: knowledge.CatCase, Body: "要点", Status: "现行"})
	hits := FindSimilar(store, "桩基施工 要点（修订）", 0.55)
	if len(hits) != 1 || hits[0].Name != "pile" || hits[0].Score < 0.6 {
		t.Errorf("FindSimilar = %+v, want pile 高分", hits)
	}
}

func TestMatchRowsSimilarNote(t *testing.T) {
	store := newTestStore(t)
	_ = store.Save(knowledge.Entry{Name: "pile", Title: "桩基施工要点", Category: knowledge.CatCase, Body: "要点", Status: "现行"})
	rows := MatchRows([]Row{{
		Title: "桩基施工 要点（修订稿）", Category: "工程案例", Status: "现行",
		Body: "新内容",
	}}, store)
	if len(rows) != 1 || rows[0].MatchNote != "新增" || rows[0].SimilarName == "" {
		t.Errorf("expected similar suggestion: %+v", rows[0])
	}
}
