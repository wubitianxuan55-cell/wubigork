package app

// IN2-03（2026-10-02 全仓审计·批次十）：app 层曾有五份「backend 名 → 手写构造
// ai 图片后端 → SetImageBackend」的 switch/构造样板（App.initImageBackend、
// mediaState.SetImageBackend、buildImageClientFor、writingState.restoreImageBackend、
// mediaState.GetComfyUILoras），分支集合已互相漂移（restore 副本缺 glm 分支、其
// comfyui 分支里还嵌了一段 herdsman 覆盖）。本文件把它们收敛为单一解析器：
// 后端名 + 当前环境（ComfyUI 地址 / 引擎启用状态 / GLM Key）→ 后端实例；构造
// 不了的原因用结构化错误带回，各调用方保留自己的站点语义（设置页原文案报错 /
// 启动恢复静默跳过 / 初始化 glm 回退 xAI）。
//
// 与 internal/ai 注册表（NewImageBackend）的关系：注册表工厂自带严格校验
// （空 BaseURL/Key 直接报错），而 app 层五份副本的既有口径是「先自行前置校验
// 再宽松构造」，为保证零行为变化，本解析器沿用直接构造函数，不绕注册表。

import (
	"strings"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

// 图片后端暂不可构造的原因（imageBackendResolveError.Reason）。
const (
	reasonEngineDisabled  = "engine-disabled"     // 引擎不存在或未启用（herdsman/ollama/glm）
	reasonGLMKeyMissing   = "glm-key-missing"     // GLM 引擎已启用但 API Key 未配置
	reasonComfyURLMissing = "comfyui-url-missing" // ComfyUI 地址未配置
)

// imageBackendResolveError 后端名当前无法构造实例的结构化原因。调用方按各自
// 口径消费：转用户报错文案、静默跳过、或按 Kind 决定回退（glm→xAI）。
type imageBackendResolveError struct {
	Reason  string // reason* 常量之一
	Backend string // 请求的后端名
}

func (e *imageBackendResolveError) Error() string {
	switch e.Reason {
	case reasonEngineDisabled:
		return e.Backend + " 引擎未启用，请先在模型中心启用"
	case reasonGLMKeyMissing:
		return "GLM API Key 未配置，请先在模型中心 GLM 卡片保存 Key"
	case reasonComfyURLMissing:
		return "未配置 ComfyUI 地址"
	}
	return "图片后端不可用: " + e.Backend
}

// resolvedImageBackend 解析结果。xai（含空名）走 client 内置 xAI 原生管线，
// 无需实例——Backend 为 nil。
type resolvedImageBackend struct {
	Backend ai.ImageBackend // 图片后端实例；xai 为 nil
	Kind    string          // 规范后端名（"xai"/"comfyui"/"herdsman"/"ollama"/"glm"）
	BaseURL string          // 构造所用服务地址（引擎来源时为 eng.BaseURL；仅日志用）
}

// resolveImageBackend 按后端名与当前环境构造图片生成后端（IN2-03 单一来源）。
// "xai"、空名及未知名一律 fail-closed 回 xAI 原生管线（Backend=nil, Kind="xai"），
// 与收敛前各副本 default 分支一致。前置条件不满足时返回 *imageBackendResolveError
// （此时 Kind 仍为规范名，供调用方按名字决定回退/跳过）。
func resolveImageBackend(name string, cfg *config.Config, mgr *modelengine.Manager) (resolvedImageBackend, error) {
	switch name {
	case "comfyui":
		if cfg == nil || strings.TrimSpace(cfg.ComfyUIURL) == "" {
			return resolvedImageBackend{Kind: "comfyui"}, &imageBackendResolveError{Reason: reasonComfyURLMissing, Backend: name}
		}
		return resolvedImageBackend{Backend: ai.NewComfyUIBackend(cfg.ComfyUIURL), Kind: "comfyui", BaseURL: cfg.ComfyUIURL}, nil
	case "herdsman", "ollama":
		eng, err := imageEngineReady(mgr, name)
		if err != nil {
			return resolvedImageBackend{Kind: name}, err
		}
		return resolvedImageBackend{Backend: ai.NewOpenAIImageBackend(eng.BaseURL, eng.APIKey), Kind: name, BaseURL: eng.BaseURL}, nil
	case "glm":
		eng, err := imageEngineReady(mgr, "glm")
		if err != nil {
			return resolvedImageBackend{Kind: "glm"}, err
		}
		key := mgr.GLMKey()
		if key == "" {
			return resolvedImageBackend{Kind: "glm"}, &imageBackendResolveError{Reason: reasonGLMKeyMissing, Backend: name}
		}
		return resolvedImageBackend{Backend: ai.NewGLMImageBackend(eng.BaseURL, key), Kind: "glm", BaseURL: eng.BaseURL}, nil
	default: // "xai"、空或未知名
		return resolvedImageBackend{Kind: "xai"}, nil
	}
}

// imageEngineReady 引擎存在且启用则返回配置；否则带 reasonEngineDisabled。
func imageEngineReady(mgr *modelengine.Manager, id string) (*modelengine.EngineConfig, error) {
	if mgr == nil {
		return nil, &imageBackendResolveError{Reason: reasonEngineDisabled, Backend: id}
	}
	eng, ok := mgr.GetEngine(id)
	if !ok || eng == nil || !eng.Enabled {
		return nil, &imageBackendResolveError{Reason: reasonEngineDisabled, Backend: id}
	}
	return eng, nil
}

// imageEngineDisplayName 后端名 → 用户报错文案里的引擎显示名（原
// mediaState.SetImageBackend 各分支的首字母大写口径：Herdsman/Ollama/GLM）。
func imageEngineDisplayName(backend string) string {
	switch backend {
	case "herdsman":
		return "Herdsman"
	case "ollama":
		return "Ollama"
	case "glm":
		return "GLM"
	}
	return backend
}
