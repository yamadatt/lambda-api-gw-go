package main

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

// SwaggerSpec represents a minimal subset of the swagger spec.
// Adjust the structure if you need to validate additional fields.
type SwaggerSpec struct {
	Swagger string `yaml:"swagger"` // Swagger 2.0用
	OpenAPI string `yaml:"openapi"` // OpenAPI 3.0用
	Info    struct {
		Title   string `yaml:"title"`
		Version string `yaml:"version"`
	} `yaml:"info"`
}

// TestSwaggerYAMLValid verifies that the swagger.yaml file is a valid YAML
// and includes a non-empty "swagger" field.
func TestSwaggerYAMLValid(t *testing.T) {
	data, err := os.ReadFile("swagger.yaml")
	if err != nil {
		t.Fatalf("failed to read swagger.yaml: %v", err)
	}

	var spec SwaggerSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		t.Fatalf("failed to parse swagger.yaml: %v", err)
	}

	if spec.Swagger == "" && spec.OpenAPI == "" {
		t.Fatalf("invalid swagger spec: missing or empty 'swagger' or 'openapi' field")
	}

	// Optionally, check that basic info is present.
	if spec.Info.Title == "" || spec.Info.Version == "" {
		t.Log("warning: swagger info block is empty; consider specifying title and version")
	}
}
