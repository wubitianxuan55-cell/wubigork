package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/wubigork/wubigork/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(surveyReport{}) }

type surveyReport struct{}

func (surveyReport) Name() string { return "survey_report" }

func (surveyReport) Description() string {
	return "生成土壤污染状况调查报告框架（初调/详调）：输入地块信息、检测数据，输出结构化报告大纲（项目概况、污染识别、布点方案、检测评价、结论建议）。"
}

func (surveyReport) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "report_type":{"type":"string","description":"报告类型：初调（初步调查）、详调（详细调查）","default":"初调"},
  "site_name":{"type":"string","description":"地块名称"},
  "site_address":{"type":"string","description":"地块地址"},
  "site_area":{"type":"number","description":"地块面积（平方米）"},
  "land_use":{"type":"string","description":"规划用地性质，如居住用地、工业用地、商业用地"},
  "past_use":{"type":"string","description":"历史用途，如化工生产、电镀、垃圾填埋等"},
  "pollutants_suspected":{"type":"array","items":{"type":"string"},"description":"疑似污染物列表"},
  "sampling_points":{"type":"integer","description":"采样点数","default":0},
  "client":{"type":"string","description":"委托单位"},
  "survey_company":{"type":"string","description":"调查单位"},
  "include_toc":{"type":"boolean","description":"是否包含目录框架","default":true}
},
"required":["site_name"]
}`)
}

func (surveyReport) ReadOnly() bool { return true }

func (surveyReport) CompactDescription() string     { return compactDesc["survey_report"] }
func (surveyReport) CompactSchema() json.RawMessage { return compactSchema["survey_report"] }

type surveyInput struct {
	ReportType          string   `json:"report_type"`
	SiteName            string   `json:"site_name"`
	SiteAddress         string   `json:"site_address,omitempty"`
	SiteArea            float64  `json:"site_area,omitempty"`
	LandUse             string   `json:"land_use,omitempty"`
	PastUse             string   `json:"past_use,omitempty"`
	PollutantsSuspected []string `json:"pollutants_suspected,omitempty"`
	SamplingPoints      int      `json:"sampling_points,omitempty"`
	Client              string   `json:"client,omitempty"`
	SurveyCompany       string   `json:"survey_company,omitempty"`
	IncludeTOC          bool     `json:"include_toc,omitempty"`
}

func (surveyReport) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p surveyInput
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	p.SiteName = strings.TrimSpace(p.SiteName)
	if p.SiteName == "" {
		return "", fmt.Errorf("site_name 不能为空")
	}

	isDetail := strings.Contains(p.ReportType, "详调")
	isPrelim := !isDetail
	t := "初步调查"
	if isDetail {
		t = "详细调查"
	}

	date := time.Now().Format("2006年01月")

	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", p.SiteName)
	fmt.Fprintf(&b, "## 土壤污染状况%s报告\n\n", t)
	fmt.Fprintf(&b, "**编制单位**：%s\n", orDefault(p.SurveyCompany, "(调查单位名称)"))
	fmt.Fprintf(&b, "**委托单位**：%s\n", orDefault(p.Client, "(委托单位名称)"))
	fmt.Fprintf(&b, "**编制日期**：%s\n\n", date)
	fmt.Fprintf(&b, "---\n\n")

	if p.IncludeTOC {
		fmt.Fprintf(&b, "## 目录\n\n")
		fmt.Fprintf(&b, "1 前言\n")
		fmt.Fprintf(&b, "2 概述\n")
		fmt.Fprintf(&b, "  2.1 地块基本情况\n")
		fmt.Fprintf(&b, "  2.2 地块地理位置\n")
		fmt.Fprintf(&b, "  2.3 地块周边环境\n")
		fmt.Fprintf(&b, "  2.4 地块规划用途\n")
		if isPrelim {
			fmt.Fprintf(&b, "3 第一阶段调查 污染识别\n")
			fmt.Fprintf(&b, "  3.1 资料收集与分析\n")
			fmt.Fprintf(&b, "  3.2 现场踏勘\n")
			fmt.Fprintf(&b, "  3.3 人员访谈\n")
			fmt.Fprintf(&b, "  3.4 污染识别结论\n")
			fmt.Fprintf(&b, "4 第二阶段调查 初步采样\n")
		} else {
			fmt.Fprintf(&b, "3 第一阶段调查 污染识别结论（引用初调报告）\n")
			fmt.Fprintf(&b, "4 第二阶段调查 详细采样\n")
		}
		fmt.Fprintf(&b, "  4.1 采样方案\n")
		fmt.Fprintf(&b, "  4.2 现场采样与实验室检测\n")
		fmt.Fprintf(&b, "  4.3 质量控制\n")
		fmt.Fprintf(&b, "5 检测结果与分析\n")
		fmt.Fprintf(&b, "  5.1 检测结果统计\n")
		fmt.Fprintf(&b, "  5.2 评价标准\n")
		fmt.Fprintf(&b, "  5.3 结果评价\n")
		fmt.Fprintf(&b, "  5.4 污染物分布特征\n")
		fmt.Fprintf(&b, "6 结论与建议\n")
		fmt.Fprintf(&b, "  6.1 结论\n")
		fmt.Fprintf(&b, "  6.2 建议\n")
		fmt.Fprintf(&b, "附件\n\n---\n\n")
	}

	fmt.Fprintf(&b, "## 1 前言\n\n")
	fmt.Fprintf(&b, "受%s委托，%s对%s开展土壤污染状况%s工作。\n\n",
		orDefault(p.Client, "(委托单位)"),
		orDefault(p.SurveyCompany, "(调查单位)"),
		p.SiteName, t)
	if isPrelim {
		fmt.Fprintf(&b, "本次调查工作依据HJ 25.1-2019、GB 36600-2018等规范开展。\n\n")
	} else {
		fmt.Fprintf(&b, "本次详细调查在初步调查基础上加密布点采样，进一步明确污染范围和程度。\n\n")
	}

	fmt.Fprintf(&b, "## 2 概述\n\n")
	fmt.Fprintf(&b, "### 2.1 地块基本情况\n\n")
	fmt.Fprintf(&b, "| 项目 | 内容 |\n|------|------|\n")
	fmt.Fprintf(&b, "| 地块名称 | %s |\n", p.SiteName)
	if p.SiteAddress != "" {
		fmt.Fprintf(&b, "| 地块地址 | %s |\n", p.SiteAddress)
	}
	if p.SiteArea > 0 {
		fmt.Fprintf(&b, "| 地块面积 | %.2f m2 |\n", p.SiteArea)
	}
	if p.LandUse != "" {
		fmt.Fprintf(&b, "| 规划用途 | %s |\n", p.LandUse)
	}
	fmt.Fprintf(&b, "| 调查阶段 | %s |\n", t)
	if p.LandUse != "" {
		landUseDesc := describeLandUse(p.LandUse)
		fmt.Fprintf(&b, "\n规划用地性质为%s，参照GB 36600-2018中「%s」筛选值评价。\n\n", p.LandUse, landUseDesc)
	}
	if p.PastUse != "" {
		fmt.Fprintf(&b, "### 2.2 历史用途\n\n")
		fmt.Fprintf(&b, "地块历史用途为：%s。\n", p.PastUse)
		fmt.Fprintf(&b, "重点关注以下区域：\n\n")
		for _, area := range inferKeyAreas(p.PastUse) {
			fmt.Fprintf(&b, "- %s\n", area)
		}
		fmt.Fprintf(&b, "\n")
	}
	if len(p.PollutantsSuspected) > 0 {
		fmt.Fprintf(&b, "### 2.3 疑似污染物\n\n")
		fmt.Fprintf(&b, "疑似污染物包括：\n\n")
		for _, pol := range p.PollutantsSuspected {
			fmt.Fprintf(&b, "- %s\n", pol)
		}
		fmt.Fprintf(&b, "\n")
	}

	fmt.Fprintf(&b, "## 3 污染识别\n\n")
	if isPrelim {
		fmt.Fprintf(&b, "### 3.1 资料收集与分析\n\n")
		fmt.Fprintf(&b, "收集了土地使用权属文件、历史卫星影像、生产工艺流程、环评资料等。\n\n")
		fmt.Fprintf(&b, "### 3.2 现场踏勘\n\n")
		fmt.Fprintf(&b, "对地块进行了现场踏勘，记录地形地貌、植被覆盖、建筑物分布、异常气味或颜色等。\n\n")
		fmt.Fprintf(&b, "### 3.3 人员访谈\n\n")
		fmt.Fprintf(&b, "对熟悉地块历史的人员进行了访谈，了解生产经营活动和潜在污染源。\n\n")
		fmt.Fprintf(&b, "### 3.4 污染识别结论\n\n")
		fmt.Fprintf(&b, "经第一阶段调查，")
		if len(p.PollutantsSuspected) > 0 {
			fmt.Fprintf(&b, "地块存在潜在污染风险，关注污染物为%s。需开展第二阶段初步采样调查。\n\n", strings.Join(p.PollutantsSuspected, "、"))
		} else {
			fmt.Fprintf(&b, "未发现明显污染痕迹，建议开展初步采样验证。\n\n")
		}
	} else {
		fmt.Fprintf(&b, "引用初步调查报告结论。\n\n")
		fmt.Fprintf(&b, "初步调查结果显示部分指标超过筛选值，需加密布点明确污染范围和深度。\n\n")
	}

	fmt.Fprintf(&b, "## 4 采样方案\n\n")
	samplingDesc := "系统布点法"
	grid := "40m x 40m"
	if isDetail {
		samplingDesc = "加密系统布点法"
		grid = "20m x 20m"
	}
	fmt.Fprintf(&b, "采用%s布设采样点，基本网格密度为%s。\n", samplingDesc, grid)
	fmt.Fprintf(&b, "采样深度根据地质条件和污染分布确定，表层样0~0.5m，深层样穿透污染层。\n\n")
	if p.SamplingPoints > 0 {
		fmt.Fprintf(&b, "共布设 **%d** 个采样点。\n\n", p.SamplingPoints)
	}
	fmt.Fprintf(&b, "检测项目包括GB 36600-2018表1基本项目45项")
	if len(p.PollutantsSuspected) > 0 {
		fmt.Fprintf(&b, "及特征污染物：%s", strings.Join(p.PollutantsSuspected, "、"))
	}
	fmt.Fprintf(&b, "。\n\n")

	fmt.Fprintf(&b, "## 5 检测结果与评价\n\n")
	fmt.Fprintf(&b, "采用GB 36600-2018「%s」筛选值评价。\n\n", describeLandUse(p.LandUse))
	fmt.Fprintf(&b, "| 污染物 | 样品数 | 最小值 | 最大值 | 平均值 | 筛选值 | 超标率 |\n")
	fmt.Fprintf(&b, "|--------|--------|--------|--------|--------|--------|--------|\n")
	fmt.Fprintf(&b, "| (按实际检测数据填写) | | | | | |\n\n")
	fmt.Fprintf(&b, "（描述污染物的平面与垂向分布特征）\n\n")

	fmt.Fprintf(&b, "## 6 结论与建议\n\n")
	if isPrelim {
		fmt.Fprintf(&b, "本次初步调查完成污染识别和初步采样。检测结果表明：\n")
		fmt.Fprintf(&b, "- (填写是否超标、超标污染物、超标范围)\n\n")
		fmt.Fprintf(&b, "建议：若存在超标，开展详细调查；若未超标，可正常开发利用。\n")
	} else {
		fmt.Fprintf(&b, "本次详细调查明确了污染范围和深度。\n")
		fmt.Fprintf(&b, "- 污染面积：约() m2\n")
		fmt.Fprintf(&b, "- 污染方量：约() m3\n\n")
		fmt.Fprintf(&b, "建议：开展风险评估(HJ 25.3-2019)，编制修复方案。\n")
	}

	fmt.Fprintf(&b, "\n---\n*报告由 gaea survey_report 生成，需经专业人员审核。*\n")
	return tool.WrapText(b.String()), nil
}

func orDefault(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

func describeLandUse(landUse string) string {
	l := strings.ToLower(landUse)
	if strings.Contains(l, "居住") || strings.Contains(l, "住宅") || strings.Contains(l, "学校") || strings.Contains(l, "敏感") || strings.Contains(l, "公共") || strings.Contains(l, "公园") {
		return "第一类用地（敏感用地）"
	}
	if strings.Contains(l, "工业") || strings.Contains(l, "商业") || strings.Contains(l, "仓储") || strings.Contains(l, "物流") {
		return "第二类用地（非敏感用地）"
	}
	return "第二类用地（非敏感用地）"
}

func inferKeyAreas(pastUse string) []string {
	u := strings.ToLower(pastUse)
	switch {
	case strings.Contains(u, "化工") || strings.Contains(u, "化学") || strings.Contains(u, "制药"):
		return []string{"生产车间区域", "原料储罐区", "危废暂存间", "污水处理设施区域"}
	case strings.Contains(u, "电镀") || strings.Contains(u, "金属") || strings.Contains(u, "冶炼"):
		return []string{"电镀/生产车间", "酸洗/清洗区域", "废水处理池", "原料堆放区"}
	case strings.Contains(u, "印染") || strings.Contains(u, "纺织"):
		return []string{"印染车间", "染料仓库", "污水处理区域"}
	case strings.Contains(u, "垃圾") || strings.Contains(u, "填埋") || strings.Contains(u, "固废"):
		return []string{"填埋区", "渗滤液收集池", "堆体区域"}
	case strings.Contains(u, "加油站") || strings.Contains(u, "石油") || strings.Contains(u, "油库"):
		return []string{"储油罐区", "加油作业区", "输油管线区域"}
	case strings.Contains(u, "农药") || strings.Contains(u, "化肥"):
		return []string{"生产车间", "原料仓库", "产品仓库", "废水处理区域"}
	default:
		return []string{"主要生产/作业区域", "化学品/物料储存区", "废弃物暂存区", "废水处理区"}
	}
}
