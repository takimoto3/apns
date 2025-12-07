// package payload provides types for constructing the payload of an APNs notification.
package payload

// Alert represents the `alert` dictionary within the `aps` payload.
// It defines the content and appearance of the user-facing notification.
//
// For more details, see the Apple Developer Documentation:
// https://developer.apple.com/documentation/usernotifications/generating_a_remote_notification
type Alert struct {
	// Title is the title of the notification.
	Title string `json:"title,omitempty"`

	// Subtitle is the subtitle of the notification.
	Subtitle string `json:"subtitle,omitempty"`

	// Body is the main content of the notification.
	Body string `json:"body,omitempty"`

	// LaunchImage is the name of an image file in the app bundle to be displayed
	// when the user launches the app from the notification.
	LaunchImage string `json:"launch-image,omitempty"`

	// ActionLocKey is the key for a localized string to be used as the title
	// of the action button.
	ActionLocKey string `json:"action-loc-key,omitempty"`

	// --- Localization ---

	// LocKey is the key for a localized string in the app's `Localizable.strings`
	// file to be used for the notification's body.
	LocKey string `json:"loc-key,omitempty"`

	// LocArgs are the variable string values to appear in place of the format
	// specifiers in `loc-key`.
	LocArgs []string `json:"loc-args,omitempty"`

	// TitleLocKey is the key for a localized string to be used for the
	// notification's title.
	TitleLocKey string `json:"title-loc-key,omitempty"`

	// TitleLocArgs are the arguments for `title-loc-key`.
	TitleLocArgs []string `json:"title-loc-args,omitempty"`

	// SubtitleLocKey is the key for a localized string to be used for the
	// notification's subtitle.
	SubtitleLocKey string `json:"subtitle-loc-key,omitempty"`

	// SubtitleLocArgs are the arguments for `subtitle-loc-key`.
	SubtitleLocArgs []string `json:"subtitle-loc-args,omitempty"`
}
