package simple_ui

import (
	"fmt"
	"time"
)

type Spinner struct {
	Msg string
	Ch  chan struct{}
}

func NewSpinner(msg string) *Spinner {
	return &Spinner{
		Msg: msg,
		Ch:  make(chan struct{}),
	}
}

func (s *Spinner) spinner() {
	spinners := []string{
		"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
	}
	spinIdx := 0

	print := func() {
		if s.Msg != "" {
			fmt.Printf("\r%s %s", s.Msg, spinners[spinIdx])
		} else {
			fmt.Printf("\r%s", spinners[spinIdx])
		}
		spinIdx = (spinIdx + 1) % len(spinners)
	}

	print()
	for {
		select {
		case <-s.Ch:
			return
		default:
			print()
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (s *Spinner) Start() {
	go s.spinner()
}

func (s *Spinner) Stop(msg string) {
	if s.Ch != nil {
		close(s.Ch)
		s.Ch = nil
	}
	fmt.Printf("\r%s %s\n", s.Msg, msg)
}
