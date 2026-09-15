package skillstats

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRecordBasics(t *testing.T) {
	var f File
	f = Record(f, "cost-compose", true, 100)
	f = Record(f, "cost-compose", true, 200)
	f = Record(f, "cost-compose", false, 300)
	st := f.Skills["cost-compose"]
	if st.Calls != 3 || st.Ok != 2 || st.LastAt != 300 {
		t.Fatalf("累计不对: %+v", st)
	}
	if f.Version != 1 {
		t.Fatalf("version 应恒 1: %d", f.Version)
	}
}

func TestRecordEmptyNameIgnored(t *testing.T) {
	var f File
	for _, name := range []string{"", "   ", "\t"} {
		g := Record(f, name, true, 1)
		if len(g.Skills) != 0 {
			t.Fatalf("空名应忽略: %q", name)
		}
	}
}

func TestRecordMaxSkillsRejectsNewKeepsOld(t *testing.T) {
	var f File
	for i := 0; i < MaxSkills; i++ {
		f = Record(f, "s"+string(rune('a'+i%26))+string(rune('0'+i%10))+string(rune('A'+i)), true, 1)
	}
	if len(f.Skills) != MaxSkills {
		t.Fatalf("应恰好填满: %d", len(f.Skills))
	}
	g := Record(f, "brand-new-skill", true, 2)
	if len(g.Skills) != MaxSkills {
		t.Fatalf("超限应拒新名: %d", len(g.Skills))
	}
	if _, ok := g.Skills["brand-new-skill"]; ok {
		t.Fatal("新名不应入表")
	}
	// 既有名更新不受限。
	g = Record(f, "s"+string(rune('a'+0))+string(rune('0'+0))+string(rune('A'+0)), true, 9)
	if g.Skills["s"+string(rune('a'+0))+string(rune('0'+0))+string(rune('A'+0))].Calls != 2 {
		t.Fatal("既有名应可继续累计")
	}
}

func TestViewSortAndStability(t *testing.T) {
	var f File
	f = Record(f, "alpha", true, 1)
	f = Record(f, "beta", true, 2)
	f = Record(f, "beta", false, 3)
	f = Record(f, "gamma", true, 4)
	f = Record(f, "gamma", true, 5)
	f = Record(f, "gamma", false, 6)
	v := View(f)
	if len(v) != 3 {
		t.Fatalf("应有 3 条: %d", len(v))
	}
	// gamma(3) > beta(2) > alpha(1)；beta 与无同次数者，稳定不适用，此处按 calls。
	if v[0].Name != "gamma" || v[1].Name != "beta" || v[2].Name != "alpha" {
		t.Fatalf("排序不对: %+v", v)
	}
	if v[0].Calls != 3 || v[0].Ok != 2 {
		t.Fatalf("视图字段不对: %+v", v[0])
	}
	if len(View(File{})) != 0 {
		t.Fatal("空文件视图应恒非 nil 空表")
	}
}

func TestJSONShape(t *testing.T) {
	var f File
	f = Record(f, "cost-compose", true, 1726440000)
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{`"version":1`, `"calls":1`, `"ok":1`, `"lastAt":1726440000`} {
		if !strings.Contains(s, want) {
			t.Fatalf("JSON 形状缺 %q: %s", want, s)
		}
	}
	if strings.Contains(s, "Calls") || strings.Contains(s, "LastAt") {
		t.Fatalf("PascalCase 泄漏: %s", s)
	}
	vb, err := json.Marshal(View(f))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(vb), `"name":"cost-compose"`) {
		t.Fatalf("视图 JSON 缺 name: %s", vb)
	}
}
