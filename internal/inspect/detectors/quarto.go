package detectors

// Copyright (C) 2023 by Posit Software, PBC.

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/rstudio/connect-client/internal/config"
	"github.com/rstudio/connect-client/internal/executor"
	"github.com/rstudio/connect-client/internal/logging"
	"github.com/rstudio/connect-client/internal/util"
)

type QuartoDetector struct {
	inferenceHelper
	executor executor.Executor
	log      logging.Logger
}

func NewQuartoDetector() *QuartoDetector {
	return &QuartoDetector{
		inferenceHelper: defaultInferenceHelper{},
		executor:        executor.NewExecutor(),
		log:             logging.New(),
	}
}

type quartoInspectOutput struct {
	// Only the fields we use are included; the rest
	// are discarded by the JSON decoder.
	Quarto struct {
		Version string `json:"version"`
	} `json:"quarto"`
	Config struct {
		Project struct {
			Title      string   `json:"title"`
			PreRender  []string `json:"pre-render"`
			PostRender []string `json:"post-render"`
		} `json:"project"`
		Website struct {
			Title string `json:"title"`
		} `json:"website"`
	} `json:"config"`
	Engines []string `json:"engines"`
	Files   struct {
		Input []string `json:"input"`
	} `json:"files"`
	// For single quarto docs without _quarto.yml
	Formats struct {
		HTML struct {
			Metadata struct {
				Title string `json:"title"`
			} `json:"metadata"`
		} `json:"html"`
	} `json:"formats"`
}

func (d *QuartoDetector) quartoInspect(path util.AbsolutePath) (*quartoInspectOutput, error) {
	args := []string{"inspect", path.String()}
	out, _, err := d.executor.RunCommand("quarto", args, util.AbsolutePath{}, d.log)
	if err != nil {
		return nil, fmt.Errorf("quarto inspect failed: %w", err)
	}
	var inspectOutput quartoInspectOutput
	err = json.Unmarshal(out, &inspectOutput)
	if err != nil {
		return nil, fmt.Errorf("couldn't decode quarto inspect output: %w", err)
	}
	return &inspectOutput, nil
}

func (d *QuartoDetector) needsPython(inspectOutput *quartoInspectOutput) bool {
	if slices.Contains(inspectOutput.Engines, "jupyter") {
		return true
	}
	for _, script := range inspectOutput.Config.Project.PreRender {
		if strings.HasSuffix(script, ".py") {
			return true
		}
	}
	for _, script := range inspectOutput.Config.Project.PostRender {
		if strings.HasSuffix(script, ".py") {
			return true
		}
	}
	return false
}

func (d *QuartoDetector) needsR(inspectOutput *quartoInspectOutput) bool {
	if slices.Contains(inspectOutput.Engines, "knitr") {
		return true
	}
	for _, script := range inspectOutput.Config.Project.PreRender {
		if strings.HasSuffix(script, ".R") {
			return true
		}
	}
	for _, script := range inspectOutput.Config.Project.PostRender {
		if strings.HasSuffix(script, ".R") {
			return true
		}
	}
	return false
}

func (d *QuartoDetector) getTitle(inspectOutput *quartoInspectOutput) string {
	if inspectOutput.Config.Website.Title != "" {
		return inspectOutput.Config.Website.Title
	}
	if inspectOutput.Formats.HTML.Metadata.Title != "" {
		return inspectOutput.Formats.HTML.Metadata.Title
	}
	if inspectOutput.Config.Project.Title != "" {
		return inspectOutput.Config.Project.Title
	}
	return ""
}

var quartoSuffixes = []string{".qmd", ".Rmd", ".ipynb"}

func (d *QuartoDetector) findEntrypoints(base util.AbsolutePath) ([]util.AbsolutePath, error) {
	allPaths := []util.AbsolutePath{}

	for _, suffix := range quartoSuffixes {
		entrypointPaths, err := base.Glob("*" + suffix)
		if err != nil {
			return nil, err
		}
		allPaths = append(allPaths, entrypointPaths...)
	}
	return allPaths, nil
}

func (d *QuartoDetector) InferType(base util.AbsolutePath) ([]*config.Config, error) {
	var configs []*config.Config
	entrypointPaths, err := d.findEntrypoints(base)
	if err != nil {
		return nil, err
	}
	if len(entrypointPaths) == 0 {
		return nil, nil
	}
	isQuartoProject, err := base.Join("_quarto.yml").Exists()
	if err != nil {
		return nil, err
	}
	for _, entrypointPath := range entrypointPaths {
		var inspectOutput *quartoInspectOutput
		if isQuartoProject {
			inspectOutput, err = d.quartoInspect(base)
		} else {
			inspectOutput, err = d.quartoInspect(entrypointPath)
		}
		if err != nil {
			// Maybe this isn't really a quarto project, or maybe the user doesn't have quarto.
			// We log this error and return nil so other inspectors can have a shot at it.
			d.log.Warn("quarto inspect failed", "error", err)
			continue
		}
		if inspectOutput.Files.Input != nil && len(inspectOutput.Files.Input) == 0 {
			continue
		}
		entrypoint, err := entrypointPath.Rel(base)
		if err != nil {
			return nil, err
		}
		cfg := config.New()
		cfg.Type = config.ContentTypeQuarto
		cfg.Entrypoint = entrypoint.String()
		cfg.Title = d.getTitle(inspectOutput)

		cfg.Quarto = &config.Quarto{
			Version: inspectOutput.Quarto.Version,
			Engines: inspectOutput.Engines,
		}
		if d.needsPython(inspectOutput) {
			// Indicate that Python inspection is needed.
			cfg.Python = &config.Python{}
		}
		if d.needsR(inspectOutput) {
			// Indicate that R inspection is needed.
			cfg.R = &config.R{}
		}
		configs = append(configs, cfg)
	}
	return configs, nil
}
