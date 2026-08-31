package memory

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// mkMem 构造一条记忆（只设置晨报用到的字段）。
func mkMem(name string, typ Type, kind Kind, desc string, updated, lastUsed time.Time) Memory {
	return Memory{
		Name:        name,
		Type:        typ,
		Kind:        kind,
		Description: desc,
		UpdatedAt:   updated,
		LastUsedAt:  lastUsed,
	}
}

func TestBuildMorningBriefSortByRecency(t *testing.T) {
	base := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	// 全部为 user/project（preferred，避免优先级分区干扰纯时序断言——
	// user/project 优先行为由 TestBuildMorningBriefUserProjectPriority 单独覆盖）。
	mems := []Memory{
		mkMem("a", TypeProject, KindSemantic, "A", base, time.Time{}),                    // rank = base
		mkMem("b", TypeUser, KindSemantic, "B", base.Add(1*time.Hour), time.Time{}),      // rank = base+1h
		mkMem("c", TypeProject, KindEpisodic, "C", base, base.Add(2*time.Hour)),          // rank = base+2h（LastUsedAt 胜出）
		mkMem("d", TypeProject, KindSemantic, "D", base.Add(3*time.Hour), time.Time{}),   // rank = base+3h
		mkMem("e", TypeUser, KindSemantic, "E", base.Add(4*time.Hour), time.Time{}),      // rank = base+4h
		mkMem("f", TypeProject, KindProcedural, "F", base.Add(5*time.Hour), time.Time{}), // rank = base+5h
		mkMem("g", TypeUser, KindSemantic, "G", base.Add(6*time.Hour), time.Time{}),      // rank = base+6h
	}
	b := BuildMorningBrief(mems, 3, base.Add(24*time.Hour))
	if b.Dreamed24h != 3 {
		t.Fatalf("Dreamed24h = %d, want 3", b.Dreamed24h)
	}
	if b.GeneratedAt != base.Add(24*time.Hour).UnixMilli() {
		t.Errorf("GeneratedAt = %d, want %d", b.GeneratedAt, base.Add(24*time.Hour).UnixMilli())
	}
	// 排序键 = max(UpdatedAt,LastUsedAt) 降序：g(6h) f(5h) e(4h) d(3h) c(2h via LastUsedAt)
	wantOrder := []string{"g", "f", "e", "d", "c"}
	if len(b.Items) != 5 {
		t.Fatalf("len(Items) = %d, want 5", len(b.Items))
	}
	for i, w := range wantOrder {
		if b.Items[i].Name != w {
			t.Errorf("Items[%d].Name = %q, want %q", i, b.Items[i].Name, w)
		}
	}
	// 字段映射：Kind/Category 来自 Memory.Kind/Type
	if b.Items[4].Kind != string(KindEpisodic) || b.Items[4].Category != string(TypeProject) {
		t.Errorf("Items[4] kind/category = %q/%q, want episodic/project", b.Items[4].Kind, b.Items[4].Category)
	}
	// 时间毫秒映射：c 的 UpdatedAt=base 但 LastUsedAt=base+2h → updatedAt 应为 base 毫秒
	if b.Items[4].UpdatedAt != base.UnixMilli() || b.Items[4].LastUsedAt != base.Add(2*time.Hour).UnixMilli() {
		t.Errorf("Items[4] times = %d/%d", b.Items[4].UpdatedAt, b.Items[4].LastUsedAt)
	}
}

func TestBuildMorningBriefTieBreakByName(t *testing.T) {
	base := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	mems := []Memory{
		mkMem("zeta", TypeProject, KindSemantic, "Z", base, time.Time{}),
		mkMem("alpha", TypeProject, KindSemantic, "A", base, time.Time{}),
	}
	b := BuildMorningBrief(mems, 0, base)
	if b.Items[0].Name != "alpha" || b.Items[1].Name != "zeta" {
		t.Errorf("同刻排序应为 Name 升序，got %s,%s", b.Items[0].Name, b.Items[1].Name)
	}
}

func TestBuildMorningBriefEmptyInput(t *testing.T) {
	b := BuildMorningBrief(nil, 0, time.Now())
	if b.Items == nil || b.Rules == nil {
		t.Fatalf("空输入 Items/Rules 必须为非 nil 空数组，got %v / %v", b.Items, b.Rules)
	}
	if len(b.Items) != 0 || len(b.Rules) != 0 {
		t.Fatalf("空输入应无条目，got items=%d rules=%d", len(b.Items), len(b.Rules))
	}
	if b.Dreamed24h != 0 {
		t.Errorf("Dreamed24h = %d, want 0", b.Dreamed24h)
	}
	// 空切片序列化为 [] 而非 null（前端 JSON.parse 后可直接取 length）
	out, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"items":[]`) || !strings.Contains(string(out), `"rules":[]`) {
		t.Errorf("空结构应序列化为 [] 数组，got %s", out)
	}
}

func TestBuildMorningBriefUserProjectPriority(t *testing.T) {
	base := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	// 3 条 user/project（较旧）+ 3 条其余类型（较新）：优先级先取 user/project，
	// 不足再按时序用其余补位——preferred 全部入选（按各自时序），rest 补 2 条。
	mems := []Memory{
		mkMem("u1", TypeUser, KindSemantic, "U1", base, time.Time{}),
		mkMem("p1", TypeProject, KindSemantic, "P1", base.Add(time.Hour), time.Time{}),
		mkMem("p2", TypeProject, KindEpisodic, "P2", base.Add(2*time.Hour), time.Time{}),
		mkMem("r1", TypeReference, KindSemantic, "R1", base.Add(3*time.Hour), time.Time{}),
		mkMem("f1", TypeFeedback, KindSemantic, "F1", base.Add(4*time.Hour), time.Time{}),
		mkMem("r2", TypeReference, KindSemantic, "R2", base.Add(5*time.Hour), time.Time{}),
	}
	b := BuildMorningBrief(mems, 0, base)
	// preferred（user/project）按各自时序相对序 p2(2h)>p1(1h)>u1(base) 排前，
	// 不足再补 rest 时序前 2 条 r2(5h)>f1(4h)。
	wantOrder := []string{"p2", "p1", "u1", "r2", "f1"}
	if len(b.Items) != 5 {
		t.Fatalf("len(Items) = %d, want 5", len(b.Items))
	}
	for i, w := range wantOrder {
		if b.Items[i].Name != w {
			t.Errorf("Items[%d].Name = %q, want %q（user/project 优先补位）", i, b.Items[i].Name, w)
		}
	}
}

func TestBuildMorningBriefTruncateMultibyte(t *testing.T) {
	base := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	// 150 个中文字符（每个 3 字节）：必须按 rune 边界截断，不产生半个字符。
	long := strings.Repeat("记", 150)
	mems := []Memory{mkMem("long", TypeProject, KindSemantic, long, base, time.Time{})}
	b := BuildMorningBrief(mems, 0, base)
	got := b.Items[0].Description
	if !utf8.ValidString(got) {
		t.Fatal("截断结果不是合法 UTF-8")
	}
	if n := utf8.RuneCountInString(got); n != 120 {
		t.Fatalf("截断后 rune 数 = %d, want 120（119 + …）", n)
	}
	if !strings.HasSuffix(got, "\u2026") {
		t.Errorf("截断结果应以 … 结尾，got %q", got)
	}
	if !strings.HasPrefix(got, strings.Repeat("记", 119)) {
		t.Errorf("截断应保留前 119 rune，got %q", got)
	}
	// 预算：单条 ≤120，5 条 ≤600（总预算 ~600 rune 精神）
	if b.Items[0].Description == long {
		t.Error("超长描述未截断")
	}
}

func TestBuildMorningBriefTruncateShortKept(t *testing.T) {
	base := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	mems := []Memory{mkMem("short", TypeProject, KindSemantic, "刚好 120 个字的描述……", base, time.Time{})}
	b := BuildMorningBrief(mems, 0, base)
	if b.Items[0].Description != "刚好 120 个字的描述……" {
		t.Errorf("短描述不应被截断，got %q", b.Items[0].Description)
	}
}

func TestBuildMorningBriefRulesCap(t *testing.T) {
	base := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	mems := []Memory{
		mkMem("r5", TypeProject, KindProcedural, "规则五", base.Add(5*time.Hour), time.Time{}),
		mkMem("r4", TypeProject, KindProcedural, "规则四", base.Add(4*time.Hour), time.Time{}),
		mkMem("r3", TypeProject, KindProcedural, "规则三", base.Add(3*time.Hour), time.Time{}),
		mkMem("r2", TypeProject, KindProcedural, "规则二", base.Add(2*time.Hour), time.Time{}),
		mkMem("r1", TypeProject, KindProcedural, "规则一", base.Add(time.Hour), time.Time{}),
		// 非 procedure/rule 不得混入规则区
		mkMem("s1", TypeUser, KindSemantic, "画像事实", base.Add(6*time.Hour), time.Time{}),
		// "rule" 字符串 Kind 兼容（防御性）
		mkMem("rx", TypeProject, Kind("rule"), "规则兼容", base.Add(7*time.Hour), time.Time{}),
	}
	b := BuildMorningBrief(mems, 0, base)
	if len(b.Rules) != 3 {
		t.Fatalf("len(Rules) = %d, want 3（上限）", len(b.Rules))
	}
	// 同排序键降序：rx(7h) r5(5h) r4(4h)；s1 是 semantic 不参与
	wantRules := []string{"规则兼容", "规则五", "规则四"}
	for i, w := range wantRules {
		if b.Rules[i] != w {
			t.Errorf("Rules[%d] = %q, want %q", i, b.Rules[i], w)
		}
	}
}

func TestBuildMorningBriefNoRules(t *testing.T) {
	base := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	mems := []Memory{
		mkMem("s1", TypeUser, KindSemantic, "画像", base, time.Time{}),
		mkMem("e1", TypeProject, KindEpisodic, "经验", base, time.Time{}),
	}
	b := BuildMorningBrief(mems, 0, base)
	if b.Rules == nil || len(b.Rules) != 0 {
		t.Fatalf("无规则时应为空数组，got %v", b.Rules)
	}
	if len(b.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(b.Items))
	}
}

func TestBuildMorningBriefBudget(t *testing.T) {
	base := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	// 5 条超长描述（300 rune 中文）→ 全部截断到 120 rune → 总预算 ≤600。
	mems := make([]Memory, 0, 5)
	for i := 0; i < 5; i++ {
		mems = append(mems, mkMem(
			strings.ToLower(string(rune('a'+i)))+"-mem",
			TypeProject, KindSemantic,
			strings.Repeat("长", 300),
			base.Add(time.Duration(i)*time.Hour), time.Time{},
		))
	}
	b := BuildMorningBrief(mems, 0, base)
	if len(b.Items) != 5 {
		t.Fatalf("items = %d, want 5", len(b.Items))
	}
	total := 0
	for _, it := range b.Items {
		total += utf8.RuneCountInString(it.Description)
	}
	if total > 600 {
		t.Errorf("描述总 rune = %d, 超出 600 预算", total)
	}
	if total != 5*120 {
		t.Errorf("描述总 rune = %d, want %d", total, 5*120)
	}
}

func TestBuildMorningBriefDoesNotMutateInput(t *testing.T) {
	base := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	mems := []Memory{
		mkMem("b", TypeProject, KindSemantic, "B", base, time.Time{}),
		mkMem("a", TypeProject, KindSemantic, "A", base.Add(time.Hour), time.Time{}),
	}
	orig := append([]Memory(nil), mems...)
	_ = BuildMorningBrief(mems, 0, base)
	for i := range orig {
		if orig[i].Name != mems[i].Name {
			t.Fatalf("BuildMorningBrief 不应改写输入切片（%s vs %s）", orig[i].Name, mems[i].Name)
		}
	}
}
