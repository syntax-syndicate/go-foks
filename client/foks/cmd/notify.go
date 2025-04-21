// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package cmd

import (
	"github.com/foks-proj/go-foks/client/agent"
	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/proto/lcl"
	"github.com/spf13/cobra"
)

func clearDeviceNag(m libclient.MetaContext, top *cobra.Command) {
	var reset bool
	ret := &cobra.Command{
		Use:          "clear-device-nag",
		Short:        "clear device nag",
		Long:         `Clear device nag`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, arg []string) error {
			return quickStartLambda(
				m,
				&agent.StartupOpts{},
				func(c lcl.GeneralClient) error {
					return c.ClearDeviceNag(m.Ctx(), !reset)
				})
		},
	}
	ret.Flags().BoolVar(&reset, "reset", false, "reset device nag ")
	top.AddCommand(ret)
}

func notifyCmd(m libclient.MetaContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "notifications",
		Aliases:      []string{"notif", "notify", "notifications", "notifs"},
		Short:        "notify operations",
		Long:         `Notify operations`,
		SilenceUsage: true,
	}
	clearDeviceNag(m, cmd)
	return cmd
}

func init() {
	AddCmd(notifyCmd)
}
