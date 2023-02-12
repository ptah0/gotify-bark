// SPDX-License-Identifier: GPL-3.0-or-later

package core // Package core import "github.com/ptah0/gotify-bark/core"

import (
	"net/http"

	"github.com/hellofresh/health-go/v5"
	// healthHttp "github.com/hellofresh/health-go/v5/checks/http"
	"github.com/rs/zerolog/log"
)

func startActuator() {
	// config
	h, _ := health.New(health.WithSystemInfo())

	// start server
	log.Info().Msg("Startup /status endpoint")
	go func() {
		http.Handle("/status", h.Handler())
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatal().Err(err).Msg("Startup failed")
		}
	}()
}
