package contextview

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// NodeDetailFor 从会话日志条目中按 seq 提取节点「完整调用」详情（纯函数，
// v4.80 懒加载的读端）。支持四类可展开节点：
//   - tool_result：完整输出 + 从最近一条同 id 的 tool_dispatch 回读参数
//     （与 subagentRender/trajectory 的 callId 配对同口径）；
//   - user_message / assistant_message：完整正文。
//   - header：request_header 的真实系统提示词与工具 schema。
//
// 找不到 seq 或 kind 不可展开时返回 ok=false。正文超 MaxDetailBytes 时截断
// 并标 Clamped（诚实标注，绝不静默丢弃后半段语义——由前端展示截断提示）。
func NodeDetailFor(entries []session.LogEntry, seq int64) (NodeDetail, bool) {
	idx := -1
	for i, e := range entries {
		if e.Seq == seq {
			idx = i
			break
		}
	}
	if idx < 0 {
		return NodeDetail{}, false
	}
	e := entries[idx]
	switch e.Kind {
	case session.KindToolResult:
		return toolResultDetail(entries, idx, e)
	case session.KindRequestHeader:
		return requestHeaderDetail(e)
	case session.KindUserMessage:
		var p userPayload
		if err := json.Unmarshal(e.Payload, &p); err != nil {
			return NodeDetail{}, false
		}
		if strings.TrimSpace(p.Content) == "" {
			return NodeDetail{}, false
		}
		text, clamped := clampDetail(p.Content)
		return NodeDetail{
			Seq: e.Seq, Kind: "user_message", Ts: e.Ts,
			Text: text, Lines: countLines(text), Clamped: clamped,
			ImageRefs:      ExtractImageRefs(text),
			AttachmentRefs: ExtractStoredFileRefs(text),
		}, true
	case session.KindAssistantMessage:
		var p assistantPayload
		if err := json.Unmarshal(e.Payload, &p); err != nil {
			return NodeDetail{}, false
		}
		if strings.TrimSpace(p.Text) == "" {
			return NodeDetail{}, false
		}
		text, clamped := clampDetail(p.Text)
		return NodeDetail{
			Seq: e.Seq, Kind: "assistant_message", Ts: e.Ts,
			Text: text, Lines: countLines(text), Clamped: clamped,
			ImageRefs: ExtractImageRefs(text),
		}, true
	}
	return NodeDetail{}, false
}

var storedFileRefRe = regexp.MustCompile(`(?i)\.gaea[\\/](?:uploads|attachments)[\\/][^\s<>"')]+`)

// ExtractStoredFileRefs 只提取 Gaea 持久化附件目录中的非图片文件，去重保序，
// 上限 12。普通工作区路径即使出现在正文也不冒充用户附件。
func ExtractStoredFileRefs(text string) []string {
	seen := map[string]bool{}
	var out []string
	for _, match := range storedFileRefRe.FindAllString(text, -1) {
		ref := strings.TrimRight(match, ".,;:!?]}")
		if ref == "" || seen[strings.ToLower(ref)] || detailImageExt(filepath.Ext(ref)) {
			continue
		}
		seen[strings.ToLower(ref)] = true
		out = append(out, ref)
		if len(out) >= 12 {
			break
		}
	}
	return out
}

func detailImageExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".tif", ".tiff":
		return true
	}
	return false
}

// requestHeaderDetail 解出模型请求实际携带的 system prompt 与工具 schema。
// system/tools 两个浏览器组节点共用 header Seq；详情同时返回两者，前端按
// 当前展开分类展示，避免为同一日志行造两个不可区分的读取接口。
func requestHeaderDetail(e session.LogEntry) (NodeDetail, bool) {
	var p requestHeaderPayload
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return NodeDetail{}, false
	}
	text, clamped := clampDetail(p.System)
	d := NodeDetail{
		Seq: e.Seq, Kind: "header", Ts: e.Ts,
		Text: text, Lines: countLines(text), Clamped: clamped,
		ImageRefs: ExtractImageRefs(text),
	}
	if len(p.Tools) == 0 && strings.TrimSpace(text) == "" {
		return NodeDetail{}, false
	}
	remaining := MaxDetailBytes
	for _, tool := range p.Tools {
		if strings.TrimSpace(tool.Name) == "" {
			continue
		}
		schema := tool.Schema
		if len(schema) > remaining {
			cut := remaining
			for cut > 0 && cut < len(schema) && !utf8RuneStart(schema[cut]) {
				cut--
			}
			schema = schema[:cut]
			d.Clamped = true
		}
		remaining -= len(schema)
		d.Tools = append(d.Tools, NodeToolDetail{
			Name: tool.Name, Schema: schema,
			Tokens: estimateTokens(tool.Name + schema),
			Source: tool.Source,
		})
		if remaining <= 0 {
			break
		}
	}
	return d, true
}

// toolResultDetail 提取工具结果全文，并向最近一条同 id 的 tool_dispatch
// 回读参数（配对失败参数留空——诚实缺失，不猜）。dispatch 落盘有
// "tool_dispatch"（live）与 tool_call（legacy 迁移）两种 kind，同样回读。
func toolResultDetail(entries []session.LogEntry, idx int, e session.LogEntry) (NodeDetail, bool) {
	var p toolResultPayload
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return NodeDetail{}, false
	}
	d := NodeDetail{
		Seq: e.Seq, Kind: "tool_result", Ts: e.Ts,
		Tool: p.Name, Output: p.Output, Err: p.Err, Truncated: p.Truncated,
	}
	if p.ID != "" {
		for i := idx - 1; i >= 0; i-- {
			if entries[i].Kind != "tool_dispatch" && entries[i].Kind != session.KindToolCall {
				continue
			}
			var dp toolDispatchPayload
			if err := json.Unmarshal(entries[i].Payload, &dp); err != nil || dp.ID != p.ID {
				continue
			}
			d.Args = dp.Args
			if d.Tool == "" {
				d.Tool = dp.Name
			}
			break
		}
	}
	body := d.Output
	if body == "" && d.Err != "" {
		body = d.Err // 错误文本走 Output 槽展示（前端 Raw 面板单字段），Err 保留语义位
	}
	body, clamped := clampDetail(body)
	d.Output = body
	d.Lines = countLines(body)
	d.Clamped = clamped
	// 2.5b 后半：图片引用 = 参数 JSON 字符串值（识图 image_path 等）+ 输出
	// 文本自由匹配（生成类工具的产物路径）。输出正文与参数都可能引用同一
	// 张图，mergeImageRefs 去重保序。
	d.ImageRefs = mergeImageRefs(ExtractImageRefsFromArgs(d.Args), ExtractImageRefs(body))
	return d, true
}

// clampDetail 把详情正文限制在 MaxDetailBytes 内（字节口径，UTF-8 安全截断
// 到最近 rune 边界）。
func clampDetail(s string) (string, bool) {
	if len(s) <= MaxDetailBytes {
		return s, false
	}
	cut := MaxDetailBytes
	for cut > 0 && !utf8RuneStart(s[cut]) {
		cut--
	}
	return s[:cut], true
}

func utf8RuneStart(b byte) bool { return b&0xC0 != 0x80 }

// countLines 返回正文行数（空串为 0；末尾换行不另计一行）。
func countLines(s string) int {
	if s == "" {
		return 0
	}
	n := int64(1)
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			n++
		}
	}
	return int(n)
}
