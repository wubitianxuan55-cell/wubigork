package evidence

// evidence_coverage_test.go — GA5-07（round 14 线2）：complete_step 证据校验由
// 「每个 completed todo 重扫全部 receipts」改为「单遍建命中集合再做集合差」。
// 本文件做两件事：
//  1. TestUnverifiedCompletedTodosLegacyEquivalence：把重构前的实现逐字复制为
//     legacy*（仅测试用），对「多 todos × 多 receipts」矩阵逐例比对新旧结论；
//  2. TestUnverifiedCompletedTodosMixedScenario：钉住混合场景（命中/未命中/多
//     complete_step/空集/缓存序号位移）的具体期望值，作为独立于 legacy 副本的
//     行为锚（改坏判据时此处必红）。

import (
	"reflect"
	"strings"
	"testing"
)

// legacyHasSuccessfulCompleteStepForTodo 是重构前 hasSuccessfulCompleteStepForTodo
// 的逐字副本（对照用，不参与生产路径）。
func legacyHasSuccessfulCompleteStepForTodo(receipts []Receipt, index int, current []TodoItem) bool {
	for _, r := range receipts {
		if !r.Success || r.ToolName != "complete_step" || strings.TrimSpace(r.Step) == "" {
			continue
		}
		if r.TodoStep != nil && r.TodoStep.Found {
			if index >= 1 && index <= len(current) && sameTodoMatch(current[index-1], *r.TodoStep) {
				return true
			}
			continue
		}
		match := matchTodoStep(r.Step, current)
		if match.Found && match.Index == index {
			return true
		}
	}
	return false
}

// legacyUnverifiedCompletedTodos 是重构前 UnverifiedCompletedTodos 的对照实现
// （同样的基线挑选 + 逐 todo 调 legacy 判据）。
func legacyUnverifiedCompletedTodos(l *Ledger, current []TodoItem) (missing []TodoStepMatch, hasBaseline bool) {
	current = normalizeTodos(current)
	if l == nil {
		return nil, false
	}
	l.mu.Lock()
	receipts := append([]Receipt(nil), l.receipts...)
	l.mu.Unlock()

	var previous []TodoItem
	for i := len(receipts) - 1; i >= 0; i-- {
		r := receipts[i]
		if !r.Success || r.ToolName != "todo_write" {
			continue
		}
		previous = r.Todos
		hasBaseline = true
		break
	}
	if !hasBaseline {
		return nil, false
	}
	for i, t := range current {
		if todoStatus(t.Status) != "completed" {
			continue
		}
		index := i + 1
		if previousTodoCompleted(index, t, previous) {
			continue
		}
		if legacyHasSuccessfulCompleteStepForTodo(receipts, index, current) {
			continue
		}
		missing = append(missing, TodoStepMatch{
			Found:      true,
			Index:      index,
			Content:    t.Content,
			Status:     todoStatus(t.Status),
			ActiveForm: t.ActiveForm,
		})
	}
	return missing, true
}

func buildLedger(receipts []Receipt) *Ledger {
	l := NewLedger()
	for _, r := range receipts {
		l.Record(r)
	}
	return l
}

func todoWrite(todos ...TodoItem) Receipt {
	return Receipt{ToolName: "todo_write", Success: true, Todos: todos}
}

func completeStep(step string) Receipt {
	return Receipt{ToolName: "complete_step", Step: step, Success: true}
}

// TestUnverifiedCompletedTodosLegacyEquivalence 是新旧口径等价矩阵：todo 列表 ×
// 基线列表 × receipts 形态（命中序号/命中标题/未命中/多条 complete_step/失败/
// 空 step/缓存被清空/缓存序号陈旧/无基线/二次基线）逐例比对 missing + hasBaseline。
func TestUnverifiedCompletedTodosLegacyEquivalence(t *testing.T) {
	currentLists := [][]TodoItem{
		{},
		{{Content: "A", Status: "completed"}},
		{{Content: "A", Status: "completed"}, {Content: "B", Status: "pending"}},
		{{Content: "A", Status: "completed"}, {Content: "B", Status: "completed"}, {Content: "C", Status: "in_progress"}},
		{{Content: "A", Status: "completed", ActiveForm: "做完 A"}, {Content: "B", Status: "completed", ActiveForm: "做完 B"}},
		{{Content: "A", Status: "completed"}, {Content: "A", Status: "completed"}},
		{{Content: "A", Status: "pending"}, {Content: "B", Status: "completed"}, {Content: "C", Status: "completed"}, {Content: "D", Status: "pending"}},
	}
	baselines := [][]TodoItem{
		{},
		{{Content: "A", Status: "pending"}, {Content: "B", Status: "pending"}},
		{{Content: "A", Status: "completed"}, {Content: "B", Status: "pending"}},
		{{Content: "A", Status: "pending", ActiveForm: "做完 A"}, {Content: "B", Status: "pending"}, {Content: "C", Status: "pending"}, {Content: "D", Status: "pending"}},
	}

	type variant struct {
		name string
		make func(baseline []TodoItem) []Receipt
	}
	variants := []variant{
		{"无 receipts", func(b []TodoItem) []Receipt { return nil }},
		{"仅基线", func(b []TodoItem) []Receipt { return []Receipt{todoWrite(b...)} }},
		{"基线+命中(序号)", func(b []TodoItem) []Receipt {
			return []Receipt{todoWrite(b...), completeStep("1"), completeStep("2")}
		}},
		{"基线+命中(标题)", func(b []TodoItem) []Receipt {
			return []Receipt{todoWrite(b...), completeStep("B")}
		}},
		{"基线+未命中", func(b []TodoItem) []Receipt {
			return []Receipt{todoWrite(b...), completeStep("zzz"), completeStep("99")}
		}},
		{"基线+多条 complete_step 重复命中", func(b []TodoItem) []Receipt {
			return []Receipt{todoWrite(b...), completeStep("1"), completeStep("1"), completeStep("2"), completeStep("1")}
		}},
		{"基线+缓存被清空(回退路径)", func(b []TodoItem) []Receipt {
			rs := []Receipt{todoWrite(b...), completeStep("2"), completeStep("B")}
			for i := range rs {
				if rs[i].ToolName == "complete_step" {
					rs[i].TodoStep = nil
				}
			}
			return rs
		}},
		{"基线+缓存序号陈旧(按内容重定位)", func(b []TodoItem) []Receipt {
			rs := []Receipt{todoWrite(b...), completeStep("2")}
			rs[1].TodoStep = &TodoStepMatch{Found: true, Index: 2, Content: "B"}
			return rs
		}},
		{"基线+缓存 Found=false", func(b []TodoItem) []Receipt {
			rs := []Receipt{todoWrite(b...), completeStep("2")}
			rs[1].TodoStep = &TodoStepMatch{Found: false, Index: 2, Content: "B"}
			return rs
		}},
		{"基线+失败 complete_step", func(b []TodoItem) []Receipt {
			return []Receipt{todoWrite(b...), {ToolName: "complete_step", Step: "1", Success: false}}
		}},
		{"基线+空 Step", func(b []TodoItem) []Receipt {
			return []Receipt{todoWrite(b...), completeStep("   ")}
		}},
		{"complete_step 先于基线(无缓存)", func(b []TodoItem) []Receipt {
			return []Receipt{completeStep("1"), todoWrite(b...)}
		}},
		{"二次基线后再 complete_step", func(b []TodoItem) []Receipt {
			return []Receipt{todoWrite(b...), completeStep("1"), todoWrite(b...), completeStep("2")}
		}},
		{"失败基线(无成功 todo_write)", func(b []TodoItem) []Receipt {
			return []Receipt{{ToolName: "todo_write", Success: false, Todos: b}, completeStep("1")}
		}},
	}

	checked := 0
	for ci, current := range currentLists {
		for bi, baseline := range baselines {
			for _, v := range variants {
				receipts := v.make(baseline)
				l := buildLedger(receipts)
				gotMissing, gotBaseline := l.UnverifiedCompletedTodos(current)
				wantMissing, wantBaseline := legacyUnverifiedCompletedTodos(l, current)
				if gotBaseline != wantBaseline || !reflect.DeepEqual(gotMissing, wantMissing) {
					t.Fatalf("current#%d baseline#%d variant=%q 新旧口径不等价:\n new  = %+v baseline=%v\n old  = %+v baseline=%v",
						ci, bi, v.name, gotMissing, gotBaseline, wantMissing, wantBaseline)
				}
				checked++
			}
		}
	}
	if checked != len(currentLists)*len(baselines)*len(variants) {
		t.Fatalf("矩阵覆盖数不符：checked=%d", checked)
	}
	t.Logf("等价矩阵覆盖 %d 例（%d current × %d baseline × %d receipts 形态）",
		checked, len(currentLists), len(baselines), len(variants))
}

// TestUnverifiedCompletedTodosMixedScenario 钉住混合场景的绝对期望（不依赖 legacy
// 副本）：多 todos × 多 receipts，含命中/未命中/多 complete_step/空集。
func TestUnverifiedCompletedTodosMixedScenario(t *testing.T) {
	baseline := []TodoItem{
		{Content: "P", Status: "pending"},
		{Content: "Q", Status: "pending"},
		{Content: "R", Status: "pending"},
		{Content: "S", Status: "pending"},
	}
	l := NewLedger()
	l.Record(todoWrite(baseline...))
	l.Record(completeStep("2"))                                             // 命中 Q（缓存 Index=2）
	l.Record(completeStep("ze-unknown"))                                    // 未命中，无缓存
	l.Record(completeStep("1"))                                             // 命中 P
	l.Record(completeStep("4"))                                             // 多条 complete_step 重复命中 S
	l.Record(Receipt{ToolName: "complete_step", Step: "3", Success: false}) // 失败：不得覆盖 R
	l.Record(completeStep("   "))                                           // 空 step：跳过

	current := []TodoItem{
		{Content: "P", Status: "completed"},
		{Content: "Q", Status: "completed"},
		{Content: "R", Status: "completed"},
		{Content: "S", Status: "completed"},
	}
	missing, hasBaseline := l.UnverifiedCompletedTodos(current)
	if !hasBaseline {
		t.Fatal("有成功 todo_write 基线，hasBaseline 应为 true")
	}
	if len(missing) != 1 || missing[0].Index != 3 || missing[0].Content != "R" {
		t.Fatalf("missing = %+v，期望仅 R@3（失败 complete_step 不得覆盖）", missing)
	}

	// 空集：所有 completed 都被覆盖 → missing 必须为空。
	currentAll := []TodoItem{
		{Content: "P", Status: "completed"},
		{Content: "Q", Status: "completed"},
		{Content: "R", Status: "completed"},
		{Content: "S", Status: "completed"},
	}
	l2 := NewLedger()
	l2.Record(todoWrite(baseline...))
	for _, step := range []string{"1", "2", "3", "4"} {
		l2.Record(completeStep(step))
	}
	if missing2, _ := l2.UnverifiedCompletedTodos(currentAll); len(missing2) != 0 {
		t.Fatalf("全覆盖场景 missing 应为空集，得到 %+v", missing2)
	}
}

// TestUnverifiedCompletedTodosRelocatesStaleCache 钉住「缓存序号位移按内容重定位」：
// complete_step 记录时命中 todo#2(B)，随后列表变为 [B,A,C]，覆盖必须落在
// 当前序号 1（内容 B）而不是陈旧序号 2（内容 A）。
func TestUnverifiedCompletedTodosRelocatesStaleCache(t *testing.T) {
	l := NewLedger()
	l.Record(todoWrite(
		TodoItem{Content: "A", Status: "pending"},
		TodoItem{Content: "B", Status: "pending"},
		TodoItem{Content: "C", Status: "pending"},
	))
	l.Record(completeStep("2")) // 命中 B，缓存 {Index:2, Content:"B"}
	l.Record(todoWrite(
		TodoItem{Content: "B", Status: "pending"},
		TodoItem{Content: "A", Status: "pending"},
		TodoItem{Content: "C", Status: "pending"},
	))

	current := []TodoItem{
		{Content: "B", Status: "completed"},
		{Content: "A", Status: "completed"},
		{Content: "C", Status: "pending"},
	}
	missing, _ := l.UnverifiedCompletedTodos(current)
	if len(missing) != 1 || missing[0].Index != 2 || missing[0].Content != "A" {
		t.Fatalf("missing = %+v，期望仅 A@2（B 按内容重定位到序号 1 被覆盖；若按陈旧序号 2 判会误覆盖 A）", missing)
	}
}

// TestCompleteStepCoveredIndicesShape 直接钉住单遍命中集合的形态（含边界：空
// todo 列表、非 complete_step、失败、空 step、越界序号回退文本匹配）。
func TestCompleteStepCoveredIndicesShape(t *testing.T) {
	current := []TodoItem{{Content: "A"}, {Content: "B"}, {Content: "C"}}
	receipts := []Receipt{
		{ToolName: "bash", Success: true, Command: "ls"},
		{ToolName: "complete_step", Step: "2", Success: false},
		{ToolName: "complete_step", Step: "", Success: true},
		{ToolName: "complete_step", Step: "1", Success: true},
		{ToolName: "complete_step", Step: "99", Success: true}, // 越界 → 回退文本匹配 → 无命中
		{ToolName: "complete_step", Step: "c", Success: true},  // 标题命中 → 3
		{ToolName: "complete_step", Step: "2", Success: true},  // 序号命中 → 2
	}
	got := completeStepCoveredIndices(receipts, current)
	want := map[int]bool{1: true, 2: true, 3: true}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("covered = %v，期望 %v", got, want)
	}
	if len(completeStepCoveredIndices(receipts, nil)) != 0 {
		t.Fatal("空 todo 列表必须返回空集合")
	}
}
