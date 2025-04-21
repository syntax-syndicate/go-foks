// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package core

import (
	"context"
	"encoding/json"
	"os"
	"sync"

	"github.com/google/go-jsonnet"
)

func JsonnetCompile(f string) (string, error) {
	vm := jsonnet.MakeVM()
	return vm.EvaluateFile(f)
}

type ConfigJSonnet[T any] struct {
	sync.RWMutex
	Path   Path
	Raw    string
	Loaded bool
	Data   T
	vm     *jsonnet.VM
}

func (j *ConfigJSonnet[T]) getVm() *jsonnet.VM {
	if j.vm != nil {
		return j.vm
	}
	j.vm = jsonnet.MakeVM()
	dirname := j.Path.Dir()
	j.vm.Importer(
		&jsonnet.FileImporter{
			JPaths: []string{dirname.String()},
		},
	)
	return j.vm
}

func (j *ConfigJSonnet[T]) AddExtVar(k, v string) {
	j.getVm().ExtVar(k, v)
}

func (j *ConfigJSonnet[T]) Create(ctx context.Context) (err error) {
	j.Lock()
	defer j.Unlock()
	err = j.Path.MakeParentDirs()
	if err != nil {
		return err
	}
	var fh *os.File
	fh, err = j.Path.OpenFile(os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer func() {
		err = fh.Close()
	}()
	_, err = fh.WriteString("{}\n")
	if err != nil {
		return err
	}
	return nil
}

func (j *ConfigJSonnet[T]) Load(ctx context.Context) error {
	j.Lock()
	defer j.Unlock()
	var err error

	dat, err := j.Path.ReadFile()
	if err != nil {
		return err
	}
	j.Raw, err = j.getVm().EvaluateAnonymousSnippet(j.Path.String(), string(dat))
	if err != nil {
		return err
	}
	err = j.readIntoTemplate()
	if err != nil {
		return err
	}
	j.Loaded = true
	return nil
}

func (j *ConfigJSonnet[T]) readIntoTemplate() error {
	err := json.Unmarshal([]byte(j.Raw), &j.Data)
	if err != nil {
		return err
	}
	return nil
}

func (e *ConfigJSonnet[T]) IsLoaded() bool {
	e.RLock()
	defer e.RUnlock()
	return e.Loaded
}
