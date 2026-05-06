package main

import (
	"github.com/spf13/cobra"
	"github.com/xtls/geoip/lib"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"l", "ls"},
	Short:   "List all available input and output formats",
	Run: func(cmd *cobra.Command, args []string) {
		lib.ListInputConverter()
		println()
		lib.ListOutputConverter()
	},
}
