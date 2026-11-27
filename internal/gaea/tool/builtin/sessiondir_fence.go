package builtin

// 会话件围栏（2026-11-16 实弹）：结构化读面（read_file/grep）对
// .gaea/sessions/ 子树 deny。会话历史是内部事件流——.jsonl 逐行巨大
// （211KB/100 行实锤），生啃必撞 D1 输出预算截断，且混系统事件非可读
// 会话原文。正规通道=@会话 引用注入 / session_search 结构化检索（两板块
// 均装配）。
//
// bash 不围（诊断需求保留出口，head/tail 分页；同 credential_fence 诚实
// 边界——归预算/审批域）；ls 不围（文件名非内容，撞不了预算）。误拦代价
// =模型收到结构化指引转正规通道；漏拦代价=浪费回合+上下文污染（D1 兜底
// 无数据损失）。升机制依据（2026-11-16 用户拍板）：纪律引导不是强制——
// 纪律在 workbench 技能内，子代理/长会话中后段读不到，到达率不足。

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/gaea/tool"
)

// sessionDirFenceRule 报告路径是否命中 .gaea/sessions/ 子树：段序列匹配
// （.gaea 段后紧跟 sessions 段），大小写不敏感，斜杠双形态归一。
func SessionDirFenceRule(p string) bool {
	lower := strings.ToLower(filepath.ToSlash(p))
	segs := strings.Split(strings.TrimPrefix(lower, "/"), "/")
	for i := 0; i+1 < len(segs); i++ {
		if segs[i] == ".gaea" && segs[i+1] == "sessions" {
			return true
		}
	}
	return false
}

// sessionDirDenied 结构化拒绝（错误即值+自愈指引）。围栏先于读取生效，
// effect none；原样重试恒被拒故不可重试。
func SessionDirDenied(toolName, p string) error {
	return &tool.Failure{
		Code: tool.CodeDenied,
		Err: fmt.Errorf("SESSION_DENIED: 会话件围栏拒绝 %s 读取 %s。"+
			".gaea/sessions/ 下是内部事件流（逐行巨大必撞输出预算截断，且混系统事件非可读会话原文）。"+
			"读历史会话请走正规通道：① session_search 不带 query——列出最近的会话（标题 / 起止 / 首条任务 / 怎么收的尾，含刚结束的那场），接续某场的工作再 session_search handoff=<会话 id> 看它的交接卡（收尾回复、动过的文件及体检）；带关键词则检索用户与助手的消息；"+
			"② @会话:<标题或文件名> 引用——注入该场会话的快照；诊断确需原始件时用 bash head/tail 分页取窗口。", toolName, p),
		EffectState: tool.EffectNone,
		NextAction:  "先 session_search 不带 query 列最近会话（接续某场的工作用 handoff=<会话 id> 看交接卡），再带关键词检索或 @会话:<标题> 引用；诊断原始件用 bash head/tail 分页",
	}
}

// SessionDirDeniedWrite 写侧对称拒绝（2026-11-17 审查批2 刀2，读侧
// SessionDirDenied 同源）：.gaea/sessions/ 是引擎内部事件流（只读资产），
// agent 不得借写族自由 path 参数落盘/改写——写改即污染会话历史。
func SessionDirDeniedWrite(toolName, p string) error {
	return &tool.Failure{
		Code: tool.CodeDenied,
		Err: fmt.Errorf("SESSION_DENIED: 会话件围栏拒绝 %s 写入 %s。"+
			".gaea/sessions/ 下是引擎内部事件流（写改即污染会话历史）。"+
			"事实沉淀用 remember；历史内容引用走 @会话/session_search；正文与笔记落工作空间常规目录。", toolName, p),
		EffectState: tool.EffectNone,
		NextAction:  "事实沉淀用 remember；历史引用走 @会话/session_search",
	}
}
