package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var debug bool

var rootCmd = &cobra.Command{
	Use:   "launchygo",
	Short: "launchygo is a tool for generating and launching minecraft",
	Long:  `launchygo is a tool for generating and launching minecraft. It provides a command line interface for generating and launching minecraft.`,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug mode")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
