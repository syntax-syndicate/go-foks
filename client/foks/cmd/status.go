// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package cmd

import (
	"encoding/json"

	"github.com/foks-proj/go-foks/client/agent"
	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/proto/lcl"
	"github.com/spf13/cobra"
)

type statusCmdConfig struct {
	forceYubiUnlock bool
}

func statusCmd(m libclient.MetaContext) *cobra.Command {
	var cfg statusCmdConfig
	cmd := &cobra.Command{
		Use:          "status",
		Short:        "show status of active users and agent",
		Long:         `Show status of active users and agent`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, arg []string) error {
			return RunStatus(m, cmd, &cfg, arg)
		},
	}
	cmd.Flags().BoolVarP(&cfg.forceYubiUnlock, "force-yubi-unlock", "y", false, "force yubikey unlock")
	return cmd
}

func JSONOutput(m libclient.MetaContext, o any) error {
	b, err := json.MarshalIndent(o, "", "    ")
	if err != nil {
		return err
	}
	m.G().UIs().Terminal.Printf("%+v\n", string(b))
	return nil
}

func RunStatus(m libclient.MetaContext, cmd *cobra.Command, cfg *statusCmdConfig, arg []string) error {
	opts := agent.StartupOpts{NeedUser: true}
	if cfg.forceYubiUnlock {
		opts.ForceYubiUnlock = true
	}
	return quickStartLambda(m, &opts, func(cli lcl.UserClient) error {
		res, err := cli.AgentStatus(m.Ctx())
		if err != nil {
			return err
		}
		return JSONOutput(m, &res)
	})
}

func init() {
	AddCmd(statusCmd)
}
