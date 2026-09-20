package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nicholas-fedor/shoutrrr/pkg/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestStreamURL(t *testing.T) {
	for _, base := range []string{"wss://example.com", "wss://example.com/gotify", "wss://example.com/gotify/", "wss://example.com/a%2Fb?other=value&token=old"} {
		got, err := streamURL(base, "placeholder&token")
		if err != nil {
			t.Fatal(err)
		}
		u, _ := url.Parse(got)
		original, _ := url.Parse(base)
		wantPath := strings.TrimRight(original.EscapedPath(), "/") + "/stream"
		if u.EscapedPath() != wantPath || u.Query().Get("token") != "placeholder&token" || u.Query().Get("other") != original.Query().Get("other") {
			t.Fatalf("unexpected stream URL: %s", got)
		}
	}
	for _, input := range [][2]string{{"", "secret"}, {"https://example.com", "secret"}, {"ws:///missing", "secret"}, {"wss://%secret", "secret"}, {"wss://example.com/#secret", "secret"}, {"wss://example.com", " "}} {
		_, err := streamURL(input[0], input[1])
		if err == nil || strings.Contains(err.Error(), "secret") {
			t.Fatalf("expected safe validation error, got %v", err)
		}
	}
}

// Shoutrrr already defines the sender interface; this adapter only supplies test behavior.
type sendFunc func(string, *types.Params) error

func (f sendFunc) Send(message string, params *types.Params) error { return f(message, params) }

func TestSendPushTimeoutAndCancellation(t *testing.T) {
	for _, cancelEarly := range []bool{false, true} {
		name := "timeout"
		if cancelEarly {
			name = "cancellation"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if cancelEarly {
					go func() { time.Sleep(time.Second); cancel() }()
				}
				completed := false
				sender := sendFunc(func(string, *types.Params) error {
					time.Sleep(11 * time.Second)
					completed = true
					return errors.New("provider-secret")
				})
				start := time.Now()
				err := sendPush(ctx, []byte(`{"message":"body"}`), []types.Sender{sender})
				wantErr, wantDuration := errDeliveryTimeout, 10*time.Second
				if cancelEarly {
					wantErr, wantDuration = context.Canceled, time.Second
				}
				if !errors.Is(err, wantErr) || time.Since(start) != wantDuration {
					t.Fatalf("got %v after %v", err, time.Since(start))
				}
				time.Sleep(12 * time.Second)
				synctest.Wait()
				if !completed {
					t.Fatal("provider did not finish")
				}
				// synctest fails if a late result leaves any goroutine blocked.
			})
		})
	}
}

func TestRunForwardsAndCancels(t *testing.T) {
	received := make(chan GotifyMsg, 1)
	bark := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct{ Title, Body string }
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		received <- GotifyMsg{Title: payload.Title, Message: payload.Body}
		_, _ = io.WriteString(w, `{"code":200,"message":"success"}`)
	}))
	defer bark.Close()
	closed := make(chan struct{})
	gotify := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(closed)
		if r.URL.Path != "/gotify/stream" || r.URL.Query().Get("token") != "placeholder" {
			t.Errorf("unexpected subscription: %s", r.URL)
		}
		c, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer c.Close()
		_ = c.WriteMessage(websocket.TextMessage, []byte(`{"title":`))
		_ = c.WriteMessage(websocket.TextMessage, []byte(`{"title":"Title","message":"Body","date":"unused","priority":"unused"}`))
		_, _, _ = c.ReadMessage()
	}))
	defer gotify.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	addr := freeAddress(t)
	done := make(chan error, 1)
	go func() {
		done <- run(ctx, Config{
			GotifyURL:    "ws" + strings.TrimPrefix(gotify.URL, "http") + "/gotify/",
			GotifyKey:    "placeholder",
			ShoutrrrURLs: []string{"bark://:placeholder@" + strings.TrimPrefix(bark.URL, "http://") + "/?scheme=http"},
		}, addr)
	}()
	select {
	case msg := <-received:
		if msg.Title != "Title" || msg.Message != "Body" {
			t.Fatalf("unexpected notification: %+v", msg)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("notification was not delivered")
	}
	client := &http.Client{Timeout: time.Second}
	response, err := client.Get("http://" + addr + "/status")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	var statusBody struct{ Status string }
	decodeErr := json.Unmarshal(body, &statusBody)
	if err != nil || decodeErr != nil || response.StatusCode != http.StatusOK || statusBody.Status != "OK" {
		t.Fatalf("unexpected status response: %s, %v", body, err)
	}
	cancel()
	if err := waitRun(t, done); err != nil {
		t.Fatal(err)
	}
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("WebSocket was not closed")
	}
	// Restarting the status server must neither retain the port nor register globally.
	status, statusDone, err := startActuator(addr, &healthState{})
	if err != nil {
		t.Fatal(err)
	}
	status.Close()
	if err := waitRun(t, statusDone); !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("unexpected server shutdown: %v", err)
	}
}

func TestRunDisconnectRetriesSafely(t *testing.T) {
	var attempts atomic.Int32
	gotify := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		if attempts.Load() == 2 {
			http.Error(w, "server-secret", http.StatusServiceUnavailable)
			return
		}
		c, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer c.Close()
		_ = c.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "server-secret"), time.Now().Add(time.Second))
	}))
	defer gotify.Close()
	var logs bytes.Buffer
	oldLogger := log.Logger
	log.Logger = zerolog.New(&logs)
	defer func() { log.Logger = oldLogger }()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := run(ctx, Config{
		GotifyURL: "ws" + strings.TrimPrefix(gotify.URL, "http") + "/path-secret?extra=query-secret", GotifyKey: "token-secret",
		ShoutrrrURLs: []string{
			"bark://:device-secret@localhost:8443/path%2Dsecret?url=https%3A%2F%2Fquery-secret#fragment-secret",
			"bark://:device-secret@[::1]:9443/path-secret",
			"generic://user-secret:password-secret@localhost/path-secret?disabletls=yes",
			"pushbullet://" + strings.Repeat("x", 28) + "secret",
		},
	}, "127.0.0.1:0")
	if err != nil || attempts.Load() < 3 || !strings.Contains(logs.String(), "retrying") || strings.Contains(logs.String(), "secret") {
		t.Fatalf("expected safe retries, got %v, attempts %d, logs %s", err, attempts.Load(), logs.String())
	}
	var startup struct {
		Host       string   `json:"gotify_host"`
		Transport  string   `json:"gotify_transport"`
		Count      int      `json:"destinations"`
		Services   []string `json:"services"`
		StatusAddr string   `json:"status_addr"`
		StatusPath string   `json:"status_path"`
	}
	if err := json.NewDecoder(strings.NewReader(logs.String())).Decode(&startup); err != nil {
		t.Fatal(err)
	}
	wantServices := "bark://:REDACTED@localhost:8443/REDACTED?REDACTED#REDACTED,bark://:REDACTED@[::1]:9443/REDACTED,generic://REDACTED:REDACTED@localhost/REDACTED?REDACTED,pushbullet://REDACTED"
	if startup.Host != strings.TrimPrefix(gotify.URL, "http://") || startup.Transport != "ws" || startup.Count != 4 || strings.Join(startup.Services, ",") != wantServices || startup.StatusPath != "/status" {
		t.Fatalf("unexpected startup details: %+v", startup)
	}
	if _, port, err := net.SplitHostPort(startup.StatusAddr); err != nil || port == "0" {
		t.Fatalf("expected bound status address, got %q", startup.StatusAddr)
	}
	if !strings.Contains(logs.String(), "Connected to Gotify") {
		t.Fatal("missing successful connection log")
	}
}

func TestRunCancelsHandshakeAndReleasesStatus(t *testing.T) {
	connected, release := make(chan struct{}), make(chan struct{})
	gotify := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(connected)
		<-release
	}))
	defer gotify.Close()
	defer close(release)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	addr := freeAddress(t)
	done := make(chan error, 1)
	go func() {
		done <- run(ctx, Config{
			GotifyURL: "ws" + strings.TrimPrefix(gotify.URL, "http"), GotifyKey: "placeholder",
			ShoutrrrURLs: []string{"bark://:placeholder@localhost"},
		}, addr)
	}()
	select {
	case <-connected:
	case <-time.After(3 * time.Second):
		t.Fatal("handshake did not start")
	}
	cancel()
	if err := waitRun(t, done); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal("status listener was not released:", err)
	}
	listener.Close()
}

func TestRunValidatesBeforeBindingAndReportsBindFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	cfg := Config{GotifyURL: "invalid-secret", GotifyKey: "placeholder", ShoutrrrURLs: []string{"bark://:placeholder@localhost"}}
	err = run(context.Background(), cfg, listener.Addr().String())
	if err == nil || !strings.Contains(err.Error(), "Gotify URL") || strings.Contains(err.Error(), "secret") {
		t.Fatalf("expected safe configuration error before bind, got %v", err)
	}
	cfg.GotifyURL = "ws://localhost"
	err = run(context.Background(), cfg, listener.Addr().String())
	if err == nil || err.Error() != "failed to start status server" {
		t.Fatalf("expected bind failure, got %v", err)
	}
}

func freeAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	listener.Close()
	return addr
}

func waitRun(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(3 * time.Second):
		t.Fatal("run did not stop")
		return nil
	}
}

func TestHealthChecksTrackForwarding(t *testing.T) {
	state := &healthState{}
	status, _, err := startActuator("127.0.0.1:0", state)
	if err != nil {
		t.Fatal(err)
	}
	defer status.Close()
	check := func(code int, failures int, name string) {
		t.Helper()
		deadline := time.Now().Add(time.Second)
		for {
			response := httptest.NewRecorder()
			status.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/status", nil))
			var body struct {
				Failures map[string]string
				System   json.RawMessage
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(response.Body.String(), "provider-secret") {
				t.Fatal("health response leaked credentials")
			}
			if response.Code == code && len(body.Failures) == failures && (name == "" || body.Failures[name] != "") {
				if len(body.System) == 0 {
					t.Fatal("system information missing")
				}
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("unexpected health response: %d %s", response.Code, response.Body.String())
			}
			time.Sleep(time.Millisecond)
		}
	}
	check(http.StatusServiceUnavailable, 1, "gotify")
	messages := make(chan string, 8)
	defer close(messages)
	gotify := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer c.Close()
		for {
			select {
			case msg := <-messages:
				if msg == "disconnect" {
					return
				}
				if err := c.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
					return
				}
			case <-t.Context().Done():
				return
			}
		}
	}))
	defer gotify.Close()
	c, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(gotify.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	state.connected.Store(true)
	done := make(chan error, 1)
	go func() {
		done <- forward(t.Context(), c, []types.Sender{sendFunc(func(message string, _ *types.Params) error {
			if message == "fail" {
				return errors.New("provider-secret")
			}
			return nil
		})}, state)
	}()
	check(http.StatusOK, 0, "") // An idle connection is healthy before its first notification.
	messages <- `{"message":"fail"}`
	check(http.StatusServiceUnavailable, 1, "notifications")
	messages <- `{"message":"success"}`
	check(http.StatusOK, 0, "")
	messages <- `{"message":"fail"}`
	check(http.StatusServiceUnavailable, 1, "notifications")
	messages <- `{"title":`
	messages <- "disconnect"
	if err := waitRun(t, done); err == nil {
		t.Fatal("disconnect was ignored")
	}
	check(http.StatusServiceUnavailable, 2, "gotify")
}

func TestBusySenderRecoversAfterTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		calls := 0
		sender := &guardedSender{Sender: sendFunc(func(string, *types.Params) error {
			calls++
			if calls == 1 {
				time.Sleep(11 * time.Second)
			}
			return nil
		})}
		send := func() error { return sendPush(t.Context(), []byte(`{"message":"body"}`), []types.Sender{sender}) }
		if err := send(); !errors.Is(err, errDeliveryTimeout) {
			t.Fatalf("expected timeout, got %v", err)
		}
		if err := send(); err == nil {
			t.Fatal("busy sender accepted overlapping delivery")
		}
		time.Sleep(2 * time.Second)
		synctest.Wait()
		if err := send(); err != nil {
			t.Fatal(err)
		}
		if calls != 2 {
			t.Fatalf("got %d provider calls", calls)
		}
	})
}

func TestForwardContinuesAfterTimeout(t *testing.T) {
	gotify := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer c.Close()
		for _, message := range []string{"slow", "next"} {
			_ = c.WriteMessage(websocket.TextMessage, []byte(`{"message":"`+message+`"}`))
		}
		_, _, _ = c.ReadMessage()
	}))
	defer gotify.Close()
	c, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(gotify.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	release := make(chan struct{})
	defer close(release)
	received := make(chan string, 2)
	senders := []types.Sender{
		&guardedSender{Sender: sendFunc(func(string, *types.Params) error { <-release; return nil })},
		sendFunc(func(message string, _ *types.Params) error { received <- message; return nil }),
	}
	done := make(chan error, 1)
	go func() { done <- forward(t.Context(), c, senders, &healthState{}) }()
	for _, want := range []string{"slow", "next"} {
		select {
		case got := <-received:
			if got != want {
				t.Fatalf("got %s, want %s", got, want)
			}
		case err := <-done:
			t.Fatalf("forward exited: %v", err)
		case <-time.After(12 * time.Second):
			t.Fatal("forwarding did not continue")
		}
	}
	c.Close()
	_ = waitRun(t, done)
}
