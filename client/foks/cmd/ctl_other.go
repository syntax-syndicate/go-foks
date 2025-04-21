// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

//go:build !darwin && !linux && !windows

package cmd

import (
	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/lib/core"
	"github.com/spf13/cobra"
)

func RunCtlStart(m libclient.MetaContext, cmd *cobra.Command, arg []string) error {
	return core.NotImplementedError{}
}

func RunCtlStatus(m libclient.MetaContext, cmd *cobra.Command, arg []string) error {
	return core.NotImplementedError{}
}

func RunCtlStop(m libclient.MetaContext, cmd *cobra.Command, arg []string) error {
	return core.NotImplementedError{}
}

func RunCtlRestart(m libclient.MetaContext, cmd *cobra.Command, arg []string) error {
	return core.NotImplementedError{}
}

func doDaemonize(m libclient.MetaContext) error {
	return core.NotImplementedError{}
}

func AddPlatformCtlCommands(m libclient.MetaContext, cmd *cobra.Command) {
}
