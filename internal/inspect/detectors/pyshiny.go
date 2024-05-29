package detectors

// Copyright (C) 2023 by Posit Software, PBC.

import (
	"fmt"
	"regexp"

	"github.com/posit-dev/publisher/internal/config"
	"github.com/posit-dev/publisher/internal/util"
)

type pyShinyDetector struct {
	inferenceHelper
}

func NewPyShinyDetector() *pyShinyDetector {
	return &pyShinyDetector{
		inferenceHelper: defaultInferenceHelper{},
	}
}

var shinyExpressImportRE = regexp.MustCompile(`(import\s+shiny.express)|(from\s+shiny.express\s+import)|(from\s+shiny\s+import.*\bexpress\b)`)

func hasShinyExpressImport(content string) bool {
	return shinyExpressImportRE.MatchString(content)
}

func fileHasShinyExpressImport(path util.AbsolutePath) (bool, error) {
	content, err := path.ReadFile()
	if err != nil {
		return false, err
	}
	return hasShinyExpressImport(string(content)), nil
}

var invalidPythonIdentifierRE = regexp.MustCompile(`(^[0-9]|[^A-Za-z0-9])`)

func shinyExpressEntrypoint(entrypoint string) string {

	safeEntrypoint := invalidPythonIdentifierRE.ReplaceAllStringFunc(entrypoint, func(match string) string {
		return fmt.Sprintf("_%x_", int(match[0]))
	})
	return "shiny.express.app:" + safeEntrypoint
}

func (d *pyShinyDetector) InferType(base util.AbsolutePath) ([]*config.Config, error) {
	var configs []*config.Config
	entrypointPaths, err := base.Glob("*.py")
	if err != nil {
		return nil, err
	}
	if len(entrypointPaths) == 0 {
		// We didn't find a matching filename
		return nil, nil
	}
	for _, entrypointPath := range entrypointPaths {
		matches, err := d.FileHasPythonImports(entrypointPath, []string{"shiny"})
		if err != nil {
			return nil, err
		}
		if !matches {
			// Not a PyShiny app
			return nil, nil
		}
		isShinyExpress, err := fileHasShinyExpressImport(entrypointPath)
		if err != nil {
			return nil, err
		}
		cfg := config.New()

		entrypoint, err := entrypointPath.Rel(base)
		if err != nil {
			return nil, err
		}
		if isShinyExpress {
			cfg.Entrypoint = shinyExpressEntrypoint(entrypoint.String())
		} else {
			cfg.Entrypoint = entrypoint.String()
		}
		cfg.Type = config.ContentTypePythonShiny
		// indicate that Python inspection is needed
		cfg.Python = &config.Python{}
		configs = append(configs, cfg)
	}
	return configs, nil
}
