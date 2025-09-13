package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:             "PagerOps",
		Width:             600,
		Height:            800,
		MinWidth:          300,
		MinHeight:         300,
		DisableResize:     false,
		Frameless:         false,
		StartHidden:       false,
		HideWindowOnClose: false,
		AlwaysOnTop:       false,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 255},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		// CSSDragProperty and CSSDragValue for custom drag handling
		CSSDragProperty:    "--wails-draggable",
		CSSDragValue:       "drag",
		LogLevel:           logger.INFO,
		LogLevelProduction: logger.ERROR,
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHiddenInset(),
			About: &mac.AboutInfo{
				Title:   "PagerOps",
				Message: "Monitor and manage your Incidents\n \n Copyright © 2025.9.4 Louie Ladiona \n Version: 1.0.0-beta.4",
			},
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
