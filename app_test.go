package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDisableThenRunDryRun(t *testing.T) {
	repo := initTestRepo(t)
	home := t.TempDir()
	createSkill(t, filepath.Join(home, ".agents", "skills"), "pdfs")
	createSkill(t, filepath.Join(home, ".agents", "skills"), "slides")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(strings.NewReader(""), &stdout, &stderr)
	app.Getwd = func() (string, error) { return repo, nil }
	app.UserHome = func() (string, error) { return home, nil }

	if err := app.Run([]string{"disable", "slides", "pdfs"}); err != nil {
		t.Fatalf("disable error = %v", err)
	}
	stdout.Reset()
	if err := app.Run([]string{"run", "--dry-run", "--", "exec", "review this"}); err != nil {
		t.Fatalf("run --dry-run error = %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "skills.config=") || !strings.Contains(output, "exec") {
		t.Fatalf("dry-run output = %q", output)
	}
	if strings.Index(output, "pdfs") > strings.Index(output, "slides") {
		t.Fatalf("skills are not sorted in dry-run output: %q", output)
	}

	cfg, exists, err := LoadConfig(filepath.Join(repo, filepath.FromSlash(configRelativePath)))
	if err != nil || !exists {
		t.Fatalf("LoadConfig() = %#v, %v, %v", cfg, exists, err)
	}
	if strings.Join(cfg.Disabled, ",") != "pdfs,slides" {
		t.Fatalf("disabled = %#v", cfg.Disabled)
	}
}

func TestSetEnabledIsAtomicOnMissingSkill(t *testing.T) {
	repo := initTestRepo(t)
	home := t.TempDir()
	createSkill(t, filepath.Join(home, ".agents", "skills"), "pdfs")

	var stdout bytes.Buffer
	app := NewApp(strings.NewReader(""), &stdout, &bytes.Buffer{})
	app.Getwd = func() (string, error) { return repo, nil }
	app.UserHome = func() (string, error) { return home, nil }

	err := app.Run([]string{"disable", "pdfs", "missing"})
	if err == nil {
		t.Fatal("disable succeeded, want error")
	}
	_, exists, loadErr := LoadConfig(filepath.Join(repo, filepath.FromSlash(configRelativePath)))
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if exists {
		t.Fatal("config was created despite validation failure")
	}
}

func initTestRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	cmd := exec.Command("git", "init", "-q", repo)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	absolute, err := filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(absolute, 0o755); err != nil {
		t.Fatal(err)
	}
	return absolute
}

func TestInitShortHelpDoesNotCreateConfig(t *testing.T) {
	repo := initTestRepo(t)
	var stdout bytes.Buffer
	app := NewApp(strings.NewReader(""), &stdout, &bytes.Buffer{})
	app.Getwd = func() (string, error) { return repo, nil }

	if err := app.Run([]string{"init", "-h"}); err != nil {
		t.Fatalf("init -h error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(configRelativePath))); !os.IsNotExist(err) {
		t.Fatalf("config exists after init -h: %v", err)
	}
}

func TestRunPassesExplicitSkillStatesAndPropagatesExitCode(t *testing.T) {
	repo := initTestRepo(t)
	home := t.TempDir()
	root := filepath.Join(home, ".agents", "skills")
	createSkill(t, root, "pdfs")
	createSkill(t, root, "slides")
	if err := SaveConfig(filepath.Join(repo, filepath.FromSlash(configRelativePath)), Config{Version: 1, Disabled: []string{"slides"}}); err != nil {
		t.Fatal(err)
	}

	app := NewApp(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	app.Getwd = func() (string, error) { return repo, nil }
	app.UserHome = func() (string, error) { return home, nil }
	app.LookPath = func(name string) (string, error) { return "/usr/bin/codex", nil }
	var gotArgs []string
	app.RunCommand = func(name string, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
		if name != "/usr/bin/codex" {
			t.Fatalf("command = %q", name)
		}
		gotArgs = append([]string(nil), args...)
		return 17, nil
	}

	err := app.Run([]string{"run", "--", "exec", "test"})
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 17 || !exitErr.Silent {
		t.Fatalf("run error = %#v, want silent exit 17", err)
	}
	joined := strings.Join(gotArgs, " ")
	if !strings.Contains(joined, `pdfs/SKILL.md",enabled=true`) {
		t.Fatalf("args do not enable pdfs: %#v", gotArgs)
	}
	if !strings.Contains(joined, `slides/SKILL.md",enabled=false`) {
		t.Fatalf("args do not disable slides: %#v", gotArgs)
	}
}
