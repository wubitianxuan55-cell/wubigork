package novelstyle

// 风格学习回灌（长篇刀5）：把统计指纹编译成可执行写作指令。
// 数值特征 → 自然语言约束（句长节奏/段落/对话占比/四字格/连接词/形副密度/
// 标点习惯/签名词），供生成时注入（零 digest 零注入）。

import (
	"fmt"
	"strings"
)

// DigestOf 从指纹编译风格指令集。fp 为 nil 返回空指令（调用方据此判定零注入）。
func DigestOf(fp *Fingerprint) string {
	if fp == nil {
		return ""
	}
	var b strings.Builder
	line := func(format string, args ...interface{}) {
		fmt.Fprintf(&b, "- "+format+"\n", args...)
	}
	// 句长节奏：均值 + 离散度（交错 vs 均匀）
	if fp.SentenceLen.Mean > 0 {
		if fp.SentenceLen.Sd > fp.SentenceLen.Mean*0.7 {
			line("句子长短交错明显（平均约 %.0f 字，标准差大）：短句作重拍、长句作铺陈，保持这种呼吸", fp.SentenceLen.Mean)
		} else {
			line("句子平均约 %.0f 字、节奏均匀：以逗号长句为主，避免把句子拆得太碎", fp.SentenceLen.Mean)
		}
	}
	if fp.ParaLen.Mean > 0 {
		if fp.ParaLen.Mean < 80 {
			line("段落短促（平均约 %.0f 字）：一段一个镜头，勤分段", fp.ParaLen.Mean)
		} else {
			line("段落平均约 %.0f 字，按戏剧单元分段，不要堆超长段", fp.ParaLen.Mean)
		}
	}
	// 对话占比
	switch {
	case fp.DialogRatio > 0.45:
		line("对话驱动（约占 %.0f%%）：多用对话推进，叙述为对话服务", fp.DialogRatio*100)
	case fp.DialogRatio < 0.15:
		line("叙述为主（对话仅约 %.0f%%）：对话惜字如金，只在交锋处开口", fp.DialogRatio*100)
	default:
		line("对话与叙述均衡（对话约 %.0f%%）", fp.DialogRatio*100)
	}
	// 四字格 / 连接词 / 形副密度（作者低用 → 明确避免）
	if fp.FourCharRatio < 2 {
		line("四字格克制（作者密度 %.1f/千字）：避免成语连用，一段至多一处", fp.FourCharRatio)
	}
	if fp.ConnectiveDensity < 3 {
		line("连接词克制（密度 %.1f/千字）：少用「然而/此外/与此同时」开头，靠内容本身衔接", fp.ConnectiveDensity)
	}
	if fp.AdjAdvDensity < 20 {
		line("形容副词稀薄（密度 %.1f/千字）：用动作与名词说话，不堆修饰", fp.AdjAdvDensity)
	}
	// 标点习惯
	if fp.Punctuation.CommaPerSentence > 2.5 {
		line("长句用逗号分层（平均每句 %.1f 个逗号）：保留这种绵延感", fp.Punctuation.CommaPerSentence)
	}
	if fp.Punctuation.Ellipsis <= 0.5 && fp.Punctuation.Dash <= 0.5 {
		line("少用省略号与破折号（作者几乎不用）")
	}
	// 词汇丰富
	if fp.TTRSd > 8 {
		line("词汇丰富多变（TTR 波动 %.1f）：避免同一词反复出现", fp.TTRSd)
	}
	// 签名词
	if n := len(fp.AuthorSignWords); n > 0 {
		words := fp.AuthorSignWords
		if n > 8 {
			words = words[:8]
		}
		line("惯用表达（自然融入，勿刻意堆砌）：%s", strings.Join(words, "、"))
	}
	return strings.TrimRight(b.String(), "\n")
}
