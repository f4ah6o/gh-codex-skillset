package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type Skill struct {
	Name      string
	Directory string
	File      string
}

type Inventory struct {
	Skills   []Skill
	Problems []string
}

func (i Inventory) ByName() map[string]Skill {
	result := make(map[string]Skill, len(i.Skills))
	for _, skill := range i.Skills {
		result[skill.Name] = skill
	}
	return result
}

func DiscoverSkills(root string) (Inventory, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return Inventory{}, fmt.Errorf("resolve user skills directory: %w", err)
	}
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return Inventory{}, nil
	}
	if err != nil {
		return Inventory{}, fmt.Errorf("read user skills directory %s: %w", root, err)
	}

	inventory := Inventory{}
	for _, entry := range entries {
		entryPath := filepath.Join(root, entry.Name())
		info, err := os.Stat(entryPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) && entry.Type()&os.ModeSymlink != 0 {
				inventory.Problems = append(inventory.Problems, fmt.Sprintf("broken skill directory symlink: %s", entryPath))
				continue
			}
			return Inventory{}, fmt.Errorf("inspect skill directory %s: %w", entryPath, err)
		}
		if !info.IsDir() {
			continue
		}

		skillFile := filepath.Join(entryPath, "SKILL.md")
		fileInfo, err := os.Stat(skillFile)
		if errors.Is(err, os.ErrNotExist) {
			if linkInfo, linkErr := os.Lstat(skillFile); linkErr == nil && linkInfo.Mode()&os.ModeSymlink != 0 {
				inventory.Problems = append(inventory.Problems, fmt.Sprintf("broken SKILL.md symlink: %s", skillFile))
			}
			continue
		}
		if err != nil {
			return Inventory{}, fmt.Errorf("inspect %s: %w", skillFile, err)
		}
		if fileInfo.IsDir() {
			inventory.Problems = append(inventory.Problems, fmt.Sprintf("SKILL.md is not a file: %s", skillFile))
			continue
		}

		canonicalDirectory, err := filepath.EvalSymlinks(entryPath)
		if err != nil {
			inventory.Problems = append(inventory.Problems, fmt.Sprintf("resolve skill directory %s: %v", entryPath, err))
			continue
		}
		canonicalDirectory, err = filepath.Abs(canonicalDirectory)
		if err != nil {
			return Inventory{}, fmt.Errorf("resolve skill directory %s: %w", entryPath, err)
		}
		canonicalFile, err := filepath.EvalSymlinks(skillFile)
		if err != nil {
			inventory.Problems = append(inventory.Problems, fmt.Sprintf("resolve SKILL.md %s: %v", skillFile, err))
			continue
		}
		canonicalFile, err = filepath.Abs(canonicalFile)
		if err != nil {
			return Inventory{}, fmt.Errorf("resolve SKILL.md %s: %w", skillFile, err)
		}

		inventory.Skills = append(inventory.Skills, Skill{
			Name:      entry.Name(),
			Directory: canonicalDirectory,
			File:      canonicalFile,
		})
	}
	sort.Slice(inventory.Skills, func(a, b int) bool {
		if inventory.Skills[a].Name != inventory.Skills[b].Name {
			return inventory.Skills[a].Name < inventory.Skills[b].Name
		}
		return inventory.Skills[a].File < inventory.Skills[b].File
	})
	sort.Strings(inventory.Problems)
	return inventory, nil
}

// DiscoverSkillsInRoots discovers direct-child skills from multiple user
// scope roots. The canonical file path is used to avoid reporting the same
// symlinked skill more than once, while different files with the same name
// remain separate inventory entries.
func DiscoverSkillsInRoots(roots []string) (Inventory, error) {
	inventory := Inventory{}
	seenFiles := make(map[string]bool)

	for _, root := range roots {
		discovered, err := DiscoverSkills(root)
		if err != nil {
			return Inventory{}, err
		}
		inventory.Problems = append(inventory.Problems, discovered.Problems...)
		for _, skill := range discovered.Skills {
			if seenFiles[skill.File] {
				continue
			}
			seenFiles[skill.File] = true
			inventory.Skills = append(inventory.Skills, skill)
		}
	}

	sort.Slice(inventory.Skills, func(a, b int) bool {
		if inventory.Skills[a].Name != inventory.Skills[b].Name {
			return inventory.Skills[a].Name < inventory.Skills[b].Name
		}
		return inventory.Skills[a].File < inventory.Skills[b].File
	})
	sort.Strings(inventory.Problems)
	return inventory, nil
}

type SkillStatus struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Path    string `json:"path"`
}
