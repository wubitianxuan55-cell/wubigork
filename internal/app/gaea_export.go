package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/gaea/wspath"
)

// deliverableNameRe 是交付物文件名非法字符正则（刀E v4.249 普查#20：包级）。
var deliverableNameRe = regexp.MustCompile(`[\\/:*?"<>|]+`)

// ExportDeliverableInput 是统一交付出口的入参：受控 Markdown + 目标格式。
type ExportDeliverableInput struct {
	Markdown string `json:"markdown"`
	Format   string `json:"format"` // docx | pptx | xlsx | md | pdf
	Title    string `json:"title"`
	Template string `json:"template"` // docx 模板：通用 | 公文 | 报告 | 合同 | 标书
	Cover    bool   `json:"cover"`
	TOC      bool   `json:"toc"`
	Header   string `json:"header"`
	Footer   string `json:"footer"`
	// 标书封面五要素（2026-11-21 反馈 P0-2）：Subtitle=包号、Author=投标人、
	// BidCopy=正本/副本标记、BidDate=落款日期（create_docx.py 标书主题消费）。
	Subtitle string `json:"subtitle"`
	Author   string `json:"author"`
	BidCopy  string `json:"bid_copy"`
	BidDate  string `json:"bid_date"`
	// TocEntries 静态目录条目（P0-2 建议③）：非空时代替 TOC 域渲染静态目录
	// （页码由 longdoc_build 两遍法回填）；md/pptx/xlsx 忽略。
	TocEntries []ExportTocEntry `json:"toc_entries,omitempty"`
	// HeadingNumbering 标题/图表编号联动开关（2026-11-24）：""=auto（docx 模板
	// 缺省，公文=字面）；"off"=全字面（文学成册——「序章」这类无编号章名被挂
	// 「第1章」是内容损伤）；"on"=强制标准三层编号。透传 create_docx.py。
	HeadingNumbering string `json:"heading_numbering,omitempty"`
	// OutputPath 声明写路径（工作区相对路径；P1a 结构化声明入口，2026-11-09
	// 众神殿裁决 action-3）：agent 侧 office_generate 必填，验收=声明路径与
	// 实际落点逐字节一致。空=历史默认（ExportsDir+时间戳命名），UI 绑定契约
	// 不变——默认分支让位方式：显式声明优先，缺省走旧路径。
	OutputPath string `json:"output_path"`
}

// ExportDeliverableResult 是交付结果（工作区相对路径）。
type ExportDeliverableResult struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	Format string `json:"format"`
	Size   int64  `json:"size"`
}

// ExportTocEntry 静态目录条目（P0-2：页码由渲染实测回填，占位「—」）。
type ExportTocEntry struct {
	Text  string `json:"text"`
	Level int    `json:"level"`
	Page  string `json:"page"`
}

// headingRe 从 markdown 抽首个标题行（交付名与 --title 源）。(?m) 必须带：
// Go 的 $ 无多行标志只匹配全文末尾，多行 markdown 的首个标题永远落不进
// 单行锚——2026-11-24 编号联动改造中暴露的存量 bug（title 恒=「gaea 交付物」
// 兜底，连带封面题名/缺省交付名全错）。
var headingRe = regexp.MustCompile(`(?m)^#{1,3}\s+(.+)$`)

// GaeaExportDeliverable 统一交付出口：事实底座/对话成果的受控 Markdown →
// docx / pptx / xlsx / md / pdf，一稿多用、多形态一致。
func (a *App) GaeaExportDeliverable(in ExportDeliverableInput) (ExportDeliverableResult, error) {
	res, _, err := a.gaeaExportDeliverable(in)
	return res, err
}

// gaeaExportDeliverable 是 GaeaExportDeliverable 的内核，另返回降级备注（本机
// create_docx.py 是旧版时哪些能力没生效；空串=没降级）。绑定面契约不变，备注由
// 工具层（longdoc_build）写进回执醒目位。
func (a *App) gaeaExportDeliverable(in ExportDeliverableInput) (ExportDeliverableResult, string, error) {
	if strings.TrimSpace(in.Markdown) == "" {
		return ExportDeliverableResult{}, "", fmt.Errorf("交付内容为空")
	}
	format := strings.ToLower(strings.TrimPrefix(in.Format, "."))
	switch format {
	case "docx", "pptx", "xlsx", "md", "pdf":
	default:
		return ExportDeliverableResult{}, "", fmt.Errorf("不支持的交付格式 %q（docx/pptx/xlsx/md/pdf）", in.Format)
	}
	if in.Template == "" {
		in.Template = "通用"
	}
	if !docxTemplateAllowed(in.Template) {
		return ExportDeliverableResult{}, "", fmt.Errorf("不支持的模板 %q（内置：通用/公文/报告/合同/标书；或已登记格式模板名——工作区 .gaea/templates/docx/<名>.json，make_template.py 登记）", in.Template)
	}

	title := strings.TrimSpace(in.Title)
	if title == "" {
		if m := headingRe.FindStringSubmatch(in.Markdown); m != nil {
			title = strings.TrimSpace(m[1])
		} else {
			title = "gaea 交付物"
		}
	}
	if in.Footer == "" {
		in.Footer = "第 {page} 页"
	}

	// 输出路径双分支（P1a 声明入口）：OutputPath 非空=声明路径（工作区内校验 +
	// 扩展名与 format 一致闸，声明路径=实际落点逐字节一致）；空=历史默认分支
	// （S4 产物路径分区：work 恒 .gaea/exports 现状不动，play 落 .gaea/play/exports）。
	var outPath string
	if strings.TrimSpace(in.OutputPath) != "" {
		// 声明写路径宽容解析（P2-11）：工作区内绝对路径自动转相对（工作区外
		// 仍拒）；相对口径逐字节不变。
		p, perr := wspath.RelIfWithin(gaeaCwd(), in.OutputPath)
		if perr != nil {
			return ExportDeliverableResult{}, "", fmt.Errorf("非法 output_path（工作区相对或工作区内绝对路径）: %s", in.OutputPath)
		}
		if ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(p), ".")); ext != format {
			return ExportDeliverableResult{}, "", fmt.Errorf("output_path 扩展名 %q 与 format %q 不一致", ext, format)
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return ExportDeliverableResult{}, "", err
		}
		outPath = p
	} else {
		exportsDir := spaces.ExportsDir(gaeaCwd(), gaeaEffectiveSpace())
		if err := os.MkdirAll(exportsDir, 0o755); err != nil {
			return ExportDeliverableResult{}, "", err
		}
		stamp := time.Now().Format("20060102-150405")
		base := safeDeliverableName(title)
		outPath = filepath.Join(exportsDir, base+"-"+stamp+"."+format)
	}

	degrade := ""
	switch format {
	case "md":
		if err := os.WriteFile(outPath, []byte(in.Markdown), 0o644); err != nil {
			return ExportDeliverableResult{}, "", err
		}
	case "docx":
		note, err := exportDocxNotes(in, title, outPath)
		if err != nil {
			return ExportDeliverableResult{}, "", err
		}
		degrade = note
	case "pptx":
		if err := exportPptx(in, title, outPath); err != nil {
			return ExportDeliverableResult{}, "", err
		}
	case "xlsx":
		if err := exportXlsx(in, outPath); err != nil {
			return ExportDeliverableResult{}, "", err
		}
	case "pdf":
		// Markdown → docx（临时，复用交付模板链路）→ LibreOffice PDF
		if err := markdownToPdf(in, title, outPath); err != nil {
			return ExportDeliverableResult{}, "", err
		}
	}

	info, err := os.Stat(outPath)
	if err != nil {
		return ExportDeliverableResult{}, "", err
	}
	rel, _ := filepath.Rel(gaeaCwd(), outPath)
	return ExportDeliverableResult{
		Path:   filepath.ToSlash(rel),
		Name:   filepath.Base(outPath),
		Format: format,
		Size:   info.Size(),
	}, degrade, nil
}

// resolveDocxTemplateIn 登记格式模板解析（2026-11-24 方案格式模板线）：同名
// 模板 工作区 .gaea/templates/docx/ 优先（项目自有格式），其次技能自带
// format_templates/（随仓分发的种子模板），都不在=空（走内置五模板名）。
// base 抽出便于测试。
func resolveDocxTemplateIn(base, name string) string {
	n := strings.TrimSpace(name)
	if n == "" {
		return ""
	}
	rel := []string{filepath.Join(".gaea", "templates", "docx", n+".json"),
		filepath.Join(".gaea", "templates", "docx", n),
		filepath.Join(".gaea", "skills", "docx", "scripts", "format_templates", n+".json"),
		filepath.Join(".gaea", "skills", "docx", "scripts", "format_templates", n)}
	for _, r := range rel {
		c := filepath.Join(base, r)
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c
		}
	}
	return ""
}

func resolveDocxTemplate(name string) string {
	return resolveDocxTemplateIn(gaeaCwd(), name)
}

// docxTemplateAllowed 内置五模板名或已登记格式模板（工作区/技能 format_templates）。
func docxTemplateAllowed(name string) bool {
	switch name {
	case "通用", "公文", "报告", "合同", "标书":
		return true
	}
	return resolveDocxTemplate(name) != ""
}

// exportDocx 经 create_docx.py（python-docx）从 Markdown 排版 Word。
func exportDocx(in ExportDeliverableInput, title, outPath string) error {
	_, err := exportDocxNotes(in, title, outPath)
	return err
}

// officeCreateDocxScript 本工作区实际会用到的 create_docx.py（空=没找到）。
func officeCreateDocxScript() string { return findSkillScript("docx", "scripts", "create_docx.py") }

// exportDocxNotes 同 exportDocx，并返回降级备注（空串=没降级）：本机脚本不支持的
// 扩展参数不发——发了就是 argparse 报错整单失败（2026-11-28 黄甲反馈：封面参数
// 让 longdoc_build 硬失败，目录两遍法 pass-1 报错后回落成空 TOC 域）。
func exportDocxNotes(in ExportDeliverableInput, title, outPath string) (string, error) {
	script := officeCreateDocxScript()
	if script == "" {
		return "", fmt.Errorf("未找到 create_docx.py（.gaea/skills/docx/scripts）")
	}
	// 中间 md 用随机临时名（2026-11-21 复盘 A1/A2：旧实现落 <输出stem>.md 再删，
	// 输出名与章节源 md 同 stem 时先覆写后删除，源文件 11 次丢失——中间文件只许
	// 是自己创建的临时件，绝不允许与任何既有文件同名）。落输出同目录：相对图片
	// 路径仍按输出目录解析，行为不变。
	tmp, err := os.CreateTemp(filepath.Dir(outPath), ".gaea-tmp-md-*.md")
	if err != nil {
		return "", err
	}
	mdPath := tmp.Name()
	if _, err := tmp.Write([]byte(in.Markdown)); err != nil {
		_ = tmp.Close()
		_ = os.Remove(mdPath)
		return "", err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(mdPath)
		return "", err
	}
	defer func() { _ = os.Remove(mdPath) }()

	args := []string{script, mdPath, outPath, "--title", title}
	if in.Cover {
		args = append(args, "--cover")
	}
	if in.TOC {
		args = append(args, "--toc")
	}
	if in.Header != "" {
		args = append(args, "--header", in.Header)
	}
	if in.Footer != "" {
		args = append(args, "--footer", in.Footer)
	}
	// 扩展参数按脚本能力发（见 docx_script_caps.go）；没发的记进降级备注。
	var skipped []string
	want := func(flag string) bool {
		if docxScriptSupports(script, flag) {
			return true
		}
		skipped = append(skipped, flag)
		return false
	}
	if in.Subtitle != "" && want("--subtitle") {
		args = append(args, "--subtitle", in.Subtitle)
	}
	if in.Author != "" && want("--author") {
		args = append(args, "--author", in.Author)
	}
	if in.BidCopy != "" && want("--bid-copy") {
		args = append(args, "--bid-copy", in.BidCopy)
	}
	if in.BidDate != "" && want("--bid-date") {
		args = append(args, "--bid-date", in.BidDate)
	}
	if len(in.TocEntries) > 0 && want("--toc-spec") {
		spec, serr := json.Marshal(in.TocEntries)
		if serr != nil {
			return "", serr
		}
		tocTmp, terr := os.CreateTemp(filepath.Dir(outPath), ".gaea-tmp-toc-*.json")
		if terr != nil {
			return "", terr
		}
		tocPath := tocTmp.Name()
		if _, werr := tocTmp.Write(spec); werr != nil {
			_ = tocTmp.Close()
			_ = os.Remove(tocPath)
			return "", werr
		}
		if cerr := tocTmp.Close(); cerr != nil {
			_ = os.Remove(tocPath)
			return "", cerr
		}
		defer func() { _ = os.Remove(tocPath) }()
		args = append(args, "--toc-spec", tocPath)
	}
	// 登记格式模板（工作区/技能 format_templates 命中）→ --template-file；未命中、
	// 或脚本太旧不认 → 内置模板名（登记名不是内置名时退回「通用」）。
	if tf := resolveDocxTemplate(in.Template); tf != "" && want("--template-file") {
		args = append(args, "--template-file", tf)
	} else {
		args = append(args, "--template", builtinDocxTemplateName(in.Template))
	}
	if hn := strings.TrimSpace(in.HeadingNumbering); hn != "" && want("--heading-numbering") {
		args = append(args, "--heading-numbering", hn)
	}
	if err := runPython(args, 120); err != nil {
		return "", fmt.Errorf("Word 导出失败: %w", err)
	}
	return docxScriptDegradeNote(script, skipped), nil
}

// builtinDocxTemplateName 给只认内置模板名的旧脚本用的模板名：登记的格式模板名
// 不是内置五模板之一时退回「通用」。
func builtinDocxTemplateName(name string) string {
	switch name {
	case "通用", "公文", "报告", "合同", "标书":
		return name
	}
	return "通用"
}

// exportPptx 把 Markdown 转成简单 slides spec，经 create_pptx.py 出 16:9 演示文稿。
func exportPptx(in ExportDeliverableInput, title, outPath string) error {
	script := findSkillScript("pptx", "scripts", "create_pptx.py")
	if script == "" {
		return fmt.Errorf("未找到 create_pptx.py（.gaea/skills/pptx/scripts）")
	}
	spec := markdownToSlides(in.Markdown, title)
	// 同 A1/A2：中间 spec 用随机临时名，不再落 <输出stem>.json（与既有同名文件互踩）。
	tmp, err := os.CreateTemp(filepath.Dir(outPath), ".gaea-tmp-spec-*.json")
	if err != nil {
		return err
	}
	specPath := tmp.Name()
	b, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(specPath)
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		_ = os.Remove(specPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(specPath)
		return err
	}
	defer func() { _ = os.Remove(specPath) }()
	if err := runPython([]string{script, specPath, outPath}, 120); err != nil {
		return fmt.Errorf("PPT 导出失败: %w", err)
	}
	return nil
}

// markdownToSlides 简单转换：# 开头开启新幻灯片，- / 数字列表与段落作为要点。
func markdownToSlides(md, title string) map[string]interface{} {
	type slide struct {
		Title  string        `json:"title"`
		Points []interface{} `json:"points"`
	}
	slides := []slide{}
	var cur *slide
	flush := func() {
		if cur != nil {
			if cur.Points == nil {
				cur.Points = []interface{}{}
			}
			if len(cur.Points) > 0 || cur.Title != "" {
				slides = append(slides, *cur)
			}
		}
	}
	for _, line := range strings.Split(md, "\n") {
		line = strings.TrimRight(line, " \t")
		if line == "" {
			continue
		}
		if m := headingRe.FindStringSubmatch(line); m != nil {
			flush()
			cur = &slide{Title: strings.TrimSpace(m[1]), Points: []interface{}{}}
			continue
		}
		if strings.HasPrefix(line, "|") {
			// 表格行并入要点（去管道符与分隔行）
			if strings.Contains(line, "---") {
				continue
			}
			cells := strings.Split(strings.Trim(line, "|"), "|")
			for i := range cells {
				cells[i] = strings.TrimSpace(cells[i])
			}
			text := strings.Join(cells, " ｜ ")
			if text != "" {
				if cur == nil {
					cur = &slide{Points: []interface{}{}}
				}
				cur.Points = append(cur.Points, text)
			}
			continue
		}
		text := strings.TrimSpace(line)
		text = strings.TrimLeft(text, "-*123456789. ")
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		if cur == nil {
			cur = &slide{Points: []interface{}{}}
		}
		cur.Points = append(cur.Points, text)
	}
	flush()
	if len(slides) == 0 {
		slides = append(slides, slide{Title: title})
	}
	return map[string]interface{}{
		"title":    title,
		"subtitle": "由 gaea 基于事实底座生成",
		"author":   "gaea",
		"slides":   slides,
	}
}

// exportXlsx 把 Markdown 中的表格提取为工作表（无表格时正文写入 Sheet1）。
func exportXlsx(in ExportDeliverableInput, outPath string) error {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	_ = f.SetSheetName("Sheet1", "摘要")
	_ = f.SetCellValue("摘要", "A1", firstLine(in.Title))

	tableIdx := 0
	var curTable []string
	flushTable := func() {
		if len(curTable) == 0 {
			return
		}
		tableIdx++
		sheet := "表" + strconv.Itoa(tableIdx)
		if tableIdx == 1 {
			sheet = "数据"
			_ = f.SetSheetName("摘要", sheet)
		} else {
			_, _ = f.NewSheet(sheet)
		}
		for ri, line := range curTable {
			line = strings.Trim(line, "|")
			cells := strings.Split(line, "|")
			for ci, c := range cells {
				axis, _ := excelize.CoordinatesToCellName(ci+1, ri+1)
				_ = f.SetCellValue(sheet, axis, strings.TrimSpace(c))
			}
		}
		curTable = nil
	}

	bodyRow := 2
	for _, line := range strings.Split(in.Markdown, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "|") {
			if strings.Contains(line, "---") {
				continue
			}
			curTable = append(curTable, line)
			continue
		}
		flushTable()
		text := strings.TrimSpace(line)
		if text != "" && !strings.HasPrefix(text, "#") {
			_ = f.SetCellValue("数据", "A"+strconv.Itoa(bodyRow), text)
			bodyRow++
		}
	}
	flushTable()
	if tableIdx == 0 {
		_ = f.SetSheetName("摘要", "数据")
	}
	return f.SaveAs(outPath)
}

func runPython(args []string, timeoutSec int) error {
	return runProcess("python", args, timeoutSec)
}

// findSkillScript 定位技能脚本（工作区 .gaea/skills 或用户 skills 镜像）。
func findSkillScript(skill, sub, name string) string {
	candidates := []string{
		filepath.Join(gaeaCwd(), ".gaea", "skills", skill, sub, name),
		filepath.Join(gaeaCwd(), "skills", skill, sub, name),
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".codex", "skills", skill, sub, name),
			filepath.Join(home, ".gaea", "skills", skill, sub, name),
		)
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c
		}
	}
	return ""
}

// safeDeliverableName 生成交付物文件名（保留中文，去掉非法路径字符）。
func safeDeliverableName(s string) string {
	re := deliverableNameRe
	s = re.ReplaceAllString(s, "-")
	s = strings.TrimSpace(s)
	if len(s) > 60 {
		s = s[:60]
	}
	if s == "" {
		s = "deliverable"
	}
	return s
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}
