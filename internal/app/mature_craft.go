package app

import "strings"

// mature_craft.go — 成人小说编制工艺区段（v4.439.0）。
//
// 口径不新立：沿用 docs/ADULT_MODE.md 的既有产品决策（个人非商用桌面应用、
// 成人内容正常写、不回避不说教不医学化），sin 板块（sin_prompt.go）已按此落过
// 一刀；本文件把同一口径带进小说编制链——差异只在粒度：小说是「一本书一个
// 定位」，档位挂 ProjectMeta.Mature（""=非成人向 / sensual=含蓄 / explicit=直白），
// 不做全局开关、不做年龄门（同 ADULT_MODE §一：不设开关门禁）。
//
// 注入纪律沿用章节生成链既有惯例：空档位返回空串，模板槽 / system 追加均按
// 空跳过——存量项目（非成人向）逐字节零变化。

const (
	// MatureSensual 含蓄档：张力与留白，正面写欲望积累，临界处收。
	MatureSensual = "sensual"
	// MatureExplicit 直白档：亲密戏正面直书。
	MatureExplicit = "explicit"
)

// normalizeMatureLevel 档位白名单归一：非法值一律落 ""（非成人向）——
// project.json 是用户可手改的磁盘文件，坏值不允许半开半关。
func normalizeMatureLevel(s string) string {
	switch s {
	case MatureSensual:
		return MatureSensual
	case MatureExplicit:
		return MatureExplicit
	default:
		return ""
	}
}

// matureLevelName 档位中文名（日志/返回值用）。
func matureLevelName(level string) string {
	switch level {
	case MatureSensual:
		return "含蓄"
	case MatureExplicit:
		return "直白"
	default:
		return "非成人向"
	}
}

// ── 正文向工艺区段（create-chapter / create-chapter-first / rewrite-chapter /
//    场景生成 system 追加共用）──────────────────────────────────────────

// 正文档位无关的头尾两段：头段立口径（对齐 ADULT_MODE 不回避不说教），尾段是
// 功能闸与硬线——功能闸防「纯发糖零推进」，硬线是成人内容的安全底线。
const matureCraftFooter = "- 功能闸：每场亲密戏至少永久改变一件事（关系/权力/信息/承诺），改变要能落进后续章节；" +
	"亲密过程中的对话与选择照常推进剧情，亲密不是叙事暂停。\n" +
	"- 硬线：所有亲密角色必须是成年人；不把胁迫包装成浪漫——可以写伤害与黑暗，但写出它的重量与后果。"

// sensual 档专属：张力优先 + 留白不等于黑幕 + 欲望具体化。
const matureCraftSensual = `- 张力优先：亲密戏的重点是欲望的积累与克制——试探、暧昧、失控边缘的身体意识（呼吸/体温/距离/触觉），临界处按本章节奏决定收还是放；
- 留白不是黑幕：切出的时机放在情绪最高点，切回后写余波（身体的痕迹、狼狈的衣物、关系的位移），让读者能拼出发生了什么；
- 欲望要具体：写清为什么是这个人、为什么是此刻（旧怨/亏欠/孤独/权力翻转），欲望是写人物的手段，不是独立桥段。`

// explicit 档专属：直书 + 节拍 + 感官纪律 + 用语一致 + 事后 + 同意节拍 + AI 腔限流。
const matureCraftExplicit = `- 正面直书：性爱场景直写过程，不借比喻逃进诗意，不用「一夜过后」式跳切代替正文；
- 节拍完整：每场亲密戏有起承转合（试探→升温→临界→爆发→余韵），不一步到位，不流程化赶进度；
- 感官纪律：触觉、呼吸、温度、声音、气味优先于视觉直述；写身体反应不写解剖说明——身体部位用动作、反应与视角人物的感受带出；
- 用语一致：直白档的用词服从本书文风（雅或俗由前文定，前后一致）；不堆网络器官黑话，不突然切医学术语；
- 事后节拍：事后（缠绵/尴尬/占有/冷战/空洞）比过程更重——它是关系位移的落点，也是下一章钩子的常用位置；
- 同意节拍：双方的意愿往复（试探的回应、停顿的确认）写进戏里，拒绝即停——这是张力与信任的来源，不是外部审查；
- AI 腔限流：「身体像被点燃」「理智断裂」「不受控制地」一类隐喻每场至多一次；「呻吟」每场至多两次；高潮段禁连用天文/宗教比喻堆砌。`

// matureCraftHeader 头段：档位名 + 档位一句话口径。
func matureCraftHeader(level string) string {
	name := matureLevelName(level)
	scope := "含蓄档：正面写欲望与张力，临界处留白"
	if level == MatureExplicit {
		scope = "直白档：亲密戏正面直书"
	}
	return "── 成人向写作纪律（本书） ──\n" +
		"本书为成人向作品（尺度：" + name + "，" + scope + "）。情欲与亲密是叙事的正式组成部分，按本书尺度正面写，" +
		"不回避、不说教、不医学化、不擅自拉黑幕，也不把成人向写成纯爱。"
}

// buildMatureCraftSection 正文向成人工艺区段：空档位返回空串（零注入）。
func buildMatureCraftSection(rawLevel string) string {
	level := normalizeMatureLevel(rawLevel)
	if level == "" {
		return ""
	}
	body := matureCraftSensual
	if level == MatureExplicit {
		body = matureCraftExplicit
	}
	return matureCraftHeader(level) + "\n" + body + "\n" + matureCraftFooter
}

// ── 计划向工艺区段（chapter-plan 模板专用：计划层要把亲密戏当事件排）──────

// buildMaturePlanSection 章计划生成用成人向纪律：涉及亲密戏的章节，计划必须
// 给出节拍与功能，不许只写「两人关系升温」。空档位返回空串。
func buildMaturePlanSection(rawLevel string) string {
	level := normalizeMatureLevel(rawLevel)
	if level == "" {
		return ""
	}
	name := matureLevelName(level)
	var b strings.Builder
	b.WriteString("── 成人向计划纪律（本书） ──\n")
	b.WriteString("本书为成人向作品（尺度：" + name + "）。涉及亲密戏的章节：\n")
	b.WriteString("- 关键事件须写出节拍与功能（张力起点→关系位移落点），不得只写「两人关系升温」这类无画面的空账；\n")
	b.WriteString("- 情绪基调给亲密戏的具体质感（试探/博弈/失控/温存），不给笼统的「甜蜜」「暧昧」；\n")
	b.WriteString("- 亲密戏与主线事件在同一章内交织编排，不整章让位给亲密戏；若本章确以亲密戏为主，计划里必须写明它永久改变了什么。")
	return b.String()
}
