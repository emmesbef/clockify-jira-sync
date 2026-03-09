package main

import (
	"embed"
	"log"
	"runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"

	"clockify-jira-sync/internal/app"
	"clockify-jira-sync/internal/config"
	"clockify-jira-sync/internal/mockserver"
)

// version is set at build time via -ldflags "-X main.version=..."
var version = "dev"

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/tray-icon.png
var trayIcon []byte

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Config warning: %v (app will start but API calls will fail)", err)
		cfg = &config.Config{}
	} else {
		// Ensure credentials are persisted to the config dir .env.
		// This migrates credentials that come from a local .env or shell env vars.
		if created, saveErr := config.EnsurePersisted(cfg); saveErr != nil {
			log.Printf("Config persistence warning: %v", saveErr)
		} else if created {
			p, _ := config.FilePath()
			log.Printf("Credentials migrated to %s", p)
		}
	}

	// Create main application
	application := app.NewApp(cfg, version)

	// Inject mock server if mock mode is on
	if cfg.MockMode {
		log.Println("Starting mock data server (MOCK_DATA=true)")
		mockSrv := mockserver.Start()
		application.SetMockMode(mockSrv.URL)
	}

	// Initialize tray (macOS only — uses dispatch_async so safe to call before run loop)
	if runtime.GOOS == "darwin" {
		application.InitTray(version, trayIcon)
	}

	// Start Wails
	err = wails.Run(&options.App{
		Title:             "Clockify ↔ Jira Time Sync",
		Width:             420,
		Height:            720,
		MinWidth:          380,
		MinHeight:         600,
		DisableResize:     false,
		Frameless:         false,
		StartHidden:       false,
		HideWindowOnClose: true,
		BackgroundColour:  &options.RGBA{R: 15, G: 15, B: 20, A: 1},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  application.Startup,
		OnDomReady: application.DomReady,
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
				Title:   "Clockify ↔ Jira Time Sync v" + version,
				Message: "A desktop time tracker that syncs between Clockify and Jira.\n\n© 2026 Fabian Emmesberger\nEmail: info@level-87.dev\nLicense: MIT Licensed",
			},
		},
	})

	if err != nil {
		log.Fatal("Error:", err)
	}
}
