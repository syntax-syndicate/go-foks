// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package cmd

import (
	"os"
	"os/exec"
	"time"

	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/lib/core"
	"github.com/spf13/cobra"
)

func initStopperFile(m libclient.MetaContext) (*libclient.StopperFile, error) {
	stopperFile, err := m.G().Cfg().GetAgentStopperFile()
	if err != nil {
		return nil, err
	}

	stat, err := stopperFile.Stat()
	if err != nil {
		return stopperFile, nil
	}

	if stat.IsDir() {
		m.Errorw("looper",
			"stage", "start",
			"file", stopperFile,
			"msg", "file is a directory")
		return nil, core.ConfigError("stopper file is a directory")
	}

	if !stat.Mode().IsRegular() {
		m.Errorw("looper",
			"stage", "start",
			"file", stopperFile,
			"msg", "file is not a regular file")
		return nil, core.ConfigError("stopper file is not a regular file")
	}

	err = stopperFile.RemoveAll()
	if err != nil {
		m.Errorw("looper",
			"stage", "start",
			"file", stopperFile,
			"msg", "failed to remove file")
		return nil, core.ConfigError("failed to remove stopper file")
	}

	return stopperFile, nil
}

func RunCtlLooper(m libclient.MetaContext, cmd *cobra.Command, arg []string, daemonize bool) error {
	m = m.WithLogTag("ctl-looper")

	if daemonize {
		err := doDaemonize(m)
		if err != nil {
			return err
		}
	}

	err := setupCtlCmd(m)
	if err != nil {
		return err
	}

	stopperFile, err := initStopperFile(m)
	if err != nil {
		return err
	}

	findFile := func() bool {
		_, err := stopperFile.Stat()
		return err == nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	m.Infow("looper", "stage", "start", "file", stopperFile)

	first := true

	for {
		if findFile() {
			m.Infow("looper", "stage", "stop", "file", stopperFile)
			break
		}
		if !first {
			time.Sleep(3 * time.Second)
		}
		cmd := exec.Command(exe,
			"agent", "--agent-check-stopper",
			"--agent-stopper-file", stopperFile.String())
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			m.Errorw("looper", "stage", "launch", "err", err)
			time.Sleep(5 * time.Second)
		}
		m.Infow("looper", "stage", "launch", "pid", cmd.Process.Pid)
		err = cmd.Wait()
		if err != nil {
			m.Errorw("looper", "stage", "proc-exit", "err", err)
		}
		first = false
	}

	return nil
}
