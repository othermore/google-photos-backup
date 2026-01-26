package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Start Google Takeout backup process",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Starting Google Takeout automation...")
		fmt.Println("TODO: Implement Go-Rod logic to request and download backup.")
	},
}
