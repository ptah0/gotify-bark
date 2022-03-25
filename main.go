// SPDX-License-Identifier: GPL-3.0-or-later

package main // import "github.com/ptah0/gotify-bark"

import (
	"log"
	"os"

	"github.com/ptah0/gotify-bark/core"
	"github.com/urfave/cli/v2"
)

// Main

func main() {
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
				Value:   "api.day.app",
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
		},
	}

	// run core
	app.Action = func(c *cli.Context) error {
		core.Run(&core.Config{
			GotifyUrl:   c.String("gotify-url"),
			GotifyKey:   c.String("gotify-key"),
			BarkUrl:     c.String("bark-url"),
			BarkDevices: c.StringSlice("bark-device"),
		})
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}
