package schema

// Copyright (C) 2023 by Posit Software, PBC.

import (
	"bytes"
	"embed"
	"io"
	"path"

	"github.com/rstudio/connect-client/internal/util"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

//go:embed schemas
var schemaFS embed.FS

const ConfigSchemaURL = "https://github.com/rstudio/publishing-client/blob/main/schemas/posit-publishing-schema-v3.json"
const DeploymentSchemaURL = "https://github.com/rstudio/publishing-client/blob/main/schemas/posit-publishing-record-schema-v3.json"

type Validator struct {
	schema *jsonschema.Schema
}

func NewValidator(schemaURL string) (*Validator, error) {
	jsonschema.Loaders = map[string]func(url string) (io.ReadCloser, error){
		"https": loadSchema,
	}
	schema, err := jsonschema.Compile(schemaURL)
	if err != nil {
		return nil, err
	}
	return &Validator{
		schema: schema,
	}, nil
}

func (v *Validator) ValidateTOMLFile(path util.Path) error {
	var content any
	err := util.ReadTOMLFile(path, &content)
	if err != nil {
		return err
	}
	return v.schema.Validate(content)
}

func loadSchema(url string) (io.ReadCloser, error) {
	name := path.Base(url)
	content, err := schemaFS.ReadFile("schemas/" + name)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(content)), nil
}
