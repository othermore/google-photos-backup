package cmd

import (
	"github.com/spf13/cobra"
)

// toolCmd represents the parent command for all technical/maintenance tools
var toolCmd = &cobra.Command{
	Use:   "tool",
	Short: "Technical and maintenance tools",
	Long:  `A collection of technical tools for maintenance, repair, and configuration of Google Photos Backup.`,
}

func init() {
	rootCmd.AddCommand(toolCmd)
}
