package cli

import (
	"fmt"

	"github.com/sanjaysundarmurthy/devops-cli/internal/core/generator"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate boilerplate configs (Dockerfiles, CI pipelines, K8s manifests)",
	Long: `Generate production-ready configuration files from templates.

Types:
  dockerfile      Multi-stage Dockerfile for Go, Python, Node.js, Java
  github-actions  CI/CD workflow with build, test, security scan, deploy
  k8s-deploy      Kubernetes Deployment + Service + HPA + PDB
  docker-compose  Multi-service docker-compose with health checks
  helm-chart      Helm chart scaffold with best practices`,
}

var genDockerfileCmd = &cobra.Command{
	Use:   "dockerfile",
	Short: "Generate a production-ready multi-stage Dockerfile",
	RunE: func(cmd *cobra.Command, args []string) error {
		lang, _ := cmd.Flags().GetString("lang")
		out, _ := cmd.Flags().GetString("file")
		content := generator.GenerateDockerfile(lang)
		if out != "" {
			return generator.WriteFile(out, content)
		}
		fmt.Println(content)
		return nil
	},
}

var genGitHubActionsCmd = &cobra.Command{
	Use:   "github-actions",
	Short: "Generate a GitHub Actions CI/CD workflow",
	RunE: func(cmd *cobra.Command, args []string) error {
		lang, _ := cmd.Flags().GetString("lang")
		out, _ := cmd.Flags().GetString("file")
		content := generator.GenerateGitHubActions(lang)
		if out != "" {
			return generator.WriteFile(out, content)
		}
		fmt.Println(content)
		return nil
	},
}

var genK8sCmd = &cobra.Command{
	Use:   "k8s-deploy",
	Short: "Generate Kubernetes Deployment + Service + HPA",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		image, _ := cmd.Flags().GetString("image")
		out, _ := cmd.Flags().GetString("file")
		content := generator.GenerateK8sDeploy(name, image)
		if out != "" {
			return generator.WriteFile(out, content)
		}
		fmt.Println(content)
		return nil
	},
}

var genDockerComposeCmd = &cobra.Command{
	Use:   "docker-compose",
	Short: "Generate a docker-compose.yml with common services",
	RunE: func(cmd *cobra.Command, args []string) error {
		stack, _ := cmd.Flags().GetString("stack")
		out, _ := cmd.Flags().GetString("file")
		content := generator.GenerateDockerCompose(stack)
		if out != "" {
			return generator.WriteFile(out, content)
		}
		fmt.Println(content)
		return nil
	},
}

func init() {
	genDockerfileCmd.Flags().String("lang", "go", "Language: go, python, node, java")
	genDockerfileCmd.Flags().String("file", "", "Output file path (default: stdout)")

	genGitHubActionsCmd.Flags().String("lang", "go", "Language: go, python, node, java")
	genGitHubActionsCmd.Flags().String("file", "", "Output file path (default: stdout)")

	genK8sCmd.Flags().String("name", "myapp", "Application name")
	genK8sCmd.Flags().String("image", "myapp:latest", "Container image")
	genK8sCmd.Flags().String("file", "", "Output file path (default: stdout)")

	genDockerComposeCmd.Flags().String("stack", "web", "Stack type: web, api, data, monitoring")
	genDockerComposeCmd.Flags().String("file", "", "Output file path (default: stdout)")

	generateCmd.AddCommand(genDockerfileCmd)
	generateCmd.AddCommand(genGitHubActionsCmd)
	generateCmd.AddCommand(genK8sCmd)
	generateCmd.AddCommand(genDockerComposeCmd)
}
