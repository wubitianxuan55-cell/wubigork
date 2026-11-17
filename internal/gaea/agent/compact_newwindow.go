package agent

import (
	"context"
	"fmt"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// token-budget 压缩形态（codex 蒸馏 T9，compact_token_budget 同构）：跳过
// 模型/服务端摘要，直接装一个新窗口——历史收敛为「固定前缀 + 白名单保留项 +
// 回合上下文」，没有摘要占位消息，但走完整压缩生命周期（起止事件 + Pre/
// PostCompact hook）。
//
// 保留语义（上游 start_new_context_window 的 gaea 折算）：
//   - 固定前缀（system 等钉住段）原样保留——人格与项目约定仍在；
//   - 白名单保留项 = partitionFold 的 kept（小用户轮 + [[keep]] 标记 +
//     保留型工具输出），对应上游 retained_client_items 与 input_goal_ids 继承
//     （用户目标不因换窗口而丢失）；
//   - 回合上下文 = msgs[start:] 原样保留。
//
// 纪律：不把整段历史原样塞回去（那等于没压缩）——超预算的保留项由既有
// keepUserTurns 预算机制截断（保留最近的）。

// compactNewWindowTrigger 是 token-budget 形态的触发标签（事件 Trigger 字段）。
const compactNewWindowTrigger = "new_window"

// postCompactHook 是可选扩展接口：实现它的 ToolHooks 在压缩完成后收到通知。
// 不并进 ToolHooks 主接口——零破坏（既有测试桩不须跟着改）。
type postCompactHook interface {
	PostCompact(ctx context.Context, trigger string)
}

// NewWindowCompact 走一次「无摘要换窗口」压缩（开关 CompactionConfig.NewWindow
// 默认关；关时本方法直接 no-op 返回，历史不动）。返回是否实际压缩。
func (a *AgentRunner) NewWindowCompact(ctx context.Context) (bool, error) {
	if !a.compaction.NewWindow {
		return false, nil
	}
	msgs := a.session.Messages
	head, start, ok := a.planCompaction(msgs, minCompactMessages)
	if !ok {
		head, start, ok = a.planCompaction(msgs, 1)
	}
	if !ok {
		return false, nil
	}
	kept, fold, retention := a.partitionFold(msgs[head:start])
	if len(fold) == 0 {
		return false, nil
	}
	// 经济学预检（与 compact() 同款）：节省太小的折叠不值得动历史。
	if !foldEconomics(fold) {
		return false, nil
	}
	// 经济学硬闸（新窗口版）：不做摘要意味着 fold 被直接丢弃——若丢弃后估算
	// 面没有变小（保留项+尾部已吃满当前估算），这次换窗口就是纯 no-op，拒。
	// 不按 trigger 判：tail 预算有 2000 token 下限（V10.11），小窗口下保留段
	// 单独就会顶过 trigger——照 trigger 判会关死小窗口（与 compact() 同款教训）。
	if estNow := a.EstimateContextTokens(); a.compaction.Window > 0 {
		estAfter := int(float64(charsOfMessages(msgs[:head])+charsOfMessages(kept)+charsOfMessages(msgs[start:])) * a.tokPerChar())
		if estAfter >= estNow {
			return false, nil
		}
	}

	// 生命周期：Pre（观测；guidance 在无摘要形态无处安放，仅保事件面）→
	// 换窗口 → Post。
	if a.hooks != nil {
		_ = a.hooks.PreCompact(ctx, compactNewWindowTrigger)
	}
	a.sink.Emit(event.Event{Kind: event.CompactionStarted,
		Compaction: event.Compaction{Trigger: compactNewWindowTrigger}})

	compacted := make([]provider.Message, 0, head+len(kept)+len(msgs)-start)
	compacted = append(compacted, msgs[:head]...)
	compacted = append(compacted, kept...)
	compacted = append(compacted, msgs[start:]...)
	a.compaction.CompactCount++
	window := a.compaction.CompactCount
	a.session.Replace(compacted)
	a.session.IncrementRewrite()

	a.sink.Emit(event.Event{Kind: event.CompactionDone,
		Compaction: event.Compaction{
			Trigger: compactNewWindowTrigger, Messages: len(fold),
			Summary: "", // 空摘要标记：无摘要的换窗口压缩，UI 据此与摘要失败区分
			Window:  window,
		}})
	if retention.Dropped > 0 {
		a.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelWarn,
			Text: fmt.Sprintf("%d of your earlier message(s) were dropped by the new-window compaction (no summary was generated). Prefix a turn with [[keep]] to hold it verbatim.", retention.Dropped)})
	}
	if pc, ok := a.hooks.(postCompactHook); ok {
		pc.PostCompact(ctx, compactNewWindowTrigger)
	}
	return true, nil
}

// MaybeDownshiftCompact 是切模型入口的挂点（codex 蒸馏 ModelDownshift 触发）：
// 新模型窗口 < 当前有效窗口且当前用量相对新窗口已超阈值时，先压缩再切换。
// 只做判断与触发，不做模型切换本身（调用方在压缩成功后再换模型——压缩发生
// 在发起请求之前，事件顺序可断言）。开关关时恒 no-op。
func (a *AgentRunner) MaybeDownshiftCompact(ctx context.Context, newWindow int) (bool, error) {
	if !a.compaction.NewWindow || newWindow <= 0 {
		return false, nil
	}
	cur := a.compaction.Window
	if cur > 0 && newWindow >= cur {
		return false, nil // 窗口没变小，无需降档
	}
	usage := a.EstimateContextTokens()
	triggerTokens := int(float64(newWindow) * a.ratio())
	if usage < triggerTokens {
		return false, nil // 新窗口装得下，不压
	}
	return a.NewWindowCompact(ctx)
}
