package appconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigFromFileReadsQuoteOverseasSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fundlive.yaml")
	if err := os.WriteFile(path, []byte(`
quote:
  default_source: sina
  overseas_source: tencent
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfigFromFile(path)
	if err != nil {
		t.Fatalf("LoadConfigFromFile() error = %v", err)
	}
	if cfg.Quote.DefaultSource != "sina" {
		t.Fatalf("DefaultSource = %q, want sina", cfg.Quote.DefaultSource)
	}
	if cfg.Quote.OverseasSource != "tencent" {
		t.Fatalf("OverseasSource = %q, want tencent", cfg.Quote.OverseasSource)
	}
}
