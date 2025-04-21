// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package cmd

import (
	"github.com/foks-proj/go-foks/client/agent"
	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/proto/lcl"
	"github.com/spf13/cobra"
)

func clearCmd(m libclient.MetaContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "clear",
		Short:        "clear all memory-resident keys",
		Long:         `Secret key material can be stored in-memory; clear it and remove all active users`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, arg []string) error {
			return RunClear(m, cmd, arg)
		},
	}
	return cmd
}

func RunClear(m libclient.MetaContext, cmd *cobra.Command, arg []string) error {
	return quickStartLambda(m, &agent.StartupOpts{}, func(cli lcl.UserClient) error {
		err := cli.Clear(m.Ctx())
		if err != nil {
			return err
		}
		return nil
	})
}

func init() {
	AddCmd(clearCmd)
}
