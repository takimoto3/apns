// package notification provides types related to the metadata of an APNs notification.
package notification

// PushType corresponds to the `apns-push-type` header field.
type PushType = string

const (
	// Alert push type is used for notifications that display an alert, play a sound, or badge the app's icon.
	Alert PushType = "alert"
	// Background push type is used for notifications that deliver content in the background.
	Background PushType = "background"
	// Complication push type is used for updating a watchOS app’s complication.
	Complication PushType = "complication"
	// Controls push type is used to update the controls of a widget.
	Controls PushType = "controls"
	// Fileprovider push type is used to signal changes to a File Provider extension.
	Fileprovider PushType = "fileprovider"
	// Liveactivity push type is used for updating a Live Activity.
	Liveactivity PushType = "liveactivity"
	// Location push type is used for location-based notifications.
	Location PushType = "location"
	// Mdm push type is for notifications to a device enrolled in a Mobile Device Management (MDM) service.
	Mdm PushType = "mdm"
	// Pushtotalk push type is for Push to Talk notifications.
	Pushtotalk PushType = "pushtotalk"
	// Voip push type is for VoIP notifications.
	Voip PushType = "voip"
	// Widgets push type is for updating a widget's content.
	Widgets PushType = "widgets"
)
