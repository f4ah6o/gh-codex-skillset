package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	exitGeneral = 1
	exitUsage   = 2
	exitConfig  = 3
	exitRepo    = 4
	exitSkill   = 5
	exitCodex   = 6
)

const configRelativePath = ".codex/user-skills.toml"

type ExitError struct {
	Code   int
	Err    error
	Silent bool
}

func (e *ExitError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("command exited with status %d", e.Code)
	}
	return e.Err.Error()
}

func usageError(err error) error  { return &ExitError{Code: exitUsage, Err: err} }
func configError(err error) error { return &ExitError{Code: exitConfig, Err: err} }
func repoError(err error) error   { return &ExitError{Code: exitRepo, Err: err} }
func skillError(err error) error  { return &ExitError{Code: exitSkill, Err: err} }
func codexError(err error) error  { return &ExitError{Code: exitCodex, Err: err} }

type App struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	Getwd      func() (string, error)
	UserHome   func() (string, error)
	Getenv     func(string) string
	LookPath   func(string) (string, error)
	RunCommand func(name string, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error)
}

func NewApp(stdin io.Reader, stdout, stderr io.Writer) *App {
	return &App{
		Stdin:    stdin,
		Stdout:   stdout,
		Stderr:   stderr,
		Getwd:    os.Getwd,
		UserHome: os.UserHomeDir,
		Getenv:   os.Getenv,
		LookPath: exec.LookPath,
		RunCommand: func(name string, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
			cmd := exec.Command(name, args...)
			cmd.Stdin = stdin
			cmd.Stdout = stdout
			cmd.Stderr = stderr
			err := cmd.Run()
			if err == nil {
				return 0, nil
			}
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				return exitErr.ExitCode(), nil
			}
			return 0, err
		},
	}
}

func (a *App) Run(args []string) error {
	if len(args) == 0 {
		a.printHelp()
		return nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		a.printHelp()
		return nil
	case "version", "--version":
		fmt.Fprintf(a.Stdout, "gh-codex-skillset %s\n", currentVersion())
		return nil
	case "init":
		return a.runInit(args[1:])
	case "list":
		return a.runList(args[1:])
	case "enable":
		return a.runSetEnabled(args[1:], true)
	case "disable":
		return a.runSetEnabled(args[1:], false)
	case "run":
		return a.runCodex(args[1:])
	case "doctor":
		return a.runDoctor(args[1:])
	default:
		return usageError(fmt.Errorf("unknown command %q; run `gh codex-skillset help`", args[0]))
	}
}

func (a *App) printHelp() {
	fmt.Fprint(a.Stdout, `gh-codex-skillset controls user-scoped Codex skills per Git repository.

Usage:
  gh codex-skillset <command> [options]

Commands:
  init       Create .codex/user-skills.toml
  list       List user-scoped skills and project status
  enable     Enable one or more skills for this repository
  disable    Disable one or more skills for this repository
  run        Start Codex with project skill overrides
  doctor     Validate repository, configuration, skills, and Codex
  version    Print the extension version

Run `+"`gh codex-skillset <command> --help`"+` for command options.
`)
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func parseFlags(fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		return usageError(err)
	}
	return nil
}

func addHelpFlags(fs *flag.FlagSet) func() bool {
	long := fs.Bool("help", false, "show help")
	short := fs.Bool("h", false, "show help")
	return func() bool { return *long || *short }
}

func (a *App) repoContext() (string, string, error) {
	cwd, err := a.Getwd()
	if err != nil {
		return "", "", repoError(fmt.Errorf("get current directory: %w", err))
	}
	root, err := FindRepositoryRoot(cwd)
	if err != nil {
		return "", "", repoError(err)
	}
	return root, filepath.Join(root, filepath.FromSlash(configRelativePath)), nil
}

func (a *App) skillRoots() ([]string, error) {
	home, err := a.UserHome()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}

	codexHome := ""
	if a.Getenv != nil {
		codexHome = a.Getenv("CODEX_HOME")
	}
	if codexHome == "" {
		codexHome = filepath.Join(home, ".codex")
	}

	roots := []string{
		filepath.Join(codexHome, "skills"),
		filepath.Join(home, ".agents", "skills"),
	}
	result := make([]string, 0, len(roots))
	seen := make(map[string]bool, len(roots))
	for _, root := range roots {
		absolute, err := filepath.Abs(root)
		if err != nil {
			return nil, fmt.Errorf("resolve skills directory %s: %w", root, err)
		}
		if seen[absolute] {
			continue
		}
		seen[absolute] = true
		result = append(result, absolute)
	}
	return result, nil
}

func (a *App) discoverSkills() (Inventory, error) {
	roots, err := a.skillRoots()
	if err != nil {
		return Inventory{}, err
	}
	return DiscoverSkillsInRoots(roots)
}

func (a *App) runInit(args []string) error {
	fs := newFlagSet("init")
	force := fs.Bool("force", false, "overwrite an existing config")
	allDisabled := fs.Bool("all-disabled", false, "disable every discovered user skill")
	showHelp := addHelpFlags(fs)
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if showHelp() {
		fmt.Fprint(a.Stdout, "Usage: gh codex-skillset init [--force] [--all-disabled]\n")
		return nil
	}
	if fs.NArg() != 0 {
		return usageError(fmt.Errorf("init does not accept positional arguments"))
	}

	_, configPath, err := a.repoContext()
	if err != nil {
		return err
	}
	if _, statErr := os.Stat(configPath); statErr == nil && !*force {
		return configError(fmt.Errorf("config already exists: %s (use --force to overwrite)", configPath))
	} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return configError(fmt.Errorf("inspect config: %w", statErr))
	}

	cfg := DefaultConfig()
	if *allDisabled {
		inventory, err := a.discoverSkills()
		if err != nil {
			return skillError(err)
		}
		for _, skill := range inventory.Skills {
			cfg.Disabled = append(cfg.Disabled, skill.Name)
		}
	}
	cfg.Normalize()
	if err := SaveConfig(configPath, cfg); err != nil {
		return configError(err)
	}
	fmt.Fprintf(a.Stdout, "created %s\n", configPath)
	return nil
}

func (a *App) runList(args []string) error {
	fs := newFlagSet("list")
	onlyEnabled := fs.Bool("enabled", false, "show enabled skills only")
	onlyDisabled := fs.Bool("disabled", false, "show disabled skills only")
	global := fs.Bool("global", false, "ignore repository configuration")
	jsonOutput := fs.Bool("json", false, "output JSON")
	quiet := fs.Bool("quiet", false, "output skill names only")
	showHelp := addHelpFlags(fs)
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if showHelp() {
		fmt.Fprint(a.Stdout, "Usage: gh codex-skillset list [--enabled|--disabled] [--global] [--json|--quiet]\n")
		return nil
	}
	if fs.NArg() != 0 {
		return usageError(fmt.Errorf("list does not accept positional arguments"))
	}
	if *onlyEnabled && *onlyDisabled {
		return usageError(fmt.Errorf("--enabled and --disabled cannot be used together"))
	}
	if *jsonOutput && *quiet {
		return usageError(fmt.Errorf("--json and --quiet cannot be used together"))
	}

	cfg := DefaultConfig()
	if !*global {
		_, configPath, err := a.repoContext()
		if err != nil {
			return err
		}
		loaded, _, err := LoadConfig(configPath)
		if err != nil {
			return configError(err)
		}
		cfg = loaded
	}

	inventory, err := a.discoverSkills()
	if err != nil {
		return skillError(err)
	}
	disabled := cfg.DisabledSet()
	rows := make([]SkillStatus, 0, len(inventory.Skills))
	for _, skill := range inventory.Skills {
		enabled := *global || !disabled[skill.Name]
		if *onlyEnabled && !enabled {
			continue
		}
		if *onlyDisabled && enabled {
			continue
		}
		rows = append(rows, SkillStatus{Name: skill.Name, Enabled: enabled, Path: skill.File})
	}

	if *jsonOutput {
		return WriteJSON(a.Stdout, rows)
	}
	if *quiet {
		for _, row := range rows {
			fmt.Fprintln(a.Stdout, row.Name)
		}
		return nil
	}
	WriteSkillTable(a.Stdout, rows)
	for _, problem := range inventory.Problems {
		fmt.Fprintf(a.Stderr, "warning: %s\n", problem)
	}
	return nil
}

func (a *App) runSetEnabled(args []string, enabled bool) error {
	commandName := "disable"
	if enabled {
		commandName = "enable"
	}
	fs := newFlagSet(commandName)
	all := fs.Bool("all", false, "apply to all discovered user skills")
	allowMissing := fs.Bool("allow-missing", false, "permit names that are not currently installed")
	showHelp := addHelpFlags(fs)
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if showHelp() {
		fmt.Fprintf(a.Stdout, "Usage: gh codex-skillset %s [--all] [--allow-missing] <skill...>\n", commandName)
		return nil
	}

	names := append([]string(nil), fs.Args()...)
	if *all && len(names) > 0 {
		return usageError(fmt.Errorf("--all cannot be combined with skill names"))
	}
	if !*all && len(names) == 0 {
		return usageError(fmt.Errorf("specify at least one skill name or use --all"))
	}

	_, configPath, err := a.repoContext()
	if err != nil {
		return err
	}
	cfg, _, err := LoadConfig(configPath)
	if err != nil {
		return configError(err)
	}
	inventory, err := a.discoverSkills()
	if err != nil {
		return skillError(err)
	}
	installed := inventory.ByName()
	if *all {
		names = make([]string, 0, len(inventory.Skills))
		for _, skill := range inventory.Skills {
			names = append(names, skill.Name)
		}
	}

	if !*allowMissing {
		missing := make([]string, 0)
		for _, name := range names {
			if _, ok := installed[name]; !ok {
				missing = append(missing, name)
			}
		}
		if len(missing) > 0 {
			sort.Strings(missing)
			return skillError(fmt.Errorf("user skill not found: %s; no changes were made", strings.Join(missing, ", ")))
		}
	}

	set := cfg.DisabledSet()
	for _, name := range names {
		if enabled {
			delete(set, name)
		} else {
			set[name] = true
		}
	}
	cfg.Disabled = setKeys(set)
	cfg.Normalize()
	if err := SaveConfig(configPath, cfg); err != nil {
		return configError(err)
	}
	state := "disabled"
	if enabled {
		state = "enabled"
	}
	for _, name := range names {
		fmt.Fprintf(a.Stdout, "%s %s\n", state, name)
	}
	return nil
}

func (a *App) runCodex(args []string) error {
	fs := newFlagSet("run")
	dryRun := fs.Bool("dry-run", false, "print the Codex invocation without running it")
	strict := fs.Bool("strict", true, "fail when a disabled skill is not installed")
	noStrict := fs.Bool("no-strict", false, "warn and skip disabled skills that are not installed")
	codexCommand := fs.String("codex", "codex", "Codex executable name or path")
	showHelp := addHelpFlags(fs)
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if showHelp() {
		fmt.Fprint(a.Stdout, "Usage: gh codex-skillset run [--dry-run] [--strict|--no-strict] [--codex PATH] [-- <codex args...>]\n")
		return nil
	}
	if *noStrict {
		*strict = false
	}

	_, configPath, err := a.repoContext()
	if err != nil {
		return err
	}
	cfg, _, err := LoadConfig(configPath)
	if err != nil {
		return configError(err)
	}
	inventory, err := a.discoverSkills()
	if err != nil {
		return skillError(err)
	}
	installed := inventory.ByName()
	missing := make([]string, 0)
	for _, name := range cfg.Disabled {
		if _, ok := installed[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		if *strict {
			return skillError(fmt.Errorf("disabled user skill is not installed: %s", strings.Join(missing, ", ")))
		}
		fmt.Fprintf(a.Stderr, "warning: skipped uninstalled disabled skills: %s\n", strings.Join(missing, ", "))
	}

	codexArgs := BuildCodexArgs(inventory.Skills, cfg.DisabledSet(), fs.Args())
	if *dryRun {
		fmt.Fprintln(a.Stdout, FormatCommand(*codexCommand, codexArgs))
		return nil
	}

	resolved, err := a.LookPath(*codexCommand)
	if err != nil {
		return codexError(fmt.Errorf("codex command was not found: %s", *codexCommand))
	}
	exitCode, err := a.RunCommand(resolved, codexArgs, a.Stdin, a.Stdout, a.Stderr)
	if err != nil {
		return codexError(fmt.Errorf("start Codex: %w", err))
	}
	if exitCode != 0 {
		return &ExitError{Code: exitCode, Silent: true}
	}
	return nil
}

func setKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
