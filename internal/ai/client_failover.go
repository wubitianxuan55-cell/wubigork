// client_failover.go — 引擎故障转移域（2026-10-04 自 client.go 纯搬移：
// FailoverEnabled/failoverTarget/transferable 判据；配方照批 25 websearch.go）。

package ai

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// FailoverEnabled 读取故障转移开关当前生效值（未注入读取函数 = 恒关）。
func (c *Client) FailoverEnabled() bool {
	c.mu.RLock()
	fn := c.engineFailoverFn
	c.mu.RUnlock()
	return fn != nil && fn()
}

// ── 引擎故障转移 v0（C 刀）────────────────────────────────────
//
// 聊天请求链（流式/非流式）首请求失败且满足可转移条件时，换候选引擎用其
// default_model 重试一次（新 URL/新 Key/新模型名）；仍失败返回原错误，
// 不递归、最多一次转移（重试请求带 failoverDone 标记）。开关默认关——
// 关闭时成功与失败路径行为均与现状逐字节一致。

// failoverTarget 判断是否应故障转移并返回（候选引擎ID, 候选default_model, true）。
// 条件：开关开启 + engineMgr 可用 + 错误可转移 + 存在候选引擎（enabled 且
// 最近 Status.Connected 且判型 llm，排除失败引擎，按 m.order 顺序取首个）。
func (c *Client) failoverTarget(failedEngineID string, err error) (string, string, bool) {
	if err == nil || !c.FailoverEnabled() || c.engineMgr == nil {
		return "", "", false
	}
	if !isTransferableChatError(err) {
		return "", "", false
	}
	for _, cand := range c.engineMgr.FailoverCandidates(failedEngineID) {
		return cand.ID, cand.DefaultModel, true
	}
	return "", "", false
}

// transferableStatusRe 提取聊天错误中的上游状态码（Chat/doStreamRequest 的
// 非统一错误格式 "API 错误 (HTTP %d): ..."）。
var transferableStatusRe = regexp.MustCompile(`API 错误 \(HTTP (\d{3})\)`)

// isTransferableChatError 判断聊天请求错误是否可转移（C 刀 v0）：
//   - 网络类（连接拒绝/DNS 解析失败/超时/tls/连接重置/代理不可达等）；
//   - HTTP 408（请求超时）/ 429（限流）/ 5xx（服务端故障）。
//
// 401/403/400/404 等配置类错误不转移——换引擎解决不了 Key/参数配错。
func isTransferableChatError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if m := transferableStatusRe.FindStringSubmatch(err.Error()); m != nil {
		code, cErr := strconv.Atoi(m[1])
		if cErr == nil {
			return code == 408 || code == 429 || code >= 500
		}
	}
	msg := strings.ToLower(err.Error())
	keywords := []string{
		"connection refused", "no such host", "dial tcp", "i/o timeout",
		"timeout", "deadline exceeded", "tls:", "connection reset",
		"broken pipe", "network is unreachable", "proxyconnect",
	}
	for _, kw := range keywords {
		if strings.Contains(msg, kw) {
			return true
		}
	}
	return false
}
