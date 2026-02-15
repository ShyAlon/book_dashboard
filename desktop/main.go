package main

import (
	"embed"
	"runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

func main() {
	// Create an instance of the app structure
	app := NewApp()
	appMenu := menu.NewMenu()
	if runtime.GOOS == "darwin" {
		appMenu.Append(menu.AppMenu())
		fileMenu := appMenu.AddSubmenu("File")
		fileMenu.AddText("Open Manuscript...", keys.CmdOrCtrl("o"), func(_ *menu.CallbackData) {
			app.PickAndAnalyzeFile()
		})
		fileMenu.AddSeparator()
		fileMenu.AddText("Export Log Package...", keys.CmdOrCtrl("l"), func(_ *menu.CallbackData) {
			app.ExportLogPackageDialog()
		})
		fileMenu.AddSeparator()
		fileMenu.AddText("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
			app.Quit()
		})
	}
	appMenu.Append(menu.EditMenu())
	if runtime.GOOS == "darwin" {
		appMenu.Append(menu.WindowMenu())
	}
	diagnosticsMenu := appMenu.AddSubmenu("Diagnostics")
	diagnosticsMenu.AddText("Export Log Package...", keys.CmdOrCtrl("l"), func(_ *menu.CallbackData) {
		app.ExportLogPackageDialog()
	})

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "Manuscript Health Dashboard",
		Width:     1480,
		Height:    940,
		MinWidth:  1200,
		MinHeight: 760,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 10, G: 10, B: 10, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Menu:             appMenu,
		Mac: &mac.Options{
			About: &mac.AboutInfo{
				Title:   "Manuscript Health Dashboard",
				Message: "Manuscript diagnostics for quality, structure, and AI slop signals.",
				Icon:    appIcon,
			},
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
