package app

// office_cost_payload_files_test.go — 批量写库载荷走文件的行为契约（2026-11-27
// AI 反馈 #9）：行由脚本生成 JSON，import_add / classify 读文件，数字不经模型转录。
// 全部走隔离库与临时工作区。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/workcost"
)

// clTempWS 给测试一个临时工作区（载荷文件只认工作区内的路径）。
func clTempWS(t *testing.T) string {
	t.Helper()
	ws := t.TempDir()
	origCfg := ga.cfg
	ga.cfg = &config.Config{Workspace: ws}
	t.Cleanup(func() { ga.cfg = origCfg })
	return ws
}

func clWriteWS(t *testing.T, ws, rel, body string) string {
	t.Helper()
	abs := filepath.Join(ws, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return rel
}

// TestCostImportPayloadFiles 三组行各自从一个 JSON 文件提交：对象套键与裸数组都认、
// BOM 不碍事、退回行号按文件顺序计、单批上限放宽；import_status export 能把暂存区
// 导回同格式的 JSON。
func TestCostImportPayloadFiles(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	tl := costLibraryTool{a: a}
	impDict(t, s)
	ws := clTempWS(t)

	resJSON, _ := json.Marshal(map[string]any{"resources": impResources()})
	quotaJSON, _ := json.Marshal(impQuotas())
	billJSON, _ := json.Marshal([]map[string]any{impBill("WP01", 100, 528.6)})
	clWriteWS(t, ws, ".gaea/work/res.json", "\xEF\xBB\xBF"+string(resJSON)) // 带 BOM 的对象
	clWriteWS(t, ws, ".gaea/work/quotas.json", string(quotaJSON))
	clWriteWS(t, ws, ".gaea/work/bills.json", string(billJSON))

	id := impOpen(t, tl, "", nil)
	out := clMust(t, tl, map[string]any{"action": "import_add", "import_id": id, "resources_file": ".gaea/work/res.json"},
		"**本批工料机**: 提交 3 行 → 入暂存 3")
	for _, want := range []string{"**读自文件**: 工料机 3 行读自 .gaea/work/res.json；", "行号按文件里的顺序计"} {
		if !strings.Contains(out, want) {
			t.Errorf("回执应含 %q：%s", want, out)
		}
	}
	// 定额与清单同一个调用各给一个文件。
	clMust(t, tl, map[string]any{"action": "import_add", "import_id": id, "quotas_file": ".gaea/work/quotas.json",
		"bills_file": ".gaea/work/bills.json"}, "**暂存合计**: 工料机 3｜定额 1｜清单 1")

	// 导出暂存区：与载荷文件同格式，提交的都在。
	export := ".gaea/work/staged.json"
	clMust(t, tl, map[string]any{"action": "import_status", "import_id": id, "export": export}, "导出暂存区 .gaea/work/staged.json")
	data, err := os.ReadFile(filepath.Join(ws, filepath.FromSlash(export)))
	if err != nil {
		t.Fatal(err)
	}
	var staged struct {
		Resources []workcost.ImportResource `json:"resources"`
		Quotas    []workcost.ImportQuota    `json:"quotas"`
		Bills     []workcost.ImportBill     `json:"bills"`
	}
	if err := json.Unmarshal(data, &staged); err != nil {
		t.Fatalf("导出不是合法 JSON: %v", err)
	}
	if len(staged.Resources) != 3 || len(staged.Quotas) != 1 || len(staged.Bills) != 1 || staged.Bills[0].Quantity != 100 {
		t.Fatalf("导出应含提交的全部行：%+v", staged)
	}
	if staged.Resources[0].Src == nil || staged.Resources[0].Src.Row != 2 {
		t.Errorf("导出应保留来源位置：%+v", staged.Resources[0])
	}

	// 文件里的行照常过规矩：第 2 行类别不在四类之内 → 退回并点名文件里的行号。
	bad, _ := json.Marshal([]map[string]any{
		{"code": "L02", "kind": "人工", "title": "技工", "unit": "工日", "price": 260},
		{"code": "X01", "kind": "其他", "title": "杂项", "unit": "项", "price": 1},
	})
	clWriteWS(t, ws, ".gaea/work/bad.json", string(bad))
	out = clMust(t, tl, map[string]any{"action": "import_add", "import_id": id, "resources_file": ".gaea/work/bad.json"}, "退回 1")
	if !strings.Contains(out, "第 2 行 X01") {
		t.Errorf("退回行应按文件里的行号点名：%s", out)
	}
}

// TestCostImportPayloadFilesErrors 载荷文件的边界：工作区外、不存在、不是行对象、
// 套错键、字段类型不对（报文件里的行号）；数组与文件同时给时以数组为准。
func TestCostImportPayloadFilesErrors(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	tl := costLibraryTool{a: a}
	impDict(t, s)
	ws := clTempWS(t)
	id := impOpen(t, tl, "", nil)
	resJSON, _ := json.Marshal(map[string]any{"resources": impResources()})

	clMustFail(t, tl, map[string]any{"action": "import_add", "import_id": id, "resources_file": "../outside.json"}, "必须在工作区内")
	clMustFail(t, tl, map[string]any{"action": "import_add", "import_id": id, "resources_file": ".gaea/work/没有.json"}, "读不到工料机载荷文件")
	clWriteWS(t, ws, ".gaea/work/str.json", `["L01","C20"]`)
	clMustFail(t, tl, map[string]any{"action": "import_add", "import_id": id, "resources_file": ".gaea/work/str.json"}, "第 1 项不是行对象")
	clWriteWS(t, ws, ".gaea/work/wrongkey.json", `{"quotas":[]}`)
	clMustFail(t, tl, map[string]any{"action": "import_add", "import_id": id, "resources_file": ".gaea/work/wrongkey.json"}, "没有 resources 键")
	clWriteWS(t, ws, ".gaea/work/broken.json", `[{ "code": "L01", "price": "一百八" }]`)
	clMustFail(t, tl, map[string]any{"action": "import_add", "import_id": id, "resources_file": ".gaea/work/broken.json"}, "第 1 行")
	// 不是合法 JSON：报路径，不把文件内容打回。
	clWriteWS(t, ws, ".gaea/work/junk.json", `{code: L01}`)
	if _, err := clRun(t, tl, map[string]any{"action": "import_add", "import_id": id, "resources_file": ".gaea/work/junk.json"}); err == nil ||
		!strings.Contains(err.Error(), ".gaea/work/junk.json") {
		t.Fatalf("非法 JSON 应报文件路径，得到：%v", err)
	}

	// 数组与文件同时给：以数组为准（文件那份是坏的也不读）。
	clWriteWS(t, ws, ".gaea/work/res.json", string(resJSON))
	clMust(t, tl, map[string]any{"action": "import_add", "import_id": id, "resources": impResources(), "resources_file": ".gaea/work/junk.json"},
		"入暂存 3")
	// 没给行也没给文件：照旧报缺参。
	clMustFail(t, tl, map[string]any{"action": "import_add", "import_id": id}, "至少一组")
}

// TestCostImportPayloadFileLimit 文件读来的行不卡 200 行的手写上限（放宽到 2000）。
func TestCostImportPayloadFileLimit(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	tl := costLibraryTool{a: a}
	impDict(t, s)
	ws := clTempWS(t)
	id := impOpen(t, tl, "", nil)

	rows := make([]map[string]any, 0, 260)
	for i := 0; i < 260; i++ {
		rows = append(rows, map[string]any{"code": fmt.Sprintf("M%03d", i), "kind": "材料", "title": fmt.Sprintf("材料%03d", i), "unit": "kg", "price": 3.5})
	}
	rowsJSON, _ := json.Marshal(rows)
	clWriteWS(t, ws, ".gaea/work/many.json", string(rowsJSON))
	clMust(t, tl, map[string]any{"action": "import_add", "import_id": id, "resources_file": ".gaea/work/many.json"}, "提交 260 行 → 入暂存 260")
	// 同样的行数走数组：仍按 200 卡，并指路文件。
	_, err := clRun(t, tl, map[string]any{"action": "import_add", "import_id": id, "resources": rows})
	if err == nil || !strings.Contains(err.Error(), "一批最多 200 行") || !strings.Contains(err.Error(), "resources_file=") {
		t.Fatalf("手写数组超 200 行应被拒并指路文件，得到：%v", err)
	}
	// 文件超 2000 行：读文件时就拒。
	tooMany := make([]map[string]any, 0, clFileMaxRows+1)
	for i := 0; i <= clFileMaxRows; i++ {
		tooMany = append(tooMany, map[string]any{"code": fmt.Sprintf("N%04d", i), "kind": "材料", "title": "材料", "unit": "kg"})
	}
	tooJSON, _ := json.Marshal(tooMany)
	clWriteWS(t, ws, ".gaea/work/toomany.json", string(tooJSON))
	clMustFail(t, tl, map[string]any{"action": "import_add", "import_id": id, "resources_file": ".gaea/work/toomany.json"}, "超过单文件上限 2000 行")
}

// TestCostClassifyPayloadFile classify 的四组从 JSON 对象文件提交；dims 回执带
// classify_file 契约；套错键、给了对象则以对象为准。
func TestCostClassifyPayloadFile(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	tl := costLibraryTool{a: a}
	_, item := ruleSeed(t, s)
	ws := clTempWS(t)

	out := clMust(t, tl, map[string]any{"action": "dims"}, "classify_file")
	if !strings.Contains(out, "classify_file string") {
		t.Errorf("dims 回执的 classify 契约应列出 classify_file：%s", out)
	}
	plan, _ := json.Marshal(map[string]any{
		"quotas": []map[string]any{{"code": "A", "chapter": "混凝土工程"}},
		"bills":  []map[string]any{{"id": item.ID, "division": "垫层工程"}},
	})
	clWriteWS(t, ws, ".gaea/work/plan.json", string(plan))
	out = clMust(t, tl, map[string]any{"action": "classify", "dry_run": true, "classify_file": ".gaea/work/plan.json"}, "将改动**: 项目 0｜清单 1｜定额 1｜工料机 0")
	if !strings.Contains(out, "**读自文件**: classify 读自 .gaea/work/plan.json；") {
		t.Errorf("classify 回执应点名读自文件：%s", out)
	}
	if q, _ := s.GetQuota("A"); q.Chapter != "" {
		t.Fatal("演练不应写定额分部")
	}
	// 条目字段类型不对：报组名与行号。
	bad, _ := json.Marshal(map[string]any{"quotas": []map[string]any{{"code": 123}}})
	clWriteWS(t, ws, ".gaea/work/badplan.json", string(bad))
	clMustFail(t, tl, map[string]any{"action": "classify", "dry_run": true, "classify_file": ".gaea/work/badplan.json"}, "quotas：第 1 行")
	// 数组/对象同时给：以 classify 对象为准。
	clMust(t, tl, map[string]any{"action": "classify", "dry_run": true, "classify": map[string]any{"quotas": []map[string]any{{"code": "A", "chapter": "混凝土工程"}}},
		"classify_file": ".gaea/work/badplan.json"}, "定额 1｜工料机 0")
}
