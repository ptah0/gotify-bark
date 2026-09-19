// SPDX-License-Identifier: GPL-3.0-or-later

package internal // Package internal import "github.com/ptah0/gotify-bark/internal"

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/containrrr/shoutrrr"
	"github.com/containrrr/shoutrrr/pkg/router"
	"github.com/containrrr/shoutrrr/pkg/types"
	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type Config struct {
	GotifyUrl    string
	GotifyKey    string
	ShoutrrrURLs []string
}

func Run(cfg *Config) error {
	if len(cfg.ShoutrrrURLs) == 0 {
		return errors.New("at least one Shoutrrr URL is required")
	}
	for i, rawURL := range cfg.ShoutrrrURLs {
		u, err := url.Parse(rawURL)
		if err != nil || u.Scheme == "" {
			return fmt.Errorf("invalid notification URL %d", i+1)
		}
		if u.Scheme == "bark" {
			key, _ := u.User.Password()
			if u.Hostname() == "" || key == "" {
				return fmt.Errorf("Bark URL %d requires a host and device key", i+1)
			}
		}
	}
	sender, err := shoutrrr.CreateSender(cfg.ShoutrrrURLs...)
	if err != nil {
		// Library errors can include credentials from the URL.
		return errors.New("invalid Shoutrrr configuration; check notification URLs")
	}
	startActuator()
	log.Info().Int("destinations", len(cfg.ShoutrrrURLs)).Msg("Read config")

	// Handle os interrupt
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	defer signal.Stop(interrupt)

	// Create url
	u, err := url.Parse(cfg.GotifyUrl)
	if err != nil {
		return errors.New("invalid Gotify URL")
	}
	u.Path = "/stream"
	q := url.Values{}
	q.Set("token", cfg.GotifyKey)
	u.RawQuery = q.Encode()
	// Init websocket
	log.Debug().Msg("Connecting to Gotify")
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return errors.New("failed to connect to Gotify")
	}
	defer c.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, m, err := c.ReadMessage()
			if err != nil {
				log.Debug().Err(err).Msg("read")
				return
			}
			if err := sendPush(m, sender); err != nil {
				log.Warn().Err(err).Msg("Notification forwarding failed")
			}
		}
	}()

	for {
		select {
		case <-done:
			return nil
		case <-interrupt:
			log.Info().Msg("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Info().Err(err).Msg("write close")
				return nil
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return nil
		}
	}
}

func sendPush(msg []byte, sender *router.ServiceRouter) error {
	var in GotifyMsg
	if err := json.Unmarshal(msg, &in); err != nil {
		return errors.New("invalid Gotify message")
	}
	params := types.Params{"title": in.Title}
	failed := false
	for _, err := range sender.Send(in.Message, &params) {
		if err != nil {
			// Delivery errors may contain credential-bearing URLs or response bodies.
			log.Warn().Msg("Notification delivery failed")
			failed = true
		}
	}
	if failed {
		return errors.New("one or more notification deliveries failed")
	}
	return nil
}

type GotifyMsg struct {
	Title    string    `json:"title"`
	Message  string    `json:"message"`
	Priority int       `json:"priority"`
	Date     time.Time `json:"date"`
}
