package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/wubigork/wubigork/internal/ai"
	"github.com/wubigork/wubigork/internal/auth"
)

// ── 登录 ──────────────────────────────────────────────────────

// configureClient 配置 AI client 的事件回调和引擎管理器（每次重建 client 后调用）
func (a *App) configureClient() {
	a.client.OnEvent = func(eventType string, data map[string]interface{}) {
		data["type"] = eventType
		a.emit("xai-output", data)
	}
	if a.engineMgr != nil {
		a.client.SetEngineManager(a.engineMgr)
		// 恢复活跃引擎设置
		if a.cfg.ActiveEngineID != "" {
			a.client.SetActiveEngine(a.cfg.ActiveEngineID)
		}
	}
}

// Login 触发 OAuth PKCE 登录流程
func (a *App) Login() (string, error) {
	result, err := auth.DoLogin(a.cfg)
	if err != nil {
		return "", fmt.Errorf("登录失败: %w", err)
	}
	// 持久化 token
	store := auth.NewTokenStore(a.cfg.TokenStorePath)
	if err := store.Save(result.Token); err != nil {
		return "", fmt.Errorf("保存 token 失败: %w", err)
	}
	// 重新初始化 client
	a.client = ai.NewClient(a.cfg)
	a.configureClient()
	// 恢复图片生成后端配置
	a.initImageBackend()
	// 更新引擎管理器中的 xAI key（用于模型列表拉取等）
	if a.engineMgr != nil {
		a.engineMgr.UpdateXAIKey(result.Token.AccessToken)
		// 后台自动刷新 xAI 模型列表
		go func() {
			if _, err := a.engineMgr.RefreshModels(context.Background(), "xai"); err != nil {
				slog.Warn("登录后刷新xAI模型列表失败", "error", err)
			}
		}()
	}
	return result.BaseURL, nil
}

// GetLoginStatus 返回是否已登录
func (a *App) GetLoginStatus() bool {
	return a.client.EnsureToken() == nil
}

// SaveToken 手动保存 token（移动端使用 — 接受完整的 token JSON 字符串）
func (a *App) SaveToken(rawJSON string) error {
	store := auth.NewTokenStore(a.cfg.TokenStorePath)
	var tok auth.Token
	if err := json.Unmarshal([]byte(rawJSON), &tok); err != nil {
		return fmt.Errorf("token 格式无效: %w", err)
	}
	tok.ObtainedAt = time.Now()
	if err := tok.Validate(); err != nil {
		return fmt.Errorf("token 无效: %w", err)
	}
	if err := store.Save(&tok); err != nil {
		return fmt.Errorf("保存 token 失败: %w", err)
	}
	// 重新初始化 client
	a.client = ai.NewClient(a.cfg)
	a.configureClient()
	a.initImageBackend()
	// 更新 xAI key
	if a.engineMgr != nil {
		a.engineMgr.UpdateXAIKey(tok.AccessToken)
	}
	return nil
}

// Logout 清除 token
func (a *App) Logout() error {
	store := auth.NewTokenStore(a.cfg.TokenStorePath)
	if err := store.Delete(); err != nil {
		return fmt.Errorf("清除 token 失败: %w", err)
	}
	a.client = ai.NewClient(a.cfg)
	a.configureClient()
	a.initImageBackend()
	return nil
}
