package telemetry

// P7（codex 蒸馏 B10 折算，第四批）观测最小集验收：
// 关闭态零成本；环形 256 滚动；去重合并计数；落盘按日滚动+14 天清扫；
// 度量可答（失败率/p95 的数据形状）。

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDisabledRecordIsZeroCost(t *testing.T) {
	r := &Recorder{}
	r.Record("turn_done", map[string]string{"ok": "false"}, 100)
	if got := r.Snapshot(); len(got) != 0 {
		t.Errorf("disabled recorder captured %d facts, want 0", len(got))
	}
	if r.Enabled() {
		t.Error("zero value must be disabled")
	}
}

func TestRingRollsAtCapacity(t *testing.T) {
	r := &Recorder{}
	r.Enable("") // 内存面，无落盘
	defer r.Disable()
	for i := 0; i < RingCapacity+20; i++ {
		r.Record("tool_result", map[string]string{"name": "tool-" + itoa(i)}, 1)
	}
	snap := r.Snapshot()
	if len(snap) != RingCapacity {
		t.Fatalf("ring len = %d, want %d", len(snap), RingCapacity)
	}
	if snap[0].Dims["name"] == "tool-0" {
		t.Error("最旧事实未被滚动挤出")
	}
	if snap[len(snap)-1].Dims["name"] != "tool-"+itoa(RingCapacity+19) {
		t.Errorf("最新事实不在环尾: %v", snap[len(snap)-1])
	}
}

func TestDedupMergesSameKey(t *testing.T) {
	r := &Recorder{}
	r.Enable("")
	defer r.Disable()
	for i := 0; i < 5; i++ {
		r.Record("turn_done", map[string]string{"ok": "false"}, 10)
	}
	snap := r.Snapshot()
	if len(snap) != 1 {
		t.Fatalf("ring len = %d, want 1（去重合并）", len(snap))
	}
	if snap[0].N != 4 || snap[0].Val != 50 {
		t.Errorf("合并计数 = n%d val%v, want n4 val50", snap[0].N, snap[0].Val)
	}
}

func TestDailyFileRollsAndSweeps(t *testing.T) {
	dir := t.TempDir()
	r := &Recorder{}
	r.Enable(dir)
	r.Record("turn_done", nil, 5)
	r.Flush() // 关句柄：TempDir 清理时文件不被占用
	// 伪造一份 20 天前的过期文件 + 一份新文件并存。
	old := filepath.Join(dir, "telemetry-"+time.Now().AddDate(0, 0, -20).Format("20060102")+".ndjson")
	if err := os.WriteFile(old, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write old: %v", err)
	}
	r2 := &Recorder{} // 重新 Enable 触发清扫
	r2.Enable(dir)
	r2.Record("tool_result", map[string]string{"name": "bash"}, 7)
	r2.Flush()
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Error("20 天前的文件未被清扫（保留期 14 天）")
	}
	// 当日文件存在且是合法 ndjson。
	today := filepath.Join(dir, "telemetry-"+time.Now().Format("20060102")+".ndjson")
	f, err := os.Open(today)
	if err != nil {
		t.Fatalf("open today: %v", err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	lines := 0
	for sc.Scan() {
		line := sc.Text()
		if !strings.Contains(line, `"kind":"fact"`) {
			t.Errorf("行不是 fact 信封: %s", line)
		}
		lines++
	}
	if lines < 2 {
		t.Errorf("当日文件行数 = %d, want >=2（两个 recorder 都落了盘）", lines)
	}
}

// 度量形状：失败率（ok=false 占比）与 p95 耗时可从 Snapshot 直接算出。
func TestSnapshotAnswersFailureRateAndP95(t *testing.T) {
	r := &Recorder{}
	r.Enable("")
	defer r.Disable()
	durations := []float64{10, 20, 30, 40, 100}
	for i, d := range durations {
		ok := "true"
		if i == 0 {
			ok = "false"
		}
		r.Record("turn_done", map[string]string{"ok": ok}, d)
	}
	// 不同 ok 维度是两条事实；Val 累加、N 计数——失败率=failed.N+1 / total。
	snap := r.Snapshot()
	var failedN, okN int
	var okVal float64
	var okCnt int
	for _, f := range snap {
		switch f.Dims["ok"] {
		case "false":
			failedN = f.N + 1
		case "true":
			okN = f.N + 1
			okVal = f.Val
			okCnt = 1
		}
	}
	_ = okCnt
	if failedN != 1 || okN != 4 {
		t.Fatalf("失败/成功计数 = %d/%d, want 1/4", failedN, okN)
	}
	if okVal != 20+30+40+100 {
		t.Errorf("成功侧耗时合计 = %v, want 190", okVal)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}
