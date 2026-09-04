package agent

// v4.63 并行子代理：批量分区语义回归。
//
// 背景：getConflictKey 曾把 task/run_skill 归为全局冲突键 "!spawn"——同一
// 回合批量派 N 路子代理会被逐个串行执行（实测 3 路各 ~4 分钟 → 墙钟
// ~12 分钟）。task/run_skill 的每次调用是独立运行（独立 Session、独立
// transcript、事件经 syncSink 串行落盘、用量经 usageMu 合并），无共享可变
// 状态，允许同批并行。install_skill 写技能目录，保持串行。

import (
	"fmt"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

var callSeq int

// call 生成带唯一 ID 的调用（真实 provider 调用 ID 恒唯一——分区器的
// spawn 键按 ID 区分，同 ID 视为冲突串行）。
func call(name string) provider.ToolCall {
	callSeq++
	return provider.ToolCall{ID: fmt.Sprintf("c-%s-%d", name, callSeq), Name: name, Arguments: "{}"}
}

// TestPartitionThreeTaskCallsRunParallel：同回合三路 task 派发必须落进同一个
// 并行批（回归：曾因 "!spawn" 全局冲突键被拆成三个串行批）。
func TestPartitionThreeTaskCallsRunParallel(t *testing.T) {
	calls := []provider.ToolCall{call("task"), call("task"), call("task")}
	batches := partitionToolCalls(nil, calls)
	if len(batches) != 1 || !batches[0].parallel || batches[0].end-batches[0].start != 3 {
		t.Fatalf("3 路 task 应为单一并行批，got %+v", batches)
	}
}

// TestPartitionRunSkillParallelTaskMixed：run_skill 与 task 混批同样并行。
func TestPartitionRunSkillParallelTaskMixed(t *testing.T) {
	// read_file 无路径 → 空冲突键 → 自成串行批（既有语义）；spawn 两路并行。
	calls := []provider.ToolCall{call("run_skill"), call("task"), call("read_file")}
	batches := partitionToolCalls(nil, calls)
	if len(batches) < 1 || !batches[0].parallel || batches[0].end-batches[0].start != 2 {
		t.Fatalf("run_skill+task 应同批并行，got %+v", batches)
	}
}

// TestPartitionInstallSkillStaysSerial：install_skill 写技能目录，保持全局串行。
func TestPartitionInstallSkillStaysSerial(t *testing.T) {
	calls := []provider.ToolCall{call("install_skill"), call("install_skill")}
	batches := partitionToolCalls(nil, calls)
	for _, b := range batches {
		if b.parallel {
			t.Fatalf("install_skill 不应并行：%+v", batches)
		}
	}
}
