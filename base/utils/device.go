package utils

import (
	"encoding/json"
	"strings"
)

// DeviceInfo holds the parsed bits you actually care about.
type DeviceInfo struct {
	OS      string `json:"os,omitempty"`
	Browser string `json:"browser,omitempty"`
	Mobile  bool   `json:"mobile,omitempty"`
}

// ExtractDeviceInfo parses the User-Agent header into a JSON-friendly struct.
func ExtractDeviceInfo(ua string) json.RawMessage {
	info := DeviceInfo{}

	// Very basic sniffing – good enough for most cases.
	lower := strings.ToLower(ua)

	// OS
	switch {
	case strings.Contains(lower, "windows"):
		info.OS = "windows"
	case strings.Contains(lower, "mac"):
		info.OS = "macos"
	case strings.Contains(lower, "linux"):
		info.OS = "linux"
	case strings.Contains(lower, "android"):
		info.OS = "android"
		info.Mobile = true
	case strings.Contains(lower, "iphone"), strings.Contains(lower, "ipad"):
		info.OS = "ios"
		info.Mobile = true
	}

	// Browser
	switch {
	case strings.Contains(lower, "firefox"):
		info.Browser = "firefox"
	case strings.Contains(lower, "edg"):
		info.Browser = "edge"
	case strings.Contains(lower, "chrome"):
		info.Browser = "chrome"
	case strings.Contains(lower, "safari"):
		info.Browser = "safari"
	}

	// Marshal to JSON (or return nil if empty)
	if info.OS == "" && info.Browser == "" {
		return json.RawMessage(`{}`)
	}
	raw, _ := json.Marshal(info)
	return raw
}
