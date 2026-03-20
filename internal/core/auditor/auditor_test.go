package auditor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAuditDockerfile(t *testing.T) {
	tmpDir := t.TempDir()

	content := "FROM ubuntu:20.04\nRUN chmod 777 /app\nEXPOSE 22\n"
	path := filepath.Join(tmpDir, "Dockerfile")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	findings, err := AuditFile(path)
	if err != nil {
		t.Fatal(err)
	}

	ruleSet := make(map[string]bool)
	for _, f := range findings {
		ruleSet[f.Rule] = true
	}

	expected := []string{"SEC-DF-001", "SEC-DF-003", "SEC-DF-004"}
	for _, rule := range expected {
		if !ruleSet[rule] {
			t.Errorf("expected rule %s in findings", rule)
		}
	}
}

func TestAuditK8sManifest(t *testing.T) {
	tmpDir := t.TempDir()

	manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  template:
    spec:
      hostNetwork: true
      containers:
        - name: app
          image: myapp:1.0
          securityContext:
            privileged: true
`
	path := filepath.Join(tmpDir, "deploy.yaml")
	if err := os.WriteFile(path, []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	findings, err := AuditFile(path)
	if err != nil {
		t.Fatal(err)
	}

	ruleSet := make(map[string]bool)
	for _, f := range findings {
		ruleSet[f.Rule] = true
	}

	if !ruleSet["SEC-K8S-001"] {
		t.Error("expected SEC-K8S-001 (hostNetwork)")
	}
	if !ruleSet["SEC-K8S-006"] {
		t.Error("expected SEC-K8S-006 (privileged)")
	}
}

func TestAuditDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	content := "FROM ubuntu:20.04\nRUN chmod 777 /app\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "Dockerfile"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	findings, err := AuditDirectory(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(findings) == 0 {
		t.Error("expected at least one finding")
	}
}
