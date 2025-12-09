// Package apns provides a client for sending push notifications to the
// Apple Push Notification service (APNs).
// It supports both token-based (.p8) and certificate-based (.p12) authentication.
package apns

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/takimoto3/apns/notification"
	"github.com/takimoto3/apns/notification/priority"
	"github.com/takimoto3/appleapi-core"
	"github.com/takimoto3/appleapi-core/token"
)

const (
	// ProductionHost is the APNs production server hostname.
	ProductionHost = "https://api.push.apple.com:443"
	// DevelopmentHost is the APNs development server hostname.
	DevelopmentHost = "https://api.sandbox.push.apple.com:443"

	// Path is the URL path for sending a notification.
	Path = "/3/device/"

	MaxTokens = 100
)

// MultiError holds a collection of errors that occurred during a batch operation.
type MultiError struct {
	// Failures is a map where the key is the device token that failed and the value is the error.
	Failures map[string]error
}

// Error implements the error interface.
func (e *MultiError) Error() string {
	return fmt.Sprintf("APNs batch failed for tokens: %v", maps.Keys(e.Failures))
}

// Error represents an error response from the APNs server.
type Error struct {
	// StatusCode is the HTTP status code returned by the server.
	StatusCode int
	// Reason is a string indicating the reason for the error.
	Reason string
	// Timestamp is the time at which the error occurred, in milliseconds since Unix epoch.
	// This field may be zero if the server did not provide a timestamp.
	Timestamp int64
}

// Error returns a string representation of the Error.
func (e *Error) Error() string {
	if e.Timestamp != 0 {
		return fmt.Sprintf("APNs error: status=%d reason=%s timestamp=%d", e.StatusCode, e.Reason, e.Timestamp)
	}
	return fmt.Sprintf("APNs error: status=%d reason=%s", e.StatusCode, e.Reason)
}

func (e *Error) TimeStamp() *time.Time {
	if e.Timestamp == 0 {
		return nil
	}
	tms := time.UnixMilli(e.Timestamp)
	return &tms
}

// Response represents a successful response from the APNs server.
type Response struct {
	// DeviceToken is the device token for which the notification was successfully sent.
	DeviceToken string
	// UniqueID is the unique ID of the notification, returned for development builds.
	// This is the same as apns-unique-id.
	UniqueID string
	// APNsID is the canonical UUID of the notification.
	// This is the same as apns-id.
	APNsID string
}

// Client is a client for sending notifications to the APNs.
type Client struct {
	inner       *appleapi.Client
	TokenLimits int
	TokenBase   bool

	// FastJson, if true, uses a high-performance custom JSON encoder for the payload.
	// This encoder is faster than the standard `encoding/json` but supports a limited
	// set of data types in the payload's CustomData.
	// See the documentation for `payload.MarshalJSONFast` for more details.
	// Defaults to true.
	FastJson bool
}

// NewClientWithToken creates a new APNs client that uses token-based authentication (.p8).
// It requires a `token.Provider` which is responsible for generating and refreshing authentication tokens.
func NewClientWithToken(tp token.Provider, opts ...appleapi.Option) (*Client, error) {
	return NewClient(appleapi.DefaultHTTPClientInitializer(), tp, opts...)
}

// NewClientWithCert creates a new APNs client that uses certificate-based authentication (.p12).
// It requires a `tls.Certificate` which is used to authenticate with the APNs server.
func NewClientWithCert(cert *tls.Certificate, opts ...appleapi.Option) (*Client, error) {
	if cert == nil {
		return nil, errors.New("certificate cannot be nil")
	}
	if len(cert.Certificate) == 0 || cert.PrivateKey == nil {
		return nil, errors.New("invalid certificate: empty certificate or private key")
	}
	config := appleapi.DefaultConfig()
	config.TLSConfig = &tls.Config{
		MinVersion:   tls.VersionTLS13, // APNs requires at least TLS 1.2, but we enforce 1.3 for better security.
		Certificates: []tls.Certificate{*cert},
	}
	return NewClient(appleapi.ConfigureHTTPClientInitializer(&config), nil, opts...)
}

// NewClient creates a new APNs client with a custom HTTP client initializer and token provider.
// This is an advanced constructor that allows for fine-grained control over the HTTP client.
// In most cases, `NewClientWithToken` or `NewClientWithCert` should be used instead.
func NewClient(initializer appleapi.HTTPClientInitializer, tp token.Provider, opts ...appleapi.Option) (*Client, error) {
	cli, err := appleapi.NewClient(initializer, ProductionHost, tp, opts...)
	if err != nil {
		return nil, err
	}
	if cli.Development {
		cli.Host = DevelopmentHost
	}
	return &Client{inner: cli, TokenBase: tp != nil, TokenLimits: MaxTokens, FastJson: true}, nil
}

// Push sends a push notification to the APNs.
// It validates the notification, marshals the payload, and sends the request.
// It returns a `Response` on success, or an `error` if something goes wrong.
// If the APNs server returns an error, it will be of type `*Error`.
//
// Note: Even if an error occurs, the returned `Response` object might still
// contain some information, such as the APNsID. This can be useful for debugging
// or preventing duplicate notifications.
func (cli *Client) Push(ctx context.Context, n *Notification) (*Response, error) {
	if err := n.Validate(); err != nil {
		return nil, err
	}
	if n.Type == notification.Location && !cli.TokenBase {
		return nil, errors.New("location push type is not allowed with certificate-based connection")
	}
	body, err := cli.newBody(n)
	if err != nil {
		return nil, err
	}

	req, err := cli.newRequest(ctx, n, body)
	if err != nil {
		return nil, err
	}

	resp, err := cli.do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send APNs request: %w", err)
	}
	defer resp.Body.Close()

	return cli.handleResponse(resp)
}

func (cli *Client) do(req *http.Request) (*http.Response, error) {
	if cli.TokenBase {
		return cli.inner.Do(req) // includes token handling
	}
	return cli.inner.HTTPClient.Do(req) // certificate based, raw http client
}

func (cli *Client) handleResponse(resp *http.Response) (*Response, error) {
	response := &Response{
		APNsID: resp.Header.Get("apns-id"),
	}

	if cli.inner.Development {
		response.UniqueID = resp.Header.Get("apns-unique-id")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, err
	}

	if resp.StatusCode == http.StatusOK {
		return response, nil
	}

	var errPayload struct {
		Reason    string `json:"reason"`
		Timestamp int64  `json:"timestamp,omitempty"`
	}

	if len(body) == 0 {
		return response, fmt.Errorf("APNs transport error: empty response body, status=%d", resp.StatusCode)
	}
	// Check if the response body contains an APNs error reason
	if err := json.Unmarshal(body, &errPayload); err != nil {
		// If unmarshalling fails, it's not a structured APNs error,
		// treat it as a generic HTTP error.
		return response, fmt.Errorf("APNs request failed with status %d: failed to parse error response: %w", resp.StatusCode, err)
	}

	// Only return Error if a reason is explicitly provided in the response body.
	// Otherwise, it's a generic HTTP error or an unknown APNs error without a specific reason.
	if errPayload.Reason != "" {
		apnsErr := &Error{
			StatusCode: resp.StatusCode,
			Reason:     errPayload.Reason,
			Timestamp:  errPayload.Timestamp,
		}
		return response, apnsErr
	}

	// If no specific APNs reason is provided, return a generic error.
	return response, fmt.Errorf("APNs request failed with status %d", resp.StatusCode)
}

func (cli *Client) newBody(n *Notification) ([]byte, error) {
	var err error
	var body []byte
	if cli.FastJson {
		body, err = n.Payload.MarshalJSONFast()
		if err != nil {
			return nil, fmt.Errorf("fail to marshal json: %w", err)
		}
	} else {
		body, err = json.Marshal(n.Payload)
		if err != nil {
			return nil, fmt.Errorf("fail to marshal json: %w", err)
		}
	}
	if n.Type == notification.Voip {
		if len(body) > 5120 {
			return nil, fmt.Errorf("payload too large for Voip: %d bytes", len(body))
		}
	} else {
		if len(body) > 4096 {
			return nil, fmt.Errorf("payload too large: %d bytes", len(body))
		}
	}
	return body, nil
}

func (cli *Client) newRequest(ctx context.Context, n *Notification, body []byte) (*http.Request, error) {
	path := cli.inner.Host + Path + url.PathEscape(n.DeviceToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, path, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("apns-push-type", string(n.Type))
	req.Header.Set("apns-topic", n.Topic())

	if n.APNsID != "" {
		req.Header.Set("apns-id", n.APNsID)
	}
	if n.Expiration != nil {
		req.Header.Set("apns-expiration", n.Expiration.String())
	}
	if n.Priority != priority.None {
		req.Header.Set("apns-priority", n.Priority.String())
	}
	if n.CollapseID != "" {
		req.Header.Set("apns-collapse-id", n.CollapseID)
	}
	return req, nil
}

// PushMulti sends the same push notification to multiple device tokens concurrently.
// It validates the notification, marshals the payload, and sends the requests in parallel.
//
// It returns a slice of `*Response` for all successful deliveries and a single
// `*MultiError` that contains all failures. If all notifications are sent successfully,
// the error will be nil.
//
// This method is more efficient than calling `Push` in a loop as it utilizes
// goroutines to send notifications concurrently.
func (cli *Client) PushMulti(ctx context.Context, n *Notification, tokens []string) ([]*Response, error) {
	if len(tokens) == 0 {
		return nil, errors.New("token list is empty")
	}
	if len(tokens) > cli.TokenLimits {
		return nil, fmt.Errorf("token limit exceeded: got %d tokens, maximum allowed is %d", len(tokens), cli.TokenLimits)
	}
	successes := make([]*Response, 0, len(tokens))

	firstToken := tokens[0]
	n.DeviceToken = firstToken
	if err := n.Validate(); err != nil {
		return nil, err
	}
	if n.Type == notification.Location && !cli.TokenBase {
		return nil, errors.New("location push type is not allowed with certificate-based connection")
	}

	body, err := cli.newBody(n)
	if err != nil {
		return nil, err
	}
	req, err := cli.newRequest(ctx, n, body)
	if err != nil {
		return nil, err
	}

	resp, err := cli.do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send APNs request: %w", err)
	}
	defer resp.Body.Close()

	response, err := cli.handleResponse(resp)
	if err != nil {
		return []*Response{response}, err
	}

	response.DeviceToken = firstToken
	successes = append(successes, response)

	remaining := tokens[1:]
	failures := make(map[string]error, len(remaining)/2)

	type result struct {
		Token string
		Resp  *Response
		Err   error
	}
	results := make(chan result, len(remaining))
	var wg sync.WaitGroup

	for _, token := range remaining {
		wg.Add(1)
		go func(token string) {
			defer wg.Done()
			if err := ctx.Err(); err != nil {
				results <- result{Token: token, Err: err}
				return
			}

			notification := n.Clone()
			notification.DeviceToken = token

			req, err := cli.newRequest(ctx, notification, body)
			if err != nil {
				results <- result{Token: token, Err: err}
				return
			}
			resp, err := cli.do(req)
			if err != nil {
				results <- result{Token: token, Err: err}
				return
			}
			defer resp.Body.Close()
			response, err := cli.handleResponse(resp)
			results <- result{Token: token, Resp: response, Err: err}
		}(token)
	}
	wg.Wait()
	close(results)

	for res := range results {
		if res.Err != nil {
			failures[res.Token] = res.Err
		} else {
			response := res.Resp
			response.DeviceToken = res.Token
			successes = append(successes, response)
		}
	}

	if len(failures) > 0 {
		return successes, &MultiError{Failures: failures}
	}
	return successes, nil
}
