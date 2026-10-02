package cost

// ── GA6-09 错误通道契约用例 ──────────────────────────────────────
//
// 生产代码（cost.go 的 List/Search 改 ([]Summary, error)）已落地，本文件补齐
// 三态契约的钉子（此前只有成功路径被覆盖）：
//   1. db == nil（未配置库）不是错误 → (nil, nil)；
//   2. 完全取不到（Query 失败，此处以关闭底层 db 注入）→ (nil, err)，不 panic；
//   3. 部分行读取失败 → 部分数据 + err（err 说明少了几行），不整体丢弃。
//
// 改坏能红锚点：把 Search 里 Query 失败分支改回 `return nil, nil`（或把
// rows.Scan 失败分支的计数/收口删掉），本文件对应用例即 FAIL。

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

// newCostErrStore 建隔离的临时成本库并返回 store 与清理函数。
func newCostErrStore(t *testing.T) (*Store, func()) {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase 返回 nil")
	}
	return Open(gdb), func() { db.CloseDatabase(dir) }
}

// TestCostStoreNilDBContract 未配置库（db == nil）：List/Search 都不是错误。
// 语义见 cost.go 的 List/Search 注释——「没配库」是正常态，不是读取失败，
// 若这里报错，前端会把「未配置」渲染成「成本库读取失败」。
func TestCostStoreNilDBContract(t *testing.T) {
	s := Open(nil)
	if s.Available() {
		t.Fatal("db == nil 时 Available() 应为 false")
	}
	got, err := s.Search("水泥", "", "")
	if got != nil || err != nil {
		t.Fatalf("Search(db==nil) = (%v, %v), want (nil, nil)", got, err)
	}
	got, err = s.List()
	if got != nil || err != nil {
		t.Fatalf("List(db==nil) = (%v, %v), want (nil, nil)", got, err)
	}
}

// TestCostSearchQueryFailureReturnsError Query 本身失败（底层 db 已关闭）：
// Search/List 都要 (nil, err) 且不 panic——这是「完全取不到」态，
// 界面必须据此上报，而不是把结果渲染成「无匹配条目」。
func TestCostSearchQueryFailureReturnsError(t *testing.T) {
	s, cleanup := newCostErrStore(t)
	defer cleanup()
	if err := s.Save(Entry{Name: "cement", Title: "P.O42.5 水泥", Price: 480, Status: "现行"}); err != nil {
		t.Fatalf("种子保存: %v", err)
	}

	// 注入读失败：关闭底层连接池——此后任何 Query 都返回 sql.ErrConnDone/database is closed。
	if err := s.DB().Close(); err != nil {
		t.Fatalf("关闭底层库: %v", err)
	}

	// 不 panic 是契约的一部分：用子测试显式声明，失败即 Fatal 而非 panic 冒泡。
	t.Run("Search", func(t *testing.T) {
		got, err := s.Search("水泥", "", "")
		if err == nil {
			t.Fatalf("Search 在 db 关闭后应返回 error，实际 (rows=%d, err=nil)", len(got))
		}
		if got != nil {
			t.Fatalf("完全取不到时应返回 nil 数据，got %+v", got)
		}
	})
	t.Run("List", func(t *testing.T) {
		got, err := s.List()
		if err == nil {
			t.Fatalf("List 在 db 关闭后应返回 error，实际 (rows=%d, err=nil)", len(got))
		}
		if got != nil {
			t.Fatalf("完全取不到时应返回 nil 数据，got %+v", got)
		}
	})
}

// TestCostSearchPartialScanFailureReturnsPartialData 部分行解析失败：
// 坏行跳过并计数，好行照常返回，err 说明「少了几行」。
//
// 注入手段：把某行 price 列写成非数值文本（SQLite 动态类型允许），
// scan 到 float64 即失败——但这只是「一行坏」，Query 与迭代都不失败，
// 因此结果必须包含其余全部好行。改坏能红：把 Scan 失败分支改回静默
// continue（不计数不收口）→ 本用例的 err != nil 断言即红。
func TestCostSearchPartialScanFailureReturnsPartialData(t *testing.T) {
	s, cleanup := newCostErrStore(t)
	defer cleanup()
	for _, name := range []string{"a-good", "b-bad", "c-good"} {
		if err := s.Save(Entry{Name: name, Title: name, Price: 1, Status: "现行"}); err != nil {
			t.Fatalf("种子 %s: %v", name, err)
		}
	}
	// 坏行注入（直写绕过 Save 的类型约束）：price 列存文本。
	if _, err := s.DB().Exec("UPDATE cost_entries SET price = 'not-a-number' WHERE name = 'b-bad'"); err != nil {
		t.Fatalf("注入坏行: %v", err)
	}

	got, err := s.Search("", "", "")
	if err == nil {
		t.Fatal("有行解析失败时必须返回 error（说明结果少了几行），实际 err=nil")
	}
	names := map[string]bool{}
	for _, e := range got {
		names[e.Name] = true
	}
	if !names["a-good"] || !names["c-good"] {
		t.Fatalf("坏行不应拖垮整体：好行 a-good/c-good 都必须在结果里，got %+v", names)
	}
	if names["b-bad"] {
		t.Fatalf("坏行不应出现在结果里: %+v", names)
	}
	if len(got) != 2 {
		t.Fatalf("部分数据应为 2 条（3 条中 1 条坏），got %d", len(got))
	}

	// List 与 Search 同一契约（List 就是 Search("", "", "")）。
	listGot, listErr := s.List()
	if listErr == nil || len(listGot) != 2 {
		t.Fatalf("List 应与 Search 同契约：got (%d 条, err=%v)", len(listGot), listErr)
	}
}

// TestCostSearchSuccessReturnsNoError 成功路径：err 恒为 nil（错误通道引入后
// 不得给正常读取平白加噪声）。同时确认 List 与 Search 结果一致。
func TestCostSearchSuccessReturnsNoError(t *testing.T) {
	s, cleanup := newCostErrStore(t)
	defer cleanup()
	if err := s.Save(Entry{Name: "hp300", Title: "HP300 液压振动锤", Category: "机械", Unit: "台班",
		Price: 3200, Spec: "300kW", Source: "市场询价", Region: "成都市区", Status: "现行"}); err != nil {
		t.Fatalf("保存: %v", err)
	}

	all, err := s.Search("", "", "")
	if err != nil {
		t.Fatalf("成功路径 Search err = %v, want nil", err)
	}
	if len(all) != 1 || all[0].Name != "hp300" {
		t.Fatalf("Search 结果 = %+v", all)
	}
	hit, err := s.Search("振动锤", "机械", "现行")
	if err != nil {
		t.Fatalf("成功路径带过滤 Search err = %v, want nil", err)
	}
	if len(hit) != 1 || hit[0].Name != "hp300" {
		t.Fatalf("带过滤 Search 结果 = %+v", hit)
	}
	listed, err := s.List()
	if err != nil {
		t.Fatalf("成功路径 List err = %v, want nil", err)
	}
	if len(listed) != len(all) || listed[0].Name != all[0].Name {
		t.Fatalf("List 与 Search 不一致: %+v vs %+v", listed, all)
	}
}
