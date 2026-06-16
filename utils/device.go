package utils

import (
	"net/http"
	"strings"
)

// String constants for device types
const (
	DeviceMobile  = "mobile"
	DeviceDesktop = "desktop"
)

// DetectDeviceType returns a string identifier ("mobile" or "desktop") based on User-Agent
func DetectDeviceType(r *http.Request) string {
	// Detect device type from User-Agent
	userAgent := r.Header.Get("User-Agent")
	return GetDeviceTypeFromUserAgent(userAgent)
}

// GetDeviceTypeFromUserAgent extracts device type from a User-Agent string
func GetDeviceTypeFromUserAgent(userAgent string) string {
	userAgent = strings.ToLower(userAgent)
	mobilePlatforms := []string{
		"android", "webos", "iphone", "ipad", "ipod", "blackberry", "windows phone",
	}

	for _, platform := range mobilePlatforms {
		if strings.Contains(userAgent, platform) {
			return DeviceMobile
		}
	}
	return DeviceDesktop
}
