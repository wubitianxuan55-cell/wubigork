package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
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
//     只认当前服务档，绑定另一档必 404）。见 visionChatFollowRoute。
//  2. 办公识图功能绑定（模型中心「识图」卡，func_vision_*）：引擎地址 + 绑定
//     模型 + 引擎 Key，改绑即生效（聊天在云端引擎时由此接管）。
//  3. 均未命中回落 gaea.toml [vision] 静态注入（默认本机 herdsman 多模态主力）。
//
// 链路判定单源在 visionRouteRO；本方法只在动态层命中时组装运行时（含
// TouchEngine 记账——解析即记账，防识图请求进行中/刚结束就被空闲收割器停机）。
func (c *core) visionBoundRuntime() (vision.VisionRuntime, bool) {
	eng, model, _, dynamic := c.visionRouteRO()
	if !dynamic {
		return vision.VisionRuntime{}, false
	}
	return c.visionLocalRuntime(eng, model), true
}

// visionRouteRO 识图生效链只读解析（不 TouchEngine 记账——展示/轮询路径
// 专用，见 VisionRoute）。返回 (engine, model, source, dynamic)：
// dynamic=false 表示动态层未接管，回落 vision 包静态注入（engine 不可知）。
// source 细分：chat-follow（跟随聊天模型）/ chat-follow-fallback（聊天引擎
// 停机，herdsman 主力兜底）/ feature（func_vision 功能绑定）。
func (c *core) visionRouteRO() (engine, model, source string, dynamic bool) {
	if c.cfg == nil || c.engineMgr == nil {
		return "", "", "", false
	}
	if eng, m, src, ok := c.visionChatFollowRoute(); ok {
		return eng, m, src, true
	}
	if c.cfg.GetFeatureModelEnabled("vision") {
		engID, m := c.cfg.GetFeatureModel("vision")
		if engID != "" && m != "" {
			offline := c.cfg.GetOfflineMode()
			if e, ok := c.engineMgr.GetEngine(engID); ok && e.Enabled && e.BaseURL != "" && !(offline && !e.Type.IsLocal()) {
				return engID, m, "feature", true
			}
		}
	}
	return "", "", "", false
}

// VisionRoute 返回识图实际生效路由（Wails 绑定，模型中心识图卡「当前生效」
// 展示用）：JSON {"engine","model","source"}。**只读不记账**——visionLocalRuntime
// 的 TouchEngine（解析即记账）只在真实识图调用路径执行；本绑定供前端刷新/
// 轮询，若走记账路径会把 model_idle_exit_minutes 空闲收割器永远喂饱。
// source：chat-follow / chat-follow-fallback / feature / static（回落 gaea.toml
// 静态注入或内置默认，engine 不可知报空、model 报静态生效值）。
func (a *App) VisionRoute() (string, error) {
	eng, model, source, dynamic := a.visionRouteRO()
	if !dynamic {
		eng, model, source = "", vision.StaticRuntime().Model, "static"
	}
	b, err := json.Marshal(map[string]string{"engine": eng, "model": model, "source": source})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// visionChatFollowRoute 识图跟随聊天模型（只读解析，不记账）：
//   - 天璇/Strata（单模型引擎）：跟随引擎当前实际服务的档位（engines.json
//     running 条目，syncManagedCatalog 维护）；引擎未运行时若 herdsman 可用
//     （启用且默认模型判型 llm，按需加载）以 herdsman 默认模型兜底——聊天
//     此刻多半已故障转移到它。
//   - herdsman（按需加载引擎）：同引擎同模型（办公聊天绑定模型，未绑则引擎
//     默认模型）。
//
// 聊天在云端引擎 / 本地引擎全停 / 办公未绑定且活跃引擎不可知（client 未装配）
// 时不接管——回落既有 func_vision 绑定链。返回 source 细分 chat-follow /
// chat-follow-fallback（聊天引擎停机，herdsman 兜底）供展示。
func (c *core) visionChatFollowRoute() (engine, model, source string, ok bool) {
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
		return "", "", "", false
	}
	switch chatEng {
	case "tianxuan", "strata":
		e, okE := c.engineMgr.GetEngine(chatEng)
		if !okE || !e.Enabled {
			return "", "", "", false
		}
		for _, m := range e.Models {
			if m.Kind == "llm" && m.Status == "running" {
				return chatEng, m.ID, "chat-follow", true
			}
		}
		eng, m, okF := c.visionHerdsmanFallbackRoute()
		return eng, m, "chat-follow-fallback", okF
	case "herdsman":
		if chatModel == "" {
			if def, err := c.engineMgr.GetDefaultModel("herdsman"); err == nil {
				chatModel = def
			}
		}
		if chatModel == "" {
			return "", "", "", false
		}
		if e, okE := c.engineMgr.GetEngine("herdsman"); !okE || !e.Enabled {
			return "", "", "", false
		}
		return "herdsman", chatModel, "chat-follow", true
	}
	return "", "", "", false
}

// visionHerdsmanFallbackRoute herdsman 主力兜底（只读解析，不记账；聊天引擎
// 未运行时）：启用 + 默认模型判型 llm 才接管（默认模型经悬空修复不变量保证
// 为 llm 或空）。
func (c *core) visionHerdsmanFallbackRoute() (engine, model string, ok bool) {
	e, ok := c.engineMgr.GetEngine("herdsman")
	if !ok || !e.Enabled || e.BaseURL == "" {
		return "", "", false
	}
	def, err := c.engineMgr.GetDefaultModel("herdsman")
	if err != nil || def == "" || modelengine.ClassifyModelByName(def) != "llm" {
		return "", "", false
	}
	return "herdsman", def, true
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
