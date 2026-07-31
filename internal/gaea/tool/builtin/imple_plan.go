package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(implePlan{}) }

type implePlan struct{}

func (implePlan) Name() string { return "imple_plan" }

func (implePlan) Description() string {
	return "生成土壤修复实施方案框架：输入修复目标、技术路线、地块条件，输出施工方案（工艺参数、设备选型、进度计划、二次污染防控）。"
}

func (implePlan) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "project_name":{"type":"string","description":"项目名称"},
  "site_address":{"type":"string","description":"项目地址"},
  "contaminants":{"type":"array","items":{"type":"string"},"description":"主要污染物"},
  "soil_volume":{"type":"number","description":"修复土方量(m3)"},
  "repair_area":{"type":"number","description":"修复面积(m2)"},
  "technology":{"type":"string","description":"修复技术，如原位化学氧化、固化/稳定化、SVE等"},
  "target_value":{"type":"string","description":"修复目标值"},
  "construction_period":{"type":"string","description":"施工工期"},
  "unit_name":{"type":"string","description":"施工单位名称"},
  "groundwater":{"type":"boolean","description":"是否包含地下水修复","default":false},
  "depth_range":{"type":"string","description":"修复深度范围"}
},
"required":["project_name","technology"]
}`)
}

func (implePlan) ReadOnly() bool { return true }

func (implePlan) CompactDescription() string     { return compactDesc["imple_plan"] }
func (implePlan) CompactSchema() json.RawMessage { return compactSchema["imple_plan"] }

type impleInput struct {
	ProjectName        string   `json:"project_name"`
	SiteAddress        string   `json:"site_address,omitempty"`
	Contaminants       []string `json:"contaminants,omitempty"`
	SoilVolume         float64  `json:"soil_volume,omitempty"`
	RepairArea         float64  `json:"repair_area,omitempty"`
	Technology         string   `json:"technology"`
	TargetValue        string   `json:"target_value,omitempty"`
	ConstructionPeriod string   `json:"construction_period,omitempty"`
	UnitName           string   `json:"unit_name,omitempty"`
	Groundwater        bool     `json:"groundwater,omitempty"`
	DepthRange         string   `json:"depth_range,omitempty"`
}

func (implePlan) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p impleInput
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	p.ProjectName = strings.TrimSpace(p.ProjectName)
	p.Technology = strings.TrimSpace(p.Technology)
	if p.ProjectName == "" {
		return "", fmt.Errorf("project_name 不能为空")
	}
	if p.Technology == "" {
		return "", fmt.Errorf("technology 不能为空")
	}

	date := time.Now().Format("2006年01月")
	techLabel := describeTech(p.Technology)

	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", p.ProjectName)
	fmt.Fprintf(&b, "## 土壤修复实施方案\n\n")
	fmt.Fprintf(&b, "**编制单位**：%s\n", orDefault(p.UnitName, "(施工单位名称)"))
	fmt.Fprintf(&b, "**编制日期**：%s\n\n", date)
	fmt.Fprintf(&b, "---\n\n")

	fmt.Fprintf(&b, "## 目录\n\n")
	fmt.Fprintf(&b, "1 编制依据\n")
	fmt.Fprintf(&b, "2 项目概况\n")
	fmt.Fprintf(&b, "3 修复目标与范围\n")
	fmt.Fprintf(&b, "4 修复技术方案\n")
	fmt.Fprintf(&b, "  4.1 技术原理\n")
	fmt.Fprintf(&b, "  4.2 工艺流程\n")
	fmt.Fprintf(&b, "  4.3 工艺参数设计\n")
	fmt.Fprintf(&b, "  4.4 药剂/材料方案\n")
	fmt.Fprintf(&b, "  4.5 设备选型\n")
	fmt.Fprintf(&b, "5 施工组织\n")
	fmt.Fprintf(&b, "6 质量保证\n")
	fmt.Fprintf(&b, "7 二次污染防控\n")
	fmt.Fprintf(&b, "8 监测计划\n")
	fmt.Fprintf(&b, "9 进度计划\n")
	fmt.Fprintf(&b, "10 人员配置\n")
	fmt.Fprintf(&b, "11 安全文明施工\n")
	fmt.Fprintf(&b, "附件\n\n---\n\n")

	fmt.Fprintf(&b, "## 1 编制依据\n\n")
	fmt.Fprintf(&b, "- HJ 25.1-2019《建设用地土壤污染状况调查技术导则》\n")
	fmt.Fprintf(&b, "- HJ 25.2-2019《建设用地土壤污染风险管控与修复技术导则》\n")
	fmt.Fprintf(&b, "- HJ 25.4-2019《建设用地土壤修复方案编制技术导则》\n")
	fmt.Fprintf(&b, "- GB 36600-2018《土壤环境质量标准 建设用地土壤污染风险管控标准》\n")
	fmt.Fprintf(&b, "- 项目招标文件及合同\n")
	fmt.Fprintf(&b, "- 场地调查报告及风险评估报告\n\n")

	fmt.Fprintf(&b, "## 2 项目概况\n\n")
	fmt.Fprintf(&b, "| 项目 | 内容 |\n|------|------|\n")
	fmt.Fprintf(&b, "| 项目名称 | %s |\n", p.ProjectName)
	if p.SiteAddress != "" {
		fmt.Fprintf(&b, "| 项目地址 | %s |\n", p.SiteAddress)
	}
	if p.RepairArea > 0 {
		fmt.Fprintf(&b, "| 修复面积 | %.2f m2 |\n", p.RepairArea)
	}
	if p.SoilVolume > 0 {
		fmt.Fprintf(&b, "| 修复方量 | %.2f m3 |\n", p.SoilVolume)
	}
	if p.DepthRange != "" {
		fmt.Fprintf(&b, "| 修复深度 | %s |\n", p.DepthRange)
	}
	if len(p.Contaminants) > 0 {
		fmt.Fprintf(&b, "| 主要污染物 | %s |\n", strings.Join(p.Contaminants, "、"))
	}
	fmt.Fprintf(&b, "| 修复技术 | %s |\n", p.Technology)
	fmt.Fprintf(&b, "| 涉及地下水 | %v |\n", p.Groundwater)
	fmt.Fprintf(&b, "\n")

	fmt.Fprintf(&b, "## 3 修复目标与范围\n\n")
	if p.TargetValue != "" {
		fmt.Fprintf(&b, "修复目标值：%s\n\n", p.TargetValue)
	} else {
		fmt.Fprintf(&b, "修复目标值应依据HJ 25.3-2019风险评估结果确定。\n\n")
	}
	fmt.Fprintf(&b, "修复范围：污染区域边界外扩不少于1m的安全边界。\n")
	if p.SoilVolume > 0 {
		fmt.Fprintf(&b, "修复总方量：%.2f m3。\n", p.SoilVolume)
	}
	if p.DepthRange != "" {
		fmt.Fprintf(&b, "修复深度范围：%s。\n", p.DepthRange)
	}
	fmt.Fprintf(&b, "\n")

	fmt.Fprintf(&b, "## 4 修复技术方案\n\n")
	fmt.Fprintf(&b, "### 4.1 技术原理\n\n")
	fmt.Fprintf(&b, "%s：%s\n\n", p.Technology, techLabel)

	fmt.Fprintf(&b, "### 4.2 工艺流程\n\n")
	b.WriteString(genProcessFlow(p.Technology))

	fmt.Fprintf(&b, "### 4.3 工艺参数设计\n\n")
	b.WriteString(genTechParams(p.Technology, p.Contaminants))
	fmt.Fprintf(&b, "\n")

	fmt.Fprintf(&b, "### 4.4 药剂/材料方案\n\n")
	b.WriteString(genMaterialPlan(p.Technology, p.Contaminants))
	fmt.Fprintf(&b, "\n")

	fmt.Fprintf(&b, "### 4.5 设备选型\n\n")
	b.WriteString(genEquipmentList(p.Technology, p.SoilVolume))
	fmt.Fprintf(&b, "\n")

	fmt.Fprintf(&b, "## 5 施工组织\n\n")
	fmt.Fprintf(&b, "施工分区：根据污染分布将修复区域划分为若干施工单元，分批实施。\n")
	fmt.Fprintf(&b, "施工流程：场地平整→设备安装调试→修复施工→过程检测→效果验收→场地恢复。\n\n")

	fmt.Fprintf(&b, "## 6 质量保证\n\n")
	fmt.Fprintf(&b, "- 原材料检验（药剂、土壤检测）\n")
	fmt.Fprintf(&b, "- 施工过程参数监控\n")
	b.WriteString("- 修复效果自检（抽检比例不低于10%）\n")
	fmt.Fprintf(&b, "- 第三方检测验证\n\n")

	fmt.Fprintf(&b, "## 7 二次污染防控\n\n")
	fmt.Fprintf(&b, "- 扬尘：洒水降尘、土方覆盖、设置围挡\n")
	fmt.Fprintf(&b, "- 废水：施工废水收集沉淀后回用或达标排放\n")
	fmt.Fprintf(&b, "- 噪声：选用低噪设备，夜间限时施工\n")
	fmt.Fprintf(&b, "- 固废：废药剂包装、废弃防护用品分类收集处置\n")
	fmt.Fprintf(&b, "- 异味：VOCs废气收集处理（如活性炭吸附）\n\n")

	fmt.Fprintf(&b, "## 8 监测计划\n\n")
	fmt.Fprintf(&b, "### 8.1 施工期环境监测\n\n")
	fmt.Fprintf(&b, "| 监测要素 | 监测项目 | 监测频次 |\n|----------|----------|----------|\n")
	fmt.Fprintf(&b, "| 环境空气 | TSP、VOCs | 1次/天 |\n")
	fmt.Fprintf(&b, "| 废水 | pH、SS、COD | 1次/周 |\n")
	fmt.Fprintf(&b, "| 噪声 | 等效声级 | 1次/天 |\n\n")

	if p.Groundwater {
		fmt.Fprintf(&b, "### 8.2 地下水监测\n\n")
		fmt.Fprintf(&b, "| 监测井编号 | 监测项目 | 频次 |\n|------------|----------|------|\n")
		fmt.Fprintf(&b, "| (上、下游各设监测井) | 特征污染物 | 1次/周 |\n\n")
	}

	period := p.ConstructionPeriod
	if period == "" {
		period = "(按合同要求)"
	}
	fmt.Fprintf(&b, "## 9 进度计划\n\n")
	fmt.Fprintf(&b, "施工工期：%s\n\n", period)
	fmt.Fprintf(&b, "| 阶段 | 工期 | 工作内容 |\n|------|------|----------|\n")
	fmt.Fprintf(&b, "| 施工准备 | (天) | 临建、设备进场 |\n")
	fmt.Fprintf(&b, "| 修复施工 | (天) | 污染土壤处理 |\n")
	fmt.Fprintf(&b, "| 效果监测 | (天) | 自检+第三方检测 |\n")
	fmt.Fprintf(&b, "| 竣工验收 | (天) | 资料归档、退场 |\n\n")

	fmt.Fprintf(&b, "## 10 人员配置\n\n")
	fmt.Fprintf(&b, "| 岗位 | 人数 |\n|------|------|\n")
	fmt.Fprintf(&b, "| 项目经理 | 1 |\n")
	fmt.Fprintf(&b, "| 技术负责人 | 1 |\n")
	fmt.Fprintf(&b, "| 施工员 | (人数) |\n")
	fmt.Fprintf(&b, "| 安全员 | 1 |\n")
	fmt.Fprintf(&b, "| 质量员 | 1 |\n")
	fmt.Fprintf(&b, "| 资料员 | 1 |\n\n")

	fmt.Fprintf(&b, "## 11 安全文明施工\n\n")
	fmt.Fprintf(&b, "- 建立安全生产责任制度\n")
	fmt.Fprintf(&b, "- 编制安全专项方案\n")
	fmt.Fprintf(&b, "- 开展安全技术交底\n")
	fmt.Fprintf(&b, "- 配备 PPE 劳动防护用品\n")
	fmt.Fprintf(&b, "- 制定事故应急预案\n\n")

	fmt.Fprintf(&b, "---\n*本方案由 gaea imple_plan 生成，需结合实际调整。*\n")
	return tool.WrapText(b.String()), nil
}

func describeTech(tech string) string {
	t := strings.ToLower(tech)
	switch {
	case strings.Contains(t, "化学氧化") || strings.Contains(t, "化学氧化"):
		return "通过向污染土壤注入氧化剂（如过硫酸钠、高锰酸钾、Fenton试剂），将有机污染物氧化分解为CO2和H2O。适用于石油烃、VOCs、SVOCs等有机物污染。"
	case strings.Contains(t, "固化") || strings.Contains(t, "稳定化"):
		return "通过添加固化剂/稳定剂（如水泥、石灰、飞灰），将重金属污染物固定在土壤基质中，降低其迁移性和生物可利用性。适用于重金属污染土壤。"
	case strings.Contains(t, "sve") || strings.Contains(t, "气相抽提"):
		return "通过真空抽提系统在非饱和带形成负压，促使土壤中的VOCs随气流进入抽提井并在地面处理。适用于VOCs、汽油组分等易挥发污染物。"
	case strings.Contains(t, "热脱附"):
		return "通过加热将有机污染物从土壤中解吸出来，尾气经处理后排放。适用于高浓度有机物、VOCs、SVOCs污染土壤。"
	case strings.Contains(t, "土壤淋洗"):
		return "利用水或淋洗液（酸/碱/表面活性剂）清洗污染土壤，将污染物从土壤颗粒转移到液相中分离处理。适用于重金属和高浓度有机物。"
	case strings.Contains(t, "生物"):
		return "利用微生物的代谢作用将有机污染物降解为无害物质。适用于石油烃、低浓度有机物，处理周期较长。"
	case strings.Contains(t, "水泥窑"):
		return "将污染土壤作为替代原料送入水泥回转窑，在高温下有机物彻底分解。适用于有机污染土壤的资源化利用。"
	case strings.Contains(t, "阻隔"):
		return "通过设置垂直帷幕、覆盖清洁土等方式阻断污染物迁移途径。适用于低风险或暂不开发地块。"
	default:
		return "根据场地条件和污染物特性确定具体工艺原理。"
	}
}

func genProcessFlow(tech string) string {
	t := strings.ToLower(tech)
	switch {
	case strings.Contains(t, "化学氧化") || strings.Contains(t, "化学氧化"):
		return "污染土壤开挖 → 破碎筛分（去除大块杂物） → 与氧化剂/活化剂混合 → 养护反应（3~7天） → 效果检测 → 合格回填/外运\n\n"
	case strings.Contains(t, "固化") || strings.Contains(t, "稳定化"):
		return "污染土壤开挖 → 破碎筛分 → 与固化剂/稳定剂混合搅拌 → 养护（7~14天） → 浸出检测 → 合格回填/资源化利用\n\n"
	case strings.Contains(t, "sve") || strings.Contains(t, "气相抽提"):
		return "布设抽提井 → 安装真空泵 → 运行抽提 → 尾气处理（活性炭/催化氧化） → 排气达标排放\n\n"
	default:
		return "（根据具体修复技术确定工艺流程）\n\n"
	}
}

func genTechParams(tech string, contaminants []string) string {
	t := strings.ToLower(tech)
	switch {
	case strings.Contains(t, "化学氧化") || strings.Contains(t, "化学氧化"):
		return fmt.Sprintf(`| 参数 | 设计值 |
|------|--------|
| 氧化剂 | 过硫酸钠（Na2S2O8） |
| 活化方式 | 碱活化/Fe2+活化 |
| 药剂投加比 | 氧化剂:污染物 = 10:1~20:1（质量比） |
| 含水率调整 | 20%%~30%% |
| 养护时间 | 7~14天 |
| 处理深度 | %s |`, orDefault("(按设计要求)", "0.5~5.0m"))
	case strings.Contains(t, "固化") || strings.Contains(t, "稳定化"):
		return fmt.Sprintf(`| 参数 | 设计值 |
|------|--------|
| 固化剂 | P.O42.5水泥 |
| 稳定剂 | 石灰/飞灰（调节pH） |
| 水泥掺量 | 10%%~20%%（质量比） |
| 养护时间 | 7~14天 |
| 处理后pH | 控制在7~9 |
| 处理后浸出浓度 | 满足GB 8978限值 |
| 处理深度 | %s |`, orDefault("(按设计要求)", "0.5~5.0m"))
	default:
		return "(根据修复技术填写工艺参数)\n"
	}
}

func genMaterialPlan(tech string, contaminants []string) string {
	t := strings.ToLower(tech)
	switch {
	case strings.Contains(t, "化学氧化") || strings.Contains(t, "化学氧化"):
		return `主要药剂：过硫酸钠（工业级，含量≥99%）、片碱（NaOH，调节pH至碱性活化）、FeSO4（活化剂）。
药剂用量：根据小试确定最佳投加比，一般氧化剂与污染物摩尔比10:1~20:1。
药剂存储：干粉密封存放于干燥仓库，避免受潮结块。`
	case strings.Contains(t, "固化") || strings.Contains(t, "稳定化"):
		return `主要材料：P.O42.5水泥、生石灰（CaO含量≥80%）。
材料用量：根据小试确定最佳掺量，水泥掺量10%~20%，石灰掺量2%~5%。
材料存储：水泥筒仓存放，石灰袋装密封防潮。`
	default:
		return "（根据修复技术填写药剂/材料方案）"
	}
}

func genEquipmentList(tech string, volume float64) string {
	t := strings.ToLower(tech)
	switch {
	case strings.Contains(t, "化学氧化") || strings.Contains(t, "化学氧化"):
		return `| 设备名称 | 规格型号 | 数量 | 用途 |
|----------|----------|------|------|
| 挖掘机 | 1.0~1.5m3 | 2台 | 土方开挖与转运 |
| 破碎筛分机 | 50~100t/h | 1台 | 土壤破碎筛分 |
| 混合搅拌机 | 连续式 | 1台 | 土壤与药剂混合 |
| 药剂配制系统 | 成套 | 1套 | 药剂溶解投加 |
| 装载机 | ZL50 | 2台 | 物料转运 |
| 自卸车 | 20t | 4台 | 土方运输 |`
	case strings.Contains(t, "固化") || strings.Contains(t, "稳定化"):
		return `| 设备名称 | 规格型号 | 数量 | 用途 |
|----------|----------|------|------|
| 挖掘机 | 1.0~1.5m3 | 2台 | 土方开挖 |
| 破碎筛分机 | 50~100t/h | 1台 | 土壤破碎 |
| 稳定化搅拌设备 | 连续式 | 1套 | 土壤与固化剂混合 |
| 水泥筒仓 | 100t | 1座 | 水泥存储 |
| 装载机 | ZL50 | 2台 | 物料转运 |`
	default:
		return `| 设备名称 | 规格 | 数量 | 用途 |
|----------|------|------|------|
| （主要设备） | （规格） | （数量） | （用途） |`
	}
}
