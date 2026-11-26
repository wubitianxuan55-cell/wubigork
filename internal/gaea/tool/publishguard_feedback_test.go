package tool

// 2026-11-26 造价反馈三批回归钉（百锦路 #1、太原 #10）：发布边界的豁免面与
// 拦截回执。实测病根三处——系统提示词要求中间表格写 .gaea/work/，本边界却把
// 那里的 xlsx 一律隔离；回执只给批次目录，文件实际在批次目录下镜像原相对路径
// 的子目录里；回执建议「move_file 移到工作区外取回」，而那个方向被写根拦死。

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func writeWsFile(t *testing.T, ws, rel, content string) {
	t.Helper()
	abs := filepath.Join(ws, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func violationRels(vs []PublishViolation) []string {
	out := make([]string, 0, len(vs))
	for _, v := range vs {
		out = append(out, v.RelPath)
	}
	sort.Strings(out)
	return out
}

// .gaea/work/ 新建与修改都豁免；正式路径照拦；文件名里的 test 子串不再误放。
func TestPublishGuardWorkDirExempt(t *testing.T) {
	g, _ := d6Guard(t)
	ws := g.roots[0]
	// Begin 前先放一个过程表：实测第二次拦截就是「改了 .gaea/work 下的件」。
	writeWsFile(t, ws, ".gaea/work/成本测算/恢复.xlsx", "old")
	if err := g.Begin(); err != nil {
		t.Fatal(err)
	}
	writeWsFile(t, ws, ".gaea/work/成本测算/恢复.xlsx", "new content")
	writeWsFile(t, ws, ".gaea/work/成本测算/新建.xlsx", "n")
	writeWsFile(t, ws, ".gaea/exports/正式交付.xlsx", "d")
	writeWsFile(t, ws, ".gaea/exports/latest.xlsx", "l")

	vs, err := g.Scan()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{".gaea/exports/latest.xlsx", ".gaea/exports/正式交付.xlsx"}
	got := violationRels(vs)
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("应只拦正式路径的两件（.gaea/work 全豁免、latest 不再撞 test）\n got %v\nwant %v", got, want)
	}
}

// 回执里的隔离件路径必须指向真实文件，且是 move_file 能直接用的工作区相对写法。
func TestExpectedPathsNoteGivesRealQuarantineFilePath(t *testing.T) {
	g, _ := d6Guard(t)
	ws := g.roots[0]
	if err := g.Begin(); err != nil {
		t.Fatal(err)
	}
	vs := d6Violations(t, g, ws, ".gaea/exports/百锦路地块土壤修复成本测算表-六表工料法.xlsx", "x")
	if _, err := g.Quarantine(vs); err != nil {
		t.Fatal(err)
	}
	note := ExpectedPathsNote(vs, nil, g.LastQuarantineDest())

	var got string
	for _, line := range strings.Split(note, "\n") {
		if strings.HasPrefix(line, "    隔离件：") {
			got = strings.TrimPrefix(line, "    隔离件：")
		}
	}
	if got == "" {
		t.Fatalf("回执应逐件给出隔离件路径：%s", note)
	}
	if filepath.IsAbs(got) || !strings.HasPrefix(got, ".gaea/publish-quarantine/") {
		t.Fatalf("应给工作区相对写法（move_file 直接能用），实得 %q", got)
	}
	// 路径必须真实存在——实测模型照回执路径取回直接 source not found。
	if _, err := os.Stat(filepath.Join(ws, filepath.FromSlash(got))); err != nil {
		t.Fatalf("回执给的隔离件路径应指向真实文件 %s: %v", got, err)
	}
	// 出路必须是走得通的：工作区内取回；旧句「移到工作区外」实测被写根拦死。
	if strings.Contains(note, "工作区外暂存位") {
		t.Fatalf("不得再建议移到工作区外取回：%s", note)
	}
	if !strings.Contains(note, "destination=.gaea/work/") {
		t.Fatalf("应给出工作区内取回的可行序列：%s", note)
	}
}

// 工作区根不在场（手工拼的 violation）时退回绝对路径，不得凭空造相对路径。
func TestExpectedPathsNoteQuarantinePathFallsBackToAbsolute(t *testing.T) {
	dest := `W:\ws\.gaea\publish-quarantine\20261120-120000`
	note := ExpectedPathsNote([]PublishViolation{{RelPath: "c.docx", Kind: "created"}}, nil, dest)
	if !strings.Contains(note, dest+string(filepath.Separator)+"c.docx") {
		t.Fatalf("无根时应给绝对落点：%s", note)
	}
}

func TestNameHasTokenRequiresLeftWordBoundaryForASCII(t *testing.T) {
	cases := []struct {
		base, tok string
		want      bool
	}{
		{"latest.xlsx", "test", false},  // la·test：test 前面是字母
		{"contest.docx", "test", false}, // con·test
		{"attest.docx", "test", false},  // at·test
		{"test1.xlsx", "test", true},    // 开头
		{"my_test.xlsx", "test", true},  // 前面是下划线
		{"v2-test.docx", "test", true},  // 前面是连字符
		{"probe探测.docx", "probe", true},
		{"临时台账.docx", "临时", true}, // 中文没有分隔符，仍按子串
		{"正式交付物.docx", "test", false},
		{"正式交付物.docx", "", false},
	}
	for _, tc := range cases {
		if got := nameHasToken(tc.base, tc.tok); got != tc.want {
			t.Errorf("nameHasToken(%q, %q) = %v, want %v", tc.base, tc.tok, got, tc.want)
		}
	}
}
