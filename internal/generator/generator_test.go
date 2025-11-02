package generator

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopenapi/internal/config"
	"gopenapi/internal/templates"
)

func TestNewGenerator(t *testing.T) {
	cfg := &config.Config{Input: "gopenapi.yaml"}
	g := NewGenerator(cfg)
	if g.cfg != cfg {
		t.Fatalf("cfg not set on generator")
	}
}

func TestGetModuleName_Success(t *testing.T) {
	tmp := t.TempDir()
	restore := chdir(t, tmp)
	defer restore()

	mod := "module example.com/awesome"
	mustWriteFile(t, filepath.Join(tmp, "go.mod"), []byte(mod))

	got, err := getModuleName()
	if err != nil {
		t.Fatalf("getModuleName returned error: %v", err)
	}
	if got != "example.com/awesome" {
		t.Fatalf("unexpected module name: got %q", got)
	}
}

func TestGetModuleName_NoGoMod(t *testing.T) {
	tmp := t.TempDir()
	restore := chdir(t, tmp)
	defer restore()

	if _, err := getModuleName(); err == nil {
		t.Fatalf("expected error when go.mod is missing")
	}
}

func TestRenderTemplate_WritesAndSupportsFuncs(t *testing.T) {
	tmp := t.TempDir()
	restore := chdir(t, tmp)
	defer restore()

	// Prepare template
	tmplDir := filepath.Join(tmp, "internal", "templates")
	mustMkdirAll(t, tmplDir)
	tmplPath := filepath.Join(tmplDir, "test.tmpl")
	mustWriteFile(t, tmplPath, []byte("Hello {{upper .Name}} {{snake .Name}} {{camel .Name}} {{pascal .Name}}"))

	// Render to file
	out := filepath.Join(tmp, "out.txt")
	data := struct{ Name string }{Name: "MyName"}
	renderTemplate(tmplPath, out, data)

	content := mustRead(t, out)
	for _, want := range []string{"HELLO", "my_name", "myName", "MyName"} {
		if !strings.Contains(strings.ToUpper(content), strings.ToUpper(want)) {
			t.Fatalf("rendered content missing %q: %q", want, content)
		}
	}
}

func TestCreateDir_CreatesAll(t *testing.T) {
	tmp := t.TempDir()
	cfg := &config.Config{
		Output: tmp,
		Packages: config.Package{
			Models: "models",
			API:    "api",
		},
	}
	createDir(cfg)

	if _, err := os.Stat(filepath.Join(tmp, "models")); err != nil {
		t.Fatalf("models dir not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "api")); err != nil {
		t.Fatalf("api dir not created: %v", err)
	}
}

func TestRenderAPI_WritesFile_WithAndWithoutOutput(t *testing.T) {
	t.Run("without cfg.Output", func(t *testing.T) {
		tmp := t.TempDir()
		restore := chdir(t, tmp)
		defer restore()

		// go.mod for module name
		mustWriteFile(t, filepath.Join(tmp, "go.mod"), []byte("module example.com/awesome"))

		// templates
		tmplDir := filepath.Join(tmp, "internal", "templates")
		mustMkdirAll(t, tmplDir)
		apiTmplPath := filepath.Join(tmplDir, "api.tmpl")
		mustWriteFile(t, apiTmplPath, []byte("tag={{.Tag}} models={{.ModelsPath}} count={{len .APIs}}"))

		cfg := &config.Config{
			Packages: config.Package{Models: "models", API: "api"},
			FileNaming: config.FileNaming{
				APISuffix: "_api.go",
			},
		}
		createDir(cfg)

		apis := templates.APIs{
			"user": {
				{OperationID: "GetUser", Method: "GET", Path: "/users/{id}"},
			},
		}
		renderAPI(apis, cfg)

		// Expected file path for "user" tag
		outFile := filepath.Join(tmp, "api", "user_api.go")
		content := mustRead(t, outFile)

		if !strings.Contains(content, "tag=User") { // Tag is capitalized by code
			t.Fatalf("expected Tag to be capitalized in template output; got: %q", content)
		}
		if !strings.Contains(content, "models=example.com/awesome/models") {
			t.Fatalf("expected ModelsPath to be module/models; got: %q", content)
		}
		if !strings.Contains(content, "count=1") {
			t.Fatalf("expected API count to be 1; got: %q", content)
		}
	})

	t.Run("with cfg.Output", func(t *testing.T) {
		tmp := t.TempDir()
		restore := chdir(t, tmp)
		defer restore()

		// go.mod for module name
		mustWriteFile(t, filepath.Join(tmp, "go.mod"), []byte("module example.com/awesome"))

		// templates
		tmplDir := filepath.Join(tmp, "internal", "templates")
		mustMkdirAll(t, tmplDir)
		apiTmplPath := filepath.Join(tmplDir, "api.tmpl")
		mustWriteFile(t, apiTmplPath, []byte("models={{.ModelsPath}}"))

		cfg := &config.Config{
			Output:   "gen",
			Packages: config.Package{Models: "models", API: "api"},
			FileNaming: config.FileNaming{
				APISuffix: "_api.go",
			},
		}
		createDir(cfg)

		apis := templates.APIs{"user": {}}
		renderAPI(apis, cfg)

		outFile := filepath.Join(tmp, "gen", "api", "user_api.go")
		content := mustRead(t, outFile)
		if !strings.Contains(content, "models=example.com/awesome/gen/models") {
			t.Fatalf("expected ModelsPath to include output dir; got: %q", content)
		}
	})
}

func TestRenderModel_WritesFile(t *testing.T) {
	tmp := t.TempDir()
	restore := chdir(t, tmp)
	defer restore()

	// templates
	tmplDir := filepath.Join(tmp, "internal", "templates")
	mustMkdirAll(t, tmplDir)
	modelTmplPath := filepath.Join(tmplDir, "model.tmpl")
	// Keep template simple; rely on filename to validate name/suffix
	mustWriteFile(t, modelTmplPath, []byte("ok"))

	cfg := &config.Config{
		Packages: config.Package{Models: "models"},
		FileNaming: config.FileNaming{
			ModelSuffix: "_model.go",
		},
	}
	createDir(cfg)

	// Use a model with Name so file name is deterministic: user_model.go
	models := []templates.Model{
		{
			// Assuming templates.Model has field Name (used by renderModel).
			Name: "User",
		},
	}
	renderModel(models, cfg)

	outFile := filepath.Join(tmp, "models", "user_model.go")
	if _, err := os.Stat(outFile); err != nil {
		t.Fatalf("expected model file to be written: %v", err)
	}
}

func TestGenerator_Generate_MissingSpec_Exits(t *testing.T) {
	// We need a subprocess since log.Fatalf calls os.Exit.
	if os.Getenv("GEN_HELPER") == "1" {
		helperRunGenerate()
		return
	}

	tmp := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=TestGenerator_Generate_MissingSpec_Exits", "-test.v")
	cmd.Env = append(os.Environ(), "GEN_HELPER=1")
	cmd.Dir = tmp // no spec file here

	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit when spec is missing; output:\n%s", string(out))
	}
	if !bytes.Contains(out, []byte("failed to load OpenAPI spec")) {
		t.Fatalf("expected error message to mention spec load failure; got:\n%s", string(out))
	}
}

// --- Helpers ---

func helperRunGenerate() {
	// Minimal template and directory setup to avoid other failures
	_ = os.MkdirAll(filepath.Join("internal", "templates"), os.ModePerm)
	_ = os.WriteFile(filepath.Join("internal", "templates", "api.tmpl"), []byte("x"), 0o644)
	_ = os.WriteFile(filepath.Join("internal", "templates", "model.tmpl"), []byte("y"), 0o644)

	cfg := &config.Config{
		Input: "does-not-exist.yaml",
		Packages: config.Package{
			Models: "models",
			API:    "api",
		},
		FileNaming: config.FileNaming{
			APISuffix:   "_api.go",
			ModelSuffix: "_model.go",
		},
	}
	NewGenerator(cfg).Generate()
}

func chdir(t *testing.T, dir string) func() {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir to temp: %v", err)
	}
	return func() {
		_ = os.Chdir(cwd)
	}
}

func mustMkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
}

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	mustMkdirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", path, err)
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", path, err)
	}
	return string(b)
}
