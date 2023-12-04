package initialize

// Copyright (C) 2023 by Posit Software, PBC.

import (
	"fmt"

	"github.com/rstudio/connect-client/internal/bundles"
	"github.com/rstudio/connect-client/internal/config"
	"github.com/rstudio/connect-client/internal/environment"
	"github.com/rstudio/connect-client/internal/inspect"
	"github.com/rstudio/connect-client/internal/logging"
	"github.com/rstudio/connect-client/internal/util"
)

var ContentDetectorFactory = inspect.NewContentTypeDetector
var PythonInspectorFactory = environment.NewPythonInspector

func inspectProjectType(path util.Path, log logging.Logger) (*inspect.ContentType, error) {
	log.Info("Detecting deployment type and entrypoint...")
	typeDetector := ContentDetectorFactory()
	contentType, err := typeDetector.InferType(path)
	if err != nil {
		return nil, fmt.Errorf("error detecting content type: %w", err)
	}
	log.Info("Deployment type", "Entrypoint", contentType.Entrypoint, "AppMode", contentType.Type)
	return contentType, nil
}

func requiresPython(contentType config.ContentType, path util.Path, python util.Path) (bool, error) {
	if contentType.IsPythonContent() {
		return true, nil
	}
	if python.Path() != "" {
		return true, nil
	}
	// Presence of requirements.txt implies Python is needed.
	// This is the preferred approach since it is unambiguous and
	// doesn't rely on environment inspection.
	requirementsPath := path.Join(bundles.PythonRequirementsFilename)
	exists, err := requirementsPath.Exists()
	if err != nil {
		return false, err
	}
	return exists, nil
}

func inspectPython(path util.Path, python util.Path, log logging.Logger) (*config.Python, error) {
	cfg := &config.Python{}
	inspector := PythonInspectorFactory(path, python, log)
	version, err := inspector.GetPythonVersion()
	if err != nil {
		return nil, err
	}
	cfg.Version = version

	err = inspector.EnsurePythonRequirementsFile()
	if err != nil {
		return nil, err
	}
	// Package list will be written to requirements.txt, if not already present.
	cfg.PackageManager = "pip"
	cfg.PackageFile = bundles.PythonRequirementsFilename
	return cfg, nil
}

func Init(path util.Path, configName string, python util.Path, log logging.Logger) (*config.Config, error) {
	if configName == "" {
		configName = config.DefaultConfigName
	}
	cfg := config.New()
	contentType, err := inspectProjectType(path, log)
	if err != nil {
		return nil, err
	}
	cfg.Type = contentType.Type
	cfg.Entrypoint = contentType.Entrypoint

	requiresPython, err := requiresPython(contentType.Type, path, python)
	if err != nil {
		return nil, err
	}
	if requiresPython {
		pythonConfig, err := inspectPython(path, python, log)
		if err != nil {
			return nil, err
		}
		cfg.Python = pythonConfig
	}
	configPath := config.GetConfigPath(path, configName)
	err = cfg.WriteFile(configPath)
	log.Info("Wrote config file", "path", configPath)

	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// InitIfNeeded runs an auto-initialize if the specified config file does not exist.
func InitIfNeeded(path util.Path, configName string, log logging.Logger) error {
	configPath := config.GetConfigPath(path, configName)
	exists, err := configPath.Exists()
	if err != nil {
		return err
	}
	if !exists {
		log.Info("Configuration file does not exist; creating it", "path", configPath.String())
		_, err = Init(path, configName, util.Path{}, log)
		if err != nil {
			return err
		}
	}
	return nil
}