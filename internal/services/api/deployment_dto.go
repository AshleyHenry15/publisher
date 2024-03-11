package api

// Copyright (C) 2023 by Posit Software, PBC.

import (
	"github.com/rstudio/connect-client/internal/accounts"
	"github.com/rstudio/connect-client/internal/config"
	"github.com/rstudio/connect-client/internal/deployment"
	"github.com/rstudio/connect-client/internal/types"
	"github.com/rstudio/connect-client/internal/util"
)

type deploymentState string

const (
	deploymentStateNew      deploymentState = "new"
	deploymentStateDeployed deploymentState = "deployed"
	deploymentStateError    deploymentState = "error"
)

type deploymentLocation struct {
	State deploymentState `json:"state"`
	Name  string          `json:"deploymentName"`
	Path  string          `json:"deploymentPath"`
}

type preDeploymentDTO struct {
	deploymentLocation
	Schema     string              `json:"schema"`
	ServerType accounts.ServerType `json:"serverType"`
	ServerURL  string              `json:"serverUrl"`
	SaveName   string              `json:"saveName"`
	CreatedAt  string              `json:"createdAt"`
	Error      *types.AgentError   `json:"error,omitempty"`
}

type fullDeploymentDTO struct {
	deploymentLocation
	deployment.Deployment
	ConfigPath string `json:"configurationPath,omitempty"`
	SaveName   string `json:"saveName"`
}

type deploymentErrorDTO struct {
	deploymentLocation
	Error *types.AgentError `json:"error,omitempty"`
}

func getConfigPath(base util.Path, configName string) util.Path {
	if configName == "" {
		return util.Path{}
	}
	configPath := config.GetConfigPath(base, configName)
	relConfigPath, err := configPath.Rel(base)
	if err != nil {
		// This error should never happen. But, if it does,
		// still return as much data as we can.
		return configPath
	}
	return relConfigPath
}

func deploymentAsDTO(d *deployment.Deployment, err error, base util.Path, path util.Path) any {
	saveName := deployment.SaveNameFromPath(path)
	if err != nil {
		return &deploymentErrorDTO{
			deploymentLocation: deploymentLocation{
				State: deploymentStateError,
				Name:  saveName,
				Path:  path.String(),
			},
			Error: types.AsAgentError(err),
		}
	} else if d.ID != "" {
		return &fullDeploymentDTO{
			deploymentLocation: deploymentLocation{
				State: deploymentStateDeployed,
				Name:  saveName,
				Path:  path.String(),
			},
			Deployment: *d,
			ConfigPath: getConfigPath(base, d.ConfigName).String(),
			SaveName:   saveName, // TODO: remove this duplicate (remove frontend references first)
		}
	} else {
		return preDeploymentDTO{
			deploymentLocation: deploymentLocation{
				State: deploymentStateNew,
				Name:  saveName,
				Path:  path.String(),
			},
			Schema:     d.Schema,
			ServerType: d.ServerType,
			ServerURL:  d.ServerURL,
			SaveName:   saveName, // TODO: remove this duplicate (remove frontend references first)
			CreatedAt:  d.CreatedAt,
			Error:      d.Error,
		}
	}
}
