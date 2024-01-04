package commands

// Copyright (C) 2023 by Posit Software, PBC.

import (
	"fmt"
	"os"

	"github.com/rstudio/connect-client/internal/accounts"
	"github.com/rstudio/connect-client/internal/cli_types"
	"github.com/rstudio/connect-client/internal/config"
	"github.com/rstudio/connect-client/internal/deployment"
	"github.com/rstudio/connect-client/internal/events"
	"github.com/rstudio/connect-client/internal/initialize"
	"github.com/rstudio/connect-client/internal/publish"
	"github.com/rstudio/connect-client/internal/state"
	"github.com/rstudio/connect-client/internal/util"
)

type CreateCmd struct {
	Path        util.Path         `help:"Path to project directory containing files to publish." arg:"" default:"."`
	AccountName string            `name:"account" short:"a" help:"Nickname of the publishing account to use (run list-accounts to see them)."`
	ConfigName  string            `name:"config" short:"c" help:"Configuration name (in .posit/publish/)"`
	SaveName    string            `name:"name" short:"n" help:"Save deployment with this name (in .posit/deployments/)"`
	Account     *accounts.Account `kong:"-"`
	Config      *config.Config    `kong:"-"`
}

func (cmd *CreateCmd) Run(args *cli_types.CommonArgs, ctx *cli_types.CLIContext) error {
	ctx.Logger = events.NewSimpleLogger(args.Verbose, os.Stderr)

	if cmd.SaveName != "" {
		err := util.ValidateFilename(cmd.SaveName)
		if err != nil {
			return err
		}
		exists, err := deployment.GetDeploymentPath(cmd.Path, cmd.SaveName).Exists()
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("there is already a deployment named '%s'; did you mean to use the 'redeploy' command?", cmd.SaveName)
		}
	}
	err := initialize.InitIfNeeded(cmd.Path, cmd.ConfigName, ctx.Logger)
	if err != nil {
		return err
	}
	stateStore, err := state.New(cmd.Path, cmd.AccountName, cmd.ConfigName, "", cmd.SaveName, ctx.Accounts)
	if err != nil {
		return err
	}
	fmt.Printf("Deploy to server %s using account %s and configuration %s, creating deployment %s\n",
		stateStore.Account.URL,
		stateStore.Account.Name,
		stateStore.ConfigName,
		stateStore.SaveName)
	publisher := publish.NewFromState(stateStore)
	return publisher.PublishDirectory(ctx.Logger)
}
