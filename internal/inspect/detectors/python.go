package detectors

// Copyright (C) 2023 by Posit Software, PBC.

import (
	"github.com/rstudio/connect-client/internal/config"
	"github.com/rstudio/connect-client/internal/util"
)

type PythonAppDetector struct {
	inferenceHelper
	contentType config.ContentType
	imports     []string
}

func NewPythonAppDetector(contentType config.ContentType, imports []string) *PythonAppDetector {
	return &PythonAppDetector{
		inferenceHelper: defaultInferenceHelper{},
		contentType:     contentType,
		imports:         imports,
	}
}

func NewFlaskDetector() *PythonAppDetector {
	return NewPythonAppDetector(config.ContentTypePythonFlask, []string{
		"flask", // also matches flask_api, flask_openapi3, etc.
		"flasgger",
		"falcon", // must check for this after falcon.asgi (FastAPI)
		"bottle",
		"pycnic",
	})
}

func NewFastAPIDetector() *PythonAppDetector {
	return NewPythonAppDetector(config.ContentTypePythonFastAPI, []string{
		"fastapi",
		"falcon.asgi",
		"quart",
		"sanic",
		"starlette",
		"vetiver",
	})
}

func NewDashDetector() *PythonAppDetector {
	return NewPythonAppDetector(config.ContentTypePythonDash, []string{
		"dash", // also matches dash_core_components, dash_bio, etc.
	})
}

func NewStreamlitDetector() *PythonAppDetector {
	return NewPythonAppDetector(config.ContentTypePythonStreamlit, []string{
		"streamlit",
	})
}

func NewBokehDetector() *PythonAppDetector {
	return NewPythonAppDetector(config.ContentTypePythonBokeh, []string{
		"bokeh",
	})
}

func NewPyShinyDetector() *PythonAppDetector {
	return NewPythonAppDetector(config.ContentTypePythonShiny, []string{
		"shiny",
	})
}

func (d *PythonAppDetector) InferType(path util.Path) (*config.Config, error) {
	entrypoint, entrypointPath, err := d.InferEntrypoint(
		path, ".py", "main.py", "app.py", "streamlit_app.py", "api.py")

	if err != nil {
		return nil, err
	}
	if entrypoint == "" {
		// We didn't find a matching import
		return nil, nil
	}
	matches, err := d.FileHasPythonImports(entrypointPath, d.imports)
	if err != nil {
		return nil, err
	}
	if matches {
		cfg := config.New()
		cfg.Entrypoint = entrypoint
		cfg.Type = d.contentType
		// indicate that Python inspection is needed
		cfg.Python = &config.Python{}
		return cfg, nil
	}
	return nil, nil
}