package app

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/gaea/gaea/internal/docmd"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/ocr"
)

// GaeaOCRText 提取图片中的文字（办公板块「提取文字」用）。
// 优先使用 Herdsman /v1/ocr（PaddleOCR），其次 /v1/documents/parse（MinerU），
// 都不可用时回退本地 OvisOCR2。
//
// 四腿编排（审计 U54/IN3-09：本函数是远端 internal/ocr 与本地 docmd 两链的
// 唯一汇合点——是编排不是第三套 OCR 客户端，自身无 HTTP 代码。腿序与配置源
// 已由 gaea_ocr_chain_test.go 钉死；把前腿折进 docmd seam 会改道本地-only
// 消费方，勿动，理由见 docmd/ocr.go seam 注释的口径边界段）：
//   - 腿 1 指定引擎：activeOCREngine/activeOCRModel（模型中心「设为 OCR」，
//     config active_ocr_engine/active_ocr_model）→ herdsmanOCRWith，模型名
//     含 mineru/parse 走 /v1/documents/parse，否则 /v1/ocr；
//   - 腿 2 herdsman /v1/ocr，模型取 HERDSMAN_OCR_MODEL 或引擎 OCR 模型；
//   - 腿 3 herdsman /v1/documents/parse（HERDSMAN_PARSE_MODEL/_MODE）；
//   - 腿 4 docmd.OCRImageText 本地链，引擎顺序归 GAEA_OCR_ENGINE 管
//     （auto=OvisOCR2→tesseract）——两链配置口径互不越界。
//
// 腿间「错误或空文本即穿透」+ IN3-08 逐腿错误收集（errors.Join）。
func (a *App) GaeaOCRText(imagePath string) (string, error) {
	// T0 图像域试点：识图-读 先经域能力注册表校验（可用性恒定，行为不变）。
	if _, err := imageDomainEntry(CapabilityVisionRead); err != nil {
		return "", err
	}
	if imagePath == "" {
		return "", fmt.Errorf("缺少图片路径")
	}
	if _, err := os.Stat(imagePath); err != nil {
		return "", fmt.Errorf("图片不存在：%s", imagePath)
	}

	// 审计 P1 IN3-08：四层降级链逐层收集真实错误——此前只判 err==nil 且从不
	// 保留 err，全部不可用时只回最后一层的错误，排障看不到前三层为何失败。
	var errs []error
	if a.activeOCREngine != "" || a.activeOCRModel != "" {
		if text, err := a.herdsmanOCRWith(a.activeOCREngine, a.activeOCRModel, imagePath); err == nil && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text), nil
		} else if err != nil {
			errs = append(errs, fmt.Errorf("指定引擎(%s/%s): %w", a.activeOCREngine, a.activeOCRModel, err))
		}
	}
	if text, err := a.herdsmanOCR(imagePath); err == nil && strings.TrimSpace(text) != "" {
		return strings.TrimSpace(text), nil
	} else if err != nil {
		errs = append(errs, fmt.Errorf("herdsman /v1/ocr: %w", err))
	}
	if text, err := a.herdsmanParseImage(imagePath); err == nil && strings.TrimSpace(text) != "" {
		return strings.TrimSpace(text), nil
	} else if err != nil {
		errs = append(errs, fmt.Errorf("herdsman parse: %w", err))
	}
	text, err := docmd.OCRImageText(imagePath)
	if err != nil {
		errs = append(errs, fmt.Errorf("本地 OCR: %w", err))
		return "", errors.Join(errs...)
	}
	return text, nil
}

// herdsmanOCRWith 使用指定引擎和模型识别图片。
// 模型名含 mineru 时走 /v1/documents/parse，否则走 /v1/ocr。
func (a *App) herdsmanOCRWith(engineID, modelID, imagePath string) (string, error) {
	if a.engineMgr == nil {
		return "", fmt.Errorf("引擎管理器未初始化")
	}
	if engineID == "" {
		engineID = "herdsman"
	}
	eng, ok := a.engineMgr.GetEngine(engineID)
	if !ok || !eng.Enabled {
		return "", fmt.Errorf("OCR 引擎未启用: %s", engineID)
	}

	model := strings.TrimSpace(modelID)
	if model == "" {
		var found bool
		model, found = pickHerdsmanModel(eng.Models, "ocr")
		if !found {
			return "", fmt.Errorf("引擎 %s 没有 OCR 模型", engineID)
		}
	}

	lower := strings.ToLower(model)
	if strings.Contains(lower, "mineru") || strings.Contains(lower, "parse") {
		return a.parseImageWithEngine(eng, model, imagePath)
	}

	client := ocr.New(eng.BaseURL, model)
	result, err := client.RecognizeImageFile(imagePath)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

// herdsmanOCR 调用 Herdsman /v1/ocr；引擎未启用或模型不可用时返回错误。
func (a *App) herdsmanOCR(imagePath string) (string, error) {
	if a.engineMgr == nil {
		return "", fmt.Errorf("引擎管理器未初始化")
	}
	eng, ok := a.engineMgr.GetEngine("herdsman")
	if !ok || !eng.Enabled {
		return "", fmt.Errorf("Herdsman 引擎未启用")
	}
	model := os.Getenv("HERDSMAN_OCR_MODEL")
	if strings.TrimSpace(model) == "" {
		var found bool
		model, found = pickHerdsmanModel(eng.Models, "ocr")
		if !found {
			return "", fmt.Errorf("Herdsman 模型列表中没有 OCR 模型")
		}
	}
	client := ocr.New(eng.BaseURL, model)
	result, err := client.RecognizeImageFile(imagePath)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

// herdsmanParseImage 调用 Herdsman /v1/documents/parse 处理图片（MinerU）。
func (a *App) herdsmanParseImage(imagePath string) (string, error) {
	if a.engineMgr == nil {
		return "", fmt.Errorf("引擎管理器未初始化")
	}
	eng, ok := a.engineMgr.GetEngine("herdsman")
	if !ok || !eng.Enabled {
		return "", fmt.Errorf("Herdsman 引擎未启用")
	}
	model := os.Getenv("HERDSMAN_PARSE_MODEL")
	if strings.TrimSpace(model) == "" {
		var found bool
		model, found = pickHerdsmanModel(eng.Models, "parse")
		if !found {
			return "", fmt.Errorf("Herdsman 模型列表中没有文档解析模型")
		}
	}
	return a.parseImageWithEngine(eng, model, imagePath)
}

func (a *App) parseImageWithEngine(eng *modelengine.EngineConfig, model, imagePath string) (string, error) {
	mode := strings.TrimSpace(os.Getenv("HERDSMAN_PARSE_MODE"))
	if mode == "" {
		mode = "pipeline"
	}
	client := ocr.New(eng.BaseURL, model)
	result, err := client.ParseDocument(ocr.ParseOptions{
		Model:  model,
		Path:   imagePath,
		Mode:   mode,
		Format: "json",
	})
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(result.Text)
	if text == "" {
		text = strings.TrimSpace(result.Markdown)
	}
	return text, nil
}

// pickHerdsmanModel 从引擎模型列表中挑出最匹配能力标签的模型。
func pickHerdsmanModel(models []modelengine.ModelInfo, capability string) (string, bool) {
	for _, m := range models {
		l := strings.ToLower(m.ID)
		switch capability {
		case "ocr":
			if strings.Contains(l, "paddleocr") || strings.Contains(l, "ocr") {
				return m.ID, true
			}
		case "parse":
			if strings.Contains(l, "mineru") {
				return m.ID, true
			}
		}
	}
	return "", false
}
