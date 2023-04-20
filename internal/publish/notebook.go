package publish

// Copyright (C) 2023 by Posit Software, PBC.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/rstudio/connect-client/internal/publish/apptypes"
	"github.com/spf13/afero"
)

type NotebookDetector struct{}

var voilaImportNames = []string{
	"ipywidgets",
	// From the Voila example notebooks
	"bqplot",
	"ipympl",
	"ipyvolume",
	// Other widget packages from PyPI
	"ipyspeck",
	"ipywebgl",
	"ipywebrtc",
}

func (d *NotebookDetector) InferType(fs afero.Fs, path string) (*ContentType, error) {
	entrypoint, err := inferEntrypoint(fs, path, ".ipynb", "index.ipynb")
	if err != nil {
		return nil, err
	}
	if entrypoint != "" {
		code, err := getNotebookFileInputs(fs, entrypoint)
		if err != nil {
			return nil, err
		}
		isVoila, err := hasPythonImports(strings.NewReader(code), voilaImportNames)
		if err != nil {
			return nil, err
		}
		t := &ContentType{
			entrypoint: entrypoint,
			runtimes:   []Runtime{PythonRuntime},
		}
		if isVoila {
			t.appMode = apptypes.JupyterVoilaMode
		} else {
			t.appMode = apptypes.StaticJupyterMode
		}
		return t, nil
	}
	return nil, nil
}

func getNotebookFileInputs(fs afero.Fs, path string) (string, error) {
	f, err := fs.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return getNotebookInputs(f)
}

var errNoCellsInNotebook = errors.New("No cells found in notebook")

func getNotebookInputs(r io.Reader) (string, error) {
	decoder := json.NewDecoder(r)

	type jsonObject = map[string]any
	var notebookContents jsonObject
	err := decoder.Decode(&notebookContents)
	if err != nil {
		return "", err
	}
	cells, ok := notebookContents["cells"].([]any)
	if !ok || len(cells) == 0 {
		return "", errNoCellsInNotebook
	}
	combinedSource := []string{}
	for cellNum, rawCell := range cells {
		cell, ok := rawCell.(jsonObject)
		if !ok {
			return "", fmt.Errorf("Notebook cell %d is not an object", cellNum)
		}
		cellType, ok := cell["cell_type"].(string)
		if !ok {
			return "", fmt.Errorf("Notebook cell %d is missing cell_type", cellNum)
		}
		if cellType == "code" {
			sourceLines, ok := cell["source"].([]any)
			if !ok {
				return "", fmt.Errorf("Notebook cell %d has an invalid source", cellNum)
			}
			for lineNum, rawLine := range sourceLines {
				line, ok := rawLine.(string)
				if !ok {
					return "", fmt.Errorf("Notebook cell %d line %d is not a string", cellNum, lineNum)
				}
				combinedSource = append(combinedSource, line)
			}
		}
	}
	return strings.Join(combinedSource, ""), nil
}
