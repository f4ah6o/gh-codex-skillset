package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverSkillsDirectChildrenOnly(t *testing.T) {
	root := t.TempDir()
	createSkill(t, root, "alpha")
	createSkill(t, root, "beta")
	if err := os.MkdirAll(filepath.Join(root, "nested", "gamma"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "nested", "gamma", "SKILL.md"), []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	inventory, err := DiscoverSkills(root)
	if err != nil {
		t.Fatalf("DiscoverSkills() error = %v", err)
	}
	if len(inventory.Skills) != 2 {
		t.Fatalf("len(Skills) = %d, want 2", len(inventory.Skills))
	}
	if inventory.Skills[0].Name != "alpha" || inventory.Skills[1].Name != "beta" {
		t.Fatalf("skills = %#v", inventory.Skills)
	}
}

func TestDiscoverSkillsInRootsKeepsDistinctSameNamedSkills(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	createSkill(t, first, "shared")
	createSkill(t, second, "shared")

	inventory, err := DiscoverSkillsInRoots([]string{first, second})
	if err != nil {
		t.Fatalf("DiscoverSkillsInRoots() error = %v", err)
	}
	if len(inventory.Skills) != 2 {
		t.Fatalf("len(Skills) = %d, want 2: %#v", len(inventory.Skills), inventory.Skills)
	}
	if inventory.Skills[0].Name != "shared" || inventory.Skills[1].Name != "shared" {
		t.Fatalf("skills = %#v, want two shared skills", inventory.Skills)
	}
	if inventory.Skills[0].File >= inventory.Skills[1].File {
		t.Fatalf("skills are not sorted by canonical path: %#v", inventory.Skills)
	}
}

func TestDiscoverSkillsInRootsDeduplicatesSameCanonicalSkill(t *testing.T) {
	root := t.TempDir()
	aliasRoot := t.TempDir()
	createSkill(t, root, "shared")
	if err := os.Symlink(filepath.Join(root, "shared"), filepath.Join(aliasRoot, "shared")); err != nil {
		t.Fatal(err)
	}

	inventory, err := DiscoverSkillsInRoots([]string{root, aliasRoot})
	if err != nil {
		t.Fatalf("DiscoverSkillsInRoots() error = %v", err)
	}
	if len(inventory.Skills) != 1 {
		t.Fatalf("len(Skills) = %d, want 1: %#v", len(inventory.Skills), inventory.Skills)
	}
}

func createSkill(t *testing.T, root, name string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test\ndescription: test\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}
