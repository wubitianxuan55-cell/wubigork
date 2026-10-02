package app

// ── AP1-09 消费端点回归：收敛闭环（converge_handler.go）的 S1/S2 分流 ──
//
// convergeCheck 此前内联 `is.Severity == "S1" || is.Severity == "S2"`（大小写敏感），
// 是审计点名的 5 处之一。本用例走真实判据（novelgate.ChapterQualityIssues）钉住：
//   - 空正文 = chapter_empty（S1）必须落进 S1S2 阻断桶，Converged=false；
//   - 大段正常正文（无 S1/S2 问题）落 S3 桶，S1S2 为空。
//
// 敏感点：若把 severityBlocking 的归一化改成大小写敏感（旧形态），S1/S2 分流本身
// 在本例仍成立（生产侧恒发大写），所以本用例定位是「消费端不回归」，真正的
// 归一化口径由 plan_severity_test.go 的四形态用例钉住。

import (
	"strings"
	"testing"
)

func TestConvergeCheckSeverityBuckets(t *testing.T) {
	// 空正文：deterministic S1（chapter_empty）→ S1S2 桶，必须判未收敛。
	ck := convergeCheck("   ", 40)
	if len(ck.S1S2) != 1 || ck.S1S2[0].Code != "chapter_empty" || ck.S1S2[0].Severity != "S1" {
		t.Fatalf("空正文应落 S1S2 桶: %+v", ck)
	}
	if len(ck.S3) != 0 {
		t.Fatalf("空正文不应有 S3 项: %+v", ck.S3)
	}
	if ck.Converged {
		t.Fatalf("有 S1/S2 阻断项时不得判收敛: %+v", ck)
	}

	// 正常长文：S1S2 桶应为空（S3 允许有，不影响 Converged 判定）。
	normal := strings.Repeat("秦昭把剑放在膝上，听着檐外的雨声，慢慢说起当年那场没打完的架，语气平得像是别人的事。", 12)
	ck = convergeCheck(normal, 40)
	for _, is := range ck.S1S2 {
		if !severityBlocking(is.Severity) {
			t.Fatalf("S1S2 桶里出现非阻断级项: %+v", is)
		}
	}
	for _, is := range ck.S3 {
		if severityBlocking(is.Severity) {
			t.Fatalf("阻断级项不得落 S3 桶（会静默放过）: %+v", is)
		}
	}
}
