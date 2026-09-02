// Package assistant — 虚拟助手管理器
// 管理多个虚拟助手实例，每个可绑定独立人格 + 微信 Token
package assistant

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gaea/gaea/internal/gaea/secure"
	"github.com/gaea/gaea/internal/whisper"
)

// wxTokenPrefix 与 internal/gaea/secure 的 "dpapi:" 前缀一致（同 auth/token.go 的迁移兼容模式）。
const wxTokenPrefix = "dpapi:"

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

	// 自定义人格（小说角色导入等，覆盖预设人格；空则用 PersonalityID 预设）
	VoiceGuide string                  `json:"voiceGuide,omitempty"` // 人格口吻设定（角色性格）
	Gender     string                  `json:"gender,omitempty"`     // male/female/neutral
	Tags       []string                `json:"tags,omitempty"`       // 角色标签
	Dims       whisper.PersonalityDims `json:"dims,omitempty"`       // 五维（T/I/S/O/R）
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

	// T6-9.2 凭据解密：带 "dpapi:" 前缀的 wxToken 解密还原为内存明文；
	// 无前缀的旧版明文读取成功并触发一次自动重写为加密（一次性迁移）。
	// 解密失败返回明确错误（含助手 ID），绝不静默清空 token。
	migrated := false
	for i := range m.assistants {
		a := &m.assistants[i]
		if a.WxToken == "" {
			continue
		}
		if !strings.HasPrefix(a.WxToken, wxTokenPrefix) {
			migrated = true // 旧版明文：读取成功，标记迁移
			continue
		}
		dec, err := secure.DecryptString(a.WxToken)
		if err != nil {
			return nil, fmt.Errorf("解密助手 %s 的 wxToken 失败: %w", a.ID, err)
		}
		a.WxToken = dec
	}
	if migrated {
		if err := m.save(); err != nil {
			return nil, fmt.Errorf("迁移旧明文 wxToken 为加密存储失败: %w", err)
		}
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

// FindByPersonality 按人格 ID 查找助手（支持自定义人格角色）。
// 返回值副本，避免锁外使用 slice 元素指针与 Update 并发写产生 data race。
func (m *Manager) FindByPersonality(personalityID string) (Assistant, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for i := range m.assistants {
		if m.assistants[i].PersonalityID == personalityID {
			return m.assistants[i], true
		}
	}
	return Assistant{}, false
}

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

	// 约束：扩展字段（WxBotID/PortraitURL/VoiceGuide/Gender/Tags/Dims）非空/非 nil 才写回，
	// 空=保留现值——防止前端部分保存（表单未携带这些字段）把已有数据清空。
	if updates.WxBotID != "" {
		a.WxBotID = updates.WxBotID
	}
	if updates.PortraitURL != "" {
		a.PortraitURL = updates.PortraitURL
	}
	if updates.VoiceGuide != "" {
		a.VoiceGuide = updates.VoiceGuide
	}
	if updates.Gender != "" {
		a.Gender = updates.Gender
	}
	if updates.Tags != nil {
		a.Tags = updates.Tags
	}
	if updates.Dims != (whisper.PersonalityDims{}) {
		a.Dims = updates.Dims
	}

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
	// 落盘加密决策（T6-9.2）：内存态保持明文——现有调用方、List/WhisperAssistantList
	// 回显明文、改动面最小；仅在此处写盘前把 WxToken 加密为 "dpapi:" 前缀（与
	// auth/token.go 同模式）。加密失败返回错误，绝不静默降级为明文落盘。
	disk := make([]Assistant, len(m.assistants))
	copy(disk, m.assistants)
	for i := range disk {
		tok := disk[i].WxToken
		if tok == "" || strings.HasPrefix(tok, wxTokenPrefix) {
			continue // 空 token 不加密；已带前缀（防御）不再重复加密
		}
		enc, err := secure.EncryptString(tok)
		if err != nil {
			return fmt.Errorf("加密助手 %s 的 wxToken 失败: %w", disk[i].ID, err)
		}
		disk[i].WxToken = enc
	}
	data, err := json.MarshalIndent(disk, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(m.storePath), 0755); err != nil {
		return err
	}
	return os.WriteFile(m.storePath, data, 0644)
}
