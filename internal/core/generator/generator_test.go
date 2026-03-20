package generator

import (
	"strings"
	"testing"
)

func TestGenerateDockerfile(t *testing.T) {
	languages := []string{"go", "python", "node", "java"}

	for _, lang := range languages {
		t.Run(lang, func(t *testing.T) {
			result := GenerateDockerfile(lang)
			if result == "" {
				t.Error("expected non-empty Dockerfile")
			}
			if !strings.Contains(result, "FROM ") {
				t.Error("expected FROM instruction")
			}
			if !strings.Contains(result, "HEALTHCHECK") {
				t.Error("expected HEALTHCHECK instruction")
			}
			if !strings.Contains(strings.ToUpper(result), "USER") {
				t.Error("expected USER instruction for non-root")
			}
		})
	}
}

func TestGenerateK8sDeploy(t *testing.T) {
	result := GenerateK8sDeploy("test-app", "test:1.0")

	checks := []string{
		"name: test-app",
		"image: test:1.0",
		"livenessProbe",
		"readinessProbe",
		"resources",
		"securityContext",
		"HorizontalPodAutoscaler",
		"PodDisruptionBudget",
	}

	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("expected %q in K8s deploy output", check)
		}
	}
}

func TestGenerateGitHubActions(t *testing.T) {
	result := GenerateGitHubActions("go")

	if !strings.Contains(result, "golangci-lint") {
		t.Error("expected golangci-lint step")
	}
	if !strings.Contains(result, "trivy") {
		t.Error("expected security scanning step")
	}
}

func TestGenerateDockerCompose(t *testing.T) {
	result := GenerateDockerCompose("web")
	if !strings.Contains(result, "healthcheck") {
		t.Error("expected healthcheck in compose")
	}
	if !strings.Contains(result, "restart") {
		t.Error("expected restart policy")
	}
}
