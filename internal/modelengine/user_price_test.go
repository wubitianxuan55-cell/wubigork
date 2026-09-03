package modelengine

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── 价目 v1：自定义引擎用户价目（user_price.go + stats.go 折算消费）────────

// resetUserPriceTable 注册表清理：包级注册表是全局态（estimatePrice 消费方
// 众多），每个价目用例退出时必须还原为空表，防止泄漏影响同包其他用例。
func resetUserPriceTable(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { replaceUserPriceTable(map[string]modelPrice{}) })
}

// pricePtr 构造价目指针（表驱动/断言辅助）。
func pricePtr(v float64) *float64 { return &v }

// assertPrice 断言 modelPrice 字段（±0.00001 惯例容差）。
func assertPrice(t *testing.T, tag string, got, want modelPrice) {
	t.Helper()
	if math.Abs(got.InputPerM-want.InputPerM) > 0.00001 ||
		math.Abs(got.OutputPerM-want.OutputPerM) > 0.00001 ||
		got.Currency != want.Currency || got.Unit != want.Unit {
		t.Fatalf("%s: got %+v, want %+v", tag, got, want)
	}
}

// TestUserPrice_NoPriceUnchanged 无价目时行为与现状完全一致（回归锁）：
// custom 引擎模型名撞内置前缀 → 沿用内置表（gpt-5 官方 USD 价）；模型名
// 不在内置表 → 恒不计价；注册表无条目 → userEnginePrice miss。
func TestUserPrice_NoPriceUnchanged(t *testing.T) {
	resetUserPriceTable(t)

	assertPrice(t, "内置前缀匹配应保持现状", estimatePrice("custom-x", "gpt-5"), modelPrice{1.25, 10, "USD", ""})
	assertPrice(t, "未知模型应不计价", estimatePrice("custom-x", "some-relay-only-model"), modelPrice{})
	if _, ok := userEnginePrice("custom-x"); ok {
		t.Fatal("空注册表不应命中用户价目")
	}
}

// TestUserPrice_OverrideAndCost 用户价目优先 + 折算正确性：
// 用户价（CNY 2/8 每百万）覆盖内置表对 gpt-5 前缀的官方价猜测；
// estimatedCostFor = in*2/1e6 + out*8/1e6；EstimateCostCNY 直用不折汇率。
func TestUserPrice_OverrideAndCost(t *testing.T) {
	resetUserPriceTable(t)
	m := NewManager("", "")
	id, err := m.AddCustomEngine("Relay", "https://relay.example.com/v1", "")
	if err != nil {
		t.Fatalf("AddCustomEngine: %v", err)
	}
	if err := m.SaveEngine(EngineConfig{ID: id, Enabled: true, UserPriceIn: pricePtr(2), UserPriceOut: pricePtr(8)}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}

	assertPrice(t, "用户价应覆盖内置前缀猜测", estimatePrice(id, "gpt-5"), modelPrice{2, 8, "CNY", ""})

	// 2*1_000_000/1e6 + 8*500_000/1e6 = 2 + 4 = 6 CNY
	cost, cur := estimatedCostFor(id, "relay-model", 1_000_000, 500_000)
	if math.Abs(cost-6) > 0.00001 || cur != "CNY" {
		t.Fatalf("estimatedCostFor = (%v, %q), want (6, CNY)", cost, cur)
	}
	// CNY 计价直用：非法汇率（0）也不影响结果（不触发 7.2 回显折算）
	if got := EstimateCostCNY(id, "relay-model", 1_000_000, 500_000, 0); math.Abs(got-6) > 0.00001 {
		t.Fatalf("EstimateCostCNY = %v, want 6（CNY 直用）", got)
	}

	// 单边价目：输入显式清零（0=清除）、只留输出价 → 输入按 0 计（注意
	// nil=不修改，想清掉输入价必须显式传 0，与三态语义一致）
	if err := m.SaveEngine(EngineConfig{ID: id, Enabled: true, UserPriceIn: pricePtr(0), UserPriceOut: pricePtr(8)}); err != nil {
		t.Fatalf("SaveEngine(仅输出价): %v", err)
	}
	cost, _ = estimatedCostFor(id, "relay-model", 1_000_000, 500_000)
	if math.Abs(cost-4) > 0.00001 {
		t.Fatalf("仅输出价: cost = %v, want 4", cost)
	}
}

// TestUserPrice_SaveEngineThreeState SaveEngine 价目三态合并语义：
// 数字=设置；局部保存（nil 指针）=不修改；0/负数=清除（清洗归一）。
func TestUserPrice_SaveEngineThreeState(t *testing.T) {
	resetUserPriceTable(t)
	m := NewManager("", "")
	id, err := m.AddCustomEngine("Relay", "https://relay.example.com/v1", "")
	if err != nil {
		t.Fatalf("AddCustomEngine: %v", err)
	}
	get := func() (float64, float64, bool) {
		eng, ok := m.GetEngine(id)
		if !ok {
			t.Fatal("GetEngine 找不到引擎")
		}
		in, out := 0.0, 0.0
		if eng.UserPriceIn != nil {
			in = *eng.UserPriceIn
		}
		if eng.UserPriceOut != nil {
			out = *eng.UserPriceOut
		}
		return in, out, eng.UserPriceIn != nil || eng.UserPriceOut != nil
	}

	// 设置
	if err := m.SaveEngine(EngineConfig{ID: id, Enabled: true, UserPriceIn: pricePtr(2), UserPriceOut: pricePtr(8)}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	if in, out, has := get(); !has || in != 2 || out != 8 {
		t.Fatalf("设置后 = (%v,%v,%v), want (2,8,true)", in, out, has)
	}

	// 局部保存（地址框/启停路径只带 id/base_url/enabled）：价目不得被误清除
	if err := m.SaveEngine(EngineConfig{ID: id, BaseURL: "https://relay.example.com/v2", Enabled: true}); err != nil {
		t.Fatalf("SaveEngine(局部): %v", err)
	}
	if in, out, has := get(); !has || in != 2 || out != 8 {
		t.Fatalf("局部保存后 = (%v,%v,%v), want 价目保持 (2,8,true)", in, out, has)
	}

	// 显式 0 = 清除
	if err := m.SaveEngine(EngineConfig{ID: id, Enabled: true, UserPriceIn: pricePtr(0), UserPriceOut: pricePtr(0)}); err != nil {
		t.Fatalf("SaveEngine(0): %v", err)
	}
	if _, _, has := get(); has {
		t.Fatal("显式 0 应清除价目")
	}
	// 清除后的局部保存保持已清状态
	if err := m.SaveEngine(EngineConfig{ID: id, Enabled: true}); err != nil {
		t.Fatalf("SaveEngine(局部): %v", err)
	}
	if _, _, has := get(); has {
		t.Fatal("清除后局部保存应保持已清")
	}

	// 负数 = 异常输入，清洗为清除（不计价）
	if err := m.SaveEngine(EngineConfig{ID: id, Enabled: true, UserPriceIn: pricePtr(-1)}); err != nil {
		t.Fatalf("SaveEngine(负数): %v", err)
	}
	if eng, _ := m.GetEngine(id); eng.UserPriceIn != nil {
		t.Fatalf("负数价目应清洗为 nil, got %v", *eng.UserPriceIn)
	}
}

// TestUserPrice_LoadStateRestore 持久化往返：价目随 engines.json 落盘（omitempty
// 零价目零字节）→ 新 Manager LoadState 恢复字段并重建注册表（折算即刻生效）；
// 手改文件塞负数脏值 → 清洗为 nil。
func TestUserPrice_LoadStateRestore(t *testing.T) {
	resetUserPriceTable(t)
	path := filepath.Join(t.TempDir(), "engines.json")

	m := NewManager("", "")
	if err := m.LoadState(path); err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	id, err := m.AddCustomEngine("Relay", "https://relay.example.com/v1", "")
	if err != nil {
		t.Fatalf("AddCustomEngine: %v", err)
	}
	if err := m.SaveEngine(EngineConfig{ID: id, Enabled: true, UserPriceIn: pricePtr(2), UserPriceOut: pricePtr(8)}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读状态文件: %v", err)
	}
	if !strings.Contains(string(data), "user_price_in") {
		t.Fatal("价目应随状态文件落盘")
	}

	m2 := NewManager("", "")
	if err := m2.LoadState(path); err != nil {
		t.Fatalf("LoadState(恢复): %v", err)
	}
	eng, ok := m2.GetEngine(id)
	if !ok {
		t.Fatal("重启后应恢复自定义引擎")
	}
	if eng.UserPriceIn == nil || eng.UserPriceOut == nil || *eng.UserPriceIn != 2 || *eng.UserPriceOut != 8 {
		t.Fatalf("重启后价目未恢复: %+v", eng)
	}
	// 注册表随 LoadState 重建：费用折算即刻生效（无需再 SaveEngine）
	assertPrice(t, "LoadState 后注册表应生效", estimatePrice(id, "relay-model"), modelPrice{2, 8, "CNY", ""})

	// 脏值清洗：手改状态文件塞负数 → 恢复为 nil（不计价）
	var f stateFile
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatalf("解析状态文件: %v", err)
	}
	e := f.Engines[id]
	e.UserPriceIn = pricePtr(-5)
	f.Engines[id] = e
	dirty, _ := json.Marshal(f)
	if err := os.WriteFile(path, dirty, 0644); err != nil {
		t.Fatalf("写脏文件: %v", err)
	}
	m3 := NewManager("", "")
	if err := m3.LoadState(path); err != nil {
		t.Fatalf("LoadState(脏值): %v", err)
	}
	if eng3, _ := m3.GetEngine(id); eng3.UserPriceIn != nil {
		t.Fatalf("负数脏值应清洗为 nil, got %v", *eng3.UserPriceIn)
	}
}

// TestUserPrice_LocalEnginesStayFree 本地引擎恒不计价：即便注册表存在条目
// （手改/异常路径写入），estimatePrice 的本地短路也先于用户价目消费。
func TestUserPrice_LocalEnginesStayFree(t *testing.T) {
	resetUserPriceTable(t)
	replaceUserPriceTable(map[string]modelPrice{"ollama": {1, 1, "CNY", ""}, "herdsman": {1, 1, "CNY", ""}})
	assertPrice(t, "本地引擎应不计价", estimatePrice("ollama", "qwen3-8b"), modelPrice{})
	assertPrice(t, "herdsman 应不计价", estimatePrice("herdsman", "qwen3-8b"), modelPrice{})
}

// TestUserPrice_RecordAndSummary 端到端：Manager.RecordCall → stats.record →
// summary 的费用折算消费用户价目（PerModel.EstimatedCost/Currency、TotalCost、
// Engines 小计、Trend.Cost 全部按 CNY 直进，无汇率折算）。
func TestUserPrice_RecordAndSummary(t *testing.T) {
	resetUserPriceTable(t)
	m := NewManager("", "")
	id, err := m.AddCustomEngine("Relay", "https://relay.example.com/v1", "")
	if err != nil {
		t.Fatalf("AddCustomEngine: %v", err)
	}
	if err := m.SaveEngine(EngineConfig{ID: id, Enabled: true, UserPriceIn: pricePtr(2), UserPriceOut: pricePtr(8)}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	m.RecordCall(ModelCallUsage{
		EngineID: id, Model: "relay-large",
		InputTokens: 1_000_000, OutputTokens: 500_000, // 2 + 4 = 6 CNY
		Success: true,
	})

	sum := m.GetModelCallStats()
	if len(sum.PerModel) != 1 {
		t.Fatalf("PerModel = %d 条, want 1", len(sum.PerModel))
	}
	pm := sum.PerModel[0]
	if math.Abs(pm.EstimatedCost-6) > 0.00001 || pm.Currency != "CNY" {
		t.Fatalf("PerModel.EstimatedCost/Currency = (%v,%q), want (6,CNY)", pm.EstimatedCost, pm.Currency)
	}
	if math.Abs(sum.TotalCost-6) > 0.00001 {
		t.Fatalf("TotalCost = %v, want 6", sum.TotalCost)
	}
	if eng := sum.Engines[id]; math.Abs(eng.EstimatedCostCNY-6) > 0.00001 {
		t.Fatalf("Engines[%s].EstimatedCostCNY = %v, want 6", id, eng.EstimatedCostCNY)
	}
	if len(sum.Trend) != 1 || math.Abs(sum.Trend[0].Cost-6) > 0.00001 {
		t.Fatalf("Trend.Cost = %v, want 6", sum.Trend)
	}
}
