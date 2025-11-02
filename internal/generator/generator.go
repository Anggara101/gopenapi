package generator

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/iancoleman/strcase"
	"gopenapi/internal/config"
	"gopenapi/internal/mapper"
	"gopenapi/internal/templates"
	"gopenapi/internal/utils"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type Generator struct {
	cfg *config.Config
}

func NewGenerator(cfg *config.Config) Generator {
	return Generator{
		cfg: cfg,
	}
}

func (g Generator) Generate() {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(g.cfg.Input)
	if err != nil {
		log.Fatalf("failed to load OpenAPI spec: %v", err)
	}

	models := mapper.MapModelsFromSchemas(doc)
	apis := mapper.MapAPIFromPaths(doc)

	createDir(g.cfg)
	renderModel(models, g.cfg)
	renderAPI(apis, g.cfg)
}

func createDir(cfg *config.Config) {
	// Determine base output directory; if empty, use current working directory
	baseOut := "."
	if cfg.Output != "" {
		baseOut = cfg.Output
		if err := os.MkdirAll(baseOut, os.ModePerm); err != nil {
			log.Fatalf("failed to create output directory: %v", err)
			return
		}
	}

	modelsDir := filepath.Join(baseOut, cfg.Packages.Models)
	if err := os.MkdirAll(modelsDir, os.ModePerm); err != nil {
		log.Fatalf("failed to create output models directory: %v", err)
		return
	}

	apiDir := filepath.Join(baseOut, cfg.Packages.API)
	if err := os.MkdirAll(apiDir, os.ModePerm); err != nil {
		log.Fatalf("failed to create output api directory: %v", err)
		return
	}

}

func renderModel(models []templates.Model, cfg *config.Config) {
	baseOut := "."
	if cfg.Output != "" {
		baseOut = cfg.Output
	}
	for _, model := range models {
		fileName := strcase.ToSnake(model.Name) + cfg.FileNaming.ModelSuffix
		filePath := filepath.Join(baseOut, cfg.Packages.Models, fileName)
		renderTemplate("internal/templates/model.tmpl", filePath, model)
		log.Printf("Generated %s", filePath)
	}
}

func renderAPI(apis templates.APIs, cfg *config.Config) {
	moduleName, err := getModuleName()
	if err != nil {
		log.Fatalf("failed to read module name: %v", err)
	}
	baseOut := "."
	if cfg.Output != "" {
		baseOut = cfg.Output
	}

	for tag, api := range apis {
		fileName := strcase.ToSnake(tag) + cfg.FileNaming.APISuffix
		filePath := filepath.Join(baseOut, cfg.Packages.API, fileName)

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

	// Useful helpers for templates
	funcs := template.FuncMap{
		"upper":  strings.ToUpper,
		"lower":  strings.ToLower,
		"snake":  strcase.ToSnake,
		"camel":  strcase.ToLowerCamel,
		"pascal": strcase.ToCamel,
	}

	tmpl, err := template.New("").Funcs(funcs).Parse(string(tmplContent))
	if err != nil {
		log.Fatalf("failed to parse template %s: %v", path, err)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(out), os.ModePerm); err != nil {
		log.Fatalf("failed to ensure output directory for %s: %v", out, err)
	}

	f, err := os.Create(out)
	if err != nil {
		log.Fatalf("failed to create file %s: %v", out, err)
	}
	defer func(f *os.File) {
		if cerr := f.Close(); cerr != nil {
			log.Fatalf("failed to close file %s: %v", out, cerr)
		}
	}(f)

	if err := tmpl.Execute(f, data); err != nil {
		log.Fatalf("failed to execute template %s: %v", path, err)
	}

}
