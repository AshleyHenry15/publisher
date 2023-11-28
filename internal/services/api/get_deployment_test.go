package api

// Copyright (C) 2023 by Posit Software, PBC.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rstudio/connect-client/internal/state"
	"github.com/rstudio/connect-client/internal/util/utiltest"
	"github.com/stretchr/testify/suite"
)

type GetDeploymentHandlerFuncSuite struct {
	utiltest.Suite
}

func TestGetDeploymentHandlerFuncSuite(t *testing.T) {
	suite.Run(t, new(GetDeploymentHandlerFuncSuite))
}

func (s *GetDeploymentHandlerFuncSuite) TestGetDeploymentHandlerFunc() {
	deploymentsService := new(MockDeploymentsService)
	deploymentsService.On("GetDeployment").Return(nil, nil)
	h := GetDeploymentHandlerFunc(deploymentsService)
	s.NotNil(h)
}

func (s *GetDeploymentHandlerFuncSuite) TestGetDeploymentHandler() {
	src := state.OldDeploymentFromState(state.Empty())
	deploymentsService := new(MockDeploymentsService)
	deploymentsService.On("GetDeployment").Return(src)
	h := GetDeploymentHandlerFunc(deploymentsService)

	rec := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "", nil)
	s.NoError(err)

	h(rec, req)

	s.Equal(http.StatusOK, rec.Result().StatusCode)
	s.Equal("application/json", rec.Header().Get("content-type"))

	res := &state.OldDeployment{}
	dec := json.NewDecoder(rec.Body)
	dec.DisallowUnknownFields()
	s.NoError(dec.Decode(res))

	s.Equal(src, res)

}
