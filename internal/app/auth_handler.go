package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/auth"
	"github.com/gaea/gaea/internal/gaea/provider/bridge"
)

// ── 登录 ──────────────────────────────────────────────────────

// configureClient 配置 AI client 的事件回调和引擎管理器（每次重建 client 后调用）
func (a *App) configureClient() {
	a.client.OnEvent = func(eventType string, data map[string]interface{}) {
		data["type"] = eventType
		a.emit("xai-output", data)
	}
	// 办公 AI 桥接 provider 始终可用：成本/知识导入 AI 解析、文件摘要等
	// 依赖 bridge 注入的 ai.LLMClient。此前只在 GaeaInit（办公引擎懒初始化）
	// 注入，用户未进过办公板块直接导入 PDF 点 AI 解析会报
	// "bridge: ai.LLMClient 未注入"。这里在每次 client 创建/重建后统一注入。
	bridge.SetClient(a.client)
	if a.engineMgr != nil {
		a.client.SetEngineManager(a.engineMgr)
		// 恢复活跃引擎设置
		if a.cfg.ActiveEngineID != "" {
			a.client.SetActiveEngine(a.cfg.ActiveEngineID)
		}
	}
}

// Login 触发 OAuth PKCE 登录流程（异步非阻塞）
//
// 由于 OAuth 流程需要等待浏览器回调（最长 5 分钟），
// 此方法立即返回，实际登录在后台 goroutine 中执行。
// 前端应轮询 GetLoginStatus() 检测登录完成，
// 或监听 "xai-login-success" / "xai-login-failed" 事件。
func (a *App) Login() error {
	slog.Info("开始 xAI OAuth 登录流程（异步）")
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("auth: OAuth login goroutine panic recovered", "panic", r)
				a.emit("xai-login-failed", map[string]interface{}{"error": "登录流程异常: " + fmt.Sprint(r)})
			}
		}()
		result, err := auth.DoLogin(a.cfg)
		if err != nil {
			slog.Error("xAI OAuth 登录失败", "error", err)
			a.emit("xai-login-failed", map[string]interface{}{"error": err.Error()})
			return
		}
		slog.Info("xAI OAuth 授权成功，保存 token")
		// 持久化 token
		store := auth.NewTokenStore(a.cfg.TokenStorePath)
		if err := store.Save(result.Token); err != nil {
			slog.Error("保存 xAI token 失败", "error", err)
			a.emit("xai-login-failed", map[string]interface{}{"error": "保存 token 失败: " + err.Error()})
			return
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
				defer func() {
					if r := recover(); r != nil {
						slog.Error("auth: refresh models goroutine panic recovered", "panic", r)
					}
				}()
				if _, err := a.engineMgr.RefreshModels(context.Background(), "xai"); err != nil {
					slog.Warn("登录后刷新xAI模型列表失败", "error", err)
				}
			}()
		}
		slog.Info("xAI 登录完成，token 已就绪", "baseURL", result.BaseURL)
		a.emit("xai-login-success", map[string]interface{}{"baseURL": result.BaseURL})
	}()
	return nil
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
