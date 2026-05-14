package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "minik8s",
	Short: "A lightweight Kubernetes-compatible container orchestrator",
	Long:  `MiniK8s is a lightweight container orchestrator that is compatible with Kubernetes core concepts.`,
}

func main() {
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("MiniK8s v0.1.0 (MVP)")
		fmt.Println("A lightweight Kubernetes-compatible container orchestrator")
	},
}
