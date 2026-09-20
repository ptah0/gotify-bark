// SPDX-License-Identifier: GPL-3.0-or-later

package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/containrrr/shoutrrr/pkg/router"
	"github.com/containrrr/shoutrrr/pkg/types"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type Config struct {
	GotifyURL    string
	GotifyKey    string
	ShoutrrrURLs []string
}

var errDeliveryTimeout = errors.New("notification delivery timed out")
var errInvalidMessage = errors.New("invalid Gotify message")

func Run(ctx context.Context, cfg Config) error {
	return run(ctx, cfg, ":8080")
}

func run(ctx context.Context, cfg Config, statusAddr string) error {
	u, err := streamURL(cfg.GotifyURL, cfg.GotifyKey)
	if err != nil {
		return err
	}
	senders, err := newSenders(cfg.ShoutrrrURLs)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if ctx.Err() != nil {
		return nil
	}
	state := &healthState{}
	status, statusDone, err := startActuator(statusAddr, state)
	if err != nil {
		return err
	}
	defer status.Close()
	log.Info().Int("destinations", len(senders)).Msg("Read config")

	done := make(chan struct{})
	go func() {
		defer close(done)
		reconnect(ctx, u, senders, state)
	}()
	defer func() { cancel(); <-done }()
	select {
	case <-statusDone:
		return errors.New("status server stopped unexpectedly")
	case <-ctx.Done():
		return nil
	}
}

func reconnect(ctx context.Context, u string, senders []types.Sender, state *healthState) {
	// Gorilla v1.5.3 does not interrupt a stalled HTTP handshake on cancellation.
	dialer := *websocket.DefaultDialer
	stopHandshakeCancel := func() bool { return false }
	dialer.NetDialContext = func(dialCtx context.Context, network, addr string) (net.Conn, error) {
		conn, err := (&net.Dialer{}).DialContext(dialCtx, network, addr)
		if err == nil {
			stopHandshakeCancel = context.AfterFunc(ctx, func() { conn.Close() })
		}
		return conn, err
	}
	for ctx.Err() == nil {
		c, _, err := dialer.DialContext(ctx, u, nil)
		stopHandshakeCancel()
		if err != nil {
			err = errors.New("failed to connect to Gotify")
		} else {
			state.connected.Store(true)
			stop := context.AfterFunc(ctx, func() {
				_ = c.WriteControl(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
				c.Close()
			})
			err = forward(ctx, c, senders, state)
			stop()
			c.Close()
		}
		if ctx.Err() != nil {
			return
		}
		log.Error().Err(err).Msg("Gotify connection lost; retrying in 1 second")
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

func streamURL(rawURL, key string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "ws" && u.Scheme != "wss") || u.Hostname() == "" || u.Fragment != "" {
		return "", errors.New("invalid Gotify URL; expected ws:// or wss:// with a host")
	}
	if strings.TrimSpace(key) == "" {
		return "", errors.New("Gotify client token is required")
	}
	u = u.JoinPath("stream")
	q := u.Query()
	q.Set("token", key)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func newSenders(urls []string) ([]types.Sender, error) {
	if len(urls) == 0 {
		return nil, errors.New("at least one Shoutrrr URL is required")
	}
	r := &router.ServiceRouter{}
	senders := make([]types.Sender, 0, len(urls))
	for i, rawURL := range urls {
		u, err := url.Parse(rawURL)
		if err != nil || u.Scheme == "" {
			return nil, fmt.Errorf("invalid notification URL %d", i+1)
		}
		if strings.EqualFold(u.Scheme, "bark") {
			key, _ := u.User.Password()
			if u.Hostname() == "" || key == "" {
				return nil, fmt.Errorf("Bark URL %d requires a host and device key", i+1)
			}
		}
		sender, err := r.Locate(rawURL)
		if err != nil {
			// Library errors can include credentials from the URL.
			return nil, fmt.Errorf("invalid Shoutrrr configuration for notification URL %d", i+1)
		}
		senders = append(senders, &guardedSender{Sender: sender})
	}
	return senders, nil
}

// Providers cannot cancel sends; skip a busy destination until its previous send returns.
type guardedSender struct {
	types.Sender
	busy atomic.Bool
}

func (s *guardedSender) Send(message string, params *types.Params) error {
	if !s.busy.CompareAndSwap(false, true) {
		return errors.New("notification provider is still busy")
	}
	defer s.busy.Store(false)
	return s.Sender.Send(message, params)
}

func forward(ctx context.Context, c *websocket.Conn, senders []types.Sender, state *healthState) error {
	defer state.connected.Store(false)
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			// Close errors may contain server-supplied secrets.
			return errors.New("Gotify connection closed unexpectedly")
		}
		err = sendPush(ctx, msg, senders)
		if !errors.Is(err, errInvalidMessage) && ctx.Err() == nil {
			state.deliveryFailed.Store(err != nil)
		}
		if err != nil {
			if ctx.Err() != nil {
				return err
			}
			log.Error().Err(err).Msg("Notification forwarding failed")
		}
	}
}

func sendPush(ctx context.Context, msg []byte, senders []types.Sender) error {
	var in GotifyMsg
	if err := json.Unmarshal(msg, &in); err != nil {
		return errInvalidMessage
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	// Bypass Shoutrrr v0.8.0's unbuffered timeout channel. Late results must not block.
	results := make(chan error, len(senders))
	for _, sender := range senders {
		go func() {
			params := types.Params{"title": in.Title}
			results <- sender.Send(in.Message, &params)
		}()
	}
	failed := 0
	for range senders {
		select {
		case err := <-results:
			if err != nil {
				failed++
			}
		case <-ctx.Done():
			// Buffered results let uncancellable provider calls finish after the deadline.
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return errDeliveryTimeout
			}
			return ctx.Err()
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d notification deliveries failed", failed)
	}
	return nil
}

type GotifyMsg struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}
