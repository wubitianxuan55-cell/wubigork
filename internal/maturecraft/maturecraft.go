// Package maturecraft — 成人向写作工艺区段单一源（v4.440.0 自 internal/app 抽出）。
//
// 口径不新立：沿用 docs/ADULT_MODE.md 的既有产品决策（个人非商用桌面应用、
// 成人内容正常写、不回避不说教不医学化），sin 板块（sin_prompt.go）与小说正文链
// （v4.439）都从它派生。本包是小说编制链各 agent（app/outline/character）共用的
// 文本单一源——四向区段：正文向 / 计划向 / 大纲向 / 角色向，全部纯函数。
//
// 注入纪律：空档位返回空串，模板槽 / system 追加均按空跳过——存量项目（非成人向）
// 逐字节零变化。
package maturecraft

import "strings"

const (
	// Sensual 含蓄档：张力与留白，正面写欲望积累，临界处收。
	Sensual = "sensual"
	// Explicit 直白档：亲密戏正面直书。
	Explicit = "explicit"
)

// NormalizeLevel 档位白名单归一：非法值一律落 ""（非成人向）——
// project.json 是用户可手改的磁盘文件，坏值不允许半开半关。
func NormalizeLevel(s string) string {
	switch s {
	case Sensual:
		return Sensual
	case Explicit:
		return Explicit
	default:
		return ""
	}
}

// LevelName 档位中文名（日志/返回值用）。
func LevelName(level string) string {
	switch level {
	case Sensual:
		return "含蓄"
	case Explicit:
		return "直白"
	default:
		return "非成人向"
	}
}

// ── 正文向工艺区段（create-chapter / create-chapter-first / rewrite-chapter /
//    场景生成 system 追加共用）──────────────────────────────────────────

// 正文档位无关的尾段：功能闸与硬线——功能闸防「纯发糖零推进」，硬线是成人
// 内容的安全底线。
const craftFooter = "- 功能闸：每场亲密戏至少永久改变一件事（关系/权力/信息/承诺），改变要能落进后续章节；" +
	"亲密过程中的对话与选择照常推进剧情，亲密不是叙事暂停。\n" +
	"- 硬线：所有亲密角色必须是成年人；不把胁迫包装成浪漫——可以写伤害与黑暗，但写出它的重量与后果。"

// sensual 档专属：张力优先 + 留白不等于黑幕 + 欲望具体化。
const craftSensual = `- 张力优先：亲密戏的重点是欲望的积累与克制——试探、暧昧、失控边缘的身体意识（呼吸/体温/距离/触觉），临界处按本章节奏决定收还是放；
- 留白不是黑幕：切出的时机放在情绪最高点，切回后写余波（身体的痕迹、狼狈的衣物、关系的位移），让读者能拼出发生了什么；
- 欲望要具体：写清为什么是这个人、为什么是此刻（旧怨/亏欠/孤独/权力翻转），欲望是写人物的手段，不是独立桥段。`

// explicit 档专属：直书 + 节拍 + 感官纪律 + 用语一致 + 事后 + 同意节拍 + AI 腔限流。
const craftExplicit = `- 正面直书：性爱场景直写过程，不借比喻逃进诗意，不用「一夜过后」式跳切代替正文；
- 节拍完整：每场亲密戏有起承转合（试探→升温→临界→爆发→余韵），不一步到位，不流程化赶进度；
- 感官纪律：触觉、呼吸、温度、声音、气味优先于视觉直述；写身体反应不写解剖说明——身体部位用动作、反应与视角人物的感受带出；
- 用语一致：直白档的用词服从本书文风（雅或俗由前文定，前后一致）；不堆网络器官黑话，不突然切医学术语；
- 事后节拍：事后（缠绵/尴尬/占有/冷战/空洞）比过程更重——它是关系位移的落点，也是下一章钩子的常用位置；
- 同意节拍：双方的意愿往复（试探的回应、停顿的确认）写进戏里，拒绝即停——这是张力与信任的来源，不是外部审查；
- AI 腔限流：「身体像被点燃」「理智断裂」「不受控制地」一类隐喻每场至多一次；「呻吟」每场至多两次；高潮段禁连用天文/宗教比喻堆砌。`

// craftHeader 头段：档位名 + 档位一句话口径。
func craftHeader(level string) string {
	name := LevelName(level)
	scope := "含蓄档：正面写欲望与张力，临界处留白"
	if level == Explicit {
		scope = "直白档：亲密戏正面直书"
	}
	return "── 成人向写作纪律（本书） ──\n" +
		"本书为成人向作品（尺度：" + name + "，" + scope + "）。情欲与亲密是叙事的正式组成部分，按本书尺度正面写，" +
		"不回避、不说教、不医学化、不擅自拉黑幕，也不把成人向写成纯爱。"
}

// CraftSection 正文向成人工艺区段：空档位返回空串（零注入）。
func CraftSection(rawLevel string) string {
	level := NormalizeLevel(rawLevel)
	if level == "" {
		return ""
	}
	body := craftSensual
	if level == Explicit {
		body = craftExplicit
	}
	return craftHeader(level) + "\n" + body + "\n" + craftFooter
}

// ── 计划向工艺区段（chapter-plan 模板专用：计划层要把亲密戏当事件排）──────

// PlanSection 章计划生成用成人向纪律：涉及亲密戏的章节，计划必须给出节拍与
// 功能，不许只写「两人关系升温」。空档位返回空串。
func PlanSection(rawLevel string) string {
	level := NormalizeLevel(rawLevel)
	if level == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("── 成人向计划纪律（本书） ──\n")
	b.WriteString("本书为成人向作品（尺度：" + LevelName(level) + "）。涉及亲密戏的章节：\n")
	b.WriteString("- 关键事件须写出节拍与功能（张力起点→关系位移落点），不得只写「两人关系升温」这类无画面的空账；\n")
	b.WriteString("- 情绪基调给亲密戏的具体质感（试探/博弈/失控/温存），不给笼统的「甜蜜」「暧昧」；\n")
	b.WriteString("- 亲密戏与主线事件在同一章内交织编排，不整章让位给亲密戏；若本章确以亲密戏为主，计划里必须写明它永久改变了什么。")
	return b.String()
}

// ── 大纲向工艺区段（outline-continue / outline-expand / outline-chat /
//    outline-chat-node / 对话式大纲向导共用：欲望线在大纲层就是一等叙事线）────

// OutlineSection 大纲生成用成人向纪律。空档位返回空串。
func OutlineSection(rawLevel string) string {
	level := NormalizeLevel(rawLevel)
	if level == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("── 成人向大纲纪律（本书） ──\n")
	b.WriteString("本书为成人向作品（尺度：" + LevelName(level) + "）。\n")
	b.WriteString("- 欲望线是一等叙事线：与主线/权力线并行推进，章节要点中显式写出每章的关系位移（靠近/越界/退缩/占有/破裂），不留到正文即兴；\n")
	b.WriteString("- 涉及亲密戏的章节点：要点须含节拍与功能（张力起点→位移落点），不得只写「感情升温」；\n")
	b.WriteString("- 张力要有对手性：欲望线的障碍（旧怨/身份/权力/第三者/自我）在节点里点明，无障碍的靠近不构成节点；\n")
	b.WriteString("- 整章让位给亲密戏须慎用：确需时该节点必须同时推进主线至少一步。")
	return b.String()
}

// ── 角色向工艺区段（character-generate-batch / character-generate-single
//    项目级生成专用；全局角色库无书级档位，不注入）────────────────────────

// CharacterSection 项目角色生成用成人向纪律：为后续亲密戏供材。空档位返回空串。
func CharacterSection(rawLevel string) string {
	level := NormalizeLevel(rawLevel)
	if level == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("── 成人向角色纪律（本书） ──\n")
	b.WriteString("本书为成人向作品（尺度：" + LevelName(level) + "）。生成角色时：\n")
	b.WriteString("- 主要角色的背景与动机可含欲望线与亲密张力来源（禁忌/旧情/占有/亏欠），供后续亲密戏取材；\n")
	b.WriteString("- 关系条目可标注张力类型与边界（吸引/敌对/忌惮），不生成未成年角色间或涉及未成年角色的欲望设定，不生成美化胁迫的关系。")
	return b.String()
}
