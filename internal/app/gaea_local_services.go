package app

// gaea_local_services.go — 模型中心「本地服务目录」（电脑操控配套刀，2026-11-20）。
//
// 背景：凡不走 engines.json 目录、由功能模块直连的本地服务，模型中心一律
// 看不见——本地 OCR（docmd 直连 Unlimited-OCR-2；2026-11-26 精简后只此一张卡，
// OvisOCR2 / python 后端 / tesseract 已退役）、本地识图（qwen35 mmproj，功能绑定
// visionRoute 内嵌）、本地 STT（语音 V2 内嵌 sherpa）。本绑定把这些以模型卡
// 形态下发（kind 对齐既有过滤器），专业页/语音页现有 UI 零改动即可展示；
// OCR 卡的「设为 OCR」经 SetActiveOCRModel 的 local: 分支持久化到
// active_ocr_local，GaeaOCRText 本地腿据此定点（详见 gaea_ocr.go）。
//
// 探活纪律（对齐 docmd/ocr_endpoints.go）：只做被动 /health 探测（800ms），
// 绝不触发服务拉起——目录列表只是看一眼。

import (
	"path/filepath"

	"github.com/gaea/gaea/internal/docmd"
)

// LocalServiceModel 模型中心本地服务卡（形态对齐 ModelCardData 消费面）。
type LocalServiceModel struct {
	EngineID string `json:"engineId"` // local:<kind>（前端按此识别本目录卡）
	ModelID  string `json:"modelId"`
	Name     string `json:"name"`
	Kind     string `json:"kind"` // ocr / vision / stt
	Detail   string `json:"detail"`
	Ready    bool   `json:"ready"` // 被动探活/文件存在
}

// localOCRServiceCard 一张本地 OCR 卡：被动探活对应端口。
func localOCRServiceCard(engineID, modelID, name, base, detail string) LocalServiceModel {
	return LocalServiceModel{
		EngineID: engineID, ModelID: modelID, Name: name, Kind: "ocr",
		Detail: detail, Ready: docmd.LocalServiceHealth(base),
	}
}

// GaeaLocalServiceModels 返回本机内嵌/直连服务的模型卡（模型中心合并展示）。
func (a *App) GaeaLocalServiceModels() []LocalServiceModel {
	// 本地 OCR：卡 ID = local:<docmd kind>，「设为 OCR」定点经 SetActiveOCRModel 的 local: 支。
	cards := []LocalServiceModel{
		localOCRServiceCard("local:uocr2", "unlimited-ocr2-llama", "Unlimited-OCR-2（本地·8139）",
			docmd.Uocr2LlamaAddr(), "本地唯一 OCR 引擎（版式/表格还原最好）：llama 后端，Q8 单页 6s；未拉起时按需自动拉起"),
	}
	// 本地识图：qwen35 档位 mmproj（统一模型根优先、历史根回退）。
	mmproj := managedProfileMmprojPath(llamaProfiles[0])
	cards = append(cards, LocalServiceModel{
		EngineID: "local:vision", ModelID: "qwen35-mmproj", Name: "本地识图（qwen35 mmproj）",
		Kind: "vision", Ready: mmproj != "",
		Detail: func() string {
			if mmproj == "" {
				return "未找到 mmproj 权重；识图走云端 vision 绑定"
			}
			return "随 llama 引擎 /v1/chat/completions 识图：" + filepath.ToSlash(mmproj)
		}(),
	})
	// 本地 STT：语音 V2 内嵌 sherpa 模型。
	stt := voiceNativeModelPath("model.int8.onnx")
	cards = append(cards, LocalServiceModel{
		EngineID: "local:stt", ModelID: "sherpa-zipformer-int8", Name: "本地语音识别（sherpa int8）",
		Kind: "stt", Ready: stt != "" && voiceV2FileExists(stt),
		Detail: "语音 V2 管线内嵌（sherpa-onnx 流式），非独立服务；模型 " + filepath.ToSlash(stt),
	})
	return cards
}
