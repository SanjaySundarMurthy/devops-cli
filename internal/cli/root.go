package cli

import (
	"github.com/spf13/cobra"
)

var (
	version = "0.1.0"
	output  string
)

var rootCmd = &cobra.Command{
	Use:   "dops",
	Short: "DevOps Swiss Army Knife — a unified CLI for cloud-native operations",
	Long: `dops is a fast, opinionated DevOps CLI tool that helps engineers
validate configurations, audit infrastructure, and automate common
DevOps workflows from a single binary.

Commands:
  validate   Validate Kubernetes manifests, Dockerfiles, Helm charts, and CI configs
  audit      Audit infrastructure for security, cost, and best practices  
  generate   Generate boilerplate configs (Dockerfiles, CI pipelines, K8s manifests)
  check      Run pre-deploy health checks across your stack
  version    Print version information`,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "table", "Output format: table, json, yaml")
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(auditCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(versionCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
