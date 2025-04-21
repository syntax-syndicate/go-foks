// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

import (
	"time"

	"github.com/foks-proj/go-foks/lib/core"
)

func MerkleRaceRetry(
	m MetaContext,
	run func() error,
	reset func(),
	which string,
) error {
	params, err := m.G().Cfg().MerkleRaceRetryConfig()
	if err != nil {
		return err
	}

	n := params.NumRetries
	wait := params.Wait

	shouldRetry := func(err error) bool {
		if err == nil {
			return false
		}
		if raceErr, ok := err.(RaceError); ok {
			return raceErr.IsRace()
		}
		if clerr, ok := err.(core.ChainLoaderError); ok && clerr.Race {
			return true
		}
		return false
	}

	for i := 0; true; i++ {

		err = run()
		if err == nil || i >= n || !shouldRetry(err) {
			return err
		}
		m.Warnw("Retry("+which+")",
			"err", err.Error(),
			"wait", wait,
			"iter", i,
		)
		if reset != nil {
			reset()
		}
		time.Sleep(wait)
		wait *= 2
	}
	panic("unreachable")
}
