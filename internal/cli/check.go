package cli

import (
	"fmt"

	"github.com/sanjaysundarmurthy/devops-cli/internal/core/checker"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check [file-or-directory]",
	Short: "Run pre-deploy health checks across your stack",
	Long: `Run comprehensive pre-deployment checks including:
  - Config file syntax validation
  - Required fields verification
  - Security baseline checks
  - Resource configuration review
  - Dependency verification`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		results, err := checker.RunChecks(target)
		if err != nil {
			return err
		}
		checker.PrintResults(results, output)

		passed := 0
		failed := 0
		for _, r := range results {
			if r.Passed {
				passed++
			} else {
				failed++
			}
		}

		fmt.Printf("\n📊 Results: %d passed, %d failed out of %d checks\n", passed, failed, len(results))
		if failed > 0 {
			return fmt.Errorf("%d checks failed", failed)
		}
		return nil
	},
}
