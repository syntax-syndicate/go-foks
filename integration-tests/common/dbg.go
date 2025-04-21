// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package common

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type stackTrace struct {
	frames []string
}

func (s *stackTrace) frame(i int) string {
	return s.frames[i]
}

func getStackTrace() *stackTrace {
	buf := make([]byte, 0x10000)
	n := runtime.Stack(buf, false)
	s := string(buf[0:n])
	lines := strings.Split(s, "\n")
	ret := stackTrace{
		frames: lines,
	}
	return &ret
}

func findLastPathSep(s string) string {
	for {
		idx := strings.IndexByte(s, filepath.Separator)
		if idx < 0 {
			return s
		}
		if idx == len(s)-1 {
			return s
		}
		s = s[(idx + 1):]
	}
}

func formatFrame(s string) string {
	s = findLastPathSep(s)
	parenIdx := strings.IndexByte(s, '(')
	if parenIdx < 0 {
		return s
	}
	return s[0:(parenIdx)] + " " + s[parenIdx:]
}

func DebugEntryAndExit() func() {
	st := getStackTrace()
	nm := formatFrame(st.frame(5))
	fmt.Printf("+TEST %s\n", nm)
	done := false
	go func() {
		time.Sleep(15 * time.Second)
		if done {
			return
		}
		log, err := os.OpenFile("slow-tests.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open slow-tests.log: %v\n", err)
		} else {
			timeS := time.Now().Format(time.RFC3339)
			fmt.Fprintf(log, "%s TEST %s still running after 15 seconds\n", timeS, nm)
			log.Close()
		}
	}()
	return func() {
		done = true
		fmt.Printf("-TEST %s\n", nm)
	}
}
