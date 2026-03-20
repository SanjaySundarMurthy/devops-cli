package cli

import (
	"fmt"
	"os"

	"github.com/sanjaysundarmurthy/devops-cli/internal/core/validators"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate [file-or-directory]",
	Short: "Validate Kubernetes manifests, Dockerfiles, Helm charts, and CI configs",
	Long: `Validate configuration files for correctness and best practices.

Supported file types:
  - Kubernetes manifests (YAML with apiVersion/kind)
  - Dockerfiles
  - Helm charts (Chart.yaml + templates/)
  - GitHub Actions workflows (.github/workflows/*.yml)
  - docker-compose.yml files`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		info, err := os.Stat(target)
		if err != nil {
			return fmt.Errorf("cannot access %s: %w", target, err)
		}

		var results []validators.ValidationResult

		if info.IsDir() {
			results, err = validators.ValidateDirectory(target)
		} else {
			results, err = validators.ValidateFile(target)
		}
		if err != nil {
			return err
		}

		validators.PrintResults(results, output)

		hasErrors := false
		for _, r := range results {
			if r.Severity == "error" {
				hasErrors = true
				break
			}
		}
		if hasErrors {
			os.Exit(1)
		}
		return nil
	},
}
