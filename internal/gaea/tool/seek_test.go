package tool

import (
	"reflect"
	"testing"
)

// 精确命中：档 0 最早位置优先。
func TestFindSequenceExact(t *testing.T) {
	lines := []string{"a", "b", "c", "b", "c"}
	idx, tier := FindSequence(lines, []string{"b", "c"}, 0)
	if idx != 1 || tier != 0 {
		t.Fatalf("idx=%d tier=%d, want 1/0", idx, tier)
	}
	// 自 at 起找，跳过早前命中。
	idx, tier = FindSequence(lines, []string{"b", "c"}, 2)
	if idx != 3 || tier != 0 {
		t.Fatalf("idx=%d tier=%d, want 3/0", idx, tier)
	}
}

// 档 1：行尾空白放宽。
func TestFindSequenceTrailingSpace(t *testing.T) {
	lines := []string{"x", "foo  ", "bar\t"}
	idx, tier := FindSequence(lines, []string{"foo", "bar"}, 0)
	if idx != 1 || tier != 1 {
		t.Fatalf("idx=%d tier=%d, want 1/1", idx, tier)
	}
}

// 档 2：两侧空白放宽。
func TestFindSequenceBothSides(t *testing.T) {
	lines := []string{"  foo", "bar  "}
	idx, tier := FindSequence(lines, []string{"foo", "bar"}, 0)
	if idx != 0 || tier != 2 {
		t.Fatalf("idx=%d tier=%d, want 0/2", idx, tier)
	}
}

// 最低档位优先：同一段在档 0 能中就绝不报档 1/2；早位置只在低档位才有意义。
func TestFindSequenceTierOrder(t *testing.T) {
	// 位置 0 只在档 2 命中，位置 2 在档 0 命中——档 0 的位置 2 赢。
	lines := []string{"  foo", "x", "foo"}
	idx, tier := FindSequence(lines, []string{"foo"}, 0)
	if idx != 2 || tier != 0 {
		t.Fatalf("idx=%d tier=%d, want 2/0", idx, tier)
	}
}

// 跨段不误命中：pat 必须整段连续，隔行即不中。
func TestFindSequenceNoCrossGap(t *testing.T) {
	lines := []string{"a", "zzz", "b"}
	if idx, _ := FindSequence(lines, []string{"a", "b"}, 0); idx != -1 {
		t.Fatalf("跨段命中: idx=%d", idx)
	}
}

// 空 pat / pat 长于文件 / at 越界：一律未命中。
func TestFindSequenceDegenerate(t *testing.T) {
	lines := []string{"a", "b"}
	if idx, _ := FindSequence(lines, nil, 0); idx != -1 {
		t.Fatalf("空 pat 应未命中")
	}
	if idx, _ := FindSequence(lines, []string{"a", "b", "c"}, 0); idx != -1 {
		t.Fatalf("pat 长于文件应未命中")
	}
	if idx, _ := FindSequence(lines, []string{"a"}, -1); idx != -1 {
		t.Fatalf("at 越界应未命中")
	}
	if idx, _ := FindSequence(lines, []string{"a"}, 2); idx != -1 {
		t.Fatalf("at 越界应未命中")
	}
}

// NearestMatches：首行相似度排序、并列行号升序、封顶 n 条。
func TestNearestMatches(t *testing.T) {
	lines := []string{"func main() {", "fmt.Println(1)", "funcm ain() {", "func helper() {}"}
	// 公共前缀：行1=13、行2=1、行3=4（funcm 第 5 字符断）、行4=5（"func " 后断）
	// → 前 2 名 = [1, 4]。
	got := NearestMatches(lines, []string{"func main() {"}, 2)
	if !reflect.DeepEqual(got, []int{1, 4}) {
		t.Fatalf("got %v, want [1 4]", got)
	}
	if NearestMatches(lines, []string{}, 3) != nil {
		t.Fatalf("空 pat 应返回空")
	}
	if NearestMatches([]string{"", "  "}, []string{"x"}, 3) != nil {
		t.Fatalf("全空白行应返回空")
	}
}
