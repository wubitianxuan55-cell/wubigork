package builtin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/knowledge"
)

func TestSpecQueryReadsKnowledgeBase(t *testing.T) {
	st, err := knowledge.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	knowledge.SetStoreForTest(st)
	t.Cleanup(knowledge.ResetForTest)
	if err := st.Save(knowledge.Entry{
		Name:     "spec-gb36600-1",
		Title:    "GB 36600-2018 表1 建设用地标准",
		Category: knowledge.CatStandard,
		Tags:     []string{"标准"},
		Source:   "GB 36600-2018",
		Body:     "砷(As)筛选值：一类20mg/kg、二类60mg/kg。",
	}); err != nil {
		t.Fatal(err)
	}
	out, err := (specQuery{}).Execute(context.Background(), json.RawMessage(`{"question":"砷 筛选值"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "建设用地标准") {
		t.Fatalf("未命中知识库条文: %s", out)
	}
}
