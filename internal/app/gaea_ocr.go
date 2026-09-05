package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/ocr"
	"github.com/gaea/gaea/internal/office/docmd"
)

// GaeaOCRText 提取图片中的文字（办公板块「提取文字」用）。
// 优先使用 Herdsman /v1/ocr（PaddleOCR），其次 /v1/documents/parse（MinerU），
// 都不可用时回退本地 OvisOCR2。
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

	if a.activeOCREngine != "" || a.activeOCRModel != "" {
		if text, err := a.herdsmanOCRWith(a.activeOCREngine, a.activeOCRModel, imagePath); err == nil && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text), nil
		}
	}
	if text, err := a.herdsmanOCR(imagePath); err == nil && strings.TrimSpace(text) != "" {
		return strings.TrimSpace(text), nil
	}
	if text, err := a.herdsmanParseImage(imagePath); err == nil && strings.TrimSpace(text) != "" {
		return strings.TrimSpace(text), nil
	}
	return docmd.OCRImageText(imagePath)
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
	mode := strings.TrimSpace(os.Getenv("HERDSMAN_PARSE_MODE"))
	if mode == "" {
		mode = "pipeline"
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
