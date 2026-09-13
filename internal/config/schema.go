package config

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

//go:embed schema/*.json
var schemaFS embed.FS

var (
	schemaOnce sync.Once
	schemaErr  error
	schemaObj  *jsonschema.Schema
)

func validateSchema(cfg Config) error {
	schema, err := loadSchema()
	if err != nil {
		return err
	}

	raw, err := normalizeJSON(cfg)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return fmt.Errorf("decode config json: %w", err)
	}
	if err := schema.Validate(decoded); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}
	return nil
}

func loadSchema() (*jsonschema.Schema, error) {
	schemaOnce.Do(func() {
		compiler := jsonschema.NewCompiler()
		compiler.Draft = jsonschema.Draft2020
		var bytes []byte
		bytes, schemaErr = schemaFS.ReadFile("schema/config.schema.json")
		if schemaErr != nil {
			return
		}
		if err := compiler.AddResource("config.schema.json", strings.NewReader(string(bytes))); err != nil {
			schemaErr = err
			return
		}
		schemaObj, schemaErr = compiler.Compile("config.schema.json")
	})
	if schemaErr != nil {
		return nil, schemaErr
	}
	return schemaObj, nil
}
