// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

//go:build windows

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/foks-proj/go-foks/client/agent"
	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/proto/lcl"
	"github.com/spf13/cobra"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const regKey = `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`
const agentName = "foksAgent"

func RunCtlStart(m libclient.MetaContext, _ *cobra.Command, arg []string) error {

	m = m.WithLogTag("ctl-start")

	err := agent.Startup(m, agent.StartupOpts{NoStandalone: true})
	if err != nil {
		return err
	}

	// Get path to current executable
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	// Add to registry for auto-start on login
	k, err := registry.OpenKey(registry.CURRENT_USER, regKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	cmd := fmt.Sprintf(`cmd.exe /C start "" "%s" ctl looper --daemonize`, exe)

	err = k.SetStringValue(agentName, cmd)
	if err != nil {
		return err
	}

	pid, err := pingAgent(m)
	if err == nil {
		m.G().UIs().Terminal.Printf("agent already running with pid=%d\n", pid)
		return nil
	}

	// Start the looper now
	command := exec.Command(exe, "ctl", "looper")
	err = command.Start()
	if err != nil {
		return err
	}

	prnt := m.G().UIs().Terminal.Printf
	prnt("Started new FOKS supervisor; pid=%d\n", command.Process.Pid)

	// Wait for the agent to start; not perfect, we might reconsider this.
	time.Sleep(2 * time.Second)

	pid, err = pingAgent(m)
	if err != nil {
		return err
	}
	prnt("Agent running with pid=%d\n", pid)
	prnt("The FOKS agent is now running in the background and will start automatically on login\n")
	prnt("If needed,`foks ctl stop` stops and uninstalls the agent\n")
	return nil
}

func RunCtlStop(m libclient.MetaContext, cmd *cobra.Command, arg []string) error {
	m = m.WithLogTag("ctl-stop")

	err := setupCtlCmd(m)
	if err != nil {
		return err
	}

	stopperFile, err := m.G().Cfg().GetAgentStopperFile()
	if err != nil {
		return err
	}

	k, err := registry.OpenKey(registry.CURRENT_USER, regKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	err = k.DeleteValue(agentName)
	if err != nil && err != registry.ErrNotExist {
		return err
	}

	err = stopperFile.Touch()
	if err != nil {
		return err
	}

	m.Infow("stop", "msg", "agent stopper file touched")
	return nil
}

func RunCtlStatus(m libclient.MetaContext, cmd *cobra.Command, arg []string) error {

	err := agent.Startup(m, agent.StartupOpts{NoStandalone: true})
	if err != nil {
		return err
	}
	pid, err := pingAgent(m)
	if err != nil {
		return err
	}
	m.G().UIs().Terminal.Printf("agent is running with pid=%d\n", pid)
	return nil
}

func pingAgent(m libclient.MetaContext) (uint64, error) {
	gcli, cleanFn, err := m.G().ConnectToAgentCli(m.Ctx())
	if err != nil {
		return 0, err
	}
	defer cleanFn()
	cli := newClient[lcl.CtlClient](m, gcli)
	pid, err := cli.PingAgent(m.Ctx())
	if err != nil {
		return 0, err
	}
	return pid, nil
}

func RunCtlRestart(m libclient.MetaContext, cmd *cobra.Command, arg []string) error {
	return RunCtlShutdown(m, cmd, arg)
}

func doDaemonize(m libclient.MetaContext) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	cmd := exec.Command(exe, "ctl", "looper")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	cmd.SysProcAttr = &windows.SysProcAttr{
		CreationFlags: windows.CREATE_NO_WINDOW, // 👈 Hides the window
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	os.Exit(0)
	return nil // Never reached
}

func AddPlatformCtlCommands(m libclient.MetaContext, cmd *cobra.Command) {
	var daemonize bool
	looper := &cobra.Command{
		Use:          "looper",
		Short:        "run the agent in a loop, useful on Windows",
		Long:         "run the agent in a loop, useful on Windows; will stop when a stopper file is present",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, arg []string) error {
			return RunCtlLooper(m, cmd, arg, daemonize)
		},
	}
	looper.Flags().BoolVar(&daemonize, "daemonize", false, "daemonize the supervisor (on windows only)")
	cmd.AddCommand(looper)
}
