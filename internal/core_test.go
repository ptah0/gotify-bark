package internal

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/containrrr/shoutrrr"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestForwardNotifications(t *testing.T) {
	type payload struct {
		DeviceKey string `json:"device_key"`
		Title     string `json:"title"`
		Body      string `json:"body"`
		Badge     int    `json:"badge"`
		Category  string `json:"category"`
	}
	received := make(chan payload, 8)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/prefix/push" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var p payload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			t.Error(err)
		}
		received <- p
		w.Header().Set("Content-Type", "application/json")
		if p.DeviceKey == "failure-secret" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"code":400,"message":"failure-secret"}`))
			return
		}
		_, _ = w.Write([]byte(`{"code":200,"message":"success"}`))
	}))
	defer server.Close()
	destination := func(key string) string {
		return "bark://:" + key + "@" + strings.TrimPrefix(server.URL, "http://") + "/prefix/?scheme=http&badge=1&category=category"
	}
	sender, err := shoutrrr.CreateSender(destination("first"), destination("second"))
	if err != nil {
		t.Fatal(err)
	}
	// Reuse the sender to ensure an empty title clears the previous title.
	for _, title := range []string{"Hello 世界", ""} {
		message, _ := json.Marshal(GotifyMsg{Title: title, Message: "Body & details"})
		if err := sendPush(message, sender); err != nil {
			t.Fatal(err)
		}
		seen := map[string]bool{}
		for range 2 {
			p := <-received
			if p.Title != title || p.Body != "Body & details" || p.Badge != 1 || p.Category != "category" {
				t.Fatalf("unexpected payload: %+v", p)
			}
			seen[p.DeviceKey] = true
		}
		if !seen["first"] || !seen["second"] {
			t.Fatalf("missing destination: %v", seen)
		}
	}
	if err := sendPush([]byte(`{"title":`), sender); err == nil {
		t.Fatal("malformed input accepted")
	}
	if len(received) != 0 {
		t.Fatal("malformed message was delivered")
	}

	sender, err = shoutrrr.CreateSender(destination("failure-secret"), destination("second"))
	if err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	oldLogger := log.Logger
	log.Logger = zerolog.New(&logs)
	defer func() { log.Logger = oldLogger }()
	err = sendPush([]byte(`{"title":"Test","message":"Body"}`), sender)
	if err == nil {
		t.Fatal("delivery failure was ignored")
	}
	if strings.Contains(logs.String()+err.Error(), "failure-secret") {
		t.Fatal("credential leaked")
	}
	if len(received) != 2 {
		t.Fatal("failure prevented delivery to the other destination")
	}
}

func TestInvalidNotificationConfig(t *testing.T) {
	for _, urls := range [][]string{nil, {""}, {"://secret"}, {"unknown://secret@host"}, {"bark://host"}, {"bark://:secret@/"}, {"bark://:secret@host/?badge=invalid"}} {
		err := Run(&Config{ShoutrrrURLs: urls})
		if err == nil {
			t.Fatalf("configuration accepted: %v", urls)
		}
		if strings.Contains(err.Error(), "secret") {
			t.Fatalf("credential leaked: %v", err)
		}
	}
}
