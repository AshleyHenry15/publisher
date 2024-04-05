package publish

// Copyright (C) 2023 by Posit Software, PBC.

import (
	"github.com/rstudio/connect-client/internal/clients/connect"
	"github.com/rstudio/connect-client/internal/events"
	"github.com/rstudio/connect-client/internal/logging"
	"github.com/rstudio/connect-client/internal/types"
)

type checkConfigurationStartData struct{}
type checkConfigurationSuccessData struct{}

func (p *defaultPublisher) checkConfiguration(client connect.APIClient, log logging.Logger) *types.AgentError {
	op := events.PublishCheckCapabilitiesOp
	log = log.WithArgs(logging.LogKeyOp, op)

	p.emitter.Emit(events.New(op, events.StartPhase, types.NoError, checkConfigurationStartData{}))
	log.Info("Checking configuration against server capabilities")

	user, agentErr := client.TestAuthentication(log)
	if agentErr != nil {
		agentErr.SetOperation(op)
		return agentErr
	}
	log.Info("Publishing with credentials", "username", user.Username, "email", user.Email)

	err := client.CheckCapabilities(p.Dir, p.Config, log)
	if err != nil {
		return types.AsAgentErrForOperation(op, err)
	}

	log.Info("Configuration OK")
	p.emitter.Emit(events.New(op, events.SuccessPhase, types.NoError, checkConfigurationSuccessData{}))
	return nil
}
