package workcost

import (
	"strings"
	"testing"
)

// ── 核算缓存化（cost_entries.price 降级为核算结果）──────────────────

// fakeCostLib 内存版成本库，供注入缝测试（无需 db，也不 import cost 包）。
type fakeCostLib struct {
	entries map[string]*RecomposeEntry
	derived map[string]bool
	// written 记录写回的载荷（RecomposeEntry 不含价格字段——它是「读取视图」，
	// 价格只存在于写回载荷里）。
	written map[string]RecomposeWrite
}

func newFakeCostLib(list ...*RecomposeEntry) *fakeCostLib {
	f := &fakeCostLib{
		entries: map[string]*RecomposeEntry{},
		derived: map[string]bool{},
		written: map[string]RecomposeWrite{},
	}
	for _, e := range list {
		f.entries[e.Name] = e
	}
	return f
}

func (f *fakeCostLib) hooks() RecomposeHooks {
	return RecomposeHooks{
		Names: func() ([]string, error) {
			out := make([]string, 0, len(f.entries))
			for k := range f.entries {
				out = append(out, k)
			}
			return out, nil
		},
		Load: func(name string) (*RecomposeEntry, error) {
			e, ok := f.entries[name]
			if !ok {
				return nil, nil
			}
			cp := *e
			return &cp, nil
		},
		Write: func(w RecomposeWrite) error {
			f.derived[w.Name] = true
			f.written[w.Name] = w
			return nil
		},
	}
}

// TestRecomposeSkipsEntriesWithoutComponents 纯资源价条目不参与重算——
// 重算会把它抹成 0（把「查到的价」降级成「算不出的价」）。
func TestRecomposeSkipsEntriesWithoutComponents(t *testing.T) {
	lib := newFakeCostLib(
		&RecomposeEntry{Name: "cement", Title: "P.O 42.5 水泥", Unit: "t"},
		&RecomposeEntry{Name: "pipe", Title: "PE给水管", Unit: "m"},
	)
	res, err := RecomposeEntries(lib.hooks())
	if err != nil {
		t.Fatalf("重算失败: %v", err)
	}
	if res.Scanned != 2 || res.WithComp != 0 || res.Updated != 0 {
		t.Fatalf("无组成条目应全部跳过：%+v", res)
	}
	if len(lib.written) != 0 {
		t.Fatal("无组成条目不应产生任何写入")
	}
}

// TestRecomposeComputesAndWrites 有组成的条目：重算 Σ(含量×单价) 并写回。
func TestRecomposeComputesAndWrites(t *testing.T) {
	lib := newFakeCostLib(&RecomposeEntry{
		Name: "ac13", Title: "细粒式沥青混凝土 AC-13", Unit: "t", CategoryPath: "综合单价/道路工程",
		Components: []RecomposeComponent{
			{Kind: "人工", Title: "普通工", Unit: "工日", Quantity: 0.048, Price: 300},
			{Kind: "材料", Title: "改性沥青", Unit: "t", Quantity: 1, Price: 4200},
			{Kind: "机械", Title: "摊铺机", Unit: "台班", Quantity: 0.0018, Price: 4200},
		},
	})
	res, err := RecomposeEntries(lib.hooks())
	if err != nil {
		t.Fatalf("重算失败: %v", err)
	}
	if res.WithComp != 1 || res.Updated != 1 {
		t.Fatalf("应有 1 条重算：%+v", res)
	}
	if !lib.derived["ac13"] {
		t.Fatal("写回必须置 price_derived 标记（由注入实现承担）")
	}
	w := lib.written["ac13"]
	closeTo(t, "重算单价", w.Price, 0.048*300+4200+0.0018*4200)
	closeTo(t, "人工费", w.LaborFee, 14.4)
	closeTo(t, "材料费", w.MaterialFee, 4200)
	closeTo(t, "机械费", w.MachineFee, 7.56)
}

// TestRecomposeZeroCompositionSkippedNotZeroed 组成核算为 0 时跳过并报错，
// 绝不把条目价改写成 0。
func TestRecomposeZeroCompositionSkippedNotZeroed(t *testing.T) {
	lib := newFakeCostLib(&RecomposeEntry{
		Name: "bad", Title: "组成缺价的条目", Unit: "项",
		Components: []RecomposeComponent{
			{Kind: "材料", Title: "无价材料", Unit: "t", Quantity: 1, Price: 0},
		},
	})
	res, err := RecomposeEntries(lib.hooks())
	if err != nil {
		t.Fatalf("重算失败: %v", err)
	}
	if res.Updated != 0 || len(res.Errors) != 1 {
		t.Fatalf("应跳过并报错：%+v", res)
	}
	if !strings.Contains(res.Errors[0], "跳过不改价") {
		t.Errorf("错误文案应说明未改价，得到 %q", res.Errors[0])
	}
	if len(lib.written) != 0 {
		t.Fatal("核算为 0 时不得写回")
	}
}

// TestRecomposeHooksRequired 缺注入实现必须显式报错，不得静默 no-op。
func TestRecomposeHooksRequired(t *testing.T) {
	if _, err := RecomposeEntries(RecomposeHooks{}); err == nil {
		t.Fatal("缺注入实现应报错")
	}
}

// TestResolveComponentKind 组成行类别判定：已标注的类别优先于关键词推测。
func TestResolveComponentKind(t *testing.T) {
	// ① 已标注为机械的行，即使标题含「材料」也保持机械。
	if got := ResolveComponentKind("机械", "斗齿材料", "t", ""); got != KindMachine {
		t.Errorf("已标注类别应优先，得到 %q", got)
	}
	// ② 未标注时按标题关键词。
	if got := ResolveComponentKind("", "挖掘机台班", "", ""); got != KindMachine {
		t.Errorf("应按标题识别机械，得到 %q", got)
	}
	// ③ 外委优先于其他关键词。
	if got := ResolveComponentKind("", "水泥窑协同处置", "", ""); got != KindOutsourced {
		t.Errorf("水泥窑处置应归外委，得到 %q", got)
	}
	// ④ 标题无线索时按条目单位暗示（工日→人工）。
	if got := ResolveComponentKind("", "综合用工", "工日", ""); got != KindLabor {
		t.Errorf("单位暗示工日应归人工，得到 %q", got)
	}
	// ⑤ 全无线索兜底材料。
	if got := ResolveComponentKind("", "某种东西", "t", ""); got != KindMaterial {
		t.Errorf("兜底应为材料，得到 %q", got)
	}
}

// TestComponentsToLines 组成行转换：保留含量/单价，类别按判定写入。
func TestComponentsToLines(t *testing.T) {
	lines := ComponentsToLines("t", "综合单价/道路工程", []RecomposeComponent{
		{Kind: "", Title: "挖掘机台班", Unit: "台班", Quantity: 2, Price: 1000},
		{Kind: "材料", Title: "沥青", Unit: "t", Quantity: 1, Price: 4000},
	})
	if len(lines) != 2 {
		t.Fatalf("应转 2 行，得到 %d", len(lines))
	}
	if lines[0].Kind != KindMachine {
		t.Errorf("第 1 行类别 = %q, want 机械", lines[0].Kind)
	}
	closeTo(t, "第 1 行含量", lines[0].Quantity, 2)
	closeTo(t, "第 2 行单价", lines[1].Price, 4000)
}
