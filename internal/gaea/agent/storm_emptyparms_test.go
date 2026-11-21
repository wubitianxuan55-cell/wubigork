package agent

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// 2026-11-21 反馈 P1-8 回归钉：空参签名（参数在传输面丢失——收到 0 键）不计入
// storm：这种失败的正确处置恰是原样重试同一调用，guard 的「换策略」建议会把
// 工具侧丢参记成模型行为问题（实测连试 3 次即被拦，回合被白耗）。
func TestBatchStormSignatureSkipsEmptyParamsTransportLoss(t *testing.T) {
	calls := []provider.ToolCall{{Name: "bash"}}
	if _, ok := batchStormSignature(calls, []toolOutcome{
		{errMsg: `command is required：本次收到 0 个参数键（）——command 缺失或值为空，把命令字符串放进 command 字段重试。原始参数回显（2 字节）："{}"`},
	}); ok {
		t.Fatal("空参传输丢失不计入 storm——正确处置是原样重试")
	}
	// 对照：收到参数但缺 command 的普通校验错误仍正常计数（真可能是模型反复拼错）。
	if _, ok := batchStormSignature(calls, []toolOutcome{
		{errMsg: `command is required：本次收到 1 个参数键（cmd）——command 缺失或值为空`},
	}); !ok {
		t.Fatal("非空参的重复校验错误应正常计数")
	}
}
