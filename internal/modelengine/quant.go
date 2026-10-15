package modelengine

// ── 量化档位解析与工具调用警告（阶段十 10.1 量化红线）────────
//
// 证据（competitor survey + 社区共识）：本地模型量化档位 <Q4 时工具调用
//（结构化输出/函数参数 JSON）崩溃率显著上升——「工具调用低比特红线」。
// 本文件做**确定性解析**（llama.cpp GGUF 量化命名法，纯函数零 IO）+ 警告
// 文案生成；警告的发射时机与去重由调用方（app 路由层）负责。
//
// 诚实口径：模型名无量化标记时 known=false（不警告）——宁漏勿误，不凭空
// 猜测未标注模型的量化档位。

import "strings"

// QuantLevel 量化档位解析结果。
type QuantLevel struct {
	Level int    // 整数档位（Q4_K_M→4，IQ2_XS→2，Q8_0→8，F16→16）；known=false 时 0
	Label string // 原始标记（如 "Q4_K_M"）
	Known bool   // 模型名含可识别的量化标记
}

// ParseQuantLevel 从模型名解析量化档位。识别 llama.cpp GGUF 命名法：
//   - 标准系列：Q2_K / Q3_K_S/M/L / Q4_0 / Q4_K_S/M / Q5_0/1 / Q5_K_S/M /
//     Q6_K / Q8_0（level = Q 后数字）
//   - I 系列：IQ1_S/M / IQ2_XXS/XS/S/XXXS/M / IQ3_XXS/XS/S/M / IQ4_XS/NL
//     （level = IQ 后数字）
//   - 全精度：F16 / BF16 / FP16 / F32（level 16/32，恒达标）
//
// 匹配不区分大小写；分隔符（- _ . 空格）不阻断。多个标记取**最低档**
// （保守：混合标注按最差档警告）。
func ParseQuantLevel(model string) QuantLevel {
	if model == "" {
		return QuantLevel{}
	}
	upper := strings.ToUpper(model)
	// 分词：按非字母数字切（保留 Q/IQ/F/BF/FP 前缀与其数字的粘连）。
	tokens := splitQuantTokens(upper)
	lowest := 0
	lowestLabel := ""
	known := false
	for _, tok := range tokens {
		lvl, label, ok := quantOfToken(tok)
		if ok && (lowest == 0 || lvl < lowest) {
			lowest, lowestLabel, known = lvl, label, true
		}
	}
	if !known {
		return QuantLevel{}
	}
	return QuantLevel{Level: lowest, Label: lowestLabel, Known: true}
}

// splitQuantTokens 大写串按非 [A-Z0-9_] 切词——下划线保留在词内（量化标记
// 本身含下划线：Q4_K_P / IQ2_XS / Q8_0，切断会丢档位后缀）。
func splitQuantTokens(upper string) []string {
	var out []string
	var cur strings.Builder
	for _, r := range upper {
		if r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
			cur.WriteRune(r)
			continue
		}
		if cur.Len() > 0 {
			out = append(out, strings.Trim(cur.String(), "_"))
			cur.Reset()
		}
	}
	if cur.Len() > 0 {
		out = append(out, strings.Trim(cur.String(), "_"))
	}
	return out
}

// quantOfToken 单词 → 档位。返回 ok=false 表示该词不是量化标记。
func quantOfToken(tok string) (level int, label string, ok bool) {
	switch {
	// 全精度
	case tok == "F16" || tok == "BF16" || tok == "FP16":
		return 16, tok, true
	case tok == "F32" || tok == "FP32":
		return 32, tok, true
	// I 系列：IQ<数字><后缀>
	case strings.HasPrefix(tok, "IQ"):
		if n := leadingDigits(tok[2:]); n > 0 {
			return n, tok, true
		}
		return 0, "", false
	// 标准系列：Q<数字>（Q4_0 / Q4_K_S 的 K/S 后缀不影响档位）
	case strings.HasPrefix(tok, "Q"):
		if n := leadingDigits(tok[1:]); n > 0 {
			return n, tok, true
		}
		return 0, "", false
	}
	return 0, "", false
}

// leadingDigits 前导数字解析（"4_K"→4，"2XXS"→2，"K"→0）。
func leadingDigits(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n
}

// ToolCallQuantWarning 工具调用低比特红线（10.1）：本地引擎 + 已识别量化档
// <4 → 返回中文警告文案；云端引擎 / 未识别 / 达标 → 空串。
//
// 证据：低比特模型跑工具调用（函数参数 JSON 结构化输出）崩溃率显著上升
// （competitor survey §5.3 社区共识）——Q4 是社区公认的可用下限。
func ToolCallQuantWarning(engineType EngineType, model string) string {
	if !engineType.IsLocal() {
		return ""
	}
	q := ParseQuantLevel(model)
	if !q.Known || q.Level >= 4 {
		return ""
	}
	return "模型「" + model + "」为 " + q.Label +
		" 低比特量化：工具调用（结构化输出）在此档位下崩溃率高，建议换用 Q4 及以上档位跑 agent 任务"
}

// ── 量化档位 × 任务面适用性（阶段十 10.2，survey §5.3 量化共识）──
//
// 共识口径（survey §5.3 原文）：
//   - 「文风对低量化的容忍度远高于知识（知识低量化非线性崩塌）」；
//   - 「Midnight Miqu IQ2_S 仍出佳品」= IQ2 档写作可行实证；
//   - 工具调用 <Q4 崩溃率高（见 ToolCallQuantWarning）。
// 据此三档适用面：
//   writing  写作/文笔——容忍度最高，IQ2 起可读（低量化写作可行）
//   knowledge 知识/事实问答——低量化非线性崩塌，建议 ≥Q4
//   agent    工具调用/agent 编排——<Q4 崩溃率高，建议 ≥Q4（红线见上）

// QuantSuitability 返回模型的任务面适用面标注（人可读，模型中心推荐位/绑定
// 卡展示用）。known=false 返回空串（无标记不猜测——宁缺勿误）。
func QuantSuitability(model string) string {
	q := ParseQuantLevel(model)
	if !q.Known {
		return ""
	}
	switch {
	case q.Level >= 8:
		return "全档位适用（Q8+/全精度无损）"
	case q.Level >= 6:
		return "全档位适用（Q6 起接近无损）"
	case q.Level == 5:
		return "写作/知识/agent 均适用"
	case q.Level == 4:
		return "写作/知识/agent 均适用（Q4 甜点档）；复杂 agent 编排建议 Q5+"
	case q.Level == 3:
		return "写作可用；知识问答开始失真、agent 工具调用不适用（建议 ≥Q4）"
	case q.Level == 2:
		return "写作可用（低量化写作有实证）；知识/agent 不适用（建议 ≥Q4）"
	default: // 1
		return "仅实验用途；各任务面均不可靠"
	}
}
