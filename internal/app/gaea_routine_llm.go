package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/modelengine"
)

// routineLLMTool 通用文本兜底工具：专业能力（识图/OCR/检索/转换/图表）都有
// 专属本地工具后，剩下的纯文本活（摘要、归一化、抽取、改写）需要一个通用
// LLM 兜底，走模型中心「常规办公」绑定（默认本地 Herdsman）。是否调用完全
// 由云端 agent 决定——只提供入口，不做强制路由；云端 agent 应优先按专长选
// vision/ocr/semantic_search/format_convert 等专业工具，routine_llm 只是兜底。
type routineLLMTool struct {
	a *App
}

func (t routineLLMTool) Name() string { return "routine_llm" }

func (t routineLLMTool) Description() string {
	return "执行一次性通用文本推理：摘要、归一化、JSON 抽取、格式整理、改写等。目标模型在模型中心「常规办公」绑定（默认本地 Herdsman，可绑定免费云端模型），也可用 engine/model 参数临时指定。本地模型通常几秒到几十秒，冷启动约 20 秒+；不消耗主模型 token。"
}

func (t routineLLMTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "prompt":{"type":"string","description":"要执行的常规任务指令（完整描述输入与期望输出）"},
  "system":{"type":"string","description":"可选：系统提示词，如输出格式/角色要求"},
  "engine":{"type":"string","description":"可选：临时指定引擎（如 herdsman/ollama/opencode-zen），默认用模型中心「常规办公」绑定，未绑定则用本地 herdsman"},
  "model":{"type":"string","description":"可选：临时指定模型名，默认用「常规办公」绑定或引擎默认模型"},
  "max_tokens":{"type":"integer","description":"可选：输出 token 上限"},
  "temperature":{"type":"number","description":"可选：采样温度（默认 0.3，常规任务偏低）"}
},
"required":["prompt"]
}`)
}

func (t routineLLMTool) ReadOnly() bool { return true }

func (t routineLLMTool) CompactDescription() string {
	return "通用文本推理（默认本地/免费）：摘要、归一化、抽取、改写"
}

func (t routineLLMTool) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"prompt":{"type":"string"},"system":{"type":"string"},"engine":{"type":"string"},"model":{"type":"string"}},"required":["prompt"]}`)
}

func (t routineLLMTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if t.a == nil || t.a.client == nil {
		return "", fmt.Errorf("routine_llm: 模型服务不可用")
	}
	var p struct {
		Prompt      string  `json:"prompt"`
		System      string  `json:"system"`
		Engine      string  `json:"engine"`
		Model       string  `json:"model"`
		MaxTokens   int     `json:"max_tokens"`
		Temperature float64 `json:"temperature"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("routine_llm: 参数无效: %w", err)
	}
	if strings.TrimSpace(p.Prompt) == "" {
		return "", fmt.Errorf("routine_llm: prompt 不能为空")
	}

	engine, model, err := t.a.resolveRoutineTarget(p.Engine, p.Model)
	if err != nil {
		return "", err
	}
	temperature := p.Temperature
	if temperature <= 0 {
		temperature = 0.3
	}
	reply, err := t.a.client.ChatSimpleStreamWithOptions(ctx, model, p.System, p.Prompt, ai.ChatSimpleOptions{
		EngineID:       engine,
		Temperature:    temperature,
		MaxTokens:      p.MaxTokens,
		TimeoutMinutes: 3,
	})
	if err != nil {
		return "", fmt.Errorf("routine_llm（%s/%s）失败: %w", engine, orDefault(model, "默认模型"), err)
	}
	return strings.TrimSpace(reply), nil
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

// resolveRoutineTarget 解析 routine_llm 的目标引擎/模型。
// 优先级：显式覆盖（engine/model 参数）→ 模型中心「常规办公」绑定 →
// 默认本地 herdsman（引擎默认模型）。不做强制路由，只是工具入口的目标解析。
func (c *core) resolveRoutineTarget(overrideEngine, overrideModel string) (engine, model string, err error) {
	if overrideEngine != "" {
		if c.engineMgr == nil {
			return "", "", fmt.Errorf("routine_llm: 模型引擎管理器不可用")
		}
		e, ok := c.engineMgr.GetEngine(overrideEngine)
		if !ok {
			return "", "", fmt.Errorf("routine_llm: 引擎 %s 不存在", overrideEngine)
		}
		if !e.Enabled {
			return "", "", fmt.Errorf("routine_llm: 引擎 %s 未启用", overrideEngine)
		}
		return overrideEngine, routineResolveModel(e, overrideModel), nil
	}

	// 模型中心「常规办公」绑定（功能启用 + 引擎存在且启用）
	if c.cfg.GetFeatureModelEnabled("routine") {
		if eng, m := c.cfg.GetFeatureModel("routine"); eng != "" && m != "" {
			if c.engineMgr != nil {
				if e, ok := c.engineMgr.GetEngine(eng); ok && e.Enabled {
					return eng, routineResolveModel(e, m), nil
				}
			}
			return "", "", fmt.Errorf("routine_llm: 常规模型引擎 %s 不存在或未启用，请在模型中心重新绑定", eng)
		}
	}

	// 默认本地 herdsman
	if c.engineMgr != nil {
		if e, ok := c.engineMgr.GetEngine("herdsman"); ok && e.Enabled {
			return "herdsman", routineResolveModel(e, ""), nil
		}
	}
	return "", "", fmt.Errorf("routine_llm: 未配置常规模型（默认本地 herdsman 不可用），请在模型中心绑定「常规办公」模型或显式传 engine/model")
}

// routineResolveModel 解析 routine_llm 的实际模型名：
// 显式指定 > 第一个 running 的 LLM > 引擎默认模型 > 第一个 LLM。
// 避免空模型名回退到全局云端模型（如 grok-4.20）发给本地服务导致 404。
func routineResolveModel(e *modelengine.EngineConfig, explicit string) string {
	if explicit != "" {
		return explicit
	}
	for _, m := range e.Models {
		if m.Status == "running" {
			return m.ID
		}
	}
	if e.DefaultModel != "" {
		return e.DefaultModel
	}
	if len(e.Models) > 0 {
		return e.Models[0].ID
	}
	return ""
}

var _ tool.Tool = routineLLMTool{}
