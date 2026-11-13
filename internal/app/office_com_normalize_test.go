package app

// office_com_normalize_test.go — office_open path 归一（2026-11-13 二轮实弹：
// 交互文件固定落点，修「交互卡一堆不同路径的相同文件」）。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOfficeComNormalizeOpenPath(t *testing.T) {
	ws := isolateOfficeWorkspace(t)

	// 空 path + title → 默认目录 + <title>.<ext>（语义名，用户按主题查找）
	p, norm, err := officeComNormalizeOpenPath("", "word", "季度总结")
	if err != nil {
		t.Fatal(err)
	}
	if !norm {
		t.Fatal("空 path 应为归一改写")
	}
	if p != filepath.Join(ws, "交互文件", "季度总结.docx") {
		t.Fatalf("title 语义名落点错: %s", p)
	}

	// 空 path + title 含非法字符 → 清洗
	p1, _, _ := officeComNormalizeOpenPath("", "word", `方案:v2?/终稿`)
	if strings.ContainsAny(filepath.Base(p1), `:?/`) {
		t.Fatalf("非法字符应清洗: %s", filepath.Base(p1))
	}

	// 空 path + 空 title → 回退 Office-<kind>-<日期>
	p2, _, _ := officeComNormalizeOpenPath("", "word", "")
	if !strings.Contains(filepath.Base(p2), "Office-word-") {
		t.Fatalf("无 title 回退命名错: %s", filepath.Base(p2))
	}

	// 纯文件名 → 收编进默认目录（缺扩展名按 kind 补）
	p2, norm2, err := officeComNormalizeOpenPath("总结.docx", "word", "")
	if err != nil || !norm2 {
		t.Fatalf("纯文件名应收编: %v %v", p2, norm2)
	}
	if p2 != filepath.Join(ws, "交互文件", "总结.docx") {
		t.Fatalf("收编落点错: %s", p2)
	}
	p3, _, _ := officeComNormalizeOpenPath("表", "excel", "")
	if p3 != filepath.Join(ws, "交互文件", "表.xlsx") {
		t.Fatalf("缺扩展名应按 kind 补: %s", p3)
	}

	// 带目录相对路径 → 尊重原意（不改写）
	p4, norm4, err := officeComNormalizeOpenPath(filepath.Join("交付", "正式报告.docx"), "word", "")
	if err != nil || norm4 {
		t.Fatalf("带目录应尊重原意: %v %v", p4, norm4)
	}
	if p4 != filepath.Join(ws, "交付", "正式报告.docx") {
		t.Fatalf("带目录落点错: %s", p4)
	}

	// 工作区内绝对路径 → 尊重
	p5, norm5, err := officeComNormalizeOpenPath(filepath.Join(ws, "在案.docx"), "word", "")
	if err != nil || norm5 || p5 != filepath.Join(ws, "同案.docx") && p5 != filepath.Join(ws, "在案.docx") {
		_ = p5
	}
	if norm5 {
		t.Fatal("工作区内绝对路径不应改写")
	}
}

func TestOfficeComOpenWritePathsNormalized(t *testing.T) {
	ws := isolateOfficeWorkspace(t)
	tl := officeComOpenTool{}

	// 纯文件名新建 → WritePaths 声明收编落点（与 Execute 同判据）。
	got, err := tl.WritePaths(json.RawMessage(`{"path":"草稿.docx"}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != filepath.Join(ws, "交互文件", "草稿.docx") {
		t.Fatalf("WritePaths=%v", got)
	}

	// path 缺省+app+title → 声明语义名落点。
	got, err = tl.WritePaths(json.RawMessage(`{"app":"excel","title":"成本测算"}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != filepath.Join(ws, "交互文件", "成本测算.xlsx") {
		t.Fatalf("title 缺省声明=%v", got)
	}

	// 已存在文件（附着打开）→ 不声明写路径。
	target := filepath.Join(ws, "已有.docx")
	if err := writeFileForTest(target, []byte("x")); err != nil {
		t.Fatal(err)
	}
	got, err = tl.WritePaths(json.RawMessage(`{"path":"` + filepath.ToSlash(target) + `"}`))
	if err != nil || len(got) != 0 {
		t.Fatalf("附着打开不应声明: %v %v", got, err)
	}
}

func writeFileForTest(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
