package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 实测反馈③：长文档混合采样对「精确核对数字」的任务危险（此前用户被迫
// awk/sed 分段精读十余次）。section 参数按标题结构整节返回。

const sectionSampleMD = `# 投标文件

## 1 报价
总价 100 万元。

### 1.1 分项
A 60 万。

### 1.2 汇总
B 40 万。

## 2 风险评估
风险值 0.37。

# 附录
附表略。
`

func sectionFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "report.md")
	if err := os.WriteFile(p, []byte(sectionSampleMD), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestReadFileSection_NumberSelector(t *testing.T) {
	p := sectionFile(t)
	r := readFile{workDir: filepath.Dir(p)}
	out, err := r.Execute(context.Background(), json.RawMessage(
		`{"path":"report.md","section":"1.1"}`))
	if err != nil {
		t.Fatal(err)
	}
	// 命中 ### 1.1，止于下一个同级 ### 1.2
	if !strings.Contains(out, "1.1 分项") || !strings.Contains(out, "A 60 万") {
		t.Fatalf("节内容缺失: %s", out)
	}
	if strings.Contains(out, "B 40 万") || strings.Contains(out, "风险评估") {
		t.Fatalf("越过节边界: %s", out)
	}
	// 绝对行号精确（标题行在第 7 行）
	if !strings.Contains(out, "7→") {
		t.Fatalf("行号应从 7 起: %s", out)
	}
}

func TestReadFileSection_LevelSemantics(t *testing.T) {
	p := sectionFile(t)
	r := readFile{workDir: filepath.Dir(p)}
	// 选 ## 1 应含其下的 ### 1.1/1.2 子节，止于同级 ## 2
	out, err := r.Execute(context.Background(), json.RawMessage(
		`{"path":"report.md","section":"1 报价"}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"总价 100 万元", "A 60 万", "B 40 万"} {
		if !strings.Contains(out, want) {
			t.Fatalf("## 节应含全部子节内容，缺 %q: %s", want, out)
		}
	}
	if strings.Contains(out, "风险值") {
		t.Fatalf("## 节不应吞同级下一节: %s", out)
	}
}

func TestReadFileSection_NoMatchListsOutline(t *testing.T) {
	p := sectionFile(t)
	r := readFile{workDir: filepath.Dir(p)}
	_, err := r.Execute(context.Background(), json.RawMessage(
		`{"path":"report.md","section":"不存在的节"}`))
	if err == nil {
		t.Fatal("未命中应报错")
	}
	// 错误回显标题大纲（一次调用拿结构）
	for _, want := range []string{"未命中", "风险评估", "附录"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("大纲缺 %q: %v", want, err)
		}
	}
}

func TestReadFileSection_PaginationAndGuards(t *testing.T) {
	p := sectionFile(t)
	r := readFile{workDir: filepath.Dir(p)}
	// 节内翻页：offset=1 跳过标题行
	out, err := r.Execute(context.Background(), json.RawMessage(
		`{"path":"report.md","section":"1.1","offset":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "1.1 分项") {
		t.Fatalf("offset=1 应跳过标题行: %s", out)
	}
	// limit 触发节内续读提示（相对节内 offset）
	out, err = r.Execute(context.Background(), json.RawMessage(
		`{"path":"report.md","section":"1 报价","limit":2}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "relative to section start") {
		t.Fatalf("续读提示应说明节内相对语义: %s", out)
	}
	// 负 offset 组合拒绝
	if _, err := r.Execute(context.Background(), json.RawMessage(
		`{"path":"report.md","section":"1.1","offset":-3}`)); err == nil ||
		!strings.Contains(err.Error(), "负 offset") {
		t.Fatalf("负 offset 组合应拒绝: %v", err)
	}
	// 非结构化文档（无标题）错误给大纲指引
	np := filepath.Join(filepath.Dir(p), "plain.txt")
	os.WriteFile(np, []byte("no headings here"), 0o644)
	_, err = r.Execute(context.Background(), json.RawMessage(
		`{"path":"plain.txt","section":"x"}`))
	if err == nil || !strings.Contains(err.Error(), "offset/limit") {
		t.Fatalf("无标题文件应指引 offset/limit: %v", err)
	}
}
