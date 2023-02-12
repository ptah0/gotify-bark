// SPDX-License-Identifier: GPL-3.0-or-later

package core // Package core import "github.com/ptah0/gotify-bark/core"

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type Config struct {
	GotifyUrl   string
	GotifyKey   string
	BarkUrl     string
	BarkDevices []string
}

func Run(cfg *Config) {
	// Actuator
	startActuator()

	// Print out values
	log.Info().
		Str("GotifyUrl", cfg.GotifyUrl).
		Str("GotifyKey", cfg.GotifyKey).
		Str("BarkUrl", cfg.BarkUrl).
		Strs("BarkDevices", cfg.BarkDevices).
		Msg("Read Config")

	// Handle os interrupt
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Create url
	u, err := url.Parse(cfg.GotifyUrl)
	if err != nil {
		log.Error().Err(err).Msg("Invalid Gotify Url")
		return
	}
	u.Path = "/stream"
	q := url.Values{}
	q.Set("token", cfg.GotifyKey)
	u.RawQuery = q.Encode()
	// Init websocket
	log.Debug().Msgf("connecting to %s", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal().Err(err).Msg("dial")
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
			log.Debug().Bytes("Msg", m).Msg("recv")
			// forward request to Bark
			for _, d := range cfg.BarkDevices {
				sendPush(m, cfg.BarkUrl, d)
			}
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Info().Msg("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Info().Err(err).Msg("write close")
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}

func sendPush(msg []byte, barkUrl string, dev string) {
	// Convert msg: Gotify -> Bark
	out := convertMsg(msg, dev)
	log.Debug().Bytes("json", out).Msg("send")

	// Create client
	client := &http.Client{}

	// Create url
	u, err := url.Parse(barkUrl)
	if err != nil {
		log.Fatal().Err(err).Msg("Bark Url Invalid")
		return
	}
	u.Path = "/push"
	// Request
	req, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(out))
	if err != nil {
		log.Error().Err(err).Msg("Failure to POST")
		return
	}
	// Headers
	req.Header.Add("Content-Type", "application/json; charset=utf-8")

	// Fetch request
	resp, err := client.Do(req)
	if err != nil {
		log.Warn().Err(err).Msg("Failure to fetch request")
		return
	}
	// Read response Body
	body, _ := ioutil.ReadAll(resp.Body)

	// handle error
	switch resp.StatusCode {
	case 400:
		log.Error().Str("body", string(body)).Msg("Bad Request")
	}

	// display results
	log.Info().Msgf("response Status : ", resp.Status)
	log.Debug().Msgf("response Headers : ", resp.Header)
	log.Debug().Msgf("response Body : ", string(body))
}

type GotifyMsg struct {
	Title    string    `json:"title"`
	Message  string    `json:"message"`
	Priority int       `json:"priority"`
	Date     time.Time `json:"date"`
}

type BarkMsg struct {
	DeviceKey string `json:"device_key"`
	Category  string `json:"category"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Badge     int    `json:"badge"`
}

func convertMsg(m []byte, d string) []byte {
	// Parse Gotify msg
	var in GotifyMsg
	err := json.Unmarshal(m, &in)
	if err != nil {
		log.Warn().Err(err).Msg("Parse Gotify message")
	}
	// Gen Bark msg
	out, err := json.Marshal(&BarkMsg{
		DeviceKey: d,
		Category:  "category",
		Title:     in.Title,
		Body:      in.Message,
		Badge:     int(1),
	})
	if err != nil {
		log.Warn().Err(err).Msg("Generate Bark message")
	}

	return out
}
