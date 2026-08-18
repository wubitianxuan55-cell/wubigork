package main

import (
	"embed"
	"fmt"
	"log/slog"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"github.com/gaea/gaea/internal/app"
	"github.com/gaea/gaea/internal/httpbridge"
)

//go:embed all:dist
var assets embed.FS

//go:embed all:prompts
var promptTemplates embed.FS

func main() {
	// CLI login 模式：不启动桌面，纯终端 OAuth
	if len(os.Args) >= 2 && os.Args[1] == "login" {
		app.CLILogin()
		return
	}

	// 启动桌面应用
	application := app.New()
	application.SetDistFS(assets)
	application.SetPromptFS(promptTemplates)

	// 网页/移动端调试桥接：设置 GAEA_HTTP_PORT（如 8080）后，浏览器页面
	// 可通过 /api/rpc + /api/stream 驱动同一个 Go 内核，与桌面端完全对齐。
	// 安全（S2-2）：桥接启用一次性 token 鉴权——GAEA_HTTP_TOKEN 未设置时
	// 每次启动自动生成随机 token 并打印在日志，/api/rpc 与 /api/stream
	// 必须携带（Authorization: Bearer / X-Gaea-Token / ?token=），
	// 防止本机浏览器页面（DNS rebinding/CSRF）无授权驱动内核。
	if port := os.Getenv("GAEA_HTTP_PORT"); port != "" {
		addr := "127.0.0.1:" + port
		token := httpbridge.SessionToken(os.Getenv("GAEA_HTTP_TOKEN"))
		go func() {
			slog.Info("HTTP 调试桥接已启动（一次性 token 鉴权）",
				"addr", addr, "token", maskToken(token))
			if err := httpbridge.ServeWithToken(addr, application, token); err != nil {
				slog.Error("HTTP 调试桥接退出", "error", err)
			}
		}()
	}

	// S2-2：WebView2 远程调试默认关闭（Wails 在 disable-features 存在时会
	// 连带开启 127.0.0.1:9333 远程调试端口）。需要调试时设 GAEA_WEBVIEW_DEBUG=1
	// 再启动：开启远程调试 + 渲染器代码完整性禁用（调试用，勿在生产常开）。
	debugWebView := os.Getenv("GAEA_WEBVIEW_DEBUG") == "1"
	if debugWebView {
		slog.Warn("GAEA_WEBVIEW_DEBUG=1：WebView2 远程调试端口 127.0.0.1:9333 已开启（仅调试用）")
	}

	err := wails.Run(&options.App{
		Title:     "gaea · 多功能 AI 助手",
		Width:     1280,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
		Windows: &windows.Options{
			WebviewDisableRendererCodeIntegrity: debugWebView,
		},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  application.Startup,
		OnShutdown: application.Shutdown,
		// S2-3「App 绑定面拆分」：不再绑定单一 App 对象，改为 10 个板块门面
		// （core/office/memory/cost/model/voice/chat/novel/image/charlib），
		// 方法纯委托、逻辑零改动；前端经 gaea/lib/bridge.ts 单点路由。
		Bind: app.NewBindings(application),
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "启动失败: %v\n", err)
		os.Exit(1)
	}
}

// maskToken 日志脱敏：只保留 token 尾 4 位（如 ***abcd），避免一次性鉴权 token 明文落日志。
func maskToken(tok string) string {
	if len(tok) <= 4 {
		return "***"
	}
	return "***" + tok[len(tok)-4:]
}
