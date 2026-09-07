package control

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// diff 确认闭环刀C（v4.148）：schedule_apply project 通道（整计划替换/生成）
// hardAsk 化——任何权限级别逐条确认、禁会话放行（拍板项 2）；ops 增量通道
// 维持现状闸门（拍板项 3：auto/yolo 豁免弹卡，后置回执+回滚兜底）。
// project 通道可读 subject 说明「要写进来的计划」（after 侧，同一 schedule 引擎）。

const (
	projArgs = `{"project":{"name":"测试工程","tasks":[{"id":"A","name":"挖土","duration":2,"level":1,"progress":0},{"id":"B","name":"垫层","duration":3,"level":1,"progress":0}],"links":[{"from":"A","to":"B","type":"FS","lag":0}]}}`
	opsArgs  = `{"ops":[{"type":"patch_task","id":"A","patch":{"duration":3}}]}`
)

// TestScheduleApplyProjectPromptsAtAsk project 通道在 ask 级（默认）弹卡，
// allow_once 放行且不记忆（remember=false）。
func TestScheduleApplyProjectPromptsAtAsk(t *testing.T) {
	c, ids, _ := approvalIDs()
	go func() { c.Approve(<-ids, DecisionAllowOnce) }()

	allow, remember, err := gateApprover{c}.Approve(context.Background(), "schedule_apply", "进度计划/当前计划.gsched.json", json.RawMessage(projArgs))
	if err != nil || !allow || remember {
		t.Fatalf("Approve = (%v,%v,%v), want allow once", allow, remember, err)
	}
}

// TestScheduleApplyProjectHardAtAutoAndYolo project 通道在 auto/yolo 级仍强制
// 弹卡（毁伤半径最大的操作不因权限级别而静默）。
func TestScheduleApplyProjectHardAtAutoAndYolo(t *testing.T) {
	for _, level := range []string{"auto", "yolo"} {
		c, ids, prompts := approvalIDs()
		c.SetPermLevel(level)
		go func() { c.Approve(<-ids, DecisionAllowOnce) }()
		allow, _, err := gateApprover{c}.Approve(context.Background(), "schedule_apply", "当前计划", json.RawMessage(projArgs))
		if err != nil || !allow {
			t.Fatalf("level %s: Approve = (%v,%v), want allow via prompt", level, allow, err)
		}
		if *prompts != 1 {
			t.Fatalf("level %s: prompted %d times, want 1", level, *prompts)
		}
	}
}

// TestScheduleApplyProjectNoSessionGrant project 通道禁会话记忆：同一调用
// 两次都逐条弹卡（alwaysPrompt 不读不写 granted），即使每次都答 allow_session。
func TestScheduleApplyProjectNoSessionGrant(t *testing.T) {
	c, ids, prompts := approvalIDs()
	go func() {
		for id := range ids {
			c.Approve(id, DecisionAllowSession)
		}
	}()
	for i := 0; i < 2; i++ {
		allow, _, err := gateApprover{c}.Approve(context.Background(), "schedule_apply", "当前计划", json.RawMessage(projArgs))
		if err != nil || !allow {
			t.Fatalf("call %d = (%v,%v), want allow", i, allow, err)
		}
	}
	if *prompts != 2 {
		t.Errorf("prompted %d times, want 2（project 通道禁会话记忆）", *prompts)
	}
}

// TestScheduleApplyOpsChannelKeepsLegacyGate ops 通道维持现状：ask 级弹卡、
// auto 级静默放行不弹卡（审批疲劳对策，拍板项 3）。
func TestScheduleApplyOpsChannelKeepsLegacyGate(t *testing.T) {
	c, ids, _ := approvalIDs()
	go func() { c.Approve(<-ids, DecisionAllowOnce) }()
	if allow, _, err := (gateApprover{c}).Approve(context.Background(), "schedule_apply", "当前计划", json.RawMessage(opsArgs)); err != nil || !allow {
		t.Fatalf("ops ask: Approve = (%v,%v), want allow via prompt", allow, err)
	}

	c2, _, prompts2 := approvalIDs()
	c2.SetPermLevel("auto")
	allow, _, err := (gateApprover{c2}).Approve(context.Background(), "schedule_apply", "当前计划", json.RawMessage(opsArgs))
	if err != nil || !allow {
		t.Fatalf("ops auto: Approve = (%v,%v), want silent allow", allow, err)
	}
	if *prompts2 != 0 {
		t.Errorf("ops auto: prompted %d times, want 0（豁免弹卡）", *prompts2)
	}
}

// TestScheduleApplyProjectSubjectReadable project 通道可读 subject：说明目标
// 文件与写入后计划概貌（after 侧，同一 schedule 引擎口径）；ops 通道不走该
// subject；无效 project JSON 诚实降级标注。
func TestScheduleApplyProjectSubjectReadable(t *testing.T) {
	s := approvalSubjectFor("schedule_apply", json.RawMessage(projArgs))
	if !strings.Contains(s, "整计划替换") || !strings.Contains(s, "总工期 5 天") || !strings.Contains(s, "2 项工作") {
		t.Fatalf("project subject 不可读：%s", s)
	}
	if s2 := approvalSubjectFor("schedule_apply", json.RawMessage(opsArgs)); strings.Contains(s2, "整计划替换") {
		t.Fatalf("ops 通道不该走 project subject：%s", s2)
	}
	if s3 := approvalSubjectFor("schedule_apply", json.RawMessage(`{"project":{"tasks":[{"id":"A","duration":1},{"id":"B","duration":1}],"links":[{"from":"A","to":"B"},{"from":"B","to":"A"}]}}`)); !strings.Contains(s3, "CPM") {
		t.Fatalf("循环依赖应在 subject 诚实标注：%s", s3)
	}
	if s4 := approvalSubjectFor("schedule_apply", json.RawMessage(`{"project":"不是对象"}`)); !strings.Contains(s4, "整计划替换") {
		t.Fatalf("解析失败应降级仍可读：%s", s4)
	}
}
