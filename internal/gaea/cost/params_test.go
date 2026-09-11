package cost

// v4.227 判定参数数据资产测试：内置默认漂移守卫 + 覆盖文件部分字段生效 +
// 非法值拒绝 + 缺失静默 + 恢复内置（全局快照会污染同包后续用例）。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckParamsEmbeddedDefaultsGuard(t *testing.T) {
	p := currentCheckParams()
	if p.AmountAbsFloor != 0.01 || p.AmountRelTol != 0.01 {
		t.Fatalf("R1 默认容差漂移: %+v", p)
	}
	if p.TotalOverRatio != 1.05 || p.TotalUnderRatio != 0.5 {
		t.Fatalf("R3 默认档位漂移: %+v", p)
	}
	if p.MinContentSamples != 3 || p.DegenerateBandTol != 0.05 {
		t.Fatalf("含量对照默认漂移: %+v", p)
	}
}

func TestLoadCheckParamsPartialOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "params.json")
	override := `{"amountRelTol": 0.05, "minContentSamples": 5}`
	if err := os.WriteFile(path, []byte(override), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		// 恢复内置（全局快照会污染同包后续用例）。
		checkParams.Store(mustLoadCheckParams())
	})

	if err := LoadCheckParams(path); err != nil {
		t.Fatalf("LoadCheckParams: %v", err)
	}
	p := currentCheckParams()
	// 覆盖字段生效。
	if p.AmountRelTol != 0.05 || p.MinContentSamples != 5 {
		t.Fatalf("部分覆盖未生效: %+v", p)
	}
	// 未覆盖字段保持内置默认。
	if p.AmountAbsFloor != 0.01 || p.DegenerateBandTol != 0.05 {
		t.Fatalf("未覆盖字段应保默认: %+v", p)
	}

	// 行为翻转 1：R1 相对容差抬到 5% 后，3% 偏差不再 warn。
	// 金额 103 vs 含量×单价 100 → 偏差 3：默认容差 max(0.01,1.03)≈1.03 触发，
	// 覆盖后 max(0.01,5.15)=5.15 不触发。
	if got := CheckComposeComponents([]Component{{Title: "普通工", Unit: "工日", Quantity: 1, Price: 100, Amount: 103}}, 0); len(got) != 0 {
		t.Fatalf("5%% 相对容差下 3%% 偏差不应触发: %+v", got)
	}

	// 行为翻转 2：minContentSamples=5 后，3 例同类池不再对照（静默）。
	comp := Component{Title: "普通工", Unit: "工日", Quantity: 99, Price: 1, Amount: 99}
	pool := [][]Component{
		{{Title: "普通工", Unit: "工日", Quantity: 1, Price: 1, Amount: 1}},
		{{Title: "普通工", Unit: "工日", Quantity: 1, Price: 1, Amount: 1}},
		{{Title: "普通工", Unit: "工日", Quantity: 1, Price: 1, Amount: 1}},
	}
	if got := CheckContentBaseline([]Component{comp}, pool); len(got) != 0 {
		t.Fatalf("样本不足应静默: %+v", got)
	}

	// 非法值拒绝（≤0 不落快照）。
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte(`{"amountRelTol": -1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := LoadCheckParams(bad); err != nil {
		t.Fatalf("≤0 字段应被忽略而非报错: %v", err)
	}
	if got := currentCheckParams().AmountRelTol; got != 0.05 {
		t.Fatalf("非法覆盖不应改动快照: %v", got)
	}

	// 坏 JSON 显式报错（不带病运行）。
	broken := filepath.Join(dir, "broken.json")
	if err := os.WriteFile(broken, []byte(`{oops`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := LoadCheckParams(broken); err == nil || !strings.Contains(err.Error(), "解析失败") {
		t.Fatalf("坏 JSON 应报解析错误: %v", err)
	}

	// 文件不存在 = 静默无覆盖。
	if err := LoadCheckParams(filepath.Join(dir, "missing.json")); err != nil {
		t.Fatalf("缺失覆盖文件应静默: %v", err)
	}
}
