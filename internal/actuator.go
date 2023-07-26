// SPDX-License-Identifier: GPL-3.0-or-later

package internal // Package internal import "github.com/ptah0/gotify-bark/internal"

import (
	"net/http"

	"github.com/hellofresh/health-go/v5"
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
