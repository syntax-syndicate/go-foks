// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package simple_ui

import (
	"fmt"
	"time"

	"github.com/foks-proj/go-foks/client/libclient"
	proto "github.com/foks-proj/go-foks/proto/lib"
)

type SSOLoginUI struct {
	spinCh chan struct{}
	msg    string
}

func (s *SSOLoginUI) spinner(m libclient.MetaContext) {
	spinners := []string{
		"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
	}
	spinIdx := 0

	print := func() {
		fmt.Printf("\r%s %s", s.msg, spinners[spinIdx])
		spinIdx = (spinIdx + 1) % len(spinners)
	}

	print()
	for {
		select {
		case <-s.spinCh:
			return
		case <-time.After(100 * time.Millisecond):
			print()
		}
	}
}

func (s *SSOLoginUI) ShowSSOLoginURL(m libclient.MetaContext, url proto.URLString) error {
	fmt.Printf("🔗 Plese perform an SSO login via:\n")
	fmt.Printf("\n")
	fmt.Printf("     %s\n", url)
	fmt.Printf("\n")
	s.spinCh = make(chan struct{})
	s.msg = "..... (polling for login completion) .... "
	go s.spinner(m)
	return nil
}

func (s *SSOLoginUI) ShowSSOLoginResult(m libclient.MetaContext, res proto.SSOLoginRes, err error) error {
	close(s.spinCh)
	fmt.Printf("\r%s", s.msg)
	if err != nil {
		fmt.Printf(" ❌ SSO login failed: %v\n", err)
		return nil
	}
	fmt.Printf(" 🎉 success\n\n")
	fmt.Printf("    🏛️  Issuer: %s\n", res.Issuer)
	fmt.Printf("    📛 User  : %s\n", res.Username)
	fmt.Printf("    📧 Email : %s\n\n", res.Email)

	return nil
}

var _ libclient.SSOLoginUIer = &SSOLoginUI{}
