package proposal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	officedb "github.com/gaea/gaea/internal/office/db"
)

func writeLegacyFixture(t *testing.T, dir string) {
	t.Helper()
	proposalDir := filepath.Join(dir, "office", "proposals")
	if err := os.MkdirAll(proposalDir, 0755); err != nil {
		t.Fatal(err)
	}
	p := Proposal{
		ID: "legacy-1", Title: "旧方案", Category: "环保工程", Template: "soil-remediation-bid",
		Requirements: "旧需求", Status: "writing",
		Sections: []ProposalSection{
			{ID: "s1", ProposalID: "legacy-1", Index: 0, Level: 1, Title: "第一章", Status: "completed", Content: "正文"},
		},
		CreatedAt: "2026-07-01 10:00:00", UpdatedAt: "2026-07-01 10:00:00",
		BidSummary: &BidSummary{RawFiles: []FileDoc{{Name: "招标.pdf", Path: filepath.Join(proposalDir, "uploads", "legacy-1", "招标.pdf"), Size: 10}}},
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proposalDir, "legacy-1.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	idx, _ := json.Marshal([]indexEntry{{ID: "legacy-1", Title: "旧方案", Category: "环保工程", Template: "soil-remediation-bid", Status: "writing", UpdatedAt: "2026-07-01 10:00:00"}})
	if err := os.WriteFile(filepath.Join(proposalDir, "index.json"), idx, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestMigrateLegacyJSON(t *testing.T) {
	dir := t.TempDir()
	writeLegacyFixture(t, dir)

	db := officedb.GetDatabase(filepath.Join(dir, "office"))
	if db == nil {
		t.Fatal("office.db 不可用")
	}
	defer officedb.CloseDatabase(filepath.Join(dir, "office"))
	st := NewStore(db, filepath.Join(dir, "office"))

	n, err := MigrateLegacyJSON(st)
	if err != nil {
		t.Fatalf("MigrateLegacyJSON: %v", err)
	}
	if n != 1 {
		t.Fatalf("migrated = %d, want 1", n)
	}
	p, err := st.Get("legacy-1")
	if err != nil {
		t.Fatalf("Get migrated: %v", err)
	}
	if p.Title != "旧方案" || p.ProjectID != "default" || p.Version != 1 {
		t.Fatalf("migrated proposal 异常: %+v", p)
	}
	if len(p.Sections) != 1 || p.Sections[0].Content != "正文" {
		t.Fatalf("migrated sections 异常: %+v", p.Sections)
	}
	var fn int
	if err := db.QueryRow("SELECT COUNT(*) FROM files WHERE proposal_id = ?", "legacy-1").Scan(&fn); err != nil {
		t.Fatal(err)
	}
	if fn != 1 {
		t.Errorf("files rows = %d, want 1", fn)
	}

	// 幂等：再次迁移返回 0
	n2, err := MigrateLegacyJSON(st)
	if err != nil || n2 != 0 {
		t.Fatalf("second migrate = %d, %v; want 0, nil", n2, err)
	}
}
