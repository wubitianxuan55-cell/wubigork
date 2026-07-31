package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wubigork/wubigork/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(projectInit{}) }

// projectInit creates a standardized project directory structure.
type projectInit struct{}

func (projectInit) Name() string { return "project_init" }

func (projectInit) Description() string {
	return "初始化工程项目目录结构（construction/mechanical/electrical/civil）。自动创建 docs/ drawings/ calculations/ specs/ reports/ 等标准子目录及 README。"
}

func (projectInit) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "name":{"type":"string","description":"项目名称"},
  "type":{"type":"string","enum":["construction","mechanical","electrical","civil"],"description":"项目类型"}
},
"required":["name","type"]
}`)
}

func (projectInit) ReadOnly() bool { return false }

func (projectInit) CompactDescription() string     { return compactDesc["project_init"] }
func (projectInit) CompactSchema() json.RawMessage { return compactSchema["project_init"] }

// projectTemplate defines the directory structure for a project type.
type projectTemplate struct {
	Name        string
	Description string
	Dirs        []string
	Files       map[string]string // filename -> content
}

var projectTemplates = map[string]projectTemplate{
	"construction": {
		Name:        "土木/建筑工程",
		Description: "Construction / Civil Engineering Project",
		Dirs: []string{
			"docs", "drawings", "calculations", "specs", "reports",
			"drawings/architectural", "drawings/structural", "drawings/MEP",
			"docs/site_investigation", "docs/permits",
			"calculations/structural", "calculations/geotechnical",
			"specs/materials", "specs/standards",
			"reports/progress", "reports/inspection",
		},
		Files: map[string]string{
			"README.md": `# {{.Name}}

> {{.Type}} 工程项目

## 目录结构

| 目录 | 说明 |
|------|------|
| docs/ | 项目文档（勘察报告、许可证等） |
| drawings/ | 图纸（建筑/结构/机电） |
| calculations/ | 计算书（结构/岩土） |
| specs/ | 技术规格书（材料/标准） |
| reports/ | 项目报告（进度/检测） |

## 项目信息

- **类型**: {{.Type}}
- **创建日期**: {{.Date}}
`,
			"docs/README.md":         "# 项目文档\n\n存放勘察报告、施工许可、环评报告等。\n",
			"drawings/README.md":     "# 图纸目录\n\n- architectural/ — 建筑图\n- structural/ — 结构图\n- MEP/ — 机电图\n",
			"calculations/README.md": "# 计算书\n\n存放结构计算、岩土计算等。\n",
			"specs/README.md":        "# 技术规格书\n\n- materials/ — 材料规格\n- standards/ — 引用标准\n",
			"reports/README.md":      "# 项目报告\n\n- progress/ — 进度报告\n- inspection/ — 检测报告\n",
		},
	},
	"mechanical": {
		Name:        "机械工程",
		Description: "Mechanical Engineering Project",
		Dirs: []string{
			"docs", "drawings", "calculations", "specs", "reports",
			"drawings/assembly", "drawings/parts", "drawings/exploded",
			"docs/requirements", "docs/design_reviews",
			"calculations/strength", "calculations/thermo", "calculations/fluid",
			"specs/materials", "specs/tolerances",
			"reports/test", "reports/prototyping",
			"cad", "bom",
		},
		Files: map[string]string{
			"README.md": `# {{.Name}}

> {{.Type}} 工程项目

## 目录结构

| 目录 | 说明 |
|------|------|
| docs/ | 设计文档（需求/评审） |
| drawings/ | 工程图（装配/零件/爆炸图） |
| cad/ | CAD 模型文件 |
| bom/ | 物料清单 |
| calculations/ | 计算书（强度/热力/流体） |
| specs/ | 技术规格（材料/公差） |
| reports/ | 报告（测试/样机） |

## 项目信息

- **类型**: {{.Type}}
- **创建日期**: {{.Date}}
`,
			"docs/README.md":         "# 设计文档\n\n存放需求文档、设计评审记录等。\n",
			"drawings/README.md":     "# 工程图目录\n\n- assembly/ — 装配图\n- parts/ — 零件图\n- exploded/ — 爆炸图\n",
			"calculations/README.md": "# 计算书\n\n- strength/ — 强度计算\n- thermo/ — 热力计算\n- fluid/ — 流体计算\n",
			"cad/README.md":          "# CAD 模型\n\n存放 SolidWorks / Inventor / Fusion 360 等模型文件。\n",
			"bom/README.md":          "# 物料清单 (BOM)\n\n存放 BOM 表格（Excel / CSV）。\n",
			"reports/README.md":      "# 项目报告\n\n- test/ — 测试报告\n- prototyping/ — 样机报告\n",
		},
	},
	"electrical": {
		Name:        "电气工程",
		Description: "Electrical Engineering Project",
		Dirs: []string{
			"docs", "drawings", "calculations", "specs", "reports",
			"drawings/schematics", "drawings/pcb_layout", "drawings/wiring",
			"docs/requirements", "docs/test_plans",
			"calculations/power", "calculations/signal",
			"specs/components", "specs/standards",
			"reports/test", "reports/compliance",
			"firmware", "simulation",
		},
		Files: map[string]string{
			"README.md": `# {{.Name}}

> {{.Type}} 工程项目

## 目录结构

| 目录 | 说明 |
|------|------|
| docs/ | 设计文档（需求/测试方案） |
| drawings/ | 电气图（原理图/PCB/接线图） |
| firmware/ | 固件源码 |
| simulation/ | 仿真模型 |
| calculations/ | 计算书（功率/信号） |
| specs/ | 规格书（元器件/标准） |
| reports/ | 报告（测试/认证） |

## 项目信息

- **类型**: {{.Type}}
- **创建日期**: {{.Date}}
`,
			"drawings/README.md":     "# 电气图目录\n\n- schematics/ — 原理图\n- pcb_layout/ — PCB 布局\n- wiring/ — 接线图\n",
			"docs/README.md":         "# 设计文档\n\n存放需求说明、测试方案等。\n",
			"calculations/README.md": "# 计算书\n\n- power/ — 功率计算\n- signal/ — 信号完整性分析\n",
			"firmware/README.md":     "# 固件\n\n存放 MCU 固件源码（C/C++/MicroPython 等）。\n",
			"simulation/README.md":   "# 仿真\n\n存放 SPICE / Simulink / LTspice 等仿真模型。\n",
			"reports/README.md":      "# 项目报告\n\n- test/ — 测试报告\n- compliance/ — 合规认证报告\n",
		},
	},
	"civil": {
		Name:        "市政/水利工程",
		Description: "Civil / Municipal / Hydraulic Engineering Project",
		Dirs: []string{
			"docs", "drawings", "calculations", "specs", "reports",
			"drawings/site_plan", "drawings/profiles", "drawings/details",
			"docs/survey", "docs/geotechnical", "docs/environmental",
			"calculations/hydraulic", "calculations/structural", "calculations/geotechnical",
			"specs/materials", "specs/construction",
			"reports/progress", "reports/quality",
		},
		Files: map[string]string{
			"README.md": `# {{.Name}}

> {{.Type}} 工程项目

## 目录结构

| 目录 | 说明 |
|------|------|
| docs/ | 项目文档（勘察/地勘/环评） |
| drawings/ | 图纸（总平面/纵断面/大样） |
| calculations/ | 计算书（水力/结构/岩土） |
| specs/ | 技术规格（材料/施工） |
| reports/ | 报告（进度/质量） |

## 项目信息

- **类型**: {{.Type}}
- **创建日期**: {{.Date}}
`,
			"docs/README.md":         "# 项目文档\n\n存放测量报告、地勘报告、环评报告等。\n",
			"drawings/README.md":     "# 图纸目录\n\n- site_plan/ — 总平面图\n- profiles/ — 纵断面图\n- details/ — 大样图\n",
			"calculations/README.md": "# 计算书\n\n- hydraulic/ — 水力计算\n- structural/ — 结构计算\n- geotechnical/ — 岩土计算\n",
			"reports/README.md":      "# 项目报告\n\n- progress/ — 进度报告\n- quality/ — 质量检测报告\n",
		},
	},
}

func (projectInit) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if p.Name == "" {
		return "", fmt.Errorf("project name is required")
	}
	if p.Type == "" {
		return "", fmt.Errorf("project type is required")
	}

	tmpl, ok := projectTemplates[p.Type]
	if !ok {
		types := make([]string, 0, len(projectTemplates))
		for k := range projectTemplates {
			types = append(types, k)
		}
		return "", fmt.Errorf("unknown project type %q. Supported types: %s", p.Type, strings.Join(types, ", "))
	}

	root := filepath.Clean(p.Name)

	// Create directories
	created := 0
	for _, dir := range tmpl.Dirs {
		fullPath := filepath.Join(root, dir)
		if err := os.MkdirAll(fullPath, 0o755); err != nil {
			return "", fmt.Errorf("create dir %s: %w", fullPath, err)
		}
		created++
	}

	// Create files
	written := 0
	now := time.Now().Format("2006-01-02")
	for path, content := range tmpl.Files {
		fullPath := filepath.Join(root, path)
		// Apply template substitutions
		rendered := strings.ReplaceAll(content, "{{.Name}}", p.Name)
		rendered = strings.ReplaceAll(rendered, "{{.Type}}", tmpl.Name)
		rendered = strings.ReplaceAll(rendered, "{{.Date}}", now)

		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("create dir %s: %w", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(rendered), 0o644); err != nil {
			return "", fmt.Errorf("write %s: %w", fullPath, err)
		}
		written++
	}

	return fmt.Sprintf("项目 %q 初始化完成（类型: %s）\n- 创建目录: %d 个\n- 创建文件: %d 个", p.Name, tmpl.Name, created, written), nil
}
