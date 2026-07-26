package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/wubigork/wubigork/internal/app"
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

	err := wails.Run(&options.App{
		Title:     "wubigork · 让灵感成为故事",
		Width:     1280,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
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
