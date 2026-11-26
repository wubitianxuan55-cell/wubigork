package xlsxedit

// recalc_smoke_sample_test.go — 内置测算样表的实弹重算钉，默认跳过：
//
//	GAEA_SMOKE_RECALC=1 go test ./internal/office/xlsxedit -run TestSmokeRecalcCostSample -v
//
// 样表是版式真源（随程序内嵌），公式改坏了要到模型填完表才发现。这里拿真
// LibreOffice 把它重算一遍：零错误值、零自引用、零未重算；剩下的只能是未用槽位
// 上「结果为空串」的取价公式（empty_result，不阻塞）。

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSmokeRecalcCostSample(t *testing.T) {
	smokeRecalcReady(t)
	src := filepath.Join("..", "..", "gaea", "skill", "assets", "cost-estimate-sample.xlsx")
	raw, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("读不到内置样表 %s: %v", src, err)
	}
	path := filepath.Join(t.TempDir(), "sample.xlsx")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := Recalc(path)
	if err != nil {
		t.Fatalf("样表重算失败：%v", err)
	}
	if rep.TotalErrors != 0 {
		t.Fatalf("样表重算不应有公式错误：%+v", rep)
	}
	audit, err := Audit(path)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]int{}
	for _, f := range audit.Findings {
		kinds[f.Kind]++
	}
	t.Logf("公式 %d 个；重算后审计发现按类：%v", audit.TotalFormulas, kinds)
	if kinds["error_value"] != 0 || kinds["self_ref"] != 0 || kinds["missing_cache"] != 0 {
		t.Fatalf("样表重算后不该有错误值 / 自引用 / 未重算：%+v", audit.Findings)
	}
}
