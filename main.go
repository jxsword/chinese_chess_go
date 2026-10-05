package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

// 版本号经 ldflags 注入（00 文档 §5）：
//
//	go build -ldflags "-X main.version=<tag>"
//
// Release tag 对齐 Electron 版 v1.0.0 口径。
var version = "dev"

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "中国象棋 Ultra",
		Width:     1280,
		Height:    800,
		MinWidth:  1024,
		MinHeight: 700,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		// 生命周期收口（07 §2 映射表）：关闭/退出前广播 close/before-quit 相位，
		// 驱动渲染层 GameAutoSave 离开保存后放行退出。
		OnBeforeClose: app.beforeClose,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
