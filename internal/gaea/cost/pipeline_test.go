package cost

// 审计 GA6-04 / GA6-06 配套钉子：
//   1. summaryDocText / RerankDocText 各自的逐字节 golden（文档串改动=移动
//      对应排序，任何触碰必须显式过审）；
//   2. Enhance 统一管线的阈值/编排语义（app 绑定面与 tool 面共用实现）；
//   3. 直写路径（绕过 Store.Save/SaveTx，不推进版本）后检索必须看到新数据
//      ——GA6-06「隐式失效契约」删除后的反向证据：ranker key 含语料指纹，
//      写路径漏掉失效义务也会因指纹变化自动换 key。

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

// docFixture 全字段条目（golden 基准）。
var docFixture = Summary{
	Name: "hp300", Title: "HP300 高频液压振动锤", Code: "A1-12", Category: "机械",
	Unit: "台班", Price: 3200, Spec: "300kW", Source: "市场询价", Region: "成都",
	PriceType: "到场价", PriceDate: "2026-08", Tags: []string{"振动锤", "桩基"},
}

// TestSummaryDocTextGolden 钉死 BM25 语料文档串（改它=移 BM25 排序）。
func TestSummaryDocTextGolden(t *testing.T) {
	got := summaryDocText(docFixture)
	want := "hp300 HP300 高频液压振动锤 A1-12 台班 300kW 市场询价 成都 到场价 2026-08 振动锤 桩基"
	if got != want {
		t.Fatalf("summaryDocText 逐字节变化（会移动 BM25 排序）:\n got %q\nwant %q", got, want)
	}
}

// TestRerankDocTextCanonical 钉死精排文档串唯一实现（GA6-04 单源后的
// 规范串，与改前 app costDocString / builtin costDocText 逐字节一致）。
func TestRerankDocTextCanonical(t *testing.T) {
	got := RerankDocText(docFixture)
	want := "HP300 高频液压振动锤（300kW） 单位台班 单价3200.00元 分类机械 来源市场询价 标签振动锤,桩基"
	if got != want {
		t.Fatalf("RerankDocText 逐字节变化（会移动两消费面的精排分数）:\n got %q\nwant %q", got, want)
	}
	// 可选段省略 + 价格两位小数。
	got2 := RerankDocText(Summary{Name: "x", Title: "水泥", Price: 480.5})
	if want2 := "水泥 单价480.50元"; got2 != want2 {
		t.Fatalf("RerankDocText 最小串:\n got %q\nwant %q", got2, want2)
	}
	// 与 BM25 语料串刻意不同型（同一实体两空间各用各的文本，不许顺手对齐）。
	if RerankDocText(docFixture) == summaryDocText(docFixture) {
		t.Fatal("RerankDocText 与 summaryDocText 不应相同（分属精排/BM25 两个打分空间）")
	}
}

// TestEnhanceStages 统一管线的编排与阈值：补召回触发线（<3）、召回条数
// （10）、精排 limit 透传、fired 语义（tool 面据此截断）、nil 钩子直通。
func TestEnhanceStages(t *testing.T) {
	mk := func(names ...string) []Summary {
		out := make([]Summary, 0, len(names))
		for _, n := range names {
			out = append(out, Summary{Name: n, Title: "t-" + n})
		}
		return out
	}

	t.Run("补召回_不足3条_触发且透传10", func(t *testing.T) {
		var gotHave []Summary
		var gotTopN int
		hooks := SearchHooks{Recall: func(q string, have []Summary, topN int) []Summary {
			if q != "打桩锤" {
				t.Errorf("Recall query = %q", q)
			}
			gotHave = have
			gotTopN = topN
			return mk("sem-1")
		}}
		list, reranked := Enhance("打桩锤", mk("a", "b"), hooks, 20)
		if gotTopN != RecallTopN {
			t.Fatalf("Recall topN = %d, want 10", gotTopN)
		}
		if len(gotHave) != 2 {
			t.Fatalf("Recall have = %v, want 原关键词结果", gotHave)
		}
		if len(list) != 1 || list[0].Name != "sem-1" {
			t.Fatalf("补召回结果未替换列表: %+v", list)
		}
		if reranked {
			t.Fatal("无 Rerank 钩子时 fired 应为 false")
		}
	})

	t.Run("足量命中_不补召回", func(t *testing.T) {
		called := false
		hooks := SearchHooks{Recall: func(string, []Summary, int) []Summary {
			called = true
			return nil
		}}
		list, _ := Enhance("打桩锤", mk("a", "b", "c"), hooks, 20)
		if called {
			t.Fatal("≥3 条命中不应触发补召回")
		}
		if len(list) != 3 {
			t.Fatalf("list 被改动: %+v", list)
		}
	})

	t.Run("空查询_不补召回", func(t *testing.T) {
		called := false
		hooks := SearchHooks{Recall: func(string, []Summary, int) []Summary {
			called = true
			return nil
		}}
		Enhance("   ", mk("a"), hooks, 20)
		if called {
			t.Fatal("纯空白查询不应触发补召回")
		}
	})

	t.Run("补召回为空_保留关键词结果", func(t *testing.T) {
		hooks := SearchHooks{Recall: func(string, []Summary, int) []Summary { return nil }}
		list, _ := Enhance("打桩锤", mk("a", "b"), hooks, 20)
		if len(list) != 2 || list[0].Name != "a" {
			t.Fatalf("空召回应保留原列表: %+v", list)
		}
	})

	t.Run("精排生效_替换且fired", func(t *testing.T) {
		var gotLimit int
		hooks := SearchHooks{Rerank: func(q string, list []Summary, limit int) []Summary {
			gotLimit = limit
			if len(list) != 2 {
				t.Errorf("Rerank 收到 %d 条, want 补召回后的 2 条", len(list))
			}
			return mk("rr")
		}}
		list, reranked := Enhance("q", mk("a", "b"), hooks, 12)
		if gotLimit != 12 {
			t.Fatalf("Rerank limit = %d, want 透传 12", gotLimit)
		}
		if !reranked || len(list) != 1 || list[0].Name != "rr" {
			t.Fatalf("fired=%v list=%+v, want 精排结果替换", reranked, list)
		}
	})

	t.Run("精排为空_保留原列表_fired为false", func(t *testing.T) {
		hooks := SearchHooks{Rerank: func(string, []Summary, int) []Summary { return nil }}
		list, reranked := Enhance("q", mk("a", "b", "c"), hooks, 20)
		if reranked || len(list) != 3 {
			t.Fatalf("fired=%v list=%+v, want 原列表", reranked, list)
		}
	})

	t.Run("nil钩子_直通", func(t *testing.T) {
		in := mk("a", "b")
		list, reranked := Enhance("q", in, SearchHooks{}, 20)
		if reranked || len(list) != 2 {
			t.Fatalf("fired=%v list=%+v, want 原样直通", reranked, list)
		}
	})
}

// TestDirectWriteInvalidatesRankerCache（GA6-06 反向证据·绿面）：
// 绕过 Store.Save/SaveTx 的直写（模拟 app 批量导入或任何未来直写路径）
// 不推进版本，检索仍必须立即看到新数据的排序效果——ranker key 含语料
// 指纹（行数），直写后指纹变化强制重建倒排。
func TestDirectWriteInvalidatesRankerCache(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := Open(gdb)

	mustSave := func(name, title string) {
		t.Helper()
		if err := s.Save(Entry{Name: name, Title: title, Category: "机械", Unit: "台班", Price: 100}); err != nil {
			t.Fatal(err)
		}
	}
	// 两条命中「振动」：a1 词频高（3 次），BM25 下排前。
	mustSave("zzz-a1", "振动锤 振动锤 振动锤")
	mustSave("zzz-a2", "振动筛")

	first, err := s.Search("振动", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 || first[0].Name != "zzz-a1" || first[1].Name != "zzz-a2" {
		t.Fatalf("预热排序 = %+v, want [zzz-a1 zzz-a2]", first)
	}

	// 直写一条词频更高（8 次）的条目：不经 Save/SaveTx，无版本推进。名字
	// 刻意排在语料末尾（zzz-z9）——否则陈旧倒排会按**语料位置**错位记分
	//（旧 doc0 的高分挂到新语料 doc0 头上），掩盖失效语义破坏。
	if _, err := gdb.Exec(`INSERT INTO cost_entries(name, title, category, unit, price, status, created_at, updated_at)
		VALUES('zzz-z9', '振动锤 振动锤 振动锤 振动锤 振动锤 振动锤 振动锤 振动锤', '机械', '台班', 300, '现行', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}

	after, err := s.Search("振动", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 3 {
		t.Fatalf("直写后命中 = %d, want 3", len(after))
	}
	if after[0].Name != "zzz-z9" || after[1].Name != "zzz-a1" || after[2].Name != "zzz-a2" {
		t.Fatalf("直写后排序 = [%s %s %s], want [zzz-z9 zzz-a1 zzz-a2]（新语料必须参与 BM25 排序——指纹失效语义破坏）",
			after[0].Name, after[1].Name, after[2].Name)
	}
}
