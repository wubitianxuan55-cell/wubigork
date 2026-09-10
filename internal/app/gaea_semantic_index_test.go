package app

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/cost"
)

// costEntryForIndexTest 同名条目改价 → DocText 变化 → Coverage 判未覆盖。
func costEntryForIndexTest() cost.Entry {
	return cost.Entry{Name: "pile-rent", Title: "振动锤租赁台班", Category: "机械", Unit: "台班", Price: 7200}
}

// TestSemanticIndexStatusAndBackfill 索引显形+显式补齐闭环：
// 空索引状态诚实（indexed=0）→ 补齐后全覆盖 → 二次补齐幂等零重嵌 →
// 状态与 Coverage 同口径（正文变了算未覆盖）。
func TestSemanticIndexStatusAndBackfill(t *testing.T) {
	a, _, _ := countingEmbedEnv(t)
	seedSemanticData(t, a) // 1 条成本条目（振动锤租赁台班）

	// 补齐前：覆盖为零。
	st := a.GaeaSemanticIndexStatus()
	if st.CostTotal != 1 || st.CostIndexed != 0 || !st.Available {
		t.Fatalf("初始状态异常: %+v", st)
	}

	// 首次补齐：1 条全部向量化。
	bf, err := a.GaeaSemanticIndexBackfill()
	if err != nil {
		t.Fatal(err)
	}
	if bf["total"].(int) != 1 || bf["updated"].(int) != 1 || bf["indexed"].(int) != 1 || bf["missing"].(int) != 0 {
		t.Fatalf("首次补齐异常: %+v", bf)
	}

	// 状态随后全覆盖。
	st2 := a.GaeaSemanticIndexStatus()
	if st2.CostIndexed != 1 {
		t.Fatalf("补齐后状态应全覆盖: %+v", st2)
	}

	// 二次补齐幂等：无重嵌。
	bf2, err := a.GaeaSemanticIndexBackfill()
	if err != nil {
		t.Fatal(err)
	}
	if bf2["updated"].(int) != 0 {
		t.Fatalf("二次补齐应零重嵌: %+v", bf2)
	}

	// 正文变化 → Coverage 判未覆盖（与 Ensure 同口径），补齐后恢复。
	if err := a.hubCostStore().Save(costEntryForIndexTest()); err != nil {
		t.Fatal(err)
	}
	st3 := a.GaeaSemanticIndexStatus()
	if st3.CostIndexed != 0 || st3.CostTotal != 1 {
		t.Fatalf("正文变更应判未覆盖: %+v", st3)
	}
	if _, err := a.GaeaSemanticIndexBackfill(); err != nil {
		t.Fatal(err)
	}
	st4 := a.GaeaSemanticIndexStatus()
	if st4.CostIndexed != 1 {
		t.Fatalf("重嵌后应恢复覆盖: %+v", st4)
	}
}

// TestSemanticIndexStatus_ModelUnavailable 模型不可用时状态诚实说人话、
// 补齐拒绝而非静默假装成功。
func TestSemanticIndexStatus_ModelUnavailable(t *testing.T) {
	a, _, _ := countingEmbedEnv(t)
	seedSemanticData(t, a)
	SetAppEmbedderForTest(nil) // 引擎未配置 → localSearchEmbedder nil

	st := a.GaeaSemanticIndexStatus()
	if st.Available {
		t.Fatalf("无模型应报 Available=false: %+v", st)
	}
	if !strings.Contains(st.Error, "未配置") {
		t.Fatalf("Error 应说人话: %+v", st)
	}
	if _, err := a.GaeaSemanticIndexBackfill(); err == nil || !strings.Contains(err.Error(), "未配置") {
		t.Fatalf("无模型时补齐应拒绝并说人话: %v", err)
	}
}
