package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/spf13/cobra"
	"limeal.fr/launchygo/pkg/connectors"
	"limeal.fr/launchygo/pkg/game/folder"
)

var publishCmd = &cobra.Command{
	Use:   "publish <pack_name> <uri>",
	Short: "Publish a minecraft game folder",
	Long: `
Publish a minecraft game folder.

Arguments:
  <pack_name>      The name of the pack to publish.
  <output_uri>     The uri where the pack will be published.

The publish command will publish a minecraft game folder, to do that it will first update the manifest.json with extra files (aka: mods, options, etc.).
And then upload the pack to the specified uri.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		packName := args[0]
		outputURI := args[1]

		if !slices.Contains(listAvailablePacks(), packName) {
			fmt.Println("❌ Pack not found")
			fmt.Println("Available packs:")
			for _, pack := range listAvailablePacks() {
				fmt.Println("  - " + pack)
			}
			cmd.Help()
			return
		}

		connector := connectors.FindConnectorFromURI(outputURI)
		if connector == nil {
			fmt.Println("❌ The uri provided is not valid")
			fmt.Println("[Format] <scheme>://<path>")
			cmd.Help()
			return
		}

		fmt.Println("Connecting to connector")
		fmt.Println("Connector: ", connector.GetURI())
		err := connector.Connect()
		if err != nil {
			fmt.Println("❌ Failed to connect to the connector")
			fmt.Println(err)
			return
		}
		defer connector.Close()

		fmt.Println("Publishing game folder")
		folder.PublishGameFolder(connector, packName)
	},
}

func listAvailablePacks() []string {
	packsDir := "./packs"
	entries, err := os.ReadDir(packsDir)
	if err != nil {
		return nil
	}

	if len(entries) == 0 {
		return nil
	}

	packs := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			// Check if the directory contains a manifest.json file
			manifestPath := filepath.Join(packsDir, entry.Name(), "manifest.json")
			if _, err := os.Stat(manifestPath); err == nil {
				packs = append(packs, entry.Name())
			}
		}
	}

	return packs
}

func init() {
	rootCmd.AddCommand(publishCmd)
}
