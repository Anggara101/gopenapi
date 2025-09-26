package config

import (
	"errors"
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	Module     string     `yaml:"module"`
	Input      string     `yaml:"input"`
	Output     string     `yaml:"output"`
	Packages   Package    `yaml:"packages"`
	Options    Option     `yaml:"options"`
	FileNaming FileNaming `yaml:"fileNaming"`
}

type Package struct {
	Models string `yaml:"models"`
	API    string `yaml:"api"`
}

type Option struct {
	SplitModels         bool `yaml:"splitModels"`
	SplitAPIs           bool `yaml:"splitAPIs"`
	InlineNestedSchemas bool `yaml:"inlineNestedSchemas"`
	GenerateRegister    bool `yaml:"generateRegister"`
}

type FileNaming struct {
	APISuffix   string `yaml:"apiSuffix"`
	ModelSuffix string `yaml:"modelSuffix"`
}

func ParseConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Input == "" {
		return nil, errors.New("input is required")
	}
	if cfg.Packages.Models == "" {
		cfg.Packages.Models = "models"
	}
	if cfg.Packages.API == "" {
		cfg.Packages.API = "api"
	}
	if cfg.FileNaming.ModelSuffix == "" {
		cfg.FileNaming.ModelSuffix = "_model.go"
	}
	if cfg.FileNaming.APISuffix == "" {
		cfg.FileNaming.APISuffix = "_api.go"
	}
	return &cfg, nil
}
