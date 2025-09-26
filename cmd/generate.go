/*
Package cmd
Copyright © 2025 NAME HERE anggarayusuf96@gmail.com
*/
package cmd

import (
	"fmt"
	"gopenapi/internal/config"
	"gopenapi/internal/mapper"
	"gopenapi/internal/templates"
	"gopenapi/internal/utils"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/iancoleman/strcase"
	"github.com/spf13/cobra"
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
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(cfg.Input)
	if err != nil {
		log.Fatalf("failed to load OpenAPI spec: %v", err)
	}

	models := mapper.MapModelsFromSchemas(doc)
	apis := mapper.MapAPIFromPaths(doc)

	if cfg.Output != "" {
		err = os.MkdirAll(cfg.Output, os.ModePerm)
		if err != nil {
			log.Fatalf("failed to create output directory: %v", err)
			return
		}
	}
	err = os.MkdirAll(cfg.Output+"/"+cfg.Packages.Models, os.ModePerm)
	if err != nil {
		log.Fatalf("failed to create output models directory: %v", err)
		return
	}
	err = os.MkdirAll(cfg.Output+"/"+cfg.Packages.API, os.ModePerm)
	if err != nil {
		log.Fatalf("failed to create output api directory: %v", err)
		return
	}

	for _, model := range models {
		fileName := strcase.ToSnake(model.Name) + cfg.FileNaming.ModelSuffix
		filePath := cfg.Output + "/" + cfg.Packages.Models + "/" + fileName
		renderTemplate("internal/templates/model.tmpl", filePath, model)
		log.Printf("Generated %s", filePath)
	}

	moduleName, err := getModuleName()
	if err != nil {
		log.Fatalf("failed to read module name: %v", err)
	}

	for tag, api := range apis {
		fileName := strcase.ToSnake(tag) + cfg.FileNaming.APISuffix
		filePath := cfg.Output + "/" + cfg.Packages.API + "/" + fileName
		modelPath := moduleName + "/" + cfg.Packages.Models
		if cfg.Output != "" {
			modelPath = moduleName + "/" + cfg.Output + "/" + cfg.Packages.Models
		}
		data := struct {
			Tag        string
			APIs       []templates.API
			ModelsPath string
		}{
			Tag:        utils.CapitalizeFirstWord(tag),
			APIs:       api,
			ModelsPath: modelPath,
		}
		renderTemplate("internal/templates/api.tmpl", filePath, data)
		log.Printf("Generated %s", filePath)
	}
}

func getModuleName() (string, error) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", fmt.Errorf("module name not found in go.mod")
}

func renderTemplate(path, out string, data any) {
	tmplContent, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read template %s: %v", path, err)
	}
	tmpl, err := template.New("").Funcs(template.FuncMap{"upper": strings.ToUpper}).Parse(string(tmplContent))
	if err != nil {
		log.Fatalf("failed to parse template %s: %v", path, err)
	}
	f, err := os.Create(out)
	if err != nil {
		log.Fatalf("failed to create file %s: %v", out, err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Fatalf("failed to close file %s: %v", out, err)
		}
	}(f)
	if err := tmpl.Execute(f, data); err != nil {
		log.Fatalf("failed to execute template %s: %v", path, err)
	}
}
