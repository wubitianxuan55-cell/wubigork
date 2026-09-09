// engine_custom.go 自定义引擎域（A 刀：OpenAI 兼容自定义服务商）：
// 仅 custom- 前缀且 Type=custom 的引擎允许增删改，Key 只存 Manager 内存
// customKeys（落盘走 config 层密文），内置引擎受保护。
package modelengine

import (
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"unicode"

	"github.com/gaea/gaea/internal/gaea/strutil"
)

// ── 自定义引擎（A 刀：OpenAI 兼容自定义服务商）────────────────

// customEnginePrefix 自定义引擎 ID 前缀：仅该前缀且 Type=custom 的引擎允许
// 经 UpdateCustomEngine/RemoveCustomEngine 修改或删除（内置引擎受保护）。
const customEnginePrefix = "custom-"

// AddCustomEngine 创建自定义引擎并返回 engineID。
// ID 规则：custom- 前缀 + name 生成的安全小写 slug（去非法字符，空 slug 用
// "engine"），与现有 id 冲突时追加 -2、-3…。baseURL 校验 http(s) scheme +
// host 非空——云端引擎不露地址框防线（v4.9.1 Key 粘错框事故）的延伸，把
// API Key 当地址粘进来必须在此拒绝。Key 只存内存 customKeys，不进
// EngineConfig.APIKey（saveState/GetEngines 因此永不下发/落盘）。
func (m *Manager) AddCustomEngine(name, baseURL, apiKey string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("引擎名称不能为空")
	}
	u, err := validateCustomBaseURL(baseURL)
	if err != nil {
		return "", err
	}
	m.mu.Lock()
	id := m.allocCustomIDLocked(customSlug(name))
	m.engines[id] = &EngineConfig{
		ID:      id,
		Name:    "custom",
		Type:    EngineCustom,
		Label:   name,
		BaseURL: u,
		Enabled: true,
	}
	m.order = append(m.order, id)
	if m.customKeys == nil {
		m.customKeys = make(map[string]string)
	}
	m.customKeys[id] = apiKey
	m.mu.Unlock()
	m.saveState()
	slog.Info("自定义引擎已创建", "engine", id)
	return id, nil
}

// UpdateCustomEngine 更新自定义引擎（engineID 必须是 custom- 前缀的自定义引擎）。
// apiKey 传空串 = 保留原 Key 不变（前端不回显 Key，编辑时留空即「不改」）。
func (m *Manager) UpdateCustomEngine(engineID, name, baseURL, apiKey string) error {
	if !strings.HasPrefix(engineID, customEnginePrefix) {
		return fmt.Errorf("引擎 %s 不是自定义引擎，禁止修改", engineID)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("引擎名称不能为空")
	}
	u, err := validateCustomBaseURL(baseURL)
	if err != nil {
		return err
	}
	m.mu.Lock()
	eng, ok := m.engines[engineID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("引擎 %s 不存在", engineID)
	}
	if eng.Type != EngineCustom {
		m.mu.Unlock()
		return fmt.Errorf("引擎 %s 不是自定义引擎，禁止修改", engineID)
	}
	eng.Label = name
	eng.BaseURL = u
	if apiKey != "" {
		if m.customKeys == nil {
			m.customKeys = make(map[string]string)
		}
		m.customKeys[engineID] = apiKey
	}
	m.mu.Unlock()
	m.saveState()
	slog.Info("自定义引擎已更新", "engine", engineID)
	return nil
}

// RemoveCustomEngine 删除自定义引擎（仅允许 custom- 前缀且 Type=custom），
// 一并清掉内存 Key 并从展示顺序中摘除。
func (m *Manager) RemoveCustomEngine(engineID string) error {
	if !strings.HasPrefix(engineID, customEnginePrefix) {
		return fmt.Errorf("引擎 %s 不是自定义引擎，禁止删除", engineID)
	}
	m.mu.Lock()
	eng, ok := m.engines[engineID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("引擎 %s 不存在", engineID)
	}
	if eng.Type != EngineCustom {
		m.mu.Unlock()
		return fmt.Errorf("引擎 %s 不是自定义引擎，禁止删除", engineID)
	}
	delete(m.engines, engineID)
	for i, id := range m.order {
		if id == engineID {
			m.order = append(m.order[:i], m.order[i+1:]...)
			break
		}
	}
	delete(m.customKeys, engineID)
	// 健康巡检连续失败计数一并清理（C 刀 v0）
	delete(m.probeFails, engineID)
	m.mu.Unlock()
	// 价目 v1：删除引擎后同步清除注册表旧条目（防止已删引擎继续按旧价计费）
	m.SyncUserPrices()
	m.saveState()
	slog.Info("自定义引擎已删除", "engine", engineID)
	return nil
}

// SetCustomEngineKeys 批量注入自定义引擎 Key（【解密后的明文】，仅存内存；
// app 层启动时从 config custom_engine_keys 解密后调用）。传 nil/空 map = 清空。
func (m *Manager) SetCustomEngineKeys(keys map[string]string) {
	cp := make(map[string]string, len(keys))
	for id, k := range keys {
		cp[id] = k
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.customKeys = cp
}

// CustomEngineKey 返回自定义引擎 Key（明文，仅内存）；未配置返回空串。
// 消费口径与 GLMKey 一致：Key 存 manager 而非 EngineConfig.APIKey。
func (m *Manager) CustomEngineKey(engineID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.customKeys[engineID]
}

// CustomEngineKeys 返回全部自定义引擎 Key 副本（明文；app 层加密后落
// config custom_engine_keys）。
func (m *Manager) CustomEngineKeys() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]string, len(m.customKeys))
	for id, k := range m.customKeys {
		out[id] = k
	}
	return out
}

// allocCustomIDLocked 生成 custom- 前缀 ID：与现有 id（内置或 custom）冲突时
// 追加 -2、-3…（调用方需持写锁）。
func (m *Manager) allocCustomIDLocked(base string) string {
	id := customEnginePrefix + base
	for n := 2; ; n++ {
		if _, exists := m.engines[id]; !exists {
			return id
		}
		id = fmt.Sprintf("%s%s-%d", customEnginePrefix, base, n)
	}
}

// customSlug 由 name 生成安全小写 slug：小写化后仅保留字母/数字/连字符，
// 空白折叠为 '-'（其余非法字符——中文/符号等——剔除），压缩连续 '-'、
// 修剪首尾，空结果回退 "engine"；超 40 字符截断（同样修剪尾部连字符）。
func customSlug(name string) string {
	// Engine IDs intentionally stay ASCII. First retain the old alphabet
	// policy, then let the canonical TitleSlug do folding/trimming/fallback.
	var b strings.Builder
	hasAlnum := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case unicode.IsSpace(r):
			b.WriteRune('-')
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-':
			b.WriteRune(r)
			if r != '-' {
				hasAlnum = true
			}
		}
	}
	if !hasAlnum {
		return "engine"
	}
	out := strutil.TitleSlug(b.String())
	if len(out) > 40 {
		out = strings.Trim(out[:40], "-")
	}
	return out
}

// validateCustomBaseURL 自定义引擎地址校验：url.Parse 可解析 + scheme 为
// http/https + host 非空。v4.9.1「Key 粘错框」防线的延伸——把 API Key 当
// 地址粘进来（无 scheme/host）必须在此拒绝，不接受仅前缀判断的宽松口径。
func validateCustomBaseURL(u string) (string, error) {
	u = strings.TrimSpace(u)
	parsed, err := url.Parse(u)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", fmt.Errorf("引擎地址无效：必须是 http:// 或 https:// 开头的完整地址（请勿把 API Key 粘进地址框）")
	}
	return u, nil
}
