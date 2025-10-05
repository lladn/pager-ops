package main

import (
	"embed"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func GetVersion(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	// Trim spaces and newlines just in case
	version := strings.TrimSpace(string(data))
	return version, nil
}

func main() {
	// Create an instance of the app structure
	app := NewApp()

	version, err := GetVersion("VERSION")
	if err != nil {
		version = "Missing Version file"
	}

	// Create application menu with zoom support
	appMenu := menu.NewMenu()

	// Add App Menu for macOS
	appMenu.Append(menu.AppMenu())

	// Add Edit Menu (enables Cmd+C, Cmd+V, etc.)
	appMenu.Append(menu.EditMenu())

	// Add View Menu with Zoom options
	viewMenu := appMenu.AddSubmenu("View")
	viewMenu.AddText("Zoom In", keys.CmdOrCtrl("="), func(_ *menu.CallbackData) {
		app.ZoomIn()
	})
	viewMenu.AddText("Zoom Out", keys.CmdOrCtrl("-"), func(_ *menu.CallbackData) {
		app.ZoomOut()
	})
	viewMenu.AddText("Actual Size", keys.CmdOrCtrl("0"), func(_ *menu.CallbackData) {
		app.ZoomReset()
	})

	// Create application with options
	err = wails.Run(&options.App{
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
		Menu:              appMenu,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 255},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		CSSDragProperty:    "--wails-draggable",
		CSSDragValue:       "drag",
		LogLevel:           logger.INFO,
		LogLevelProduction: logger.ERROR,
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHiddenInset(),
			About: &mac.AboutInfo{
				Title: "PagerOps",
				Message: "Monitor and manage your Incidents\n \n" +
					"Copyright © 2025.9.4 Louie Ladiona \n" +
					"Version: " + version,
			},
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
