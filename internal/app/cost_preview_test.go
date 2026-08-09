package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSmokeCostPreview 用真实成本测算文件验证 GaeaPreview（默认跳过）：
//   GAEA_SMOKE_COST=<xlsx 路径> go test ./internal/app -run TestSmokeCostPreview -v
func TestSmokeCostPreview(t *testing.T) {
	src := os.Getenv("GAEA_SMOKE_COST")
	if src == "" {
		t.Skip("未设置 GAEA_SMOKE_COST")
	}
	t.Chdir(t.TempDir())
	rel := filepath.Join(".gaea", "uploads", "cost.xlsx")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rel, b, 0o644); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	got := a.GaeaPreview(filepath.ToSlash(rel))
	if got.Kind != "xlsx" {
		t.Fatalf("kind = %q, want xlsx（%s）", got.Kind, got.Error)
	}
	if len(got.Body) < 500 || !containsStr(got.Body, `"sheets"`) || !containsStr(got.Body, `"ref"`) {
		t.Error("预览 JSON 缺少工作表/单元格数据")
	}

	// 回归：公式单元格必须带计算结果（GaeaPreview 应自动补 LibreOffice 重算）
	var pr struct {
		Sheets []struct {
			Name string `json:"name"`
			Rows [][]struct {
				Ref     string `json:"ref"`
				Value   string `json:"value"`
				Formula string `json:"formula,omitempty"`
			} `json:"rows"`
		} `json:"sheets"`
	}
	if err := json.Unmarshal([]byte(got.Body), &pr); err != nil {
		t.Fatalf("预览 JSON 解析失败: %v", err)
	}
	formulaCells := 0
	emptyResults := 0
	for _, sh := range pr.Sheets {
		for _, row := range sh.Rows {
			for _, c := range row {
				if c.Formula == "" {
					continue
				}
				formulaCells++
				if strings.TrimSpace(c.Value) == "" {
					emptyResults++
					t.Logf("公式无结果：%s!%s = %s", sh.Name, c.Ref, c.Formula)
				}
			}
		}
	}
	if formulaCells == 0 {
		t.Error("工作簿中没有公式单元格，回归断言失效")
	}
	if emptyResults > 0 {
		t.Errorf("%d/%d 个公式单元格没有计算结果", emptyResults, formulaCells)
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
