package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(bidProposal{}) }

type bidProposal struct{}

func (bidProposal) Name() string { return "bid_proposal" }

func (bidProposal) Description() string {
	return "生成土壤修复项目投标方案（技术标）框架：输入招标信息、地块概况，输出技术路线、施工组织、人员配置、进度计划等章节。"
}

func (bidProposal) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "project_name":{"type":"string","description":"项目名称"},
  "bidder":{"type":"string","description":"投标单位名称"},
  "site_area":{"type":"number","description":"修复面积(m2)"},
  "soil_volume":{"type":"number","description":"修复土方量(m3)"},
  "contaminants":{"type":"array","items":{"type":"string"},"description":"主要污染物列表"},
  "technology":{"type":"string","description":"拟采用修复技术路线"},
  "target_value":{"type":"string","description":"修复目标值简述"},
  "construction_period":{"type":"string","description":"施工工期要求"},
  "team_size":{"type":"integer","description":"拟投入人员数量"},
  "key_equipment":{"type":"string","description":"关键设备配置"}
},
"required":["project_name","bidder"]
}`)
}

func (bidProposal) ReadOnly() bool { return true }

func (bidProposal) CompactDescription() string     { return compactDesc["bid_proposal"] }
func (bidProposal) CompactSchema() json.RawMessage { return compactSchema["bid_proposal"] }

type bidInput struct {
	ProjectName        string   `json:"project_name"`
	Bidder             string   `json:"bidder"`
	SiteArea           float64  `json:"site_area,omitempty"`
	SoilVolume         float64  `json:"soil_volume,omitempty"`
	Contaminants       []string `json:"contaminants,omitempty"`
	Technology         string   `json:"technology,omitempty"`
	TargetValue        string   `json:"target_value,omitempty"`
	ConstructionPeriod string   `json:"construction_period,omitempty"`
	TeamSize           int      `json:"team_size,omitempty"`
	KeyEquipment       string   `json:"key_equipment,omitempty"`
}

func (bidProposal) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p bidInput
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	p.ProjectName = strings.TrimSpace(p.ProjectName)
	p.Bidder = strings.TrimSpace(p.Bidder)
	if p.ProjectName == "" || p.Bidder == "" {
		return "", fmt.Errorf("project_name 和 bidder 不能为空")
	}

	date := time.Now().Format("2006年01月")

	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", p.ProjectName)
	fmt.Fprintf(&b, "## 技术投标文件\n\n")
	fmt.Fprintf(&b, "**投标单位**：%s\n", p.Bidder)
	fmt.Fprintf(&b, "**编制日期**：%s\n\n", date)
	fmt.Fprintf(&b, "---\n\n")

	// 目录
	fmt.Fprintf(&b, "## 目录\n\n")
	fmt.Fprintf(&b, "1 项目概述\n")
	fmt.Fprintf(&b, "2 投标单位资质与业绩\n")
	fmt.Fprintf(&b, "3 技术方案\n")
	fmt.Fprintf(&b, "  3.1 修复目标\n")
	fmt.Fprintf(&b, "  3.2 技术路线比选\n")
	fmt.Fprintf(&b, "  3.3 推荐工艺方案\n")
	fmt.Fprintf(&b, "  3.4 工艺参数设计\n")
	fmt.Fprintf(&b, "4 施工组织设计\n")
	fmt.Fprintf(&b, "  4.1 施工总体部署\n")
	fmt.Fprintf(&b, "  4.2 施工流程\n")
	fmt.Fprintf(&b, "  4.3 设备人员配置\n")
	fmt.Fprintf(&b, "  4.4 施工进度计划\n")
	fmt.Fprintf(&b, "5 质量控制方案\n")
	fmt.Fprintf(&b, "6 安全文明施工\n")
	fmt.Fprintf(&b, "7 二次污染防控\n")
	fmt.Fprintf(&b, "8 监测方案\n")
	fmt.Fprintf(&b, "9 项目管理机构\n")
	fmt.Fprintf(&b, "10 类似业绩\n")
	fmt.Fprintf(&b, "附件\n\n---\n\n")

	// 章节内容
	fmt.Fprintf(&b, "## 1 项目概述\n\n")
	fmt.Fprintf(&b, "本项目为%s。\n\n", p.ProjectName)
	if p.SiteArea > 0 {
		fmt.Fprintf(&b, "修复面积约 %.2f m2，", p.SiteArea)
	}
	if p.SoilVolume > 0 {
		fmt.Fprintf(&b, "修复土方量约 %.2f m3。\n\n", p.SoilVolume)
	}
	if len(p.Contaminants) > 0 {
		fmt.Fprintf(&b, "主要污染物为：%s。\n\n", strings.Join(p.Contaminants, "、"))
	}

	fmt.Fprintf(&b, "## 2 投标单位资质与业绩\n\n")
	fmt.Fprintf(&b, "%s具备以下资质：\n\n", p.Bidder)
	fmt.Fprintf(&b, "- 环保工程专业承包资质\n")
	fmt.Fprintf(&b, "- 安全生产许可证\n")
	fmt.Fprintf(&b, "- ISO9001质量管理体系认证\n")
	fmt.Fprintf(&b, "- ISO14001环境管理体系认证\n\n")
	fmt.Fprintf(&b, "同类项目业绩：\n\n")
	fmt.Fprintf(&b, "| 序号 | 项目名称 | 污染物 | 修复方量 | 修复技术 |\n")
	fmt.Fprintf(&b, "|------|----------|--------|----------|----------|\n")
	fmt.Fprintf(&b, "| 1 | (类似项目1) | (污染物) | (方量) | (技术) |\n")
	fmt.Fprintf(&b, "| 2 | (类似项目2) | (污染物) | (方量) | (技术) |\n\n")

	fmt.Fprintf(&b, "## 3 技术方案\n\n")
	fmt.Fprintf(&b, "### 3.1 修复目标\n\n")
	if p.TargetValue != "" {
		fmt.Fprintf(&b, "修复目标值：%s\n\n", p.TargetValue)
	} else {
		fmt.Fprintf(&b, "修复目标值依据HJ 25.3-2019风险评估结果确定，或按照招标文件要求执行。\n\n")
	}

	fmt.Fprintf(&b, "### 3.2 技术路线比选\n\n")
	if p.Technology != "" {
		fmt.Fprintf(&b, "拟采用技术路线：**%s**\n\n", p.Technology)
	} else if len(p.Contaminants) > 0 {
		fmt.Fprintf(&b, "根据污染物类型推荐以下技术路线：\n\n")
		for _, c := range p.Contaminants {
			tech := recommendTech(c)
			fmt.Fprintf(&b, "- **%s**：%s\n", c, tech)
		}
		fmt.Fprintf(&b, "\n")
	} else {
		fmt.Fprintf(&b, "（根据污染物类型和场地条件选择合适修复技术）\n\n")
	}

	fmt.Fprintf(&b, "### 3.3 推荐工艺方案\n\n")
	fmt.Fprintf(&b, "结合场地条件和修复目标，推荐采用以下工艺方案：\n\n")
	fmt.Fprintf(&b, "- 工艺流程：\n")
	fmt.Fprintf(&b, "- 关键工艺参数：\n")
	fmt.Fprintf(&b, "- 药剂/材料方案：\n\n")

	fmt.Fprintf(&b, "## 4 施工组织设计\n\n")
	fmt.Fprintf(&b, "### 4.1 施工总体部署\n\n")
	fmt.Fprintf(&b, "施工区划分为：\n")
	fmt.Fprintf(&b, "- 修复作业区\n")
	fmt.Fprintf(&b, "- 药剂配制区\n")
	fmt.Fprintf(&b, "- 临时办公生活区\n")
	fmt.Fprintf(&b, "- 车辆设备停放区\n\n")

	fmt.Fprintf(&b, "### 4.2 施工流程\n\n")
	fmt.Fprintf(&b, "1. 施工准备（场地平整、临建搭设、设备进场）\n")
	fmt.Fprintf(&b, "2. 修复施工（按工艺方案分区分批实施）\n")
	fmt.Fprintf(&b, "3. 过程监测与调整\n")
	fmt.Fprintf(&b, "4. 自检与效果评估\n")
	fmt.Fprintf(&b, "5. 场地恢复与退场\n\n")

	fmt.Fprintf(&b, "### 4.3 设备人员配置\n\n")
	if p.TeamSize > 0 {
		fmt.Fprintf(&b, "拟投入项目人员 **%d** 人，其中：\n\n", p.TeamSize)
	} else {
		fmt.Fprintf(&b, "拟投入项目人员配置如下：\n\n")
	}
	fmt.Fprintf(&b, "| 岗位 | 人数 | 职责 |\n|------|------|------|\n")
	fmt.Fprintf(&b, "| 项目经理 | 1 | 全面负责 |\n")
	fmt.Fprintf(&b, "| 技术负责人 | 1 | 技术方案编制与指导 |\n")
	fmt.Fprintf(&b, "| 施工员 | (人数) | 现场施工管理 |\n")
	fmt.Fprintf(&b, "| 安全员 | 1 | 安全管理 |\n")
	fmt.Fprintf(&b, "| 质量员 | 1 | 质量控制 |\n")
	fmt.Fprintf(&b, "| 资料员 | 1 | 资料编制归档 |\n\n")

	fmt.Fprintf(&b, "主要施工设备：\n\n")
	if p.KeyEquipment != "" {
		fmt.Fprintf(&b, "%s\n\n", p.KeyEquipment)
	} else {
		fmt.Fprintf(&b, "| 设备名称 | 型号 | 数量 | 用途 |\n|----------|------|------|------|\n")
		fmt.Fprintf(&b, "| （挖掘机） |（型号）|（数量）| 土方开挖 |\n")
		fmt.Fprintf(&b, "| （修复设备） |（型号）|（数量）| 污染土壤处理 |\n")
		fmt.Fprintf(&b, "| （检测仪器） |（型号）|（数量）| 过程监测 |\n\n")
	}

	fmt.Fprintf(&b, "### 4.4 施工进度计划\n\n")
	period := p.ConstructionPeriod
	if period == "" {
		period = "(按招标文件要求)"
	}
	fmt.Fprintf(&b, "施工工期：%s\n\n", period)
	fmt.Fprintf(&b, "| 阶段 | 工期 | 主要工作内容 |\n|------|------|--------------|\n")
	fmt.Fprintf(&b, "| 施工准备 | (天) | 临建、设备进场、场地平整 |\n")
	fmt.Fprintf(&b, "| 修复施工 | (天) | 污染土壤处理 |\n")
	fmt.Fprintf(&b, "| 效果监测 | (天) | 自检与第三方检测 |\n")
	fmt.Fprintf(&b, "| 竣工验收 | (天) | 验收、资料归档、退场 |\n\n")

	fmt.Fprintf(&b, "## 5 质量控制方案\n\n")
	fmt.Fprintf(&b, "建立三级质量保证体系：\n")
	fmt.Fprintf(&b, "1. 施工班组自检\n")
	fmt.Fprintf(&b, "2. 项目部专检\n")
	fmt.Fprintf(&b, "3. 第三方检测验证\n\n")
	fmt.Fprintf(&b, "关键质量控制点：\n")
	fmt.Fprintf(&b, "- 药剂质量检验\n")
	fmt.Fprintf(&b, "- 施工工艺参数控制\n")
	fmt.Fprintf(&b, "- 修复效果自检\n\n")

	fmt.Fprintf(&b, "## 6 安全文明施工\n\n")
	fmt.Fprintf(&b, "- 建立健全安全生产责任制\n")
	fmt.Fprintf(&b, "- 编制安全专项施工方案\n")
	fmt.Fprintf(&b, "- 设置安全警示标识\n")
	fmt.Fprintf(&b, "- 配备劳动防护用品\n")
	fmt.Fprintf(&b, "- 制定应急预案\n\n")

	fmt.Fprintf(&b, "## 7 二次污染防控\n\n")
	fmt.Fprintf(&b, "- 扬尘控制：洒水降尘、覆盖防尘网\n")
	fmt.Fprintf(&b, "- 废水控制：收集处理达标后排入市政管网\n")
	fmt.Fprintf(&b, "- 噪声控制：选用低噪声设备，合理安排施工时间\n")
	fmt.Fprintf(&b, "- 固废控制：分类收集、合规处置\n\n")

	fmt.Fprintf(&b, "## 8 监测方案\n\n")
	fmt.Fprintf(&b, "- 施工期环境监测：废气、废水、噪声、土壤\n")
	fmt.Fprintf(&b, "- 修复效果监测：修复后土壤检测\n")
	fmt.Fprintf(&b, "- 监测频次与点位按HJ 25.5-2019执行\n\n")

	fmt.Fprintf(&b, "## 9 项目管理机构\n\n")
	fmt.Fprintf(&b, "项目组织机构图：\n\n")
	fmt.Fprintf(&b, "项目经理 → 技术负责人 → 施工组、质量组、安全组、综合组\n\n")

	fmt.Fprintf(&b, "## 10 类似业绩\n\n")
	fmt.Fprintf(&b, "（列表展示投标单位近3-5年的同类土壤修复项目业绩，含合同、验收证明等）\n\n")

	fmt.Fprintf(&b, "---\n*本方案由 gaea bid_proposal 生成，需结合实际项目情况调整。*\n")
	return tool.WrapText(b.String()), nil
}

func recommendTech(contaminant string) string {
	c := strings.ToLower(contaminant)
	if strings.Contains(c, "砷") || strings.Contains(c, "as") || strings.Contains(c, "镉") || strings.Contains(c, "cd") ||
		strings.Contains(c, "铅") || strings.Contains(c, "pb") || strings.Contains(c, "汞") || strings.Contains(c, "hg") ||
		strings.Contains(c, "铬") || strings.Contains(c, "cr") || strings.Contains(c, "镍") || strings.Contains(c, "ni") ||
		strings.Contains(c, "铜") || strings.Contains(c, "cu") || strings.Contains(c, "锌") || strings.Contains(c, "zn") ||
		strings.Contains(c, "重金属") {
		return "推荐采用固化/稳定化技术或土壤淋洗技术。重金属污染土壤还可考虑客土法或植物修复（低浓度）。"
	}
	if strings.Contains(c, "石油") || strings.Contains(c, "tph") || strings.Contains(c, "烃") {
		return "推荐采用异位化学氧化技术或生物堆腐技术。石油烃浓度较高时可采用热脱附。"
	}
	if strings.Contains(c, "voc") || strings.Contains(c, "苯") || strings.Contains(c, "甲苯") || strings.Contains(c, "二甲苯") ||
		strings.Contains(c, "氯") || strings.Contains(c, "四氯化碳") {
		return "推荐采用原位化学氧化或SVE气相抽提技术。VOCs浓度高时可采用热脱附。"
	}
	if strings.Contains(c, "农药") || strings.Contains(c, "有机") || strings.Contains(c, "svoc") || strings.Contains(c, "多环") {
		return "推荐采用化学氧化技术（过硫酸盐活化）或水泥窑协同处置。"
	}
	return "需根据具体污染物性质开展修复技术筛选试验（ treatability study）。"
}
