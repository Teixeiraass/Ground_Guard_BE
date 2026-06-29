package mqtt

import (
	"testing"

	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/stretchr/testify/require"
)

func TestNormalizeTopicPrefix(t *testing.T) {
	require.Equal(t, "ground-guard", NormalizeTopicPrefix(""))
	require.Equal(t, "gg", NormalizeTopicPrefix("gg"))
	require.Equal(t, "gg", NormalizeTopicPrefix("/gg/"))
}

func TestDeviceTopics(t *testing.T) {
	prefix := "ground-guard"
	deviceUID := "ESP32-ABC123"

	require.Equal(t, "ground-guard/devices/ESP32-ABC123/telemetry", DeviceTelemetryTopic(prefix, deviceUID))
	require.Equal(t, "ground-guard/devices/+/telemetry", DeviceTelemetryWildcard(prefix))
	require.Equal(t, "ground-guard/devices/ESP32-ABC123/commands", DeviceCommandTopic(prefix, deviceUID))
}

func TestDeviceUIDFromTelemetryTopic(t *testing.T) {
	prefix := "ground-guard"

	uid, ok := DeviceUIDFromTelemetryTopic(prefix, "ground-guard/devices/ESP32-ABC123/telemetry")
	require.True(t, ok)
	require.Equal(t, "ESP32-ABC123", uid)

	_, ok = DeviceUIDFromTelemetryTopic(prefix, "ground-guard/devices//telemetry")
	require.False(t, ok)

	_, ok = DeviceUIDFromTelemetryTopic(prefix, "invalid/topic")
	require.False(t, ok)
}

func TestNewPahoClientWhenDisabled(t *testing.T) {
	config := util.Config{
		MQTTEnabled: false,
	}

	client, err := NewPahoClient(config)
	require.NoError(t, err)
	require.IsType(t, &NoopClient{}, client)
}

func TestNewPahoClientRequiresBrokerURL(t *testing.T) {
	config := util.Config{
		MQTTEnabled: true,
	}

	client, err := NewPahoClient(config)
	require.Error(t, err)
	require.Nil(t, client)
	require.Contains(t, err.Error(), "MQTT_BROKER_URL")
}

func TestNoopClient(t *testing.T) {
	client := NewNoopClient()

	require.NoError(t, client.Publish("topic", []byte("payload")))
	require.NoError(t, client.Subscribe("topic", func(string, []byte) {}))
	require.Equal(t, "ground-guard", client.TopicPrefix())
	client.Close()
}

func TestPublishCommand(t *testing.T) {
	client := NewNoopClient()
	payload := []byte(`{"action":"irrigate","data":{"duration_seconds":30}}`)

	require.NoError(t, PublishCommand(client, "ESP32-001", payload))
}
