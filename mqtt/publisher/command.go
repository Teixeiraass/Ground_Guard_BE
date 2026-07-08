package publisher

import (
	"github.com/Teixeiraass/ground_guard_be/mqtt"
	"github.com/Teixeiraass/ground_guard_be/mqtt/client"
)

// PublishCommand sends a command to a device via MQTT.
func PublishCommand(c client.Client, deviceUID string, payload []byte) error {
	topic := mqtt.DeviceCommandTopic(c.TopicPrefix(), deviceUID)
	return c.Publish(topic, payload)
}
