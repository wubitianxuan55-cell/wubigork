package app

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/config"
)

// resolveChatModelRoute 解析聊天主链路的实际路由，并在显式功能绑定命中时
// 先确保绑定引擎就绪。
//
// 与通用 routeModel 的关键差异：聊天一旦绑定了具体引擎/模型，进程未运行
// 不能直接进入全局故障转移。自管 llama/Strata 由本层按需拉起；启动失败
// 原样返回错误，让聊天界面显示“绑定模型未就绪”，而不是悄悄换成 xAI。
func (a *App) resolveChatModelRoute() (engine, model, source string, err error) {
	if a == nil || a.core == nil || a.cfg == nil {
		return "", "", "", fmt.Errorf("聊天模型路由未初始化")
	}
	if a.engineMgr == nil {
		return "", "", "", fmt.Errorf("模型引擎管理器未初始化")
	}

	if a.cfg.GetFeatureModelEnabled("chat") {
		boundEngine, boundModel := a.cfg.GetFeatureModel("chat")
		if boundEngine != "" && boundModel != "" {
			if bound, ok := a.engineMgr.GetEngine(boundEngine); ok && bound.Enabled &&
				!isTTSOnlyEngine(bound.Type) &&
				!(a.cfg.GetOfflineMode() && !bound.Type.IsLocal()) {
				boundModel = a.normalizeChatBoundModel(boundEngine, boundModel)
				if ensureErr := ensureChatBindingEngineFn(a, boundEngine, boundModel); ensureErr != nil {
					return "", "", "", fmt.Errorf(
						"聊天绑定的模型 %s/%s 未能启动: %w",
						boundEngine, boundModel, ensureErr,
					)
				}
				a.emitModelRoute("chat", boundEngine, boundModel, "feature")
				return boundEngine, boundModel, "feature", nil
			}
		}
	}

	engine, model, source = a.routeModel("chat")
	return engine, model, source, nil
}

// normalizeChatBoundModel 修复历史持久化中的 llama 裸 GGUF 路径。llama-server
// 虽通常忽略 model 字段，但模型中心/回执会因此显示易漂移的绝对路径；自管
// 引擎只有稳定档位 qwen35，聊天绑定统一落这个 ID。
func (a *App) normalizeChatBoundModel(engineID, model string) string {
	if engineID != "llama" {
		return model
	}
	if _, ok := managedProfileForModel(llamaProfiles, model); ok {
		return model
	}
	lower := strings.ToLower(strings.TrimSpace(model))
	if strings.HasSuffix(lower, ".gguf") || strings.Contains(lower, "qwen3.6-35b") {
		const stable = "qwen35"
		a.cfg.SetFeatureModel("chat", engineID, stable)
		if err := config.Save(config.KeyFuncChatModel, stable); err != nil {
			slog.Warn("修复聊天 llama 模型绑定失败，本次请求仍使用稳定档位",
				"from", model, "to", stable, "error", err)
		} else {
			slog.Info("聊天 llama 模型绑定已归一为稳定档位", "from", model, "to", stable)
		}
		return stable
	}
	return model
}

// ensureChatBindingEngineFn 是进程启动缝：生产走本机真实生命周期，测试可
// 替换为记录/失败桩，避免测试机真的拉起 35B。
var ensureChatBindingEngineFn = func(a *App, engineID, model string) error {
	return a.ensureChatBindingEngine(engineID, model)
}

func (a *App) ensureChatBindingEngine(engineID, model string) error {
	switch engineID {
	case "llama":
		result := a.qwen35Ensure()
		if running, _ := result["running"].(bool); !running {
			reason := strings.TrimSpace(fmt.Sprint(result["error"]))
			if reason == "" {
				reason = "未知启动错误"
			}
			return fmt.Errorf("%s", reason)
		}
		served, ok := waitLlamaModelsReady(30 * time.Second)
		if !ok {
			return fmt.Errorf("llama-server 已启动但 /v1/models 未就绪")
		}
		profile := llamaProfileFromModels(served)
		a.syncManagedCatalog("llama", llamaProfiles, served, profile)
		if a.core != nil && a.engineMgr != nil {
			a.engineMgr.SetStatus("llama", managedEngineStatus("llama", true, len(served), 0, ""))
		}
		return nil

	case "strata":
		profile, ok := managedProfileForModel(strataProfiles, model)
		if !ok {
			return fmt.Errorf("模型 %s 不在 Strata 档位目录", model)
		}
		result := a.StrataEngineStart(profile.ID)
		if started, _ := result["started"].(bool); started {
			return nil
		}
		reason := strings.TrimSpace(fmt.Sprint(result["reason"]))
		if reason == "already-running" {
			active := strings.TrimSpace(fmt.Sprint(result["profile"]))
			if active == "" || active == profile.ID {
				return nil
			}
			return fmt.Errorf("Strata 正在运行其它档位 %s", active)
		}
		detail := strings.TrimSpace(fmt.Sprint(result["error"]))
		if detail == "" {
			detail = reason
		}
		if detail == "" {
			detail = "未知启动错误"
		}
		return fmt.Errorf("%s", detail)
	}

	// 云端与按需加载引擎（herdsman）沿用各自既有连接/加载链。
	return nil
}

func waitLlamaModelsReady(timeout time.Duration) ([]string, bool) {
	deadline := time.Now().Add(timeout)
	for {
		if served, ok := llamaProbe(); ok {
			return served, true
		}
		if time.Now().After(deadline) {
			return nil, false
		}
		time.Sleep(500 * time.Millisecond)
	}
}
