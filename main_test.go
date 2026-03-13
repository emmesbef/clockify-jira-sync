package main

import (
	"encoding/json"
	"testing"
)

func embeddedProductVersion(t *testing.T) string {
	t.Helper()

	var cfg struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}
	if err := json.Unmarshal(wailsConfigJSON, &cfg); err != nil {
		t.Fatalf("unmarshal embedded wails.json: %v", err)
	}
	if cfg.Info.ProductVersion == "" {
		t.Fatal("embedded wails.json has empty info.productVersion")
	}

	return cfg.Info.ProductVersion
}

func TestResolveVersionFallsBackToWailsProductVersion(t *testing.T) {
	original := version
	version = "dev"
	t.Cleanup(func() {
		version = original
	})

	want := embeddedProductVersion(t)
	got := resolveVersion()
	if got != want {
		t.Fatalf("resolveVersion() = %q, want %q", got, want)
	}
}

func TestResolveVersionUsesBuildTimeInjectedVersion(t *testing.T) {
	original := version
	version = "9.9.9-test"
	t.Cleanup(func() {
		version = original
	})

	got := resolveVersion()
	if got != "9.9.9-test" {
		t.Fatalf("resolveVersion() = %q, want %q", got, "9.9.9-test")
	}
}
