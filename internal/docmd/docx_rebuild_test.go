package docmd

// docx_rebuild_test.go — 重建器单测与脏 docx 集成测（夹具在测试内构造最小 OOXML）。
// 口径对齐沿江包1 复盘：合并网格展开列数恒一致、折行语义重拼、目录域/残表剔除、
// 伪表格转图片引用、结构体检指标。

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testDocNS = `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" ` +
	`xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" ` +
	`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` +
	`<w:body>%s</w:body></w:document>`

const testRelsXML = `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
	`<Relationship Id="rId1" Target="media/image1.png"/></Relationships>`

var testPNG = []byte{
	0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 13, 'I', 'H', 'D', 'R',
}

// writeTestDocx 落一个最小 docx（只有 document.xml / rels / media，解析器够用）。
func writeTestDocx(t *testing.T, body string, withImage bool) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	mustWrite := func(name, data string) {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(data))
	}
	mustWrite("word/document.xml", fmt.Sprintf(testDocNS, body))
	mustWrite("word/_rels/document.xml.rels", testRelsXML)
	if withImage {
		w, err := zw.Create("word/media/image1.png")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(testPNG); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "dirty.docx")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// foldCell 造一个「视觉折行」单元格：两个独立 w:p。
func foldCell(lines ...string) string {
	out := "<w:tc><w:tcPr/><w:tbl>"
	_ = out
	var b strings.Builder
	b.WriteString("<w:tc>")
	for _, l := range lines {
		b.WriteString("<w:p><w:r><w:t>" + l + "</w:t></w:r></w:p>")
	}
	b.WriteString("</w:tc>")
	return b.String()
}

func TestRebuildSmartJoin(t *testing.T) {
	cases := []struct {
		name  string
		lines []string
		want  string
	}{
		// 断词：中文接中文直接相连（复盘 §2.1 的原句）。
		{"断词接续", []string{"按照相关技术导", "则及生态环境行政主管部门要"}, "按照相关技术导则及生态环境行政主管部门要"},
		// ASCII 边界补空格。
		{"ASCII 边界", []string{"采用 GB", "50493 标准"}, "采用 GB 50493 标准"},
		// 列表项分行。
		{"列表项分行", []string{"准备工作如下", "（1）场地平整", "（2）临建搭设"}, "准备工作如下<br>（1）场地平整<br>（2）临建搭设"},
		// 句末标点后分行。
		{"句末分行", []string{"符合规范要求。", "检测频率为每天一次。"}, "符合规范要求。<br>检测频率为每天一次。"},
		// 空段剔除。
		{"空段剔除", []string{"", "内容", "", ""}, "内容"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := smartJoin(c.lines); got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestRebuildTableGridExpansion(t *testing.T) {
	// 4 网格列：表头带 gridSpan=2，数据行带折行碎片与 vMerge。
	body := "<w:tbl>" +
		"<w:tblGrid><w:gridCol/><w:gridCol/><w:gridCol/><w:gridCol/></w:tblGrid>" +
		"<w:tr>" +
		"<w:tc><w:tcPr/><w:p><w:r><w:t>类别</w:t></w:r></w:p></w:tc>" +
		"<w:tc><w:tcPr><w:gridSpan w:val=\"2\"/></w:tcPr><w:p><w:r><w:t>指标</w:t></w:r></w:p></w:tc>" +
		"<w:tc><w:tcPr/><w:p><w:r><w:t>备注</w:t></w:r></w:p></w:tc>" +
		"</w:tr>" +
		"<w:tr>" +
		foldCell("土壤", "修复") +
		foldCell("按照相关技术导", "则及生态环境行政主管部门要") +
		"<w:tc><w:tcPr/><w:p/></w:tc>" +
		"<w:tc><w:tcPr><w:vMerge w:val=\"restart\"/></w:tcPr><w:p/></w:tc>" +
		"</w:tr>" +
		"<w:tr>" +
		"<w:tc><w:tcPr/><w:p><w:r><w:t>市政</w:t></w:r></w:p></w:tc>" +
		"<w:tc><w:tcPr/><w:p><w:r><w:t>460</w:t></w:r></w:p></w:tc>" +
		"<w:tc><w:tcPr/><w:p/></w:tc>" +
		"<w:tc><w:tcPr><w:vMerge/></w:tcPr><w:p/></w:tc>" +
		"</w:tr>" +
		"</w:tbl>"
	path := writeTestDocx(t, body, false)
	md, rep, err := RebuildDocx(path)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Rebuilt {
		t.Fatal("应走重建路线")
	}
	if rep.MergedCells < 3 {
		t.Fatalf("合并网格应识别 ≥3 处（gridSpan + 2 vMerge），得到 %d", rep.MergedCells)
	}
	if rep.FragmentedCells != 2 {
		t.Fatalf("折行碎片单元格应为 2，得到 %d", rep.FragmentedCells)
	}
	if rep.GridRows != 3 || rep.GridConformant != 3 {
		t.Fatalf("展开后 3 行应全列数一致，得到 %d/%d", rep.GridConformant, rep.GridRows)
	}
	if got := rep.GridConformanceRate(); got != 1 {
		t.Fatalf("展开前列数一致率应为 1（夹具行列数与网格一致），得到 %v", got)
	}
	if !strings.Contains(md, "| 类别 | 指标 |  | 备注 |") {
		t.Fatalf("gridSpan 展开应补空列：%s", md)
	}
	if !strings.Contains(md, "按照相关技术导则及生态环境行政主管部门要") {
		t.Fatalf("折行断词应重拼：%s", md)
	}
	if !strings.Contains(md, "| 土壤修复 | 按照相关技术导则及生态环境行政主管部门要 |  |  |") {
		t.Fatalf("数据行应为 4 列：%s", md)
	}
	if strings.Count(md, "\n") < 3 {
		t.Fatalf("应有表头+分隔+2 数据行：%s", md)
	}
}

func TestRebuildDirtyRouteOnInconsistentGrid(t *testing.T) {
	// 源表行不写满网格列（列数矛盾签名）→ 脏文档 → 重建路线 + 一致率 <100%。
	body := "<w:tbl>" +
		"<w:tblGrid><w:gridCol/><w:gridCol/><w:gridCol/></w:tblGrid>" +
		"<w:tr><w:tc><w:tcPr/><w:p><w:r><w:t>甲</w:t></w:r></w:p></w:tc>" +
		"<w:tc><w:tcPr/><w:p><w:r><w:t>乙</w:t></w:r></w:p></w:tc>" +
		"<w:tc><w:tcPr/><w:p><w:r><w:t>丙</w:t></w:r></w:p></w:tc></w:tr>" +
		"<w:tr><w:tc><w:tcPr/><w:p><w:r><w:t>缺格</w:t></w:r></w:p></w:tc></w:tr>" +
		"</w:tbl>"
	path := writeTestDocx(t, body, false)
	rep, err := AnalyzeDocx(path)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.dirtyDocx() {
		t.Fatal("列数与网格不一致应判脏")
	}
	if rep.GridConformant != 1 || rep.GridRows != 2 {
		t.Fatalf("第一行一致、第二行缺格：得到 %d/%d", rep.GridConformant, rep.GridRows)
	}
	if rate := rep.GridConformanceRate(); rate != 0.5 {
		t.Fatalf("列数一致率应为 0.5，得到 %v", rate)
	}
	// 路由：脏文档在 markitdown 之前就被重建接管；输出行恒 3 列。
	md, rep2, err := RebuildDocx(path)
	if err != nil {
		t.Fatal(err)
	}
	if !rep2.Rebuilt {
		t.Fatal("应走重建")
	}
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(line, "|") && !strings.HasPrefix(line, "| ---") {
			if n := strings.Count(line, "|"); n != 4 { // 3 列 = 4 个竖线
				t.Fatalf("输出行列数应恒为 3（%d 竖线）：%s", n, line)
			}
		}
	}
}

func TestRebuildCatalogTableAndSdtTocSkipped(t *testing.T) {
	tocRows := ""
	for _, r := range []string{"1 工程概况", "2 编制依据", "3 施工方案", "4 质量保证"} {
		tocRows += "<w:tr><w:tc><w:tcPr/><w:p><w:r><w:t>" + r + "</w:t></w:r></w:p></w:tc>" +
			"<w:tc><w:tcPr/><w:p><w:r><w:t>12</w:t></w:r></w:p></w:tc></w:tr>"
	}
	body := "<w:sdt><w:sdtPr/><w:sdtContent>" +
		"<w:p><w:r><w:t>目录</w:t></w:r></w:p>" +
		"<w:p><w:r><w:t>1 工程概况</w:t></w:r></w:p>" +
		"<w:p><w:r><w:t>2 编制依据</w:t></w:r></w:p>" +
		"</w:sdtContent></w:sdt>" +
		"<w:tbl><w:tblGrid><w:gridCol/><w:gridCol/></w:tblGrid>" + tocRows + "</w:tbl>" +
		"<w:p><w:r><w:t>1 总体部署</w:t></w:r></w:p><w:p><w:r><w:t>正文段落保留。</w:t></w:r></w:p>"
	path := writeTestDocx(t, body, false)
	md, rep, err := RebuildDocx(path)
	if err != nil {
		t.Fatal(err)
	}
	if rep.TocSkipped != 1 {
		t.Fatalf("目录域应整块剔除 1，得到 %d", rep.TocSkipped)
	}
	if rep.CatalogSkipped != 1 {
		t.Fatalf("目录残表应剔除 1，得到 %d", rep.CatalogSkipped)
	}
	if rep.Tables != 0 {
		t.Fatalf("不应保留任何表，得到 %d", rep.Tables)
	}
	if strings.Contains(md, "编制依据") && !strings.Contains(md, "1 总体部署") {
		t.Fatal("正文标题被误剔")
	}
	if !strings.Contains(md, "# 1 总体部署") {
		t.Fatalf("编号标题应识别（深度 1）：%s", md)
	}
	if !strings.Contains(md, "正文段落保留。") {
		t.Fatalf("正文段落应保留：%s", md)
	}
}

func TestRebuildHeadingGuards(t *testing.T) {
	body := "<w:p><w:r><w:t>1 ）残形括号不是标题</w:t></w:r></w:p>" +
		"<w:p><w:r><w:t>表 1-2 主要施工设备表</w:t></w:r></w:p>" +
		"<w:p><w:r><w:t>图 3-1 工艺流程图</w:t></w:r></w:p>" +
		"<w:p><w:r><w:t>2.1.1 这是一句很长很长很长很长很长很长很长很长很长很长很长很长的说明文字，带标点</w:t></w:r></w:p>" +
		"<w:p><w:r><w:t>3.2 施工平面布置</w:t></w:r></w:p>"
	path := writeTestDocx(t, body, false)
	md, rep, err := RebuildDocx(path)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Headings != 1 {
		t.Fatalf("只有「3.2 施工平面布置」是标题，得到 %d：%s", rep.Headings, md)
	}
	for _, banned := range []string{"# 1 ）", "# 表 1-2", "# 图 3-1"} {
		if strings.Contains(md, banned) {
			t.Fatalf("不该判成标题：%s", banned)
		}
	}
}

func TestRebuildPseudoTableToImages(t *testing.T) {
	// 1 列表、唯一非空单元格只有图 → 伪表格，转图片引用。
	imageCell := "<w:tc><w:tcPr/><w:p><w:r><w:drawing><a:blip r:embed=\"rId1\"/></w:drawing></w:r></w:p></w:tc>"
	body := "<w:tbl><w:tblGrid><w:gridCol/></w:tblGrid><w:tr>" + imageCell + "</w:tr></w:tbl>"
	path := writeTestDocx(t, body, true)
	md, rep, err := RebuildDocx(path)
	if err != nil {
		t.Fatal(err)
	}
	if rep.PseudoTables != 1 {
		t.Fatalf("应识别 1 张伪表格，得到 %d：%s", rep.PseudoTables, md)
	}
	if rep.Tables != 0 {
		t.Fatalf("伪表格不应保留为表，得到 %d", rep.Tables)
	}
	if !strings.Contains(md, "![](data:image/png;base64,") {
		t.Fatalf("应转成 data URI 图引：%s", md)
	}
	// 对照：同结构但带文字单元格 → 真表格保留。
	body2 := "<w:tbl><w:tblGrid><w:gridCol/><w:gridCol/></w:tblGrid><w:tr>" +
		"<w:tc><w:tcPr/><w:p><w:r><w:t>工艺流程</w:t></w:r></w:p></w:tc>" +
		imageCell + "</w:tr></w:tbl>"
	path2 := writeTestDocx(t, body2, true)
	md2, rep2, err := RebuildDocx(path2)
	if err != nil {
		t.Fatal(err)
	}
	if rep2.Tables != 1 || rep2.PseudoTables != 0 {
		t.Fatalf("带文字单元格应是真表格：%s", md2)
	}
}

func TestConvertDocxSmartRouteDirty(t *testing.T) {
	// convertDocxSmart：脏文档直接被重建接管（不会去碰 markitdown，测试环境确定性）。
	body := "<w:tbl><w:tblGrid><w:gridCol/><w:gridCol/></w:tblGrid>" +
		"<w:tr><w:tc><w:tcPr><w:gridSpan w:val=\"2\"/></w:tcPr><w:p><w:r><w:t>跨两列的表头</w:t></w:r></w:p></w:tc></w:tr>" +
		"<w:tr><w:tc><w:tcPr/><w:p><w:r><w:t>甲</w:t></w:r></w:p></w:tc>" +
		"<w:tc><w:tcPr/><w:p><w:r><w:t>乙</w:t></w:r></w:p></w:tc></w:tr></w:tbl>"
	path := writeTestDocx(t, body, false)
	md, rep, err := convertDocxSmart(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep == nil || !rep.Rebuilt {
		t.Fatal("脏文档应走重建路线并带报告")
	}
	if rep.GridConformanceRate() != 1 {
		t.Fatalf("展开后应全一致，得到 %v", rep.GridConformanceRate())
	}
	if !strings.Contains(md, "跨两列的表头") || !strings.Contains(md, "| --- |") {
		t.Fatalf("产物应是规整表格：%s", md)
	}
}

func TestAnalyzeDocxCleanNotDirty(t *testing.T) {
	// 干净文档（无合并/无碎片/无残表）不该判脏——路由留给 markitdown。
	body := "<w:tbl><w:tblGrid><w:gridCol/><w:gridCol/></w:tblGrid>" +
		"<w:tr><w:tc><w:tcPr/><w:p><w:r><w:t>名称</w:t></w:r></w:p></w:tc>" +
		"<w:tc><w:tcPr/><w:p><w:r><w:t>数量</w:t></w:r></w:p></w:tc></w:tr>" +
		"<w:tr><w:tc><w:tcPr/><w:p><w:r><w:t>挖掘机</w:t></w:r></w:p></w:tc>" +
		"<w:tc><w:tcPr/><w:p><w:r><w:t>2</w:t></w:r></w:p></w:tc></w:tr></w:tbl>"
	path := writeTestDocx(t, body, false)
	rep, err := AnalyzeDocx(path)
	if err != nil {
		t.Fatal(err)
	}
	if rep.dirtyDocx() {
		t.Fatalf("干净表不应判脏：%+v", rep)
	}
	if rep.GridConformanceRate() != 1 {
		t.Fatalf("一致率应为 1，得到 %v", rep.GridConformanceRate())
	}
}
