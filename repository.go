package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func FindRepositoryRoot(directory string) (string, error) {
	cmd := exec.Command("git", "-C", directory, "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		if errorsText, ok := err.(*exec.ExitError); ok {
			message := strings.TrimSpace(string(errorsText.Stderr))
			if message != "" {
				return "", fmt.Errorf("current directory is not inside a Git repository: %s", message)
			}
		}
		return "", fmt.Errorf("current directory is not inside a Git repository")
	}
	root := strings.TrimSpace(string(output))
	if root == "" {
		return "", fmt.Errorf("git returned an empty repository root")
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve repository root: %w", err)
	}
	return root, nil
}
