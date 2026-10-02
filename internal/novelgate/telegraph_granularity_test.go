package novelgate

// 审计 IN1-04 对照用例：gate 与 novelreview 是两套确定性体检，同一个「电报体」域
// 各持**不同粒度**的判据——
//   - gate（本包）：句级。≥20 句、短句(≤5字)占比 >40% 且平均句长 <10 → S2（app 侧
//     severityBlocking 视 S1/S2 为阻断，进收敛链）；
//   - novelreview：段级。format_readability 维度，≥12 段且平均段长 ≤12 字 → WARN
//     （建议级，只进面板结论计数）。
//
// 两判据源自上游 quality-rubric 的两个不同维度行（句长节奏 / 格式可读性），阈值各自
// 独立、**刻意分层**（阻断信号 vs 建议信号），不是门槛漂移——勿用本文件的矛盾样本去
// 「对齐」任何一侧的默认值。本文件把这个分层事实变成机器可见：两个方向的单侧命中
// 各钉一例，其中方向①同时是「gate 报 S2 阻断、review 整体 APPROVE」的既证矛盾样本。

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/novelreview"
)

// telegraphPool 全短句池：每句内容 ≤5 字（gate 口径的「短句」），并自带情绪节点
// （笑/哭/慌，novelreview emotion 词表）供评审侧密度维度达标。
var telegraphPool = []string{
	"他站住。", "她回头。", "风很大。", "灯灭了。", "雨停了。",
	"门开了。", "他笑了。", "她哭了。", "天黑了。", "路很长。",
	"水很凉。", "手很冷。", "心很慌。", "他追上去。", "她没停。",
}

// telegraphDialog 台词块：内容全部 ≤5 字（不破坏 gate 短句占比），引号内非空白
// 字数供 novelreview 对话占比达标；也充当开篇钩子维度的「台词」信号。
const telegraphDialog = "「你站住。别过来。滚出去。我不认识你。放开我。」"

// buildTelegraphWall 四大段全短句墙（每段 ~400 字）：段少而长 → novelreview 的
// 段级电报体判据（≥12 段）够不着，但 gate 的句级判据全额命中。
func buildTelegraphWall() string {
	pool := strings.Join(telegraphPool, "")
	var paras []string
	for i := 0; i < 4; i++ {
		// 每段：1 个台词块 + 5 轮短句池（对话占比、开篇钩子、情绪密度同时喂饱）。
		paras = append(paras, telegraphDialog+strings.Repeat(pool, 5))
	}
	// 章尾挂一个短问句：novelreview 章尾钩子维度按「疑问」判 pass。
	paras[3] += "门外是谁？"
	return strings.Join(paras, "\n")
}

// TestIN104_GateTelegraphBlocksWhileReviewApproves 方向①（矛盾样本）：
// 同一正文——gate 报 telegraph_style S2（阻断级），novelreview 整体 APPROVE。
// 这不是缺陷，是句级/段级分层的既证事实；若未来此断言变红（review 不再 APPROVE
// 或 gate 不再报 S2），说明有人动了任一侧判据语义，须回到两处注释核对分层约定。
func TestIN104_GateTelegraphBlocksWhileReviewApproves(t *testing.T) {
	text := buildTelegraphWall()

	// gate：句级判据命中，且是阻断级 S2。
	got := codes(ChapterQualityIssues(text))
	if got["telegraph_style"] != "S2" {
		t.Fatalf("全短句墙应报 telegraph_style/S2（阻断级），实际: %+v", got)
	}

	// novelreview：段级判据够不着（4 段 < 12），format_readability 只按长段堆叠给
	// FAIL/S3（计数 < 3 且无 S1/S2）→ 整体 APPROVE。
	rep, err := novelreview.Review(text, "general", novelreview.Options{})
	if err != nil {
		t.Fatalf("Review 失败: %v", err)
	}
	if rep.Verdict != novelreview.VerdictApprove {
		var bad []string
		for _, d := range rep.Dimensions {
			if d.Verdict == "warn" || d.Verdict == "fail" {
				bad = append(bad, d.ID+"="+d.Verdict+"/"+d.Severity+"("+d.Detail+")")
			}
		}
		t.Fatalf("同一正文 novelreview 应 APPROVE（gate 侧 S2 见上），实际 %s counts=%v 命中维度: %v",
			rep.Verdict, rep.Counts, bad)
	}
	var pace *novelreview.Dimension
	for i := range rep.Dimensions {
		if rep.Dimensions[i].ID == "format_readability" {
			pace = &rep.Dimensions[i]
		}
	}
	if pace == nil {
		t.Fatal("报告缺 format_readability 维度")
	}
	if strings.Contains(pace.Detail, "电报体") {
		t.Fatalf("段级电报体判据不应命中（段少而长）: %s", pace.Detail)
	}
}

// TestIN104_ReviewParagraphTelegraphGateClean 方向②（反向不对称）：
// 「一段一句」×20 段——novelreview 的段级电报体判据命中（≥12 段且平均段长 ≤12），
// gate 的句级判据全额够不着（每句 10 字：不算短句、平均句长恰在门槛上）。
// 证明两判据互不覆盖，谁也不是谁的子集。
func TestIN104_ReviewParagraphTelegraphGateClean(t *testing.T) {
	// 句内容 10 字（含逗号）：gate 短句口径 ≤5 不命中；平均句长 = 10，不触发
	// gate 的 avg < 10；段落 11 字 ≤ 12，20 段 ≥ 12 → review 段级判据命中。
	line := "他走进屋里，雨还没停。"
	text := strings.Repeat(line+"\n", 20)

	if got := ChapterQualityIssues(text); len(got) != 0 {
		t.Fatalf("段级碎段不应触发 gate 句级判据，实际: %+v", got)
	}

	rep, err := novelreview.Review(text, "general", novelreview.Options{})
	if err != nil {
		t.Fatalf("Review 失败: %v", err)
	}
	var pace *novelreview.Dimension
	for i := range rep.Dimensions {
		if rep.Dimensions[i].ID == "format_readability" {
			pace = &rep.Dimensions[i]
		}
	}
	if pace == nil {
		t.Fatal("报告缺 format_readability 维度")
	}
	if pace.Verdict != "warn" || !strings.Contains(pace.Detail, "电报体") {
		t.Fatalf("段级电报体应 warn 并点名电报体，实际 %s（%s）", pace.Verdict, pace.Detail)
	}
}
