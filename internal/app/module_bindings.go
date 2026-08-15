package app

import (
	"encoding/json"
	"fmt"
)

// initModules 注册主脑可派发的模块（闭包包住现有 App 方法；不经由任何模块的直接路径）。
func (a *App) initModules() {
	a.modules = NewModuleRegistry()
	_ = a.modules.Register(Module{
		ID: "gaea", Name: "主脑",
		Intents: []string{"chat"},
		Handle: func(input map[string]any) (map[string]any, error) {
			msg, _ := input["message"].(string)
			out, err := a.ChatGeneral(msg)
			if err != nil {
				return nil, err
			}
			return out, nil
		},
	})
	_ = a.modules.Register(Module{
		ID: "whisper", Name: "轻语",
		Intents: []string{"chat"},
		Handle: func(input map[string]any) (map[string]any, error) {
			msg, _ := input["message"].(string)
			pid, _ := input["personality_id"].(string)
			if pid == "" {
				pid = "轻语"
			}
			return a.WhisperChat(msg, pid, false)
		},
	})
	_ = a.modules.Register(Module{
		ID: "novel", Name: "小说",
		Intents: []string{"create_chapter"},
		Handle: func(input map[string]any) (map[string]any, error) {
			setting, _ := input["setting"].(string)
			plotReq, _ := input["plot_req"].(string)
			num := intField(input, "chapter_num", 1)
			return a.CreateChapter(setting, "", plotReq, num, "", "", 0, 0)
		},
	})
	_ = a.modules.Register(Module{
		ID: "imagegen", Name: "绘梦",
		Intents: []string{"generate"},
		Handle: func(input map[string]any) (map[string]any, error) {
			prompt, _ := input["prompt"].(string)
			negative, _ := input["negative"].(string)
			size, _ := input["size"].(string)
			style, _ := input["style"].(string)
			model, _ := input["model"].(string)
			return a.GenerateFreeImage(prompt, negative, size, style, model,
				intField(input, "seed", 0), intField(input, "n", 1), "")
		},
	})
	_ = a.modules.Register(Module{
		ID: "office", Name: "方案",
		Intents: []string{"create"},
		Handle: func(input map[string]any) (map[string]any, error) {
			msg, _ := input["prompt"].(string)
			if msg == "" {
				msg, _ = input["message"].(string)
			}
			if msg == "" {
				return nil, fmt.Errorf("office: 缺少 prompt/message 输入")
			}
			// D8 决策：office.create 路由到现成 GaeaSend（异步，结果经 gaea-event
			// 回调），不实现不存在的 ProposalCreate。App 未装配（测试等退化场景）
			// 时跳过引擎初始化，直接返回提交语义，避免裸 App 触发 GaeaInit。
			if a.core != nil {
				a.GaeaSend(msg)
			}
			return map[string]any{"status": "submitted", "message": msg}, nil
		},
	})
}

func intField(input map[string]any, key string, def int) int {
	if v, ok := input[key].(float64); ok {
		return int(v)
	}
	if v, ok := input[key].(int); ok {
		return v
	}
	return def
}

// RunModule 统一派发（Wails 绑定；主脑可选编排用）。
func (a *App) RunModule(moduleID, intent, inputJSON string) (string, error) {
	if a.modules == nil {
		return "", fmt.Errorf("modules not initialized")
	}
	var input map[string]any
	if inputJSON != "" {
		if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
			return "", err
		}
	}
	out, err := a.modules.Dispatch(moduleID, intent, input)
	if err != nil {
		return "", err
	}
	b, _ := json.Marshal(out)
	return string(b), nil
}
