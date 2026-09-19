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
		Usage: "Gotify notification forwarder",
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
			&cli.StringSliceFlag{
				Name:     "shoutrrr-url",
				Usage:    "Notification URL (repeat flag or use comma-separated URLs)",
				Sources:  cli.EnvVars("APP_SHOUTRRR_URLS"),
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
			return internal.Run(&internal.Config{
				GotifyUrl:    c.String("gotify-url"),
				GotifyKey:    c.String("gotify-key"),
				ShoutrrrURLs: c.StringSlice("shoutrrr-url"),
			})
		},
	}

	err := cmd.Run(context.Background(), os.Args)
	if err != nil {
		log.Fatal().Err(err).Msg("Failure to run cmd")
	}

}
