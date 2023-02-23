package commands

// Copyright (C) 2023 by Posit Software, PBC.

import (
	"connect-client/accounts"
	"connect-client/services"

	"github.com/rstudio/platform-lib/pkg/rslog"
)

type UIArgs struct {
	Interactive bool   `short:"i" help:"Launch a browser to show the UI."`
	Listen      string `help:"Network address to listen on." placeholder:"HOST[:PORT]"`
	AccessLog   bool   `help:"Log all HTTP requests."`
	TLSKeyFile  string `help:"Path to TLS private key file for the UI server."`
	TLSCertFile string `help:"Path to TLS certificate chain file for the UI server."`
}

type CommonArgs struct {
	Debug debugFlag `help:"Enable debug mode." env:"CONNECT_DEBUG"`
}

type CLIContext struct {
	Accounts   *accounts.AccountList
	LocalToken services.LocalToken
	Logger     rslog.Logger `kong:"-"`
}

func NewCLIContext(logger rslog.Logger) (*CLIContext, error) {
	accountList := accounts.NewAccountList(logger)
	err := accountList.Load()
	if err != nil {
		return nil, err
	}

	token, err := services.NewLocalToken()
	if err != nil {
		return nil, err
	}

	return &CLIContext{
		Accounts:   accountList,
		LocalToken: token,
		Logger:     logger,
	}, nil
}
