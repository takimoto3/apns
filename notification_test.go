package apns_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/takimoto3/apns"
	"github.com/takimoto3/apns/notification"
	"github.com/takimoto3/apns/notification/priority"
	"github.com/takimoto3/apns/payload"
)

func TestNotification_Topic(t *testing.T) {
	// Bundle ID used for testing
	const bundleID = "com.example.myapp"

	// Table of test cases
	tests := []struct {
		name     string
		pushType notification.PushType
		want     string // Expected value (The final calculated string is hardcoded here)
	}{
		// 1. Default/Alert/Background/Mdm (No suffix, just the BundleID)
		{"Alert", notification.Alert, "com.example.myapp"},
		{"Background", notification.Background, "com.example.myapp"},
		{"Mdm", notification.Mdm, "com.example.myapp"},
		{"Default_Fallback", notification.PushType("unknown"), "com.example.myapp"},

		// 2. Types with special suffixes
		{"Complication", notification.Complication, "com.example.myapp.complication"},
		{"Controls", notification.Controls, "com.example.myapp.push-type.controls"},
		{"Fileprovider", notification.Fileprovider, "com.example.myapp.pushkit.fileprovider"},
		{"Liveactivity", notification.Liveactivity, "com.example.myapp.push-type.liveactivity"},
		{"Location", notification.Location, "com.example.myapp.location-query"},
		{"Pushtotalk", notification.Pushtotalk, "com.example.myapp.voip-ptt"},
		{"Voip", notification.Voip, "com.example.myapp.voip"},
		{"Widgets", notification.Widgets, "com.example.myapp.push-type.widgets"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the Notification struct instance
			n := apns.Notification{
				// Use the hardcoded Bundle ID string literal
				BundleID: bundleID,
				Type:     tt.pushType,
			}

			// Execute the Topic() method
			got := n.Topic()

			// Verify the result
			if !cmp.Equal(got, tt.want) {
				t.Errorf("Topic() with PushType %s (-got +want):\n%s", tt.pushType, cmp.Diff(got, tt.want))
			}
		})
	}
}

func TestNotification_Validate(t *testing.T) {
	validPayload := &apns.Payload{
		APS: payload.APS{
			Alert: &payload.Alert{
				Title: "title",
				Body:  "body",
			},
		},
	}

	testCases := map[string]struct {
		notification *apns.Notification
		expectErr    bool
		errContains  string
	}{
		"Valid notification": {
			notification: &apns.Notification{
				BundleID:    "com.example.app",
				DeviceToken: "some-device-token",
				Type:        notification.Alert,
				Payload:     validPayload,
			},
			expectErr: false,
		},
		"Missing BundleID": {
			notification: &apns.Notification{
				DeviceToken: "some-device-token",
				Type:        notification.Alert,
			},
			expectErr:   true,
			errContains: "BundleID is required",
		},
		"Missing DeviceToken": {
			notification: &apns.Notification{
				BundleID: "com.example.app",
				Type:     notification.Alert,
			},
			expectErr:   true,
			errContains: "DeviceToken is required",
		},
		"Missing PushType": {
			notification: &apns.Notification{
				BundleID:    "com.example.app",
				DeviceToken: "some-device-token",
			},
			expectErr:   true,
			errContains: "apns-push-type is required",
		},
		"Invalid PushType": {
			notification: &apns.Notification{
				BundleID:    "com.example.app",
				DeviceToken: "some-device-token",
				Type:        "invalid-push-type",
			},
			expectErr:   true,
			errContains: "invalid apns-push-type",
		},
		"Invalid APNsID": {
			notification: &apns.Notification{
				BundleID:    "com.example.app",
				DeviceToken: "some-device-token",
				Type:        notification.Alert,
				APNsID:      "invalid-uuid",
			},
			expectErr:   true,
			errContains: "invalid APNsID",
		},
		"Valid APNsID": {
			notification: &apns.Notification{
				BundleID:    "com.example.app",
				DeviceToken: "some-device-token",
				Type:        notification.Alert,
				APNsID:      uuid.NewString(),
				Payload:     validPayload,
			},
			expectErr: false,
		},
		"Invalid Priority": {
			notification: &apns.Notification{
				BundleID:    "com.example.app",
				DeviceToken: "some-device-token",
				Type:        notification.Alert,
				Priority:    999, // Invalid priority
			},
			expectErr:   true,
			errContains: "invalid apns-priority",
		},
		"Valid Priority None": {
			notification: &apns.Notification{
				BundleID:    "com.example.app",
				DeviceToken: "some-device-token",
				Type:        notification.Alert,
				Payload:     validPayload,
			},
			expectErr: false,
		},
		"Valid Priority Immediate": {
			notification: &apns.Notification{
				BundleID:    "com.example.app",
				DeviceToken: "some-device-token",
				Type:        notification.Alert,
				Priority:    priority.Immediate,
				Payload:     validPayload,
			},
			expectErr: false,
		},
		"Valid Priority Conserve": {
			notification: &apns.Notification{
				BundleID:    "com.example.app",
				DeviceToken: "some-device-token",
				Type:        notification.Alert,
				Priority:    priority.Conserve,
				Payload:     validPayload,
			},
			expectErr: false,
		},
		"Missing Payload for Alert": {
			notification: &apns.Notification{
				BundleID:    "com.example.app",
				DeviceToken: "some-device-token",
				Type:        notification.Alert,
			},
			expectErr:   true,
			errContains: "Payload is required for alert push type",
		},
		"Valid Priority PowerOnly": {
			notification: &apns.Notification{
				BundleID:    "com.example.app",
				DeviceToken: "some-device-token",
				Type:        notification.Alert,
				Priority:    priority.PowerOnly,
				Payload:     validPayload,
			},
			expectErr: false,
		},
		"Missing Payload for Background": {
			notification: &apns.Notification{
				BundleID:    "com.example.app",
				DeviceToken: "some-device-token",
				Type:        notification.Background,
			},
			expectErr:   true,
			errContains: "Payload is required for background push type",
		},
		"Invalid Payload": {
			notification: &apns.Notification{
				BundleID:    "com.example.app",
				DeviceToken: "some-device-token",
				Type:        notification.Alert,
				Payload: &apns.Payload{
					APS: payload.APS{}, // Missing alert for alert type
				},
			},
			expectErr:   true,
			errContains: "aps dictionary must not be empty",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := tc.notification.Validate()
			if tc.expectErr {
				if err == nil {
					t.Errorf("expected an error, but got nil")
				} else if !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("expected error to contain %q, but got %q", tc.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("did not expect an error, but got: %v", err)
				}
			}
		})
	}
}
