/*
Package cmd
Copyright © 2025 NAME HERE anggarayusuf96@gmail.com
*/
package cmd

import (
	"github.com/spf13/cobra"
	"gopenapi/internal/config"
	"gopenapi/internal/generator"
	"log"
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate code from an OpenAPI spec",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		generate()
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// generateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// generateCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func generate() {
	cfg, err := config.ParseConfig("gopenapi.yaml")
	if err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}
	gen := generator.NewGenerator(cfg)
	gen.Generate()
}
