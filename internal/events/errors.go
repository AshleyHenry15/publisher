package events

// Copyright (C) 2023 by Posit Software, PBC.

import (
	"fmt"
	"time"

	"github.com/rstudio/connect-client/internal/types"
)

func NewAgentEvent(e types.EventableError) AgentEvent {
	data := e.GetData()
	data["msg"] = e.Error()

	return AgentEvent{
		Time: time.Now().UTC(),
		Type: fmt.Sprintf("%s/failure/%s", e.GetOperation(), e.GetCode()),
		Data: data,
	}
}