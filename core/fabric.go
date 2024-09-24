package core

import (
	"fmt"
	"os/exec"
	"strings"
)

// RunFabric runs the fabric command with the given pattern and model
func RunFabric(input, pattern, model string) (string, error) {
	var cmd *exec.Cmd
	fmt.Println("Running fabric with pattern:", pattern, "and model:", model)
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

func ListModels() ([]Model, error) {
	cmd := exec.Command("fabric", "-L")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error listing models: %v", err)
	}
	return CleanModels(string(output)), nil
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

// CleanModels takes the raw output from fabric -L and returns a cleaned list of Models
func CleanModels(output string) []Model {
	lines := strings.Split(output, "\n")
	var models []Model
	var currentProvider string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "[") {
			currentProvider = strings.TrimSuffix(line, ":")
		} else if currentProvider != "" {
			// Remove the index number and any leading/trailing whitespace
			parts := strings.SplitN(line, "]", 2)
			if len(parts) == 2 {
				modelName := strings.TrimSpace(parts[1])
				models = append(models, Model{
					Provider: currentProvider,
					Name:     modelName,
				})
			}
		}
	}
	return models
}

// PrintCleanModels prints the cleaned models in the desired format
func PrintCleanModels(models []Model) {
	for _, model := range models {
		fmt.Printf("{\n  provider: %s,\n  model: \"%s\"\n},\n", model.Provider, model.Name)
	}
}
