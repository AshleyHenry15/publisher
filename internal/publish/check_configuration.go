package publish

// Copyright (C) 2023 by Posit Software, PBC.

import (
	"fmt"

	"github.com/rstudio/connect-client/internal/clients/connect"
	"github.com/rstudio/connect-client/internal/events"
	"github.com/rstudio/connect-client/internal/logging"
	"github.com/rstudio/connect-client/internal/types"
)

type checkConfigurationStartData struct{}
type checkConfigurationSuccessData struct{}

func (p *defaultPublisher) checkConfiguration(client connect.APIClient, log logging.Logger) error {
	op := events.PublishCheckCapabilitiesOp
	log = log.WithArgs(logging.LogKeyOp, op)

	p.emitter.Emit(events.New(op, events.StartPhase, events.NoError, checkConfigurationStartData{}))
	log.Info("Checking configuration against server capabilities")

	user, err := client.TestAuthentication(log)
	if err != nil {
		return types.OperationError(op, err)
	}
	log.Info("Publishing with credentials", "username", user.Username, "email", user.Email)

	if p.Target != nil && p.Target.Configuration != nil {
		previousType := p.Target.Configuration.Type
		currentType := p.Config.Type

		if currentType != previousType {
			return types.OperationError(op, fmt.Errorf("configuration type changed from %s to %s", previousType, currentType))
		}
	}
	err = client.CheckCapabilities(p.Dir, p.Config, log)
	if err != nil {
		return types.OperationError(op, err)
	}

	log.Info("Configuration OK")
	p.emitter.Emit(events.New(op, events.SuccessPhase, events.NoError, checkConfigurationSuccessData{}))
	return nil
}
