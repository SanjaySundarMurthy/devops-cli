package validators

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateDockerfile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		content  string
		wantErrs int
		wantRule string
	}{
		{
			name:     "latest tag warning",
			content:  "FROM ubuntu:latest\nRUN apt-get update\n",
			wantRule: "DF-001",
		},
		{
			name:     "missing USER instruction",
			content:  "FROM golang:1.22-alpine\nCOPY . .\nRUN go build -o app\n",
			wantRule: "DF-008",
		},
		{
			name:     "hardcoded secret",
			content:  "FROM alpine:3.19\nENV DB_PASSWORD=\"supersecret\"\n",
			wantRule: "DF-006",
		},
		{
			name:     "curl pipe to shell",
			content:  "FROM alpine:3.19\nRUN curl http://example.com/script | sh\nUSER nobody\n",
			wantRule: "DF-004",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmpDir, "Dockerfile."+tt.name)
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			results, err := ValidateFile(path)
			if err != nil {
				t.Fatal(err)
			}

			found := false
			for _, r := range results {
				if r.Rule == tt.wantRule {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected rule %s in results, got: %v", tt.wantRule, results)
			}
		})
	}
}

func TestValidateK8sManifest(t *testing.T) {
	tmpDir := t.TempDir()

	manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
        - name: app
          image: myapp:latest
`
	path := filepath.Join(tmpDir, "deployment.yaml")
	if err := os.WriteFile(path, []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	results, err := ValidateFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// Should flag missing resource limits, probes, and :latest tag
	ruleSet := make(map[string]bool)
	for _, r := range results {
		ruleSet[r.Rule] = true
	}

	expectedRules := []string{"K8S-005", "K8S-006", "K8S-007", "K8S-009"}
	for _, rule := range expectedRules {
		if !ruleSet[rule] {
			t.Errorf("expected rule %s in results", rule)
		}
	}
}

func TestValidateDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	dockerfile := "FROM alpine:latest\nRUN echo hello\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "Dockerfile"), []byte(dockerfile), 0644); err != nil {
		t.Fatal(err)
	}

	results, err := ValidateDirectory(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) == 0 {
		t.Error("expected at least one validation result")
	}
}

func TestValidateUnsupportedFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ValidateFile(path)
	if err == nil {
		t.Error("expected error for unsupported file type")
	}
}
