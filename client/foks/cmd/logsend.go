package cmd

import (
	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/client/libterm"
	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/lcl"
	"github.com/spf13/cobra"
)

func logsendCmd(m libclient.MetaContext) *cobra.Command {
	var nLogs int
	ret := &cobra.Command{
		Use:   "logsend",
		Short: "Send logs to FOKS developers",
		Long: libterm.MustRewrapSense(`Send FOKS logs to the developer team to
help debug. FOKS logs do not comatin any sensitive data, but we ask users to check them
before sending, just to be sure.`, 0),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogSend(m, args, nLogs)
		},
	}
	ret.Flags().IntVarP(&nLogs, "num-logs", "n", 3, "max number of logs to send")
	return ret
}

func runLogSend(m libclient.MetaContext, args []string, nLogs int) error {
	cli, clean, err := quickStart[lcl.LogSendClient](m, nil)
	if err != nil {
		return err
	}
	defer clean()
	if len(args) != 0 {
		return ArgsError("no args allowed")
	}
	if nLogs < 1 {
		return ArgsError("num-logs must be at least 1")
	}
	if nLogs > 30 {
		return ArgsError("num-logs must be at most 30")
	}
	sl, err := cli.LogSendList(m.Ctx(), uint64(nLogs))
	if err != nil {
		return err
	}
	if len(sl.Files) == 0 {
		return core.NotFoundError("no logs found")
	}
	lsui := m.G().UIs().LogSend
	err = lsui.ApproveLogs(m, sl)
	if err != nil {
		return err
	}
	_ = lsui.ShowStartSend(m)
	res, err := cli.LogSend(m.Ctx(), sl)
	_ = lsui.ShowCompleteSend(m, err)
	if err != nil {
		return err
	}
	err = lsui.ShowLogSendRes(m, res)
	if err != nil {
		return err
	}
	return nil
}

func init() {
	AddCmd(logsendCmd)
}
