package agent

// 刀0 测试（dsh-ops 第二波 2026-11-10）：compressGrep 控制行生存——
// 生产 bug 根修的钉死测试（A1' 游标/零命中行此前被 parseGrepLines 整体洗掉）。

import (
	"fmt"
	"strings"
	"testing"
)

func TestCompressGrepPreservesControlLines(t *testing.T) {
	// >30 条命中走压缩路径：控制行必须前置存活。
	var b strings.Builder
	b.WriteString("[grep page] offset=0 shown=200 — more matches may remain; re-run with offset=200\n")
	for i := 1; i <= 40; i++ {
		fmt.Fprintf(&b, "dir/file%02d.go:%d: needle hit\n", i, i)
	}
	b.WriteString("[truncated at 200 results — refine the pattern, narrow the path, raise max_results, or re-run with offset=200]")
	got := compressGrep(b.String())
	first := strings.SplitN(got, "\n", 2)[0]
	if !strings.HasPrefix(first, "[grep page]") {
		t.Fatalf("页头必须前置存活（压缩路径）: first=%q", first)
	}
	if !strings.Contains(got, "[truncated at") {
		t.Fatalf("截断尾行必须存活: %q", got)
	}

	// passthrough 路径（≤30 条）：同样前置。
	small := "[grep page] offset=3 shown=5\ndir/a.txt:1: needle\ndir/b.txt:2: needle"
	got2 := compressGrep(small)
	if !strings.HasPrefix(got2, "[grep page]") {
		t.Fatalf("页头必须前置存活（passthrough 路径）: %q", got2)
	}
	if !strings.Contains(got2, "dir/a.txt:1:") {
		t.Fatalf("命中行不得丢: %q", got2)
	}
}

func TestCompressGrepZeroMatchesNotEmpty(t *testing.T) {
	// 零命中 grep：旧实现压缩后变空串（模型看不到「没有」）。
	got := compressGrep(`[grep page] offset=0 shown=0 — credential-fence skipped 1 file(s)
(no matches for "needle" under .)`)
	if strings.TrimSpace(got) == "" {
		t.Fatal("零命中+记账行不得被压成空串")
	}
	if !strings.Contains(got, "(no matches") {
		t.Fatalf("零命中行必须存活: %q", got)
	}
}
