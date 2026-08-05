package builtin

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	officedb "github.com/gaea/gaea/internal/office/db"
	"github.com/gaea/gaea/internal/office/proposal"
)

func TestProposalTools_ListReadWrite(t *testing.T) {
	dir := t.TempDir()
	svc := proposal.NewService(dir, nil)
	t.Cleanup(func() { _ = officedb.CloseDatabase(filepath.Join(dir, "office")) })
	proposal.SetGlobalServiceForTest(svc)
	t.Cleanup(proposal.ResetGlobalServiceForTest)

	proj, err := svc.CreateProject("项目", "环保工程", "客户")
	if err != nil {
		t.Fatal(err)
	}
	p, err := svc.Create("某方案", "blank", "需求", "环保工程", proj.ID)
	if err != nil {
		t.Fatal(err)
	}
	list, err := (proposalList{}).Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil || !strings.Contains(list, p.Title) {
		t.Fatalf("proposal_list: %v %s", err, list)
	}
	out, err := (proposalWrite{}).Execute(context.Background(), json.RawMessage(`{"proposal_id":"`+p.ID+`","section_id":"","content":"新需求内容"}`))
	if err != nil || !strings.Contains(out, "成功") {
		t.Fatalf("proposal_write: %v %s", err, out)
	}
	got, err := svc.Get(p.ID)
	if err != nil || got.Requirements != "新需求内容" {
		t.Fatalf("写入需求未生效: %v %+v", err, got)
	}
	export, err := (proposalExport{}).Execute(context.Background(), json.RawMessage(`{"proposal_id":"`+p.ID+`"}`))
	if err != nil || !strings.Contains(export, ".md") {
		t.Fatalf("proposal_export: %v %s", err, export)
	}
}
