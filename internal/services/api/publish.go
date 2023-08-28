package api

// Copyright (C) 2023 by Posit Software, PBC.

import (
	"net/http"

	"github.com/rstudio/connect-client/internal/accounts"
	"github.com/rstudio/connect-client/internal/cli_types"
	"github.com/rstudio/connect-client/internal/publish"
	"github.com/rstudio/platform-lib/pkg/rslog"
)

func PostPublishHandlerFunc(publishArgs *cli_types.PublishArgs, lister accounts.AccountList, logger rslog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodPost:
			err := publish.PublishManifestFiles(publishArgs, lister, logger)
			if err != nil {
				InternalError(w, logger, err)
			}
		default:
			return
		}
	}
}