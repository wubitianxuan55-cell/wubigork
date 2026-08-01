// Package assistant — 虚拟助手管理器
// 管理多个虚拟助手实例，每个可绑定独立人格 + 微信 Token
package assistant

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// ─── 数据模型 ────────────────────────────────────────────────

type Assistant struct {
	ID            string `json:"id"`            // 唯一标识
	Name          string `json:"name"`          // 助手名称
	PersonalityID string `json:"personalityId"` // 绑定人格（deredere/tsundere...）
	WxToken       string `json:"wxToken"`       // 微信 ClawBot Token
	WxBotID       string `json:"wxBotId"`       // 微信 ClawBot Bot ID（ilink_bot_id，回复消息的 from_user_id）
	WxUserID      string `json:"wxUserId"`      // 绑定的微信用户 OpenID（空=不限）
	Enabled       bool   `json:"enabled"`       // 是否启用
	PortraitURL   string `json:"portraitUrl"`   // 角色剧照 URL（AI 生成，可选）
}

// ─── Manager ─────────────────────────────────────────────────

type Manager struct {
	mu         sync.RWMutex
	assistants []Assistant
	byID       map[string]*Assistant
	byWxUser   map[string]*Assistant // 微信用户 → 助手映射
	storePath  string
}

// NewEmpty 创建空助手管理器（数据文件损坏且重试仍失败时兜底，避免 nil 解引用）
func NewEmpty(dataDir string) *Manager {
	return &Manager{
		storePath: filepath.Join(dataDir, "assistants.json"),
		byID:      make(map[string]*Assistant),
		byWxUser:  make(map[string]*Assistant),
	}
}

// Load 从 JSON 文件加载助手列表
func Load(dataDir string) (*Manager, error) {
	m := NewEmpty(dataDir)

	data, err := os.ReadFile(m.storePath)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Info("[assistant] 无现有助手，创建空列表")
			return m, nil
		}
		return nil, fmt.Errorf("读取助手配置失败: %w", err)
	}

	if err := json.Unmarshal(data, &m.assistants); err != nil {
		return nil, fmt.Errorf("解析助手配置失败: %w", err)
	}

	for i := range m.assistants {
		a := &m.assistants[i]
		m.byID[a.ID] = a
		if a.WxUserID != "" {
			m.byWxUser[a.WxUserID] = a
		}
	}

	slog.Info("[assistant] 已加载助手", "count", len(m.assistants))
	return m, nil
}

// ─── 查询 ────────────────────────────────────────────────────

func (m *Manager) List() []Assistant {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Assistant, len(m.assistants))
	copy(result, m.assistants)
	return result
}

func (m *Manager) Get(id string) *Assistant {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.byID[id]
}

// FindByWxUser 根据微信用户 ID 查找绑定的助手
func (m *Manager) FindByWxUser(wxUserID string) *Assistant {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.byWxUser[wxUserID]
}

// Enabled 返回所有启用的助手
func (m *Manager) Enabled() []Assistant {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []Assistant
	for _, a := range m.assistants {
		if a.Enabled && a.WxToken != "" {
			result = append(result, a)
		}
	}
	return result
}

// ─── 修改 ────────────────────────────────────────────────────

func (m *Manager) Add(a Assistant) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if a.ID == "" {
		return fmt.Errorf("助手 ID 不能为空")
	}
	if _, exists := m.byID[a.ID]; exists {
		return fmt.Errorf("助手 %s 已存在", a.ID)
	}

	m.assistants = append(m.assistants, a)
	m.byID[a.ID] = &m.assistants[len(m.assistants)-1]
	if a.WxUserID != "" {
		m.byWxUser[a.WxUserID] = &m.assistants[len(m.assistants)-1]
	}
	return m.save()
}

func (m *Manager) Update(id string, updates Assistant) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	a, exists := m.byID[id]
	if !exists {
		return fmt.Errorf("助手 %s 不存在", id)
	}

	// 清除旧微信映射
	if a.WxUserID != "" {
		delete(m.byWxUser, a.WxUserID)
	}

	a.Name = updates.Name
	a.PersonalityID = updates.PersonalityID
	a.WxToken = updates.WxToken
	a.WxUserID = updates.WxUserID
	a.Enabled = updates.Enabled

	// 更新微信映射
	if a.WxUserID != "" {
		m.byWxUser[a.WxUserID] = a
	}

	return m.save()
}

func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	a, exists := m.byID[id]
	if !exists {
		return fmt.Errorf("助手 %s 不存在", id)
	}

	if a.WxUserID != "" {
		delete(m.byWxUser, a.WxUserID)
	}
	delete(m.byID, id)

	for i, item := range m.assistants {
		if item.ID == id {
			m.assistants = append(m.assistants[:i], m.assistants[i+1:]...)
			break
		}
	}
	return m.save()
}

// ─── 持久化 ──────────────────────────────────────────────────

func (m *Manager) save() error {
	data, err := json.MarshalIndent(m.assistants, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(m.storePath), 0755); err != nil {
		return err
	}
	return os.WriteFile(m.storePath, data, 0644)
}
