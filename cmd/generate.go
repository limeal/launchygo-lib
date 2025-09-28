package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"limeal.fr/launchygo/game/folder/generator"
)

var generateCmd = &cobra.Command{
	Use:   "generate <vanilla|fabric> <pack_name> <version>",
	Short: "Generate a minecraft game folder",
	Long: `Generate a minecraft game folder.
Arguments:
  <type>           The type of game folder to generate. Must be either "vanilla" or "fabric".
  <version>        The Minecraft version to use for the generated folder (e.g., "1.20.1").

The generate command will generate a minecraft game folder (vanilla or fabric) and write it in the 'packs/<pack_name>' folder.`,
	Args: cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		generatorType := args[0]
		packName := args[1]
		version := args[2]

		var gen generator.Generator
		if generatorType == "vanilla" {
			gen = generator.InitVanillaGenerator(packName, version)
		} else if args[0] == "fabric" {
			gen = generator.InitFabricGenerator(packName, version)
		} else {
			fmt.Println("Invalid generator type")
			return
		}

		gen.Generate(debug, nil)
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
