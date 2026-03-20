package checker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"gopkg.in/yaml.v3"
)

// CheckResult represents a single pre-deploy check result
type CheckResult struct {
	Check   string `json:"check" yaml:"check"`
	File    string `json:"file,omitempty" yaml:"file,omitempty"`
	Passed  bool   `json:"passed" yaml:"passed"`
	Message string `json:"message" yaml:"message"`
}

// RunChecks runs all pre-deployment checks against a target
func RunChecks(target string) ([]CheckResult, error) {
	var results []CheckResult

	info, err := os.Stat(target)
	if err != nil {
		return nil, fmt.Errorf("cannot access %s: %w", target, err)
	}

	if info.IsDir() {
		results = append(results, checkDirectoryStructure(target)...)
		results = append(results, checkRequiredFiles(target)...)
		results = append(results, checkYAMLSyntax(target)...)
		results = append(results, checkGitIgnore(target)...)
		results = append(results, checkSecrets(target)...)
	} else {
		results = append(results, checkSingleFile(target)...)
	}

	return results, nil
}

func checkDirectoryStructure(dir string) []CheckResult {
	var results []CheckResult

	// Check for common anti-patterns
	largeDirs := []string{"node_modules", ".terraform", "vendor"}
	for _, d := range largeDirs {
		fullPath := filepath.Join(dir, d)
		if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
			results = append(results, CheckResult{
				Check:   "directory-structure",
				File:    fullPath,
				Passed:  false,
				Message: fmt.Sprintf("Directory '%s' should not be committed — add to .gitignore", d),
			})
		}
	}

	return results
}

func checkRequiredFiles(dir string) []CheckResult {
	var results []CheckResult

	requiredFiles := map[string]string{
		"README.md":   "Repository should have a README for documentation",
		".gitignore":  "Repository should have .gitignore to exclude build artifacts",
	}

	for file, reason := range requiredFiles {
		fullPath := filepath.Join(dir, file)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			results = append(results, CheckResult{
				Check:   "required-files",
				File:    file,
				Passed:  false,
				Message: reason,
			})
		} else {
			results = append(results, CheckResult{
				Check:   "required-files",
				File:    file,
				Passed:  true,
				Message: fmt.Sprintf("%s found", file),
			})
		}
	}

	return results
}

func checkYAMLSyntax(dir string) []CheckResult {
	var results []CheckResult

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if info.Name() == ".git" || info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yml" && ext != ".yaml" {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}

		var doc interface{}
		if yamlErr := yaml.Unmarshal(data, &doc); yamlErr != nil {
			results = append(results, CheckResult{
				Check:   "yaml-syntax",
				File:    path,
				Passed:  false,
				Message: fmt.Sprintf("Invalid YAML: %v", yamlErr),
			})
		}

		return nil
	})

	if err == nil && len(results) == 0 {
		results = append(results, CheckResult{
			Check:   "yaml-syntax",
			Passed:  true,
			Message: "All YAML files have valid syntax",
		})
	}

	return results
}

func checkGitIgnore(dir string) []CheckResult {
	var results []CheckResult

	gitignorePath := filepath.Join(dir, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		return results
	}

	content := string(data)
	patterns := []string{".env", "*.tfstate", "*.tfvars", ".terraform"}
	for _, pattern := range patterns {
		if !strings.Contains(content, pattern) {
			results = append(results, CheckResult{
				Check:   "gitignore-patterns",
				File:    ".gitignore",
				Passed:  false,
				Message: fmt.Sprintf("Consider adding '%s' to .gitignore", pattern),
			})
		}
	}

	return results
}

func checkSecrets(dir string) []CheckResult {
	var results []CheckResult
	secretPatterns := []string{".env", ".env.local", ".env.production"}

	for _, pattern := range secretPatterns {
		fullPath := filepath.Join(dir, pattern)
		if _, err := os.Stat(fullPath); err == nil {
			results = append(results, CheckResult{
				Check:   "secret-files",
				File:    pattern,
				Passed:  false,
				Message: fmt.Sprintf("File '%s' found — ensure it's in .gitignore and not committed", pattern),
			})
		}
	}

	return results
}

func checkSingleFile(path string) []CheckResult {
	var results []CheckResult

	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".yml" || ext == ".yaml" {
		data, err := os.ReadFile(path)
		if err != nil {
			results = append(results, CheckResult{
				Check:   "file-readable",
				File:    path,
				Passed:  false,
				Message: fmt.Sprintf("Cannot read file: %v", err),
			})
			return results
		}

		var doc interface{}
		if yamlErr := yaml.Unmarshal(data, &doc); yamlErr != nil {
			results = append(results, CheckResult{
				Check:   "yaml-syntax",
				File:    path,
				Passed:  false,
				Message: fmt.Sprintf("Invalid YAML: %v", yamlErr),
			})
		} else {
			results = append(results, CheckResult{
				Check:   "yaml-syntax",
				File:    path,
				Passed:  true,
				Message: "Valid YAML syntax",
			})
		}
	}

	return results
}

// PrintResults outputs check results in the specified format
func PrintResults(results []CheckResult, format string) {
	switch format {
	case "json":
		data, _ := json.MarshalIndent(results, "", "  ")
		fmt.Println(string(data))
	default:
		passColor := color.New(color.FgGreen)
		failColor := color.New(color.FgRed)

		for _, r := range results {
			var icon string
			if r.Passed {
				icon = passColor.Sprint("✅")
			} else {
				icon = failColor.Sprint("❌")
			}

			file := ""
			if r.File != "" {
				file = fmt.Sprintf(" (%s)", r.File)
			}

			fmt.Printf("  %s [%s]%s %s\n", icon, r.Check, file, r.Message)
		}
	}
}
