package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

func BuildCodexOverride(skills []Skill, disabled map[string]bool) string {
	if len(skills) == 0 {
		return ""
	}
	skills = append([]Skill(nil), skills...)
	sort.Slice(skills, func(i, j int) bool {
		if skills[i].Name != skills[j].Name {
			return skills[i].Name < skills[j].Name
		}
		return skills[i].File < skills[j].File
	})
	var b strings.Builder
	b.WriteString("skills.config=[")
	for i, skill := range skills {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "{path=%s,enabled=%t}", strconv.Quote(skill.File), !disabled[skill.Name])
	}
	b.WriteByte(']')
	return b.String()
}

func BuildCodexArgs(skills []Skill, disabled map[string]bool, forwarded []string) []string {
	args := make([]string, 0, len(forwarded)+2)
	if override := BuildCodexOverride(skills, disabled); override != "" {
		args = append(args, "-c", override)
	}
	args = append(args, forwarded...)
	return args
}

func FormatCommand(name string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, quoteCommandArg(name))
	for _, arg := range args {
		parts = append(parts, quoteCommandArg(arg))
	}
	return strings.Join(parts, " ")
}

func quoteCommandArg(value string) string {
	if value != "" && strings.IndexFunc(value, func(r rune) bool {
		return unicode.IsSpace(r) || strings.ContainsRune("'\"\\$`;&|<>()[]{}*!?", r)
	}) == -1 {
		return value
	}
	return strconv.Quote(value)
}
