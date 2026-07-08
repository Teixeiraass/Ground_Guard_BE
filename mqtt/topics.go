package mqtt

import (
	"fmt"
	"strings"
)

const DefaultTopicPrefix = "ground-guard"

func NormalizeTopicPrefix(prefix string) string {
	if prefix == "" {
		return DefaultTopicPrefix
	}
	return strings.Trim(prefix, "/")
}

func DeviceTelemetryTopic(prefix, deviceUID string) string {
	return fmt.Sprintf("%s/devices/%s/telemetry", NormalizeTopicPrefix(prefix), deviceUID)
}

func DeviceTelemetryWildcard(prefix string) string {
	return fmt.Sprintf("%s/devices/+/telemetry", NormalizeTopicPrefix(prefix))
}

func DeviceCommandTopic(prefix, deviceUID string) string {
	return fmt.Sprintf("%s/devices/%s/commands", NormalizeTopicPrefix(prefix), deviceUID)
}

func DeviceUIDFromTelemetryTopic(prefix, topic string) (string, bool) {
	expectedPrefix := NormalizeTopicPrefix(prefix) + "/devices/"
	suffix := "/telemetry"

	if !strings.HasPrefix(topic, expectedPrefix) || !strings.HasSuffix(topic, suffix) {
		return "", false
	}

	deviceUID := strings.TrimPrefix(topic, expectedPrefix)
	deviceUID = strings.TrimSuffix(deviceUID, suffix)

	if deviceUID == "" || strings.Contains(deviceUID, "/") {
		return "", false
	}

	return deviceUID, true
}

func DeviceEventTopic(prefix, deviceUID string) string {
	return fmt.Sprintf("%s/devices/%s/events", NormalizeTopicPrefix(prefix), deviceUID)
}

func DeviceEventWildcard(prefix string) string {
	return fmt.Sprintf("%s/devices/+/events", NormalizeTopicPrefix(prefix))
}

func DeviceUIDFromEventTopic(prefix, topic string) (string, bool) {
	expectedPrefix := NormalizeTopicPrefix(prefix) + "/devices/"
	suffix := "/events"

	if !strings.HasPrefix(topic, expectedPrefix) || !strings.HasSuffix(topic, suffix) {
		return "", false
	}

	deviceUID := strings.TrimPrefix(topic, expectedPrefix)
	deviceUID = strings.TrimSuffix(deviceUID, suffix)

	if deviceUID == "" || strings.Contains(deviceUID, "/") {
		return "", false
	}

	return deviceUID, true
}
