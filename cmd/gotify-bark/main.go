// SPDX-License-Identifier: GPL-3.0-or-later

package main // import "github.com/ptah0/gotify-bark"

import (
	"context"
	"os"

	"github.com/ptah0/gotify-bark/internal"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

// Main
func main() {
	// Setup log
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Cli
	cmd := &cli.Command{
		Name:  "main",
		Usage: "Gotify Bark Forwarder",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "gotify-url",
				Aliases:  []string{"g"},
				Usage:    "Gotify server URL",
				Sources:  cli.EnvVars("APP_GOTIFY_URL"),
				Required: true,
			},
			&cli.StringFlag{
				Name:     "gotify-key",
				Aliases:  []string{"k"},
				Usage:    "Gotify server auth key",
				Sources:  cli.EnvVars("APP_GOTIFY_KEY"),
				Required: true,
			},
			&cli.StringFlag{
				Name:    "bark-url",
				Aliases: []string{"b"},
				Value:   "https://api.day.app",
				Usage:   "Gotify server URL",
				Sources: cli.EnvVars("APP_BARK_URL"),
			},
			&cli.StringSliceFlag{
				Name:     "bark-device",
				Aliases:  []string{"d"},
				Usage:    "Bark notification device(s)",
				Sources:  cli.EnvVars("APP_BARK_DEVICE"),
				Required: true,
			},
			&cli.BoolFlag{
				Name:    "debug",
				Usage:   "Enable debug output",
				Sources: cli.EnvVars("APP_DEBUG"),
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			// Debug
			if c.Bool("debug") {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
			}
			// Run Core App
			internal.Run(&internal.Config{
				GotifyUrl:   c.String("gotify-url"),
				GotifyKey:   c.String("gotify-key"),
				BarkUrl:     c.String("bark-url"),
				BarkDevices: c.StringSlice("bark-device"),
			})
			return nil
		},
	}

	err := cmd.Run(context.Background(), os.Args)
	if err != nil {
		log.Fatal().Err(err).Msg("Failure to run cmd")
	}

}
