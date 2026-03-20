package auditor

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

// Finding represents a security or best-practice audit finding
type Finding struct {
	File        string `json:"file" yaml:"file"`
	Line        int    `json:"line,omitempty" yaml:"line,omitempty"`
	Rule        string `json:"rule" yaml:"rule"`
	Category    string `json:"category" yaml:"category"`
	Message     string `json:"message" yaml:"message"`
	Severity    string `json:"severity" yaml:"severity"` // critical, high, medium, low
	Remediation string `json:"remediation" yaml:"remediation"`
}

// AuditFile audits a single file
func AuditFile(path string) ([]Finding, error) {
	ext := strings.ToLower(filepath.Ext(path))
	base := strings.ToLower(filepath.Base(path))

	switch {
	case base == "dockerfile" || strings.HasPrefix(base, "dockerfile."):
		return auditDockerfile(path)
	case ext == ".yml" || ext == ".yaml":
		return auditYAML(path)
	case ext == ".tf":
		return auditTerraform(path)
	default:
		return nil, fmt.Errorf("unsupported file type for audit: %s", base)
	}
}

// AuditDirectory recursively audits all supported files
func AuditDirectory(dir string) ([]Finding, error) {
	var allFindings []Finding

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if info.Name() == ".git" || info.Name() == "node_modules" || info.Name() == ".terraform" {
				return filepath.SkipDir
			}
			return nil
		}

		findings, auditErr := AuditFile(path)
		if auditErr != nil {
			return nil
		}
		allFindings = append(allFindings, findings...)
		return nil
	})

	return allFindings, err
}

func auditDockerfile(path string) ([]Finding, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var findings []Finding
	scanner := bufio.NewScanner(f)
	lineNum := 0
	hasUser := false
	runAsRoot := false

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		upper := strings.ToUpper(line)

		if strings.HasPrefix(upper, "USER ") {
			hasUser = true
			user := strings.TrimSpace(strings.TrimPrefix(upper, "USER "))
			if user == "ROOT" || user == "0" {
				runAsRoot = true
			}
		}

		if strings.HasPrefix(upper, "RUN ") {
			if strings.Contains(line, "chmod 777") {
				findings = append(findings, Finding{
					File:        path,
					Line:        lineNum,
					Rule:        "SEC-DF-001",
					Category:    "security",
					Message:     "chmod 777 grants excessive permissions",
					Severity:    "high",
					Remediation: "Use least-privilege permissions (e.g., chmod 755 for executables)",
				})
			}
			if strings.Contains(line, "--allow-root") {
				findings = append(findings, Finding{
					File:        path,
					Line:        lineNum,
					Rule:        "SEC-DF-002",
					Category:    "security",
					Message:     "Running package manager as root explicitly",
					Severity:    "medium",
					Remediation: "Create a non-root user and install packages before switching",
				})
			}
		}

		if strings.HasPrefix(upper, "EXPOSE ") {
			port := strings.TrimSpace(strings.TrimPrefix(upper, "EXPOSE "))
			if port == "22" {
				findings = append(findings, Finding{
					File:        path,
					Line:        lineNum,
					Rule:        "SEC-DF-003",
					Category:    "security",
					Message:     "Exposing SSH port 22 in container",
					Severity:    "high",
					Remediation: "Containers should not run SSH — use kubectl exec or docker exec",
				})
			}
		}
	}

	if !hasUser || runAsRoot {
		findings = append(findings, Finding{
			File:        path,
			Rule:        "SEC-DF-004",
			Category:    "security",
			Message:     "Container runs as root user",
			Severity:    "high",
			Remediation: "Add 'USER nonroot' or 'USER 1000' to run as non-root",
		})
	}

	return findings, nil
}

func auditYAML(path string) ([]Finding, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var doc map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, nil // not valid YAML, skip
	}

	if _, ok := doc["apiVersion"]; ok {
		return auditK8sManifest(path, doc)
	}

	return nil, nil
}

func auditK8sManifest(path string, doc map[string]interface{}) ([]Finding, error) {
	var findings []Finding
	kind, _ := doc["kind"].(string)

	spec, _ := doc["spec"].(map[string]interface{})
	if spec == nil {
		return findings, nil
	}

	// Check Pod-level security
	var podSpec map[string]interface{}
	switch kind {
	case "Deployment", "StatefulSet", "DaemonSet", "Job":
		template, _ := spec["template"].(map[string]interface{})
		if template != nil {
			podSpec, _ = template["spec"].(map[string]interface{})
		}
	case "Pod":
		podSpec = spec
	case "CronJob":
		jobTemplate, _ := spec["jobTemplate"].(map[string]interface{})
		if jobTemplate != nil {
			jSpec, _ := jobTemplate["spec"].(map[string]interface{})
			if jSpec != nil {
				template, _ := jSpec["template"].(map[string]interface{})
				if template != nil {
					podSpec, _ = template["spec"].(map[string]interface{})
				}
			}
		}
	}

	if podSpec == nil {
		return findings, nil
	}

	// Check hostNetwork
	if hostNet, ok := podSpec["hostNetwork"]; ok && hostNet == true {
		findings = append(findings, Finding{
			File:        path,
			Rule:        "SEC-K8S-001",
			Category:    "security",
			Message:     "Pod uses hostNetwork — can access node-level network",
			Severity:    "critical",
			Remediation: "Remove hostNetwork: true unless absolutely required",
		})
	}

	// Check hostPID
	if hostPID, ok := podSpec["hostPID"]; ok && hostPID == true {
		findings = append(findings, Finding{
			File:        path,
			Rule:        "SEC-K8S-002",
			Category:    "security",
			Message:     "Pod uses hostPID — can see all processes on node",
			Severity:    "critical",
			Remediation: "Remove hostPID: true unless running a monitoring DaemonSet",
		})
	}

	// Check service account
	if sa, ok := podSpec["serviceAccountName"]; ok {
		if sa == "default" || sa == "" {
			findings = append(findings, Finding{
				File:        path,
				Rule:        "SEC-K8S-003",
				Category:    "security",
				Message:     "Using default service account",
				Severity:    "medium",
				Remediation: "Create a dedicated service account with minimal RBAC permissions",
			})
		}
	}

	automount, _ := podSpec["automountServiceAccountToken"].(bool)
	if !automount {
		// Check if the key exists at all
		if _, exists := podSpec["automountServiceAccountToken"]; !exists {
			findings = append(findings, Finding{
				File:        path,
				Rule:        "SEC-K8S-004",
				Category:    "security",
				Message:     "Service account token auto-mounted (default behavior)",
				Severity:    "low",
				Remediation: "Set automountServiceAccountToken: false if API access is not needed",
			})
		}
	}

	// Check containers
	containers, _ := podSpec["containers"].([]interface{})
	for _, c := range containers {
		container, _ := c.(map[string]interface{})
		if container == nil {
			continue
		}
		name, _ := container["name"].(string)

		secCtx, _ := container["securityContext"].(map[string]interface{})
		if secCtx == nil {
			findings = append(findings, Finding{
				File:        path,
				Rule:        "SEC-K8S-005",
				Category:    "security",
				Message:     fmt.Sprintf("Container '%s' missing securityContext", name),
				Severity:    "medium",
				Remediation: "Add securityContext with runAsNonRoot, readOnlyRootFilesystem, and capabilities drop",
			})
		} else {
			if priv, ok := secCtx["privileged"]; ok && priv == true {
				findings = append(findings, Finding{
					File:        path,
					Rule:        "SEC-K8S-006",
					Category:    "security",
					Message:     fmt.Sprintf("Container '%s' running in privileged mode", name),
					Severity:    "critical",
					Remediation: "Remove privileged: true — this grants full host access",
				})
			}
			if ro, ok := secCtx["readOnlyRootFilesystem"]; !ok || ro != true {
				findings = append(findings, Finding{
					File:        path,
					Rule:        "SEC-K8S-007",
					Category:    "security",
					Message:     fmt.Sprintf("Container '%s' has writable root filesystem", name),
					Severity:    "medium",
					Remediation: "Set readOnlyRootFilesystem: true and use emptyDir for writable paths",
				})
			}
		}
	}

	return findings, nil
}

func auditTerraform(path string) ([]Finding, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var findings []Finding
	lines := strings.Split(string(data), "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "publicly_accessible") && strings.Contains(trimmed, "true") {
			findings = append(findings, Finding{
				File:        path,
				Line:        i + 1,
				Rule:        "SEC-TF-001",
				Category:    "security",
				Message:     "Resource is publicly accessible",
				Severity:    "high",
				Remediation: "Set publicly_accessible = false and use private endpoints",
			})
		}

		if strings.Contains(trimmed, "encrypted") && strings.Contains(trimmed, "false") {
			findings = append(findings, Finding{
				File:        path,
				Line:        i + 1,
				Rule:        "SEC-TF-002",
				Category:    "security",
				Message:     "Encryption is disabled",
				Severity:    "high",
				Remediation: "Enable encryption at rest with a KMS key",
			})
		}

		if strings.Contains(trimmed, "enable_logging") && strings.Contains(trimmed, "false") {
			findings = append(findings, Finding{
				File:        path,
				Line:        i + 1,
				Rule:        "SEC-TF-003",
				Category:    "security",
				Message:     "Logging is disabled",
				Severity:    "medium",
				Remediation: "Enable logging for audit trail and security monitoring",
			})
		}
	}

	return findings, nil
}

// PrintFindings outputs findings in the specified format
func PrintFindings(findings []Finding, format string) {
	if len(findings) == 0 {
		color.Green("✅ No security issues found!")
		return
	}

	switch format {
	case "json":
		data, _ := json.MarshalIndent(findings, "", "  ")
		fmt.Println(string(data))
	default:
		critColor := color.New(color.FgRed, color.Bold)
		highColor := color.New(color.FgRed)
		medColor := color.New(color.FgYellow)
		lowColor := color.New(color.FgCyan)

		for _, f := range findings {
			var prefix string
			switch f.Severity {
			case "critical":
				prefix = critColor.Sprint("🔴 CRIT")
			case "high":
				prefix = highColor.Sprint("🟠 HIGH")
			case "medium":
				prefix = medColor.Sprint("🟡 MED ")
			default:
				prefix = lowColor.Sprint("🔵 LOW ")
			}

			location := f.File
			if f.Line > 0 {
				location = fmt.Sprintf("%s:%d", f.File, f.Line)
			}

			fmt.Printf("  %s  [%s] %s\n         %s\n         💡 %s\n\n", prefix, f.Rule, location, f.Message, f.Remediation)
		}

		crit, high, med, low := 0, 0, 0, 0
		for _, f := range findings {
			switch f.Severity {
			case "critical":
				crit++
			case "high":
				high++
			case "medium":
				med++
			default:
				low++
			}
		}
		fmt.Printf("  Summary: %d critical, %d high, %d medium, %d low\n", crit, high, med, low)
	}
}
