package app

// GaeaSubagentStop — 运行中子代理的独立 Stop（D6 只读态四短路复核收窄项，
// dsh 蒸馏计划 01 §4·D6）：父离线/独立面板场景「禁用输入 + 独立 Stop」的
// 后半件。后端 InterruptSubagent（排队与运行期都可取消，fire-and-return）
// 早已存在并有工具面覆盖，此前只缺 bridge 暴露与面板按钮。

import (
	"errors"

	gaeaAgent "github.com/gaea/gaea/internal/gaea/agent"
)

// GaeaSubagentStop 中断一个运行中/排队中的子代理。信号发出即返回，目标在
// 下一个取消点停；终态（aborted 落账）由 runner 的收尾写完成，前端 3s 轮询
// 自校正——与 TaskCancel 的交互契约同款，不引入第二条状态通道。
func (a *App) GaeaSubagentStop(sessionPath, ref string) error {
	_ = sessionPath // 中断面按 ref 全局寻址（subRuns），路径仅作参数对称
	if !gaeaAgent.ValidRunRef(ref) {
		return errors.New("invalid subagent reference")
	}
	return gaeaAgent.InterruptSubagent(ref)
}
