package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/office/crosslink"
)

// CrossEmbedInput 是跨应用联动入参：xlsx 数据 → 图表 → 嵌入 docx/pptx。
type CrossEmbedInput struct {
	XlsxRel   string `json:"xlsxRel"`
	Sheet     string `json:"sheet"`
	Range     string `json:"range"`     // 可选 "A1:B6"；空 = 自动
	ChartType string `json:"chartType"` // bar | line | pie | scatter
	Title     string `json:"title"`
	Into      string `json:"into"`   // docx | pptx
	Output    string `json:"output"` // 可选输出相对路径
}

// CrossEmbedResult 是联动产物（工作区相对路径 + 生成的图表）。
type CrossEmbedResult struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	ChartPath string `json:"chartPath"`
}

// GaeaCrossEmbed 跨应用联动：从 xlsx 提取数据 → matplotlib 图表 →
// 嵌入 docx（报告模板：标题+图表+数据表）或 pptx（图表页+数据明细页）。
// 数据源更新后重新导出即刷新图表，实现「图表与数据保持同步」。
func (a *App) GaeaCrossEmbed(in CrossEmbedInput) (CrossEmbedResult, error) {
	if in.XlsxRel == "" {
		return CrossEmbedResult{}, fmt.Errorf("缺少 xlsx 数据源路径")
	}
	into := strings.ToLower(in.Into)
	switch into {
	case "docx", "pptx":
	default:
		return CrossEmbedResult{}, fmt.Errorf("不支持的嵌入目标 %q（docx/pptx）", in.Into)
	}
	if in.ChartType == "" {
		in.ChartType = "bar"
	}
	switch in.ChartType {
	case "bar", "line", "pie", "scatter":
	default:
		return CrossEmbedResult{}, fmt.Errorf("不支持的图表类型 %q（bar/line/pie/scatter）", in.ChartType)
	}

	xlsxPath := in.XlsxRel
	if !filepath.IsAbs(in.XlsxRel) {
		xlsxPath = filepath.Join(gaeaCwd(), in.XlsxRel)
	}
	if _, err := os.Stat(xlsxPath); err != nil {
		return CrossEmbedResult{}, fmt.Errorf("xlsx 不存在：%s", in.XlsxRel)
	}

	labels, values, header, rows, err := crosslink.ExtractChartData(xlsxPath, in.Sheet, in.Range)
	if err != nil {
		return CrossEmbedResult{}, err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(in.XlsxRel), filepath.Ext(in.XlsxRel))
	}

	exportsDir := filepath.Join(gaeaCwd(), ".gaea", "exports")
	if err := os.MkdirAll(exportsDir, 0o755); err != nil {
		return CrossEmbedResult{}, err
	}
	stamp := time.Now().Format("20060102-150405")
	base := safeDeliverableName(title)
	chartPng := filepath.Join(exportsDir, base+"-chart-"+stamp+".png")
	if err := crosslink.GenerateChartPNG(labels, values, in.ChartType, title, chartPng); err != nil {
		return CrossEmbedResult{}, err
	}

	outPath := in.Output
	if outPath == "" {
		outPath = filepath.Join(exportsDir, base+"-"+stamp+"."+into)
	} else if !filepath.IsAbs(outPath) {
		outPath = filepath.Join(gaeaCwd(), outPath)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return CrossEmbedResult{}, err
	}

	switch into {
	case "docx":
		if err := embedDocxWithChart(title, chartPng, header, rows, outPath); err != nil {
			return CrossEmbedResult{}, err
		}
	case "pptx":
		if err := embedPptxWithChart(title, chartPng, header, rows, outPath); err != nil {
			return CrossEmbedResult{}, err
		}
	}

	info, err := os.Stat(outPath)
	if err != nil {
		return CrossEmbedResult{}, err
	}
	rel, _ := filepath.Rel(gaeaCwd(), outPath)
	chartRel, _ := filepath.Rel(gaeaCwd(), chartPng)
	return CrossEmbedResult{
		Path:      filepath.ToSlash(rel),
		Name:      filepath.Base(outPath),
		Size:      info.Size(),
		ChartPath: filepath.ToSlash(chartRel),
	}, nil
}

func embedDocxWithChart(title, chartPng string, header []string, rows [][]string, outPath string) error {
	script := findSkillScript("docx", "scripts", "create_docx.py")
	if script == "" {
		return fmt.Errorf("未找到 create_docx.py（.gaea/skills/docx/scripts）")
	}
	spec := crosslink.BuildDocxSpec(title, chartPng, header, rows)
	specPath := strings.TrimSuffix(outPath, ".docx") + ".json"
	b, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	if err := os.WriteFile(specPath, b, 0o644); err != nil {
		return err
	}
	defer os.Remove(specPath)
	return runPython([]string{script, specPath, outPath, "--spec"}, 120)
}

func embedPptxWithChart(title, chartPng string, header []string, rows [][]string, outPath string) error {
	script := findSkillScript("pptx", "scripts", "create_pptx.py")
	if script == "" {
		return fmt.Errorf("未找到 create_pptx.py（.gaea/skills/pptx/scripts）")
	}
	spec := crosslink.BuildPptxSpec(title, chartPng, header, rows)
	specPath := strings.TrimSuffix(outPath, ".pptx") + ".json"
	b, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	if err := os.WriteFile(specPath, b, 0o644); err != nil {
		return err
	}
	defer os.Remove(specPath)
	return runPython([]string{script, specPath, outPath}, 120)
}
