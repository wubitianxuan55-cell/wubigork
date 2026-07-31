package app

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/types"
)

// ── 世界观 ────────────────────────────────────────────────────

// ChatWorldview 与世界观 Agent 对话（注入角色+大纲上下文，自动保存）
func (a *App) ChatWorldview(userMsg string, currentContent string) (map[string]interface{}, error) {
	if a.worldviewAgent == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	reply, err := a.worldviewAgent.ChatWithAutoSave(a.ctx, userMsg, currentContent)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"reply":     reply,
		"worldview": a.worldviewAgent.GetCurrent(),
		"sections":  a.worldviewAgent.GetSections(),
	}, nil
}

// SaveWorldviewSection 保存单个世界观维度
func (a *App) SaveWorldviewSection(sectionID, content string) error {
	if a.worldviewAgent == nil {
		return fmt.Errorf("请先打开项目")
	}
	return a.worldviewAgent.SaveSection(sectionID, content)
}

// SaveAllWorldviewSections 保存全部维度
func (a *App) SaveAllWorldviewSections(sectionsJSON string) error {
	if a.worldviewAgent == nil {
		return fmt.Errorf("请先打开项目")
	}
	var sections []types.WorldviewSection
	if err := json.Unmarshal([]byte(sectionsJSON), &sections); err != nil {
		return fmt.Errorf("解析维度数据失败: %w", err)
	}
	return a.worldviewAgent.SaveAllSections(sections)
}

// GetWorldviewSections 获取结构化世界观
func (a *App) GetWorldviewSections() (map[string]interface{}, error) {
	if a.worldviewAgent == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	wf := a.worldviewAgent.GetSections()
	if wf == nil {
		return map[string]interface{}{"sections": []types.WorldviewSection{}}, nil
	}
	return map[string]interface{}{
		"sections": wf.Sections,
	}, nil
}

// SaveWorldview 保存世界观（向后兼容）
func (a *App) SaveWorldview(content string) error {
	if a.worldviewAgent == nil {
		return fmt.Errorf("请先打开项目")
	}
	return a.worldviewAgent.Save(content)
}

// GetWorldview 获取当前世界观 markdown
func (a *App) GetWorldview() string {
	if a.worldviewAgent == nil {
		return ""
	}
	return a.worldviewAgent.GetCurrent()
}

// SaveWorldMapImage 将世界地图图片保存到项目根目录 world_map.png
func (a *App) SaveWorldMapImage(imageData string) error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}
	b64 := imageData
	if idx := strings.Index(imageData, ","); idx != -1 {
		b64 = imageData[idx+1:]
	}
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return fmt.Errorf("解码图片失败: %w", err)
	}
	return os.WriteFile(filepath.Join(pm.Dir, "world_map.png"), data, 0644)
}

// GetWorldMapImage 读取项目根目录的 world_map.png，返回 base64 data URL
func (a *App) GetWorldMapImage() string {
	pm := a.getPM()
	if pm == nil {
		return ""
	}
	path := filepath.Join(pm.Dir, "world_map.png")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	mime := "image/png"
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
}
