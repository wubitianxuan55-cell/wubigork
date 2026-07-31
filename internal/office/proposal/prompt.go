package proposal

// soilPromptSuffix 土壤修复方案专用的 prompt 后缀
const soilPromptSuffix = "\n【专业要求】\n- 你是环保工程投标方案撰写专家，精通土壤修复领域\n- 撰写时请参考以下修复技术知识库，确保技术描述准确\n- 对于技术比选章节，应从适用性、经济性、工期、二次污染、成熟度五个维度对比\n- 修复目标值应引用 GB 36600 或地方标准的具体限值\n- 所有技术参数应符合 HJ 25.4 规范要求"

func enrichSoilPrompt(templateName string, basePrompt string) string {
	if templateName == "soil-remediation-bid" {
		return basePrompt + soilPromptSuffix + SoilRemediationKB
	}
	return basePrompt
}
