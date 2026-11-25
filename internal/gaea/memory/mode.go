package memory

// mode.go — 记忆的运行模式维度（2026-11-25 办公工具梳理批）。
//
// 记忆此前只有「工作区 × 板块空间」两维：桶键是工作区绝对路径，空间列分办公/
// 图文。办公三档运行模式（标准/造价/方案）是后加的组合声明层，没给记忆层补
// 这一维——造价档存下的取费规矩照样注进方案档（用户实测「不同模式的都混在
// 一起」）。本文件补上：
//
//   - 存储：facts.mode_id（SchemaV42）。空=通用；非空=所属运行模式（cost/
//     proposal…）。存量记忆没有标记，一律按通用处理（不回填、不猜）。
//   - 写入默认规则（resolveRememberMode）：用户画像与工作方式反馈（type=user/
//     feedback）跨模式通用；项目事实、参考资料、过程规矩默认归属当前模式。
//     模型可用 scope 显式指定 general / mode；按名修订已有记忆时不改归属。
//   - 读取（modeView）：会话只看到「通用 + 本模式」。标准档只看通用；模式维度
//     未启用的引擎（图文板块、未设 Mode 的装配）不过滤，行为逐字节不变。
//     管理面（记忆面板 hubOfficeStore）不走视图，各模式都看得到。
//
// 这里只管「哪些记忆进本会话的上下文」，是取数范围不是行为闸——跨模式的知识
// 没有丢，面板里都在，换个模式就在。

import (
	"fmt"
	"strings"
)

const (
	// ModeStandard 标准档：只看通用记忆（模式专属的不注入）。
	ModeStandard = "standard"
	// ModeGeneral 显式通用的内存态标记：落库前归一成空串。用来区分「模型明确
	// 说这条通用」与「没说」——后者才按默认规则盖当前模式。
	ModeGeneral = "-"
)

// modeLabels 运行模式中文名（索引标签与面板展示共用）。未登记的原样返回。
var modeLabels = map[string]string{
	ModeStandard: "通用",
	"cost":       "造价",
	"proposal":   "方案",
}

// ModeLabel 返回运行模式的中文名；空串=通用。
func ModeLabel(mode string) string {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		return modeLabels[ModeStandard]
	}
	if label, ok := modeLabels[mode]; ok {
		return label
	}
	return mode
}

// renderIndexTagged 渲染注入用索引，并给模式专属的条目打上模式名（〔造价〕）：
// 通用条目不加标签，与未启用模式维度时的索引逐字节一致。
func renderIndexTagged(mems []Memory, sessionMode string) string {
	sessionMode = strings.TrimSpace(sessionMode)
	if sessionMode == "" {
		return renderIndex(mems)
	}
	if len(mems) == 0 {
		return ""
	}
	var buf strings.Builder
	buf.WriteString("# Memory\n\n")
	for _, m := range mems {
		kindTag := ""
		switch NormalizeKind(string(m.Kind)) {
		case KindEpisodic:
			kindTag = "[E] "
		case KindProcedural:
			kindTag = "[P] "
		}
		modeTag := ""
		if strings.TrimSpace(m.Mode) != "" {
			modeTag = "〔" + ModeLabel(m.Mode) + "〕"
		}
		fmt.Fprintf(&buf, "- %s[%s](%s.md)%s — %s\n", kindTag, displayTitle(m.Title, m.Name), m.Name, modeTag, oneLine(m.Description))
	}
	return strings.TrimSpace(buf.String())
}

// modeVisible 报告一条记忆在给定会话模式下是否可见。sessionMode 为空=模式
// 维度未启用，全部可见；标准档只看通用；其他档=通用 + 本模式。
func modeVisible(sessionMode string, m Memory) bool {
	sessionMode = strings.TrimSpace(sessionMode)
	if sessionMode == "" {
		return true
	}
	mode := strings.TrimSpace(m.Mode)
	return mode == "" || (sessionMode != ModeStandard && mode == sessionMode)
}

// resolveRememberMode 算一条新记忆的归属标记（返回值交给 normalizeModeForSave
// 归一落库）。scope：general=通用；mode=当前模式（标准档/未启用时等于通用）；
// 其他（含空）=默认规则——type=user/feedback 通用，其余归当前模式。
func resolveRememberMode(scope string, t Type, sessionMode string) string {
	sessionMode = strings.TrimSpace(sessionMode)
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "general":
		return ModeGeneral
	case "mode":
		if sessionMode == "" || sessionMode == ModeStandard {
			return ModeGeneral
		}
		return sessionMode
	}
	if sessionMode == "" || sessionMode == ModeStandard || t == TypeUser || t == TypeFeedback {
		return ModeGeneral
	}
	return sessionMode
}

// normalizeModeForSave 把内存态的归属标记归一成落库值：显式通用归空；未标注
// 且会话模式启用时按默认规则盖当前模式（非 remember 入口——面板快捷添加、
// 梦境蒸馏、会话事实转正——写进来的也走这条，造价会话里蒸馏出的条目不会
// 因为没带标记而变成通用）。
func normalizeModeForSave(m *Memory, sessionMode string) {
	sessionMode = strings.TrimSpace(sessionMode)
	switch {
	case strings.TrimSpace(m.Mode) == ModeGeneral:
		m.Mode = ""
	case strings.TrimSpace(m.Mode) != "":
		// 已显式指定：落库值只认通用或当前模式，别的模式的名字不代存。
		if sessionMode == "" || m.Mode == sessionMode {
			return
		}
		m.Mode = ""
	default:
		if r := resolveRememberMode("auto", m.Type, sessionMode); r != ModeGeneral {
			m.Mode = r
		}
	}
}
