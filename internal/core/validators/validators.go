package validators

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"gopkg.in/yaml.v3"
)

// ValidationResult represents a single validation finding
type ValidationResult struct {
	File     string `json:"file" yaml:"file"`
	Line     int    `json:"line,omitempty" yaml:"line,omitempty"`
	Rule     string `json:"rule" yaml:"rule"`
	Message  string `json:"message" yaml:"message"`
	Severity string `json:"severity" yaml:"severity"` // error, warning, info
}

// ValidateFile validates a single file based on its type
func ValidateFile(path string) ([]ValidationResult, error) {
	ext := strings.ToLower(filepath.Ext(path))
	base := strings.ToLower(filepath.Base(path))

	switch {
	case base == "dockerfile" || strings.HasPrefix(base, "dockerfile."):
		return validateDockerfile(path)
	case base == "docker-compose.yml" || base == "docker-compose.yaml" || base == "compose.yml" || base == "compose.yaml":
		return validateDockerCompose(path)
	case base == "chart.yaml":
		return validateHelmChart(filepath.Dir(path))
	case ext == ".yml" || ext == ".yaml":
		return validateYAML(path)
	case ext == ".tf":
		return validateTerraform(path)
	default:
		return nil, fmt.Errorf("unsupported file type: %s", base)
	}
}

// ValidateDirectory recursively validates all supported files in a directory
func ValidateDirectory(dir string) ([]ValidationResult, error) {
	var allResults []ValidationResult

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable files
		}
		if info.IsDir() {
			if info.Name() == ".git" || info.Name() == "node_modules" || info.Name() == ".terraform" {
				return filepath.SkipDir
			}
			return nil
		}

		results, validateErr := ValidateFile(path)
		if validateErr != nil {
			return nil // skip unsupported files
		}
		allResults = append(allResults, results...)
		return nil
	})

	return allResults, err
}

func validateDockerfile(path string) ([]ValidationResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var results []ValidationResult
	scanner := bufio.NewScanner(f)
	lineNum := 0
	hasFrom := false
	hasUser := false
	hasHealthcheck := false
	lastInstruction := ""

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		upper := strings.ToUpper(line)

		if strings.HasPrefix(upper, "FROM ") {
			hasFrom = true
			if strings.Contains(line, ":latest") {
				results = append(results, ValidationResult{
					File:     path,
					Line:     lineNum,
					Rule:     "DF-001",
					Message:  "Avoid using :latest tag — pin to a specific version for reproducibility",
					Severity: "warning",
				})
			}
			if strings.Contains(line, "ubuntu") || strings.Contains(line, "debian") {
				results = append(results, ValidationResult{
					File:     path,
					Line:     lineNum,
					Rule:     "DF-002",
					Message:  "Consider using alpine or distroless base images for smaller attack surface",
					Severity: "info",
				})
			}
		}

		if strings.HasPrefix(upper, "USER ") {
			hasUser = true
		}

		if strings.HasPrefix(upper, "HEALTHCHECK ") {
			hasHealthcheck = true
		}

		if strings.HasPrefix(upper, "RUN ") {
			if strings.Contains(line, "apt-get install") && !strings.Contains(line, "--no-install-recommends") {
				results = append(results, ValidationResult{
					File:     path,
					Line:     lineNum,
					Rule:     "DF-003",
					Message:  "Use --no-install-recommends with apt-get install to reduce image size",
					Severity: "warning",
				})
			}
			if strings.Contains(line, "curl ") && strings.Contains(line, "| sh") {
				results = append(results, ValidationResult{
					File:     path,
					Line:     lineNum,
					Rule:     "DF-004",
					Message:  "Piping curl to shell is insecure — download and verify before executing",
					Severity: "error",
				})
			}
		}

		if strings.HasPrefix(upper, "ADD ") && !strings.Contains(line, ".tar") && !strings.Contains(line, "http") {
			results = append(results, ValidationResult{
				File:     path,
				Line:     lineNum,
				Rule:     "DF-005",
				Message:  "Use COPY instead of ADD for local files (ADD has implicit tar extraction)",
				Severity: "warning",
			})
		}

		if strings.HasPrefix(upper, "ENV ") {
			lower := strings.ToLower(line)
			if strings.Contains(lower, "password") || strings.Contains(lower, "secret") || strings.Contains(lower, "api_key") {
				results = append(results, ValidationResult{
					File:     path,
					Line:     lineNum,
					Rule:     "DF-006",
					Message:  "Avoid hardcoding secrets in ENV — use build args or secret mounts",
					Severity: "error",
				})
			}
		}

		lastInstruction = upper
	}

	if !hasFrom {
		results = append(results, ValidationResult{
			File:     path,
			Rule:     "DF-007",
			Message:  "Dockerfile must have at least one FROM instruction",
			Severity: "error",
		})
	}

	if !hasUser {
		results = append(results, ValidationResult{
			File:     path,
			Rule:     "DF-008",
			Message:  "No USER instruction found — container will run as root",
			Severity: "warning",
		})
	}

	if !hasHealthcheck {
		results = append(results, ValidationResult{
			File:     path,
			Rule:     "DF-009",
			Message:  "No HEALTHCHECK instruction — add one for container orchestration",
			Severity: "info",
		})
	}

	_ = lastInstruction

	return results, nil
}

func validateYAML(path string) ([]ValidationResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var results []ValidationResult

	// Check valid YAML
	var doc map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		results = append(results, ValidationResult{
			File:     path,
			Rule:     "YAML-001",
			Message:  fmt.Sprintf("Invalid YAML syntax: %v", err),
			Severity: "error",
		})
		return results, nil
	}

	// Check if Kubernetes manifest
	if apiVersion, ok := doc["apiVersion"]; ok {
		return validateK8sManifest(path, doc, apiVersion)
	}

	// Check if GitHub Actions workflow
	if _, ok := doc["on"]; ok {
		if _, ok2 := doc["jobs"]; ok2 {
			return validateGitHubActions(path, doc)
		}
	}

	return results, nil
}

func validateK8sManifest(path string, doc map[string]interface{}, apiVersion interface{}) ([]ValidationResult, error) {
	var results []ValidationResult

	kind, _ := doc["kind"].(string)
	if kind == "" {
		results = append(results, ValidationResult{
			File:     path,
			Rule:     "K8S-001",
			Message:  "Missing 'kind' field in Kubernetes manifest",
			Severity: "error",
		})
	}

	metadata, _ := doc["metadata"].(map[string]interface{})
	if metadata == nil {
		results = append(results, ValidationResult{
			File:     path,
			Rule:     "K8S-002",
			Message:  "Missing 'metadata' section",
			Severity: "error",
		})
	} else {
		if _, ok := metadata["labels"]; !ok {
			results = append(results, ValidationResult{
				File:     path,
				Rule:     "K8S-003",
				Message:  "Missing labels in metadata — add app, version, and managed-by labels",
				Severity: "warning",
			})
		}
		if _, ok := metadata["namespace"]; !ok && kind != "Namespace" && kind != "ClusterRole" && kind != "ClusterRoleBinding" {
			results = append(results, ValidationResult{
				File:     path,
				Rule:     "K8S-004",
				Message:  "No namespace specified — resource will deploy to 'default' namespace",
				Severity: "warning",
			})
		}
	}

	if kind == "Deployment" || kind == "StatefulSet" || kind == "DaemonSet" {
		spec, _ := doc["spec"].(map[string]interface{})
		if spec != nil {
			template, _ := spec["template"].(map[string]interface{})
			if template != nil {
				tSpec, _ := template["spec"].(map[string]interface{})
				if tSpec != nil {
					containers, _ := tSpec["containers"].([]interface{})
					for _, c := range containers {
						container, _ := c.(map[string]interface{})
						if container == nil {
							continue
						}
						if _, ok := container["resources"]; !ok {
							results = append(results, ValidationResult{
								File:     path,
								Rule:     "K8S-005",
								Message:  fmt.Sprintf("Container '%v' missing resource limits/requests", container["name"]),
								Severity: "warning",
							})
						}
						if _, ok := container["livenessProbe"]; !ok {
							results = append(results, ValidationResult{
								File:     path,
								Rule:     "K8S-006",
								Message:  fmt.Sprintf("Container '%v' missing livenessProbe", container["name"]),
								Severity: "warning",
							})
						}
						if _, ok := container["readinessProbe"]; !ok {
							results = append(results, ValidationResult{
								File:     path,
								Rule:     "K8S-007",
								Message:  fmt.Sprintf("Container '%v' missing readinessProbe", container["name"]),
								Severity: "warning",
							})
						}
						secCtx, _ := container["securityContext"].(map[string]interface{})
						if secCtx != nil {
							if priv, ok := secCtx["privileged"]; ok && priv == true {
								results = append(results, ValidationResult{
									File:     path,
									Rule:     "K8S-008",
									Message:  fmt.Sprintf("Container '%v' is running as privileged — security risk", container["name"]),
									Severity: "error",
								})
							}
						}
						image, _ := container["image"].(string)
						if strings.HasSuffix(image, ":latest") || !strings.Contains(image, ":") {
							results = append(results, ValidationResult{
								File:     path,
								Rule:     "K8S-009",
								Message:  fmt.Sprintf("Container '%v' using :latest or untagged image", container["name"]),
								Severity: "warning",
							})
						}
					}
				}
			}
		}
	}

	return results, nil
}

func validateGitHubActions(path string, doc map[string]interface{}) ([]ValidationResult, error) {
	var results []ValidationResult

	jobs, _ := doc["jobs"].(map[string]interface{})
	for jobName, jobData := range jobs {
		job, _ := jobData.(map[string]interface{})
		if job == nil {
			continue
		}

		steps, _ := job["steps"].([]interface{})
		for _, s := range steps {
			step, _ := s.(map[string]interface{})
			if step == nil {
				continue
			}

			uses, _ := step["uses"].(string)
			if uses != "" && !strings.Contains(uses, "@") {
				results = append(results, ValidationResult{
					File:     path,
					Rule:     "GHA-001",
					Message:  fmt.Sprintf("Job '%s': action '%s' missing version pin — use @vX.Y.Z or @sha", jobName, uses),
					Severity: "warning",
				})
			}
			if uses != "" && strings.Contains(uses, "@master") {
				results = append(results, ValidationResult{
					File:     path,
					Rule:     "GHA-002",
					Message:  fmt.Sprintf("Job '%s': action '%s' pinned to master — use a tagged release", jobName, uses),
					Severity: "warning",
				})
			}
		}

		if _, ok := job["timeout-minutes"]; !ok {
			results = append(results, ValidationResult{
				File:     path,
				Rule:     "GHA-003",
				Message:  fmt.Sprintf("Job '%s' missing timeout-minutes — add to prevent hung workflows", jobName),
				Severity: "info",
			})
		}
	}

	return results, nil
}

func validateDockerCompose(path string) ([]ValidationResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var results []ValidationResult
	var doc map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		results = append(results, ValidationResult{
			File:     path,
			Rule:     "DC-001",
			Message:  fmt.Sprintf("Invalid YAML: %v", err),
			Severity: "error",
		})
		return results, nil
	}

	services, _ := doc["services"].(map[string]interface{})
	for svcName, svcData := range services {
		svc, _ := svcData.(map[string]interface{})
		if svc == nil {
			continue
		}

		if _, ok := svc["restart"]; !ok {
			results = append(results, ValidationResult{
				File:     path,
				Rule:     "DC-002",
				Message:  fmt.Sprintf("Service '%s' missing restart policy", svcName),
				Severity: "warning",
			})
		}

		if _, ok := svc["healthcheck"]; !ok {
			results = append(results, ValidationResult{
				File:     path,
				Rule:     "DC-003",
				Message:  fmt.Sprintf("Service '%s' missing healthcheck", svcName),
				Severity: "info",
			})
		}

		image, _ := svc["image"].(string)
		if strings.HasSuffix(image, ":latest") {
			results = append(results, ValidationResult{
				File:     path,
				Rule:     "DC-004",
				Message:  fmt.Sprintf("Service '%s' using :latest tag", svcName),
				Severity: "warning",
			})
		}
	}

	return results, nil
}

func validateHelmChart(chartDir string) ([]ValidationResult, error) {
	var results []ValidationResult

	chartYaml := filepath.Join(chartDir, "Chart.yaml")
	data, err := os.ReadFile(chartYaml)
	if err != nil {
		return nil, err
	}

	var chart map[string]interface{}
	if err := yaml.Unmarshal(data, &chart); err != nil {
		results = append(results, ValidationResult{
			File:     chartYaml,
			Rule:     "HELM-001",
			Message:  fmt.Sprintf("Invalid Chart.yaml: %v", err),
			Severity: "error",
		})
		return results, nil
	}

	if _, ok := chart["version"]; !ok {
		results = append(results, ValidationResult{
			File:     chartYaml,
			Rule:     "HELM-002",
			Message:  "Missing 'version' in Chart.yaml",
			Severity: "error",
		})
	}

	if _, ok := chart["appVersion"]; !ok {
		results = append(results, ValidationResult{
			File:     chartYaml,
			Rule:     "HELM-003",
			Message:  "Missing 'appVersion' in Chart.yaml",
			Severity: "warning",
		})
	}

	templatesDir := filepath.Join(chartDir, "templates")
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		results = append(results, ValidationResult{
			File:     chartDir,
			Rule:     "HELM-004",
			Message:  "Missing templates/ directory",
			Severity: "error",
		})
	}

	valuesFile := filepath.Join(chartDir, "values.yaml")
	if _, err := os.Stat(valuesFile); os.IsNotExist(err) {
		results = append(results, ValidationResult{
			File:     chartDir,
			Rule:     "HELM-005",
			Message:  "Missing values.yaml",
			Severity: "warning",
		})
	}

	return results, nil
}

func validateTerraform(path string) ([]ValidationResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var results []ValidationResult
	content := string(data)
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "password") || strings.Contains(trimmed, "secret") {
			if strings.Contains(trimmed, "=") && strings.Contains(trimmed, "\"") && !strings.Contains(trimmed, "var.") {
				results = append(results, ValidationResult{
					File:     path,
					Line:     i + 1,
					Rule:     "TF-001",
					Message:  "Possible hardcoded secret detected — use variables or secret manager",
					Severity: "error",
				})
			}
		}

		if strings.Contains(trimmed, `cidr_blocks = ["0.0.0.0/0"]`) {
			results = append(results, ValidationResult{
				File:     path,
				Line:     i + 1,
				Rule:     "TF-002",
				Message:  "Security group allows traffic from 0.0.0.0/0 — restrict to specific CIDRs",
				Severity: "warning",
			})
		}
	}

	if strings.Contains(content, "resource ") && !strings.Contains(content, "tags") {
		results = append(results, ValidationResult{
			File:     path,
			Rule:     "TF-003",
			Message:  "Resources defined without tags — add tags for cost tracking and organization",
			Severity: "info",
		})
	}

	return results, nil
}

// PrintResults outputs results in the specified format
func PrintResults(results []ValidationResult, format string) {
	if len(results) == 0 {
		color.Green("✅ All validations passed!")
		return
	}

	switch format {
	case "json":
		data, _ := json.MarshalIndent(results, "", "  ")
		fmt.Println(string(data))
	default:
		errorColor := color.New(color.FgRed)
		warnColor := color.New(color.FgYellow)
		infoColor := color.New(color.FgCyan)

		for _, r := range results {
			var prefix string
			switch r.Severity {
			case "error":
				prefix = errorColor.Sprint("✖ ERROR")
			case "warning":
				prefix = warnColor.Sprint("⚠ WARN ")
			default:
				prefix = infoColor.Sprint("ℹ INFO ")
			}

			location := r.File
			if r.Line > 0 {
				location = fmt.Sprintf("%s:%d", r.File, r.Line)
			}

			fmt.Printf("  %s  [%s] %s\n         %s\n", prefix, r.Rule, location, r.Message)
		}

		errors, warnings, infos := 0, 0, 0
		for _, r := range results {
			switch r.Severity {
			case "error":
				errors++
			case "warning":
				warnings++
			default:
				infos++
			}
		}
		fmt.Printf("\n  Summary: %d errors, %d warnings, %d info\n", errors, warnings, infos)
	}
}
