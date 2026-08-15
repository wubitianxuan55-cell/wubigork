package app

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/gaea/gaea/internal/app/board"
)

// moduleOfIntent (板块ID, 意图) → 主脑模块 ID。模块空间与板块空间不同构：
// whisper 是 chat 板块的模块化身（§3.1：chat 板块意图域 = whisper），office
// 是 gaea 办公域的模块化身（D8 决策保留 office 模块，office.create→GaeaSend）。
// manifest 声明的意图必须出现在此表，否则装配报错（完整性断言）。
var moduleOfIntent = map[string]string{
	"novel.create_chapter": "novel",
	"chat.chat":            "whisper",
	"imagegen.generate":    "imagegen",
	"gaea.chat":            "gaea",
	"gaea.create":          "office",
}

// intentHandlers 主脑模块意图 → App 方法闭包（manifest 声明的意图处理器）。
// 闭包必须包住现有 App 方法；不经由任何模块的直接路径（§5.2）。
func (a *App) intentHandlers() map[string]map[string]Handler {
	return map[string]map[string]Handler{
		"gaea": {
			"chat": func(input map[string]any) (map[string]any, error) {
				msg, _ := input["message"].(string)
				return a.ChatGeneral(msg)
			},
		},
		"whisper": {
			"chat": func(input map[string]any) (map[string]any, error) {
				msg, _ := input["message"].(string)
				pid, _ := input["personality_id"].(string)
				if pid == "" {
					pid = "轻语"
				}
				return a.WhisperChat(msg, pid, false)
			},
		},
		"novel": {
			"create_chapter": func(input map[string]any) (map[string]any, error) {
				setting, _ := input["setting"].(string)
				plotReq, _ := input["plot_req"].(string)
				num := intField(input, "chapter_num", 1)
				return a.CreateChapter(setting, "", plotReq, num, "", "", 0, 0)
			},
		},
		"imagegen": {
			"generate": func(input map[string]any) (map[string]any, error) {
				prompt, _ := input["prompt"].(string)
				negative, _ := input["negative"].(string)
				size, _ := input["size"].(string)
				style, _ := input["style"].(string)
				model, _ := input["model"].(string)
				return a.GenerateFreeImage(prompt, negative, size, style, model,
					intField(input, "seed", 0), intField(input, "n", 1), "")
			},
		},
		"office": {
			"create": func(input map[string]any) (map[string]any, error) {
				msg, _ := input["prompt"].(string)
				if msg == "" {
					msg, _ = input["message"].(string)
				}
				if msg == "" {
					return nil, fmt.Errorf("office: 缺少 prompt/message 输入")
				}
				// D8 决策：office.create 路由到现成 GaeaSend（异步，结果经
				// gaea-event 回调），不实现不存在的 ProposalCreate。App 未装配
				// （测试等退化场景）时跳过引擎初始化，直接返回提交语义。
				if a.core != nil {
					a.GaeaSend(msg)
				}
				return map[string]any{"status": "submitted", "message": msg}, nil
			},
		},
	}
}

// resolveIntent 由 (板块, 意图) 解析主脑模块与处理器（FillFromManifests 的
// resolver）。返回 ok=false 表示无处理器（未知意图或闭包表缺失）。
func (a *App) resolveIntent(boardID, intent string) (string, Handler, bool) {
	moduleID, ok := moduleOfIntent[boardID+"."+intent]
	if !ok {
		return "", nil, false
	}
	h, ok := a.intentHandlers()[moduleID][intent]
	if !ok {
		return "", nil, false
	}
	return moduleID, h, true
}

// initModules 注册主脑可派发的模块（§5.2 manifest 驱动）：板块清单声明意图 →
// resolveIntent 解析处理器闭包 → FillFromManifests 装配。
//
// 完整性断言（缺陷 2 机器保证）：任何 manifest 声明的意图解析不到处理器，
// FillFromManifests 返回 error 并记录到注册表（a.modules.Err() 可读），启动
// 路径日志显式报错——不再有「intent 无 handler 静默跳过」。
func (a *App) initModules() {
	a.modules = NewModuleRegistry()
	if err := a.modules.FillFromManifests(board.Builtins(), a.resolveIntent); err != nil {
		slog.Error("initModules: 板块 manifest 装配失败（intent 无 handler，缺陷 2 防复发）", "error", err)
	}
}

// CheckModuleIntegrity 启动自检：用 canonical 板块清单重跑 manifest 装配并
// 返回完整性错误（intent 无 handler / 未知意图 / 重复模块 → 显式 error，
// 不静默）。Startup 的 initModules 已装配一次；此方法供自检/测试显式断言。
func (a *App) CheckModuleIntegrity() error {
	reg := NewModuleRegistry()
	if err := reg.FillFromManifests(board.Builtins(), a.resolveIntent); err != nil {
		return err
	}
	return nil
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
