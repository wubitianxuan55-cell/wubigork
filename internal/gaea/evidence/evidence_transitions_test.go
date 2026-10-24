package evidence

import (
	"reflect"
	"testing"
)

func tw(todos ...TodoItem) Receipt {
	return Receipt{ToolName: "todo_write", Success: true, Todos: todos}
}

func cs(step string, success bool) Receipt {
	return Receipt{ToolName: "complete_step", Step: step, Success: success}
}

func pend(content string) TodoItem { return TodoItem{Content: content, Status: "pending"} }
func done(content string) TodoItem { return TodoItem{Content: content, Status: "completed"} }

// contents extracts the flagged items for assertion (order-stable).
func contents(missing []TodoStepMatch) []string {
	out := make([]string, len(missing))
	for i, m := range missing {
		out[i] = m.Content
	}
	return out
}

func TestUnverifiedCompletedTransitions(t *testing.T) {
	t.Run("no todo_write receipts", func(t *testing.T) {
		l := NewLedger()
		l.Record(Receipt{ToolName: "bash", Command: "ls", Success: true})
		if got := l.UnverifiedCompletedTransitions(); got != nil {
			t.Errorf("got %v, want nil without todo_write receipts", got)
		}
	})

	t.Run("first write is the baseline", func(t *testing.T) {
		// 首写即 completed 可能是前几轮携带的完成态，回合台账看不到更早
		// 的收据——不判，判了就是多轮会话的常态误报。
		l := NewLedger()
		l.Record(tw(done("carried over")))
		if got := l.UnverifiedCompletedTransitions(); got != nil {
			t.Errorf("got %v, want nil (baseline)", got)
		}
	})

	t.Run("mid-turn transition without complete_step is flagged", func(t *testing.T) {
		l := NewLedger()
		l.Record(tw(pend("A")))
		l.Record(tw(done("A")))
		want := []string{"A"}
		if got := contents(l.UnverifiedCompletedTransitions()); !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("verify after marking still covers", func(t *testing.T) {
		// 普通模式允许先标 completed 后补 complete_step（与写时闸的严格
		// 先验证相对），覆盖判定看全回合收据。
		l := NewLedger()
		l.Record(tw(pend("A")))
		l.Record(tw(done("A")))
		l.Record(cs("A", true))
		if got := l.UnverifiedCompletedTransitions(); got != nil {
			t.Errorf("got %v, want nil (covered after the fact)", got)
		}
	})

	t.Run("carried-over item stays exempt, new transition flagged", func(t *testing.T) {
		l := NewLedger()
		l.Record(tw(done("carried")))
		l.Record(tw(done("carried"), done("new work")))
		want := []string{"new work"}
		if got := contents(l.UnverifiedCompletedTransitions()); !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("failed complete_step does not cover", func(t *testing.T) {
		l := NewLedger()
		l.Record(tw(pend("A")))
		l.Record(tw(done("A")))
		l.Record(cs("A", false))
		want := []string{"A"}
		if got := contents(l.UnverifiedCompletedTransitions()); !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("revert and re-complete flags once", func(t *testing.T) {
		l := NewLedger()
		l.Record(tw(pend("A")))
		l.Record(tw(done("A")))
		l.Record(tw(pend("A")))
		l.Record(tw(done("A")))
		want := []string{"A"}
		if got := contents(l.UnverifiedCompletedTransitions()); !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want single deduped entry %v", got, want)
		}
	})

	t.Run("renamed item counts as new transition", func(t *testing.T) {
		l := NewLedger()
		l.Record(tw(pend("A")))
		l.Record(tw(done("A renamed")))
		want := []string{"A renamed"}
		if got := contents(l.UnverifiedCompletedTransitions()); !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("activeForm reported in match", func(t *testing.T) {
		l := NewLedger()
		l.Record(tw(pend("A")))
		l.Record(tw(TodoItem{Content: "A", Status: "completed", ActiveForm: "Finishing A"}))
		got := l.UnverifiedCompletedTransitions()
		if len(got) != 1 || got[0].ActiveForm != "Finishing A" {
			t.Errorf("got %+v, want ActiveForm carried through", got)
		}
	})
}

// TestUnverifiedCompletedTransitionsNilLedger 防护栏：nil 接收者不得 panic。
func TestUnverifiedCompletedTransitionsNilLedger(t *testing.T) {
	var l *Ledger
	if got := l.UnverifiedCompletedTransitions(); got != nil {
		t.Errorf("nil ledger got %v, want nil", got)
	}
}
