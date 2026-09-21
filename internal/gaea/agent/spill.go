package agent

import (
	"fmt"
)

// ── spill 泄洪策略（v4.379，dsh packages/spill/spill-policy 蒸馏·取道不取器）
//
// 上游语义：工具最终结果超过 maxInlineBytes 时，全文先存会话级 spill 存储，
// 模型只见有界预览+locator+取回指引；read 豁免防「read→spill→read again」
// 循环；泄洪失败绝不把成功调用变错误或藏掉内联结果（best-effort）。
//
// gaea 落点：executeOne 成功路径在 SmartCompress/truncateToolOutput **之前**
// 捕获原文——上游的泄洪对象是「最终文本」，gaea 的按工具压缩（bash 24KB 等）
// 与全局 48KB 截断会先销毁中段，泄洪必须抢在压缩前，预览仍交给既有压缩
// 管线（比上游的裸 head/tail 预览更聪明）。存储是 runner 侧 spill.Store
//（内存+总预算 FIFO 驱逐，重启清零即天然会话级——磁盘版唯一收益是重启
// 幸存，不值得无主痕迹文件的卫生面）。

// spillMinBytes 是触发泄洪的结果字节下限：低于此值的结果基本能完整内联
// （bash/通用截断帽 24KB 档），不值得花一次存储。
const spillMinBytes = 24 * 1024

// spillNotice 是前置在工具结果头部的 locator 指引行。放头部是截断生存性的
// 结论而非偏好：信封是单行紧凑 JSON，超 48KB 时 selectHygieneLines 全行保留
// 但首行被 snapToRuneBoundary 头部截断——尾缀指引会被切掉；head+tail 多行
// 路径下头部行同样必保。两种截断形态下前缀 locator 都必然存活。
func spillNotice(id string, totalBytes int) string {
	return fmt.Sprintf("[spilled] full output (%d bytes) saved as spill %s — the inline copy below may be elided. "+
		"Call read_spill(id=%q) for the start, or read_spill(id=%q, offset=N) to page through. Spills last for this session only.",
		totalBytes, id, id, id)
}

// spillCandidate 报告一个工具的成功结果是否参与泄洪。豁免=read_file（自带
// offset/limit 分页+缓存，且是上游 read 豁免的同义——防取回循环）、
// read_spill（取回工具自身）、task（子代理最终回答必须原样抵达父级）。
func spillCandidate(name string) bool {
	switch name {
	case "read_file", "read_spill", "task":
		return false
	}
	return true
}
