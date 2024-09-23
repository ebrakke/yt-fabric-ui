package core

import (
	"fmt"
	"os/exec"
	"strings"
)

// RunFabric runs the fabric command with the given pattern and model
func RunFabric(input, pattern, model string) (string, error) {
	var cmd *exec.Cmd
	if model != "" && model != "default" {
		cmd = exec.Command("fabric", "--pattern", pattern, "--model", model)
	} else {
		cmd = exec.Command("fabric", "--pattern", pattern)
	}
	cmd.Stdin = strings.NewReader(input)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error executing fabric pattern: %v", err)
	}
	return string(output), nil
}

func ListPatterns() ([]string, error) {
	cmd := exec.Command("fabric", "-l")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error listing patterns: %v", err)
	}
	lines := strings.Split(string(output), "\n")
	patterns := make([]string, 0, len(lines))
	for _, line := range lines {
		if line != "" && !strings.HasPrefix(line, "Patterns:") {
			patterns = append(patterns, strings.TrimSpace(line))
		}
	}
	return patterns, nil
}

func ListModels() ([]string, error) {
	cmd := exec.Command("fabric", "-L")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error listing models: %v", err)
	}
	models, err := ParseModels(string(output))
	if err != nil {
		return nil, fmt.Errorf("error parsing models: %v", err)
	}
	modelNames := make([]string, 0, len(models))
	for _, model := range models {
		modelNames = append(modelNames, model.Name)
	}
	return modelNames, nil
}

// Model represents a model with its provider and name
type Model struct {
	Provider string
	Name     string
}

// ParseModels parses the output of the fabric -L command into a list of Models
func ParseModels(output string) ([]Model, error) {
	lines := strings.Split(output, "\n")
	var models []Model
	var currentProvider string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasSuffix(line, ":") {
			currentProvider = strings.TrimSuffix(line, ":")
		} else if currentProvider != "" {
			models = append(models, Model{
				Provider: currentProvider,
				Name:     line,
			})
		}
	}
	return models, nil
}
