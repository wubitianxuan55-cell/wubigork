// Package whisper — sync_light_write.go
// 100% 对齐 ackem memory/syncLightWrite.ts
// 同步轻量规则写入：毫秒级事实写入，供下一轮立即可见

package whisper

import "strings"

// SyncLightWriteArgs 同步轻写入参数
type SyncLightWriteArgs struct {
	SessionID string
	TurnIndex int
	UserMsg   string
	Store     *FactStore
}

// WriteSyncLightFacts 同步轻量事实写入（简化版：按事件类型快速写入）
func WriteSyncLightFacts(args SyncLightWriteArgs) []string {
	if args.Store == nil {
		return nil
	}
	var ids []string

	// 按关键词快速检测并写入事实
	drafts := extractLightFactDrafts(args.UserMsg)
	for _, d := range drafts {
		f := args.Store.Add(MemoryFact{
			Domain:          d.Domain,
			Subcategory:     d.Subcategory,
			Subject:         d.Subject,
			Summary:         d.Summary,
			Weight:          d.Weight,
			Confidence:      d.Confidence,
			SourceSessionID: args.SessionID,
			SourceTurnIndex: args.TurnIndex,
			FactLayer:       "raw",
		})
		ids = append(ids, f.ID)
	}
	return ids
}

type lightFactDraft struct {
	Domain      string
	Subcategory string
	Subject     string
	Summary     string
	Weight      float64
	Confidence  float64
}

func extractLightFactDrafts(msg string) []lightFactDraft {
	var drafts []lightFactDraft
	// 简单关键词→事实映射
	rules := []struct {
		keywords    []string
		domain      string
		subcategory string
		subject     string
		template    string
		weight      float64
	}{
		{[]string{"喜欢", "爱", "想", "想念"}, "user_profile", "TASTES", "用户喜好", "用户表达了关于「%s」的倾向", 0.5},
		{[]string{"生日", "出生"}, "user_profile", "BASIC_PROFILE", "用户生日", "用户提到了生日相关信息：%s", 0.7},
		{[]string{"工作", "上班", "公司"}, "user_profile", "BASIC_PROFILE", "用户职业", "用户提到了工作相关信息：%s", 0.5},
		{[]string{"家", "住", "搬家"}, "user_profile", "BASIC_PROFILE", "用户住址", "用户提到了居住相关信息：%s", 0.5},
	}
	for _, r := range rules {
		for _, kw := range r.keywords {
			if strings.Contains(msg, kw) {
				summary := r.template
				if strings.Count(summary, "%s") > 0 {
					summary = strings.Replace(summary, "%s", truncStr(msg, 80), 1)
				}
				drafts = append(drafts, lightFactDraft{
					Domain: r.domain, Subcategory: r.subcategory, Subject: r.subject,
					Summary: summary, Weight: r.weight, Confidence: 0.6,
				})
				break
			}
		}
	}
	return drafts
}
