package computer

// W2 树文法 golden（施工单验收 1/2）：首行正则 / 节点行正则+序号连续 /
// 163 截断几何 / 32KB 行级截断标记。夹具 testdata/tree_gaea.json，
// 不依赖真实屏幕。

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

func loadFixture(t *testing.T) (Window, *treeNode) {
	t.Helper()
	b, err := os.ReadFile("testdata/tree_gaea.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var f struct {
		Window Window    `json:"window"`
		Tree   *treeNode `json:"tree"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	return f.Window, f.Tree
}

var (
	reFirstLine = regexp.MustCompile(`^Window: ".*", App: .+\.$`)
	reNodeLine  = regexp.MustCompile(`^\t*(\d+) (\S+)(?: \(([^)]*)\))?(?: .*)?$`)
)

func TestRenderTree_Golden(t *testing.T) {
	win, root := loadFixture(t)
	out := RenderTree(win, root)
	lines := strings.Split(out, "\n")

	if !reFirstLine.MatchString(lines[0]) {
		t.Fatalf("首行正则不过: %q", lines[0])
	}
	if lines[0] != `Window: "gaea · 多功能 AI 助手", App: gaea.` {
		t.Fatalf("首行 golden 错: %q", lines[0])
	}

	// 节点行正则 + 序号从 0 连续递增 + 缩进=深度先序。
	prev := -1
	prevDepth := 0
	for _, ln := range lines[1:] {
		m := reNodeLine.FindStringSubmatch(ln)
		if m == nil {
			t.Fatalf("节点行正则不过: %q", ln)
		}
		idx := 0
		fmt.Sscanf(m[1], "%d", &idx)
		if idx != prev+1 {
			t.Fatalf("序号必须连续: prev=%d got=%d (%q)", prev, idx, ln)
		}
		prev = idx
		depth := (len(ln) - len(strings.TrimLeft(ln, "\t"))) // tab 深度
		if depth > prevDepth+1 {
			t.Fatalf("缩进不得跳层: %q", ln)
		}
		prevDepth = depth
	}

	// 抽样断言（与 codex 样本同构）：
	//   树项目 selected 态、按钮 (collapsed) + Expand、编辑 (settable, string)。
	if !strings.Contains(out, "按钮 (collapsed) 切换主题（当前 暗夜紫） Secondary Actions: Expand") {
		t.Fatalf("按钮行与 codex 文法不符:\n%s", out)
	}
	if !strings.Contains(out, "编辑 (settable, string) 搜索会话名称") {
		t.Fatalf("编辑行与 codex 文法不符:\n%s", out)
	}
	if !strings.Contains(out, "文档 gaea · 多功能 AI 助手 Value: http://wails.localhost/ ID: RootWebArea") {
		t.Fatalf("Value/ID 后缀序与 codex 不符:\n%s", out)
	}
	if !strings.Contains(out, "窗口 gaea · 多功能 AI 助手 Secondary Actions: Raise") {
		t.Fatalf("根行 Secondary Actions 不符:\n%s", out)
	}
}

func TestTrunc163(t *testing.T) {
	const marker = "... [truncated]"
	short := strings.Repeat("x", 163)
	if got := trunc163(short); got != short {
		t.Fatalf("163 字以内不得截断")
	}
	long := strings.Repeat("a", 200) + strings.Repeat("b", 200)
	got := trunc163(long)
	r := []rune(got)
	if len(r) != 163 {
		t.Fatalf("截断总长必须 163: %d", len(r))
	}
	if !strings.Contains(got, marker) {
		t.Fatalf("缺中段标记: %s", got)
	}
	if !strings.HasPrefix(got, strings.Repeat("a", 75)) || !strings.HasSuffix(got, strings.Repeat("b", 73)) {
		t.Fatalf("头 75 尾 73 几何错: %s", got)
	}
}

func TestRenderTree_ByteCap(t *testing.T) {
	win := Window{ID: 1, App: "process:C:\\x\\app.exe", Title: "t"}
	// 造一棵 >32KB 的树：600 节点 × ~100 字符名。
	root := &treeNode{Role: "窗口", Name: "big"}
	for i := 0; i < 600; i++ {
		root.Children = append(root.Children, &treeNode{
			Role: "文本",
			Name: fmt.Sprintf("%04d %s", i, strings.Repeat("长文内容", 30)),
		})
	}
	out := RenderTree(win, root)
	if !strings.Contains(out, treeTruncatedMarker) {
		t.Fatal("超 32KB 必须带行级截断标记")
	}
	if len(out) > maxTreeBytes+2*len(treeTruncatedMarker)+8 {
		t.Fatalf("产出超帽: %d bytes", len(out))
	}
	if strings.Count(out, "\n") < 50 {
		t.Fatal("截断必须按行（保住前部完整行）")
	}
}

func TestDisplayNameFromApp(t *testing.T) {
	cases := []struct{ in, want string }{
		{"process:C:\\AI\\wubigrok\\build\\bin\\gaea.exe", "gaea"},
		{"process:C:\\Windows\\System32\\notepad.exe", "notepad"},
		{"process:/usr/bin/app.d/Contents/x", "x"},
		{"plain", "plain"},
		{"process:", "process:"},
	}
	for _, c := range cases {
		if got := DisplayNameFromApp(c.in); got != c.want {
			t.Fatalf("DisplayNameFromApp(%q)=%q want %q", c.in, got, c.want)
		}
	}
}
