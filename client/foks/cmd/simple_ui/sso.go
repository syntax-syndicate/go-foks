// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package simple_ui

import (
	"fmt"

	"github.com/foks-proj/go-foks/client/libclient"
	proto "github.com/foks-proj/go-foks/proto/lib"
)

type SSOLoginUI struct {
	spinner *Spinner
}

func (s *SSOLoginUI) ShowSSOLoginURL(m libclient.MetaContext, url proto.URLString) error {
	fmt.Printf("🔗 Plese perform an SSO login via:\n")
	fmt.Printf("\n")
	fmt.Printf("     %s\n", url)
	fmt.Printf("\n")
	s.spinner = NewSpinner("..... (polling for login completion) .... ")
	s.spinner.Start()
	return nil
}

func (s *SSOLoginUI) ShowSSOLoginResult(m libclient.MetaContext, res proto.SSOLoginRes, err error) error {
	var msg string
	if err != nil {
		msg = fmt.Sprintf(" ❌ SSO login failed: %v", err)
	} else {
		msg = " 🎉 success\n"
	}
	s.spinner.Stop(msg)
	if err != nil {
		return nil
	}

	fmt.Printf("    🏛️  Issuer: %s\n", res.Issuer)
	fmt.Printf("    📛 User  : %s\n", res.Username)
	fmt.Printf("    📧 Email : %s\n\n", res.Email)

	return nil
}

var _ libclient.SSOLoginUIer = &SSOLoginUI{}
