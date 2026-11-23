package app

// wenxue_feedback_batch_test.go — 2026-11-23 实测反馈批（《软件不足与建议》）
// 工具层五刀行为锁：汉字口径/字数硬门禁/角色卡四级匹配/单章交付/定稿内联体检。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWenxueCountHanzi(t *testing.T) {
	// 「第二章工具报 3652，汉字只有 3026」的口径差：标点/空白不计。
	got := wenxueCountHanzi("雨夜。巷口的路灯忽明忽暗， he raised his knife——！")
	if got != 11 { // 雨夜巷口的路灯忽明忽暗 = 11 汉字
		t.Fatalf("wenxueCountHanzi = %d, want 11（标点/空白/拉丁不计）", got)
	}
	if wenxueCountWords("雨夜。 he") != 5 { // 旧口径对照：2 汉字+1 标点+2 拉丁字母
		t.Fatalf("wenxueCountWords 口径漂移: %d", wenxueCountWords("雨夜。 he"))
	}
}

func TestWenxueHanziGateRejectsShortChapter(t *testing.T) {
	root := wenxueToolsTestRoot(t)
	// 不过线：拒写且不落盘。
	if _, err := wenxueChapterWriteTool().Execute(t.Context(), json.RawMessage(
		`{"number":1,"title":"开局","content":"雨夜。","min_hanzi":3000}`)); err == nil {
		t.Fatal("min_hanzi 门禁应拒写过短章")
	}
	if _, err := os.Stat(filepath.Join(root, "章节", "001-开局.md")); !os.IsNotExist(err) {
		t.Fatalf("门禁拒绝后不应落盘: %v", err)
	}
	// 达线：写盘且回执带 hanzi。
	content := strings.Repeat("雨夜的路灯忽明忽暗。", 200) // 200*9=1800 汉字
	out := mustRun(t, wenxueChapterWriteTool(),
		`{"number":1,"title":"开局","min_hanzi":1000,"content":"`+content+`"}`)
	var receipt map[string]any
	if err := json.Unmarshal([]byte(out), &receipt); err != nil {
		t.Fatal(err)
	}
	if hanzi, _ := receipt["hanzi"].(float64); int(hanzi) != 1800 {
		t.Fatalf("回执 hanzi = %v, want 1800", receipt["hanzi"])
	}
}

func TestWenxueFinalWriteGateAndInlineCheck(t *testing.T) {
	root := wenxueToolsTestRoot(t)
	// min_hanzi 硬门禁同样作用于定稿。
	if _, err := wenxueFinalWriteTool().Execute(t.Context(), json.RawMessage(
		`{"number":1,"title":"定","content":"太短。","min_hanzi":100}`)); err == nil {
		t.Fatal("final_write min_hanzi 门禁应拒写")
	}
	// 过线但体量单薄：入库成功 + quality.thinChapter 提示（非阻断）。
	out := mustRun(t, wenxueFinalWriteTool(),
		`{"number":2,"title":"入库","content":"`+strings.Repeat("雪落了下来。", 80)+`"}`)
	var receipt map[string]any
	if err := json.Unmarshal([]byte(out), &receipt); err != nil {
		t.Fatal(err)
	}
	if _, ok := receipt["hanzi"]; !ok {
		t.Fatal("final_write 回执缺 hanzi")
	}
	q, _ := receipt["quality"].(map[string]any)
	if q == nil || q["thinChapter"] == nil {
		t.Fatalf("单薄章应带 quality.thinChapter 提示: %v", receipt["quality"])
	}
	_ = root
}

func TestWenxueResolveCharacterCardLadder(t *testing.T) {
	root := wenxueToolsTestRoot(t)
	if err := os.MkdirAll(filepath.Join(root, "角色"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, "角色", name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("舅妈.md", "# 舅妈（王秀芬）\n\n## 设定\n远房亲戚。\n")
	write("表姨.md", "## 设定\n无标题行卡。\n")
	write("舅妈的闺蜜.md", "# 舅妈的闺蜜\n\n## 设定\n另一张卡。\n")

	// ① 精确（标题全等）；② stem（文件名兜底）；③ 去括号；③b 唯一子串。
	for _, tc := range []struct{ want, expect string }{
		{"舅妈（王秀芬）", "角色/舅妈.md"},
		{"表姨", "角色/表姨.md"},    // stem 等值（无标题行）
		{"舅妈", "角色/舅妈.md"},      // 去括号等值命中 舅妈（王秀芬）
		{"闺蜜", "角色/舅妈的闺蜜.md"}, // 唯一子串（卡名=舅妈的闺蜜）
	} {
		got, err := wenxueResolveCharacterCard(root, tc.want)
		if err != nil {
			t.Fatalf("定位 %q: %v", tc.want, err)
		}
		if got != tc.expect {
			t.Fatalf("定位 %q = %s, want %s", tc.want, got, tc.expect)
		}
	}
	// 多张子串命中=歧义报名单。
	if _, err := wenxueResolveCharacterCard(root, "妈"); err == nil || !strings.Contains(err.Error(), "歧义") {
		t.Fatalf("多张命中应报歧义: %v", err)
	}
	// 全不命中=报实际卡名。
	_, err := wenxueResolveCharacterCard(root, "不存在的人")
	if err == nil || !strings.Contains(err.Error(), "舅妈（王秀芬）") {
		t.Fatalf("失败应列实际卡名: %v", err)
	}
	// 工具链贯通：character_read 按短名读到长标题卡。
	out := mustRun(t, wenxueCharacterReadTool(), `{"name":"舅妈"}`)
	if !strings.Contains(out, "王秀芬") {
		t.Fatalf("character_read 未命中去括号定位: %s", out)
	}
}

func TestWenxueChapterDeliverTool(t *testing.T) {
	root := wenxueToolsTestRoot(t)
	mustRun(t, wenxueChapterWriteTool(),
		`{"number":3,"title":"夜访","content":"`+strings.Repeat("他敲了敲门。", 50)+`"}`)

	// 缺省落点：.gaea/exports/单章/003-夜访.md；回执给绝对路径+hanzi+阅读页声明。
	out := mustRun(t, wenxueChapterDeliverTool(), `{"number":3}`)
	var receipt map[string]any
	if err := json.Unmarshal([]byte(out), &receipt); err != nil {
		t.Fatal(err)
	}
	rel, _ := receipt["delivered"].(string)
	if !strings.Contains(filepath.ToSlash(rel), ".gaea/exports/单章/003-夜访.md") {
		t.Fatalf("交付落点不符: %v", receipt["delivered"])
	}
	if _, err := os.Stat(rel); err != nil {
		t.Fatalf("交付件未落盘: %v", err)
	}
	if h, _ := receipt["hanzi"].(float64); int(h) != 250 {
		t.Fatalf("交付回执 hanzi = %v, want 250", receipt["hanzi"])
	}
	if hint, _ := receipt["hint"].(string); !strings.Contains(hint, "不会收录") {
		t.Fatal("交付回执应声明阅读页不收录交付件")
	}

	// 重复交付同名=拒（宁拒不误改）。
	if _, err := wenxueChapterDeliverTool().Execute(t.Context(), json.RawMessage(`{"number":3}`)); err == nil {
		t.Fatal("交付件已存在应拒")
	}
	// 禁止写进定稿域。
	if _, err := wenxueChapterDeliverTool().Execute(t.Context(), json.RawMessage(
		`{"number":3,"output_path":"定稿/交付.md"}`)); err == nil {
		t.Fatal("交付件禁止写定稿目录")
	}
	// source=final 从定稿取源。
	if _, err := wenxueFinalWriteTool().Execute(t.Context(), json.RawMessage(
		`{"number":3,"title":"夜访","overwrite":true,"content":"`+strings.Repeat("定稿正文。", 60)+`"}`)); err != nil {
		t.Fatal(err)
	}
	out2 := mustRun(t, wenxueChapterDeliverTool(), `{"number":3,"source":"final","output_path":".gaea/exports/单章/003-夜访-定稿版.md"}`)
	var receipt2 map[string]any
	if err := json.Unmarshal([]byte(out2), &receipt2); err != nil {
		t.Fatal(err)
	}
	if src, _ := receipt2["source"].(string); src != "定稿/003-夜访.md" {
		t.Fatalf("source=final 应从定稿取源: %v", receipt2["source"])
	}
	_ = root
}
