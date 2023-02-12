// SPDX-License-Identifier: GPL-3.0-or-later

package main // import "github.com/ptah0/gotify-bark"

import (
    "os"

	"github.com/ptah0/gotify-bark/core"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

// Main

func main() {
	// Setup log
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Cli
	app := &cli.App{
		Name:  "main",
		Usage: "Gotify Bark Forwarder",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "gotify-url",
				Aliases:  []string{"g"},
				Usage:    "Gotify server URL",
				EnvVars:  []string{"APP_GOTIFY_URL"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "gotify-key",
				Aliases:  []string{"k"},
				Usage:    "Gotify server auth key",
				EnvVars:  []string{"APP_GOTIFY_KEY"},
				Required: true,
			},
			&cli.StringFlag{
				Name:    "bark-url",
				Aliases: []string{"b"},
				Value:   "https://api.day.app",
				Usage:   "Gotify server URL",
				EnvVars: []string{"APP_BARK_URL"},
			},
			&cli.StringSliceFlag{
				Name:     "bark-device",
				Aliases:  []string{"d"},
				Usage:    "Bark notification device(s)",
				EnvVars:  []string{"APP_BARK_DEVICE"},
				Required: true,
			},
			&cli.BoolFlag{
				Name:    "debug",
				Usage:   "Enable debug output",
				EnvVars: []string{"APP_DEBUG"},
			},
		},
		Action: func(c *cli.Context) error {
			// Debug
			if c.Bool("debug") {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
			}
			// Run Core App
			core.Run(&core.Config{
				GotifyUrl:   c.String("gotify-url"),
				GotifyKey:   c.String("gotify-key"),
				BarkUrl:     c.String("bark-url"),
				BarkDevices: c.StringSlice("bark-device"),
			})
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal().Err(err).Msg("Failure to run app")
	}

}
