package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// 实测反馈①：9 子代理并行写撞表号 10 处+图号跳号，工具侧无提示，事后
// 写映射表全量重排返工 2 轮。四类检出+干净面全钉。

func mdvRun(t *testing.T, content string, checks string) (string, error) {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "doc.md")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	args := `{"path":"doc.md"` + checks + `}`
	return mdValidate{workDir: dir}.Execute(context.Background(), json.RawMessage(args))
}

func TestMDValidate_DuplicateAndGap(t *testing.T) {
	out, err := mdvRun(t, `# 第一章

引言见表1-4。

表1-4 参数一览
| a | b |
|---|---|
| 1 | 2 |

表1-4 重复题注（撞号）

表1-8 汇总表

图1-11 流程图
`, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "表1-4 撞号") {
		t.Fatalf("撞号未检出: %s", out)
	}
	if !strings.Contains(out, "跳号") || !strings.Contains(out, "缺 表1-5") {
		t.Fatalf("跳号未检出: %s", out)
	}
}

func TestMDValidate_OrphanRefAndUnreferenced(t *testing.T) {
	out, err := mdvRun(t, `正文引用表2-1 与表2-3。

表2-1 甲表
内容

表2-2 乙表
内容
`, `,"checks":["orphans"]`)
	if err != nil {
		t.Fatal(err)
	}
	// 2026-11-21 反馈 P1-4：单文件模式下本地无题注的引用不再冒充「孤立引用」
	// 硬伤——归类 cross_ref（跨章引用·待全册核验），防诱发把有效跨章引用改成
	// 模糊描述（实测曾因此赶出 9 条误报）。
	if !strings.Contains(out, "cross_ref") || !strings.Contains(out, "表2-3") || !strings.Contains(out, "待全册核验") {
		t.Fatalf("跨章引用应归类 cross_ref 待核: %s", out)
	}
	if strings.Contains(out, "孤立引用") {
		t.Fatalf("单文件模式不得再把跨章引用报成孤立引用: %s", out)
	}
	if !strings.Contains(out, "未引用") || !strings.Contains(out, "表2-2") {
		t.Fatalf("未引用题注未检出: %s", out)
	}
}

// 多文件全册模式（P1-4）：跨章引用在他章题注中闭合即通过（不误报）；并集
// 仍无=真孤立；跨文件撞号点名两侧。
func mdvRunMulti(t *testing.T, files map[string]string, checks string) (string, error) {
	t.Helper()
	dir := t.TempDir()
	names := make([]string, 0, len(files))
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	sort.Strings(names)
	pathsJSON, _ := json.Marshal(names)
	args := `{"paths":` + string(pathsJSON) + checks + `}`
	return mdValidate{workDir: dir}.Execute(context.Background(), json.RawMessage(args))
}

func TestMDValidate_MultiFileCrossRefClosed(t *testing.T) {
	out, err := mdvRunMulti(t, map[string]string{
		"ch1.md": "本章安排见 表2-1。\n\n表1-1 本章表\n\n引用 表1-1。\n",
		"ch2.md": "表2-1 里程碑计划\n\n节点日期见 表1-1。\n",
	}, `,"checks":["orphans","numbering"]`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "OK:") {
		t.Fatalf("跨章引用闭合应 OK: %s", out)
	}
	if !strings.Contains(out, "跨章引用 2 处已在全册并集闭合") {
		t.Fatalf("应汇总跨章闭合数: %s", out)
	}
}

func TestMDValidate_MultiFileTrueOrphanAndCrossDup(t *testing.T) {
	out, err := mdvRunMulti(t, map[string]string{
		"ch1.md": "引用 表9-9（全册无此表）。\n\n表1-1 甲\n",
		"ch2.md": "表1-1 乙（与 ch1 撞号）\n",
	}, `,"checks":["orphans","numbering"]`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "孤立引用（全册并集）") || !strings.Contains(out, "表9-9") {
		t.Fatalf("全册真孤立应检出: %s", out)
	}
	if !strings.Contains(out, "表1-1 撞号") || !strings.Contains(out, "ch1.md L3") {
		t.Fatalf("跨文件撞号应点名两侧: %s", out)
	}
}

func TestMDValidate_TableColumnMismatch(t *testing.T) {
	out, err := mdvRun(t, `表3-1 坏表
| a | b | c |
|---|---|---|
| 1 | 2 |
`, `,"checks":["table_columns"]`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "列数不一致") {
		t.Fatalf("列数不一致未检出: %s", out)
	}
}

func TestMDValidate_CleanDocOK(t *testing.T) {
	out, err := mdvRun(t, `见 表1-1 与 图1-1。

表1-1 好数据
| a | b |
|---|---|
| 1 | 2 |

图1-1 好图
（图内容）
`, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "OK:") {
		t.Fatalf("干净文档应 OK 开头: %s", out)
	}
	if !strings.Contains(out, "编号唯一且连续") {
		t.Fatalf("OK 行缺摘要: %s", out)
	}
}

// 后缀变体（表1-8B）不误报跳号：主干连续即可，后缀是分支形态。
// （只跑 numbering——无正文引用时 orphans 的「未引用」是正常检出，不混侧。）
func TestMDValidate_SuffixVariantNoFalsePositive(t *testing.T) {
	out, err := mdvRun(t, `表1-8 主表
表1-8B 分表
表1-9 续表
`, `,"checks":["numbering"]`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "OK:") {
		t.Fatalf("后缀变体主干连续不应报跳号: %s", out)
	}
}

// SpaceTag 自声明与静态表/registrymap 同口径（work）。
func TestMDValidate_SpaceTag(t *testing.T) {
	var v mdValidate
	if v.SpaceTag() != "work" {
		t.Fatalf("SpaceTag=%q, want work", v.SpaceTag())
	}
	if !v.ReadOnly() {
		t.Fatal("md_validate 必须只读")
	}
}

// 2026-11-21 复盘 B6：「附图8/附表2/插图3/图幅5」是附录/外部编号体系，不得
// 被读成内部引用「图8/表2/图3/图5」制造假阳性；中文长破折号「表1—3」与
// 「表1-3」应归一为同一编号互相对账。
func TestMDValidate_CompoundPrefixAndEmDash(t *testing.T) {
	// 复合前缀：全文只有 图3-1 一对题注/引用，附图表等外部编号不产生孤儿。
	out, err := mdvRun(t, `# 第三章

平面布置见图3-1。

图3-1 平面布置图

外部编号体系不在对账范围：附图8、附表2、插图3、图幅5 均为附录编号，与本文档无关。
`, "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "孤立引用") || strings.Contains(out, "未引用") || strings.Contains(out, "issue") {
		t.Fatalf("复合前缀不得产生假阳性（B6）：%s", out)
	}

	// 长破折号：题注「表1—3」与引用「表1-3」「表1—3」互认。
	out2, err := mdvRun(t, `# 第一章

参数见表1-3，另见 表1—3 的说明。

表1—3 参数表
| a | b |
|---|---|
| 1 | 2 |
`, "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out2, "孤立引用") || strings.Contains(out2, "未引用") {
		t.Fatalf("长破折号应归一互认（B6）：%s", out2)
	}
}
