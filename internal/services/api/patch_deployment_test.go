package api

// Copyright (C) 2023 by Posit Software, PBC.

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/rstudio/connect-client/internal/config"
	"github.com/rstudio/connect-client/internal/deployment"
	"github.com/rstudio/connect-client/internal/logging"
	"github.com/rstudio/connect-client/internal/publish"
	"github.com/rstudio/connect-client/internal/state"
	"github.com/rstudio/connect-client/internal/util"
	"github.com/rstudio/connect-client/internal/util/utiltest"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

type PatchDeploymentHandlerFuncSuite struct {
	utiltest.Suite
	cwd util.AbsolutePath
}

func TestPatchDeploymentHandlerFuncSuite(t *testing.T) {
	suite.Run(t, new(PatchDeploymentHandlerFuncSuite))
}

func (s *PatchDeploymentHandlerFuncSuite) SetupTest() {
	stateFactory = state.New
	publisherFactory = publish.NewFromState

	afs := afero.NewMemMapFs()
	cwd, err := util.Getwd(afs)
	s.Nil(err)
	s.cwd = cwd
	s.cwd.MkdirAll(0700)
}

func (s *PatchDeploymentHandlerFuncSuite) TestPatchDeploymentHandlerFunc() {
	log := logging.New()

	rec := httptest.NewRecorder()
	req, err := http.NewRequest("PATCH", "/api/deployments/myTargetName", nil)
	s.NoError(err)
	req = mux.SetURLVars(req, map[string]string{"name": "myTargetName"})

	path := deployment.GetDeploymentPath(s.cwd, "myTargetName")
	d := deployment.New()
	err = d.WriteFile(path)
	s.NoError(err)

	cfg := config.New()
	err = cfg.WriteFile(config.GetConfigPath(s.cwd, "myConfig"))
	s.NoError(err)

	req.Body = io.NopCloser(strings.NewReader(`{"configurationName": "myConfig"}`))

	handler := PatchDeploymentHandlerFunc(s.cwd, log)
	handler(rec, req)

	s.Equal(http.StatusOK, rec.Result().StatusCode)

	updated, err := deployment.FromFile(path)
	s.NoError(err)
	s.Equal("myConfig", updated.ConfigName)
}

func (s *PatchDeploymentHandlerFuncSuite) TestPatchDeploymentHandlerFuncBadJSON() {
	log := logging.New()

	rec := httptest.NewRecorder()
	req, err := http.NewRequest("PATCH", "/api/deployments/myTargetName", nil)
	s.NoError(err)
	req = mux.SetURLVars(req, map[string]string{"name": "myTargetName"})

	req.Body = io.NopCloser(strings.NewReader(`{"random": "123"}`))

	handler := PatchDeploymentHandlerFunc(s.cwd, log)
	handler(rec, req)
	s.Equal(http.StatusBadRequest, rec.Result().StatusCode)
}

func (s *PatchDeploymentHandlerFuncSuite) TestPatchDeploymentHandlerDeploymentNotFound() {
	log := logging.New()

	rec := httptest.NewRecorder()
	req, err := http.NewRequest("PATCH", "/api/deployments/myTargetName", nil)
	s.NoError(err)
	req = mux.SetURLVars(req, map[string]string{"name": "myTargetName"})

	req.Body = io.NopCloser(strings.NewReader(`{"configurationName": "myConfig"}`))

	handler := PatchDeploymentHandlerFunc(s.cwd, log)
	handler(rec, req)
	s.Equal(http.StatusNotFound, rec.Result().StatusCode)
}

func (s *PatchDeploymentHandlerFuncSuite) TestPatchDeploymentHandlerConfigNotFound() {
	log := logging.New()

	rec := httptest.NewRecorder()
	req, err := http.NewRequest("PATCH", "/api/deployments/myTargetName", nil)
	s.NoError(err)
	req = mux.SetURLVars(req, map[string]string{"name": "myTargetName"})

	d := deployment.New()
	err = d.WriteFile(deployment.GetDeploymentPath(s.cwd, "myTargetName"))
	s.NoError(err)

	req.Body = io.NopCloser(strings.NewReader(`{"configurationName": "myConfig"}`))

	handler := PatchDeploymentHandlerFunc(s.cwd, log)
	handler(rec, req)
	s.Equal(http.StatusUnprocessableEntity, rec.Result().StatusCode)
}

func (s *PatchDeploymentHandlerFuncSuite) TestPatchDeploymentHandlerBadDeploymentFile() {
	log := logging.New()

	rec := httptest.NewRecorder()
	req, err := http.NewRequest("PATCH", "/api/deployments/myTargetName", nil)
	s.NoError(err)
	req = mux.SetURLVars(req, map[string]string{"name": "myTargetName"})

	path := deployment.GetDeploymentPath(s.cwd, "myTargetName")
	err = path.WriteFile([]byte(`invalid_config=1\n`), 0666)
	s.NoError(err)

	cfg := config.New()
	err = cfg.WriteFile(config.GetConfigPath(s.cwd, "myConfig"))
	s.NoError(err)

	req.Body = io.NopCloser(strings.NewReader(`{"configurationName": "myConfig"}`))

	handler := PatchDeploymentHandlerFunc(s.cwd, log)
	handler(rec, req)
	s.Equal(http.StatusUnprocessableEntity, rec.Result().StatusCode)
}
