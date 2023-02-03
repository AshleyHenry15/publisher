package commands

// Copyright (C) 2023 by Posit Software, PBC.

import (
	"fmt"

	"connect-client/project"

	"github.com/alecthomas/kong"
)

type VersionFlag bool

func (v VersionFlag) BeforeReset(cli *kong.Kong) error {
	fmt.Println(project.Version)
	cli.Exit(0)
	return nil
}
