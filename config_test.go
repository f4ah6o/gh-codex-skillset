package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseConfig(t *testing.T) {
	cfg, err := ParseConfig(`# comment
version = 1

disabled = [
  "slides",
  "pdfs", # inline comment
]
`)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	want := Config{Version: 1, Disabled: []string{"slides", "pdfs"}}
	if !reflect.DeepEqual(cfg, want) {
		t.Fatalf("ParseConfig() = %#v, want %#v", cfg, want)
	}
}

func TestParseConfigRejectsUnknownField(t *testing.T) {
	_, err := ParseConfig("version = 1\ndisabled = []\nother = true\n")
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("ParseConfig() error = %v, want unknown field error", err)
	}
}

func TestSaveConfigIsNormalized(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".codex", "user-skills.toml")
	cfg := Config{Version: 1, Disabled: []string{"slides", "pdfs", "slides"}}
	if err := SaveConfig(path, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Count(text, `"slides"`) != 1 {
		t.Fatalf("saved config contains duplicate: %s", text)
	}
	if strings.Index(text, `"pdfs"`) > strings.Index(text, `"slides"`) {
		t.Fatalf("saved config is not sorted: %s", text)
	}
	loaded, exists, err := LoadConfig(path)
	if err != nil || !exists {
		t.Fatalf("LoadConfig() = %#v, %v, %v", loaded, exists, err)
	}
	want := []string{"pdfs", "slides"}
	if !reflect.DeepEqual(loaded.Disabled, want) {
		t.Fatalf("disabled = %#v, want %#v", loaded.Disabled, want)
	}
}
