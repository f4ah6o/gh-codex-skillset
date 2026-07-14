package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type DoctorCheck struct {
	Level   string `json:"level"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

type DoctorReport struct {
	Checks   []DoctorCheck `json:"checks"`
	Warnings int           `json:"warnings"`
	Errors   int           `json:"errors"`
}

func (r *DoctorReport) add(level, name, message string) {
	r.Checks = append(r.Checks, DoctorCheck{Level: level, Name: name, Message: message})
	switch level {
	case "WARN":
		r.Warnings++
	case "ERROR":
		r.Errors++
	}
}

func (a *App) runDoctor(args []string) error {
	fs := newFlagSet("doctor")
	jsonOutput := fs.Bool("json", false, "output JSON")
	codexCommand := fs.String("codex", "codex", "Codex executable name or path")
	showHelp := addHelpFlags(fs)
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if showHelp() {
		fmt.Fprint(a.Stdout, "Usage: gh codex-skillset doctor [--json] [--codex PATH]\n")
		return nil
	}
	if fs.NArg() != 0 {
		return usageError(fmt.Errorf("doctor does not accept positional arguments"))
	}

	report := DoctorReport{}
	root, configPath, err := a.repoContext()
	if err != nil {
		report.add("ERROR", "repository", err.Error())
	} else {
		report.add("OK", "repository", root)
	}

	cfg := DefaultConfig()
	configLoaded := false
	if configPath != "" {
		loaded, exists, loadErr := LoadConfig(configPath)
		if loadErr != nil {
			report.add("ERROR", "config", loadErr.Error())
		} else {
			cfg = loaded
			configLoaded = true
			if exists {
				report.add("OK", "config", configPath)
			} else {
				report.add("WARN", "config", fmt.Sprintf("not found; defaults apply: %s", configPath))
			}
		}
	}

	skillsRoot, homeErr := a.skillsRoot()
	var inventory Inventory
	if homeErr != nil {
		report.add("ERROR", "skills", homeErr.Error())
	} else {
		if _, statErr := os.Stat(skillsRoot); statErr != nil {
			if os.IsNotExist(statErr) {
				report.add("WARN", "skills-root", fmt.Sprintf("not found: %s", skillsRoot))
			} else {
				report.add("ERROR", "skills-root", statErr.Error())
			}
		} else {
			report.add("OK", "skills-root", skillsRoot)
		}
		discovered, discoverErr := DiscoverSkills(skillsRoot)
		if discoverErr != nil {
			report.add("ERROR", "skills", discoverErr.Error())
		} else {
			inventory = discovered
			report.add("OK", "skills", fmt.Sprintf("detected %d user skills", len(inventory.Skills)))
			for _, problem := range inventory.Problems {
				report.add("ERROR", "skill-path", problem)
			}
		}
	}

	if configLoaded {
		seen := map[string]bool{}
		duplicates := []string{}
		for _, name := range cfg.Disabled {
			if seen[name] {
				duplicates = append(duplicates, name)
			}
			seen[name] = true
		}
		if len(duplicates) > 0 {
			sort.Strings(duplicates)
			report.add("ERROR", "config-duplicates", fmt.Sprintf("duplicate disabled skills: %v", duplicates))
		} else {
			report.add("OK", "config-duplicates", "none")
		}
		installed := inventory.ByName()
		missing := []string{}
		for _, name := range cfg.Disabled {
			if _, ok := installed[name]; !ok {
				missing = append(missing, name)
			}
		}
		if len(missing) > 0 {
			sort.Strings(missing)
			report.add("WARN", "disabled-skills", fmt.Sprintf("not installed: %v", missing))
		} else {
			report.add("OK", "disabled-skills", "all configured skills are installed")
		}
	}

	if resolved, lookErr := a.LookPath(*codexCommand); lookErr != nil {
		report.add("ERROR", "codex", fmt.Sprintf("command not found: %s", *codexCommand))
	} else {
		if absolute, absErr := filepath.Abs(resolved); absErr == nil {
			resolved = absolute
		}
		report.add("OK", "codex", resolved)
	}

	if *jsonOutput {
		if err := WriteJSON(a.Stdout, report); err != nil {
			return err
		}
	} else {
		for _, check := range report.Checks {
			fmt.Fprintf(a.Stdout, "%-5s %-18s %s\n", check.Level, check.Name, check.Message)
		}
		fmt.Fprintf(a.Stdout, "\n%d warning(s), %d error(s)\n", report.Warnings, report.Errors)
	}
	if report.Errors > 0 {
		return &ExitError{Code: exitGeneral, Silent: true}
	}
	return nil
}
