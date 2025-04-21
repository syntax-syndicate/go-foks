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

func runSSOLogin(m libclient.MetaContext, cmd *cobra.Command, arg []string) error {
	err := agent.Startup(m, agent.StartupOpts{NeedUser: true, NeedUnlockedUser: true})
	if err != nil {
		return err
	}
	gcli, cleanFn, err := m.G().ConnectToAgentCli(m.Ctx())
	if err != nil {
		return err
	}
	defer cleanFn()
	genCli := lcl.GeneralClient{Cli: gcli, ErrorUnwrapper: core.StatusToError}
	userCli := newClient[lcl.UserClient](m, gcli)
	sessId, err := genCli.NewSession(m.Ctx(), proto.UISessionType_SSOLogin)
	if err != nil {
		return err
	}
	flow, err := userCli.LoginStartSsoLoginFlow(m.Ctx(), sessId)
	if err != nil {
		return err
	}
	err = m.G().UIs().SSOLogin.ShowSSOLoginURL(m, flow.Url)
	if err != nil {
		return err
	}

	res, err := userCli.LoginWaitForSsoLogin(m.Ctx(), sessId)
	err2 := m.G().UIs().SSOLogin.ShowSSOLoginResult(m, res, err)

	if err != nil {
		return err
	}
	if err2 != nil {
		return err
	}
	return nil
}

func ssoCmd(m libclient.MetaContext) *cobra.Command {
	top := &cobra.Command{
		Use:          "sso",
		Short:        "SSO operations",
		Long:         `Single-Sign-On (SSO) operations`,
		SilenceUsage: true,
	}
	login := &cobra.Command{
		Use:          "login",
		Short:        "login via identity provider (IdP)",
		Long:         `Login via identity provider (IdP)`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, arg []string) error {
			return runSSOLogin(m, cmd, arg)
		},
	}
	top.AddCommand(login)
	return top
}

func init() {
	AddCmd(ssoCmd)
}
