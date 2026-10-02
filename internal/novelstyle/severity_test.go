package novelstyle

// 严重度枚举单源（IN1-09）的锁定测试：novelstyle 产出 S1-S4（与 novelgate/
// novelreview 同一枚举，旧 low/medium/high/blocker → S4/S3/S2/S1 线性换名）。
// 分数零漂移由 TestScoreGolden_SeverityRenameZeroDrift 锁定——六个样本的分值
// 在换名前（旧 low/medium/high/blocker 词表）抓取，换名后必须同值：
//   aiText         6×medium(9) + 4×high(16) = 118 → 封顶 100
//   emotion        2×medium(9)              = 18
//   negation       1×high(16)               = 16
//   explanatory    1×medium(9) + 1×low(4)   = 13
//   blacklist      1×medium(9) + 1×high(16) = 25
//   punctEllipsis  1×medium(9) + 2×low(4)   = 17

import (
	"strings"
	"testing"
)

// resetEmbeddedPatterns 恢复内置模式表：TestLoadPatternsFile_FailClosedAndReplace
// 会整体换出全局表且不回滚（替换语义），跨测试有顺序依赖；本文件的三组测试
// 依赖内置默认表，跑前显式复位，保证与文件序无关。
func resetEmbeddedPatterns(t *testing.T) {
	t.Helper()
	patterns.Store(mustLoadEmbeddedPatterns())
	t.Cleanup(func() { patterns.Store(mustLoadEmbeddedPatterns()) })
}

// TestSeverityToWeight_Table 权重表锁：S1-S4 → 30/16/9/4；未知值按 S4 兜底
// （与旧 default→weightLow 口径一致，保证历史/透传脏值不放大扣分）。
func TestSeverityToWeight_Table(t *testing.T) {
	cases := map[string]int{
		"S1":    30,
		"S2":    16,
		"S3":    9,
		"S4":    4,
		"bogus": 4, // 未知档兜底 = S4（最轻）
		"":      4,
	}
	for sev, want := range cases {
		if got := severityToWeight(sev); got != want {
			t.Fatalf("severityToWeight(%q)=%d, want %d", sev, got, want)
		}
	}
}

// TestRuleSeverityLevels 规则产出档位锁：每条规则命中的 issue 档位与
// score.go 中 severityToWeight 注释里的映射一一对应。
func TestRuleSeverityLevels(t *testing.T) {
	resetEmbeddedPatterns(t)
	samples := []struct {
		name string
		text string
		rule string // Reason 子串
		want string
	}{
		{"规则3 情绪直述", "他内心充满了恐惧。", "show-don't-tell", "S3"},
		{"规则6 句长均匀", "他把这一切扛了下来。之所以，他不能退。", "均匀", "S3"},
		{"规则8 标点滥用", "他走了……她没说话……风停了……雨来了……灯灭了……门关了……", "标点滥用", "S4"},
		{"规则9 黑名单", "他思维缜密地推演全局。眼帘低垂。", "黑名单", "S2"},
		{"规则10 否定翻转", "这不是失败，而是他计划的第一步。", "否定铺垫后肯定翻转", "S2"},
		{"规则11 解释腔", "他把这一切扛了下来。之所以，他不能退。", "解释腔标记", "S4"},
	}
	for _, tc := range samples {
		ts, err := ScoreTextNoRef(tc.text)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		found := false
		for _, iss := range ts.Issues {
			if strings.Contains(iss.Reason, tc.rule) {
				found = true
				if iss.Severity != tc.want {
					t.Fatalf("%s 应产出 %s，got %q（reason=%q）", tc.name, tc.want, iss.Severity, iss.Reason)
				}
			}
		}
		if !found {
			t.Fatalf("%s：样本未命中规则 %q，issues=%+v", tc.name, tc.rule, ts.Issues)
		}
	}
}

// TestScoreGolden_SeverityRenameZeroDrift 零漂移 golden：分值在换名前后必须
// 一字不变（IN1-09 硬验收线：AI 味分数零漂移）。任一权重数值被改动，非封顶
// 样本（emotion/negation/explanatory/blacklist/punctEllipsis）立即变红。
func TestScoreGolden_SeverityRenameZeroDrift(t *testing.T) {
	resetEmbeddedPatterns(t)
	samples := []struct {
		name string
		text string
		want int
	}{
		{"aiText 封顶", aiText, 100},
		{"情绪直述", "他内心充满了恐惧。", 18},
		{"否定翻转", "这不是失败，而是他计划的第一步。", 16},
		{"解释腔+句长均匀", "他把这一切扛了下来。之所以，他不能退。", 13},
		{"黑名单+句长均匀", "他思维缜密地推演全局。眼帘低垂。", 25},
		{"标点滥用+句长均匀", "他走了……她没说话……风停了……雨来了……灯灭了……门关了……", 17},
	}
	for _, tc := range samples {
		ts, err := ScoreTextNoRef(tc.text)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if ts.Score != tc.want {
			t.Fatalf("golden 漂移 %s: got %d, want %d（换名前基线值，权重表不得改动）", tc.name, ts.Score, tc.want)
		}
	}
}
