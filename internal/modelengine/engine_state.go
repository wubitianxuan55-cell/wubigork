// engine_state.go 引擎状态持久化（engines.json）：stateFile 结构、
// LoadState（重启重建 custom 引擎/清洗脏数据/恢复价目）与 saveState
// （读锁快照 + 原子写盘；未设路径时静默跳过）。
package modelengine

import (
	"encoding/json"
	"log/slog"
	"os"
	"strings"

	"github.com/gaea/gaea/internal/gaea/fileutil"
)

// ── 状态持久化（engines.json）───────────────────────────────

// stateFile 磁盘状态文件结构。
type stateFile struct {
	Engines map[string]EngineConfig `json:"engines"`
}

// LoadState 从 path 加载引擎状态并设置自动落盘路径。
// 首次启动文件不存在时静默降级（保留预置默认）。
func (m *Manager) LoadState(path string) error {
	m.mu.Lock()
	m.statePath = path
	m.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var f stateFile
	if err := json.Unmarshal(data, &f); err != nil {
		slog.Warn("引擎状态文件解析失败", "path", path, "error", err)
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	for id, st := range f.Engines {
		eng, ok := m.engines[id]
		if !ok {
			// custom- 前缀条目：重启后重建（Manager 启动只种子内置引擎）。
			// 防线：Type 必须是 custom、BaseURL 必须合法（或空）——状态文件
			// 被手改成脏地址/伪 custom 条目时不采纳，沿用 v4.9.1 防线口径。
			// Key 不从此处恢复（状态文件从不存 Key），由 app 层从 config
			// custom_engine_keys 解密后经 SetCustomEngineKeys 注入。
			if strings.HasPrefix(id, customEnginePrefix) && st.Type == EngineCustom &&
				(st.BaseURL == "" || validBaseURL(st.BaseURL)) {
				eng = &EngineConfig{
					ID:      id,
					Name:    "custom",
					Type:    EngineCustom,
					Label:   st.Label,
					BaseURL: st.BaseURL,
					Enabled: st.Enabled,
					// 价目 v1：随引擎配置持久化，重启恢复（清洗归一，脏值归 nil）
					UserPriceIn:  sanitizeUserPricePtr(st.UserPriceIn),
					UserPriceOut: sanitizeUserPricePtr(st.UserPriceOut),
				}
				m.engines[id] = eng
				m.order = append(m.order, id)
			} else {
				// 未知引擎（新版本移除/手改文件）不创建
				continue
			}
		}
		// 无 http(s) 前缀的存量脏地址不采纳（保留预置默认）——v4.9.1 真机
		// 实测：API Key 被粘进地址框保存后，引擎永久报 unsupported protocol scheme。
		if st.BaseURL != "" && validBaseURL(st.BaseURL) {
			eng.BaseURL = st.BaseURL
		}
		eng.Enabled = st.Enabled
		if st.DefaultModel != "" {
			eng.DefaultModel = st.DefaultModel
		}
		if st.Models != nil {
			eng.Models = st.Models
		}
		if st.Status.LastChecked != "" {
			eng.Status = st.Status
		}
		// 价目 v1：状态文件随引擎配置持久化，重启恢复（含内置引擎手改文件的
		// 场景；nil 已由清洗归一，脏值不采纳）。
		if st.UserPriceIn != nil {
			eng.UserPriceIn = sanitizeUserPricePtr(st.UserPriceIn)
		}
		if st.UserPriceOut != nil {
			eng.UserPriceOut = sanitizeUserPricePtr(st.UserPriceOut)
		}
	}
	// 价目 v1：加载完成后全量重建用户价目注册表（持写锁内直接快照替换，
	// 避免解锁后再 Sync 的锁重入；见 user_price.go 锁序说明）。
	replaceUserPriceTable(m.snapshotUserPrices())
	return nil
}

// saveState 将当前引擎状态快照写回状态文件（path 未设置时跳过）。
// 调用方不得持有写锁；内部自行加读锁快照。
func (m *Manager) saveState() {
	m.mu.RLock()
	path := m.statePath
	if path == "" {
		m.mu.RUnlock()
		return
	}
	f := stateFile{Engines: make(map[string]EngineConfig, len(m.engines))}
	for id, e := range m.engines {
		cfg := *e
		cfg.APIKey = ""
		f.Engines[id] = cfg
	}
	m.mu.RUnlock()

	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		slog.Warn("序列化引擎状态失败", "error", err)
		return
	}
	if err := fileutil.AtomicWrite(path, data, 0644); err != nil {
		slog.Warn("保存引擎状态失败", "path", path, "error", err)
	}
}
