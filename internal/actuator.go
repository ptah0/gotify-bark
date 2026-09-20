// SPDX-License-Identifier: GPL-3.0-or-later

package internal

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/hellofresh/health-go/v5"
)

type healthState struct {
	connected      atomic.Bool
	deliveryFailed atomic.Bool
}

func startActuator(addr string, state *healthState) (*http.Server, <-chan error, error) {
	h, err := health.New(health.WithSystemInfo(), health.WithChecks(
		health.Config{Name: "gotify", Check: func(context.Context) error {
			if !state.connected.Load() {
				return errors.New("Gotify is not connected")
			}
			return nil
		}},
		health.Config{Name: "notifications", Check: func(context.Context) error {
			if state.deliveryFailed.Load() {
				return errors.New("last notification failed for one or more destinations")
			}
			return nil
		}},
	))
	if err != nil {
		return nil, nil, err
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, errors.New("failed to start status server")
	}
	mux := http.NewServeMux()
	mux.Handle("/status", h.Handler())
	server := &http.Server{Addr: listener.Addr().String(), Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	return server, done, nil
}
