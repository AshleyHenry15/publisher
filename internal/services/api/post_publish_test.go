package api

// Copyright (C) 2023 by Posit Software, PBC.

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rstudio/connect-client/internal/accounts"
	"github.com/rstudio/connect-client/internal/logging"
	"github.com/rstudio/connect-client/internal/publish"
	"github.com/rstudio/connect-client/internal/state"
	"github.com/rstudio/connect-client/internal/util"
	"github.com/rstudio/connect-client/internal/util/utiltest"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type PublishHandlerFuncSuite struct {
	utiltest.Suite
}

func TestPublishHandlerFuncSuite(t *testing.T) {
	suite.Run(t, new(PublishHandlerFuncSuite))
}

type mockPublisher struct {
	mock.Mock
}

func (m *mockPublisher) PublishDirectory(log logging.Logger) error {
	args := m.Called(log)
	return args.Error(0)
}

func (s *PublishHandlerFuncSuite) TestPublishHandlerFunc() {
	stateStore := state.Empty()
	oldID := stateStore.LocalID
	log := logging.New()

	rec := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/api/publish", nil)
	s.NoError(err)

	lister := &accounts.MockAccountList{}
	req.Body = io.NopCloser(strings.NewReader("{\"account\":\"local\", \"config\":\"default\"}"))

	publisher := &mockPublisher{}
	publisher.On("PublishDirectory", mock.Anything).Return(nil)
	publisherFactory := func(*state.State) publish.Publisher {
		return publisher
	}
	stateFactory := func(
		path util.Path,
		accountName, configName, targetID string,
		accountList accounts.AccountList) (*state.State, error) {
		return state.Empty(), nil
	}
	handler := PostPublishHandlerFunc(stateStore, util.Path{}, log, lister, stateFactory, publisherFactory)
	handler(rec, req)

	s.Equal(http.StatusAccepted, rec.Result().StatusCode)
	s.Equal("application/json", rec.Header().Get("content-type"))

	res := &PostPublishReponse{}
	dec := json.NewDecoder(rec.Body)
	dec.DisallowUnknownFields()
	s.NoError(dec.Decode(res))

	s.NotEqual(state.LocalDeploymentID(""), stateStore.LocalID)
	s.NotEqual(oldID, stateStore.LocalID)
	s.Equal(stateStore.LocalID, res.LocalID)
}
