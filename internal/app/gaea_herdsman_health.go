package app

// Herdsman 服务健康检查（H0-2）：一次性探测本机 herdsman 服务的端口占用、
// API 存活，并按已装模型归类各本地能力链（聊天/视觉/Embedding/Rerank/OCR/
// 文档解析/ASR/TTS/生图/翻译）是否就绪，结果供前端一次性展示。

import (
	"time"

	"github.com/gaea/gaea/internal/herdsman"
)

// HerdsmanHealth 对 Herdsman 本地服务执行一次性健康检查：
// 端口/API 探测由 herdsman.HealthCheck 完成；模型列表取自模型中心 herdsman
// 引擎（后端已按 classifyModelKind 分类出 Kind），转换为健康检查用的
// herdsman.ModelInfo 后一并传入。结果可直接 JSON 序列化供前端展示。
func (a *App) HerdsmanHealth() herdsman.HealthResult {
	if a.engineMgr == nil {
		r := herdsman.NewResult()
		r.Summary = []string{"模型引擎管理器未初始化"}
		return r
	}
	eng, ok := a.engineMgr.GetEngine("herdsman")
	if !ok {
		r := herdsman.NewResult()
		r.Summary = []string{"Herdsman 引擎未配置"}
		return r
	}

	models := make([]herdsman.ModelInfo, 0, len(eng.Models))
	for _, m := range eng.Models {
		models = append(models, herdsman.ModelInfo{
			ID:         m.ID,
			Status:     m.Status,
			Capability: modelKindToCapability(m.Kind),
		})
	}
	return herdsman.HealthCheck(eng.BaseURL, models, 3*time.Second)
}

// modelKindToCapability 把模型中心引擎的 Kind 归类映射为健康检查的能力键。
// tts/stt/embedding/rerank/image 的归类明确且无歧义，直接直译；"ocr" 与 "llm"
// 不映射——kind=ocr 同时覆盖 paddleocr（→ocr）与 mineru（→parse）两类，kind=llm
// 覆盖聊天/视觉模型，二者交由 ClassifyModelCapability 按 ID 关键词精确区分
// （如 paddleocr → ocr、mineru → parse、qwen2.5 → chat）。
func modelKindToCapability(kind string) string {
	switch kind {
	case "tts":
		return "tts"
	case "stt":
		return "asr"
	case "embedding":
		return "embedding"
	case "rerank":
		return "rerank"
	case "image":
		return "imagegen"
	}
	return ""
}
