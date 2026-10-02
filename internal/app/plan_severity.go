package app

// ── 计划/质量问题的严重度判据（审计 AP1-09 收口）──
//
// 「什么级别算阻断」此前在 5 处各写一份，且大小写口径不一致：
//   - create_chapter_context.go  planSeverityBlocking  大小写敏感、不 Trim；
//   - novel_plan_handler.go      planBlockingSeverity  ToUpper+TrimSpace；
//   - converge_handler.go        内联 == 比较            大小写敏感、不 Trim；
//   - novel_book_health.go       两处内联 == 比较        大小写敏感、不 Trim。
// 后果：写前硬闸（走前者）与计划落盘校验（走后者）对同一条问题可能结论相反，
// 用户看到「保存被拒但生成却起来了」。此处落唯一判据，五处一律改调本函数。
//
// 判据严格照抄 types.PlanProblem.Severity / types.PlanGateReport.Blocking 的
// 既有契约（S1|S2 阻断、S3|S4 仅提示），只把口径统一为「先 TrimSpace 再 ToUpper」：
// 生产侧（novelgate.PlanContractIssues/OutlineContractIssues/ChapterQualityIssues）
// 恒发大写，归一化是防未来/外部来源混入小写或首尾空白的加固，不改变既有判定。

import "strings"

// severityBlocking 报告某严重度是否属阻断级（S1/S2 阻断；S3/S4/空/未知仅提示）。
// 大小写与首尾空白不敏感（"s1"、" S2 " 均判阻断）。
func severityBlocking(severity string) bool {
	s := strings.ToUpper(strings.TrimSpace(severity))
	return s == "S1" || s == "S2"
}
