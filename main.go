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

func main() {
	// CLI login 模式：不启动桌面，纯终端 OAuth
	if len(os.Args) >= 2 && os.Args[1] == "login" {
		app.CLILogin()
		return
	}

	// 启动桌面应用
	application := app.New()
	application.SetDistFS(assets)

	// 网页/移动端调试桥接：设置 GAEA_HTTP_PORT（如 8080）后，浏览器页面
	// 可通过 /api/rpc + /api/stream 驱动同一个 Go 内核，与桌面端完全对齐。
	if port := os.Getenv("GAEA_HTTP_PORT"); port != "" {
		addr := "127.0.0.1:" + port
		go func() {
			slog.Info("HTTP 调试桥接已启动", "addr", addr)
			if err := httpbridge.Serve(addr, application); err != nil {
				slog.Error("HTTP 调试桥接退出", "error", err)
			}
		}()
	}

	err := wails.Run(&options.App{
		Title:     "gaea · 多功能 AI 助手",
		Width:     1280,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
		// 启用 WebView2 渲染进程远程调试（Wails 会开 127.0.0.1:9333）：
		// 卡死时可抓取渲染现场（截图/控制台）。本地桌面应用，可接受。
		Windows: &windows.Options{
			WebviewDisableRendererCodeIntegrity: true,
		},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  application.Startup,
		OnShutdown: application.Shutdown,
		Bind: []interface{}{
			application,
		},
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "启动失败: %v\n", err)
		os.Exit(1)
	}
}
