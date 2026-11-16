package app

// wenxue_engine_preflight.go — 图文回合本地引擎预检 + 图文空闲租约（2026-11-16）。
//
// 背景：图文引擎（wx）构建期只做 bridge.SetFeature 注入（wenxue_engine.go），
// 请求期 ai.Client 直连。绑定本地托管引擎（llama/strata/whirl）时，进程被
// 空闲收割器停机（managed_engine_idle.go，全局阈值默认 5 分钟）或尚未拉起时，
// 请求首包即 connection refused，被 isTransferableChatError 判为可转移 →
// 故障转移悄悄换引擎（开关关闭时则原样报错）。聊天板块有 resolveChatModelRoute
// 预检（chat_engine_preflight.go），图文此前没有。
//
// 本文件给图文补两件事（不采用聊天板块函数——图文回落链不同）：
//  1. 每轮回合前预检：回落链 wenxue 绑定 → gaea 绑定 → 全局活跃，与
//     wenxueBuildController 的 bridge.SetFeature 注入链同构。引擎可能在任意
//     两轮之间被收割/崩出，故每轮发送都过闸，不限于首轮（WenxueSteer 是
//     运行中插话，回合内请求持续记账，不重复预检）。
//  2. 图文空闲租约（10 分钟）：图文回合碰过的托管引擎，在其最后一次图文
//     使用满 10 分钟前，收割器不得停机。全局阈值（默认 5 分钟）仍是硬底线
//     ——两闸同时满足才收割。图文创作轮间隔长（写作/评审间隙远超聊天），
//     只吃全局阈值会反复杀引擎导致每轮冷启动。
//
// 就绪等待预算沿用引擎自身口径：llama 冷启动 llamaEnsure 内部已等
// qwen35StartWait（300s），strata/whirl 的 Start 同为 300s；本层补等的只是
// 「他方正在拉起」的在途窗口，上限对齐 qwen35StartWait——到期时对方 ensure
// 的自有等待必然已出结果（成功探活或自停）。

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// wenxueIdleLease 图文空闲租约时长。
const wenxueIdleLease = 10 * time.Minute

// wenxueLlamaReadyWait 图文侧 llama 在途补等上限（var 供测试改写）。
var wenxueLlamaReadyWait = qwen35StartWait

// wenxueLeases 图文引擎租约表（engineID → 最后一次图文回合就绪时刻）。
// 进程级小状态，与收割器只读共享；收割判定经 wenxueLeaseAlive。
var (
	wenxueLeasesMu sync.Mutex
	wenxueLeases   = map[string]time.Time{}
)

// wenxueTouchEngineAt 记一笔图文引擎使用（注入时刻便于测试）。
func wenxueTouchEngineAt(engineID string, at time.Time) {
	if engineID == "" {
		return
	}
	wenxueLeasesMu.Lock()
	defer wenxueLeasesMu.Unlock()
	wenxueLeases[engineID] = at
}

// wenxueLeaseAlive 报告引擎是否仍在图文租约保护期内（收割器谓词）。
func wenxueLeaseAlive(engineID string, now time.Time) bool {
	wenxueLeasesMu.Lock()
	defer wenxueLeasesMu.Unlock()
	last, ok := wenxueLeases[engineID]
	if !ok {
		return false
	}
	return now.Sub(last) < wenxueIdleLease
}

// wenxuePreflightFn 进程启动缝：测试注入桩，避免测试机真拉起大模型
// （同 ensureChatBindingEngineFn 先例）。返回值=是否发生了冷启动（供调用方
// 发「正在启动」phase 事件——已在运行时静默，不每轮刷屏）。
var wenxuePreflightFn = func(a *App, engineID, model string) (bool, error) {
	return a.wenxueEnsureEngine(engineID, model)
}

// wenxuePreflight 图文回合预检：解析本回合生效的 (engine, model)，托管本地
// 引擎未就绪则按需拉起。失败返回错误（WenxueSend 转 error 事件），绝不静默
// 让请求落到 connection refused → 故障转移换引擎。
func (a *App) wenxuePreflight() error {
	if a == nil || a.core == nil || a.cfg == nil || a.engineMgr == nil {
		return nil
	}
	eng, mdl := a.wenxueResolveRoute()
	if eng == "" {
		return nil
	}
	e, ok := a.engineMgr.GetEngine(eng)
	if !ok || !e.Enabled || isTTSOnlyEngine(e.Type) ||
		(a.cfg.GetOfflineMode() && !e.Type.IsLocal()) {
		// 与聊天预检同口径：守卫不过交回请求期既有行为（配置类问题换引擎
		// 解决不了，也不该在这里拦）。
		return nil
	}
	if _, err := wenxuePreflightFn(a, eng, mdl); err != nil {
		return fmt.Errorf("图文绑定的模型 %s/%s 未能就绪: %w", eng, mdl, err)
	}
	wenxueTouchEngineAt(eng, time.Now())
	return nil
}

// wenxueResolveRoute 解析图文回合生效引擎。回落链与 wenxueBuildController
// 的 bridge.SetFeature 注入同构：wenxue 绑定 → gaea 绑定 → 全局活跃。
func (a *App) wenxueResolveRoute() (engine, model string) {
	if a.cfg.GetFeatureModelEnabled("wenxue") {
		engine, model = a.cfg.GetFeatureModel("wenxue")
	}
	if engine == "" {
		engine, model = a.cfg.GetFeatureModel("gaea")
	}
	if engine == "" {
		engine = a.core.GetActiveEngine()
		model = a.core.GetActiveModel()
	}
	if model == "" {
		// 与 ai.Client 请求期动态解析对齐：模型空 → 引擎默认档（否则 llama
		// 档位查找拿空串必误报不在目录）。
		if a.engineMgr != nil {
			if dm, err := a.engineMgr.GetDefaultModel(engine); err == nil {
				model = dm
			}
		}
	}
	// llama 裸 GGUF 路径归一只读版（聊天侧的持久化归一是 chat 键域，图文
	// 不代写配置；归一只为保证档位查找命中）。
	if engine == "llama" {
		if _, ok := managedProfileForModel(llamaProfiles, model); !ok {
			if stable := llamaProfileFromModels([]string{model}); stable != "" {
				model = stable
			}
		}
	}
	return engine, model
}

// wenxueEnsureEngine 托管本地引擎按需拉起（图文预算版）。冷启动先发
// 「正在启动本地模型」phase（已在运行时静默）。llama 在 llamaEnsure 之上
// 多一步在途补等；strata/whirl 的 Start 自带 300s 就绪等待，与聊天同构；
// 云端与按需加载引擎（herdsman）沿用各自既有连接/加载链。
func (a *App) wenxueEnsureEngine(engineID, model string) (bool, error) {
	switch engineID {
	case "llama":
		profile, ok := managedProfileForModel(llamaProfiles, model)
		if !ok {
			return false, fmt.Errorf("模型 %s 不在 llama 档位目录", model)
		}
		cold := false
		if _, running := llamaProbe(); !running {
			cold = true
			a.emit(wenxueEventChannel, map[string]interface{}{"kind": "phase", "text": "正在启动本地模型"})
		}
		result := a.llamaEnsure(profile.ID)
		if running, _ := result["running"].(bool); !running {
			return cold, fmt.Errorf("%s", wenxueEnsureReason(result))
		}
		if active := strings.TrimSpace(fmt.Sprint(result["profile"])); active != "" && active != profile.ID {
			return cold, fmt.Errorf("llama 正在运行其它档位 %s，无法就绪图文绑定的 %s", active, profile.ID)
		}
		// llamaEnsure 冷启动路径内部已等 qwen35StartWait；此处补等覆盖
		// reason=starting（他方在途拉起）窗口。
		served, ok := waitLlamaModelsReady(wenxueLlamaReadyWait)
		if !ok {
			return cold, fmt.Errorf("llama-server 已启动但 /v1/models 未就绪")
		}
		a.syncManagedCatalog("llama", llamaProfiles, served, llamaProfileFromModels(served))
		if a.engineMgr != nil {
			a.engineMgr.SetStatus("llama", managedEngineStatus("llama", true, len(served), 0, ""))
		}
		return cold, nil

	case "strata":
		profile, ok := managedProfileForModel(strataProfiles, model)
		if !ok {
			return false, fmt.Errorf("模型 %s 不在 Strata 档位目录", model)
		}
		cold := false
		if _, running := strataProbe(); !running {
			cold = true
			a.emit(wenxueEventChannel, map[string]interface{}{"kind": "phase", "text": "正在启动本地模型"})
		}
		result := a.StrataEngineStart(profile.ID)
		if started, _ := result["started"].(bool); started {
			return cold, nil
		}
		reason := strings.TrimSpace(fmt.Sprint(result["reason"]))
		if reason == "already-running" {
			active := strings.TrimSpace(fmt.Sprint(result["profile"]))
			if active == "" || active == profile.ID {
				return cold, nil
			}
			return cold, fmt.Errorf("Strata 正在运行其它档位 %s", active)
		}
		detail := wenxueEnsureReason(result)
		return cold, fmt.Errorf("%s", detail)

	case "whirl":
		profile, ok := managedProfileForModel(whirlProfiles, model)
		if !ok {
			return false, fmt.Errorf("模型 %s 不在 WHIRL 档位目录", model)
		}
		cold := false
		if _, running := whirlProbe(); !running {
			cold = true
			a.emit(wenxueEventChannel, map[string]interface{}{"kind": "phase", "text": "正在启动本地模型"})
		}
		result := a.WhirlEngineStart(profile.ID)
		if started, _ := result["started"].(bool); started {
			return cold, nil
		}
		reason := strings.TrimSpace(fmt.Sprint(result["reason"]))
		if reason == "already-running" {
			active := strings.TrimSpace(fmt.Sprint(result["profile"]))
			if active == "" || active == profile.ID {
				return cold, nil
			}
			return cold, fmt.Errorf("WHIRL 正在运行其它档位 %s", active)
		}
		detail := wenxueEnsureReason(result)
		return cold, fmt.Errorf("%s", detail)
	}

	// 云端与按需加载引擎（herdsman）：沿用各自既有连接/加载链。
	return false, nil
}

// wenxueEnsureReason 从 ensure 结果取人话错误（error → reason → 兜底）。
func wenxueEnsureReason(result map[string]interface{}) string {
	if v := strings.TrimSpace(fmt.Sprint(result["error"])); v != "" && v != "<nil>" {
		return v
	}
	if v := strings.TrimSpace(fmt.Sprint(result["reason"])); v != "" && v != "<nil>" {
		return v
	}
	return "未知启动错误"
}
