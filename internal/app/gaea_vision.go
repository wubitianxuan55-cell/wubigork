package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"image/png"
	"strings"

	"github.com/gaea/gaea/internal/gaea/vision"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/screen"
)

// visionRecognize 可注入 seam（测试替换；生产恒为 vision.RecognizeImage）。
var visionRecognize = vision.RecognizeImage

// visionBoundRuntime 办公识图后端解析（经 vision.SetVisionRuntimeResolver 挂为
// 动态钩子，每次识图调用执行；只读内存——配置 + 引擎表，无 IO）。解析顺序：
//
//  1. 跟随聊天模型（2026-10-19 用户拍板：聊天用的本地模型本身多模态时，识图
//     不再另找第二个识图模型——多载一份权重浪费 UMA，单模型引擎上天璇/Strata
//     只认当前服务档，绑定另一档必 404）。见 visionChatFollowRuntime。
//  2. 办公识图功能绑定（模型中心「识图」卡，func_vision_*）：引擎地址 + 绑定
//     模型 + 引擎 Key，改绑即生效（聊天在云端引擎时由此接管）。
//  3. 均未命中回落 gaea.toml [vision] 静态注入（默认本机 herdsman 多模态主力）。
func (c *core) visionBoundRuntime() (vision.VisionRuntime, bool) {
	if c.cfg == nil || c.engineMgr == nil {
		return vision.VisionRuntime{}, false
	}
	if r, ok := c.visionChatFollowRuntime(); ok {
		return r, true
	}
	if !c.cfg.GetFeatureModelEnabled("vision") {
		return vision.VisionRuntime{}, false
	}
	engID, model := c.cfg.GetFeatureModel("vision")
	if engID == "" || model == "" {
		return vision.VisionRuntime{}, false
	}
	offline := c.cfg.GetOfflineMode()
	e, ok := c.engineMgr.GetEngine(engID)
	if !ok || !e.Enabled || e.BaseURL == "" || (offline && !e.Type.IsLocal()) {
		return vision.VisionRuntime{}, false
	}
	return c.visionLocalRuntime(engID, model), true
}

// visionChatFollowRuntime 识图跟随聊天模型：仅当聊天实际用的引擎是本地引擎
// 且能就地取到多模态主力时接管（本机天璇/Strata 档位 serve 配置全部带
// vision、herdsman 目录标 uncensored 家族 type=multimodal——2026-10-19 核实）。
//   - 天璇/Strata（单模型引擎）：跟随引擎当前实际服务的档位（engines.json
//     running 条目，syncManagedCatalog 维护）；引擎未运行时若 herdsman 可用
//     （启用且默认模型判型 llm，按需加载）以 herdsman 默认模型兜底——聊天
//     此刻多半已故障转移到它。
//   - herdsman（按需加载引擎）：同引擎同模型（办公聊天绑定模型，未绑则引擎
//     默认模型）。
//
// 聊天在云端引擎 / 本地引擎全停 / 办公未绑定且活跃引擎不可知（client 未装配）
// 时不接管——回落既有 func_vision 绑定链。
func (c *core) visionChatFollowRuntime() (vision.VisionRuntime, bool) {
	chatEng, chatModel := "", ""
	if c.cfg.GetFeatureModelEnabled("gaea") {
		chatEng, chatModel = c.cfg.GetFeatureModel("gaea")
	}
	if chatEng == "" && c.client != nil {
		chatEng = c.client.ActiveEngineID()
		if def, err := c.engineMgr.GetDefaultModel(chatEng); err == nil {
			chatModel = def
		}
	}
	if chatEng == "" {
		return vision.VisionRuntime{}, false
	}
	switch chatEng {
	case "tianxuan", "strata":
		e, ok := c.engineMgr.GetEngine(chatEng)
		if !ok || !e.Enabled {
			return vision.VisionRuntime{}, false
		}
		for _, m := range e.Models {
			if m.Kind == "llm" && m.Status == "running" {
				return c.visionLocalRuntime(chatEng, m.ID), true
			}
		}
		return c.visionHerdsmanFallback()
	case "herdsman":
		if chatModel == "" {
			if def, err := c.engineMgr.GetDefaultModel("herdsman"); err == nil {
				chatModel = def
			}
		}
		if chatModel == "" {
			return vision.VisionRuntime{}, false
		}
		if e, ok := c.engineMgr.GetEngine("herdsman"); !ok || !e.Enabled {
			return vision.VisionRuntime{}, false
		}
		return c.visionLocalRuntime("herdsman", chatModel), true
	}
	return vision.VisionRuntime{}, false
}

// visionHerdsmanFallback herdsman 主力兜底（聊天引擎未运行时）：启用 + 默认
// 模型判型 llm 才接管（默认模型经悬空修复不变量保证为 llm 或空）。
func (c *core) visionHerdsmanFallback() (vision.VisionRuntime, bool) {
	e, ok := c.engineMgr.GetEngine("herdsman")
	if !ok || !e.Enabled || e.BaseURL == "" {
		return vision.VisionRuntime{}, false
	}
	def, err := c.engineMgr.GetDefaultModel("herdsman")
	if err != nil || def == "" || modelengine.ClassifyModelByName(def) != "llm" {
		return vision.VisionRuntime{}, false
	}
	return c.visionLocalRuntime("herdsman", def), true
}

// visionLocalRuntime 组装本地引擎的视觉运行时（OpenAI 兼容端点 + 引擎 Key，
// 本地引擎 Key 为空=免鉴权）。识图也是「真实使用」：解析即记账，防识图请求
// 进行中/刚结束就被空闲收割器停机。
func (c *core) visionLocalRuntime(engineID, model string) vision.VisionRuntime {
	c.engineMgr.TouchEngine(engineID)
	e, _ := c.engineMgr.GetEngine(engineID)
	return vision.VisionRuntime{
		Kind:    vision.VisionProviderKindOpenAI,
		BaseURL: e.BaseURL,
		Model:   model,
		APIKey:  c.engineMgr.EngineAPIKey(engineID),
	}
}

// visionOCRTextPrompt 微信识图提示词：完整解读——文字按版式逐行提取优先
// （保持原文），画面主体/对象/布局/图表要点随后；两者都要（用户发图后常
// 用下一条文本追问，注入描述必须足以支撑追问，纯文字提取会丢视觉语境）。
const visionOCRTextPrompt = "请完整解读这张图片：先按原版式逐行提取所有可见文字（保持原文，不翻译不改写）；再简述画面主体、对象、布局与图表要点（如有）。"

// visionOCRText 微信识图（v4.8.3）：优先多模态主模型（vision 运行时，默认
// 本机 Qwen3.6-35B——真机探针实测视觉链路完好），手写体/低质量图显著强于
// PaddleOCR 专线；主模型失败（服务忙/冷启动超时等）回退 GaeaOCRText
// （PaddleOCR → MinerU → OvisOCR2 三级既有链）。
// 注意：本函数是「看懂图」不是纯 OCR——文字提取+画面理解都要；纯文字提取
// 场景（办公提取文字工具）继续走 GaeaOCRText，两者分岗。
func (a *App) visionOCRText(imagePath string) (string, error) {
	text, err := visionRecognize(context.Background(), imagePath, visionOCRTextPrompt)
	if err == nil && strings.TrimSpace(text) != "" {
		return strings.TrimSpace(text), nil
	}
	return a.GaeaOCRText(imagePath)
}

// GaeaCaptureScreen 捕获整个屏幕，返回 PNG data URL（办公板块截图用）。
func (a *App) GaeaCaptureScreen() (string, error) {
	img, err := screen.Capture()
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// GaeaRecognizeImage 用本地视觉模型识别图片，返回文本描述（办公板块识图用）。
func (a *App) GaeaRecognizeImage(imagePath, prompt string) (string, error) {
	// T0 图像域试点：识图-懂 先经域能力注册表校验（可用性恒定，行为不变）。
	if _, err := imageDomainEntry(CapabilityVisionUnderstand); err != nil {
		return "", err
	}
	return vision.RecognizeImage(context.Background(), imagePath, prompt)
}
