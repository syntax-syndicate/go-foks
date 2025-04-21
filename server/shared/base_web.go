// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package shared

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func WebServeWithSignals(m MetaContext, ws WebServer, launchCh chan<- error) error {

	mux := chi.NewRouter()
	launchErr := func(e error) error {
		if launchCh != nil {
			launchCh <- e
		}
		return e
	}

	bindAddr, listener, isTLS, err := doListen(m, ws)
	if err != nil {
		return launchErr(err)
	}

	err = ws.InitRouter(m, mux)
	if err != nil {
		return launchErr(err)
	}

	srv := http.Server{
		Addr:    string(bindAddr),
		Handler: mux,
	}

	ws.SetListening(listener, isTLS)
	m.G().TakeServerConfig(ws, listener)

	m.Infow(ws.ServerType().ToString(),
		"stage", "startup",
		"bindAddr", bindAddr,
		"port", port(listener),
		"isTLS", isTLS)

	exitCh := make(chan struct{})

	// Before going to serve in the background, report that
	// we have launched without a problem
	if launchCh != nil {
		launchCh <- nil
	}

	go func() {
		err := srv.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			m.Infow("webServ", "stage", "shutdown-bg", "status", "OK")
		} else if err != nil {
			m.Warnw("webServ", "stage", "shutdown-bg", "err", err)
		}
		exitCh <- struct{}{}
	}()

	// Wait until context canceled
	<-m.Ctx().Done()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(5))
	defer cancel()
	err = srv.Shutdown(ctx)
	if err != nil {
		m.Warnw("webServ", "stage", "shutdown", "err", err)
	}
	<-exitCh

	return nil
}
