# apns
apns is a simple Go client for Apple Push Notification Service (APNs).
It is built on top of [appleapi-core](https://github.com/takimoto3/appleapi-core) and provides:

* Token-based and certificate-based authentication
* Automatic APNs topic handling based on BundleID and PushType
* High-performance JSON marshaling (optional, enabled by default)
* Simple and type-safe APS payload construction
* Flexible custom data support
* Clear error handling with typed responses

## Installation

```bash
go get github.com/takimoto3/apns
```

## Usage

This library supports two primary authentication methods for connecting to APNs: **Token-based Authentication (HTTP/2)** and **Certificate-based Authentication**. The choice of authentication method primarily affects how the APNs client is created. Once the client is initialized, the process for creating and sending notifications is largely the same.

### 1. Client Creation

> **Note:** Both `apns.NewClientWithToken` and `apns.NewClientWithCert` accept optional `appleapi.Option` arguments. These options, provided by the underlying appleapi-core library, allow for advanced client customization (e.g., setting the environment with appleapi.WithDevelopment()). Note that these options are for configuring the underlying appleapi-core client, and it is not possible to provide a custom HTTP client directly through them. Refer to the [appleapi-core documentation](https://github.com/takimoto3/appleapi-core?tab=readme-ov-file#configuration-options) for a full list of available options.

#### Token-based Client

Token-based authentication uses JSON Web Tokens (JWT). It is more flexible and recommended for most modern applications. You will need your APNs Auth Key (`.p8` file path), Key ID, and Team ID.

```go

authKeyPath := "path/to/your/AuthKey_KEYID.p8"
keyID       := "YOUR_KEY_ID"
teamID      := "YOUR_TEAM_ID"

privateKey, err := token.LoadPKCS8File(authKeyPath)
if err != nil {
	log.Fatalf("Failed to load private key from %s: %v", authKeyPath, err)
}

tokenProvider, err := token.NewTokenProvider(privateKey, keyID, teamID)
if err != nil {
	log.Fatalf("Failed to create token provider: %v", err)
}

// Create a new APNs client using the token provider.
// **Production is the default environment.**
// Use `appleapi.WithDevelopment()` to switch to the sandbox environment.
client, err := apns.NewClientWithToken(tokenProvider, appleapi.WithDevelopment())
if err != nil {
	log.Fatalf("Failed to create APNs client: %v", err)
}
```

#### Certificate-based Client

Certificate-based authentication uses a TLS certificate (`.p12` or `.pem` file) for authentication. While still supported, Apple recommends token-based authentication for new development. You will need your certificate (`.p12` or `.pem` format) and its password if applicable.

```go

certificatePath := "path/to/your/cert.p12"
certificatePass := "YOUR_CERT_PASSWORD"

tlsCert, err := certificate.LoadP12File(certificatePath, certificatePass)
if err != nil {
	log.Fatalf("Failed to load .p12 certificate: %v", err)
}

// Create a new APNs client using the TLS certificate.
// **Production is the default environment.**
// Use `appleapi.WithDevelopment()` to switch to the sandbox environment.
client, err := apns.NewClientWithCert(tlsCert, appleapi.WithDevelopment())
if err != nil {
	log.Fatalf("Failed to create APNs client: %v", err)
}
```

#### Optional: Fast JSON Marshaling

> By default, APNs payloads are marshaled using the optimized JSON implementation for better performance.
> This optimization is enabled by default (`client.FastJson = true`) and reduces allocations, improving throughput when sending many notifications.
> To maximize compatibility or disable the optimization, set:
>```go
>client.FastJson = false
>```

> Enabling `client.FastJson` significantly improves performance, reducing both CPU time and memory allocations. In typical benchmarks on an Apple M1 machine, the fast JSON marshaler is roughly 2–5x faster and uses much less memory per payload compared to the standard JSON marshaler.
> You can measure performance in your own environment by running:
>```bash
>go test -bench=. -benchmem payload_benchmark_test.go
>```

### 2. Notification Creation

Once you have an initialized `apns.Client` (either token-based or certificate-based), the next step is to construct the notification. This involves defining the payload (the `aps` dictionary and any custom data) and setting various APNs headers.

```go

// Create an APNs payload (the 'aps' dictionary)
aps := &payload.APS{
	Alert: &payload.Alert{
		Title: "Notification Title!",
		Body:  "This is the body of the notification.",
	},
	Badge: 1,
	Sound: "default",
	// Add other APS fields as needed, e.g., category, thread-id, interruption-level
	// Category: "NEWS_CATEGORY",
}

p := &apns.Payload{
	APS: aps,
	// Optionally, add custom data here that your app can read
	CustomData: map[string]any{
		"article_id": "12345",
		"action":     "view",
	},
}

// Create the notification object with device token, payload, and headers
n := &notification.Notification{
	BundleID:    bundleID,                                 // Required for validation. The `apns-topic` header is automatically derived from BundleID and PushType, so you do not need to set it manually.
	DeviceToken: deviceToken,
	Payload:     p,
	PushType:    notification.Alert,                       // e.g., Alert, Background, Voip
	// No explicit `Topic` field is needed here.
	Expiration:  notification.NewEpochTime(time.Now().Add(time.Hour)), // Notification expires in 1 hour
	Priority:    priority.Immediate,                   // Immediate or Conserve (for silent updates)
	// APNsID:      "a-unique-uuid",                        // Optional: Custom APNs-ID
	// CollapseID:  "my-collapse-id",                       // Optional: For grouping notifications
}
```

### 3. Sending the Notification

With the client created and the notification constructed, you can now send it using the client's `Push` method. Remember to use a `context` with a timeout to prevent indefinite hangs.

```go

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel() // Ensure the context is cancelled when done

// 'client' is your initialized apns.Client
// 'n' is your constructed *notification.Notification
resp, err := client.Push(ctx, n)
if err != nil {
	// Handle errors from sending the request or network issues.
	// This could be an *apns.Error if the APNs server responded with an error.
	if apnsErr, ok := err.(*apns.Error); ok {
		log.Fatalf("APNs responded with error: %v", apnsErr) // apnsErr.Error() will be called automatically
	}
	log.Fatalf("Failed to send notification or network issue: %v", err)
}

// If err is nil, the notification was successfully accepted by APNs.
// The APNsID is a canonical UUID that identifies the notification and is crucial
// for debugging and preventing duplicate notifications on the APNs side.
log.Printf("Notification successfully accepted by APNs! APNs ID: %s", resp.APNsID)
if resp.UniqueID != "" {
	log.Printf("Unique ID (development only): %s", resp.UniqueID)
}
```

### 4. Sending to Multiple Devices (`PushMulti`)

For sending the same notification to multiple device tokens, the `PushMulti` method provides an efficient, concurrent way to handle batch operations. It returns all successful responses and a single `MultiError` containing all failures.

```go
tokens := []string{
	"a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
	"b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3",
	"c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4",
}

// 'client' is your initialized apns.Client
// 'n' is your constructed *notification.Notification (DeviceToken will be ignored)
successes, err := client.PushMulti(ctx, n, tokens)
if err != nil {
	// A MultiError indicates that some, but not necessarily all, requests failed.
	if multiErr, ok := err.(*apns.MultiError); ok {
		log.Printf("%d notifications failed:", len(multiErr.Failures))
		for token, reason := range multiErr.Failures {
			log.Printf("  - Token: %s, Reason: %v", token, reason)
		}
	} else {
		// A different error occurred before the batch operation started.
		log.Fatalf("Failed to send notifications: %v", err)
	}
}

log.Printf("%d notifications sent successfully!", len(successes))
for _, resp := range successes {
	log.Printf("  - Token: %s, APNs ID: %s", resp.DeviceToken, resp.APNsID)
}
```

#### Optional: Token Limit

> To prevent overwhelming the APNs service and to manage client resources, `PushMulti` enforces a limit on the number of tokens that can be sent in a single call.
> The default limit is **100** tokens. If the number of tokens provided exceeds this limit, `PushMulti` will return an error before sending any notifications.
> You can customize this limit by setting the `TokenLimits` field on the `Client` instance:
>```go
>client.TokenLimits = 200 // Set a custom limit
>```

## 5. Quick Start

> **Note:** In a production environment, always check errors for all function calls.
<br> The Quick Start example simplifies error handling for brevity.

```go
import (
	"context"
	"log"
	"time"

	"github.com/takimoto3/apns"
	"github.com/takimoto3/apns/notification"
	"github.com/takimoto3/apns/payload"
	"github.com/takimoto3/appleapi-core/token"
)

// Example usage (replace with your actual values)
var keyID = "YOUR_KEY_ID"
var teamID = "YOUR_TEAM_ID"
var bundleID = "com.example.app"
var deviceToken = "your_device_token_here"

privateKey, err := token.LoadPKCS8File("AuthKey.p8")
if err != nil {
	log.Fatalf("Failed to load private key: %v", err)
}
provider, err := token.NewTokenProvider(privateKey, keyID, teamID)
if err != nil {
	log.Fatalf("Failed to create token provider: %v", err)
}
client, err := apns.NewClientWithToken(provider, appleapi.WithDevelopment())
if err != nil {
	log.Fatalf("Failed to create APNs client: %v", err)
}

aps := &payload.APS{ Alert: &payload.Alert{ Title: "Hi", Body: "Test" } }
p := &apns.Payload{ APS: aps }

n := &notification.Notification{
    BundleID:    bundleID,
    DeviceToken: deviceToken,
    Payload:     p,
    PushType:    notification.Alert,
}

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
resp, err := client.Push(ctx, n)
if err != nil {
	if apnsErr, ok := err.(*apns.Error); ok {
		log.Fatalf("APNs responded with error: %v", apnsErr)
	}
	log.Fatalf("Failed to send notification: %v", err)
}

log.Printf("Notification sent successfully! APNs ID: %s", resp.APNsID)
```

## References

This project leverages information and best practices from the official Apple Developer documentation for Push Notifications:

*   [Setting up a remote notification server](https://developer.apple.com/documentation/usernotifications/setting-up-a-remote-notification-server)
*   [Establishing a token-based connection to APNs](https://developer.apple.com/documentation/usernotifications/establishing-a-token-based-connection-to-apns)
*   [Establishing a certificate-based connection to APNs](https://developer.apple.com/documentation/usernotifications/establishing-a-certificate-based-connection-to-apns)
*   [Sending notification requests to APNs](https://developer.apple.com/documentation/usernotifications/sending-notification-requests-to-apns)
*   [Generating a remote notification](https://developer.apple.com/documentation/usernotifications/generating-a-remote-notification)
*   [Handling notification responses from APNs](https://developer.apple.com/documentation/usernotifications/handling-notification-responses-from-apns)

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
