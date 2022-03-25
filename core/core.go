// SPDX-License-Identifier: GPL-3.0-or-later

package core // Package core import "github.com/ptah0/gotify-bark/core"

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"
)

type Config struct {
	GotifyUrl   string
	GotifyKey   string
	BarkUrl     string
	BarkDevices []string
}

func Run(cfg *Config) {
	log.SetFlags(0)
	// print out values
	log.Println("Starting!")
	log.Printf("Gotify URL: %s, Key: %s\n", cfg.GotifyUrl, cfg.GotifyKey)
	log.Printf("Bark URL: %s, Devices: %s\n", cfg.BarkUrl, cfg.BarkDevices)
	//log.Println("Gotify URL:", gUrl)
	//log.Println("Gotify Key:", gKey)
	//log.Println("Bark URL:", bUrl)
	//log.Println("Bark Devices:", bDevices)

	// handle os interrupt
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// create url
	u, err := url.Parse(cfg.GotifyUrl)
	if err != nil {
		log.Fatal("gUrl:", err)
		return
	}
	u.Path = "/stream"
	q := url.Values{}
	q.Set("token", cfg.GotifyKey)
	u.RawQuery = q.Encode()
	// init websocket
	log.Printf("connecting to %s", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, m, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", m)
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
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
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

func sendPush(msg []byte, u string, d string) {
	// convert msg: Gotify -> Bark
	json := convertMsg(msg, d)
	log.Printf("send: %s", json)

	// create client
	client := &http.Client{}

	// create url
	url, err := url.Parse(u)
	if err != nil {
		log.Fatal("bark Url:", err)
		return
	}
	url.Path = "/push"
	// request
	req, err := http.NewRequest("POST", url.String(), bytes.NewBuffer(json))
	if err != nil {
		log.Println("Failure : ", err)
		return
	}
	// headers
	req.Header.Add("Content-Type", "application/json; charset=utf-8")

	// fetch request
	res, err := client.Do(req)
	if err != nil {
		log.Println("Failure : ", err)
		return
	}
	// read response Body
	body, _ := ioutil.ReadAll(res.Body)

	// handle error
	switch res.StatusCode {
	case 400:
		log.Fatal("Bad Request : ", string(body))
	}

	// display results
	log.Println("response Status : ", res.Status)
	//log.Println("response Headers : ", res.Header)
	log.Println("response Body : ", string(body))
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
	// parse Gotify msg
	var in GotifyMsg
	err := json.Unmarshal(m, &in)
	if err != nil {
		log.Println("Failure: ", err)
	}
	// Bark msg
	out, err := json.Marshal(&BarkMsg{
		DeviceKey: d,
		Category:  "category",
		Title:     in.Title,
		Body:      in.Message,
		Badge:     int(1),
	})
	if err != nil {
		log.Println("Failure : ", err)
	}

	return out
}
