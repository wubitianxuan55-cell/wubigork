package session

import (
	"encoding/json"
	"os"

	"github.com/gaea/gaea/internal/gaea/event"
)

// 注意力事务的「悬空卡闭合」（2026-11-02 第二刀）。
//
// 场景：进程在卡还开着的时候被杀（崩溃、强制退出、关机）。内存里的
// approvals/asks 随进程消失，日志却永远停在那一行 approval_request 上——
// 统计器只能把它读成「用户一直没回答」，于是打断计数偏高、解决率偏低，
// 而且「放弃」与「未解决」两种完全不同的结局在账本里分不开。
//
// 恢复会话时补写一条 outcome=abandoned 的解决事件，让每个请求都有结局：
// 事务在真相层闭合，重放可终结，「放弃」不再是沉默。

// 注意力事务的四个日志 kind（与 KindString 单点映射保持同字面量）。
const (
	KindApprovalRequest  = "approval_request"
	KindApprovalResolved = "approval_resolved"
	KindAskRequest       = "ask_request"
	KindAskResolved      = "ask_resolved"
	// KindApprovalAuto 是自动放行账本（没弹卡、由机器决定），不属于注意力
	// 事务：它没有 id、没有等待，只是「谁替人批准了什么」的审计记录。
	KindApprovalAuto = "approval_auto"
)

// OrphanAttention 是一张在日志里有请求、没有结果的注意力卡。
type OrphanAttention struct {
	Kind    string // KindApprovalRequest / KindAskRequest
	ID      string
	Tool    string
	Subject string
	Source  string
}

// UnresolvedAttention 返回日志里未闭合的注意力卡（按请求出现顺序）。
// 同一 id 的重复请求只算一张（日志修复/重放可能留下重复行）；结果事件
// 与其 kind 对应，审批的结果不会关闭一张提问卡。
func UnresolvedAttention(entries []LogEntry) []OrphanAttention {
	type key struct{ kind, id string }
	open := make(map[key]bool)
	seen := make(map[key]bool)
	info := make(map[key]OrphanAttention)
	var order []key

	for _, e := range entries {
		switch e.Kind {
		case KindApprovalRequest, KindAskRequest:
			var p struct {
				ID      string `json:"id"`
				Tool    string `json:"tool"`
				Subject string `json:"subject"`
				Source  string `json:"source"`
			}
			if json.Unmarshal(e.Payload, &p) != nil || p.ID == "" {
				continue
			}
			k := key{e.Kind, p.ID}
			if seen[k] {
				continue
			}
			seen[k] = true
			open[k] = true
			order = append(order, k)
			info[k] = OrphanAttention{
				Kind: e.Kind, ID: p.ID, Tool: p.Tool, Subject: p.Subject, Source: p.Source,
			}
		case KindApprovalResolved, KindAskResolved:
			var p struct {
				ID string `json:"id"`
			}
			if json.Unmarshal(e.Payload, &p) != nil || p.ID == "" {
				continue
			}
			delete(open, key{KindApprovalRequest, p.ID})
			delete(open, key{KindAskRequest, p.ID})
		}
	}

	out := make([]OrphanAttention, 0, len(open))
	for _, k := range order {
		if open[k] {
			out = append(out, info[k])
		}
	}
	return out
}

// CloseOrphanAttention 为日志里遗留的悬空卡补写解决事件（outcome=abandoned，
// latency_ms=0 表示「不知道它挂了多久」，统计器不把 0 计入均等待）。
// 返回补写张数；日志不存在或没有悬空卡时返回 0。
//
// 幂等：补写后重跑不再有悬空卡。失败时返回错误且不写半张——调用方（会话
// 恢复链）应把它当尽力而为的账本收口，不因失败阻断恢复。
func CloseOrphanAttention(logPath, space string) (int, error) {
	if logPath == "" {
		return 0, nil
	}
	entries, err := ReadLogRepaired(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	orphans := UnresolvedAttention(entries)
	if len(orphans) == 0 {
		return 0, nil
	}
	w, err := OpenLog(logPath, "", space)
	if err != nil {
		return 0, err
	}
	defer func() { _ = w.Close() }()

	for _, o := range orphans {
		kind := KindApprovalResolved
		var payload any
		if o.Kind == KindAskRequest {
			kind = KindAskResolved
			payload = map[string]any{
				"id":         o.ID,
				"outcome":    event.OutcomeAbandoned,
				"answered":   0,
				"latency_ms": 0,
			}
		} else {
			payload = map[string]any{
				"id":         o.ID,
				"tool":       o.Tool,
				"subject":    o.Subject,
				"source":     o.Source,
				"decision":   "",
				"outcome":    event.OutcomeAbandoned,
				"latency_ms": 0,
			}
		}
		if _, err := w.Append(kind, payload); err != nil {
			return 0, err
		}
	}
	return len(orphans), nil
}
