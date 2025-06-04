package main

import (
	"fmt"

	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/server/shared"
	"github.com/spf13/cobra"
)

type VersionCmd struct {
	CLIAppBase
}

func (v *VersionCmd) CobraConfig() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version of the FOKS tool",
	}
}

func (v *VersionCmd) CheckArgs(args []string) error {
	if len(args) != 0 {
		return core.BadArgsError("no args allowed")
	}
	return nil
}

func (v *VersionCmd) Run(m shared.MetaContext) error {
	var lv string
	if LinkerVersion != "unknown" {
		lv = fmt.Sprintf(" (linked as: %s)", LinkerVersion)
	}
	fmt.Printf("%s%s\n",
		core.CurrentSoftwareVersion.String(),
		lv,
	)
	return nil
}

func (v *VersionCmd) SetGlobalContext(g *shared.GlobalContext) {}
func (v *VersionCmd) TweakOpts(opts *shared.GlobalCLIConfigOpts) {
	opts.SkipNetwork = true
	opts.SkipConfig = true
	opts.NoStartupMsg = true
}

var _ shared.CLIApp = (*VersionCmd)(nil)

func init() {
	AddCmd(&VersionCmd{})
}
