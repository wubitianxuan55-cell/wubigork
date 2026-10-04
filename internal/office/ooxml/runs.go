package ooxml

// runs.go — 段落按 run 重建的共享核心（2026-10-04 第三轮审计 §3.2：自
// docxedit/pptxedit 两份 rebuildParagraph 逐字收敛分组循环与受影响区间
// 计算；发射端是两个方言——docx 带 w:ins/w:del 修订记号、pptx 纯 a:t
// 替换——各自保留在原包，勿顺手对齐）。

// TextSegCore 是 run 内文本段的共用数据面：两包各自的 textSeg（docx 侧多
// 一个 link 字段）在 rebuildParagraph 入口适配转换到此。
type TextSegCore struct {
	Text             string
	RunTagStart      int    // run 起始标签起点（如 <w:r w:rsidRPr=".."> / <a:r>）
	RunTagEnd        int    // run 起始标签终点
	RunEnd           int    // run 结束标签终点（含 </w:r> / </a:r>）
	RPrStart, RPrEnd int    // run 内 rPr 原始字节区间；无则 -1
	TAttrs           string // 文本起始标签属性子串（不含 <w:t|<a:t 与 >）
}

// RunGroup 是同一 run 起始标签下的文本段聚合（跨 w:t/a:t 分片的 run 归一）。
type RunGroup struct {
	TagStart, TagEnd int
	End              int
	RPrStart, RPrEnd int
	TAttrs           string
	Segs             []TextSegCore
}

// GroupRuns 按 run 起始标签区间把文本段分组（保序）。
func GroupRuns(segs []TextSegCore) []*RunGroup {
	groups := []*RunGroup{}
	groupByRun := map[[2]int]*RunGroup{}
	for _, seg := range segs {
		key := [2]int{seg.RunTagStart, seg.RunTagEnd}
		g := groupByRun[key]
		if g == nil {
			g = &RunGroup{TagStart: seg.RunTagStart, TagEnd: seg.RunTagEnd, End: seg.RunEnd,
				RPrStart: -1, RPrEnd: seg.RPrEnd, TAttrs: seg.TAttrs}
			groupByRun[key] = g
			groups = append(groups, g)
		}
		g.Segs = append(g.Segs, seg)
		if seg.RunEnd > g.End {
			g.End = seg.RunEnd
		}
		if seg.RPrStart >= 0 && (g.RPrStart < 0 || seg.RPrStart < g.RPrStart) {
			g.RPrStart = seg.RPrStart
		}
		if seg.RPrEnd > g.RPrEnd {
			g.RPrEnd = seg.RPrEnd
		}
	}
	return groups
}

// RunSpan 是一个 run 在段落拼接文本中的字符区间及其被选区覆盖的子区间。
type RunSpan struct {
	G              *RunGroup
	Start, End     int // run 文本区间（rune 偏移）
	DelFrom, DelTo int // run 内删除区间（run 内 rune 偏移）
}

// AffectedSpans 计算各 run 文本区间并与 [s,e) 求交，返回被选区覆盖的 run；
// 命中数为 0 时 ok=false（选区未命中任何文本）。
func AffectedSpans(groups []*RunGroup, s, e int) (affected []*RunSpan, ok bool) {
	cursor := 0
	for _, g := range groups {
		runText := ""
		for _, seg := range g.Segs {
			runText += seg.Text
		}
		runLen := len([]rune(runText))
		rs := RunSpan{G: g, Start: cursor, End: cursor + runLen}
		df := MaxInt(rs.Start, s) - rs.Start
		dt := MinInt(rs.End, e) - rs.Start
		if dt > df {
			rs.DelFrom, rs.DelTo = df, dt
			affected = append(affected, &rs)
		}
		cursor += runLen
	}
	return affected, len(affected) > 0
}
