package proposal

import (
	"path/filepath"
	"testing"

	officedb "github.com/gaea/gaea/internal/office/db"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	db := officedb.GetDatabase(dir)
	if db == nil {
		t.Fatal("office.db 不可用")
	}
	t.Cleanup(func() { _ = officedb.CloseDatabase(dir) })
	return NewStore(db, dir)
}

func TestProjectCRUD(t *testing.T) {
	s := newTestStore(t)
	def, err := s.EnsureDefaultProject()
	if err != nil {
		t.Fatalf("EnsureDefaultProject: %v", err)
	}
	if def.ID != "default" || def.Name != "未归档项目" {
		t.Errorf("default project = %+v", def)
	}
	got, err := s.GetProject(def.ID)
	if err != nil || got.Name != "未归档项目" {
		t.Fatalf("GetProject(default): %v %+v", err, got)
	}
	p, err := s.CreateProject("某修复项目", "环保工程", "某公司")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	projs, err := s.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projs) != 2 {
		t.Errorf("len(projects) = %d, want 2", len(projs))
	}
	if err := s.DeleteProject(p.ID); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	if _, err := s.GetProject(p.ID); err == nil {
		t.Error("GetProject 删除后应报错")
	}
}

func TestProposalCRUDWithSectionsAndVersion(t *testing.T) {
	s := newTestStore(t)
	proj, _ := s.EnsureDefaultProject()
	sections := []ProposalSection{
		{Index: 0, Title: "第一章", Level: 1, Status: "pending", Children: []ProposalSection{
			{Index: 1, Title: "1.1", Level: 2, Status: "pending"},
		}},
		{Index: 2, Title: "第二章", Level: 1, Status: "pending"},
	}
	p, err := s.Create("土壤修复方案", "soil-remediation-bid", "需求", "环保工程", proj.ID, sections)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.Version != 1 || len(p.Sections) != 2 || len(p.Sections[0].Children) != 1 {
		t.Fatalf("Create 结果异常: version=%d sections=%d", p.Version, len(p.Sections))
	}
	got, err := s.Get(p.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ProjectID != proj.ID || len(got.Sections[0].Children) != 1 {
		t.Fatalf("Get 树形恢复异常: %+v", got.Sections)
	}
	got.Sections[0].Content = "第一章内容"
	got.Sections[0].Status = "completed"
	if err := s.Update(got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	updated, err := s.Get(p.ID)
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if updated.Version != 2 {
		t.Errorf("version = %d, want 2", updated.Version)
	}
	if updated.Sections[0].Content != "第一章内容" {
		t.Errorf("content 未持久化: %q", updated.Sections[0].Content)
	}
	var vn int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM versions WHERE proposal_id = ?", p.ID).Scan(&vn); err != nil {
		t.Fatal(err)
	}
	if vn != 2 {
		t.Errorf("versions rows = %d, want 2", vn)
	}
	list, err := s.List()
	if err != nil || len(list) != 1 {
		t.Fatalf("List: %v len=%d", err, len(list))
	}
	if err := s.Delete(p.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	var cn int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM sections WHERE proposal_id = ?", p.ID).Scan(&cn); err != nil {
		t.Fatal(err)
	}
	if cn != 0 {
		t.Errorf("级联删除后 sections = %d, want 0", cn)
	}
}

func TestStoreAddFile(t *testing.T) {
	s := newTestStore(t)
	proj, _ := s.EnsureDefaultProject()
	p, err := s.Create("方案", "blank", "", "其他", proj.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddFile(p.ID, "attachment", "a.pdf", filepath.Join(s.FilesDir(), p.ID, "a.pdf"), 10); err != nil {
		t.Fatalf("AddFile: %v", err)
	}
	var fn int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM files WHERE proposal_id = ?", p.ID).Scan(&fn); err != nil {
		t.Fatal(err)
	}
	if fn != 1 {
		t.Errorf("files rows = %d, want 1", fn)
	}
}

func TestParseResultsCRUD(t *testing.T) {
	s := newTestStore(t)
	proj, _ := s.EnsureDefaultProject()
	p, err := s.Create("方案", "blank", "", "其他", proj.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	items := []ParseResultItem{
		{FileID: "f1", Field: "duration", Value: "90 日历天", Page: 3, Start: 100, End: 120, Snippet: "工期 90 日历天", Confidence: 1},
		{FileID: "f1", Field: "techScoring", Value: "施工方案 20 分", Page: 5, Start: 300, End: 330, Snippet: "施工方案（20 分）", Confidence: 0.8},
	}
	if err := s.SaveParseResults(p.ID, items); err != nil {
		t.Fatalf("SaveParseResults: %v", err)
	}
	got, err := s.ListParseResults(p.ID)
	if err != nil {
		t.Fatalf("ListParseResults: %v", err)
	}
	if len(got) != 2 || got[0].Field != "duration" || got[0].Page != 3 {
		t.Fatalf("解析结果异常: %+v", got)
	}
	// 幂等重存：先删后插
	if err := s.SaveParseResults(p.ID, []ParseResultItem{{FileID: "f1", Field: "overview", Value: "项目概况", Snippet: "本项目为…"}}); err != nil {
		t.Fatal(err)
	}
	got2, _ := s.ListParseResults(p.ID)
	if len(got2) != 1 || got2[0].Field != "overview" {
		t.Fatalf("重存后异常: %+v", got2)
	}
}

func TestProjectFactsRoundTrip(t *testing.T) {
	s := newTestStore(t)
	proj, _ := s.EnsureDefaultProject()
	facts := map[string]string{
		"工期": "90 日历天", "业主单位": "某区生态环境局",
		"修复目标": "砷 ≤ 60 mg/kg", "项目经理": "张三",
	}
	if err := s.SaveProjectFacts(proj.ID, facts); err != nil {
		t.Fatalf("SaveProjectFacts: %v", err)
	}
	got, err := s.GetProjectFacts(proj.ID)
	if err != nil {
		t.Fatalf("GetProjectFacts: %v", err)
	}
	if got["工期"] != "90 日历天" || got["修复目标"] != "砷 ≤ 60 mg/kg" {
		t.Fatalf("facts 往返异常: %+v", got)
	}
}
