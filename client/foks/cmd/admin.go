// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package cmd

import (
	"github.com/foks-proj/go-foks/client/agent"
	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/lcl"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/spf13/cobra"
)

var adminOpts = agent.StartupOpts{
	NeedUser:         true,
	NeedUnlockedUser: true,
}

func adminWebCmd(
	m libclient.MetaContext,
) *cobra.Command {
	return &cobra.Command{
		Use:          "web",
		Short:        "create a login-link for Web admin panel",
		Long:         "Create a login-link for Web admin panel",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return quickStartLambda(m, &adminOpts, func(cli lcl.AdminClient) error {
				res, err := cli.WebAdminPanelLink(m.Ctx())
				if err != nil {
					return err
				}
				if m.G().Cfg().JSONOutput() {
					return JSONOutput(m, res)
				}
				m.G().UIs().Terminal.Printf("Log in to Web Admin Panel via: %s\n", res)
				return nil
			})
		},
	}
}

func adminCheckLinkCmd(
	m libclient.MetaContext,
) *cobra.Command {
	return &cobra.Command{
		Use:          "check-link",
		Short:        "check a link for validity",
		Long:         "Check a link for validity; that the session is active and legitimate for the given user",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return core.BadArgsError("expected exactly one argument -- the URL to check")
			}
			return quickStartLambda(m, &adminOpts, func(cli lcl.AdminClient) error {
				err := cli.CheckLink(m.Ctx(), proto.URLString(args[0]))
				if err != nil {
					return err
				}
				return nil
			})
		},
	}
}

func adminCmd(
	m libclient.MetaContext,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "admin",
		Short:        "admin operations",
		SilenceUsage: true,
	}
	cmd.AddCommand(adminWebCmd(m))
	cmd.AddCommand(adminCheckLinkCmd(m))
	return cmd
}

func init() {
	AddCmd(adminCmd)
}
