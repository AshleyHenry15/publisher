package api

// Copyright (C) 2023 by Posit Software, PBC.

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rstudio/connect-client/internal/config"
	"github.com/rstudio/connect-client/internal/logging"
	"github.com/rstudio/connect-client/internal/schema"
	"github.com/rstudio/connect-client/internal/util"
)

func PutConfigurationHandlerFunc(base util.Path, log logging.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		name := mux.Vars(req)["name"]
		err := util.ValidateFilename(name)
		if err != nil {
			BadRequest(w, req, log, err)
			return
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			InternalError(w, req, log, err)
			return
		}

		// First, decode into a map for schema validation
		rawDecoder := json.NewDecoder(bytes.NewReader(body))
		var rawConfig map[string]any
		err = rawDecoder.Decode(&rawConfig)
		if err != nil {
			BadRequest(w, req, log, err)
			return
		}
		t, ok := rawConfig["type"]
		if ok && t == string(config.ContentTypeUnknown) {
			// We permit configurations with `unknown` type to be created,
			// even though they don't pass validation. Pass a known
			// type to the validator.
			rawConfig["type"] = string(config.ContentTypeHTML)
		}
		validator, err := schema.NewValidator[config.Config](schema.ConfigSchemaURL)
		if err != nil {
			InternalError(w, req, log, err)
			return
		}
		err = validator.ValidateContent(rawConfig)
		if err != nil {
			BadRequest(w, req, log, err)
			return
		}

		// Then decode into a Config to be written to file.
		dec := json.NewDecoder(bytes.NewReader(body))
		dec.DisallowUnknownFields()
		var cfg config.Config
		err = dec.Decode(&cfg)
		if err != nil {
			BadRequest(w, req, log, err)
			return
		}

		var response configDTO
		configPath := config.GetConfigPath(base, name)

		err = cfg.WriteFile(configPath)
		if err != nil {
			InternalError(w, req, log, err)
			return
		}
		response = configDTO{
			Name:          name,
			Path:          configPath.String(),
			Configuration: &cfg,
		}
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}