package app

// wenxue_skills.go — 图文板块技能面（2026-11-15 刀2）。
//
// 图文引擎的技能装配走 allowlist（boot.Options.SkillFilter）：小说创作域技能
// + 中性元技能可见，办公域技能（公文/办公编辑纪律/CAD/进度/造价/format-convert
// 族/docx·pptx·pdf·xlsx 索引）deny-by-absence——索引/slash/run_skill/
// read_skill/能力抽屉五面同源收口（同一 filter）。
// docx/pptx 技能目录保留：图文导出链（WenxueExportDeliverable）按路径找
// create_docx.py/create_pptx.py 脚本，不走技能索引——藏索引不破导出。
//
// 内嵌三件（SkillExtras，仿 office builtin 先例随二进制分发）：
//   · wenxue-workbench：图文工作台纪律（tuwen 目录约定+专属工具用法）；
//   · story-long-write / story-deslop：自 skills/ 写作域（小说创作页专用面）
//     蒸馏改写的引擎面版本，只对图文会话可见。
// 办公引擎零改动（双 nil=现状逐字节）。

import (
	"github.com/gaea/gaea/internal/gaea/skill"
)

// wenxueAllowedSkills 图文引擎放行技能名单（deny-by-absence）。
var wenxueAllowedSkills = map[string]bool{
	// 小说创作域（引擎面现役 + 写作域蒸馏）
	"novel-agents":     true, // 七角色子代理卡（spawn 模板引用）
	"novel-deslop":     true, // 去 AI 味口径
	"novel-review":     true, // 章节评审
	"story-long-write": true, // 长篇网文工作流（内嵌）
	"story-deslop":     true, // 去 AI 味润色（内嵌）
	"wenxue-workbench": true, // 图文工作台纪律（内嵌）
	// 中性元技能
	"genui":         true, // GenUI 词汇（回答即 UI）
	"skill-creator": true, // 造技能元技能
}

// wenxueSkillFilter 报告技能是否对图文会话可见。
func wenxueSkillFilter(name string) bool {
	return wenxueAllowedSkills[name]
}

// wenxueSkillExtras 图文引擎内嵌技能三件（boot.Options.SkillExtras）。
func wenxueSkillExtras() []skill.Skill {
	return []skill.Skill{
		{
			Name:        "wenxue-workbench",
			Description: "图文工作台纪律 — tuwen 目录约定与专属工具用法（写作会话开场自读）",
			Scope:       skill.ScopeBuiltin,
			Path:        "(builtin)",
			Body:        wenxueWorkbenchBody,
		},
		{
			Name:        "story-long-write",
			Description: "长篇网文写作指导 — 黄金三章/节奏/爽点密度/对话技巧",
			Scope:       skill.ScopeBuiltin,
			Path:        "(builtin)",
			Body:        storyLongWriteBody,
		},
		{
			Name:        "story-deslop",
			Description: "去 AI 味润色 — 句子呼吸/身体反应代情绪词/对话自然度",
			Scope:       skill.ScopeBuiltin,
			Path:        "(builtin)",
			Body:        storyDeslopBody,
		},
	}
}

const wenxueWorkbenchBody = `# 图文工作台纪律（长篇小说创作）

本工作台是长篇小说创作专属板块：工作空间=C:\AI\tuwen 下的当前小说目录，
每部小说一个子目录=独立工作空间。办公域工具（Office/CAD/造价/进度计划/
浏览器自动化）不装配——写作任务只用通用底座与图文专属工具。

## 目录约定（宽容匹配：中文/英文目录名均识别）

- 章节/（或 chapters/）——章节正文，一章一个文件：
  文件名带序号前缀排序（如 001-第一章 标题.md），正文即发布稿。
- 角色/（或 characters/）——角色卡，一角色一个 md 文件：
  首行一级标题=角色名，正文分「设定 / 外貌 / 性格 / 口头禅 / 关系 / 当前状态」。
- 时间线/（或 timeline/）——时间线事件，一事件一文件或单文件分行条目：
  每条带章节锚点（如「第12章」）+ 一句话事件。
- 设定/（或 worldview/）——世界观/体系/势力/地理等设定文档。
- 定稿/（或 final/）——**发布稿专区：只放定稿不放草稿**。用户明确说
  「定稿/入库/发布稿」时用 wenxue_final_write 写入；草稿一律在 章节/ 演进。
  阅读标签页据此目录生成小说目录与阅读窗。
- .gaea/——工作台数据（会话/导出/固定资料），勿手动整理。

## 会话纪律

- 长跑会话：一章一会话或一幕一会话；跨会话连续性靠 时间线/ + 角色卡当前状态
  + @会话 引用（" @会话:标题" 注入历史会话快照，每书检索域天然隔离）。
- 开新章前先读：前一章结尾（衔接）+ 相关角色卡「当前状态」+ 时间线最近事件。
- 写完一章必须回写：角色卡状态演进、时间线新事件、（如有）伏笔登记。
- 固定常用资料（.gaea/pinned.json， WenxuePinMaterial）自动进新会话上下文。

## 专属工具（wenxue_* 族，仅本引擎装配）

- wenxue_chapter_list / wenxue_chapter_read / wenxue_chapter_write：章节结构
  感知读写（序号/标题/字数/前后章衔接回执）。
- wenxue_character_list / wenxue_character_read / wenxue_character_write：
  角色卡读写。
- wenxue_timeline_read / wenxue_timeline_append：时间线条目读/追加。
- wenxue_final_list / wenxue_final_read / wenxue_final_write：定稿域专用
  （发布稿读取/审校对照/入库；域隔离——章节工具写不进定稿目录）。
- wenxue_stats：全书字数/文件统计快照。

## 交付

- 导出：md/docx（ExportMenu）与产物打包（ZipDeliverables）保留；
  网文平台交稿以 txt/docx 为主，正文章节保持纯正文（不带工作台标注）。
`

const storyLongWriteBody = `# 长篇网文写作指导

## 黄金三章原则
第一章必须：
- 前 500 字内出现矛盾/冲突
- 主角性格通过行动体现（不可旁白说明）
- 一个让读者想继续的悬念

## 节奏控制
- 每 2000 字至少一个小高潮（冲突/转折/揭露）
- 章节末留悬念钩子（但不要用"欲知后事如何"）
- 日常/抒情段落控制篇幅为下一个冲突蓄力（关键氛围段可放宽，但不连排）

## 爽点密度
- 网文读者每 3000 字需要一个爽点
- 爽点类型：打脸、升级、获得宝物、真相揭露、情感升温
- 爽点之后必须有代价或新危机，防止节奏泄气

## 对话技巧
- 每个角色有独特的口头禅或句式
- 对话占比约 35-45%
- 对话推动情节，而非解释情节

## 禁止事项
- 大段旁白解释世界观
- 与剧情无关的大段内心独白（关键转折处的心理铺陈不算）
- 战斗描写纯力量对轰（要有策略和代价）
- "原来""竟然""殊不知"等上帝视角词
`

const storyDeslopBody = `# 去 AI 味润色指南

去 AI 味 ≠ 把文字削短。电报体（通篇主谓宾短句、句句独立无修饰）本身就是一种
AI 味——简洁是删掉坏东西之后的结果，不是目标。

## 禁止使用的 AI 套话
以下词汇和句式绝不出现在正文中：
- "总之" "综上所述" "此外" "另外"
- "不仅如此" "更重要的是"
- "值得注意的是" "显而易见"
- "欲知后事如何，且听下回分解" "此章到此结束" "让我们回到"

## 句子的呼吸（核心）
- 主谓宾只是骨架：要有定语、状语、感官与心理的嵌入成分，句子才有肌理
- 长句铺陈蓄势（三四十字以上、带修饰或从句），短句收束点穴——节奏来自两者的对比
- 极短句（3-5 字）是点穴用的，一段至多一处，不要为短而短
- 连续三句不得以相同结构开头，也不得连续三句都是「主语+动词+宾语」的裸句

## 描写原则
- 用身体反应代替情绪词："他握紧拳头" > "他很生气"
- 用具体动作代替抽象状态："她反复翻看那页" > "她很犹豫"
- 场景感：读者能"看到"画面，而非"被告知"信息
- 白描是收，细节是放——有放才有收，不要通篇白描

## 对话自然度
- 口头禅、重复、打断、沉默——真实对话的特征
- 不同角色句式不同：文化水平、性格、情绪
- 对话要有长度差：短答与长话交替，话多话少本身是性格

## 节奏
- 该慢则慢：关键情绪、转折、氛围处允许延宕铺陈，不要一刀切短
- 段落长度服务节奏：铺陈段可以长，冲刺段要短——连续多段等长就是单调
- 每段只讲一件事
`
