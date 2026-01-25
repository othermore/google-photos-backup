package cmd

import (
	"fmt"
	"google-photos-backup/internal/api"
	"os"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize photos (Download new items)",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Starting sync process...")

		// 1. Inicializar cliente API
		client, err := api.NewClient()
		if err != nil {
			fmt.Printf("Error initializing API client: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("✅ API Client authenticated successfully.")
		fmt.Println("Fetching first page of media items...")

		// 2. Probar listado (Fase 4 - Test)
		items, nextToken, err := client.ListMediaItems(10, "")
		if err != nil {
			fmt.Printf("Error listing items: %v\n", err)
			os.Exit(1)
		}

		// 3. Mostrar resultados
		for _, item := range items {
			fmt.Printf("- [%s] %s (%s)\n", item.MediaMetadata.CreationTime, item.Filename, item.ID)
		}

		if nextToken != "" {
			fmt.Printf("\n(Hay más páginas disponibles. NextToken: %s...)\n", nextToken[:10])
		}
	},
}
