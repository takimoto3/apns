//go:build race

package apns

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/takimoto3/apns/notification"
	"github.com/takimoto3/apns/payload"
)

func TestPushMulti_RaceCondition(t *testing.T) {
	tokenCount := 200
	tokens := make([]string, tokenCount)
	for i := 0; i < tokenCount; i++ {
		tokens[i] = fmt.Sprintf("token-%d", i)
	}

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// レース検知を容易にするため、微小な遅延を入れる
		time.Sleep(1 * time.Microsecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"apns-id":"race-id"}`))
	}))
	defer server.Close()

	client, _ := NewClientWithToken(&MockTokenProvider{Token: "test-token"})
	tr := client.inner.HTTPClient.Transport.(*http.Transport)
	tr.TLSClientConfig.InsecureSkipVerify = true
	client.inner.Host = server.URL
	client.TokenLimits = tokenCount

	n := &Notification{
		BundleID: "com.example.app",
		Type:     notification.Alert,
		Payload:  &Payload{APS: payload.APS{Alert: "test"}},
	}

	// --- execute testing ---

	t.Run("HighConcurrency_RaceCheck", func(t *testing.T) {
		_, err := client.PushMulti(context.Background(), n, tokens)
		if err != nil {
			t.Errorf("Unexpected error in race test: %v", err)
		}
	})
	t.Run("Concurrent_PushMulti_Calls", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 5; i++ { // 5つのスレッドから同時にPushMulti
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = client.PushMulti(context.Background(), n, tokens[:10])
			}()
		}
		wg.Wait()
	})

	t.Run("Context_Cancellation_Race", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Millisecond)
		defer cancel()

		_, _ = client.PushMulti(ctx, n, tokens)
	})
}
