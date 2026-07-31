// Package proposal — 方案模板定义
package proposal

// DefaultTemplates 预设模板列表
var DefaultTemplates = []Template{
	{
		ID:          "business-plan",
		Name:        "商业计划书",
		Description: "面向投资人或合作伙伴的商业方案",
		Sections:    []string{"项目概述", "市场分析", "产品与服务", "商业模式", "营销策略", "团队介绍", "财务规划", "风险与对策"},
	},
	{
		ID:          "tech-proposal",
		Name:        "技术方案",
		Description: "软件/系统技术方案文档",
		Sections:    []string{"项目背景", "需求分析", "技术架构", "详细设计", "实施计划", "测试方案", "运维方案", "风险评估"},
	},
	{
		ID:          "project-plan",
		Name:        "项目计划书",
		Description: "项目管理与执行计划",
		Sections:    []string{"项目背景与目标", "项目范围", "工作分解(WBS)", "进度计划", "资源分配", "质量管理", "沟通计划", "风险管理"},
	},
	{
		ID:          "weekly-report",
		Name:        "周报",
		Description: "工作周报模板",
		Sections:    []string{"本周工作概述", "重点工作进展", "遇到的问题", "下周工作计划", "需要协调的事项"},
	},
	{
		ID:          "meeting-minutes",
		Name:        "会议纪要",
		Description: "会议记录与决议跟踪",
		Sections:    []string{"会议基本信息", "参会人员", "会议议题", "讨论内容", "决议事项", "待办任务", "下次会议安排"},
	},
	{
		ID:          "research-report",
		Name:        "调研报告",
		Description: "市场/技术/竞品调研报告",
		Sections:    []string{"调研背景与目的", "调研方法", "调研结果", "数据分析", "结论与建议", "附录"},
	},
	{
		ID:          "blank",
		Name:        "空白方案",
		Description: "从零开始自由编写",
		Sections:    []string{},
	},
	{
		ID:          "soil-remediation-bid",
		Name:        "土壤修复投标技术方案",
		Description: "环保工程土壤修复方向投标技术方案，含15章标准结构，覆盖HJ 25.4规范要求",
		Sections: []string{
			"项目概述与背景",
			"编制依据",
			"场地污染现状分析",
			"污染风险评估",
			"修复目标与范围",
			"修复技术方案比选",
			"推荐修复方案详细设计",
			"施工组织设计",
			"环境保护与安全措施",
			"质量控制方案",
			"施工进度计划",
			"人员组织与设备配置",
			"投资估算与工程量清单",
			"风险分析与应急预案",
			"售后服务与质量保障",
		},
	},
}

// GetTemplate 获取模板
func GetTemplate(id string) *Template {
	for _, t := range DefaultTemplates {
		if t.ID == id {
			return &t
		}
	}
	return nil
}

// SectionsFromTemplate 从模板生成章节列表
func SectionsFromTemplate(t *Template) []ProposalSection {
	var sections []ProposalSection
	for i, title := range t.Sections {
		sections = append(sections, ProposalSection{
			Index:  i,
			Title:  title,
			Status: "pending",
		})
	}
	return sections
}
