package cli

import (
	"fmt"
	"os"

	"github.com/sanjaysundarmurthy/devops-cli/internal/core/auditor"
	"github.com/spf13/cobra"
)

var auditCmd = &cobra.Command{
	Use:   "audit [file-or-directory]",
	Short: "Audit infrastructure for security, cost, and best practices",
	Long: `Run security and best-practice audits against infrastructure configs.

Checks include:
  - Kubernetes: privileged containers, missing resource limits, missing probes
  - Dockerfiles: running as root, missing HEALTHCHECK, large base images
  - Terraform: hardcoded secrets, missing tags, public access
  - Helm: missing security contexts, default service accounts`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		info, err := os.Stat(target)
		if err != nil {
			return fmt.Errorf("cannot access %s: %w", target, err)
		}

		var findings []auditor.Finding

		if info.IsDir() {
			findings, err = auditor.AuditDirectory(target)
		} else {
			findings, err = auditor.AuditFile(target)
		}
		if err != nil {
			return err
		}

		auditor.PrintFindings(findings, output)

		criticalCount := 0
		for _, f := range findings {
			if f.Severity == "critical" || f.Severity == "high" {
				criticalCount++
			}
		}
		if criticalCount > 0 {
			fmt.Fprintf(os.Stderr, "\n⚠️  Found %d critical/high severity issues\n", criticalCount)
			os.Exit(1)
		}
		return nil
	},
}
