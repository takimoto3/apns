package apns

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp" // Import go-cmp
	"github.com/takimoto3/apns/notification"
	"github.com/takimoto3/apns/notification/priority"
	"github.com/takimoto3/apns/payload" // Import the payload package
	"github.com/takimoto3/appleapi-core"
)

// MockTokenProvider is a mock implementation of token.Provider
type MockTokenProvider struct {
	Token string
	Err   error
}

func (m *MockTokenProvider) GetToken(t time.Time) (string, error) { // Corrected signature
	return m.Token, m.Err
}

type mockRoundTripper struct {
	resp *http.Response
}

func (m *mockRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return m.resp, nil
}

func TestNewClient(t *testing.T) {
	tp := &MockTokenProvider{}

	forProduction, err := NewClient(appleapi.DefaultHTTPClientInitializer(), tp)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	if forProduction.inner.Host != ProductionHost {
		t.Errorf("Expected host %s, but got %s for production client", ProductionHost, forProduction.inner.Host)
	}
	if !forProduction.TokenBase {
		t.Errorf("Expected TokenBase to be true, but got false")
	}
	if forProduction.TokenLimits != MaxTokens {
		t.Errorf("Expected TokenLimits to be %d, but got %d", MaxTokens, forProduction.TokenLimits)
	}

	forDevelopment, err := NewClient(appleapi.DefaultHTTPClientInitializer(), tp, appleapi.WithDevelopment())
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	if forDevelopment.inner.Host != DevelopmentHost {
		t.Errorf("Expected host %s, but got %s for development client", DevelopmentHost, forDevelopment.inner.Host)
	}
	if !forDevelopment.TokenBase {
		t.Errorf("Expected TokenBase to be true, but got false")
	}
	if forDevelopment.TokenLimits != MaxTokens {
		t.Errorf("Expected TokenLimits to be %d, but got %d", MaxTokens, forDevelopment.TokenLimits)
	}
}

func TestNewClientWithCert(t *testing.T) {
	tests := map[string]struct {
		args    *tls.Certificate
		wantErr string
	}{
		"nil case": {
			nil,
			"certificate cannot be nil",
		},
		"empty case": {
			&tls.Certificate{},
			"invalid certificate: empty certificate or private key",
		},
		"success case": {
			createCert(t),
			"",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			client, err := NewClientWithCert(tt.args)
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("NewClientWithCert expect return error")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("NewClientWithCert got unexpect error got:%v, want:%v", err.Error(), tt.wantErr)
				}
			} else {
				// success path
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if client.inner.Host != ProductionHost {
					t.Errorf("unexpected host: %s", client.inner.Host)
				}
				if client.TokenBase {
					t.Errorf("expect TokenBase to be false, but got true")
				}
				tr, ok := client.inner.HTTPClient.Transport.(*http.Transport)
				if !ok {
					t.Fatalf("transport must be *http.Transport")
				}
				if len(tr.TLSClientConfig.Certificates) == 0 {
					t.Fatalf("certificate must be loaded")
				}
			}
		})
	}
}

func TestClient_Push(t *testing.T) {
	now := time.Now().Add(time.Hour)
	expectedToken := "Bearer test-token"
	deviceToken := "test-device-token"
	apnsID := "123e4567-e89b-12d3-a456-4266554400a0" // Corrected to a valid UUID
	collapseID := "test-collapse-id"
	bundleID := "com.example.app"

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		expectedPath := fmt.Sprintf("%s%s", Path, deviceToken)
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		// Verify headers
		authHeader := r.Header.Get("authorization")
		if authHeader != expectedToken {
			t.Errorf("Expected Authorization header %s, got %s", expectedToken, authHeader)
		}
		if r.Header.Get("apns-push-type") != notification.Alert {
			t.Errorf("Expected apns-push-type %s, got %s", notification.Alert, r.Header.Get("apns-push-type"))
		}
		if r.Header.Get("apns-topic") != bundleID { // Assuming topic is bundleID for Alert type
			t.Errorf("Expected apns-topic %s, got %s", bundleID, r.Header.Get("apns-topic"))
		}
		if r.Header.Get("apns-id") != apnsID {
			t.Errorf("Expected apns-id %s, got %s", apnsID, r.Header.Get("apns-id"))
		}
		expectedExpiration := notification.NewEpochTime(now).String()
		if r.Header.Get("apns-expiration") != expectedExpiration {
			t.Errorf("Expected apns-expiration %s, got %s", expectedExpiration, r.Header.Get("apns-expiration"))
		}
		if r.Header.Get("apns-priority") != priority.Immediate.String() {
			t.Errorf("Expected apns-priority %s, got %s", priority.Immediate.String(), r.Header.Get("apns-priority"))
		}
		if r.Header.Get("apns-collapse-id") != collapseID {
			t.Errorf("Expected apns-collapse-id %s, got %s", collapseID, r.Header.Get("apns-collapse-id"))
		}

		// Verify body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read request body: %v", err)
		}
		var receivedPayload map[string]interface{}
		err = json.Unmarshal(body, &receivedPayload)
		if err != nil {
			t.Fatalf("Failed to unmarshal request body: %v", err)
		}
		expectedPayload := map[string]interface{}{"aps": map[string]interface{}{"alert": "test"}}
		if diff := cmp.Diff(expectedPayload, receivedPayload); diff != "" {
			t.Errorf("Payload mismatch (-want +got):\n%s", diff)
		}

		w.Header().Set("apns-id", apnsID)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"aps":{"alert":"test"}}`))
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	expiration := notification.NewEpochTime(now)
	n := &Notification{
		BundleID:    bundleID,
		DeviceToken: deviceToken,
		Type:        notification.Alert,
		APNsID:      apnsID,
		Expiration:  expiration,
		Priority:    priority.Immediate,
		CollapseID:  collapseID,
		Payload:     &Payload{APS: payload.APS{Alert: "test"}}, // Corrected Payload and APS initialization
	}

	// Token base cleint --------------------
	tp := &MockTokenProvider{Token: "test-token"}
	client, err := NewClientWithToken(tp)
	if err != nil {
		t.Fatalf("NewClientWithToken failed: %v", err)
	}
	tr, ok := client.inner.HTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Errorf("Client transport type check failed. Expected *http.Transport")
	}
	tr.TLSClientConfig.InsecureSkipVerify = true
	client.inner.Host = server.URL // Manually set the host for testing

	res, err := client.Push(context.Background(), n)
	if err != nil {
		t.Fatalf("Client.Push failed: %v", err)
	}

	if res.APNsID != apnsID {
		t.Errorf("Expected APNsID %s, got %s", apnsID, res.APNsID)
	}
	if res.UniqueID != "" { // Not set in mock server
		t.Errorf("Expected UniqueID to be empty, got %s", res.UniqueID)
	}
}

func TestCertificateBaseClient_Push(t *testing.T) {
	now := time.Now().Add(time.Hour)
	deviceToken := "test-device-token"
	apnsID := "123e4567-e89b-12d3-a456-4266554400a0" // Corrected to a valid UUID
	collapseID := "test-collapse-id"
	bundleID := "com.example.app"

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		expectedPath := fmt.Sprintf("%s%s", Path, deviceToken)
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		// Verify headers
		authHeader := r.Header.Get("authorization")
		if authHeader != "" {
			t.Errorf("Unexpected Authorization header sent. Expected empty string, but got: %s", authHeader)
		}
		if r.Header.Get("apns-push-type") != notification.Alert {
			t.Errorf("Expected apns-push-type %s, got %s", notification.Alert, r.Header.Get("apns-push-type"))
		}
		if r.Header.Get("apns-topic") != bundleID { // Assuming topic is bundleID for Alert type
			t.Errorf("Expected apns-topic %s, got %s", bundleID, r.Header.Get("apns-topic"))
		}
		if r.Header.Get("apns-id") != apnsID {
			t.Errorf("Expected apns-id %s, got %s", apnsID, r.Header.Get("apns-id"))
		}
		expectedExpiration := notification.NewEpochTime(now).String()
		if r.Header.Get("apns-expiration") != expectedExpiration {
			t.Errorf("Expected apns-expiration %s, got %s", expectedExpiration, r.Header.Get("apns-expiration"))
		}
		if r.Header.Get("apns-priority") != priority.Immediate.String() {
			t.Errorf("Expected apns-priority %s, got %s", priority.Immediate.String(), r.Header.Get("apns-priority"))
		}
		if r.Header.Get("apns-collapse-id") != collapseID {
			t.Errorf("Expected apns-collapse-id %s, got %s", collapseID, r.Header.Get("apns-collapse-id"))
		}

		// Verify body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read request body: %v", err)
		}
		var receivedPayload map[string]interface{}
		err = json.Unmarshal(body, &receivedPayload)
		if err != nil {
			t.Fatalf("Failed to unmarshal request body: %v", err)
		}
		expectedPayload := map[string]interface{}{"aps": map[string]interface{}{"alert": "test"}}
		if diff := cmp.Diff(expectedPayload, receivedPayload); diff != "" {
			t.Errorf("Payload mismatch (-want +got):\n%s", diff)
		}

		w.Header().Set("apns-id", apnsID)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"aps":{"alert":"test"}}`))
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	expiration := notification.NewEpochTime(now)
	n := &Notification{
		BundleID:    bundleID,
		DeviceToken: deviceToken,
		Type:        notification.Alert,
		APNsID:      apnsID,
		Expiration:  expiration,
		Priority:    priority.Immediate,
		CollapseID:  collapseID,
		Payload:     &Payload{APS: payload.APS{Alert: "test"}}, // Corrected Payload and APS initialization
	}

	cert := server.TLS.Certificates[0]
	client, err := NewClientWithCert(&cert) // Pass address of cert
	if err != nil {
		t.Fatalf("NewClientWithToken failed: %v", err)
	}
	tr, ok := client.inner.HTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Errorf("Client transport type check failed. Expected *http.Transport")
	}
	tr.TLSClientConfig.InsecureSkipVerify = true
	client.inner.Host = server.URL // Manually set the host for testing
	client.FastJson = false

	res, err := client.Push(context.Background(), n)
	if err != nil {
		t.Fatalf("Client.Push failed: %v", err)
	}

	if res.APNsID != apnsID {
		t.Errorf("Expected APNsID %s, got %s", apnsID, res.APNsID)
	}
	if res.UniqueID != "" { // Not set in mock server
		t.Errorf("Expected UniqueID to be empty, got %s", res.UniqueID)
	}

}

func TestClient_Push_Error(t *testing.T) {
	testCases := map[string]struct {
		args    Notification
		wantErr string
	}{
		"Empty BundleID": {
			Notification{DeviceToken: "DEVICE_TOKEN", Type: notification.Alert},
			"BundleID is required",
		},
		"Empty DeviceToken": {
			Notification{BundleID: "BUNDLE_ID", Type: notification.Alert},
			"DeviceToken is required",
		},
		"Invalid APNsID": {
			Notification{BundleID: "BUNDLE_ID", DeviceToken: "DEVICE_TOKEN", Type: notification.Alert, APNsID: "invalid-uuid"},
			"invalid APNsID",
		},
		"Notification type Location": {
			Notification{Type: notification.Location, BundleID: "BUNDLE_ID", DeviceToken: "DEVICE_TOKEN"},
			"location push type is not allowed with certificate-based connection",
		},
		"Large paylod error": {
			Notification{
				Type:        notification.Alert,
				BundleID:    "BUNDLE_ID",
				DeviceToken: "DEVICE_TOKEN",
				Payload:     &Payload{APS: payload.APS{Alert: strings.Repeat("A", 4077)}}, // 20byte {"aps":{"alert":{"A....."}}}
			},
			"payload too large: 4097 bytes",
		},
		"Large paylod": {
			Notification{
				Type:        notification.Alert,
				BundleID:    "BUNDLE_ID",
				DeviceToken: "DEVICE_TOKEN",
				Payload:     &Payload{APS: payload.APS{Alert: strings.Repeat("A", 4076)}},
			},
			"",
		},
		"Notification type VOIP(Large paylod error)": {
			Notification{
				Type:        notification.Voip,
				BundleID:    "BUNDLE_ID",
				DeviceToken: "DEVICE_TOKEN",
				Payload:     &Payload{APS: payload.APS{Alert: strings.Repeat("A", 5101)}},
			},
			"payload too large for Voip: 5121 bytes",
		},
		"Notification type VOIP(Large paylod)": {
			Notification{
				Type:        notification.Voip,
				BundleID:    "BUNDLE_ID",
				DeviceToken: "DEVICE_TOKEN",
				Payload:     &Payload{APS: payload.APS{Alert: strings.Repeat("A", 5099)}},
			},
			"",
		},
	}
	mockTransport := &mockRoundTripper{resp: &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"aps":{"alert":"test"}}`)),
		Header:     http.Header{"apns-id": []string{"dummy-id"}},
	}}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			dummyCert := createCert(t)
			cli, err := NewClientWithCert(dummyCert, appleapi.WithTransport(mockTransport))
			if err != nil {
				t.Fatal(err)
			}
			_, err = cli.Push(context.Background(), &tc.args)
			if err == nil {
				if tc.wantErr != "" {
					t.Fatal("expected an error, but got nil")
				}
				return
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("Expected message '%s', got '%s'", tc.wantErr, err)
			}
		})
	}
}

func TestClient_Push_ServerError(t *testing.T) {
	testCases := map[string]struct {
		statusCode int
		reason     string
		apnsID     string
		wantErr    string // Expected error string (APNsReason, or part of a generic error message)
	}{
		"BadRequest_BadDeviceToken": {
			statusCode: http.StatusBadRequest,
			reason:     "BadDeviceToken",
			apnsID:     "error-apns-id-1",
			wantErr:    "BadDeviceToken",
		},
		"Forbidden_MissingProviderToken": {
			statusCode: http.StatusForbidden,
			reason:     "MissingProviderToken",
			apnsID:     "error-apns-id-2",
			wantErr:    "MissingProviderToken",
		},
		"Forbidden_ExpiredProviderToken": {
			statusCode: http.StatusForbidden,
			reason:     "ExpiredProviderToken",
			apnsID:     "error-apns-id-3",
			wantErr:    "ExpiredProviderToken",
		},
		"Forbidden_InvalidProviderToken": {
			statusCode: http.StatusForbidden,
			reason:     "InvalidProviderToken",
			apnsID:     "error-apns-id-4",
			wantErr:    "InvalidProviderToken",
		},
		"Forbidden_Forbidden": {
			statusCode: http.StatusForbidden,
			reason:     "Forbidden",
			apnsID:     "error-apns-id-5",
			wantErr:    "Forbidden",
		},
		"BadRequest_BadPath": {
			statusCode: http.StatusBadRequest,
			reason:     "BadPath",
			apnsID:     "error-apns-id-6",
			wantErr:    "BadPath",
		},
		"BadRequest_BadMessageId": {
			statusCode: http.StatusBadRequest,
			reason:     "BadMessageId",
			apnsID:     "error-apns-id-7",
			wantErr:    "BadMessageId",
		},
		"BadRequest_MissingTopic": {
			statusCode: http.StatusBadRequest,
			reason:     "MissingTopic",
			apnsID:     "error-apns-id-8",
			wantErr:    "MissingTopic",
		},
		"BadRequest_TopicDiscrepancy": {
			statusCode: http.StatusBadRequest,
			reason:     "TopicDiscrepancy",
			apnsID:     "error-apns-id-9",
			wantErr:    "TopicDiscrepancy",
		},
		"BadRequest_DeviceTokenNotForTopic": {
			statusCode: http.StatusBadRequest,
			reason:     "DeviceTokenNotForTopic",
			apnsID:     "error-apns-id-10",
			wantErr:    "DeviceTokenNotForTopic",
		},
		"Gone_Unregistered": {
			statusCode: http.StatusGone,
			reason:     "Unregistered",
			apnsID:     "error-apns-id-11",
			wantErr:    "Unregistered",
		},
		"PayloadTooLarge": {
			statusCode: http.StatusRequestEntityTooLarge,
			reason:     "PayloadTooLarge",
			apnsID:     "error-apns-id-12",
			wantErr:    "PayloadTooLarge",
		},
		"TooManyRequests_TooManyProviderTokenUpdates": {
			statusCode: http.StatusTooManyRequests,
			reason:     "TooManyProviderTokenUpdates",
			apnsID:     "error-apns-id-13",
			wantErr:    "TooManyProviderTokenUpdates",
		},
		"TooManyRequests_TooManyRequests": {
			statusCode: http.StatusTooManyRequests,
			reason:     "TooManyRequests",
			apnsID:     "error-apns-id-14",
			wantErr:    "TooManyRequests",
		},
		"InternalServerError": {
			statusCode: http.StatusInternalServerError,
			reason:     "InternalServerError",
			apnsID:     "error-apns-id-15",
			wantErr:    "InternalServerError",
		},
		"ServiceUnavailable": {
			statusCode: http.StatusServiceUnavailable,
			reason:     "ServiceUnavailable",
			apnsID:     "error-apns-id-16",
			wantErr:    "ServiceUnavailable",
		},
		"UnknownError_501": { // e.g., 501 Not Implemented, when no APNs reason is provided
			statusCode: http.StatusNotImplemented,
			reason:     "", // Reason is empty for non-APNs errors or unknown ones
			apnsID:     "error-apns-id-17",
			wantErr:    fmt.Sprintf("APNs request failed with status %d", http.StatusNotImplemented),
		},
		"Success_ButNotError": { // Case where 200 OK does not result in an Error
			statusCode: http.StatusOK,
			reason:     "",
			apnsID:     "success-apns-id",
			wantErr:    "", // Expect no error
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("apns-id", tc.apnsID)
				w.WriteHeader(tc.statusCode)
				if tc.reason != "" {
					_, _ = w.Write([]byte(fmt.Sprintf(`{"reason": "%s"}`, tc.reason)))
				} else if tc.statusCode != http.StatusOK {
					_, _ = w.Write([]byte(`{"message": "Generic error"}`))
				}
			}))
			defer server.Close()

			mockInitializer := appleapi.DefaultHTTPClientInitializer()
			tp := &MockTokenProvider{Token: "dummy-token"}

			client, err := NewClient(mockInitializer, tp)
			if err != nil {
				t.Fatalf("NewClient failed: %v", err)
			}
			client.inner.Host = server.URL

			n := &Notification{
				BundleID:    "com.example.app",
				DeviceToken: "invalid-device-token",
				Type:        notification.Alert,
				Payload:     &Payload{APS: payload.APS{Alert: "test"}},
			}

			_, err = client.Push(context.Background(), n)
			if tc.wantErr == "" {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("Expected an error, but got nil")
			}

			apnsErr, isError := err.(*Error)

			if tc.reason != "" { // Expect Error if Reason is set
				if !isError {
					t.Fatalf("Expected error of type *Error for reason '%s', got %T", tc.reason, err)
				}
				if apnsErr.Reason != tc.wantErr {
					t.Errorf("Expected reason '%s', got '%s'", tc.wantErr, apnsErr.Reason)
				}
				if apnsErr.StatusCode != tc.statusCode {
					t.Errorf("Expected status code %d, got %d", tc.statusCode, apnsErr.StatusCode)
				}
			} else { // Expect generic error if Reason is not set
				if isError {
					t.Fatalf("Did not expect error of type *Error for status %d, but got one: %v", tc.statusCode, apnsErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("Expected error message to contain '%s', but got: %v", tc.wantErr, err)
				}
			}
		})
	}
}

func TestClient_Push_WithTimeout(t *testing.T) {
	// Reverted to httptest.NewServer approach due to synctest limitations with net/http's internal I/O blocking.
	// synctest is powerful for CPU-bound concurrency and deterministic time advancement with durably blocked goroutines,
	// but it struggles with real (or simulated via net.Pipe) network I/O which is not considered "durably blocked".
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Simulate a slow response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Mock initializer for NewClient
	mockInitializer := appleapi.DefaultHTTPClientInitializer()
	tp := &MockTokenProvider{Token: "dummy-token"}

	client, err := NewClient(mockInitializer, tp)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	client.inner.Host = server.URL                          // Manually set the host for testing
	client.inner.HTTPClient.Timeout = 50 * time.Millisecond // Manually set the timeout

	n := &Notification{
		BundleID:    "com.example.app",
		DeviceToken: "test-device-token",
		Type:        notification.Alert,
		Payload:     &Payload{APS: payload.APS{Alert: "test"}},
	}

	// Use context with a slightly longer timeout to ensure HTTP client's timeout is tested
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err = client.Push(ctx, n)
	duration := time.Since(start)

	if err == nil {
		t.Fatal("Expected a timeout error, but got nil")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		// ok
	} else if ne, ok := err.(net.Error); ok && ne.Timeout() {
		// ok
	} else {
		t.Fatalf("Expected timeout error, got: %v", err)
	}

	// Check for specific timeout errors
	if !strings.Contains(err.Error(), "Client.Timeout exceeded") && !strings.Contains(err.Error(), "context deadline exceeded") && !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Expected a timeout error, but got: %v (type %T)", err, err)
	}

	// Assert that the timeout occurred approximately within the expected time frame
	expectedTimeout := client.inner.HTTPClient.Timeout
	// Allow some buffer for real-world timing variations
	if duration < expectedTimeout || duration > expectedTimeout*3 {
		t.Errorf("Expected timeout duration around %v, but got %v", expectedTimeout, duration)
	}
}

func TestError_Error(t *testing.T) {
	testCases := []struct {
		name     string
		apnsErr  *Error
		expected string
	}{
		{
			name: "Without Timestamp",
			apnsErr: &Error{
				StatusCode: 400,
				Reason:     "BadDeviceToken",
				Timestamp:  0,
			},
			expected: "APNs error: status=400 reason=BadDeviceToken",
		},
		{
			name: "With Timestamp",
			apnsErr: &Error{
				StatusCode: 500,
				Reason:     "InternalServerError",
				Timestamp:  1678886400000,
			},
			expected: "APNs error: status=500 reason=InternalServerError timestamp=1678886400000",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if errStr := tc.apnsErr.Error(); errStr != tc.expected {
				t.Errorf("Error.Error() got = %q, want %q", errStr, tc.expected)
			}
		})
	}
}

func createCert(t *testing.T) *tls.Certificate {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test.local",
		},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	return &tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  priv,
	}
}

func TestClient_PushMulti(t *testing.T) {
	bundleID := "com.example.app"
	successApnsID := "success-apns-id"

	// Mock APNs server
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.URL.Path, Path)
		switch token {
		case "token-success-1", "token-success-2", "token-success-3":
			w.Header().Set("apns-id", successApnsID)
			w.WriteHeader(http.StatusOK)
		case "token-fail-baddevicetoken":
			w.Header().Set("apns-id", "fail-apns-id")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"reason":"BadDeviceToken"}`))
		case "token-fail-server-error":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"reason":"InternalServerError"}`))
		default:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"reason":"UnknownToken"}`))
		}
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	baseNotification := &Notification{
		BundleID: bundleID,
		Type:     notification.Alert,
		Payload:  &Payload{APS: payload.APS{Alert: "test"}},
	}
	invalidNotification := &Notification{
		Type:    notification.Alert,
		Payload: &Payload{APS: payload.APS{Alert: "test"}},
		// Missing BundleID
	}

	testCases := map[string]struct {
		notification    *Notification
		tokens          []string
		tokenLimits     int
		wantSuccesses   int
		wantFailures    int
		wantErrStr      string
		checkMultiError func(t *testing.T, err error)
	}{
		"All Success": {
			notification:  baseNotification,
			tokens:        []string{"token-success-1", "token-success-2", "token-success-3"},
			wantSuccesses: 3,
			wantFailures:  0,
		},
		"Partial Failure": {
			notification:  baseNotification,
			tokens:        []string{"token-success-1", "token-fail-baddevicetoken", "token-success-2"},
			wantSuccesses: 2,
			wantFailures:  1,
			wantErrStr:    "APNs batch failed",
			checkMultiError: func(t *testing.T, err error) {
				multiErr, ok := err.(*MultiError)
				if !ok {
					t.Fatalf("Expected *MultiError, got %T", err)
				}
				if _, exists := multiErr.Failures["token-fail-baddevicetoken"]; !exists {
					t.Errorf("Expected failure for 'token-fail-baddevicetoken'")
				}
			},
		},
		"First Token Fails": {
			notification:  baseNotification,
			tokens:        []string{"token-fail-server-error", "token-success-1"},
			wantSuccesses: 1, // Expect one response object even on failure
			wantFailures:  0, // Not a MultiError
			wantErrStr:    "InternalServerError",
		},
		"Empty Token List": {
			notification: baseNotification,
			tokens:       []string{},
			wantErrStr:   "token list is empty",
		},
		"Single Token": {
			notification:  baseNotification,
			tokens:        []string{"token-success-1"},
			wantSuccesses: 1,
			wantFailures:  0,
		},
		"Pre-validation Failure": {
			notification: invalidNotification,
			tokens:       []string{"token-success-1"},
			wantErrStr:   "BundleID is required",
		},
		"Custom Token Limit Exceeded": {
			notification: baseNotification,
			tokens:       []string{"token1", "token2", "token3", "token4", "token5"},
			tokenLimits:  3,
			wantErrStr:   "token limit exceeded: got 5 tokens, maximum allowed is 3",
		},
	}

	tp := &MockTokenProvider{Token: "test-token"}
	client, err := NewClientWithToken(tp)
	if err != nil {
		t.Fatalf("NewClientWithToken failed: %v", err)
	}
	tr, ok := client.inner.HTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("Client transport type check failed")
	}
	tr.TLSClientConfig.InsecureSkipVerify = true
	client.inner.Host = server.URL

	for name, tc := range testCases {
		if tc.tokenLimits == 0 {
			client.TokenLimits = MaxTokens
		} else {
			client.TokenLimits = tc.tokenLimits
		}
		t.Run(name, func(t *testing.T) {
			responses, err := client.PushMulti(context.Background(), tc.notification, tc.tokens)

			if tc.wantErrStr != "" {
				if err == nil {
					t.Fatalf("Expected error containing '%s', but got nil", tc.wantErrStr)
				}
				if !strings.Contains(err.Error(), tc.wantErrStr) {
					t.Errorf("Expected error '%s', got '%s'", tc.wantErrStr, err.Error())
				}
			} else if err != nil {
				t.Fatalf("Expected no error, but got: %v", err)
			}

			if len(responses) != tc.wantSuccesses {
				t.Errorf("Expected %d successful responses, got %d", tc.wantSuccesses, len(responses))
			}

			if multiErr, ok := err.(*MultiError); ok {
				if len(multiErr.Failures) != tc.wantFailures {
					t.Errorf("Expected %d failures, got %d", tc.wantFailures, len(multiErr.Failures))
				}
				if tc.checkMultiError != nil {
					tc.checkMultiError(t, err)
				}
			} else if tc.wantFailures > 0 {
				t.Errorf("Expected MultiError with %d failures, but didn't get a MultiError", tc.wantFailures)
			}
		})
	}
}
