package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestBuildCodexArgs(t *testing.T) {
	skills := []Skill{
		{Name: "slides", File: "/home/me/.agents/skills/slides/SKILL.md"},
		{Name: "pdfs", File: "/home/me/.agents/skills/pdfs/SKILL.md"},
	}
	got := BuildCodexArgs(skills, map[string]bool{"slides": true}, []string{"exec", "review this"})
	wantPrefix := `skills.config=[{path="/home/me/.agents/skills/pdfs/SKILL.md",enabled=true},{path="/home/me/.agents/skills/slides/SKILL.md",enabled=false}]`
	want := []string{"-c", wantPrefix, "exec", "review this"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildCodexArgs() = %#v, want %#v", got, want)
	}
}

func TestBuildCodexArgsExplicitlyEnablesAllSkills(t *testing.T) {
	skills := []Skill{{Name: "pdfs", File: "/home/me/.agents/skills/pdfs/SKILL.md"}}
	got := BuildCodexArgs(skills, map[string]bool{}, nil)
	if len(got) != 2 || !strings.Contains(got[1], "enabled=true") {
		t.Fatalf("BuildCodexArgs() = %#v, want explicit enabled=true", got)
	}
}

func TestBuildCodexArgsWithoutUserSkills(t *testing.T) {
	got := BuildCodexArgs(nil, nil, []string{"exec", "test"})
	want := []string{"exec", "test"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildCodexArgs() = %#v, want %#v", got, want)
	}
}
