package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"

	"clockify-jira-sync/internal/app"
	"clockify-jira-sync/internal/config"
	"clockify-jira-sync/internal/mockserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Config warning: %v (app will start but API calls will fail)", err)
		cfg = &config.Config{}
	}

	// Create main application
	application := app.NewApp(cfg)

	// Inject mock server if mock mode is on
	if cfg.MockMode {
		log.Println("Starting mock data server (MOCK_DATA=true)")
		mockSrv := mockserver.Start()
		application.SetMockMode(mockSrv.URL)
	}

	// Start Wails
	err = wails.Run(&options.App{
		Title:            "Clockify ↔ Jira Time Sync",
		Width:            420,
		Height:           720,
		MinWidth:         380,
		MinHeight:        600,
		DisableResize:    false,
		Frameless:        false,
		StartHidden:      false,
		BackgroundColour: &options.RGBA{R: 15, G: 15, B: 20, A: 1},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  application.Startup,
		OnShutdown: application.Shutdown,
		Bind: []interface{}{
			application,
		},
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:                  true,
				HideTitleBar:               false,
				FullSizeContent:            true,
				UseToolbar:                 false,
			},
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			About: &mac.AboutInfo{
				Title:   "Clockify ↔ Jira Time Sync",
				Message: "A desktop time tracker that syncs between Clockify and Jira.\n\n© 2026 Fabian Emmesberger\nEmail: info@level-87.dev\nLicense: MIT Licensed",
			},
		},
	})

	if err != nil {
		log.Fatal("Error:", err)
	}
}
