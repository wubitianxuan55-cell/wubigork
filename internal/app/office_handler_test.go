package app

import (
	"path/filepath"
	"testing"

	officedb "github.com/gaea/gaea/internal/office/db"
	"github.com/gaea/gaea/internal/office/proposal"
)

func TestOfficeProjectBindings(t *testing.T) {
	dir := t.TempDir()
	st := &officeState{proposalSvc: proposal.NewService(dir, nil)}
	t.Cleanup(func() { _ = officedb.CloseDatabase(filepath.Join(dir, "office")) })

	proj, err := st.ProposalProjectCreate("某修复项目", "环保工程", "某公司")
	if err != nil {
		t.Fatalf("ProposalProjectCreate: %v", err)
	}
	if proj["name"] != "某修复项目" || proj["category"] != "环保工程" {
		t.Fatalf("创建结果异常: %+v", proj)
	}
	pid, _ := proj["id"].(string)

	list := st.ProposalProjectList()
	if len(list) != 2 { // default + 新建
		t.Fatalf("ProposalProjectList len = %d, want 2", len(list))
	}

	created, err := st.ProposalCreate("方案", "soil-remediation-bid", "需求", "环保工程", pid)
	if err != nil {
		t.Fatalf("ProposalCreate: %v", err)
	}
	if created["projectId"] != pid {
		t.Errorf("projectId = %v, want %s", created["projectId"], pid)
	}
	if created["version"] != 1 {
		t.Errorf("version = %v, want 1", created["version"])
	}
	// sections 应带 sources 字段
	secs, _ := created["sections"].([]map[string]interface{})
	if len(secs) > 0 {
		if _, ok := secs[0]["sources"]; !ok {
			t.Error("sections[0] 缺少 sources 字段")
		}
	}

	if err := st.ProposalProjectDelete(pid); err != nil {
		t.Fatalf("ProposalProjectDelete: %v", err)
	}
	remaining := st.ProposalProjectList()
	if len(remaining) != 1 {
		t.Errorf("删除后项目数 = %d, want 1", len(remaining))
	}
}

func TestBidSummaryRoundTripKeepsNewFields(t *testing.T) {
	bs := &proposal.BidSummary{
		Overview:      "项目概况",
		Qualification: []proposal.BidItem{{Name: "资质", Content: "三级", Sources: []proposal.SourceRef{{Snippet: "原文"}}}},
		ParseStatus:   "done",
	}
	m := btm(bs)
	got := bsf(m)
	if got.ParseStatus != "done" || len(got.Qualification) != 1 || got.Qualification[0].Sources[0].Snippet != "原文" {
		t.Fatalf("往返丢失新字段: %+v", got)
	}
}
