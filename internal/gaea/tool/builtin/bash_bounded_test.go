package builtin

import (
	"bytes"
	"strings"
	"testing"
)

// ─── 刀6: 运行中封顶的前台输出缓冲（v4.374，Reasonix shellrun/bounded.go 蒸馏）──

func TestBoundedOutputSmallPassthrough(t *testing.T) {
	var b boundedOutput
	if _, err := b.Write([]byte("hello world")); err != nil {
		t.Fatal(err)
	}
	if b.String() != "hello world" {
		t.Fatalf("small output must pass through verbatim, got %q", b.String())
	}
}

func TestBoundedOutputCapsWhileRunning(t *testing.T) {
	var b boundedOutput
	big := bytes.Repeat([]byte("x"), outputHeadKeep+outputTailKeep*2+123)
	if _, err := b.Write(big); err != nil {
		t.Fatal(err)
	}
	got := b.String()
	if len(got) >= len(big) {
		t.Fatalf("buffer must stay bounded: got %d bytes for %d input", len(got), len(big))
	}
	if !strings.Contains(got, "truncated while running") {
		t.Fatal("missing truncation marker")
	}
	if !strings.HasPrefix(got, "x") || !strings.HasSuffix(got, "x") {
		t.Fatal("head and tail must be preserved")
	}
}

// 多次写入累计滚动：尾环应保留最后写入的尾部而非第一次的。
func TestBoundedOutputRollingTail(t *testing.T) {
	var b boundedOutput
	if _, err := b.Write(bytes.Repeat([]byte("a"), outputHeadKeep+10)); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Write(bytes.Repeat([]byte("z"), outputTailKeep)); err != nil {
		t.Fatal(err)
	}
	got := b.String()
	if !strings.HasSuffix(got, strings.Repeat("z", outputTailKeep)) {
		t.Fatal("rolling tail must keep the most recent bytes")
	}
}
