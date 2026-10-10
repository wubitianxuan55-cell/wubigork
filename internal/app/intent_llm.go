package app

// intent_llm.go — 意图内核 LLM 兜底分类器（v4.8）。
//
// 规则引擎（intent.Parse）未命中时的冷路径兜底：一次轻量 LLM 分类调用，
// 输出经 intent.ParseFallback 受控校验（白名单 navigate/status/read_screen +
// 0.75 置信门），命中后复用 routeIntentMode 既有执行层——不新开执行路径。
//
// 纪律：
//   - 默认关（intents_llm_fallback=false）：本开关把「宁可漏判」姿态交给模型，
//     且给语音回路（同步 whisper 回调）串行加延迟。
//   - 硬超时（intents_llm_timeout_ms，默认 2000）：超时/错误/解析失败一律
//     立即返回 nil → 原聊天管道，不重试。
//   - dryRun 恒不调用：命令面板逐键搜索绝不打 LLM（预览-确认制口径一致——
//     预览不到的动作也无法从面板执行）。

import (
	"context"
	"log/slog"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/intent"
)

// classifyIntentFallback 规则未命中时的兜底入口；返回 nil = 走原聊天管道。
func (a *App) classifyIntentFallback(text string) *intent.Intent {
	if a == nil || a.core == nil || a.cfg == nil || a.clientRef() == nil {
		return nil
	}
	if !a.cfg.GetIntentsLLMFallback() {
		return nil
	}
	if a.intentClassifierFn != nil {
		return a.intentClassifierFn(text)
	}
	return a.classifyIntentWithLLM(text)
}

// intentFallbackSystemPrompt 组装 LLM 兜底分类器的系统提示词。boards 是当前
// manifest 的板块 id 白名单串（boardIDList 单一来源，AP8-08：此前这里手写第二份
// 清单，含已删的 code、缺 schedule/sin/knowledge）。
//
// boards 为空（清单拼装不出）时**不打白名单约束**：提示词只给动作语义，不 panic、
// 不编造清单——navigate 的 target 仍由 intent.ParseFallback（动作白名单+置信门）
// 与 boardLabel（manifest 校验）两道闸兜底，宁漏勿误。
func intentFallbackSystemPrompt(boards string) string {
	navLine := "- navigate：想打开/切换某个板块。target=板块id"
	if boards != "" {
		navLine += "，只能是：" + boards + " 之一"
	}
	navLine += "\n"
	return "你是桌面助手的指令分类器。把用户的一句话分类为以下动作之一，只输出 JSON，" +
		"格式：{\"action\":\"navigate|status|read_screen|none\",\"target\":\"\",\"confidence\":0.0到1.0}\n" +
		navLine +
		"- status：询问当前用的什么模型/引擎。target=\"model\"\n" +
		"- read_screen：想读取/查看屏幕上显示的内容（可含第几块屏幕）。target=\"screen\"或\"screen:2\"或\"screen:primary\"\n" +
		"- none：闲聊寒暄、对已完成事物的评价（如\"画得不错\"）、询问知识、与上述无关的一切请求\n" +
		"示例：{\"action\":\"navigate\",\"target\":\"imagegen\",\"confidence\":0.9}\n" +
		"拿不准、语义含混或 confidence<0.75 一律 action=none。宁可 none 不可猜。"
}

// classifyIntentWithLLM 内置 LLM 分类：routine 目标解析（常规办公绑定 → 默认
// 本地 herdsman）+ 2s 级硬超时 + 受控 JSON 校验。
func (a *App) classifyIntentWithLLM(text string) *intent.Intent {
	engine, model, err := a.resolveRoutineTarget("", "")
	if err != nil {
		slog.Debug("[intent] 兜底分类无可用模型", "err", err)
		return nil
	}
	// 全局离线模式：分类调用只允许本地引擎（Type.IsLocal 单一来源，覆盖
	// strata/modelhub 等后来者，替代旧硬编码 herdsman/ollama 名单），云端
	// 目标直接放弃（走聊天）。
	if a.cfg.GetOfflineMode() {
		if e, ok := a.engineMgr.GetEngine(engine); !ok || !e.Type.IsLocal() {
			return nil
		}
	}

	sysPrompt := intentFallbackSystemPrompt(a.boardIDList())

	ms := a.cfg.GetIntentsLLMTimeoutMS()
	if ms <= 0 {
		ms = 2000
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(ms)*time.Millisecond)
	defer cancel()

	reply, err := a.clientRef().ChatSimpleStreamWithOptions(ctx, model,
		sysPrompt, "用户输入："+text, ai.ChatSimpleOptions{
			EngineID:       engine,
			Feature:        "routine",
			Temperature:    0.1,
			MaxTokens:      64,
			TimeoutMinutes: 1,
		})
	if err != nil {
		slog.Debug("[intent] 兜底分类调用失败", "err", err)
		return nil
	}
	it := intent.ParseFallback(reply)
	if it == nil {
		slog.Debug("[intent] 兜底分类未通过校验（低置信/白名单外/坏输出）")
		return nil
	}
	// navigate 的 target 必须在当前 manifest 中（与规则引擎同口径校验）。
	if it.Action == intent.ActionNavigate && a.boardLabel(it.Target) == "" {
		return nil
	}
	if it.Action == intent.ActionStatus {
		it.Target = "model"
	}
	slog.Info("[intent] LLM 兜底命中", "action", it.Action, "target", it.Target)
	return it
}
