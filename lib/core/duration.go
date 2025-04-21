// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package core

import (
	"encoding/json"
	"time"

	"github.com/spf13/pflag"
)

type Duration struct {
	time.Duration
}

func (d *Duration) IsZero() bool {
	return d.Duration == 0
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	d.Duration, err = time.ParseDuration(s)
	return err
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

type Durations struct {
	Durations []time.Duration
}

func (d *Durations) UnmarshalJSON(data []byte) error {
	var ss []string
	err := json.Unmarshal(data, &ss)
	if err != nil {
		return err
	}
	var res []time.Duration
	for _, s := range ss {
		tmp, err := time.ParseDuration(s)
		if err != nil {
			return err
		}
		res = append(res, tmp)
	}
	d.Durations = res
	return nil
}

func (d Durations) MarshalJSON() ([]byte, error) {
	var ss []string
	for _, dur := range d.Durations {
		ss = append(ss, dur.String())
	}
	return json.Marshal(ss)
}

func (d *Duration) Set(s string) error {
	tmp, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = tmp
	return nil
}

func (d *Duration) Value() string {
	return d.Duration.String()
}

func (d *Duration) Type() string {
	return "duration"
}

var _ pflag.Value = (*Duration)(nil)
