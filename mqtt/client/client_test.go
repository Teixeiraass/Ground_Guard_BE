package client_test

import (
	"testing"

	"github.com/Teixeiraass/ground_guard_be/mqtt"
	"github.com/Teixeiraass/ground_guard_be/mqtt/client"
	"github.com/Teixeiraass/ground_guard_be/mqtt/publisher"
	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/stretchr/testify/require"
)

func TestTopics(t *testing.T) {
	require.Equal(t, mqtt.DefaultTopicPrefix, mqtt.NormalizeTopicPrefix(""))
	require.Equal(t, "custom-prefix", mqtt.NormalizeTopicPrefix("custom-prefix/"))
	require.Equal(t, "custom-prefix", mqtt.NormalizeTopicPrefix("/custom-prefix/"))

	deviceUID := "ESP32-123456"

	expectedTelemetryTopic := "ground-guard/devices/ESP32-123456/telemetry"
	expectedTelemetryWildcard := "ground-guard/devices/+/telemetry"
	expectedCommandTopic := "ground-guard/devices/ESP32-123456/commands"

	require.Equal(t, expectedTelemetryTopic, mqtt.DeviceTelemetryTopic("", deviceUID))
	require.Equal(t, expectedTelemetryWildcard, mqtt.DeviceTelemetryWildcard(""))
	require.Equal(t, expectedCommandTopic, mqtt.DeviceCommandTopic("", deviceUID))

	uid, ok := mqtt.DeviceUIDFromTelemetryTopic("", expectedTelemetryTopic)
	require.True(t, ok)
	require.Equal(t, deviceUID, uid)

	_, ok = mqtt.DeviceUIDFromTelemetryTopic("", "invalid/topic/format")
	require.False(t, ok)

	_, ok = mqtt.DeviceUIDFromTelemetryTopic("", "ground-guard/devices/ESP32-123456/commands")
	require.False(t, ok)
}

func TestNewPahoClientWhenDisabled(t *testing.T) {
	config := util.Config{
		MQTTEnabled: false,
	}

	c, err := client.NewPahoClient(config)
	require.NoError(t, err)
	require.IsType(t, &client.NoopClient{}, c)
}

func TestNewPahoClientRequiresBrokerURL(t *testing.T) {
	config := util.Config{
		MQTTEnabled: true,
	}

	c, err := client.NewPahoClient(config)
	require.Error(t, err)
	require.Nil(t, c)
	require.Contains(t, err.Error(), "MQTT_BROKER_URL")
}

func TestNoopClient(t *testing.T) {
	c := client.NewNoopClient()

	require.NoError(t, c.Publish("topic", []byte("payload")))
	c2 := client.NewNoopClient()
	err := c2.Subscribe("some/topic", func(topic string, payload []byte) {})
	require.NoError(t, err)
	require.Equal(t, "ground-guard", c.TopicPrefix())
	c.Close()
}

func TestPublishCommand(t *testing.T) {
	c := client.NewNoopClient()
	payload := []byte(`{"action":"irrigate","data":{"duration_seconds":30}}`)

	require.NoError(t, publisher.PublishCommand(c, "ESP32-001", payload))
}
