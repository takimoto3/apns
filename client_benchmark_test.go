package apns

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/takimoto3/apns/notification"
	"github.com/takimoto3/apns/notification/priority"
	"github.com/takimoto3/apns/payload"
	"github.com/takimoto3/appleapi-core"
	"github.com/takimoto3/appleapi-core/token"
)

var benchmarkPayloads = map[string]*Payload{
	"Minimal": {
		APS: payload.APS{Alert: "Hi"},
	},
	"FullAlert": {
		APS: payload.APS{
			Alert: payload.Alert{
				Title:    "Game Request",
				Subtitle: "Five Card Draw",
				Body:     "Bob wants to play poker",
				LocKey:   "GAME_PLAY_REQUEST_FORMAT",
				LocArgs:  []string{"Bob"},
			},
			Badge: 1,
			Sound: "default",
		},
		CustomData: map[string]any{"game_id": "abc123", "level": 5},
	},
	"Background": {
		APS: payload.APS{ContentAvailable: 1},
		CustomData: map[string]any{
			"update_type": "location",
			"lat":         35.6895,
			"lng":         139.6917,
		},
	},
	"VoIP": {
		APS: payload.APS{Alert: "Incoming call", Sound: "ringtone.caf"},
		CustomData: map[string]any{
			"call_id":    "call-xyz",
			"caller":     "Alice",
			"video_call": true,
		},
	},
	"LiveActivity": {
		APS: payload.APS{Alert: "Workout started", Badge: 3},
		CustomData: map[string]any{
			"activity_type": "running",
			"start_time":    time.Now().Unix(),
			"goal":          5000,
		},
	},
}

func benchmarkClientPush(b *testing.B, payload *Payload, useFast bool) {
	// Dummy usage to avoid "imported and not used" error for token package
	var _ token.Provider = &MockTokenProvider{}

	expectedToken := "Bearer benchmark-token"
	deviceToken := "benchmark-device-token"
	apnsID := "123e4567-e89b-12d3-a456-4266554400a0" // Valid UUID
	bundleID := "com.example.benchmark"

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("apns-id", apnsID)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"aps":{"alert":"benchmark"}}`))
	}))
	defer server.Close()
	server.EnableHTTP2 = true
	server.StartTLS()

	tp := &MockTokenProvider{Token: strings.TrimPrefix(expectedToken, "Bearer ")}

	conf := appleapi.DefaultConfig()
	conf.TLSConfig.InsecureSkipVerify = true
	init := appleapi.ConfigureHTTPClientInitializer(&conf)
	client, err := NewClient(init, tp)
	if err != nil {
		b.Fatalf("NewClient failed: %v", err)
	}
	client.inner.Host = server.URL
	client.FastJson = useFast

	expiration := notification.NewEpochTime(time.Now().Add(time.Hour))
	n := &Notification{
		BundleID:    bundleID,
		DeviceToken: deviceToken,
		Type:        notification.Alert,
		APNsID:      apnsID,
		Expiration:  expiration,
		Priority:    priority.Immediate,
		Payload:     payload,
	}

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.Push(ctx, n)
			if err != nil {
				b.Fatalf("Client.Push failed: %v", err)
			}
		}
	})
}

func BenchmarkClient_Push(b *testing.B) {
	for name, payload := range benchmarkPayloads {
		for _, useFast := range []bool{false, true} {
			mode := "Standard"
			if useFast {
				mode = "Fast"
			}
			b.Run(fmt.Sprintf("%s_%s", name, mode), func(b *testing.B) {
				benchmarkClientPush(b, payload, useFast)
			})
		}
	}
}

func benchmarkClientPushInternal(b *testing.B, payload *Payload, useFast bool) {
	mockResp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"aps":{"alert":"test"}}`)),
		Header:     http.Header{"apns-id": []string{"dummy-id"}},
	}

	tp := &MockTokenProvider{Token: "dummy"}

	conf := appleapi.DefaultConfig()
	init := appleapi.ConfigureHTTPClientInitializer(&conf)

	client, _ := NewClient(init, tp)
	client.FastJson = useFast

	client.inner.HTTPClient.Transport = &mockRoundTripper{resp: mockResp}

	n := &Notification{
		BundleID:    "com.example",
		DeviceToken: "aaa",
		Type:        notification.Alert,
		Payload:     payload,
	}

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.Push(ctx, n)
			if err != nil {
				b.Fatalf("Client.Push failed: %v", err)
			}
		}
	})

}

func BenchmarkClientPushInternal(b *testing.B) {
	for name, payload := range benchmarkPayloads {
		for _, useFast := range []bool{false, true} {
			mode := "Standard"
			if useFast {
				mode = "Fast"
			}
			b.Run(fmt.Sprintf("%s_%s", name, mode), func(b *testing.B) {
				benchmarkClientPushInternal(b, payload, useFast)
			})
		}
	}
}

func benchmarkClientPushMulti(b *testing.B, payload *Payload, useFast bool, numTokens int) {
	apnsID := "benchmark-multi-apns-id"

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("apns-id", apnsID)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	server.EnableHTTP2 = true
	server.StartTLS()

	tp := &MockTokenProvider{Token: "benchmark-token"}
	conf := appleapi.DefaultConfig()
	conf.TLSConfig.InsecureSkipVerify = true
	init := appleapi.ConfigureHTTPClientInitializer(&conf)
	client, err := NewClient(init, tp)
	if err != nil {
		b.Fatalf("NewClient failed: %v", err)
	}
	client.inner.Host = server.URL
	client.FastJson = useFast
	client.TokenLimits = 10000

	notification := &Notification{
		BundleID: "com.example.benchmark.multi",
		Type:     notification.Alert,
		Payload:  payload,
	}

	tokens := make([]string, numTokens)
	for i := 0; i < numTokens; i++ {
		tokens[i] = fmt.Sprintf("token-%d", i)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.PushMulti(ctx, notification, tokens)
		if err != nil {
			b.Fatalf("PushMulti failed: %v", err)
		}
	}
}

func BenchmarkClient_PushMulti(b *testing.B) {
	payload := benchmarkPayloads["Minimal"]
	tokenCounts := []int{1, 10, 100, 1000}

	for _, useFast := range []bool{false, true} {
		mode := "Standard"
		if useFast {
			mode = "Fast"
		}
		for _, count := range tokenCounts {
			b.Run(fmt.Sprintf("%s_%d_tokens", mode, count), func(b *testing.B) {
				benchmarkClientPushMulti(b, payload, useFast, count)
			})
		}
	}
}
